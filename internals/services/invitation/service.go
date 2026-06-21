package invitation_service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/yca-software/2chi-go-api/internals/constants"
	"github.com/yca-software/2chi-go-api/internals/models"
	"github.com/yca-software/2chi-go-api/internals/repositories"
	billing_account_repository "github.com/yca-software/2chi-go-api/internals/repositories/billing_account"
	invitation_repository "github.com/yca-software/2chi-go-api/internals/repositories/invitation"
	organization_member_repository "github.com/yca-software/2chi-go-api/internals/repositories/org_member"
	organization_repository "github.com/yca-software/2chi-go-api/internals/repositories/organization"
	role_repository "github.com/yca-software/2chi-go-api/internals/repositories/role"
	user_repository "github.com/yca-software/2chi-go-api/internals/repositories/user"
	"github.com/yca-software/2chi-go-api/internals/packages/authz"
	platform_subscription "github.com/yca-software/2chi-go-api/internals/packages/subscription"
	chi_aws_ses "github.com/yca-software/2chi-go-aws/ses"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_localizer "github.com/yca-software/2chi-go-localizer"
	chi_logger "github.com/yca-software/2chi-go-logger"
	chi_repository "github.com/yca-software/2chi-go-repository"
	chi_template "github.com/yca-software/2chi-go-template"
	chi_types "github.com/yca-software/2chi-go-types"
	chi_validator "github.com/yca-software/2chi-go-validator"
)

type Dependencies struct {
	InvitationTTL  time.Duration
	AppURL         string
	GenerateID     func() (uuid.UUID, error)
	Now            func() time.Time
	Validator      chi_validator.Validator
	Logger         chi_logger.Logger
	GenerateToken  func() (string, error)
	HashToken      func(token string) string
	Authorizer     *authz.Authorizer
	Repositories   *repositories.Repositories
	RunInTx        repositories.TxRunner
	SessionCache   *authz.SessionCache
	Localizer      chi_localizer.Localizer
	EmailSender    chi_aws_ses.SES
	EmailTemplates *chi_template.HTML
}

type Service interface {
	CreateInvitation(ctx context.Context, req *CreateInvitationRequest, access *chi_types.AccessInfo) (*CreateInvitationResponse, error)
	RevokeInvitation(ctx context.Context, req *RevokeInvitationRequest, access *chi_types.AccessInfo) error
	ListInvitations(ctx context.Context, req *ListInvitationsRequest, access *chi_types.AccessInfo) (*[]models.Invitation, error)
	CleanupStaleInvitations(ctx context.Context) error
}

type service struct {
	invitationTTL           time.Duration
	appURL                  string
	generateID              func() (uuid.UUID, error)
	now                     func() time.Time
	validator               chi_validator.Validator
	logger                  chi_logger.Logger
	generateToken           func() (string, error)
	hashToken               func(token string) string
	authorizer              *authz.Authorizer
	runInTx                 repositories.TxRunner
	invitationsRepo         invitation_repository.InvitationsRepository
	orgsRepo                organization_repository.OrganizationsRepository
	billingAccountsRepo     billing_account_repository.OrganizationBillingAccountsRepository
	organizationMembersRepo organization_member_repository.OrganizationMembersRepository
	usersRepo               user_repository.UsersRepository
	rolesRepo               role_repository.RolesRepository
	sessionCache            *authz.SessionCache
	localizer               chi_localizer.Localizer
	emailSender             chi_aws_ses.SES
	emailTemplates          *chi_template.HTML
}

func New(deps Dependencies) Service {
	runInTx := deps.RunInTx
	if runInTx == nil && deps.Repositories != nil {
		runInTx = deps.Repositories.RunInTx
	}
	ttl := deps.InvitationTTL
	if ttl <= 0 {
		ttl = constants.INVITATION_TOKEN_TTL
	}
	return &service{
		invitationTTL:           ttl,
		appURL:                  deps.AppURL,
		generateID:              deps.GenerateID,
		now:                     deps.Now,
		validator:               deps.Validator,
		logger:                  deps.Logger,
		generateToken:           deps.GenerateToken,
		hashToken:               deps.HashToken,
		authorizer:              deps.Authorizer,
		billingAccountsRepo:     deps.Repositories.OrganizationBillingAccounts,
		runInTx:                 runInTx,
		invitationsRepo:         deps.Repositories.Invitations,
		orgsRepo:                deps.Repositories.Organizations,
		organizationMembersRepo: deps.Repositories.OrganizationMembers,
		usersRepo:               deps.Repositories.Users,
		rolesRepo:               deps.Repositories.Roles,
		sessionCache:            deps.SessionCache,
		localizer:               deps.Localizer,
		emailSender:             deps.EmailSender,
		emailTemplates:          deps.EmailTemplates,
	}
}

