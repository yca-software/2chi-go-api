package authz

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/yca-software/2chi-go-api/internals/models"
	admin_access_repository "github.com/yca-software/2chi-go-api/internals/repositories/admin_access"
	organization_member_repository "github.com/yca-software/2chi-go-api/internals/repositories/org_member"
	organization_repository "github.com/yca-software/2chi-go-api/internals/repositories/organization"
	user_repository "github.com/yca-software/2chi-go-api/internals/repositories/user"
	user_refresh_token_repository "github.com/yca-software/2chi-go-api/internals/repositories/user_refresh_token"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_types "github.com/yca-software/2chi-go-types"
)

// LoadUserAccessDeps supplies repositories for building a session profile on cache miss.
type LoadUserAccessDeps struct {
	AdminAccessRepo         admin_access_repository.Repository
	UsersRepo               user_repository.Repository
	UserRefreshTokensRepo   user_refresh_token_repository.Repository
	OrganizationsRepo       organization_repository.Repository
	OrganizationMembersRepo organization_member_repository.Repository
}

// BuildUserAccessInfo builds AccessInfo for Redis session caching from user and org role rows.
func BuildUserAccessInfo(
	user *models.User,
	isAdmin bool,
	orgRoles *[]models.OrganizationMemberWithOrganizationAndRole,
	impersonatedBy uuid.NullUUID,
	impersonatedByEmail string,
) *chi_types.AccessInfo {
	roles := []chi_types.JWTAccessTokenPermissionData{}
	if orgRoles != nil {
		for _, member := range *orgRoles {
			roles = append(roles, chi_types.JWTAccessTokenPermissionData{
				OrganizationID: member.OrganizationID,
				Permissions:    member.RolePermissions,
			})
		}
	}
	return &chi_types.AccessInfo{
		Type:                chi_types.AccessTypeUser,
		SubjectID:           user.ID,
		Email:               user.Email,
		IsAdmin:             isAdmin,
		Roles:               roles,
		ImpersonatedBy:      impersonatedBy,
		ImpersonatedByEmail: impersonatedByEmail,
	}
}

func loadImpersonationFromRefreshToken(
	ctx context.Context,
	deps LoadUserAccessDeps,
	userID string,
) (uuid.NullUUID, string, error) {
	if deps.UserRefreshTokensRepo == nil {
		return uuid.NullUUID{}, "", nil
	}

	token, err := deps.UserRefreshTokensRepo.GetActiveImpersonationByUserID(ctx, userID)
	if err != nil {
		if e, ok := err.(*chi_error.Error); ok && e.StatusCode == http.StatusNotFound {
			return uuid.NullUUID{}, "", nil
		}
		return uuid.NullUUID{}, "", err
	}
	if token == nil || !token.ImpersonatedBy.Valid {
		return uuid.NullUUID{}, "", nil
	}

	impersonator, err := deps.UsersRepo.GetByID(ctx, token.ImpersonatedBy.UUID.String())
	if err != nil {
		return uuid.NullUUID{}, "", err
	}

	return token.ImpersonatedBy, impersonator.Email, nil
}

// LoadUserAccess builds the session profile for a human user (admin flag, organization roles, impersonation).
func LoadUserAccess(ctx context.Context, deps LoadUserAccessDeps, userID string) (*chi_types.AccessInfo, error) {
	user, err := deps.UsersRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	orgRoles, err := deps.OrganizationMembersRepo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	isAdmin := false
	adminAccess, err := deps.AdminAccessRepo.GetByUserID(ctx, userID)
	if err != nil {
		if e, ok := err.(*chi_error.Error); ok && e.StatusCode == http.StatusNotFound {
			isAdmin = false
		} else {
			return nil, err
		}
	} else if adminAccess != nil {
		isAdmin = true
	}

	impersonatedBy, impersonatedByEmail, err := loadImpersonationFromRefreshToken(ctx, deps, userID)
	if err != nil {
		return nil, err
	}

	return BuildUserAccessInfo(user, isAdmin, orgRoles, impersonatedBy, impersonatedByEmail), nil
}

// LoadUserAccessForBootstrap loads access using a parsed user id (used when JWT subject is already validated).
func LoadUserAccessForBootstrap(ctx context.Context, deps LoadUserAccessDeps, userID string) (*chi_types.AccessInfo, error) {
	if _, err := uuid.Parse(userID); err != nil {
		return nil, chi_error.NewUnauthorizedError(errors.New("invalid user id"), "InvalidToken", nil)
	}
	return LoadUserAccess(ctx, deps, userID)
}
