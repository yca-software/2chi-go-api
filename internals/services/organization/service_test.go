package organization_service_test

import (
	"context"
	"errors"
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
	organization_member_repository "github.com/yca-software/2chi-go-api/internals/repositories/org_member"
	organization_repository "github.com/yca-software/2chi-go-api/internals/repositories/organization"
	role_repository "github.com/yca-software/2chi-go-api/internals/repositories/role"
	user_repository "github.com/yca-software/2chi-go-api/internals/repositories/user"
	audit_service "github.com/yca-software/2chi-go-api/internals/services/audit"
	billing_service "github.com/yca-software/2chi-go-api/internals/services/billing"
	invitation_service "github.com/yca-software/2chi-go-api/internals/services/invitation"
	location_service "github.com/yca-software/2chi-go-api/internals/services/location"
	organization_service "github.com/yca-software/2chi-go-api/internals/services/organization"
	chi_archive "github.com/yca-software/2chi-go-archive"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_google "github.com/yca-software/2chi-go-google/maps"
	chi_logger "github.com/yca-software/2chi-go-logger"
	chi_types "github.com/yca-software/2chi-go-types"
	chi_validator "github.com/yca-software/2chi-go-validator"
)

type OrganizationServiceSuite struct {
	suite.Suite
	ctx             context.Context
	now             time.Time
	userID          uuid.UUID
	orgRepo         *organization_repository.MockRepository
	orgMembersRepo  *organization_member_repository.MockRepository
	billingAccounts *billing_account_repository.MockRepository
	rolesRepo       *role_repository.MockRepository
	usersRepo       *user_repository.MockRepository
	locationSvc     *location_service.MockService
	auditSvc        *audit_service.MockService
	invitationsSvc  *invitation_service.MockService
	billingSvc      *billing_service.MockService
	sessionCache    *authz.SessionCache
	svc             organization_service.Service
}

func TestOrganizationServiceSuite(t *testing.T) {
	suite.Run(t, new(OrganizationServiceSuite))
}

func (s *OrganizationServiceSuite) SetupTest() {
	s.ctx = context.Background()
	s.now = time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	s.userID = uuid.MustParse("018f1234-5678-7abc-8def-012345678901")

	s.orgRepo = &organization_repository.MockRepository{}
	s.orgMembersRepo = &organization_member_repository.MockRepository{}
	s.billingAccounts = &billing_account_repository.MockRepository{}
	s.rolesRepo = &role_repository.MockRepository{}
	s.usersRepo = &user_repository.MockRepository{}
	s.locationSvc = &location_service.MockService{}
	s.auditSvc = &audit_service.MockService{}
	s.billingSvc = &billing_service.MockService{}
	s.invitationsSvc = &invitation_service.MockService{}
	s.sessionCache = authz.NewTestSessionCache(s.T(), time.Hour)

	s.svc = organization_service.New(organization_service.Dependencies{
		GenerateID: uuid.NewV7,
		Now:        func() time.Time { return s.now },
		Validator:  chi_validator.New(),
		Logger:     mockLogger(),
		Authorizer: authz.NewAuthorizer(func() time.Time { return s.now }),
		Repositories: &repositories.Repositories{
			Organizations:               s.orgRepo,
			OrganizationMembers:         s.orgMembersRepo,
			OrganizationBillingAccounts: s.billingAccounts,
			Roles:                       s.rolesRepo,
			Users:                       s.usersRepo,
		},
		RunInTx:            testutil.InlineRunInTx,
		SessionCache:       s.sessionCache,
		AuditService:       s.auditSvc,
		LocationService:    s.locationSvc,
		BillingService:     s.billingSvc,
		InvitationsService: s.invitationsSvc,
	})
}

func (s *OrganizationServiceSuite) userAccess() *chi_types.AccessInfo {
	return &chi_types.AccessInfo{
		Type:      chi_types.AccessTypeUser,
		SubjectID: s.userID,
		Email:     "owner@example.com",
	}
}

