package auth_service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/yca-software/2chi-go-api/internals/constants"
	"github.com/yca-software/2chi-go-api/internals/models"
	"github.com/yca-software/2chi-go-api/internals/packages/authz"
	platform_i18n "github.com/yca-software/2chi-go-api/internals/packages/i18n"
	platform_oauth "github.com/yca-software/2chi-go-api/internals/packages/oauth"
	platform_repository "github.com/yca-software/2chi-go-api/internals/packages/repository"
	platform_subscription "github.com/yca-software/2chi-go-api/internals/packages/subscription"
	"github.com/yca-software/2chi-go-api/internals/repositories"
	admin_access_repository "github.com/yca-software/2chi-go-api/internals/repositories/admin_access"
	organization_billing_account_repository "github.com/yca-software/2chi-go-api/internals/repositories/billing_account"
	impersonation_session_repository "github.com/yca-software/2chi-go-api/internals/repositories/impersonation_session"
	invitation_repository "github.com/yca-software/2chi-go-api/internals/repositories/invitation"
	organization_member_repository "github.com/yca-software/2chi-go-api/internals/repositories/org_member"
	organization_repository "github.com/yca-software/2chi-go-api/internals/repositories/organization"
	user_repository "github.com/yca-software/2chi-go-api/internals/repositories/user"
	user_email_verification_token_repository "github.com/yca-software/2chi-go-api/internals/repositories/user_email_verification_token"
	user_identity_repository "github.com/yca-software/2chi-go-api/internals/repositories/user_identity"
	user_legal_document_acceptance_repository "github.com/yca-software/2chi-go-api/internals/repositories/user_legal_document_acceptance"
	user_password_reset_token_repository "github.com/yca-software/2chi-go-api/internals/repositories/user_password_reset_token"
	user_refresh_token_repository "github.com/yca-software/2chi-go-api/internals/repositories/user_refresh_token"
	chi_aws_ses "github.com/yca-software/2chi-go-aws/ses"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_google_oauth "github.com/yca-software/2chi-go-google/oauth"
	chi_localizer "github.com/yca-software/2chi-go-localizer"
	chi_logger "github.com/yca-software/2chi-go-logger"
	chi_password "github.com/yca-software/2chi-go-password"
	chi_repository "github.com/yca-software/2chi-go-repository"
	chi_template "github.com/yca-software/2chi-go-template"
	chi_types "github.com/yca-software/2chi-go-types"
	chi_validator "github.com/yca-software/2chi-go-validator"
)

type Dependencies struct {
	GenerateID        func() (uuid.UUID, error)
	Now               func() time.Time
	Validator         chi_validator.Validator
	Logger            chi_logger.Logger
	PasswordHashFn    func(password string) (string, error)
	GenerateToken     func() (string, error)
	HashToken         func(token string) string
	Authorizer        *authz.Authorizer
	Repositories      *repositories.Repositories
	RunInTx           repositories.TxRunner
	SessionCache      *authz.SessionCache
	AccessTokenSecret string
	AppURL            string
	Localizer         chi_localizer.Localizer
	EmailSender       chi_aws_ses.SES
	EmailTemplates    *chi_template.HTML
	GoogleOAuth       chi_google_oauth.OAuth
}

type Service interface {
	AuthenticateWithGoogle(ctx context.Context, req *AuthenticateWithGoogleRequest) (*AuthenticateResponse, error)
	AuthenticateWithPassword(ctx context.Context, req *AuthenticateWithPasswordRequest) (*AuthenticateResponse, error)
	ForgotPassword(ctx context.Context, req *ForgotPasswordRequest) error
	Logout(ctx context.Context, req *LogoutRequest, access *chi_types.AccessInfo) error
	RefreshAccessToken(ctx context.Context, req *RefreshAccessTokenRequest) (*RefreshAccessTokenResponse, error)
	ResetPassword(ctx context.Context, req *ResetPasswordRequest) error
	SignUp(ctx context.Context, req *SignUpRequest) (*SignUpResponse, error)
	VerifyEmail(ctx context.Context, req *VerifyEmailRequest) error
	Impersonate(ctx context.Context, req *ImpersonateRequest, access *chi_types.AccessInfo) (*AuthenticateResponse, error)
}

