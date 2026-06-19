package user_refresh_token_repository

import (
	"context"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"

	chi_archive "github.com/yca-software/2chi-go-archive"
	"github.com/yca-software/2chi-go-api/internals/domains/core/models"
	chi_observer "github.com/yca-software/2chi-go-observer"
	chi_repository "github.com/yca-software/2chi-go-repository"
)

const (
	UserRefreshTokensTableName = "user_refresh_tokens"
)

var (
	UserRefreshTokensColumns = []string{"id", "created_at", "updated_at", "user_id", "expires_at", "revoked_at", "ip", "user_agent", "token_hash", "impersonated_by"}
)

type UserRefreshTokenRepository interface {
	WithTx(tx chi_repository.Tx) UserRefreshTokenRepository

	CreateRefreshToken(ctx context.Context, token *models.UserRefreshToken) error
	GetRefreshTokenByHash(ctx context.Context, hash string) (*models.UserRefreshToken, error)
	GetActiveRefreshTokensByUserID(ctx context.Context, userID string) (*[]models.UserRefreshToken, error)
	GetActiveImpersonationRefreshTokenByUserID(ctx context.Context, userID string) (*models.UserRefreshToken, error)
	CleanupStaleUnusedRefreshTokens(ctx context.Context) error
	RevokeRefreshTokenByID(ctx context.Context, userID, tokenID string) error
	RevokeRefreshTokenByHash(ctx context.Context, hash string) error
	RevokeAllRefreshTokensByUserID(ctx context.Context, userID string, excludeTokenID *string) error
}

type userRefreshTokenRepository struct {
	refreshTokensRepo chi_repository.Repository[models.UserRefreshToken]
}

func NewUserRefreshTokenRepository(db *sqlx.DB, metricsHook chi_observer.QueryMetricsHook) UserRefreshTokenRepository {
	return &userRefreshTokenRepository{
		refreshTokensRepo: chi_repository.NewRepository[models.UserRefreshToken](db, UserRefreshTokensTableName, UserRefreshTokensColumns, metricsHook),
	}
}

func (r *userRefreshTokenRepository) WithTx(tx chi_repository.Tx) UserRefreshTokenRepository {
	return &userRefreshTokenRepository{
		refreshTokensRepo: r.refreshTokensRepo.WithTx(tx),
	}
}

func (r *userRefreshTokenRepository) CreateRefreshToken(ctx context.Context, token *models.UserRefreshToken) error {
	now := time.Now()
	row := map[string]any{
		"id":         token.ID,
		"created_at": now,
		"updated_at": now,
		"user_id":    token.UserID,
		"expires_at": token.ExpiresAt,
		"ip":         token.IP,
		"user_agent": token.UserAgent,
		"token_hash": token.TokenHash,
	}
	if token.ImpersonatedBy.Valid {
		row["impersonated_by"] = token.ImpersonatedBy.UUID
	}
	return r.refreshTokensRepo.Create(ctx, row)
}

func (r *userRefreshTokenRepository) GetRefreshTokenByHash(ctx context.Context, hash string) (*models.UserRefreshToken, error) {
	return r.refreshTokensRepo.Get(ctx, squirrel.Eq{"token_hash": hash}, nil)
}

func (r *userRefreshTokenRepository) GetActiveRefreshTokensByUserID(ctx context.Context, userID string) (*[]models.UserRefreshToken, error) {
	return r.refreshTokensRepo.Select(ctx, squirrel.And{
		squirrel.Eq{"user_id": userID},
		squirrel.Eq{"revoked_at": nil},
		squirrel.Gt{"expires_at": time.Now()},
		squirrel.Eq{"impersonated_by": nil},
	}, nil, "created_at DESC")
}

func (r *userRefreshTokenRepository) GetActiveImpersonationRefreshTokenByUserID(ctx context.Context, userID string) (*models.UserRefreshToken, error) {
	return r.refreshTokensRepo.Get(ctx, squirrel.And{
		squirrel.Eq{"user_id": userID},
		squirrel.Eq{"revoked_at": nil},
		squirrel.Gt{"expires_at": time.Now()},
		squirrel.NotEq{"impersonated_by": nil},
	}, nil)
}

func (r *userRefreshTokenRepository) CleanupStaleUnusedRefreshTokens(ctx context.Context) error {
	threshold := time.Now().Add(-chi_archive.ArchivedDataRetentionPeriod)
	return r.refreshTokensRepo.Delete(ctx, squirrel.Or{
		squirrel.And{
			squirrel.NotEq{"revoked_at": nil},
			squirrel.LtOrEq{"revoked_at": threshold},
		},
		squirrel.And{
			squirrel.Eq{"revoked_at": nil},
			squirrel.LtOrEq{"expires_at": threshold},
		},
	})
}

func (r *userRefreshTokenRepository) RevokeRefreshTokenByID(ctx context.Context, userID, tokenID string) error {
	return r.refreshTokensRepo.Update(ctx, squirrel.And{
		squirrel.Eq{"user_id": userID},
		squirrel.Eq{"id": tokenID},
		squirrel.Eq{"revoked_at": nil},
		squirrel.Gt{"expires_at": time.Now()},
	}, map[string]any{
		"revoked_at": time.Now(),
		"updated_at": time.Now(),
	})
}

func (r *userRefreshTokenRepository) RevokeRefreshTokenByHash(ctx context.Context, hash string) error {
	return r.refreshTokensRepo.Update(ctx, squirrel.And{
		squirrel.Eq{"token_hash": hash},
		squirrel.Eq{"revoked_at": nil},
		squirrel.Gt{"expires_at": time.Now()},
	}, map[string]any{
		"revoked_at": time.Now(),
		"updated_at": time.Now(),
	})
}

func (r *userRefreshTokenRepository) RevokeAllRefreshTokensByUserID(ctx context.Context, userID string, excludeTokenID *string) error {
	condition := squirrel.And{
		squirrel.Eq{"user_id": userID},
		squirrel.Eq{"revoked_at": nil},
		squirrel.Gt{"expires_at": time.Now()},
	}
	if excludeTokenID != nil {
		condition = append(condition, squirrel.NotEq{"id": *excludeTokenID})
	}
	return r.refreshTokensRepo.Update(ctx, condition, map[string]any{
		"revoked_at": time.Now(),
		"updated_at": time.Now(),
	})
}
