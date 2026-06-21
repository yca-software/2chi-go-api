package role_service_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/yca-software/2chi-go-api/internals/constants"
	"github.com/yca-software/2chi-go-api/internals/models"
	"github.com/yca-software/2chi-go-api/internals/repositories"
	billing_account_repository "github.com/yca-software/2chi-go-api/internals/repositories/billing_account"
	organization_member_repository "github.com/yca-software/2chi-go-api/internals/repositories/org_member"
	organization_repository "github.com/yca-software/2chi-go-api/internals/repositories/organization"
	role_repository "github.com/yca-software/2chi-go-api/internals/repositories/role"
	audit_service "github.com/yca-software/2chi-go-api/internals/services/audit"
	role_service "github.com/yca-software/2chi-go-api/internals/services/role"
	"github.com/yca-software/2chi-go-api/internals/packages/authz"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_logger "github.com/yca-software/2chi-go-logger"
	chi_types "github.com/yca-software/2chi-go-types"
	chi_validator "github.com/yca-software/2chi-go-validator"
)

type RoleServiceSuite struct {
	suite.Suite
	ctx             context.Context
	now             time.Time
	orgID           uuid.UUID
	rolesRepo       *role_repository.MockRolesRepository
	orgsRepo        *organization_repository.MockOrganizationsRepository
	billingAccounts *billing_account_repository.MockOrganizationBillingAccountsRepository
	membersRepo     *organization_member_repository.MockOrganizationMembersRepository
	auditSvc        *audit_service.MockService
	svc             role_service.Service
}

func TestRoleServiceSuite(t *testing.T) {
	suite.Run(t, new(RoleServiceSuite))
}

func (s *RoleServiceSuite) SetupTest() {
	s.ctx = context.Background()
	s.now = time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	s.orgID = uuid.New()
	s.rolesRepo = &role_repository.MockRolesRepository{}
	s.orgsRepo = &organization_repository.MockOrganizationsRepository{}
	s.billingAccounts = &billing_account_repository.MockOrganizationBillingAccountsRepository{}
	s.membersRepo = &organization_member_repository.MockOrganizationMembersRepository{}
	s.auditSvc = &audit_service.MockService{}

	s.svc = role_service.New(role_service.Dependencies{
		GenerateID: uuid.NewV7,
		Now:        func() time.Time { return s.now },
		Validator:  chi_validator.New(),
		Logger:     mockLogger(),
		Authorizer: authz.NewAuthorizer(func() time.Time { return s.now }),
		Repositories: &repositories.Repositories{
			Roles:                       s.rolesRepo,
			Organizations:               s.orgsRepo,
			OrganizationBillingAccounts: s.billingAccounts,
			OrganizationMembers:         s.membersRepo,
		},
		AuditService: s.auditSvc,
	})
}

func (s *RoleServiceSuite) expectPaidOrg() {
	expiresAt := s.now.Add(24 * time.Hour)
	s.orgsRepo.On("GetOrganizationByID", s.ctx, s.orgID.String()).
		Return(&models.Organization{
			ModelBaseWithArchive: chi_types.ModelBaseWithArchive{
				ModelBase: chi_types.ModelBase{ID: s.orgID},
			},
			Name: "Acme",
		}, nil).Once()
	s.billingAccounts.On("GetOrganizationBillingAccountByOrganizationID", s.ctx, s.orgID.String()).
		Return(&models.OrganizationBillingAccount{
			ModelBase:             chi_types.ModelBase{ID: s.orgID},
			OrganizationID:        s.orgID,
			Provider:              constants.BILLING_PROVIDER_PADDLE,
			SubscriptionTier:      constants.TIER_PRO,
			SubscriptionExpiresAt: &expiresAt,
		}, nil).Once()
}

