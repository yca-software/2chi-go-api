package authz_test

import (
	"context"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/yca-software/2chi-go-api/internals/models"
	"github.com/yca-software/2chi-go-api/internals/packages/authz"
	user_repository "github.com/yca-software/2chi-go-api/internals/repositories/user"
	user_refresh_token_repository "github.com/yca-software/2chi-go-api/internals/repositories/user_refresh_token"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_types "github.com/yca-software/2chi-go-types"
)

func TestApplyAccessTokenImpersonationClaims(t *testing.T) {
	adminID := uuid.MustParse("33333333-3333-4333-8333-333333333301")
	access := &chi_types.AccessInfo{
		Type:      chi_types.AccessTypeUser,
		SubjectID: uuid.New(),
		Email:     "user@example.com",
	}

	claims := jwt.MapClaims{
		"impersonatedBy":      adminID.String(),
		"impersonatedByEmail": "admin@example.com",
	}

	authz.ApplyAccessTokenImpersonationClaims(access, claims)

	require.True(t, access.ImpersonatedBy.Valid)
	require.Equal(t, adminID, access.ImpersonatedBy.UUID)
	require.Equal(t, "admin@example.com", access.ImpersonatedByEmail)
}

func TestApplyAccessTokenImpersonationClaims_NoClaims(t *testing.T) {
	access := &chi_types.AccessInfo{
		Type:      chi_types.AccessTypeUser,
		SubjectID: uuid.New(),
		Email:     "user@example.com",
	}

	authz.ApplyAccessTokenImpersonationClaims(access, jwt.MapClaims{})

	require.False(t, access.ImpersonatedBy.Valid)
	require.Empty(t, access.ImpersonatedByEmail)
}

func TestApplyValidatedAccessTokenImpersonationClaims_MatchesDB(t *testing.T) {
	ctx := context.Background()
	userID := uuid.MustParse("11111111-1111-4111-8111-111111111101")
	adminID := uuid.MustParse("33333333-3333-4333-8333-333333333301")

	refreshTokensRepo := &user_refresh_token_repository.MockRepository{}
	refreshTokensRepo.On("GetActiveImpersonationByUserID", ctx, userID.String()).
		Return(&models.UserRefreshToken{
			ImpersonatedBy: uuid.NullUUID{UUID: adminID, Valid: true},
		}, nil)

	usersRepo := &user_repository.MockRepository{}
	usersRepo.On("GetByID", ctx, adminID.String()).
		Return(&models.User{
			ModelBaseWithArchive: chi_types.ModelBaseWithArchive{
				ModelBase: chi_types.ModelBase{ID: adminID},
			},
			Email: "admin@example.com",
		}, nil)

	access := &chi_types.AccessInfo{SubjectID: userID}
	claims := jwt.MapClaims{
		"impersonatedBy":      adminID.String(),
		"impersonatedByEmail": "forged@example.com",
	}

	err := authz.ApplyValidatedAccessTokenImpersonationClaims(ctx, authz.LoadUserAccessDeps{
		UserRefreshTokensRepo: refreshTokensRepo,
		UsersRepo:             usersRepo,
	}, userID.String(), access, claims)
	require.NoError(t, err)
	require.True(t, access.ImpersonatedBy.Valid)
	require.Equal(t, adminID, access.ImpersonatedBy.UUID)
	require.Equal(t, "admin@example.com", access.ImpersonatedByEmail)
}

func TestApplyValidatedAccessTokenImpersonationClaims_RejectsForgedJWT(t *testing.T) {
	ctx := context.Background()
	userID := uuid.MustParse("11111111-1111-4111-8111-111111111101")
	forgedAdminID := uuid.MustParse("44444444-4444-4444-8444-444444444401")

	refreshTokensRepo := &user_refresh_token_repository.MockRepository{}
	refreshTokensRepo.On("GetActiveImpersonationByUserID", ctx, userID.String()).
		Return(nil, chi_error.NewNotFoundError(nil, "NotFound", nil))

	access := &chi_types.AccessInfo{
		SubjectID: userID,
		ImpersonatedBy: uuid.NullUUID{
			UUID:  forgedAdminID,
			Valid: true,
		},
		ImpersonatedByEmail: "forged@example.com",
	}
	claims := jwt.MapClaims{
		"impersonatedBy":      forgedAdminID.String(),
		"impersonatedByEmail": "forged@example.com",
	}

	err := authz.ApplyValidatedAccessTokenImpersonationClaims(ctx, authz.LoadUserAccessDeps{
		UserRefreshTokensRepo: refreshTokensRepo,
	}, userID.String(), access, claims)
	require.NoError(t, err)
	require.False(t, access.ImpersonatedBy.Valid)
	require.Empty(t, access.ImpersonatedByEmail)
}

func TestApplyValidatedAccessTokenImpersonationClaims_NoJWTClaimUsesDB(t *testing.T) {
	ctx := context.Background()
	userID := uuid.MustParse("11111111-1111-4111-8111-111111111101")
	adminID := uuid.MustParse("33333333-3333-4333-8333-333333333301")

	refreshTokensRepo := &user_refresh_token_repository.MockRepository{}
	refreshTokensRepo.On("GetActiveImpersonationByUserID", ctx, userID.String()).
		Return(&models.UserRefreshToken{
			ImpersonatedBy: uuid.NullUUID{UUID: adminID, Valid: true},
		}, nil)

	usersRepo := &user_repository.MockRepository{}
	usersRepo.On("GetByID", ctx, adminID.String()).
		Return(&models.User{
			ModelBaseWithArchive: chi_types.ModelBaseWithArchive{
				ModelBase: chi_types.ModelBase{ID: adminID},
			},
			Email: "admin@example.com",
		}, nil)

	access := &chi_types.AccessInfo{SubjectID: userID}

	err := authz.ApplyValidatedAccessTokenImpersonationClaims(ctx, authz.LoadUserAccessDeps{
		UserRefreshTokensRepo: refreshTokensRepo,
		UsersRepo:             usersRepo,
	}, userID.String(), access, jwt.MapClaims{})
	require.NoError(t, err)
	require.True(t, access.ImpersonatedBy.Valid)
	require.Equal(t, adminID, access.ImpersonatedBy.UUID)
}
