package user_identity_repository

import (
	"context"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"

	"github.com/yca-software/2chi-go-api/internals/models"
	chi_observer "github.com/yca-software/2chi-go-observer"
	chi_repository "github.com/yca-software/2chi-go-repository"
)

const (
	TableName = "user_identities"
)

var (
	Columns = []string{"id", "created_at", "updated_at", "user_id", "provider", "provider_user_id"}
)

type Repository interface {
	WithTx(tx chi_repository.Tx) Repository

	Create(ctx context.Context, identity *models.UserIdentity) error
	Update(ctx context.Context, identity *models.UserIdentity) error
	GetByProviderAndProviderUserID(ctx context.Context, provider, providerUserID string) (*models.UserIdentity, error)
	GetByUserIDAndProvider(ctx context.Context, userID, provider string) (*models.UserIdentity, error)
}

type repository struct {
	repo chi_repository.Repository[models.UserIdentity]
}

func NewRepository(db *sqlx.DB, metricsHook chi_observer.QueryMetricsHook) Repository {
	return &repository{
		repo: chi_repository.NewRepository[models.UserIdentity](db, TableName, Columns, metricsHook),
	}
}

func (r *repository) WithTx(tx chi_repository.Tx) Repository {
	return &repository{
		repo: r.repo.WithTx(tx),
	}
}

func (r *repository) Create(ctx context.Context, identity *models.UserIdentity) error {
	now := time.Now()
	return r.repo.Create(ctx, map[string]any{
		"id":               identity.ID,
		"created_at":       now,
		"updated_at":       now,
		"user_id":          identity.UserID,
		"provider":         identity.Provider,
		"provider_user_id": identity.ProviderUserID,
	})
}

func (r *repository) Update(ctx context.Context, identity *models.UserIdentity) error {
	return r.repo.Update(ctx, squirrel.Eq{"id": identity.ID}, map[string]any{
		"provider_user_id": identity.ProviderUserID,
		"updated_at":       time.Now(),
	})
}

func (r *repository) GetByProviderAndProviderUserID(ctx context.Context, provider, providerUserID string) (*models.UserIdentity, error) {
	return r.repo.Get(ctx, squirrel.Eq{
		"provider":         provider,
		"provider_user_id": providerUserID,
	}, nil)
}

func (r *repository) GetByUserIDAndProvider(ctx context.Context, userID, provider string) (*models.UserIdentity, error) {
	return r.repo.Get(ctx, squirrel.Eq{
		"user_id":  userID,
		"provider": provider,
	}, nil)
}