func (s *OrganizationServiceSuite) adminAccess() *chi_types.AccessInfo {
	return &chi_types.AccessInfo{
		Type:      chi_types.AccessTypeUser,
		SubjectID: s.userID,
		Email:     "admin@example.com",
		IsAdmin:   true,
	}
}

func (s *OrganizationServiceSuite) orgAccess(orgID uuid.UUID, perms ...string) *chi_types.AccessInfo {
	return &chi_types.AccessInfo{
		Type:      chi_types.AccessTypeUser,
		SubjectID: s.userID,
		Email:     "owner@example.com",
		Roles: []chi_types.JWTAccessTokenPermissionData{{
			OrganizationID: orgID,
			Permissions:    perms,
		}},
	}
}

func (s *OrganizationServiceSuite) organization(orgID uuid.UUID, name string) *models.Organization {
	return &models.Organization{
		ModelBaseWithArchive: chi_types.ModelBaseWithArchive{
			ModelBase: chi_types.ModelBase{ID: orgID},
		},
		Name: name,
	}
}

func (s *OrganizationServiceSuite) basicBillingAccount(orgID uuid.UUID) *models.OrganizationBillingAccount {
	expiresAt := s.now.Add(24 * time.Hour)
	return &models.OrganizationBillingAccount{
		OrganizationID:              orgID,
		Provider:                    constants.BILLING_PROVIDER_PADDLE,
		ProviderCustomerID:          "ctm_1",
		SubscriptionTier:            constants.TIER_BASIC,
		SubscriptionSeats:           constants.SUBSCRIPTION_TYPE_SEATS_INCLUDED_BASIC,
		SubscriptionExpiresAt:       &expiresAt,
		SubscriptionPaymentInterval: constants.PAYMENT_INTERVAL_MONTHLY,
	}
}

func (s *OrganizationServiceSuite) organizationMember(memberID, orgID, userID, roleID uuid.UUID) *models.OrganizationMember {
	return &models.OrganizationMember{
		OrganizationID: orgID,
		UserID:         userID,
		RoleID:         roleID,
	}
}

func (s *OrganizationServiceSuite) role(roleID, orgID uuid.UUID) *models.Role {
	return &models.Role{
		OrganizationID: orgID,
		Name:           "Member",
	}
}

func (s *OrganizationServiceSuite) expectBasicBillingAccount(orgID uuid.UUID) *mock.Call {
	return s.billingAccounts.On("GetByOrganizationID", s.ctx, orgID.String()).
		Return(s.basicBillingAccount(orgID), nil)
}

func (s *OrganizationServiceSuite) setupCreateOrgTxMocks() {
	s.orgRepo.On("WithTx", mock.Anything).Return(s.orgRepo)
	s.billingAccounts.On("WithTx", mock.Anything).Return(s.billingAccounts)
	s.rolesRepo.On("WithTx", mock.Anything).Return(s.rolesRepo)
	s.orgMembersRepo.On("WithTx", mock.Anything).Return(s.orgMembersRepo)
}

func (s *OrganizationServiceSuite) TestCreateOrganization_Validation_MissingName() {
	resp, err := s.svc.CreateOrganization(s.ctx, &organization_service.CreateOrganizationRequest{
		Name:    "",
		PlaceID: "place_1",
	}, s.userAccess())
	s.Error(err)
	s.Nil(resp)
}

func (s *OrganizationServiceSuite) TestCreateOrganization_APIKeyForbidden() {
	access := &chi_types.AccessInfo{Type: chi_types.AccessTypeAPIKey, SubjectID: s.userID}
	resp, err := s.svc.CreateOrganization(s.ctx, &organization_service.CreateOrganizationRequest{
		Name:         "Acme",
		PlaceID:      "place_1",
		BillingEmail: "billing@example.com",
	}, access)
	s.Error(err)
	s.Nil(resp)
	if apiErr, ok := chi_error.AsError(err); ok {
		s.Equal("APICannotCreateOrganization", apiErr.ErrorCode)
	}
}

