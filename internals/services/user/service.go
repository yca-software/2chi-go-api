package user_service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	chi_aws_ses "github.com/yca-software/2chi-go-aws/ses"
	"github.com/yca-software/2chi-go-api/internals/constants"
	"github.com/yca-software/2chi-go-api/internals/models"
	"github.com/yca-software/2chi-go-api/internals/repositories"
	admin_access_repository "github.com/yca-software/2chi-go-api/internals/repositories/admin_access"
	organization_member_repository "github.com/yca-software/2chi-go-api/internals/repositories/org_member"
	user_repository "github.com/yca-software/2chi-go-api/internals/repositories/user"
	user_email_verification_token_repository "github.com/yca-software/2chi-go-api/internals/repositories/user_email_verification_token"
	user_legal_document_acceptance_repository "github.com/yca-software/2chi-go-api/internals/repositories/user_legal_document_acceptance"
	user_password_reset_token_repository "github.com/yca-software/2chi-go-api/internals/repositories/user_password_reset_token"
	user_refresh_token_repository "github.com/yca-software/2chi-go-api/internals/repositories/user_refresh_token"
	"github.com/yca-software/2chi-go-api/internals/packages/authz"
	platform_repository "github.com/yca-software/2chi-go-api/internals/packages/repository"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_localizer "github.com/yca-software/2chi-go-localizer"
	chi_logger "github.com/yca-software/2chi-go-logger"
	chi_password "github.com/yca-software/2chi-go-password"
	chi_repository "github.com/yca-software/2chi-go-repository"
	chi_template "github.com/yca-software/2chi-go-template"
	chi_types "github.com/yca-software/2chi-go-types"
	chi_validator "github.com/yca-software/2chi-go-validator"
	"golang.org/x/sync/errgroup"
)

type Dependencies struct {
	GenerateID     func() (uuid.UUID, error)
	Now            func() time.Time
	Validator      chi_validator.Validator
	Logger         chi_logger.Logger
	PasswordHashFn func(password string) (string, error)
	GenerateToken  func() (string, error)
	HashToken      func(token string) string
	Authorizer     *authz.Authorizer
	Repositories   *repositories.Repositories
	RunInTx        repositories.TxRunner
	SessionCache   *authz.SessionCache
	AppURL         string
	Localizer      chi_localizer.Localizer
	EmailSender    chi_aws_ses.SES
	EmailTemplates *chi_template.HTML
}

type Service interface {
	AcceptTerms(ctx context.Context, req *AcceptTermsRequest, access *chi_types.AccessInfo) (*UserProfile, error)
	ChangePassword(ctx context.Context, req *ChangePasswordRequest, access *chi_types.AccessInfo) error
	UpdateProfile(ctx context.Context, req *UpdateProfileRequest, access *chi_types.AccessInfo) (*models.User, error)
	UpdateLanguage(ctx context.Context, req *UpdateLanguageRequest, access *chi_types.AccessInfo) (*models.User, error)

	ArchiveUser(ctx context.Context, req *ArchiveUserRequest, access *chi_types.AccessInfo) error
	RestoreUser(ctx context.Context, req *RestoreUserRequest, access *chi_types.AccessInfo) (*models.User, error)

	RevokeUserRefreshToken(ctx context.Context, req *RevokeUserRefreshTokenRequest, access *chi_types.AccessInfo) error
	RevokeUserAllRefreshTokens(ctx context.Context, req *RevokeUserAllRefreshTokensRequest, access *chi_types.AccessInfo) error
	RevokeUserAdminAccess(ctx context.Context, req *RevokeUserAdminAccessRequest, access *chi_types.AccessInfo) error

	GetUser(ctx context.Context, req *GetUserRequest, access *chi_types.AccessInfo) (*GetUserResponse, error)
	ListUsers(ctx context.Context, req *ListUsersRequest, access *chi_types.AccessInfo) (*ListUsersResponse, error)
	ListUserActiveRefreshTokens(ctx context.Context, req *ListUserActiveRefreshTokensRequest, access *chi_types.AccessInfo) (*[]models.UserRefreshToken, error)

	ResendVerificationEmail(ctx context.Context, req *ResendVerificationEmailRequest, access *chi_types.AccessInfo) error

	CleanupArchivedUsers(ctx context.Context) error
	CleanupStaleUnusedUserTokens(ctx context.Context) error
}