func (s *service) CreateInvitation(ctx context.Context, req *CreateInvitationRequest, access *chi_types.AccessInfo) (*CreateInvitationResponse, error) {
	if err := s.validator.ValidateStruct(req); err != nil {
		return nil, chi_error.NewUnprocessableEntityError(errors.New("validation failed"), "", err)
	}

	org, err := s.orgsRepo.GetOrganizationByID(ctx, req.OrganizationID)
	if err != nil {
		return nil, err
	}

	if err := s.checkCreateInvitationPermission(ctx, access, req.OrganizationID); err != nil {
		return nil, err
	}

	role, err := s.rolesRepo.GetRoleByID(ctx, req.OrganizationID, req.RoleID)
	if err != nil {
		return nil, err
	}

	if err := s.ensureNoPendingInvitation(ctx, req.OrganizationID, req.Email); err != nil {
		return nil, err
	}

	emailLower := strings.ToLower(req.Email)
	existingUser, userErr := s.usersRepo.GetUserByEmail(ctx, emailLower)
	if userErr != nil {
		if apiErr, ok := userErr.(*chi_error.Error); !ok || apiErr.StatusCode != http.StatusNotFound {
			return nil, userErr
		}
	}

	now := s.now()

	if existingUser != nil {
		var memberWithUser *models.OrganizationMemberWithUser
		if txErr := s.runInTx(ctx, func(tx chi_repository.Tx) error {
			membersRepo := s.organizationMembersRepo.WithTx(tx)
			members, listErr := membersRepo.ListByOrganizationID(ctx, req.OrganizationID)
			if listErr != nil {
				return listErr
			}
			for _, member := range *members {
				if member.UserID == existingUser.ID {
					return chi_error.NewConflictError(errors.New("user is already a member"), "UserAlreadyMember", nil)
				}
			}

			billingAccount, billingErr := s.billingAccountsRepo.GetOrganizationBillingAccountByOrganizationID(ctx, req.OrganizationID)
			if billingErr != nil {
				return billingErr
			}
			if platform_subscription.OrganizationAtSeatLimit(len(*members), billingAccount.SubscriptionSeats) {
				return chi_error.NewForbiddenError(errors.New("organization seats limit reached"), "OrganizationSeatsLimit", nil)
			}

			memberID, genErr := s.generateID()
			if genErr != nil {
				return genErr
			}

			member := &models.OrganizationMember{
				ModelBase: chi_types.ModelBase{
					ID:        memberID,
					CreatedAt: now,
				},
				OrganizationID: org.ID,
				UserID:         existingUser.ID,
				RoleID:         role.ID,
			}

			if err := membersRepo.CreateOrganizationMember(ctx, member); err != nil {
				return err
			}

			var fetchErr error
			memberWithUser, fetchErr = membersRepo.GetOrganizationMemberByMembershipIDWithUser(ctx, org.ID.String(), member.ID.String())
			return fetchErr
		}); txErr != nil {
			return nil, txErr
		}

		if err := s.sessionCache.InvalidateSession(ctx, existingUser.ID.String()); err != nil {
			s.logger.WithContext(ctx).Error("failed to invalidate session", "error", err, "organizationId", org.ID.String())
		}

		return &CreateInvitationResponse{Member: memberWithUser}, nil
	}

	invitationID, err := s.generateID()
	if err != nil {
		return nil, err
	}

	inviteToken, err := s.generateToken()
	if err != nil {
		return nil, err
	}

	parsedInvitedByID, err := uuid.Parse(req.InvitedByID)
	if err != nil {
		return nil, chi_error.NewUnprocessableEntityError(errors.New("validation failed"), "", err)
	}

	invitation := &models.Invitation{
		ModelBase: chi_types.ModelBase{
			ID:        invitationID,
			CreatedAt: now,
		},
		ExpiresAt:      now.Add(s.invitationTTL),
		OrganizationID: org.ID,
		Email:          emailLower,
		RoleID:         role.ID,
		InvitedByID:    uuid.NullUUID{UUID: parsedInvitedByID, Valid: true},
		InvitedByEmail: req.InvitedByEmail,
		TokenHash:      s.hashToken(inviteToken),
	}

	if err := s.invitationsRepo.CreateInvitation(ctx, invitation); err != nil {
		return nil, err
	}

	if err := s.sendInvitationEmail(ctx, req.Language, org.Name, emailLower, inviteToken); err != nil {
		return nil, err
	}

	return &CreateInvitationResponse{Invitation: invitation}, nil
}

