package api_key_service_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/yca-software/2chi-go-api/internals/constants"
	"github.com/yca-software/2chi-go-api/internals/domains/core/models"
	"github.com/yca-software/2chi-go-api/internals/domains/core/repositories"
	api_key_repository "github.com/yca-software/2chi-go-api/internals/domains/core/repositories/api_key"
	billing_account_repository "github.com/yca-software/2chi-go-api/internals/domains/core/repositories/organization/billing_account"
	organization_repository "github.com/yca-software/2chi-go-api/internals/domains/core/repositories/organization/organization"
	audit_service "github.com/yca-software/2chi-go-api/internals/domains/core/services/audit"
	api_key_service "github.com/yca-software/2chi-go-api/internals/domains/core/services/api_key"
	"github.com/yca-software/2chi-go-api/internals/platform/authz"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_logger "github.com/yca-software/2chi-go-logger"
	chi_types "github.com/yca-software/2chi-go-types"
	chi_token "github.com/yca-software/2chi-go-token"
	chi_validator "github.com/yca-software/2chi-go-validator"
)

var testTokenHasher = chi_token.NewHasher("test-pepper")

type APIKeyServiceSuite struct {
	suite.Suite
	ctx             context.Context
	now             time.Time
	orgID           uuid.UUID
	apiKeysRepo     *api_key_repository.MockAPIKeysRepository
	orgsRepo        *organization_repository.MockOrganizationsRepository
	billingAccounts *billing_account_repository.MockOrganizationBillingAccountsRepository
	auditSvc        *audit_service.MockService
	svc             api_key_service.Service
}

func TestAPIKeyServiceSuite(t *testing.T) {
	suite.Run(t, new(APIKeyServiceSuite))
}

func (s *APIKeyServiceSuite) SetupTest() {
	s.ctx = context.Background()
	s.now = fixedNow()
	s.orgID = uuid.New()
	s.apiKeysRepo = &api_key_repository.MockAPIKeysRepository{}
	s.orgsRepo = &organization_repository.MockOrganizationsRepository{}
	s.billingAccounts = &billing_account_repository.MockOrganizationBillingAccountsRepository{}
	s.auditSvc = &audit_service.MockService{}

	s.svc = api_key_service.New(api_key_service.Dependencies{
		GenerateID:    uuid.NewV7,
		Now:           func() time.Time { return s.now },
		Validator:     chi_validator.New(),
		Logger:        mockLogger(),
		GenerateToken: chi_token.GenerateOpaqueToken,
		HashToken:     testTokenHashFn(),
		Authorizer:    authz.NewAuthorizer(func() time.Time { return s.now }),
		Repositories: &repositories.Repositories{
			APIKeys:                     s.apiKeysRepo,
			Organizations:               s.orgsRepo,
			OrganizationBillingAccounts: s.billingAccounts,
		},
		AuditService: s.auditSvc,
	})
}

func (s *APIKeyServiceSuite) expectProOrg() {
	expiresAt := s.now.Add(24 * time.Hour)
	s.orgsRepo.On("GetOrganizationByID", s.ctx, s.orgID.String()).
		Return(organization(s.orgID, "Acme"), nil).Once()
	s.billingAccounts.On("GetOrganizationBillingAccountByOrganizationID", s.ctx, s.orgID.String()).
		Return(billingAccount(s.orgID, constants.TIER_PRO, constants.BILLING_PROVIDER_PADDLE, &expiresAt), nil).Once()
}

func (s *APIKeyServiceSuite) writeAccess() *chi_types.AccessInfo {
	return &chi_types.AccessInfo{
		Type:      chi_types.AccessTypeUser,
		SubjectID: uuid.New(),
		Email:     "admin@example.com",
		Roles: []chi_types.JWTAccessTokenPermissionData{{
			OrganizationID: s.orgID,
			Permissions:    []string{constants.PERMISSION_API_KEY_WRITE},
		}},
	}
}

func (s *APIKeyServiceSuite) TestCreateAPIKey_InvalidPermission() {
	resp, err := s.svc.CreateAPIKey(s.ctx, &api_key_service.CreateAPIKeyRequest{
		OrganizationID: s.orgID.String(),
		Name:           "CI",
		Permissions:    []string{constants.PERMISSION_AUDIT_READ},
		ExpiresAt:      s.now.Add(24 * time.Hour),
	}, s.writeAccess())
	s.Error(err)
	s.Nil(resp)
	if apiErr, ok := chi_error.AsError(err); ok {
		s.Equal("InvalidAPIKeyPermission", apiErr.ErrorCode)
	}
}

func (s *APIKeyServiceSuite) TestCreateAPIKey_Success() {
	s.expectProOrg()
	s.apiKeysRepo.On("CreateAPIKey", s.ctx, mock.AnythingOfType("*models.APIKey")).Return(nil).Once()
	s.auditSvc.On("CreateAuditLog", s.ctx, mock.Anything, mock.Anything).Return(&models.AuditLog{}, nil).Maybe()

	resp, err := s.svc.CreateAPIKey(s.ctx, &api_key_service.CreateAPIKeyRequest{
		OrganizationID: s.orgID.String(),
		Name:           "CI",
		Permissions:    []string{constants.PERMISSION_ORG_READ},
		ExpiresAt:      s.now.Add(24 * time.Hour),
	}, s.writeAccess())
	s.Require().NoError(err)
	s.Require().NotNil(resp)
	s.NotEmpty(resp.Secret)
	s.True(len(resp.Secret) > len(constants.API_KEY_PREFIX))
}