type service struct {
	generateID                  func() (uuid.UUID, error)
	now                         func() time.Time
	validator                   chi_validator.Validator
	logger                      chi_logger.Logger
	passwordHashFn              func(password string) (string, error)
	generateToken               func() (string, error)
	hashToken                   func(token string) string
	runInTx                     repositories.TxRunner
	authorizer                  *authz.Authorizer
	sessionCache                *authz.SessionCache
	usersRepo                   user_repository.UsersRepository
	adminAccessRepo             admin_access_repository.AdminAccessRepository
	organizationMembersRepo     organization_member_repository.OrganizationMembersRepository
	userRefreshTokensRepo       user_refresh_token_repository.UserRefreshTokenRepository
	userPasswordResetTokensRepo user_password_reset_token_repository.UserPasswordResetTokenRepository
	userEmailVerificationTokensRepo user_email_verification_token_repository.UserEmailVerificationTokenRepository
	legalDocumentAcceptancesRepo    user_legal_document_acceptance_repository.UserLegalDocumentAcceptanceRepository
	appURL                      string
	localizer                   chi_localizer.Localizer
	emailSender                 chi_aws_ses.SES
	emailTemplates              *chi_template.HTML
}

func New(deps Dependencies) Service {
	if deps.HashToken == nil {
		panic("user service: HashToken is required")
	}
	runInTx := deps.RunInTx
	if runInTx == nil && deps.Repositories != nil {
		runInTx = deps.Repositories.RunInTx
	}
	return &service{
		generateID:                      deps.GenerateID,
		now:                             deps.Now,
		validator:                       deps.Validator,
		logger:                          deps.Logger,
		passwordHashFn:                  deps.PasswordHashFn,
		generateToken:                   deps.GenerateToken,
		hashToken:                       deps.HashToken,
		runInTx:                         runInTx,
		authorizer:                      deps.Authorizer,
		sessionCache:                    deps.SessionCache,
		usersRepo:                       deps.Repositories.Users,
		adminAccessRepo:                 deps.Repositories.AdminAccess,
		organizationMembersRepo:         deps.Repositories.OrganizationMembers,
		userRefreshTokensRepo:           deps.Repositories.UserRefreshTokens,
		userPasswordResetTokensRepo:     deps.Repositories.UserPasswordResetTokens,
		userEmailVerificationTokensRepo: deps.Repositories.UserEmailVerificationTokens,
		legalDocumentAcceptancesRepo:    deps.Repositories.UserLegalDocumentAcceptances,
		appURL:                          deps.AppURL,
		localizer:                       deps.Localizer,
		emailSender:                     deps.EmailSender,
		emailTemplates:                  deps.EmailTemplates,
	}
}

func (s *service) AcceptTerms(ctx context.Context, req *AcceptTermsRequest, access *chi_types.AccessInfo) (*UserProfile, error) {
	if err := s.validator.ValidateStruct(req); err != nil {
		return nil, chi_error.NewUnprocessableEntityError(errors.New("validation failed"), "", err)
	}

	if err := s.authorizer.CheckOwnResource(access, req.UserID); err != nil {
		return nil, err
	}

	user, err := s.usersRepo.GetUserByID(ctx, req.UserID)
	if err != nil {
		return nil, err
	}

	if err := s.runInTx(ctx, func(tx chi_repository.Tx) error {
		return s.createLegalDocumentAcceptances(
			ctx,
			s.legalDocumentAcceptancesRepo.WithTx(tx),
			user.ID,
			req.TermsVersion,
			req.PrivacyPolicyVersion,
		)
	}); err != nil {
		return nil, err
	}

	return s.userProfileWithLegalAcceptances(ctx, user)
}