func (s *OrganizationServiceSuite) TestCreateOrganization_Success() {
	placeID := "ChIJtest"
	s.locationSvc.On("GetLocationData", s.ctx, placeID).Return(&chi_google.LocationData{
		Address:  "1 Main St",
		City:     "Oslo",
		Zip:      "0001",
		Country:  "NO",
		PlaceID:  placeID,
		Geo:      chi_google.Point{Lat: 59.9, Lng: 10.7},
		Timezone: "Europe/Oslo",
	}, nil).Once()

	s.billingSvc.On("CreateCustomer", s.ctx, mock.AnythingOfType("*billing_service.CreateCustomerInput")).Return("ctm_123", nil).Once()
	s.setupCreateOrgTxMocks()
	s.orgRepo.On("Create", s.ctx, mock.AnythingOfType("*models.Organization")).Return(nil).Once()
	s.billingAccounts.On("Create", s.ctx, mock.AnythingOfType("*models.OrganizationBillingAccount")).Return(nil).Once()
	s.rolesRepo.On("CreateMany", s.ctx, mock.AnythingOfType("*[]models.Role")).Return(nil).Once()
	s.orgMembersRepo.On("Create", s.ctx, mock.AnythingOfType("*models.OrganizationMember")).Return(nil).Once()
	s.auditSvc.On("Create", s.ctx, mock.Anything, mock.Anything).
		Return(&models.AuditLog{}, nil).Maybe()

	resp, err := s.svc.CreateOrganization(s.ctx, &organization_service.CreateOrganizationRequest{
		Name:         "  Acme Corp  ",
		PlaceID:      placeID,
		BillingEmail: "billing@example.com",
	}, s.userAccess())
	s.Require().NoError(err)
	s.Require().NotNil(resp)
	s.Equal("Acme Corp", resp.Organization.Name)
	s.Len(*resp.Roles, len(organization_service.DefaultRolesToCreateForOrganization))
	s.Equal(s.userID, resp.Member.UserID)
}

func (s *OrganizationServiceSuite) TestCreateOrganization_DBFailure() {
	placeID := "ChIJfail"
	dbErr := errors.New("db down")

	s.locationSvc.On("GetLocationData", s.ctx, placeID).Return(&chi_google.LocationData{
		PlaceID: placeID,
		Geo:     chi_google.Point{Lat: 1, Lng: 2},
	}, nil).Once()
	s.billingSvc.On("CreateCustomer", s.ctx, mock.AnythingOfType("*billing_service.CreateCustomerInput")).Return("ctm_123", nil).Once()
	s.setupCreateOrgTxMocks()
	s.orgRepo.On("Create", s.ctx, mock.AnythingOfType("*models.Organization")).Return(dbErr).Once()
	s.billingSvc.On("ReleaseProvisionedCustomer", s.ctx, mock.Anything, mock.Anything).Return(nil).Maybe()

	resp, err := s.svc.CreateOrganization(s.ctx, &organization_service.CreateOrganizationRequest{
		Name:         "Acme",
		PlaceID:      placeID,
		BillingEmail: "billing@example.com",
	}, s.userAccess())
	s.Error(err)
	s.Nil(resp)
	s.Equal(dbErr, err)
}

func (s *OrganizationServiceSuite) TestUpdateOrganization_DBFailure() {
	orgID := uuid.New()
	existing := s.organization(orgID, "Old")
	existing.PlaceID = "place_old"

	s.billingSvc.On("UpdateCustomer", s.ctx, mock.AnythingOfType("*billing_service.UpdateCustomerInput")).Return(nil).Twice()
	s.orgRepo.On("GetByID", s.ctx, orgID.String()).Return(existing, nil).Once()
	s.expectBasicBillingAccount(orgID).Once()
	s.orgRepo.On("WithTx", mock.Anything).Return(s.orgRepo)
	s.orgRepo.On("Update", s.ctx, mock.AnythingOfType("*models.Organization")).
		Return(errors.New("update failed")).Once()

	access := s.orgAccess(orgID, constants.PERMISSION_ORG_WRITE)
	_, err := s.svc.UpdateOrganization(s.ctx, &organization_service.UpdateOrganizationRequest{
		OrganizationID: orgID.String(),
		Name:           "New Name",
		PlaceID:        "place_old",
	}, access)
	s.Error(err)
}