func (s *APIKeyServiceSuite) TestCreateAPIKey_FreePlanFeatureDenied() {
	s.orgsRepo.On("GetOrganizationByID", s.ctx, s.orgID.String()).
		Return(organization(s.orgID, "Acme"), nil).Once()
	s.billingAccounts.On("GetOrganizationBillingAccountByOrganizationID", s.ctx, s.orgID.String()).
		Return(billingAccount(s.orgID, constants.TIER_FREE, constants.BILLING_PROVIDER_PADDLE, nil), nil).Once()

	resp, err := s.svc.CreateAPIKey(s.ctx, &api_key_service.CreateAPIKeyRequest{
		OrganizationID: s.orgID.String(),
		Name:           "CI",
		Permissions:    []string{constants.PERMISSION_ORG_READ},
		ExpiresAt:      s.now.Add(24 * time.Hour),
	}, s.writeAccess())
	s.Error(err)
	s.Nil(resp)
}

func (s *APIKeyServiceSuite) TestListAPIKeys_Success() {
	keys := []models.APIKey{*apiKey(uuid.New(), s.orgID, "CI")}
	s.expectProOrg()
	s.apiKeysRepo.On("ListAPIKeysByOrganizationID", s.ctx, s.orgID.String()).Return(&keys, nil).Once()

	access := s.writeAccess()
	access.Roles[0].Permissions = []string{constants.PERMISSION_API_KEY_READ}

	result, err := s.svc.ListAPIKeys(s.ctx, &api_key_service.ListAPIKeysRequest{
		OrganizationID: s.orgID.String(),
	}, access)
	s.Require().NoError(err)
	s.Len(*result, 1)
}

func (s *APIKeyServiceSuite) TestUpdateAPIKey_InvalidPermission() {
	resp, err := s.svc.UpdateAPIKey(s.ctx, &api_key_service.UpdateAPIKeyRequest{
		OrganizationID: s.orgID.String(),
		APIKeyID:       uuid.New().String(),
		Name:           "CI",
		Permissions:    []string{constants.PERMISSION_AUDIT_READ},
	}, s.writeAccess())
	s.Error(err)
	s.Nil(resp)
}

func (s *APIKeyServiceSuite) TestUpdateAPIKey_Success() {
	keyID := uuid.New()
	existing := apiKey(keyID, s.orgID, "Old")
	existing.Permissions = []string{constants.PERMISSION_ORG_READ}
	s.expectProOrg()
	s.apiKeysRepo.On("GetAPIKeyByID", s.ctx, s.orgID.String(), keyID.String()).Return(existing, nil).Once()
	s.apiKeysRepo.On("UpdateAPIKey", s.ctx, mock.AnythingOfType("*models.APIKey")).Return(nil).Once()
	s.auditSvc.On("CreateAuditLog", s.ctx, mock.Anything, mock.Anything).Return(&models.AuditLog{}, nil).Maybe()

	updated, err := s.svc.UpdateAPIKey(s.ctx, &api_key_service.UpdateAPIKeyRequest{
		OrganizationID: s.orgID.String(),
		APIKeyID:       keyID.String(),
		Name:           "New",
		Permissions:    []string{constants.PERMISSION_MEMBERS_READ},
	}, s.writeAccess())
	s.Require().NoError(err)
	s.Equal("New", updated.Name)
}

func (s *APIKeyServiceSuite) TestDeleteAPIKey_Success() {
	keyID := uuid.New()
	existing := apiKey(keyID, s.orgID, "CI")
	s.expectProOrg()
	s.apiKeysRepo.On("GetAPIKeyByID", s.ctx, s.orgID.String(), keyID.String()).Return(existing, nil).Once()
	s.apiKeysRepo.On("DeleteAPIKey", s.ctx, s.orgID.String(), keyID.String()).Return(nil).Once()
	s.auditSvc.On("CreateAuditLog", s.ctx, mock.Anything, mock.Anything).Return(&models.AuditLog{}, nil).Maybe()

	access := s.writeAccess()
	access.Roles[0].Permissions = []string{constants.PERMISSION_API_KEY_DELETE}

	err := s.svc.DeleteAPIKey(s.ctx, &api_key_service.DeleteAPIKeyRequest{
		OrganizationID: s.orgID.String(),
		APIKeyID:       keyID.String(),
	}, access)
	s.NoError(err)
}

func fixedNow() time.Time {
	return time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
}

func testTokenHashFn() func(string) string {
	return testTokenHasher.Hash
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

func organization(orgID uuid.UUID, name string) *models.Organization {
	return &models.Organization{
		ModelBaseWithArchive: chi_types.ModelBaseWithArchive{
			ModelBase: chi_types.ModelBase{ID: orgID},
		},
		Name: name,
	}
}

func billingAccount(orgID uuid.UUID, tier, provider string, expiresAt *time.Time) *models.OrganizationBillingAccount {
	return &models.OrganizationBillingAccount{
		ModelBase:             chi_types.ModelBase{ID: orgID},
		OrganizationID:        orgID,
		Provider:              provider,
		SubscriptionTier:      tier,
		SubscriptionExpiresAt: expiresAt,
	}
}

func apiKey(id, orgID uuid.UUID, name string) *models.APIKey {
	return &models.APIKey{
		ModelBase:      chi_types.ModelBase{ID: id},
		OrganizationID: orgID,
		Name:           name,
		KeyPrefix:      "ak_test",
		KeyHash:        "hash",
		Permissions:    []string{constants.PERMISSION_ORG_READ},
		ExpiresAt:      fixedNow().Add(24 * time.Hour),
	}
}