func (s *service) ChangePassword(ctx context.Context, req *ChangePasswordRequest, access *chi_types.AccessInfo) error {
	if err := s.validator.ValidateStruct(req); err != nil {
		return chi_error.NewUnprocessableEntityError(errors.New("validation failed"), "", err)
	}

	if err := s.authorizer.CheckOwnResource(access, req.UserID); err != nil {
		return err
	}

	user, err := s.usersRepo.GetUserByID(ctx, req.UserID)
	if err != nil {
		return err
	}

	if user.Password != "" {
		if isMatch := chi_password.Compare(req.OldPassword, user.Password); !isMatch {
			return chi_error.NewUnprocessableEntityError(errors.New("old password mismatch"), "OldPasswordMismatch", nil)
		}
	}

	hashedPassword, err := s.passwordHashFn(req.NewPassword)
	if err != nil {
		return err
	}

	user.Password = hashedPassword

	if err := s.usersRepo.UpdateUser(ctx, user); err != nil {
		return err
	}
	if err := s.userRefreshTokensRepo.RevokeAllRefreshTokensByUserID(ctx, req.UserID, nil); err != nil {
		return err
	}
	return s.sessionCache.InvalidateSession(ctx, req.UserID)
}

func (s *service) UpdateProfile(ctx context.Context, req *UpdateProfileRequest, access *chi_types.AccessInfo) (*models.User, error) {
	if err := s.validator.ValidateStruct(req); err != nil {
		return nil, chi_error.NewUnprocessableEntityError(errors.New("validation failed"), "", err)
	}

	if err := s.authorizer.CheckOwnResource(access, req.UserID); err != nil {
		return nil, err
	}

	user, err := s.usersRepo.GetUserByID(ctx, req.UserID)
	if err != nil {
		return nil, err
	}

	user.FirstName = req.FirstName
	user.LastName = req.LastName

	return user, s.usersRepo.UpdateUser(ctx, user)
}

func (s *service) UpdateLanguage(ctx context.Context, req *UpdateLanguageRequest, access *chi_types.AccessInfo) (*models.User, error) {
	if err := s.validator.ValidateStruct(req); err != nil {
		return nil, chi_error.NewUnprocessableEntityError(errors.New("validation failed"), "", err)
	}

	if err := s.authorizer.CheckOwnResource(access, req.UserID); err != nil {
		return nil, err
	}

	user, err := s.usersRepo.GetUserByID(ctx, req.UserID)
	if err != nil {
		return nil, err
	}

	user.Language = req.Language

	return user, s.usersRepo.UpdateUser(ctx, user)
}

func (s *service) ArchiveUser(ctx context.Context, req *ArchiveUserRequest, access *chi_types.AccessInfo) error {
	if err := s.validator.ValidateStruct(req); err != nil {
		return chi_error.NewUnprocessableEntityError(errors.New("validation failed"), "", err)
	}

	if err := s.authorizer.CheckOwnResource(access, req.UserID); err != nil {
		return err
	}

	user, err := s.usersRepo.GetUserByID(ctx, req.UserID)
	if err != nil {
		return err
	}

	return s.usersRepo.ArchiveUser(ctx, user)
}

func (s *service) RestoreUser(ctx context.Context, req *RestoreUserRequest, access *chi_types.AccessInfo) (*models.User, error) {
	if err := s.validator.ValidateStruct(req); err != nil {
		return nil, chi_error.NewUnprocessableEntityError(errors.New("validation failed"), "", err)
	}

	if err := s.authorizer.CheckOwnResource(access, req.UserID); err != nil {
		return nil, err
	}

	user, err := s.usersRepo.GetUserByIDIncludeArchived(ctx, req.UserID)
	if err != nil {
		return nil, err
	}

	return user, s.usersRepo.RestoreUser(ctx, req.UserID)
}

func (s *service) RevokeUserRefreshToken(ctx context.Context, req *RevokeUserRefreshTokenRequest, access *chi_types.AccessInfo) error {
	if err := s.validator.ValidateStruct(req); err != nil {
		return chi_error.NewUnprocessableEntityError(errors.New("validation failed"), "", err)
	}

	if err := s.authorizer.CheckOwnResource(access, req.UserID); err != nil {
		return err
	}

	if err := s.userRefreshTokensRepo.RevokeRefreshTokenByID(ctx, req.UserID, req.RefreshTokenID); err != nil {
		return err
	}

	return s.sessionCache.InvalidateSession(ctx, req.UserID)
}

