package user_email_verification_token_repository

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
	UserEmailVerificationTokensTableName = "user_email_verification_tokens"
)

var (
	UserEmailVerificationTokensColumns = []string{"id", "created_at", "updated_at", "user_id", "expires_at", "used_at", "token_hash"}
)

type UserEmailVerificationTokenRepository interface {
	WithTx(tx chi_repository.Tx) UserEmailVerificationTokenRepository

	CreateEmailVerificationToken(ctx context.Context, token *models.UserEmailVerificationToken) error
	GetEmailVerificationTokenByHash(ctx context.Context, hash string) (*models.UserEmailVerificationToken, error)
	MarkEmailVerificationTokenAsUsed(ctx context.Context, tokenID string) error
	CleanupStaleUnusedEmailVerificationTokens(ctx context.Context) error
}

type userEmailVerificationTokenRepository struct {
	emailVerificationTokensRepo chi_repository.Repository[models.UserEmailVerificationToken]
}

func NewUserEmailVerificationTokenRepository(db *sqlx.DB, metricsHook chi_observer.QueryMetricsHook) UserEmailVerificationTokenRepository {
	return &userEmailVerificationTokenRepository{
		emailVerificationTokensRepo: chi_repository.NewRepository[models.UserEmailVerificationToken](db, UserEmailVerificationTokensTableName, UserEmailVerificationTokensColumns, metricsHook),
	}
}

func (r *userEmailVerificationTokenRepository) WithTx(tx chi_repository.Tx) UserEmailVerificationTokenRepository {
	return &userEmailVerificationTokenRepository{
		emailVerificationTokensRepo: r.emailVerificationTokensRepo.WithTx(tx),
	}
}

func (r *userEmailVerificationTokenRepository) CreateEmailVerificationToken(ctx context.Context, token *models.UserEmailVerificationToken) error {
	now := time.Now()
	return r.emailVerificationTokensRepo.Create(ctx, map[string]any{
		"id":         token.ID,
		"created_at": now,
		"updated_at": now,
		"user_id":    token.UserID,
		"expires_at": token.ExpiresAt,
		"token_hash": token.TokenHash,
	})
}

func (r *userEmailVerificationTokenRepository) GetEmailVerificationTokenByHash(ctx context.Context, hash string) (*models.UserEmailVerificationToken, error) {
	return r.emailVerificationTokensRepo.Get(ctx, squirrel.Eq{"token_hash": hash}, nil)
}

func (r *userEmailVerificationTokenRepository) MarkEmailVerificationTokenAsUsed(ctx context.Context, tokenID string) error {
	return r.emailVerificationTokensRepo.Update(ctx, squirrel.Eq{"id": tokenID, "used_at": nil}, map[string]any{
		"used_at":    time.Now(),
		"updated_at": time.Now(),
	})
}

func (r *userEmailVerificationTokenRepository) CleanupStaleUnusedEmailVerificationTokens(ctx context.Context) error {
	threshold := time.Now().Add(-chi_archive.ArchivedDataRetentionPeriod)
	return r.emailVerificationTokensRepo.Delete(ctx, squirrel.Or{
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
