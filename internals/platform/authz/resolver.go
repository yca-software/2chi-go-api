package authz

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/yca-software/2chi-go-api/internals/constants"
	api_key_repository "github.com/yca-software/2chi-go-api/internals/domains/core/repositories/api_key"
	billing_account_repository "github.com/yca-software/2chi-go-api/internals/domains/core/repositories/organization/billing_account"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_types "github.com/yca-software/2chi-go-types"
)

// PermissionResolver resolves AccessInfo for JWT users and API keys.
type PermissionResolver interface {
	ResolveUserAccess(ctx context.Context, userID string) (*chi_types.AccessInfo, error)
	ResolveAPIKeyAccess(ctx context.Context, plainKey string) (*chi_types.AccessInfo, error)
}

// PermissionResolverDeps wires session cache and repositories for access resolution.
type PermissionResolverDeps struct {
	SessionCache                    *SessionCache
	LoadUserAccess                  LoadUserAccessDeps
	APIKeysRepo                     api_key_repository.APIKeysRepository
	OrganizationBillingAccountsRepo billing_account_repository.OrganizationBillingAccountsRepository
	HashToken                       func(token string) string
	Now                             func() time.Time
}

type permissionResolver struct {
	sessionCache                    *SessionCache
	loadUserAccess                  LoadUserAccessDeps
	apiKeysRepo                     api_key_repository.APIKeysRepository
	organizationBillingAccountsRepo billing_account_repository.OrganizationBillingAccountsRepository
	hashToken                       func(token string) string
	now                             func() time.Time
}

func NewPermissionResolver(deps PermissionResolverDeps) PermissionResolver {
	if deps.SessionCache == nil {
		panic("permission resolver: session cache is required")
	}
	now := deps.Now
	if now == nil {
		now = time.Now
	}
	if deps.HashToken == nil {
		panic("permission resolver: HashToken is required")
	}
	return &permissionResolver{
		sessionCache:                    deps.SessionCache,
		loadUserAccess:                  deps.LoadUserAccess,
		apiKeysRepo:                     deps.APIKeysRepo,
		organizationBillingAccountsRepo: deps.OrganizationBillingAccountsRepo,
		hashToken:                       deps.HashToken,
		now:                             now,
	}
}

func (r *permissionResolver) ResolveUserAccess(ctx context.Context, subjectID string) (*chi_types.AccessInfo, error) {
	if access, ok := r.sessionCache.Get(ctx, subjectID); ok {
		return access, nil
	}

	if r.sessionCache.IsRevoked(ctx, subjectID) {
		return nil, chi_error.NewUnauthorizedError(errors.New("session revoked"), "InvalidToken", nil)
	}

	access, err := LoadUserAccessForBootstrap(ctx, r.loadUserAccess, subjectID)
	if err != nil {
		return nil, err
	}

	if err := r.sessionCache.Set(ctx, access); err != nil {
		return nil, err
	}
	return access, nil
}

func (r *permissionResolver) ResolveAPIKeyAccess(ctx context.Context, plainKey string) (*chi_types.AccessInfo, error) {
	if r.apiKeysRepo == nil {
		return nil, chi_error.NewUnauthorizedError(errors.New("api key resolution is not configured"), "InvalidApiKey", nil)
	}

	rawKey := strings.TrimPrefix(plainKey, constants.API_KEY_PREFIX)

	apiKey, err := r.apiKeysRepo.GetAPIKeyByHash(ctx, r.hashToken(rawKey))
	if err != nil {
		if e, ok := err.(*chi_error.Error); ok && e.StatusCode == http.StatusNotFound {
			return nil, chi_error.NewUnauthorizedError(errors.New("invalid api key"), "InvalidApiKey", nil)
		}
		return nil, err
	}

	if apiKey.ExpiresAt.Before(r.now()) {
		return nil, chi_error.NewUnauthorizedError(errors.New("api key has expired"), "ExpiredToken", nil)
	}

	org, err := r.organizationBillingAccountsRepo.GetOrganizationBillingAccountByOrganizationID(ctx, apiKey.OrganizationID.String())
	if err != nil {
		return nil, err
	}

	allowedTypes := constants.FEATURES_FOR_PLANS[constants.FEATURE_API_ACCESS]
	if !slices.Contains(allowedTypes, org.SubscriptionTier) {
		return nil, chi_error.NewForbiddenError(errors.New("api access is not included in the current plan"), "FeatureNotIncluded", nil)
	}

	return &chi_types.AccessInfo{
		Type:      chi_types.AccessTypeAPIKey,
		SubjectID: apiKey.ID,
		Email:     apiKey.Name,
		Roles: []chi_types.JWTAccessTokenPermissionData{{
			OrganizationID: apiKey.OrganizationID,
			Permissions:    apiKey.Permissions,
		}},
		IsAdmin: false,
	}, nil
}
