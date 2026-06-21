package authz_test

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
	api_key_repository "github.com/yca-software/2chi-go-api/internals/repositories/api_key"
	billing_account_repository "github.com/yca-software/2chi-go-api/internals/repositories/billing_account"
	organization_repository "github.com/yca-software/2chi-go-api/internals/repositories/organization"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_token "github.com/yca-software/2chi-go-token"
	chi_types "github.com/yca-software/2chi-go-types"
)

var resolverTestHasher = chi_token.NewHasher("resolver-test-pepper-at-least-32-chars")

type PermissionResolverSuite struct {
	suite.Suite
	ctx         context.Context
	now         time.Time
	apiKeysRepo *api_key_repository.MockAPIKeysRepository
	orgsRepo    *organization_repository.MockOrganizationsRepository
	billingRepo *billing_account_repository.MockOrganizationBillingAccountsRepository
	session     *authz.SessionCache
	resolver    authz.PermissionResolver
}

func TestPermissionResolverSuite(t *testing.T) {
	suite.Run(t, new(PermissionResolverSuite))
}

func (s *PermissionResolverSuite) SetupTest() {
	s.ctx = context.Background()
	s.now = time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	s.apiKeysRepo = &api_key_repository.MockAPIKeysRepository{}
	s.orgsRepo = &organization_repository.MockOrganizationsRepository{}
	s.billingRepo = &billing_account_repository.MockOrganizationBillingAccountsRepository{}
	s.session = authz.NewTestSessionCache(s.T(), time.Hour)
	s.resolver = authz.NewPermissionResolver(authz.PermissionResolverDeps{
		SessionCache:                    s.session,
		APIKeysRepo:                     s.apiKeysRepo,
		OrganizationsRepo:               s.orgsRepo,
		OrganizationBillingAccountsRepo: s.billingRepo,
		HashToken:                       resolverTestHasher.Hash,
		Now:                             func() time.Time { return s.now },
	})
}

func (s *PermissionResolverSuite) TestResolveAPIKeyAccess_ArchivedOrganizationRejected() {
	orgID := uuid.MustParse("22222222-2222-2222-2222-222222222401")
	keyID := uuid.MustParse("88888888-8888-8888-8888-888888888401")
	plainKey := "ak_test_plain_key_value"
	deletedAt := s.now.Add(-time.Hour)

	s.apiKeysRepo.On("GetAPIKeyByHash", s.ctx, resolverTestHasher.Hash(plainKey)).
		Return(&models.APIKey{
			ModelBase:      chi_types.ModelBase{ID: keyID},
			OrganizationID: orgID,
			Name:           "Archived Org Key",
			ExpiresAt:      s.now.Add(time.Hour),
			Permissions:    []string{constants.PERMISSION_ORG_READ},
		}, nil).Once()
	s.orgsRepo.On("GetOrganizationByIDIncludeArchived", s.ctx, orgID.String()).
		Return(&models.Organization{
			ModelBaseWithArchive: chi_types.ModelBaseWithArchive{
				ModelBase: chi_types.ModelBase{ID: orgID},
				DeletedAt: &deletedAt,
			},
			Name: "Archived Org",
		}, nil).Once()

	access, err := s.resolver.ResolveAPIKeyAccess(s.ctx, constants.API_KEY_PREFIX+plainKey)
	s.Error(err)
	s.Nil(access)
	if apiErr, ok := chi_error.AsError(err); ok {
		s.Equal("InvalidApiKey", apiErr.ErrorCode)
	}
}

func (s *PermissionResolverSuite) TestResolveAPIKeyAccess_ActiveOrganizationAllowed() {
	orgID := uuid.MustParse("22222222-2222-2222-2222-222222222402")
	keyID := uuid.MustParse("88888888-8888-8888-8888-888888888402")
	plainKey := "ak_test_active_org_key"

	s.apiKeysRepo.On("GetAPIKeyByHash", s.ctx, mock.Anything).
		Return(&models.APIKey{
			ModelBase:      chi_types.ModelBase{ID: keyID},
			OrganizationID: orgID,
			Name:           "Active Org Key",
			ExpiresAt:      s.now.Add(time.Hour),
			Permissions:    []string{constants.PERMISSION_ORG_READ},
		}, nil).Once()
	s.orgsRepo.On("GetOrganizationByIDIncludeArchived", s.ctx, orgID.String()).
		Return(&models.Organization{
			ModelBaseWithArchive: chi_types.ModelBaseWithArchive{
				ModelBase: chi_types.ModelBase{ID: orgID},
			},
			Name: "Active Org",
		}, nil).Once()
	s.billingRepo.On("GetOrganizationBillingAccountByOrganizationID", s.ctx, orgID.String()).
		Return(&models.OrganizationBillingAccount{
			OrganizationID:   orgID,
			SubscriptionTier: constants.TIER_PRO,
		}, nil).Once()

	access, err := s.resolver.ResolveAPIKeyAccess(s.ctx, constants.API_KEY_PREFIX+plainKey)
	s.Require().NoError(err)
	s.Equal(chi_types.AccessTypeAPIKey, access.Type)
	s.Equal(keyID, access.SubjectID)
}