func (s *OrganizationServiceSuite) TestGetOrganization_ForbiddenWithoutPermission() {
	orgID := uuid.New()
	s.orgRepo.On("GetByID", s.ctx, orgID.String()).
		Return(s.organization(orgID, "Acme"), nil).Once()

	_, err := s.svc.GetOrganization(s.ctx, &organization_service.GetOrganizationRequest{
		OrganizationID: orgID.String(),
	}, s.userAccess())
	s.Error(err)
}

func (s *OrganizationServiceSuite) TestCleanupArchivedOrganizations_DelegatesToRepo() {
	s.orgRepo.On("CleanupArchived", s.ctx).Return(nil).Once()
	s.NoError(s.svc.CleanupArchivedOrganizations(s.ctx))
}

func (s *OrganizationServiceSuite) TestGetOrganization_Success() {
	orgID := uuid.New()
	org := s.organization(orgID, "Acme")
	s.orgRepo.On("GetByID", s.ctx, orgID.String()).Return(org, nil).Once()

	result, err := s.svc.GetOrganization(s.ctx, &organization_service.GetOrganizationRequest{
		OrganizationID: orgID.String(),
	}, s.orgAccess(orgID, constants.PERMISSION_ORG_READ))
	s.Require().NoError(err)
	s.Equal(orgID, result.ID)
}

func (s *OrganizationServiceSuite) TestUpdateOrganization_Success() {
	orgID := uuid.New()
	existing := s.organization(orgID, "Old")
	existing.PlaceID = "place_old"

	s.orgRepo.On("GetByID", s.ctx, orgID.String()).Return(existing, nil).Once()
	s.expectBasicBillingAccount(orgID).Once()
	s.billingSvc.On("UpdateCustomer", s.ctx, mock.AnythingOfType("*billing_service.UpdateCustomerInput")).Return(nil).Once()
	s.orgRepo.On("WithTx", mock.Anything).Return(s.orgRepo)
	s.orgRepo.On("Update", s.ctx, mock.AnythingOfType("*models.Organization")).Return(nil).Once()
	s.auditSvc.On("Create", s.ctx, mock.Anything, mock.Anything).Return(&models.AuditLog{}, nil).Maybe()

	result, err := s.svc.UpdateOrganization(s.ctx, &organization_service.UpdateOrganizationRequest{
		OrganizationID: orgID.String(),
		Name:           "New",
		PlaceID:        "place_old",
	}, s.orgAccess(orgID, constants.PERMISSION_ORG_WRITE))
	s.Require().NoError(err)
	s.Equal("New", result.Name)
}

func (s *OrganizationServiceSuite) TestUpdateOrganizationSubscription_RequiresAdmin() {
	_, err := s.svc.UpdateOrganizationSubscription(s.ctx, &organization_service.UpdateOrganizationSubscriptionRequest{
		OrganizationID:        uuid.New().String(),
		CustomSubscription:    true,
		SubscriptionType:      constants.TIER_BASIC,
		SubscriptionSeats:     3,
		SubscriptionExpiresAt: s.now.Add(30 * 24 * time.Hour),
	}, s.userAccess())
	s.Error(err)
}