type service struct {
	generateID                      func() (uuid.UUID, error)
	now                             func() time.Time
	validator                       chi_validator.Validator
	logger                          chi_logger.Logger
	passwordHashFn                  func(password string) (string, error)
	generateToken                   func() (string, error)
	hashToken                       func(token string) string
	runInTx                         repositories.TxRunner
	authorizer                      *authz.Authorizer
	usersRepo                       user_repository.Repository
	adminAccessRepo                 admin_access_repository.Repository
	userIdentitiesRepo              user_identity_repository.Repository
	userRefreshTokensRepo           user_refresh_token_repository.Repository
	userPasswordResetTokensRepo     user_password_reset_token_repository.Repository
	userEmailVerificationTokensRepo user_email_verification_token_repository.Repository
	legalDocumentAcceptancesRepo    user_legal_document_acceptance_repository.Repository
	impersonationSessionsRepo       impersonation_session_repository.Repository
	organizationsRepo               organization_repository.Repository
	billingAccountsRepo             organization_billing_account_repository.Repository
	organizationMembersRepo         organization_member_repository.Repository
	invitationsRepo                 invitation_repository.Repository
	sessionCache                    *authz.SessionCache
	accessTokenSecret               string
	appURL                          string
	localizer                       chi_localizer.Localizer
	emailSender                     chi_aws_ses.SES
	emailTemplates                  *chi_template.HTML
	googleOAuth                     chi_google_oauth.OAuth
}

func New(deps Dependencies) Service {
	runInTx := deps.RunInTx
	if runInTx == nil && deps.Repositories != nil {
		runInTx = deps.Repositories.RunInTx
	}
	if deps.HashToken == nil {
		panic("auth service: HashToken is required")
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
		billingAccountsRepo:             deps.Repositories.OrganizationBillingAccounts,
		usersRepo:                       deps.Repositories.Users,
		adminAccessRepo:                 deps.Repositories.AdminAccess,
		userIdentitiesRepo:              deps.Repositories.UserIdentities,
		userRefreshTokensRepo:           deps.Repositories.UserRefreshTokens,
		userPasswordResetTokensRepo:     deps.Repositories.UserPasswordResetTokens,
		userEmailVerificationTokensRepo: deps.Repositories.UserEmailVerificationTokens,
		legalDocumentAcceptancesRepo:    deps.Repositories.UserLegalDocumentAcceptances,
		impersonationSessionsRepo:       deps.Repositories.ImpersonationSessions,
		organizationsRepo:               deps.Repositories.Organizations,
		organizationMembersRepo:         deps.Repositories.OrganizationMembers,
		invitationsRepo:                 deps.Repositories.Invitations,
		sessionCache:                    deps.SessionCache,
		accessTokenSecret:               deps.AccessTokenSecret,
		appURL:                          deps.AppURL,
		localizer:                       deps.Localizer,
		emailSender:                     deps.EmailSender,
		emailTemplates:                  deps.EmailTemplates,
		googleOAuth:                     deps.GoogleOAuth,
	}
}

func (s *service) AuthenticateWithPassword(ctx context.Context, req *AuthenticateWithPasswordRequest) (*AuthenticateResponse, error) {
	if err := s.validator.ValidateStruct(req); err != nil {
		return nil, chi_error.NewUnprocessableEntityError(errors.New("validation failed"), "", err)
	}

	emailLower := strings.ToLower(req.Email)

	user, err := s.usersRepo.GetByEmail(ctx, emailLower)
	if err != nil {
		if platform_repository.IsNotFound(err) {
			return nil, chi_error.NewNotFoundError(errors.New("invalid credentials"), "PasswordMismatch", nil)
		}
		return nil, err
	}

	if user.Password == "" || !chi_password.Compare(req.Password, user.Password) {
		return nil, chi_error.NewNotFoundError(errors.New("invalid credentials"), "PasswordMismatch", nil)
	}

	return s.issueAuthTokens(ctx, user, "", "", req.IPAddress, req.UserAgent, constants.REFRESH_TOKEN_TTL)
}

