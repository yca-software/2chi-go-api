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
	organization_repository "github.com/yca-software/2chi-go-api/internals/domains/core/repositories/organization/organization"
	subscription_repository "github.com/yca-software/2chi-go-api/internals/domains/core/repositories/organization/subscription"
	api_key_service "github.com/yca-software/2chi-go-api/internals/domains/core/services/api_key"
	audit_service "github.com/yca-software/2chi-go-api/internals/domains/core/services/audit"
	"github.com/yca-software/2chi-go-api/internals/platform/authz"
	testutil "github.com/yca-software/2chi-go-api/internals/platform/testing"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_token "github.com/yca-software/2chi-go-token"
	chi_types "github.com/yca-software/2chi-go-types"
	chi_validator "github.com/yca-software/2chi-go-validator"
)

type APIKeyServiceSuite struct {
	suite.Suite
	ctx         context.Context
	now         time.Time
	orgID       uuid.UUID
	apiKeysRepo *api_key_repository.MockAPIKeysRepository
	orgsRepo    *organization_repository.MockOrganizationsRepository
	subsRepo    *subscription_repository.MockSubscriptionsRepository
	auditSvc    *audit_service.MockService
	svc         api_key_service.Service
}

func TestAPIKeyServiceSuite(t *testing.T) {
	suite.Run(t, new(APIKeyServiceSuite))
}

func (s *APIKeyServiceSuite) SetupTest() {
	s.ctx = context.Background()
	s.now = testutil.FixedNow()
	s.orgID = uuid.New()
	s.apiKeysRepo = &api_key_repository.MockAPIKeysRepository{}
	s.orgsRepo = &organization_repository.MockOrganizationsRepository{}
	s.subsRepo = &subscription_repository.MockSubscriptionsRepository{}
	s.auditSvc = &audit_service.MockService{}

	s.svc = api_key_service.New(api_key_service.Dependencies{
		GenerateID:           uuid.NewV7,
		Now:                  func() time.Time { return s.now },
		Validator:            chi_validator.New(),
		Logger:               testutil.MockLogger(),
		GenerateToken:        chi_token.GenerateOpaqueToken,
		HashToken:            testutil.TestTokenHashFn(),
		Authorizer:           authz.NewAuthorizer(func() time.Time { return s.now }),
		BillingContextLoader: testutil.BillingContextLoader(s.subsRepo),
		Repositories: &repositories.Repositories{
			APIKeys:       s.apiKeysRepo,
			Organizations: s.orgsRepo,
			Subscriptions: s.subsRepo,
		},
		AuditLogsService: s.auditSvc,
	})
}

func (s *APIKeyServiceSuite) expectProOrg() {
	s.orgsRepo.On("GetOrganizationByID", s.ctx, s.orgID.String()).
		Return(testutil.Organization(s.orgID, "Acme"), nil).Once()
	s.subsRepo.On("GetSubscriptionByOrganizationID", s.ctx, s.orgID.String()).
		Return(testutil.ProSubscription(s.orgID, s.now), nil).Once()
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
		Return(testutil.Organization(s.orgID, "Acme"), nil).Once()
	s.subsRepo.On("GetSubscriptionByOrganizationID", s.ctx, s.orgID.String()).
		Return(nil, testutil.SubscriptionNotFoundError()).Once()

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
	keys := []models.APIKey{*testutil.APIKey(uuid.New(), s.orgID, "CI")}
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
	existing := testutil.APIKey(keyID, s.orgID, "Old")
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
	existing := testutil.APIKey(keyID, s.orgID, "CI")
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