func (s *OrganizationServiceSuite) TestUpdateOrganizationSubscription_Success() {
	orgID := uuid.New()
	expires := s.now.Add(30 * 24 * time.Hour)
	existingAccount := s.basicBillingAccount(orgID)

	s.orgRepo.On("GetByID", s.ctx, orgID.String()).
		Return(s.organization(orgID, "Acme"), nil).Once()
	s.orgMembersRepo.On("ListByOrganizationID", s.ctx, orgID.String()).
		Return(&[]models.OrganizationMemberWithUser{}, nil).Once()
	s.billingAccounts.On("GetByOrganizationID", s.ctx, orgID.String()).Return(existingAccount, nil).Once()
	s.billingAccounts.On("WithTx", mock.Anything).Return(s.billingAccounts)
	s.billingAccounts.On("Update", s.ctx, mock.AnythingOfType("*models.OrganizationBillingAccount")).Return(nil).Once()

	result, err := s.svc.UpdateOrganizationSubscription(s.ctx, &organization_service.UpdateOrganizationSubscriptionRequest{
		OrganizationID:        orgID.String(),
		CustomSubscription:    true,
		SubscriptionType:      constants.TIER_PRO,
		SubscriptionSeats:     25,
		SubscriptionExpiresAt: expires,
	}, s.adminAccess())
	s.Require().NoError(err)
	s.Equal(constants.TIER_PRO, result.SubscriptionTier)
}

func (s *OrganizationServiceSuite) TestUpdateOrganizationSubscription_RejectsSeatReductionBelowMemberCount() {
	orgID := uuid.New()
	members := make([]models.OrganizationMemberWithUser, 6)
	s.orgRepo.On("GetByID", s.ctx, orgID.String()).
		Return(s.organization(orgID, "Acme"), nil).Once()
	s.orgMembersRepo.On("ListByOrganizationID", s.ctx, orgID.String()).Return(&members, nil).Once()

	_, err := s.svc.UpdateOrganizationSubscription(s.ctx, &organization_service.UpdateOrganizationSubscriptionRequest{
		OrganizationID:        orgID.String(),
		CustomSubscription:    true,
		SubscriptionType:      constants.TIER_BASIC,
		SubscriptionSeats:     constants.SUBSCRIPTION_TYPE_SEATS_INCLUDED_BASIC,
		SubscriptionExpiresAt: s.now.Add(30 * 24 * time.Hour),
	}, s.adminAccess())
	s.Error(err)
	if apiErr, ok := chi_error.AsError(err); ok {
		s.Equal("OrganizationSeatsBelowMemberCount", apiErr.ErrorCode)
	}
}

func (s *OrganizationServiceSuite) TestArchiveOrganization_Success() {
	orgID := uuid.New()
	org := s.organization(orgID, "Acme")
	s.orgRepo.On("GetByID", s.ctx, orgID.String()).Return(org, nil).Once()
	s.orgRepo.On("Archive", s.ctx, org).Return(nil).Once()
	s.billingAccounts.On("GetByOrganizationID", s.ctx, orgID.String()).
		Return(s.basicBillingAccount(orgID), nil).Once()
	s.auditSvc.On("Create", s.ctx, mock.Anything, mock.Anything).Return(&models.AuditLog{}, nil).Maybe()

	err := s.svc.ArchiveOrganization(s.ctx, &organization_service.ArchiveOrganizationRequest{
		OrganizationID: orgID.String(),
	}, s.orgAccess(orgID, constants.PERMISSION_ORG_DELETE))
	s.NoError(err)
}

func (s *OrganizationServiceSuite) TestRestoreOrganization_NotArchived() {
	orgID := uuid.New()
	s.orgRepo.On("GetByIDIncludeArchived", s.ctx, orgID.String()).Return(&models.Organization{
		ModelBaseWithArchive: chi_types.ModelBaseWithArchive{
			DeletedAt: nil,
		},
	}, nil).Once()

	_, err := s.svc.RestoreOrganization(s.ctx, &organization_service.RestoreOrganizationRequest{
		OrganizationID: orgID.String(),
	}, s.adminAccess())
	s.Error(err)
}