func (s *service) AuthenticateWithGoogle(ctx context.Context, req *AuthenticateWithGoogleRequest) (*AuthenticateResponse, error) {
	if err := s.validator.ValidateStruct(req); err != nil {
		return nil, chi_error.NewUnprocessableEntityError(errors.New("validation failed"), "", err)
	}
	if s.googleOAuth == nil {
		return nil, chi_error.NewInternalServerError(errors.New("google oauth is not configured"), "InternalServerError", nil)
	}

	googleUser, err := s.googleOAuth.GetUserInfo(ctx, req.Code)
	if err != nil {
		return nil, chi_error.NewUnauthorizedError(err, "InvalidToken", nil)
	}
	if !googleUser.VerifiedEmail {
		return nil, chi_error.NewUnauthorizedError(errors.New("email not verified"), "InvalidToken", nil)
	}

	emailLower := strings.ToLower(googleUser.Email)
	googleID := googleUser.ID
	now := s.now()

	var user *models.User

	identity, err := s.userIdentitiesRepo.GetByProviderAndProviderUserID(ctx, constants.USER_IDENTITY_PROVIDER_GOOGLE, googleID)
	if err != nil && !platform_repository.IsNotFound(err) {
		return nil, err
	}
	if identity != nil {
		user, err = s.usersRepo.GetByID(ctx, identity.UserID.String())
		if err != nil {
			return nil, err
		}
		if req.InvitationToken != "" {
			if err := s.acceptInvitationForUser(ctx, req.InvitationToken, emailLower, user, req.TermsVersion, req.PrivacyPolicyVersion, false); err != nil {
				return nil, err
			}
		}
	} else {
		existingUser, err := s.usersRepo.GetByEmail(ctx, emailLower)
		if err != nil && !platform_repository.IsNotFound(err) {
			return nil, err
		}

		if existingUser != nil {
			if googleUser.Picture != "" {
				existingUser.AvatarURL = googleUser.Picture
			}
			if existingUser.EmailVerifiedAt == nil {
				existingUser.EmailVerifiedAt = &now
			}
			if err := s.runInTx(ctx, func(tx chi_repository.Tx) error {
				if err := s.usersRepo.WithTx(tx).Update(ctx, existingUser); err != nil {
					return err
				}
				return s.createGoogleIdentity(ctx, tx, existingUser.ID, googleID)
			}); err != nil {
				return nil, err
			}
			user = existingUser
			if req.InvitationToken != "" {
				if err := s.acceptInvitationForUser(ctx, req.InvitationToken, emailLower, user, req.TermsVersion, req.PrivacyPolicyVersion, false); err != nil {
					return nil, err
				}
			}
		} else {
			userID, err := s.generateID()
			if err != nil {
				return nil, err
			}

			language := platform_i18n.NormalizeLanguage(req.Language)
			firstName, lastName := platform_oauth.GoogleUserNames(googleUser)

			user = &models.User{
				ModelBaseWithArchive: chi_types.ModelBaseWithArchive{
					ModelBase: chi_types.ModelBase{
						ID:        userID,
						CreatedAt: now,
					},
				},
				FirstName:       firstName,
				LastName:        lastName,
				Language:        language,
				Email:           emailLower,
				AvatarURL:       googleUser.Picture,
				EmailVerifiedAt: &now,
			}

			if req.InvitationToken != "" {
				if err := s.acceptInvitationForUser(ctx, req.InvitationToken, emailLower, user, req.TermsVersion, req.PrivacyPolicyVersion, true); err != nil {
					return nil, err
				}
				if err := s.createGoogleIdentity(ctx, nil, user.ID, googleID); err != nil {
					return nil, err
				}
			} else if err := s.runInTx(ctx, func(tx chi_repository.Tx) error {
				if err := s.usersRepo.WithTx(tx).Create(ctx, user); err != nil {
					return err
				}
				if err := s.createLegalDocumentAcceptances(ctx, s.legalDocumentAcceptancesRepo.WithTx(tx), user.ID, req.TermsVersion, req.PrivacyPolicyVersion); err != nil {
					return err
				}
				return s.createGoogleIdentity(ctx, tx, user.ID, googleID)
			}); err != nil {
				return nil, err
			}
		}
	}

	return s.issueAuthTokens(ctx, user, "", "", req.IPAddress, req.UserAgent, constants.REFRESH_TOKEN_TTL)
}

func (s *service) ForgotPassword(ctx context.Context, req *ForgotPasswordRequest) error {
	if err := s.validator.ValidateStruct(req); err != nil {
		return chi_error.NewUnprocessableEntityError(errors.New("validation failed"), "", err)
	}

	emailLower := strings.ToLower(req.Email)

	user, err := s.usersRepo.GetByEmail(ctx, emailLower)
	if err != nil {
		if platform_repository.IsNotFound(err) {
			return nil
		}
		return err
	}

	resetToken, err := s.generateToken()
	if err != nil {
		return err
	}

	resetTokenID, err := s.generateID()
	if err != nil {
		return err
	}

	now := s.now()
	if err := s.userPasswordResetTokensRepo.Create(ctx, &models.UserPasswordResetToken{
		ModelBase: chi_types.ModelBase{
			ID: resetTokenID,
		},
		UserID:    user.ID,
		ExpiresAt: now.Add(constants.PASSWORD_RESET_TOKEN_TTL),
		TokenHash: s.hashToken(resetToken),
	}); err != nil {
		return err
	}

	if s.emailSender == nil || s.emailTemplates == nil || s.localizer == nil {
		return chi_error.NewInternalServerError(errors.New("email is not configured"), "InternalServerError", nil)
	}

	language := platform_i18n.NormalizeLanguage(req.Language)
	body, err := s.emailTemplates.Render("reset", map[string]any{
		"Lang":         language,
		"Title":        s.localizer.Translate(language, "email.reset.title", nil),
		"Greeting":     s.localizer.Translate(language, "email.reset.greeting", nil),
		"Content":      s.localizer.Translate(language, "email.reset.content", nil),
		"Warning":      s.localizer.Translate(language, "email.reset.warning", nil),
		"ButtonText":   s.localizer.Translate(language, "email.reset.button", nil),
		"FooterIgnore": s.localizer.Translate(language, "email.reset.footer.ignore", nil),
		"FooterLink":   s.localizer.Translate(language, "email.reset.footer.link", nil),
		"C2ALink":      fmt.Sprintf("%s/reset-password?token=%s", s.appURL, resetToken),
	})
	if err != nil {
		return chi_error.NewInternalServerError(err, "InternalServerError", nil)
	}

	subject := s.localizer.Translate(language, "email.reset.subject", nil)
	return s.emailSender.Send(ctx, chi_aws_ses.SESEmailDataPayload{To: user.Email, Subject: subject, HTML: body})
}