func (s *RoleServiceSuite) userAccess() *chi_types.AccessInfo {
	return &chi_types.AccessInfo{
		Type:      chi_types.AccessTypeUser,
		SubjectID: uuid.New(),
		Roles: []chi_types.JWTAccessTokenPermissionData{{
			OrganizationID: s.orgID,
			Permissions:    []string{constants.PERMISSION_ROLE_WRITE},
		}},
	}
}

func (s *RoleServiceSuite) TestCreateRole_Validation_MissingName() {
	role, err := s.svc.CreateRole(s.ctx, &role_service.CreateRoleRequest{
		OrganizationID: s.orgID.String(),
		Name:           "",
		Permissions:    []string{constants.PERMISSION_ORG_READ},
	}, s.userAccess())
	s.Error(err)
	s.Nil(role)
}

func (s *RoleServiceSuite) TestCreateRole_FreePlanFeatureDenied() {
	s.orgsRepo.On("GetOrganizationByID", s.ctx, s.orgID.String()).
		Return(&models.Organization{
			ModelBaseWithArchive: chi_types.ModelBaseWithArchive{
				ModelBase: chi_types.ModelBase{ID: s.orgID},
			},
			Name: "Acme",
		}, nil).Once()
	s.billingAccounts.On("GetOrganizationBillingAccountByOrganizationID", s.ctx, s.orgID.String()).
		Return(&models.OrganizationBillingAccount{
			ModelBase:        chi_types.ModelBase{ID: s.orgID},
			OrganizationID:   s.orgID,
			Provider:         constants.BILLING_PROVIDER_PADDLE,
			SubscriptionTier: constants.TIER_FREE,
		}, nil).Once()

	role, err := s.svc.CreateRole(s.ctx, &role_service.CreateRoleRequest{
		OrganizationID: s.orgID.String(),
		Name:           "Custom",
		Permissions:    []string{constants.PERMISSION_ORG_READ},
	}, s.userAccess())
	s.Error(err)
	s.Nil(role)
}

func (s *RoleServiceSuite) TestDeleteRole_LockedRole() {
	roleID := uuid.New()
	s.expectPaidOrg()
	s.rolesRepo.On("GetRoleByID", s.ctx, s.orgID.String(), roleID.String()).
		Return(&models.Role{
			ModelBase:      chi_types.ModelBase{ID: roleID},
			OrganizationID: s.orgID,
			Locked:         true,
		}, nil).Once()

	access := s.userAccess()
	access.Roles[0].Permissions = []string{constants.PERMISSION_ROLE_DELETE}

	err := s.svc.DeleteRole(s.ctx, &role_service.DeleteRoleRequest{
		OrganizationID: s.orgID.String(),
		RoleID:         roleID.String(),
	}, access)
	s.Error(err)
	if apiErr, ok := chi_error.AsError(err); ok {
		s.Equal("RoleLocked", apiErr.ErrorCode)
	}
}

func (s *RoleServiceSuite) TestListRoles_Success() {
	ownerRoleID := uuid.New()
	roles := []models.Role{{
		ModelBase:      chi_types.ModelBase{ID: ownerRoleID},
		OrganizationID: s.orgID,
		Name:           "Owner",
	}}
	s.orgsRepo.On("GetOrganizationByID", s.ctx, s.orgID.String()).
		Return(&models.Organization{
			ModelBaseWithArchive: chi_types.ModelBaseWithArchive{
				ModelBase: chi_types.ModelBase{ID: s.orgID},
			},
			Name: "Acme",
		}, nil).Once()
	s.rolesRepo.On("ListRolesByOrganizationID", s.ctx, s.orgID.String()).Return(&roles, nil).Once()

	access := s.userAccess()
	access.Roles[0].Permissions = []string{constants.PERMISSION_ROLE_READ}

	result, err := s.svc.ListRoles(s.ctx, &role_service.ListRolesRequest{
		OrganizationID: s.orgID.String(),
	}, access)
	s.Require().NoError(err)
	s.Len(*result, 1)
}