func (s *OrganizationServiceSuite) TestDeleteOrganizationMember_CannotDeleteSelf() {
	orgID := uuid.New()
	memberID := uuid.New()
	roleID := uuid.New()
	s.orgRepo.On("GetByID", s.ctx, orgID.String()).
		Return(s.organization(orgID, "Acme"), nil).Once()
	s.expectBasicBillingAccount(orgID).Once()
	s.orgMembersRepo.On("GetByMemberID", s.ctx, orgID.String(), memberID.String()).
		Return(s.organizationMember(memberID, orgID, s.userID, roleID), nil).Once()

	access := s.orgAccess(orgID, constants.PERMISSION_MEMBERS_DELETE)
	err := s.svc.DeleteOrganizationMember(s.ctx, &organization_service.DeleteOrganizationMemberRequest{
		OrganizationID: orgID.String(),
		MemberID:       memberID.String(),
	}, access)
	s.Error(err)
}

func (s *OrganizationServiceSuite) TestGetArchivedOrganization_RequiresAdmin() {
	_, err := s.svc.GetArchivedOrganization(s.ctx, &organization_service.GetOrganizationRequest{
		OrganizationID: uuid.New().String(),
	}, s.userAccess())
	s.Error(err)
}

func (s *OrganizationServiceSuite) TestListOrganizations_Success() {
	orgs := []models.Organization{{ModelBaseWithArchive: chi_types.ModelBaseWithArchive{ModelBase: chi_types.ModelBase{ID: uuid.New()}}, Name: "Acme"}}
	s.orgRepo.On("Search", s.ctx, "", chi_archive.ArchiveFilterActive, 21, 1).Return(&orgs, nil).Once()

	resp, err := s.svc.ListOrganizations(s.ctx, &organization_service.ListOrganizationsRequest{
		ArchiveFilter: chi_archive.ArchiveFilterActive,
		Limit:         20,
		Offset:        1,
	}, s.adminAccess())
	s.Require().NoError(err)
	s.Len(resp.Items, 1)
}

func (s *OrganizationServiceSuite) TestListOrganizations_AcceptsZeroOffset() {
	orgs := []models.Organization{{ModelBaseWithArchive: chi_types.ModelBaseWithArchive{ModelBase: chi_types.ModelBase{ID: uuid.New()}}, Name: "Acme"}}
	s.orgRepo.On("Search", s.ctx, "", chi_archive.ArchiveFilterActive, 21, 0).Return(&orgs, nil).Once()

	resp, err := s.svc.ListOrganizations(s.ctx, &organization_service.ListOrganizationsRequest{
		ArchiveFilter: chi_archive.ArchiveFilterActive,
		Limit:         20,
		Offset:        0,
	}, s.adminAccess())
	s.Require().NoError(err)
	s.Len(resp.Items, 1)
}

func (s *OrganizationServiceSuite) TestListOrganizationMembers_Success() {
	orgID := uuid.New()
	members := []models.OrganizationMemberWithUser{{OrganizationMember: *s.organizationMember(uuid.New(), orgID, uuid.New(), uuid.New())}}
	s.orgRepo.On("GetByID", s.ctx, orgID.String()).
		Return(s.organization(orgID, "Acme"), nil).Once()
	s.orgMembersRepo.On("ListByOrganizationID", s.ctx, orgID.String()).Return(&members, nil).Once()

	result, err := s.svc.ListOrganizationMembers(s.ctx, &organization_service.ListOrganizationMembersRequest{
		OrganizationID: orgID.String(),
	}, s.orgAccess(orgID, constants.PERMISSION_MEMBERS_READ))
	s.Require().NoError(err)
	s.Len(*result, 1)
}

func (s *OrganizationServiceSuite) TestListOrganizationRolesForUser_Success() {
	targetID := uuid.New()
	roles := []models.OrganizationMemberWithOrganizationAndRole{}
	s.orgMembersRepo.On("ListByUserID", s.ctx, targetID.String()).Return(&roles, nil).Once()

	result, err := s.svc.ListOrganizationRolesForUser(s.ctx, &organization_service.ListOrganizationRolesForUserRequest{
		UserID: targetID.String(),
	}, &chi_types.AccessInfo{Type: chi_types.AccessTypeUser, SubjectID: targetID})
	s.Require().NoError(err)
	s.NotNil(result)
}

