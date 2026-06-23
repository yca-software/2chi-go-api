package invitation_service_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/yca-software/2chi-go-api/internals/constants"
	"github.com/yca-software/2chi-go-api/internals/models"
	"github.com/yca-software/2chi-go-api/internals/packages/authz"
	"github.com/yca-software/2chi-go-api/internals/packages/testutil"
	"github.com/yca-software/2chi-go-api/internals/repositories"
	billing_account_repository "github.com/yca-software/2chi-go-api/internals/repositories/billing_account"
	invitation_repository "github.com/yca-software/2chi-go-api/internals/repositories/invitation"
	organization_member_repository "github.com/yca-software/2chi-go-api/internals/repositories/org_member"
	organization_repository "github.com/yca-software/2chi-go-api/internals/repositories/organization"
	role_repository "github.com/yca-software/2chi-go-api/internals/repositories/role"
	user_repository "github.com/yca-software/2chi-go-api/internals/repositories/user"
	invitation_service "github.com/yca-software/2chi-go-api/internals/services/invitation"
	chi_aws_ses "github.com/yca-software/2chi-go-aws/ses"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_localizer "github.com/yca-software/2chi-go-localizer"
	chi_logger "github.com/yca-software/2chi-go-logger"
	chi_template "github.com/yca-software/2chi-go-template"
	chi_repository "github.com/yca-software/2chi-go-repository"
	chi_token "github.com/yca-software/2chi-go-token"
	chi_types "github.com/yca-software/2chi-go-types"
	chi_validator "github.com/yca-software/2chi-go-validator"
)

var testTokenHasher = chi_token.NewHasher("test-pepper")

type InvitationServiceSuite struct {
	suite.Suite
	ctx                 context.Context
	now                 time.Time
	orgID               uuid.UUID
	invitesRepo         *invitation_repository.MockRepository
	orgsRepo            *organization_repository.MockRepository
	membersRepo         *organization_member_repository.MockRepository
	billingAccountsRepo *billing_account_repository.MockRepository
	usersRepo           *user_repository.MockRepository
	rolesRepo           *role_repository.MockRepository
	svc                 invitation_service.Service
}

func TestInvitationServiceSuite(t *testing.T) {
	suite.Run(t, new(InvitationServiceSuite))
}

func (s *InvitationServiceSuite) SetupTest() {
	s.ctx = context.Background()
	s.now = time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	s.orgID = uuid.New()
	s.invitesRepo = &invitation_repository.MockRepository{}
	s.orgsRepo = &organization_repository.MockRepository{}
	s.membersRepo = &organization_member_repository.MockRepository{}
	s.billingAccountsRepo = &billing_account_repository.MockRepository{}
	s.usersRepo = &user_repository.MockRepository{}
	s.rolesRepo = &role_repository.MockRepository{}

	s.svc = invitation_service.New(invitation_service.Dependencies{
		InvitationTTL: constants.INVITATION_TOKEN_TTL,
		AppURL:        "https://app.example.com",
		GenerateID:    uuid.NewV7,
		Now:           func() time.Time { return s.now },
		Validator:     chi_validator.New(),
		Logger:        mockLogger(),
		GenerateToken: chi_token.GenerateOpaqueToken,
		HashToken:     testTokenHasher.Hash,
		Authorizer:    authz.NewAuthorizer(func() time.Time { return s.now }),
		Repositories: &repositories.Repositories{
			Invitations:                 s.invitesRepo,
			Organizations:               s.orgsRepo,
			OrganizationMembers:         s.membersRepo,
			OrganizationBillingAccounts: s.billingAccountsRepo,
			Users:                       s.usersRepo,
			Roles:                       s.rolesRepo,
		},
		RunInTx:      inlineRunInTx,
		SessionCache: authz.NewTestSessionCache(s.T(), constants.ACCESS_TOKEN_TTL),
	})
}

func (s *InvitationServiceSuite) organization() *models.Organization {
	return &models.Organization{
		ModelBaseWithArchive: chi_types.ModelBaseWithArchive{
			ModelBase: chi_types.ModelBase{ID: s.orgID},
		},
		Name: "Acme",
	}
}

func (s *InvitationServiceSuite) basicBillingAccount(seats int) *models.OrganizationBillingAccount {
	expiresAt := s.now.Add(24 * time.Hour)
	return &models.OrganizationBillingAccount{
		OrganizationID:              s.orgID,
		Provider:                    constants.BILLING_PROVIDER_PADDLE,
		SubscriptionTier:            constants.TIER_BASIC,
		SubscriptionSeats:           seats,
		SubscriptionExpiresAt:       &expiresAt,
		SubscriptionPaymentInterval: constants.PAYMENT_INTERVAL_MONTHLY,
	}
}