func (s *service) Logout(ctx context.Context, req *LogoutRequest, access *chi_types.AccessInfo) error {
	if err := s.validator.ValidateStruct(req); err != nil {
		return chi_error.NewUnprocessableEntityError(errors.New("validation failed"), "", err)
	}

	refreshToken, err := s.userRefreshTokensRepo.GetByHash(ctx, s.hashToken(req.RefreshToken))
	if err != nil {
		if platform_repository.IsNotFound(err) {
			return chi_error.NewUnauthorizedError(errors.New("invalid refresh token"), "InvalidToken", nil)
		}
		return err
	}

	if access == nil || access.Type != chi_types.AccessTypeUser || access.SubjectID != refreshToken.UserID {
		return chi_error.NewForbiddenError(errors.New("refresh token does not belong to the authenticated user"), "Forbidden", nil)
	}

	if refreshToken.RevokedAt != nil {
		return chi_error.NewUnauthorizedError(errors.New("refresh token revoked"), "InvalidToken", nil)
	}
	if refreshToken.ExpiresAt.Before(s.now()) {
		return chi_error.NewUnauthorizedError(errors.New("refresh token expired"), "ExpiredToken", nil)
	}

	if err := s.runInTx(ctx, func(tx chi_repository.Tx) error {
		if err := s.userRefreshTokensRepo.WithTx(tx).RevokeByHash(ctx, refreshToken.TokenHash); err != nil {
			return err
		}
		if refreshToken.ImpersonatedBy.Valid {
			if err := s.impersonationSessionsRepo.WithTx(tx).EndByRefreshTokenID(
				ctx, refreshToken.ID, s.now(), constants.IMPERSONATION_END_REASON_LOGOUT,
			); err != nil && !platform_repository.IsNotFound(err) {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}

	if refreshToken.ImpersonatedBy.Valid {
		s.logger.WithContext(ctx).Info("impersonation_ended",
			"refresh_token_id", refreshToken.ID.String(),
			"target_user_id", refreshToken.UserID.String(),
			"end_reason", constants.IMPERSONATION_END_REASON_LOGOUT,
		)
	}

	return s.sessionCache.InvalidateSession(ctx, refreshToken.UserID.String())
}

func (s *service) RefreshAccessToken(ctx context.Context, req *RefreshAccessTokenRequest) (*RefreshAccessTokenResponse, error) {
	if err := s.validator.ValidateStruct(req); err != nil {
		return nil, chi_error.NewUnprocessableEntityError(errors.New("validation failed"), "", err)
	}

	refreshToken, err := s.userRefreshTokensRepo.GetByHash(ctx, s.hashToken(req.RefreshToken))
	if err != nil {
		if platform_repository.IsNotFound(err) {
			return nil, chi_error.NewUnauthorizedError(errors.New("invalid refresh token"), "InvalidToken", nil)
		}
		return nil, err
	}

	user, err := s.usersRepo.GetByID(ctx, refreshToken.UserID.String())
	if err != nil {
		if platform_repository.IsNotFound(err) {
			return nil, chi_error.NewUnauthorizedError(errors.New("invalid refresh token"), "InvalidToken", nil)
		}
		return nil, err
	}

	if refreshToken.RevokedAt != nil {
		s.logger.WithContext(ctx).Error("refresh attempt with revoked token",
			"refresh_token_id", refreshToken.ID,
			"user_id", refreshToken.UserID,
		)
		return nil, chi_error.NewUnauthorizedError(errors.New("refresh token revoked"), "InvalidToken", nil)
	}
	if refreshToken.ExpiresAt.Before(s.now()) {
		return nil, chi_error.NewUnauthorizedError(errors.New("refresh token expired"), "ExpiredToken", nil)
	}

	var impersonatedBy, impersonatedByEmail string
	if refreshToken.ImpersonatedBy.Valid {
		impersonatedBy = refreshToken.ImpersonatedBy.UUID.String()
		impersonator, err := s.usersRepo.GetByID(ctx, impersonatedBy)
		if err != nil {
			if platform_repository.IsNotFound(err) {
				return nil, chi_error.NewUnauthorizedError(errors.New("invalid refresh token"), "InvalidToken", nil)
			}
			return nil, err
		}
		impersonatedByEmail = impersonator.Email
	}

	accessToken, err := s.generateAccessToken(ctx, user, impersonatedBy, impersonatedByEmail)
	if err != nil {
		return nil, err
	}

	if err := s.cacheUserSession(ctx, user.ID.String(), impersonatedBy, impersonatedByEmail); err != nil {
		return nil, err
	}

	return &RefreshAccessTokenResponse{AccessToken: accessToken}, nil
}

func (s *service) ResetPassword(ctx context.Context, req *ResetPasswordRequest) error {
	if err := s.validator.ValidateStruct(req); err != nil {
		return chi_error.NewUnprocessableEntityError(errors.New("validation failed"), "", err)
	}

	resetToken, err := s.userPasswordResetTokensRepo.GetByHash(ctx, s.hashToken(req.Token))
	if err != nil {
		if platform_repository.IsNotFound(err) {
			return chi_error.NewUnauthorizedError(errors.New("invalid password reset token"), "InvalidPasswordResetToken", nil)
		}
		return err
	}

	user, err := s.usersRepo.GetByID(ctx, resetToken.UserID.String())
	if err != nil {
		return err
	}

	now := s.now()
	if resetToken.ExpiresAt.Before(now) {
		return chi_error.NewUnauthorizedError(errors.New("password reset token expired"), "ExpiredPasswordResetToken", nil)
	}
	if resetToken.UsedAt != nil {
		return chi_error.NewUnauthorizedError(errors.New("password reset token already used"), "InvalidPasswordResetToken", nil)
	}

	hashedPassword, err := s.passwordHashFn(req.Password)
	if err != nil {
		return err
	}
	user.Password = hashedPassword

	if err := s.runInTx(ctx, func(tx chi_repository.Tx) error {
		if err := s.userPasswordResetTokensRepo.WithTx(tx).MarkAsUsed(ctx, resetToken.ID.String()); err != nil {
			return err
		}
		if err := s.usersRepo.WithTx(tx).Update(ctx, user); err != nil {
			return err
		}
		return s.userRefreshTokensRepo.WithTx(tx).RevokeAllByUserID(ctx, user.ID.String(), nil)
	}); err != nil {
		return err
	}

	return s.sessionCache.InvalidateSession(ctx, user.ID.String())
}

func (s *service) SignUp(ctx context.Context, req *SignUpRequest) (*SignUpResponse, error) {
	if err := s.validator.ValidateStruct(req); err != nil {
		return nil, chi_error.NewUnprocessableEntityError(errors.New("validation failed"), "", err)
	}

	emailLower := strings.ToLower(req.Email)

	_, err := s.usersRepo.GetByEmail(ctx, emailLower)
	if err == nil {
		return nil, chi_error.NewConflictError(errors.New("email already in use"), "EmailAlreadyInUse", nil)
	}
	if !platform_repository.IsNotFound(err) {
		return nil, err
	}

	now := s.now()

	userID, err := s.generateID()
	if err != nil {
		return nil, err
	}

	hashedPassword, err := s.passwordHashFn(req.Password)
	if err != nil {
		return nil, err
	}

	language := platform_i18n.NormalizeLanguage(req.Language)

	user := &models.User{
		ModelBaseWithArchive: chi_types.ModelBaseWithArchive{
			ModelBase: chi_types.ModelBase{
				ID:        userID,
				CreatedAt: now,
			},
		},
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Language:  language,
		Email:     emailLower,
		Password:  hashedPassword,
	}

	if req.InvitationToken != "" {
		if err := s.acceptInvitationForUser(ctx, req.InvitationToken, emailLower, user, req.TermsVersion, req.PrivacyPolicyVersion, true); err != nil {
			return nil, err
		}
	} else if err := s.runInTx(ctx, func(tx chi_repository.Tx) error {
		if err := s.usersRepo.WithTx(tx).Create(ctx, user); err != nil {
			return err
		}
		return s.createLegalDocumentAcceptances(ctx, s.legalDocumentAcceptancesRepo.WithTx(tx), user.ID, req.TermsVersion, req.PrivacyPolicyVersion)
	}); err != nil {
		return nil, err
	}

	authResp, err := s.issueAuthTokens(ctx, user, "", "", req.IPAddress, req.UserAgent, constants.REFRESH_TOKEN_TTL)
	if err != nil {
		return nil, err
	}

	verificationToken, err := s.generateToken()
	if err != nil {
		return nil, err
	}
	verificationTokenID, err := s.generateID()
	if err != nil {
		return nil, err
	}

	if err := s.userEmailVerificationTokensRepo.Create(ctx, &models.UserEmailVerificationToken{
		ModelBase: chi_types.ModelBase{
			ID: verificationTokenID,
		},
		UserID:    user.ID,
		ExpiresAt: now.Add(constants.EMAIL_VERIFICATION_TOKEN_TTL),
		TokenHash: s.hashToken(verificationToken),
	}); err != nil {
		return nil, err
	}

	if s.emailSender != nil && s.emailTemplates != nil && s.localizer != nil {
		body, renderErr := s.emailTemplates.Render("verification", map[string]any{
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
		if renderErr != nil {
			s.logger.WithContext(ctx).Error("failed to render verification email after sign up", "error", renderErr, "userId", user.ID.String())
		} else {
			subject := s.localizer.Translate(language, "email.verification.subject", nil)
			if sendErr := s.emailSender.Send(ctx, chi_aws_ses.SESEmailDataPayload{To: user.Email, Subject: subject, HTML: body}); sendErr != nil {
				s.logger.WithContext(ctx).Error("failed to send verification email after sign up", "error", sendErr, "userId", user.ID.String())
			}
		}
	}

	return &SignUpResponse{
		AccessToken:  authResp.AccessToken,
		RefreshToken: authResp.RefreshToken,
	}, nil
}

func (s *service) VerifyEmail(ctx context.Context, req *VerifyEmailRequest) error {
	if err := s.validator.ValidateStruct(req); err != nil {
		return chi_error.NewUnprocessableEntityError(errors.New("validation failed"), "", err)
	}

	verificationToken, err := s.userEmailVerificationTokensRepo.GetByHash(ctx, s.hashToken(req.Token))
	if err != nil {
		if platform_repository.IsNotFound(err) {
			return chi_error.NewUnauthorizedError(errors.New("invalid verification token"), "InvalidVerificationToken", nil)
		}
		return err
	}

	user, err := s.usersRepo.GetByID(ctx, verificationToken.UserID.String())
	if err != nil {
		return err
	}

	if verificationToken.ExpiresAt.Before(s.now()) {
		return chi_error.NewUnauthorizedError(errors.New("verification token expired"), "ExpiredVerificationToken", nil)
	}
	if verificationToken.UsedAt != nil {
		return chi_error.NewUnauthorizedError(errors.New("verification token already used"), "InvalidVerificationToken", nil)
	}

	verifiedAt := s.now()
	user.EmailVerifiedAt = &verifiedAt

	return s.runInTx(ctx, func(tx chi_repository.Tx) error {
		if err := s.userEmailVerificationTokensRepo.WithTx(tx).MarkAsUsed(ctx, verificationToken.ID.String()); err != nil {
			return err
		}
		return s.usersRepo.WithTx(tx).Update(ctx, user)
	})
}

func (s *service) Impersonate(ctx context.Context, req *ImpersonateRequest, access *chi_types.AccessInfo) (*AuthenticateResponse, error) {
	if err := s.validator.ValidateStruct(req); err != nil {
		return nil, chi_error.NewUnprocessableEntityError(errors.New("validation failed"), "", err)
	}

	if err := s.authorizer.CheckPlatformAdmin(access); err != nil {
		return nil, err
	}

	user, err := s.usersRepo.GetByID(ctx, req.UserID)
	if err != nil {
		return nil, err
	}

	impersonatedBy := access.SubjectID.String()
	return s.issueAuthTokens(ctx, user, impersonatedBy, access.Email, req.IPAddress, req.UserAgent, time.Hour)
}

func (s *service) issueAuthTokens(
	ctx context.Context,
	user *models.User,
	impersonatedBy, impersonatedByEmail, ip, userAgent string,
	refreshTTL time.Duration,
) (*AuthenticateResponse, error) {
	accessToken, err := s.generateAccessToken(ctx, user, impersonatedBy, impersonatedByEmail)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.generateToken()
	if err != nil {
		return nil, err
	}

	refreshTokenID, err := s.generateID()
	if err != nil {
		return nil, err
	}

	now := s.now()
	tokenModel := &models.UserRefreshToken{
		ModelBase: chi_types.ModelBase{
			ID: refreshTokenID,
		},
		UserID:    user.ID,
		ExpiresAt: now.Add(refreshTTL),
		IP:        ip,
		UserAgent: userAgent,
		TokenHash: s.hashToken(refreshToken),
	}
	if impersonatedBy != "" {
		parsed, err := uuid.Parse(impersonatedBy)
		if err != nil {
			return nil, chi_error.NewInternalServerError(err, "InternalServerError", nil)
		}
		tokenModel.ImpersonatedBy = uuid.NullUUID{UUID: parsed, Valid: true}
	}

	var sessionID uuid.UUID
	if err := s.runInTx(ctx, func(tx chi_repository.Tx) error {
		if err := s.userRefreshTokensRepo.WithTx(tx).Create(ctx, tokenModel); err != nil {
			return err
		}
		if impersonatedBy == "" {
			return nil
		}

		var genErr error
		sessionID, genErr = s.generateID()
		if genErr != nil {
			return genErr
		}
		adminID, parseErr := uuid.Parse(impersonatedBy)
		if parseErr != nil {
			return chi_error.NewInternalServerError(parseErr, "InternalServerError", nil)
		}
		return s.impersonationSessionsRepo.WithTx(tx).Create(ctx, &models.ImpersonationSession{
			ID:              sessionID,
			AdminID:         adminID,
			AdminEmail:      impersonatedByEmail,
			TargetUserID:    user.ID,
			TargetUserEmail: user.Email,
			RefreshTokenID:  refreshTokenID,
			IP:              ip,
			UserAgent:       userAgent,
		})
	}); err != nil {
		return nil, err
	}

	if impersonatedBy != "" {
		s.logger.WithContext(ctx).Info("impersonation_started",
			"admin_id", impersonatedBy,
			"target_user_id", user.ID.String(),
			"session_id", sessionID.String(),
		)
	}

	if err := s.cacheUserSession(ctx, user.ID.String(), impersonatedBy, impersonatedByEmail); err != nil {
		return nil, err
	}

	return &AuthenticateResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *service) cacheUserSession(ctx context.Context, userID, impersonatedBy, impersonatedByEmail string) error {
	access, err := authz.LoadUserAccess(ctx, authz.LoadUserAccessDeps{
		UsersRepo:               s.usersRepo,
		AdminAccessRepo:         s.adminAccessRepo,
		OrganizationMembersRepo: s.organizationMembersRepo,
		OrganizationsRepo:       s.organizationsRepo,
		UserRefreshTokensRepo:   s.userRefreshTokensRepo,
	}, userID)
	if err != nil {
		return err
	}
	if impersonatedBy != "" {
		parsed, err := uuid.Parse(impersonatedBy)
		if err != nil {
			return err
		}
		access.ImpersonatedBy = uuid.NullUUID{UUID: parsed, Valid: true}
		access.ImpersonatedByEmail = impersonatedByEmail
	}
	return s.sessionCache.Set(ctx, access)
}

func (s *service) generateAccessToken(ctx context.Context, user *models.User, impersonatedBy, impersonatedByEmail string) (string, error) {
	orgRoles, err := s.organizationMembersRepo.ListByUserID(ctx, user.ID.String())
	if err != nil {
		return "", err
	}

	isAdmin := false
	_, err = s.adminAccessRepo.GetByUserID(ctx, user.ID.String())
	if err != nil {
		if !platform_repository.IsNotFound(err) {
			return "", err
		}
	} else {
		isAdmin = true
	}

	permissions := make([]chi_types.JWTAccessTokenPermissionData, 0)
	if orgRoles != nil {
		for _, role := range *orgRoles {
			permissions = append(permissions, chi_types.JWTAccessTokenPermissionData{
				OrganizationID: role.OrganizationID,
				Permissions:    role.RolePermissions,
			})
		}
	}

	claims := jwt.MapClaims{
		"sub":         user.ID.String(),
		"email":       user.Email,
		"exp":         s.now().Add(constants.ACCESS_TOKEN_TTL).Unix(),
		"iat":         s.now().Unix(),
		"permissions": permissions,
		"isAdmin":     isAdmin,
	}
	if impersonatedBy != "" {
		claims["impersonatedBy"] = impersonatedBy
		claims["impersonatedByEmail"] = impersonatedByEmail
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.accessTokenSecret))
}

func (s *service) createGoogleIdentity(ctx context.Context, tx chi_repository.Tx, userID uuid.UUID, googleID string) error {
	identityID, err := s.generateID()
	if err != nil {
		return err
	}
	repo := s.userIdentitiesRepo
	if tx != nil {
		repo = s.userIdentitiesRepo.WithTx(tx)
	}
	return repo.Create(ctx, &models.UserIdentity{
		ModelBase: chi_types.ModelBase{
			ID: identityID,
		},
		UserID:         userID,
		Provider:       constants.USER_IDENTITY_PROVIDER_GOOGLE,
		ProviderUserID: googleID,
	})
}

func (s *service) acceptInvitationForUser(
	ctx context.Context,
	invitationToken, emailLower string,
	user *models.User,
	termsVersion, privacyPolicyVersion string,
	createUser bool,
) error {
	invitation, err := s.validateInvitationAcceptance(ctx, invitationToken, emailLower)
	if err != nil {
		return err
	}
	if err := s.persistUserAndInvitationAcceptance(ctx, invitation, user, createUser, termsVersion, privacyPolicyVersion); err != nil {
		return err
	}
	if s.sessionCache != nil {
		return s.sessionCache.InvalidateSession(ctx, user.ID.String())
	}
	return nil
}

func (s *service) validateInvitationAcceptance(ctx context.Context, invitationToken, emailLower string) (*models.Invitation, error) {
	invitation, err := s.invitationsRepo.GetByTokenHash(ctx, s.hashToken(invitationToken))
	if err != nil {
		if platform_repository.IsNotFound(err) {
			return nil, chi_error.NewUnprocessableEntityError(errors.New("invalid invitation token"), "InvalidInvitationToken", nil)
		}
		return nil, err
	}

	if invitation.RevokedAt != nil {
		return nil, chi_error.NewUnprocessableEntityError(errors.New("invitation revoked"), "InvitationRevoked", nil)
	}
	if invitation.AcceptedAt != nil {
		return nil, chi_error.NewUnprocessableEntityError(errors.New("invitation already accepted"), "InvitationAlreadyAccepted", nil)
	}
	if invitation.ExpiresAt.Before(s.now()) {
		return nil, chi_error.NewUnprocessableEntityError(errors.New("invitation expired"), "InvitationExpired", nil)
	}
	if strings.ToLower(invitation.Email) != emailLower {
		return nil, chi_error.NewForbiddenError(errors.New("invitation email mismatch"), "InvitationEmailMismatch", nil)
	}

	if _, err := s.organizationsRepo.GetByID(ctx, invitation.OrganizationID.String()); err != nil {
		return nil, err
	}

	billing, err := s.billingAccountsRepo.GetByOrganizationID(ctx, invitation.OrganizationID.String())
	if err != nil {
		return nil, err
	}

	members, err := s.organizationMembersRepo.ListByOrganizationID(ctx, invitation.OrganizationID.String())
	if err != nil {
		return nil, err
	}
	if platform_subscription.OrganizationAtSeatLimit(len(*members), billing.SubscriptionSeats) {
		return nil, chi_error.NewForbiddenError(errors.New("organization seats limit reached"), "OrganizationSeatsLimit", nil)
	}

	return invitation, nil
}

func (s *service) persistUserAndInvitationAcceptance(ctx context.Context, invitation *models.Invitation, user *models.User, createUser bool, termsVersion, privacyPolicyVersion string) error {
	acceptedAt := s.now()
	membershipID, err := s.generateID()
	if err != nil {
		return err
	}

	member := &models.OrganizationMember{
		ModelBase: chi_types.ModelBase{
			ID:        membershipID,
			CreatedAt: acceptedAt,
		},
		UserID:         user.ID,
		OrganizationID: invitation.OrganizationID,
		RoleID:         invitation.RoleID,
	}

	return s.runInTx(ctx, func(tx chi_repository.Tx) error {
		orgRepo := s.organizationsRepo.WithTx(tx)
		membersRepo := s.organizationMembersRepo.WithTx(tx)

		if _, orgErr := orgRepo.GetByID(ctx, invitation.OrganizationID.String()); orgErr != nil {
			return orgErr
		}

		billing, billingErr := s.billingAccountsRepo.GetByOrganizationID(ctx, invitation.OrganizationID.String())
		if billingErr != nil {
			return billingErr
		}

		members, listErr := membersRepo.ListByOrganizationID(ctx, invitation.OrganizationID.String())
		if listErr != nil {
			return listErr
		}
		if platform_subscription.OrganizationAtSeatLimit(len(*members), billing.SubscriptionSeats) {
			return chi_error.NewForbiddenError(errors.New("organization seats limit reached"), "OrganizationSeatsLimit", nil)
		}

		if createUser {
			if err := s.usersRepo.WithTx(tx).Create(ctx, user); err != nil {
				return err
			}
		}
		if err := s.createLegalDocumentAcceptances(ctx, s.legalDocumentAcceptancesRepo.WithTx(tx), user.ID, termsVersion, privacyPolicyVersion); err != nil {
			return err
		}

		invitation.AcceptedAt = &acceptedAt
		if err := s.invitationsRepo.WithTx(tx).Update(ctx, invitation); err != nil {
			return err
		}

		if err := membersRepo.Create(ctx, member); err != nil {
			if e, ok := chi_error.AsError(err); ok && e.StatusCode == http.StatusConflict {
				return chi_error.NewConflictError(errors.New("user is already a member"), "UserAlreadyMember", nil)
			}
			return err
		}
		return nil
	})
}