func (s *OrganizationServiceSuite) TestAdminCreateOrganization_RequiresAdmin() {
	_, err := s.svc.AdminCreateOrganization(s.ctx, &organization_service.AdminCreateOrganizationRequest{
		Name: "Acme", PlaceID: "p", BillingEmail: "b@example.com", OwnerEmail: "o@example.com",
		SubscriptionType: constants.TIER_BASIC, SubscriptionSeats: 1, Language: "en",
	}, s.userAccess())
	s.Error(err)
}

func (s *OrganizationServiceSuite) TestUpdateOrganizationMember_CannotUpdateSelf() {
	orgID := uuid.New()
	memberID := uuid.New()
	s.orgRepo.On("GetByID", s.ctx, orgID.String()).
		Return(s.organization(orgID, "Acme"), nil).Once()
	s.expectBasicBillingAccount(orgID).Once()
	s.orgMembersRepo.On("GetByMemberID", s.ctx, orgID.String(), memberID.String()).
		Return(s.organizationMember(memberID, orgID, s.userID, uuid.Nil), nil).Once()

	access := s.orgAccess(orgID, constants.PERMISSION_MEMBERS_WRITE)
	_, err := s.svc.UpdateOrganizationMember(s.ctx, &organization_service.UpdateOrganizationMemberRequest{
		OrganizationID: orgID.String(),
		MemberID:       memberID.String(),
		RoleID:         uuid.New().String(),
	}, access)
	s.Error(err)
}

func (s *OrganizationServiceSuite) TestRestoreOrganization_Success() {
	orgID := uuid.New()
	deleted := s.now.Add(-time.Hour)
	s.orgRepo.On("GetByIDIncludeArchived", s.ctx, orgID.String()).Return(&models.Organization{
		ModelBaseWithArchive: chi_types.ModelBaseWithArchive{
			DeletedAt: &deleted,
		},
	}, nil).Once()
	s.orgRepo.On("Restore", s.ctx, orgID.String()).Return(nil).Once()
	s.orgRepo.On("GetByID", s.ctx, orgID.String()).Return(&models.Organization{
		ModelBaseWithArchive: chi_types.ModelBaseWithArchive{ModelBase: chi_types.ModelBase{ID: orgID}},
	}, nil).Once()
	s.auditSvc.On("Create", s.ctx, mock.Anything, mock.Anything).Return(&models.AuditLog{}, nil).Maybe()

	result, err := s.svc.RestoreOrganization(s.ctx, &organization_service.RestoreOrganizationRequest{
		OrganizationID: orgID.String(),
	}, s.adminAccess())
	s.Require().NoError(err)
	s.Equal(orgID, result.ID)
}

func (s *OrganizationServiceSuite) TestGetArchivedOrganization_Success() {
	orgID := uuid.New()
	s.orgRepo.On("GetByIDIncludeArchived", s.ctx, orgID.String()).
		Return(&models.Organization{
			ModelBaseWithArchive: chi_types.ModelBaseWithArchive{ModelBase: chi_types.ModelBase{ID: orgID}},
		}, nil).Once()

	result, err := s.svc.GetArchivedOrganization(s.ctx, &organization_service.GetOrganizationRequest{
		OrganizationID: orgID.String(),
	}, s.adminAccess())
	s.Require().NoError(err)
	s.Equal(orgID, result.ID)
}

