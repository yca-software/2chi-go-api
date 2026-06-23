package user_refresh_token_repository

import (
	"context"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"

	"github.com/yca-software/2chi-go-api/internals/models"
	chi_archive "github.com/yca-software/2chi-go-archive"
	chi_observer "github.com/yca-software/2chi-go-observer"
	chi_repository "github.com/yca-software/2chi-go-repository"
)

const (
	TableName = "user_refresh_tokens"
)

var (
	Columns = []string{"id", "created_at", "updated_at", "user_id", "expires_at", "revoked_at", "ip", "user_agent", "token_hash", "impersonated_by"}
)

type Repository interface {
	WithTx(tx chi_repository.Tx) Repository

	Create(ctx context.Context, token *models.UserRefreshToken) error
	GetByHash(ctx context.Context, hash string) (*models.UserRefreshToken, error)
	ListActiveByUserID(ctx context.Context, userID string) (*[]models.UserRefreshToken, error)
	GetActiveImpersonationByUserID(ctx context.Context, userID string) (*models.UserRefreshToken, error)
	CleanupStaleUnused(ctx context.Context) error
	RevokeByID(ctx context.Context, userID, tokenID string) error
	RevokeByHash(ctx context.Context, hash string) error
	RevokeAllByUserID(ctx context.Context, userID string, excludeTokenID *string) error
}

type repository struct {
	repo chi_repository.Repository[models.UserRefreshToken]
}

func NewRepository(db *sqlx.DB, metricsHook chi_observer.QueryMetricsHook) Repository {
	return &repository{
		repo: chi_repository.NewRepository[models.UserRefreshToken](db, TableName, Columns, metricsHook),
	}
}

func (r *repository) WithTx(tx chi_repository.Tx) Repository {
	return &repository{
		repo: r.repo.WithTx(tx),
	}
}

func (r *repository) Create(ctx context.Context, token *models.UserRefreshToken) error {
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
	return r.repo.Create(ctx, row)
}

func (r *repository) GetByHash(ctx context.Context, hash string) (*models.UserRefreshToken, error) {
	return r.repo.Get(ctx, squirrel.Eq{"token_hash": hash}, nil)
}

func (r *repository) ListActiveByUserID(ctx context.Context, userID string) (*[]models.UserRefreshToken, error) {
	return r.repo.Select(ctx, squirrel.And{
		squirrel.Eq{"user_id": userID},
		squirrel.Eq{"revoked_at": nil},
		squirrel.Gt{"expires_at": time.Now()},
		squirrel.Eq{"impersonated_by": nil},
	}, nil, "created_at DESC")
}

func (r *repository) GetActiveImpersonationByUserID(ctx context.Context, userID string) (*models.UserRefreshToken, error) {
	return r.repo.Get(ctx, squirrel.And{
		squirrel.Eq{"user_id": userID},
		squirrel.Eq{"revoked_at": nil},
		squirrel.Gt{"expires_at": time.Now()},
		squirrel.NotEq{"impersonated_by": nil},
	}, nil)
}

func (r *repository) CleanupStaleUnused(ctx context.Context) error {
	threshold := time.Now().Add(-chi_archive.ArchivedDataRetentionPeriod)
	return r.repo.Delete(ctx, squirrel.Or{
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

func (r *repository) RevokeByID(ctx context.Context, userID, tokenID string) error {
	return r.repo.Update(ctx, squirrel.And{
		squirrel.Eq{"user_id": userID},
		squirrel.Eq{"id": tokenID},
		squirrel.Eq{"revoked_at": nil},
		squirrel.Gt{"expires_at": time.Now()},
	}, map[string]any{
		"revoked_at": time.Now(),
		"updated_at": time.Now(),
	})
}

func (r *repository) RevokeByHash(ctx context.Context, hash string) error {
	return r.repo.Update(ctx, squirrel.And{
		squirrel.Eq{"token_hash": hash},
		squirrel.Eq{"revoked_at": nil},
		squirrel.Gt{"expires_at": time.Now()},
	}, map[string]any{
		"revoked_at": time.Now(),
		"updated_at": time.Now(),
	})
}

func (r *repository) RevokeAllByUserID(ctx context.Context, userID string, excludeTokenID *string) error {
	condition := squirrel.And{
		squirrel.Eq{"user_id": userID},
		squirrel.Eq{"revoked_at": nil},
		squirrel.Gt{"expires_at": time.Now()},
	}
	if excludeTokenID != nil {
		condition = append(condition, squirrel.NotEq{"id": *excludeTokenID})
	}
	return r.repo.Update(ctx, condition, map[string]any{
		"revoked_at": time.Now(),
		"updated_at": time.Now(),
	})
}