func (s *InvitationServiceSuite) expectPaidOrg() {
	s.orgsRepo.On("GetByID", s.ctx, s.orgID.String()).
		Return(s.organization(), nil).Once()
}

func (s *InvitationServiceSuite) expectBasicBillingAccount() *mock.Call {
	return s.billingAccountsRepo.On("GetByOrganizationID", s.ctx, s.orgID.String()).
		Return(s.basicBillingAccount(10), nil)
}

func (s *InvitationServiceSuite) expectRole(roleID uuid.UUID) {
	s.rolesRepo.On("GetByID", s.ctx, s.orgID.String(), roleID.String()).
		Return(&models.Role{
			OrganizationID: s.orgID,
			Name:           "Member",
		}, nil).Once()
}

func (s *InvitationServiceSuite) expectNoPendingInvites() {
	s.invitesRepo.On("ListByOrganizationID", s.ctx, s.orgID.String()).
		Return(&[]models.Invitation{}, nil).Once()
}

func (s *InvitationServiceSuite) invitation(invID uuid.UUID) *models.Invitation {
	return &models.Invitation{
		OrganizationID: s.orgID,
		Email:          "invitee@example.com",
		ExpiresAt:      s.now.Add(24 * time.Hour),
	}
}

func (s *InvitationServiceSuite) writeAccess() *chi_types.AccessInfo {
	return &chi_types.AccessInfo{
		Type:      chi_types.AccessTypeUser,
		SubjectID: uuid.New(),
		Email:     "admin@example.com",
		Roles: []chi_types.JWTAccessTokenPermissionData{{
			OrganizationID: s.orgID,
			Permissions:    []string{constants.PERMISSION_MEMBERS_WRITE},
		}},
	}
}

func (s *InvitationServiceSuite) TestCreateInvitation_Validation_InvalidEmail() {
	resp, err := s.svc.Create(s.ctx, &invitation_service.CreateRequest{
		OrganizationID: s.orgID.String(),
		Email:          "not-an-email",
		RoleID:         uuid.New().String(),
		InvitedByID:    uuid.New().String(),
		InvitedByEmail: "admin@example.com",
		Language:       "en",
	}, s.writeAccess())
	s.Error(err)
	s.Nil(resp)
}

func (s *InvitationServiceSuite) TestCreateInvitation_UserAlreadyMember() {
	userID := uuid.New()
	roleID := uuid.New()
	s.expectPaidOrg()
	s.expectBasicBillingAccount().Twice()
	s.membersRepo.On("ListByOrganizationID", s.ctx, s.orgID.String()).
		Return(&[]models.OrganizationMemberWithUser{{
			OrganizationMember: models.OrganizationMember{UserID: userID},
		}}, nil).Once()
	s.expectRole(roleID)
	s.expectNoPendingInvites()
	s.usersRepo.On("GetByEmail", s.ctx, "member@example.com").
		Return(&models.User{
			ModelBaseWithArchive: chi_types.ModelBaseWithArchive{
				ModelBase: chi_types.ModelBase{ID: userID},
			},
			Email: "member@example.com",
		}, nil).Once()
	s.membersRepo.On("WithTx", mock.Anything).Return(s.membersRepo)
	s.membersRepo.On("ListByOrganizationID", s.ctx, s.orgID.String()).
		Return(&[]models.OrganizationMemberWithUser{{
			OrganizationMember: models.OrganizationMember{UserID: userID},
		}}, nil).Once()

	resp, err := s.svc.Create(s.ctx, &invitation_service.CreateRequest{
		OrganizationID: s.orgID.String(),
		Email:          "member@example.com",
		RoleID:         roleID.String(),
		InvitedByID:    uuid.New().String(),
		InvitedByEmail: "admin@example.com",
		Language:       "en",
	}, s.writeAccess())
	s.Error(err)
	s.Nil(resp)
	if apiErr, ok := chi_error.AsError(err); ok {
		s.Equal("UserAlreadyMember", apiErr.ErrorCode)
	}
}