func (s *RoleServiceSuite) TestCreateRole_Success() {
	s.expectPaidOrg()
	s.rolesRepo.On("CreateRole", s.ctx, mock.AnythingOfType("*models.Role")).Return(nil).Once()
	s.auditSvc.On("CreateAuditLog", s.ctx, mock.Anything, mock.Anything).Return(&models.AuditLog{}, nil).Maybe()

	role, err := s.svc.CreateRole(s.ctx, &role_service.CreateRoleRequest{
		OrganizationID: s.orgID.String(),
		Name:           "Editor",
		Permissions:    []string{constants.PERMISSION_ORG_READ},
	}, s.userAccess())
	s.Require().NoError(err)
	s.Equal("Editor", role.Name)
}

func (s *RoleServiceSuite) TestUpdateRole_LockedRole() {
	roleID := uuid.New()
	s.expectPaidOrg()
	s.rolesRepo.On("GetRoleByID", s.ctx, s.orgID.String(), roleID.String()).
		Return(&models.Role{
			ModelBase:      chi_types.ModelBase{ID: roleID},
			OrganizationID: s.orgID,
			Locked:         true,
		}, nil).Once()

	role, err := s.svc.UpdateRole(s.ctx, &role_service.UpdateRoleRequest{
		OrganizationID: s.orgID.String(),
		RoleID:         roleID.String(),
		Name:           "X",
		Permissions:    []string{constants.PERMISSION_ORG_READ},
	}, s.userAccess())
	s.Error(err)
	s.Nil(role)
	if apiErr, ok := chi_error.AsError(err); ok {
		s.Equal("RoleLocked", apiErr.ErrorCode)
	}
}

func (s *RoleServiceSuite) TestDeleteRole_Success() {
	roleID := uuid.New()
	s.expectPaidOrg()
	s.rolesRepo.On("GetRoleByID", s.ctx, s.orgID.String(), roleID.String()).
		Return(&models.Role{
			ModelBase:      chi_types.ModelBase{ID: roleID},
			OrganizationID: s.orgID,
		}, nil).Once()
	s.rolesRepo.On("DeleteRole", s.ctx, s.orgID.String(), roleID.String()).Return(nil).Once()
	s.membersRepo.On("ListUserEmailsForRole", s.ctx, s.orgID.String(), roleID.String()).Return([]string{}, nil).Once()
	s.auditSvc.On("CreateAuditLog", s.ctx, mock.Anything, mock.Anything).Return(&models.AuditLog{}, nil).Maybe()

	access := s.userAccess()
	access.Roles[0].Permissions = []string{constants.PERMISSION_ROLE_DELETE}

	err := s.svc.DeleteRole(s.ctx, &role_service.DeleteRoleRequest{
		OrganizationID: s.orgID.String(),
		RoleID:         roleID.String(),
	}, access)
	s.NoError(err)
}

func (s *RoleServiceSuite) TestUpdateRole_Success() {
	roleID := uuid.New()
	s.expectPaidOrg()
	s.rolesRepo.On("GetRoleByID", s.ctx, s.orgID.String(), roleID.String()).
		Return(&models.Role{
			ModelBase:      chi_types.ModelBase{ID: roleID},
			OrganizationID: s.orgID,
			Name:           "Old",
			Permissions:    []string{constants.PERMISSION_ORG_READ},
		}, nil).Once()
	s.rolesRepo.On("UpdateRole", s.ctx, mock.AnythingOfType("*models.Role")).Return(nil).Once()
	s.auditSvc.On("CreateAuditLog", s.ctx, mock.Anything, mock.Anything).Return(&models.AuditLog{}, nil).Maybe()

	role, err := s.svc.UpdateRole(s.ctx, &role_service.UpdateRoleRequest{
		OrganizationID: s.orgID.String(),
		RoleID:         roleID.String(),
		Name:           "New",
		Permissions:    []string{constants.PERMISSION_MEMBERS_READ},
	}, s.userAccess())
	s.Require().NoError(err)
	s.Equal("New", role.Name)
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
