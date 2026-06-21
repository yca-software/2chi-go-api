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
	UserPasswordResetTokensTableName = "user_password_reset_tokens"
)

var (
	UserPasswordResetTokensColumns = []string{"id", "created_at", "updated_at", "user_id", "expires_at", "used_at", "token_hash"}
)

type UserPasswordResetTokenRepository interface {
	WithTx(tx chi_repository.Tx) UserPasswordResetTokenRepository

	CreatePasswordResetToken(ctx context.Context, token *models.UserPasswordResetToken) error
	GetPasswordResetTokenByHash(ctx context.Context, hash string) (*models.UserPasswordResetToken, error)
	MarkPasswordResetTokenAsUsed(ctx context.Context, tokenID string) error
	CleanupStaleUnusedPasswordResetTokens(ctx context.Context) error
}

type userPasswordResetTokenRepository struct {
	passwordResetTokensRepo chi_repository.Repository[models.UserPasswordResetToken]
}

func NewUserPasswordResetTokenRepository(db *sqlx.DB, metricsHook chi_observer.QueryMetricsHook) UserPasswordResetTokenRepository {
	return &userPasswordResetTokenRepository{
		passwordResetTokensRepo: chi_repository.NewRepository[models.UserPasswordResetToken](db, UserPasswordResetTokensTableName, UserPasswordResetTokensColumns, metricsHook),
	}
}

func (r *userPasswordResetTokenRepository) WithTx(tx chi_repository.Tx) UserPasswordResetTokenRepository {
	return &userPasswordResetTokenRepository{
		passwordResetTokensRepo: r.passwordResetTokensRepo.WithTx(tx),
	}
}

func (r *userPasswordResetTokenRepository) CreatePasswordResetToken(ctx context.Context, token *models.UserPasswordResetToken) error {
	now := time.Now()
	return r.passwordResetTokensRepo.Create(ctx, map[string]any{
		"id":         token.ID,
		"created_at": now,
		"updated_at": now,
		"user_id":    token.UserID,
		"expires_at": token.ExpiresAt,
		"token_hash": token.TokenHash,
	})
}

func (r *userPasswordResetTokenRepository) GetPasswordResetTokenByHash(ctx context.Context, hash string) (*models.UserPasswordResetToken, error) {
	return r.passwordResetTokensRepo.Get(ctx, squirrel.Eq{"token_hash": hash}, nil)
}

func (r *userPasswordResetTokenRepository) MarkPasswordResetTokenAsUsed(ctx context.Context, tokenID string) error {
	return r.passwordResetTokensRepo.Update(ctx, squirrel.Eq{"id": tokenID, "used_at": nil}, map[string]any{
		"used_at":    time.Now(),
		"updated_at": time.Now(),
	})
}

func (r *userPasswordResetTokenRepository) CleanupStaleUnusedPasswordResetTokens(ctx context.Context) error {
	threshold := time.Now().Add(-chi_archive.ArchivedDataRetentionPeriod)
	return r.passwordResetTokensRepo.Delete(ctx, squirrel.Or{
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
