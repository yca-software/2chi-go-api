package authz

import (
	"errors"
	"slices"
	"time"

	"github.com/yca-software/2chi-go-api/internals/constants"
	"github.com/yca-software/2chi-go-api/internals/models"
	platform_subscription "github.com/yca-software/2chi-go-api/internals/packages/subscription"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_types "github.com/yca-software/2chi-go-types"
)

type Authorizer struct {
	now func() time.Time
}

func NewAuthorizer(now func() time.Time) *Authorizer {
	return &Authorizer{
		now: now,
	}
}

func (a *Authorizer) CheckPlatformAdmin(access *chi_types.AccessInfo) error {
	if access == nil {
		return chi_error.NewUnauthorizedError(errors.New("access info is required"), "Unauthorized", nil)
	}

	if access.Type == chi_types.AccessTypeAPIKey {
		return chi_error.NewForbiddenError(errors.New("api key access is not allowed"), "UserIdentityRequired", nil)
	}

	if access.IsAdmin {
		return nil
	}

	return chi_error.NewForbiddenError(errors.New("user is not a platform admin"), "Forbidden", nil)
}

func (a *Authorizer) CheckOwnResource(access *chi_types.AccessInfo, resourceUserID string) error {
	if access == nil {
		return chi_error.NewUnauthorizedError(errors.New("access info is required"), "Unauthorized", nil)
	}

	if access.Type == chi_types.AccessTypeAPIKey {
		return chi_error.NewForbiddenError(errors.New("api key access is not allowed"), "UserIdentityRequired", nil)
	}

	if access.IsAdmin {
		return nil
	}

	if access.SubjectID.String() == resourceUserID {
		return nil
	}

	return chi_error.NewForbiddenError(errors.New("user is not the resource owner"), "Forbidden", nil)
}

func (a *Authorizer) CheckOrganizationPermission(access *chi_types.AccessInfo, organizationID string, requiredPermissions ...string) error {
	if access == nil {
		return chi_error.NewUnauthorizedError(errors.New("access info is required"), "Unauthorized", nil)
	}

	if access.Type == chi_types.AccessTypeUser && access.IsAdmin {
		return nil
	}

	for _, role := range access.Roles {
		if role.OrganizationID.String() == organizationID && hasPermissions(role.Permissions, requiredPermissions) {
			return nil
		}
	}

	return chi_error.NewForbiddenError(errors.New("user does not have permission to access this organization"), "Forbidden", nil)
}

func (a *Authorizer) CheckOrganizationPermissionWithSubscription(access *chi_types.AccessInfo, organization *models.OrganizationBillingAccount, requiredPermissions ...string) error {
	if organization.SubscriptionTier == constants.TIER_FREE {
		return chi_error.NewForbiddenError(errors.New("feature is not included in the free plan"), "FeatureNotIncluded", nil)
	}

	if organization.Provider != constants.BILLING_PROVIDER_CUSTOM && organization.SubscriptionExpiresAt == nil {
		return chi_error.NewPaymentRequiredError(errors.New("active subscription is required"), "PaymentRequired", nil)
	}

	if platform_subscription.IsAccessBlocked(
		organization.SubscriptionTier,
		organization.Provider == constants.BILLING_PROVIDER_CUSTOM,
		organization.SubscriptionExpiresAt,
		a.now(),
	) {
		return chi_error.NewPaymentRequiredError(errors.New("subscription has expired"), "PaymentRequired", nil)
	}

	return a.CheckOrganizationPermission(access, organization.ID.String(), requiredPermissions...)
}

func (a *Authorizer) CheckOrganizationFeature(access *chi_types.AccessInfo, organization *models.OrganizationBillingAccount, featureKey string) error {
	if access != nil && access.Type == chi_types.AccessTypeUser && access.IsAdmin {
		return nil
	}
	if organization == nil {
		return chi_error.NewForbiddenError(errors.New("organization is required"), "FeatureNotAvailable", nil)
	}
	allowedTypes, ok := constants.FEATURES_FOR_PLANS[featureKey]
	if !ok {
		return chi_error.NewForbiddenError(errors.New("feature is not available"), "FeatureNotAvailable", nil)
	}
	if !slices.Contains(allowedTypes, organization.SubscriptionTier) {
		return chi_error.NewForbiddenError(errors.New("feature is not included in the current plan"), "FeatureNotIncluded", nil)
	}
	return nil
}

func hasPermissions(have, required []string) bool {
	for _, p := range required {
		if !slices.Contains(have, p) {
			return false
		}
	}
	return true
}