func (s *OrganizationServiceSuite) TestUpdateOrganizationMember_Success() {
	orgID := uuid.New()
	memberID := uuid.New()
	memberUserID := uuid.New()
	oldRoleID := uuid.New()
	newRoleID := uuid.New()

	s.orgRepo.On("GetByID", s.ctx, orgID.String()).
		Return(s.organization(orgID, "Acme"), nil).Once()
	s.expectBasicBillingAccount(orgID).Once()
	s.orgMembersRepo.On("GetByMemberID", s.ctx, orgID.String(), memberID.String()).
		Return(s.organizationMember(memberID, orgID, memberUserID, oldRoleID), nil).Once()
	oldRole := s.role(oldRoleID, orgID)
	oldRole.Name = "Member"
	s.rolesRepo.On("GetByID", s.ctx, orgID.String(), oldRoleID.String()).Return(oldRole, nil).Once()
	newRole := s.role(newRoleID, orgID)
	newRole.Name = "Admin"
	s.rolesRepo.On("GetByID", s.ctx, orgID.String(), newRoleID.String()).Return(newRole, nil).Once()
	s.orgMembersRepo.On("Update", s.ctx, mock.AnythingOfType("*models.OrganizationMember")).Return(nil).Once()
	updatedMember := s.organizationMember(memberID, orgID, memberUserID, newRoleID)
	s.orgMembersRepo.On("GetByMemberIDWithUser", s.ctx, orgID.String(), memberID.String()).
		Return(&models.OrganizationMemberWithUser{OrganizationMember: *updatedMember}, nil).Once()
	s.auditSvc.On("Create", s.ctx, mock.Anything, mock.Anything).Return(&models.AuditLog{}, nil).Maybe()

	result, err := s.svc.UpdateOrganizationMember(s.ctx, &organization_service.UpdateOrganizationMemberRequest{
		OrganizationID: orgID.String(),
		MemberID:       memberID.String(),
		RoleID:         newRoleID.String(),
	}, s.orgAccess(orgID, constants.PERMISSION_MEMBERS_WRITE))
	s.Require().NoError(err)
	s.Equal(newRoleID, result.RoleID)
}

func (s *OrganizationServiceSuite) TestDeleteOrganizationMember_Success() {
	orgID := uuid.New()
	memberID := uuid.New()
	memberUserID := uuid.New()
	roleID := uuid.New()

	s.orgRepo.On("GetByID", s.ctx, orgID.String()).
		Return(s.organization(orgID, "Acme"), nil).Once()
	s.expectBasicBillingAccount(orgID).Once()
	s.orgMembersRepo.On("GetByMemberID", s.ctx, orgID.String(), memberID.String()).
		Return(s.organizationMember(memberID, orgID, memberUserID, roleID), nil).Once()
	deleteRole := s.role(roleID, orgID)
	deleteRole.Name = "Member"
	deleteRole.Permissions = organization_service.DefaultTeamMemberPermissions
	s.rolesRepo.On("GetByID", s.ctx, orgID.String(), roleID.String()).Return(deleteRole, nil).Once()
	s.usersRepo.On("GetByID", s.ctx, memberUserID.String()).
		Return(&models.User{
			ModelBaseWithArchive: chi_types.ModelBaseWithArchive{
				ModelBase: chi_types.ModelBase{ID: memberUserID},
			},
			Email: "member@example.com",
		}, nil).Once()
	s.orgMembersRepo.On("DeleteByMemberID", s.ctx, orgID.String(), memberID.String()).Return(nil).Once()
	s.auditSvc.On("Create", s.ctx, mock.Anything, mock.Anything).Return(&models.AuditLog{}, nil).Maybe()

	err := s.svc.DeleteOrganizationMember(s.ctx, &organization_service.DeleteOrganizationMemberRequest{
		OrganizationID: orgID.String(),
		MemberID:       memberID.String(),
	}, s.orgAccess(orgID, constants.PERMISSION_MEMBERS_DELETE))
	s.NoError(err)
}

func (s *OrganizationServiceSuite) TestGetOrganizationBillingAccount_Success() {
	orgID := uuid.New()
	account := s.basicBillingAccount(orgID)

	s.orgRepo.On("GetByID", s.ctx, orgID.String()).Return(s.organization(orgID, "Acme"), nil).Once()
	s.billingAccounts.On("GetByOrganizationID", s.ctx, orgID.String()).Return(account, nil).Once()

	result, err := s.svc.GetOrganizationBillingAccount(s.ctx, &organization_service.GetOrganizationBillingAccountRequest{
		OrganizationID: orgID.String(),
	}, s.orgAccess(orgID, constants.PERMISSION_SUBSCRIPTION_READ))
	s.Require().NoError(err)
	s.Equal(account.OrganizationID, result.OrganizationID)
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