func (s *InvitationServiceSuite) TestRevokeInvitation_AlreadyAccepted() {
	invID := uuid.New()
	accepted := s.now.Add(-time.Hour)
	s.expectPaidOrg()
	s.expectBasicBillingAccount().Once()
	inv := s.invitation(invID)
	inv.AcceptedAt = &accepted
	s.invitesRepo.On("GetByID", s.ctx, s.orgID.String(), invID.String()).Return(inv, nil).Once()

	access := s.writeAccess()
	access.Roles[0].Permissions = []string{constants.PERMISSION_MEMBERS_DELETE}

	err := s.svc.Revoke(s.ctx, &invitation_service.RevokeRequest{
		OrganizationID: s.orgID.String(),
		InvitationID:   invID.String(),
	}, access)
	s.Error(err)
	if apiErr, ok := chi_error.AsError(err); ok {
		s.Equal("InvitationAlreadyAccepted", apiErr.ErrorCode)
	}
}

func (s *InvitationServiceSuite) TestListInvitations_ForbiddenWithoutReadPermission() {
	s.expectPaidOrg()

	_, err := s.svc.List(s.ctx, &invitation_service.ListRequest{
		OrganizationID: s.orgID.String(),
	}, s.writeAccess())
	s.Error(err)
}

func (s *InvitationServiceSuite) TestCleanupStaleInvitations() {
	s.invitesRepo.On("CleanupStale", s.ctx).Return(nil).Once()
	s.NoError(s.svc.CleanupStale(s.ctx))
}

func (s *InvitationServiceSuite) TestRevokeInvitation_Success() {
	invID := uuid.New()
	expires := s.now.Add(24 * time.Hour)
	s.expectPaidOrg()
	s.expectBasicBillingAccount().Once()
	inv := s.invitation(invID)
	inv.ExpiresAt = expires
	s.invitesRepo.On("GetByID", s.ctx, s.orgID.String(), invID.String()).Return(inv, nil).Once()
	s.invitesRepo.On("Update", s.ctx, mock.AnythingOfType("*models.Invitation")).Return(nil).Once()

	access := s.writeAccess()
	access.Roles[0].Permissions = []string{constants.PERMISSION_MEMBERS_DELETE}

	err := s.svc.Revoke(s.ctx, &invitation_service.RevokeRequest{
		OrganizationID: s.orgID.String(),
		InvitationID:   invID.String(),
	}, access)
	s.NoError(err)
}

func (s *InvitationServiceSuite) TestListInvitations_Success() {
	invitations := []models.Invitation{*s.invitation(uuid.New())}
	s.expectPaidOrg()
	s.invitesRepo.On("ListByOrganizationID", s.ctx, s.orgID.String()).Return(&invitations, nil).Once()

	access := s.writeAccess()
	access.Roles[0].Permissions = []string{constants.PERMISSION_MEMBERS_READ}

	result, err := s.svc.List(s.ctx, &invitation_service.ListRequest{
		OrganizationID: s.orgID.String(),
	}, access)
	s.Require().NoError(err)
	s.Len(*result, 1)
}