func (s *service) RevokeUserAdminAccess(ctx context.Context, req *RevokeUserAdminAccessRequest, access *chi_types.AccessInfo) error {
	if err := s.validator.ValidateStruct(req); err != nil {
		return chi_error.NewUnprocessableEntityError(errors.New("validation failed"), "", err)
	}

	if err := s.authorizer.CheckPlatformAdmin(access); err != nil {
		return err
	}

	if access.SubjectID.String() == req.UserID {
		return chi_error.NewForbiddenError(errors.New("cannot revoke own admin access"), "CannotRevokeOwnAdminAccess", nil)
	}

	if _, err := s.adminAccessRepo.GetAdminAccessByUserID(ctx, req.UserID); err != nil {
		return err
	}

	if err := s.adminAccessRepo.DeleteAdminAccessByUserID(ctx, req.UserID); err != nil {
		return err
	}

	return s.sessionCache.InvalidateSession(ctx, req.UserID)
}

func (s *service) RevokeUserAllRefreshTokens(ctx context.Context, req *RevokeUserAllRefreshTokensRequest, access *chi_types.AccessInfo) error {
	if err := s.validator.ValidateStruct(req); err != nil {
		return chi_error.NewUnprocessableEntityError(errors.New("validation failed"), "", err)
	}

	if err := s.authorizer.CheckOwnResource(access, req.UserID); err != nil {
		return err
	}

	if req.KeepRefreshToken == "" {
		return s.userRefreshTokensRepo.RevokeAllRefreshTokensByUserID(ctx, req.UserID, nil)
	}

	current, err := s.userRefreshTokensRepo.GetRefreshTokenByHash(ctx, s.hashToken(req.KeepRefreshToken))
	if err != nil {
		if platform_repository.IsNotFound(err) {
			return chi_error.NewUnprocessableEntityError(errors.New("refresh token to keep not found"), "RefreshTokenToKeepNotFound", nil)
		}
		return err
	}

	if current.UserID.String() != req.UserID {
		return chi_error.NewForbiddenError(errors.New("refresh token to keep is not owned by the user"), "RefreshTokenToKeepNotOwnedByUser", nil)
	}

	if current.RevokedAt != nil || current.ExpiresAt.Before(s.now()) {
		return chi_error.NewUnprocessableEntityError(errors.New("refresh token to keep is invalid"), "RefreshTokenToKeepInvalid", nil)
	}

	excludeID := current.ID.String()
	return s.userRefreshTokensRepo.RevokeAllRefreshTokensByUserID(ctx, req.UserID, &excludeID)
}

func (s *service) GetUser(ctx context.Context, req *GetUserRequest, access *chi_types.AccessInfo) (*GetUserResponse, error) {
	if err := s.validator.ValidateStruct(req); err != nil {
		return nil, chi_error.NewUnprocessableEntityError(errors.New("validation failed"), "", err)
	}

	if err := s.authorizer.CheckOwnResource(access, req.UserID); err != nil {
		return nil, err
	}

	user, err := s.usersRepo.GetUserByID(ctx, req.UserID)
	if err != nil {
		return nil, err
	}

	roles, err := s.organizationMembersRepo.ListByUserIDWithRole(ctx, req.UserID)
	if err != nil {
		return nil, err
	}

	adminAccess, err := s.adminAccessRepo.GetAdminAccessByUserID(ctx, req.UserID)
	if err != nil {
		if platform_repository.IsNotFound(err) {
			profile, profileErr := s.userProfileWithLegalAcceptances(ctx, user)
			if profileErr != nil {
				return nil, profileErr
			}
			return &GetUserResponse{User: *profile, Roles: *roles}, nil
		}
		return nil, err
	}

	profile, err := s.userProfileWithLegalAcceptances(ctx, user)
	if err != nil {
		return nil, err
	}

	return &GetUserResponse{User: *profile, AdminAccess: adminAccess, Roles: *roles}, nil
}

