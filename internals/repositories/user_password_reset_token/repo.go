package user_password_reset_token_repository

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
	TableName = "user_password_reset_tokens"
)

var (
	Columns = []string{"id", "created_at", "updated_at", "user_id", "expires_at", "used_at", "token_hash"}
)

type Repository interface {
	WithTx(tx chi_repository.Tx) Repository

	Create(ctx context.Context, token *models.UserPasswordResetToken) error
	GetByHash(ctx context.Context, hash string) (*models.UserPasswordResetToken, error)
	MarkAsUsed(ctx context.Context, tokenID string) error
	CleanupStaleUnused(ctx context.Context) error
}

type repository struct {
	repo chi_repository.Repository[models.UserPasswordResetToken]
}

func NewRepository(db *sqlx.DB, metricsHook chi_observer.QueryMetricsHook) Repository {
	return &repository{
		repo: chi_repository.NewRepository[models.UserPasswordResetToken](db, TableName, Columns, metricsHook),
	}
}

func (r *repository) WithTx(tx chi_repository.Tx) Repository {
	return &repository{
		repo: r.repo.WithTx(tx),
	}
}

func (r *repository) Create(ctx context.Context, token *models.UserPasswordResetToken) error {
	now := time.Now()
	return r.repo.Create(ctx, map[string]any{
		"id":         token.ID,
		"created_at": now,
		"updated_at": now,
		"user_id":    token.UserID,
		"expires_at": token.ExpiresAt,
		"token_hash": token.TokenHash,
	})
}

func (r *repository) GetByHash(ctx context.Context, hash string) (*models.UserPasswordResetToken, error) {
	return r.repo.Get(ctx, squirrel.Eq{"token_hash": hash}, nil)
}

func (r *repository) MarkAsUsed(ctx context.Context, tokenID string) error {
	return r.repo.Update(ctx, squirrel.Eq{"id": tokenID, "used_at": nil}, map[string]any{
		"used_at":    time.Now(),
		"updated_at": time.Now(),
	})
}

func (r *repository) CleanupStaleUnused(ctx context.Context) error {
	threshold := time.Now().Add(-chi_archive.ArchivedDataRetentionPeriod)
	return r.repo.Delete(ctx, squirrel.Or{
		squirrel.And{
			squirrel.NotEq{"used_at": nil},
			squirrel.LtOrEq{"used_at": threshold},
		},
		squirrel.And{
			squirrel.Eq{"used_at": nil},
			squirrel.LtOrEq{"expires_at": threshold},
		},
	})
}
