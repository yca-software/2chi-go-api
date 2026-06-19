package authz

import (
	"context"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	chi_types "github.com/yca-software/2chi-go-types"
)

// ApplyAccessTokenImpersonationClaims copies impersonation fields from JWT claims onto AccessInfo.
// Deprecated: use ApplyValidatedAccessTokenImpersonationClaims so JWT claims match DB-backed sessions.
func ApplyAccessTokenImpersonationClaims(access *chi_types.AccessInfo, claims jwt.MapClaims) {
	if access == nil || claims == nil {
		return
	}

	impersonatedBy, ok := claims["impersonatedBy"].(string)
	if !ok || impersonatedBy == "" {
		return
	}

	parsed, err := uuid.Parse(impersonatedBy)
	if err != nil {
		return
	}

	access.ImpersonatedBy = uuid.NullUUID{UUID: parsed, Valid: true}
	if email, ok := claims["impersonatedByEmail"].(string); ok {
		access.ImpersonatedByEmail = email
	}
}

// ApplyValidatedAccessTokenImpersonationClaims sets impersonation on AccessInfo only when JWT claims
// match an active impersonation refresh token in the database.
func ApplyValidatedAccessTokenImpersonationClaims(
	ctx context.Context,
	deps LoadUserAccessDeps,
	userID string,
	access *chi_types.AccessInfo,
	claims jwt.MapClaims,
) error {
	if access == nil {
		return nil
	}

	dbImpersonatedBy, dbImpersonatedByEmail, err := loadImpersonationFromRefreshToken(ctx, deps, userID)
	if err != nil {
		return err
	}

	jwtImpersonatedBy, _ := claims["impersonatedBy"].(string)
	if jwtImpersonatedBy != "" {
		parsed, parseErr := uuid.Parse(jwtImpersonatedBy)
		if parseErr != nil || !dbImpersonatedBy.Valid || parsed != dbImpersonatedBy.UUID {
			access.ImpersonatedBy = uuid.NullUUID{}
			access.ImpersonatedByEmail = ""
			return nil
		}
	}

	access.ImpersonatedBy = dbImpersonatedBy
	access.ImpersonatedByEmail = dbImpersonatedByEmail
	return nil
}