func (s *service) ResendVerificationEmail(ctx context.Context, req *ResendVerificationEmailRequest, access *chi_types.AccessInfo) error {
	if err := s.validator.ValidateStruct(req); err != nil {
		return chi_error.NewUnprocessableEntityError(errors.New("validation failed"), "", err)
	}

	if err := s.authorizer.CheckOwnResource(access, req.UserID); err != nil {
		return err
	}

	user, err := s.usersRepo.GetUserByID(ctx, req.UserID)
	if err != nil {
		return err
	}

	if user.EmailVerifiedAt != nil {
		return chi_error.NewConflictError(errors.New("email already verified"), "EmailAlreadyVerified", nil)
	}

	verificationToken, err := s.generateToken()
	if err != nil {
		return err
	}

	verificationTokenID, err := s.generateID()
	if err != nil {
		return err
	}

	now := s.now()
	if err := s.userEmailVerificationTokensRepo.CreateEmailVerificationToken(ctx, &models.UserEmailVerificationToken{
		ModelBase: chi_types.ModelBase{
			ID: verificationTokenID,
		},
		UserID:    user.ID,
		ExpiresAt: now.Add(constants.EMAIL_VERIFICATION_TOKEN_TTL),
		TokenHash: s.hashToken(verificationToken),
	}); err != nil {
		return err
	}

	language := req.Language
	if language == "" {
		language = user.Language
	}

	if s.emailSender == nil || s.emailTemplates == nil || s.localizer == nil {
		return chi_error.NewInternalServerError(errors.New("email is not configured"), "InternalServerError", nil)
	}

	body, err := s.emailTemplates.Render("verification", map[string]any{
		"Lang":         language,
		"Title":        s.localizer.Translate(language, "email.verification.title", nil),
		"Greeting":     s.localizer.Translate(language, "email.verification.greeting", nil),
		"Content":      s.localizer.Translate(language, "email.verification.content", nil),
		"Warning":      s.localizer.Translate(language, "email.verification.warning", nil),
		"ButtonText":   s.localizer.Translate(language, "email.verification.button", nil),
		"FooterIgnore": s.localizer.Translate(language, "email.verification.footer.ignore", nil),
		"FooterLink":   s.localizer.Translate(language, "email.verification.footer.link", nil),
		"C2ALink":      fmt.Sprintf("%s/verify-email?token=%s", s.appURL, verificationToken),
	})
	if err != nil {
		return chi_error.NewInternalServerError(err, "InternalServerError", nil)
	}

	subject := s.localizer.Translate(language, "email.verification.subject", nil)
	return s.emailSender.Send(ctx, chi_aws_ses.SESEmailDataPayload{To: user.Email, Subject: subject, HTML: body})
}

func (s *service) ListUsers(ctx context.Context, req *ListUsersRequest, access *chi_types.AccessInfo) (*ListUsersResponse, error) {
	if err := s.validator.ValidateStruct(req); err != nil {
		return nil, chi_error.NewUnprocessableEntityError(errors.New("validation failed"), "", err)
	}

	if err := s.authorizer.CheckPlatformAdmin(access); err != nil {
		return nil, err
	}

	limit := req.Limit + 1
	users, err := s.usersRepo.SearchUsers(ctx, req.SearchPhrase, req.ArchiveFilter, limit, req.Offset)
	if err != nil {
		return nil, err
	}

	items := *users
	hasNext := len(items) > req.Limit
	if hasNext {
		items = items[:req.Limit]
	}

	return &ListUsersResponse{Items: items, HasNext: hasNext}, nil
}

func (s *service) ListUserActiveRefreshTokens(ctx context.Context, req *ListUserActiveRefreshTokensRequest, access *chi_types.AccessInfo) (*[]models.UserRefreshToken, error) {
	if err := s.validator.ValidateStruct(req); err != nil {
		return nil, chi_error.NewUnprocessableEntityError(errors.New("validation failed"), "", err)
	}

	if err := s.authorizer.CheckOwnResource(access, req.UserID); err != nil {
		return nil, err
	}

	return s.userRefreshTokensRepo.GetActiveRefreshTokensByUserID(ctx, req.UserID)
}

func (s *service) CleanupArchivedUsers(ctx context.Context) error {
	return s.usersRepo.CleanupArchivedUsers(ctx)
}

func (s *service) CleanupStaleUnusedUserTokens(ctx context.Context) error {
	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return s.userRefreshTokensRepo.CleanupStaleUnusedRefreshTokens(gctx)
	})
	g.Go(func() error {
		return s.userPasswordResetTokensRepo.CleanupStaleUnusedPasswordResetTokens(gctx)
	})
	g.Go(func() error {
		return s.userEmailVerificationTokensRepo.CleanupStaleUnusedEmailVerificationTokens(gctx)
	})

	return g.Wait()
}