func (s *service) RevokeInvitation(ctx context.Context, req *RevokeInvitationRequest, access *chi_types.AccessInfo) error {
	if err := s.validator.ValidateStruct(req); err != nil {
		return chi_error.NewUnprocessableEntityError(errors.New("validation failed"), "", err)
	}

	if _, err := s.orgsRepo.GetOrganizationByID(ctx, req.OrganizationID); err != nil {
		return err
	}

	billingAccount, err := s.billingAccountsRepo.GetOrganizationBillingAccountByOrganizationID(ctx, req.OrganizationID)
	if err != nil {
		return err
	}

	if err := s.authorizer.CheckOrganizationPermissionWithSubscription(access, billingAccount, constants.PERMISSION_MEMBERS_DELETE); err != nil {
		return err
	}

	invitation, err := s.invitationsRepo.GetInvitationByID(ctx, req.OrganizationID, req.InvitationID)
	if err != nil {
		return err
	}

	if invitation.RevokedAt != nil {
		return chi_error.NewUnprocessableEntityError(errors.New("invitation revoked"), "InvitationRevoked", nil)
	}
	if invitation.AcceptedAt != nil {
		return chi_error.NewUnprocessableEntityError(errors.New("invitation already accepted"), "InvitationAlreadyAccepted", nil)
	}
	if invitation.ExpiresAt.Before(s.now()) {
		return chi_error.NewUnprocessableEntityError(errors.New("invitation expired"), "InvitationExpired", nil)
	}

	now := s.now()
	invitation.RevokedAt = &now
	return s.invitationsRepo.UpdateInvitation(ctx, invitation)
}

func (s *service) ListInvitations(ctx context.Context, req *ListInvitationsRequest, access *chi_types.AccessInfo) (*[]models.Invitation, error) {
	if err := s.validator.ValidateStruct(req); err != nil {
		return nil, chi_error.NewUnprocessableEntityError(errors.New("validation failed"), "", err)
	}

	if _, err := s.orgsRepo.GetOrganizationByID(ctx, req.OrganizationID); err != nil {
		return nil, err
	}

	if err := s.authorizer.CheckOrganizationPermission(access, req.OrganizationID, constants.PERMISSION_MEMBERS_READ); err != nil {
		return nil, err
	}

	return s.invitationsRepo.ListInvitationsByOrganizationID(ctx, req.OrganizationID)
}

func (s *service) CleanupStaleInvitations(ctx context.Context) error {
	return s.invitationsRepo.CleanupStaleInvitations(ctx)
}

func (s *service) checkCreateInvitationPermission(ctx context.Context, access *chi_types.AccessInfo, organizationID string) error {
	billingAccount, err := s.billingAccountsRepo.GetOrganizationBillingAccountByOrganizationID(ctx, organizationID)
	if err != nil {
		return err
	}

	if err := s.authorizer.CheckOrganizationPermissionWithSubscription(access, billingAccount, constants.PERMISSION_MEMBERS_WRITE); err != nil {
		return err
	}

	members, err := s.organizationMembersRepo.ListByOrganizationID(ctx, organizationID)
	if err != nil {
		return err
	}
	if platform_subscription.OrganizationAtSeatLimit(len(*members), billingAccount.SubscriptionSeats) {
		return chi_error.NewForbiddenError(errors.New("organization seats limit reached"), "OrganizationSeatsLimit", nil)
	}

	return nil
}

func (s *service) ensureNoPendingInvitation(ctx context.Context, organizationID, email string) error {
	invitations, err := s.invitationsRepo.ListInvitationsByOrganizationID(ctx, organizationID)
	if err != nil {
		return err
	}
	emailLower := strings.ToLower(email)
	now := s.now()
	for _, invitation := range *invitations {
		if !strings.EqualFold(invitation.Email, emailLower) {
			continue
		}
		if invitation.ExpiresAt.Before(now) {
			continue
		}
		return chi_error.NewConflictError(errors.New("invitation already pending"), "InvitationAlreadyPending", nil)
	}
	return nil
}

func normalizeLanguage(language string) string {
	lang := strings.ToLower(strings.TrimSpace(language))
	for _, supported := range constants.SUPPORTED_LANGUAGES {
		if lang == supported {
			return lang
		}
	}
	return constants.DEFAULT_LANGUAGE
}

func (s *service) sendInvitationEmail(ctx context.Context, language, orgName, email, inviteToken string) error {
	lang := normalizeLanguage(language)
	tokenTTLDays := max(1, int(s.invitationTTL.Hours())/24)

	body, err := s.emailTemplates.Render("invitation", map[string]any{
		"Lang":         lang,
		"Title":        s.localizer.Translate(lang, "email.invitation.title", nil),
		"Greeting":     s.localizer.Translate(lang, "email.invitation.greeting", nil),
		"Content":      s.localizer.Translate(lang, "email.invitation.content", nil),
		"ButtonText":   s.localizer.Translate(lang, "email.invitation.button", nil),
		"FooterIgnore": s.localizer.Translate(lang, "email.invitation.footer.ignore", nil),
		"FooterExpiry": s.localizer.Translate(lang, "email.invitation.footer.expiry", map[string]any{
			"TokenTTLDays": strconv.Itoa(tokenTTLDays),
		}),
		"C2ALink": fmt.Sprintf("%s/signup?invitationToken=%s", s.appURL, inviteToken),
	})
	if err != nil {
		return err
	}

	subject := s.localizer.Translate(lang, "email.invitation.subject", map[string]any{
		"OrganizationName": orgName,
	})
	return s.emailSender.Send(ctx, chi_aws_ses.SESEmailDataPayload{
		To:      email,
		Subject: subject,
		HTML:    body,
	})
}
