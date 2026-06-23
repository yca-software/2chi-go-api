package authz_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"github.com/yca-software/2chi-go-api/internals/constants"
	"github.com/yca-software/2chi-go-api/internals/models"
	"github.com/yca-software/2chi-go-api/internals/packages/authz"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_types "github.com/yca-software/2chi-go-types"
)

type AuthorizerSuite struct {
	suite.Suite
	authorizer *authz.Authorizer
	now        time.Time
}

func TestAuthorizerSuite(t *testing.T) {
	suite.Run(t, new(AuthorizerSuite))
}

func (s *AuthorizerSuite) SetupTest() {
	s.now = time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	s.authorizer = authz.NewAuthorizer(func() time.Time { return s.now })
}

func (s *AuthorizerSuite) billingAccount(orgID uuid.UUID, tier, provider string, expiresAt *time.Time) *models.OrganizationBillingAccount {
	return &models.OrganizationBillingAccount{
		OrganizationID:        orgID,
		Provider:              provider,
		SubscriptionTier:      tier,
		SubscriptionExpiresAt: expiresAt,
	}
}

func (s *AuthorizerSuite) userAccess(orgID uuid.UUID, permissions ...string) *chi_types.AccessInfo {
	return &chi_types.AccessInfo{
		Type:      chi_types.AccessTypeUser,
		SubjectID: uuid.New(),
		Roles: []chi_types.JWTAccessTokenPermissionData{{
			OrganizationID: orgID,
			Permissions:    permissions,
		}},
	}
}

func (s *AuthorizerSuite) TestCheckOrganizationPermission_WithPermission() {
	orgID := uuid.New()
	err := s.authorizer.CheckOrganizationPermission(
		s.userAccess(orgID, constants.PERMISSION_ORG_READ),
		orgID.String(),
		constants.PERMISSION_ORG_READ,
	)
	s.NoError(err)
}

func (s *AuthorizerSuite) TestCheckOrganizationPermission_WithoutPermission() {
	orgID := uuid.New()
	err := s.authorizer.CheckOrganizationPermission(
		s.userAccess(orgID, constants.PERMISSION_ORG_READ),
		orgID.String(),
		constants.PERMISSION_ORG_WRITE,
	)
	s.Error(err)
}

func (s *AuthorizerSuite) TestCheckOrganizationPermission_AdminAccess() {
	orgID := uuid.New()
	access := &chi_types.AccessInfo{
		Type:      chi_types.AccessTypeUser,
		SubjectID: uuid.New(),
		IsAdmin:   true,
	}

	err := s.authorizer.CheckOrganizationPermission(access, orgID.String(), constants.PERMISSION_ORG_READ)
	s.NoError(err)
}

func (s *AuthorizerSuite) TestCheckOrganizationPermissionWithSubscription_ActiveSubscription() {
	orgID := uuid.New()
	futureTime := s.now.Add(24 * time.Hour)
	organization := s.billingAccount(orgID, constants.TIER_BASIC, constants.BILLING_PROVIDER_PADDLE, &futureTime)

	err := s.authorizer.CheckOrganizationPermissionWithSubscription(
		s.userAccess(orgID, constants.PERMISSION_ORG_READ),
		organization,
		constants.PERMISSION_ORG_READ,
	)
	s.NoError(err)
}

func (s *AuthorizerSuite) TestCheckOrganizationPermissionWithSubscription_ExpiredSubscription() {
	orgID := uuid.New()
	pastTime := s.now.Add(-8 * 24 * time.Hour)
	organization := s.billingAccount(orgID, constants.TIER_BASIC, constants.BILLING_PROVIDER_PADDLE, &pastTime)

	err := s.authorizer.CheckOrganizationPermissionWithSubscription(
		s.userAccess(orgID, constants.PERMISSION_ORG_READ),
		organization,
		constants.PERMISSION_ORG_READ,
	)
	s.Error(err)
	if e, ok := err.(*chi_error.Error); ok {
		s.Equal("PaymentRequired", e.ErrorCode)
	}
}

func (s *AuthorizerSuite) TestCheckOrganizationPermissionWithSubscription_CustomSubscriptionWithoutExpiry() {
	orgID := uuid.New()
	organization := s.billingAccount(orgID, constants.TIER_BASIC, constants.BILLING_PROVIDER_CUSTOM, nil)

	err := s.authorizer.CheckOrganizationPermissionWithSubscription(
		s.userAccess(orgID, constants.PERMISSION_ORG_READ),
		organization,
		constants.PERMISSION_ORG_READ,
	)
	s.NoError(err)
}

func (s *AuthorizerSuite) TestCheckOrganizationPermissionWithSubscription_FreeTier() {
	orgID := uuid.New()
	organization := s.billingAccount(orgID, constants.TIER_FREE, constants.BILLING_PROVIDER_PADDLE, nil)

	err := s.authorizer.CheckOrganizationPermissionWithSubscription(
		s.userAccess(orgID, constants.PERMISSION_ORG_READ),
		organization,
		constants.PERMISSION_ORG_READ,
	)
	s.Error(err)
	if e, ok := err.(*chi_error.Error); ok {
		s.Equal("FeatureNotIncluded", e.ErrorCode)
	}
}

func (s *AuthorizerSuite) TestCheckOrganizationFeature_PlanExcludesFeature() {
	org := s.billingAccount(uuid.New(), constants.TIER_FREE, constants.BILLING_PROVIDER_PADDLE, nil)
	access := &chi_types.AccessInfo{
		Type:      chi_types.AccessTypeUser,
		SubjectID: uuid.New(),
	}

	err := s.authorizer.CheckOrganizationFeature(access, org, constants.FEATURE_AUDIT_LOG)
	s.Error(err)
	if e, ok := err.(*chi_error.Error); ok {
		s.Equal("FeatureNotIncluded", e.ErrorCode)
	}
}

func (s *AuthorizerSuite) TestCheckOrganizationFeature_IncludesFeature() {
	orgID := uuid.New()
	org := s.billingAccount(orgID, constants.TIER_BASIC, constants.BILLING_PROVIDER_PADDLE, nil)
	access := &chi_types.AccessInfo{
		Type:      chi_types.AccessTypeUser,
		SubjectID: uuid.New(),
	}

	err := s.authorizer.CheckOrganizationFeature(access, org, constants.FEATURE_AUDIT_LOG)
	s.NoError(err)
}
