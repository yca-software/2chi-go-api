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
	UserIdentitiesTableName = "user_identities"
)

var (
	UserIdentitiesColumns = []string{"id", "created_at", "updated_at", "user_id", "provider", "provider_user_id"}
)

type UserIdentityRepository interface {
	WithTx(tx chi_repository.Tx) UserIdentityRepository

	CreateUserIdentity(ctx context.Context, identity *models.UserIdentity) error
	UpdateUserIdentity(ctx context.Context, identity *models.UserIdentity) error
	GetUserIdentityByProviderAndProviderUserID(ctx context.Context, provider, providerUserID string) (*models.UserIdentity, error)
	GetUserIdentityByUserIDAndProvider(ctx context.Context, userID, provider string) (*models.UserIdentity, error)
}

type userIdentityRepository struct {
	userIdentitiesRepo chi_repository.Repository[models.UserIdentity]
}

func NewUserIdentityRepository(db *sqlx.DB, metricsHook chi_observer.QueryMetricsHook) UserIdentityRepository {
	return &userIdentityRepository{
		userIdentitiesRepo: chi_repository.NewRepository[models.UserIdentity](db, UserIdentitiesTableName, UserIdentitiesColumns, metricsHook),
	}
}

func (r *userIdentityRepository) WithTx(tx chi_repository.Tx) UserIdentityRepository {
	return &userIdentityRepository{
		userIdentitiesRepo: r.userIdentitiesRepo.WithTx(tx),
	}
}

func (r *userIdentityRepository) CreateUserIdentity(ctx context.Context, identity *models.UserIdentity) error {
	now := time.Now()
	return r.userIdentitiesRepo.Create(ctx, map[string]any{
		"id":               identity.ID,
		"created_at":       now,
		"updated_at":       now,
		"user_id":          identity.UserID,
		"provider":         identity.Provider,
		"provider_user_id": identity.ProviderUserID,
	})
}

func (r *userIdentityRepository) UpdateUserIdentity(ctx context.Context, identity *models.UserIdentity) error {
	return r.userIdentitiesRepo.Update(ctx, squirrel.Eq{"id": identity.ID}, map[string]any{
		"provider_user_id": identity.ProviderUserID,
		"updated_at":       time.Now(),
	})
}

func (r *userIdentityRepository) GetUserIdentityByProviderAndProviderUserID(ctx context.Context, provider, providerUserID string) (*models.UserIdentity, error) {
	return r.userIdentitiesRepo.Get(ctx, squirrel.Eq{
		"provider":         provider,
		"provider_user_id": providerUserID,
	}, nil)
}

func (r *userIdentityRepository) GetUserIdentityByUserIDAndProvider(ctx context.Context, userID, provider string) (*models.UserIdentity, error) {
	return r.userIdentitiesRepo.Get(ctx, squirrel.Eq{
		"user_id":  userID,
		"provider": provider,
	}, nil)
}