func (s *InvitationServiceSuite) TestCreateInvitation_EmailBodyContainsExpiryDays() {
	roleID := uuid.New()
	emailSender := &chi_aws_ses.MockSES{}
	var capturedHTML string
	var capturedSubject string
	emailSender.On("Send", s.ctx, mock.Anything).
		Run(func(args mock.Arguments) {
			payload := args.Get(1).(chi_aws_ses.SESEmailDataPayload)
			capturedHTML = payload.HTML
			capturedSubject = payload.Subject
		}).
		Return(nil).
		Once()

	svc := invitation_service.New(invitation_service.Dependencies{
		InvitationTTL: constants.INVITATION_TOKEN_TTL,
		AppURL:        "https://app.example.com",
		GenerateID:    uuid.NewV7,
		Now:           func() time.Time { return s.now },
		Validator:     chi_validator.New(),
		Logger:        mockLogger(),
		GenerateToken: chi_token.GenerateOpaqueToken,
		HashToken:     testTokenHasher.Hash,
		Authorizer:    authz.NewAuthorizer(func() time.Time { return s.now }),
		Repositories: &repositories.Repositories{
			Invitations:                 s.invitesRepo,
			Organizations:               s.orgsRepo,
			OrganizationMembers:         s.membersRepo,
			OrganizationBillingAccounts: s.billingAccountsRepo,
			Users:                       s.usersRepo,
			Roles:                       s.rolesRepo,
		},
		RunInTx:        inlineRunInTx,
		SessionCache:   authz.NewTestSessionCache(s.T(), constants.ACCESS_TOKEN_TTL),
		Localizer:      chi_localizer.New(constants.SUPPORTED_LANGUAGES, constants.DEFAULT_LANGUAGE, testutil.LocalesDir()),
		EmailSender:    emailSender,
		EmailTemplates: chi_template.NewHTML(testutil.TemplatesDir()),
	})

	s.expectPaidOrg()
	s.expectBasicBillingAccount().Twice()
	s.membersRepo.On("ListByOrganizationID", s.ctx, s.orgID.String()).
		Return(&[]models.OrganizationMemberWithUser{}, nil).Once()
	s.expectRole(roleID)
	s.expectNoPendingInvites()
	s.usersRepo.On("GetByEmail", s.ctx, "invitee@example.com").
		Return(nil, chi_error.NewNotFoundError(nil, "NotFound", nil)).Once()
	s.invitesRepo.On("Create", s.ctx, mock.AnythingOfType("*models.Invitation")).Return(nil).Once()

	resp, err := svc.Create(s.ctx, &invitation_service.CreateRequest{
		OrganizationID: s.orgID.String(),
		Email:          "invitee@example.com",
		RoleID:         roleID.String(),
		InvitedByID:    uuid.New().String(),
		InvitedByEmail: "admin@example.com",
		Language:       "en",
	}, s.writeAccess())
	s.Require().NoError(err)
	s.NotNil(resp.Invitation)
	s.Contains(capturedHTML, "This invitation expires in 7 days.")
	s.Contains(capturedSubject, "Acme")
	emailSender.AssertExpectations(s.T())
}

func (s *InvitationServiceSuite) TestCreateInvitation_AddExistingUser_Success() {
	roleID := uuid.New()
	existingUserID := uuid.New()
	s.expectPaidOrg()
	s.expectBasicBillingAccount().Twice()
	s.membersRepo.On("ListByOrganizationID", s.ctx, s.orgID.String()).
		Return(&[]models.OrganizationMemberWithUser{}, nil).Once()
	s.membersRepo.On("WithTx", mock.Anything).Return(s.membersRepo)
	s.expectRole(roleID)
	s.expectNoPendingInvites()
	s.usersRepo.On("GetByEmail", s.ctx, "existing@example.com").
		Return(&models.User{
			ModelBaseWithArchive: chi_types.ModelBaseWithArchive{
				ModelBase: chi_types.ModelBase{ID: existingUserID},
			},
			Email: "existing@example.com",
		}, nil).Once()
	s.membersRepo.On("ListByOrganizationID", s.ctx, s.orgID.String()).
		Return(&[]models.OrganizationMemberWithUser{}, nil).Once()
	s.membersRepo.On("Create", s.ctx, mock.AnythingOfType("*models.OrganizationMember")).Return(nil).Once()
	s.membersRepo.On("GetByMemberIDWithUser", s.ctx, s.orgID.String(), mock.Anything).
		Return(&models.OrganizationMemberWithUser{
			OrganizationMember: models.OrganizationMember{
				OrganizationID: s.orgID,
				UserID:         existingUserID,
				RoleID:         roleID,
			},
		}, nil).Once()

	resp, err := s.svc.Create(s.ctx, &invitation_service.CreateRequest{
		Email:          "existing@example.com",
		OrganizationID: s.orgID.String(),
		RoleID:         roleID.String(),
		InvitedByID:    uuid.New().String(),
		InvitedByEmail: "admin@example.com",
		Language:       "en",
	}, s.writeAccess())
	s.Require().NoError(err)
	s.NotNil(resp.Member)
}

func inlineRunInTx(_ context.Context, fn func(chi_repository.Tx) error) error {
	return fn(nil)
}

func mockLogger() chi_logger.Logger {
	m := new(chi_logger.MockLogger)
	for n := 0; n <= 8; n++ {
		args := make([]any, n)
		for i := range args {
			args[i] = mock.Anything
		}
		if n == 0 {
			m.On("With").Return(m).Maybe()
			continue
		}
		m.On("With", args...).Return(m).Maybe()
	}
	m.On("WithContext", mock.Anything).Return(m).Maybe()
	for _, method := range []string{"Debug", "Info", "Warn", "Error"} {
		for n := 0; n <= 8; n++ {
			args := make([]any, n+1)
			for i := range args {
				args[i] = mock.Anything
			}
			m.On(method, args...).Return().Maybe()
		}
	}
	return m
}
