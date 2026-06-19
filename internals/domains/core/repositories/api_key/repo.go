package api_key_repository

import (
	"context"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"

	"github.com/yca-software/2chi-go-api/internals/domains/core/models"
	chi_observer "github.com/yca-software/2chi-go-observer"
	chi_repository "github.com/yca-software/2chi-go-repository"
)

const (
	APIKeysTableName = "api_keys"
)

var (
	APIKeysColumns = []string{
		"id", "created_at", "updated_at", "organization_id", "expires_at", "name", "key_prefix", "key_hash", "permissions",
	}
)

type APIKeysRepository interface {
	WithTx(tx chi_repository.Tx) APIKeysRepository

	CreateAPIKey(ctx context.Context, apiKey *models.APIKey) error
	UpdateAPIKey(ctx context.Context, apiKey *models.APIKey) error
	DeleteAPIKey(ctx context.Context, organizationID, id string) error

	GetAPIKeyByID(ctx context.Context, organizationID, id string) (*models.APIKey, error)
	GetAPIKeyByHash(ctx context.Context, hash string) (*models.APIKey, error)
	ListAPIKeysByOrganizationID(ctx context.Context, organizationID string) (*[]models.APIKey, error)
}

type apiKeysRepository struct {
	apiKeysRepo chi_repository.Repository[models.APIKey]
}

func NewAPIKeysRepository(db *sqlx.DB, metricsHook chi_observer.QueryMetricsHook) APIKeysRepository {
	return &apiKeysRepository{
		apiKeysRepo: chi_repository.NewRepository[models.APIKey](db, APIKeysTableName, APIKeysColumns, metricsHook),
	}
}

func (r *apiKeysRepository) WithTx(tx chi_repository.Tx) APIKeysRepository {
	return &apiKeysRepository{apiKeysRepo: r.apiKeysRepo.WithTx(tx)}
}

func (r *apiKeysRepository) CreateAPIKey(ctx context.Context, apiKey *models.APIKey) error {
	now := time.Now()
	return r.apiKeysRepo.Create(ctx, map[string]any{
		"id":              apiKey.ID,
		"created_at":      now,
		"updated_at":      now,
		"organization_id": apiKey.OrganizationID,
		"expires_at":      apiKey.ExpiresAt,
		"name":            apiKey.Name,
		"key_prefix":      apiKey.KeyPrefix,
		"key_hash":        apiKey.KeyHash,
		"permissions":     apiKey.Permissions,
	})
}

func (r *apiKeysRepository) UpdateAPIKey(ctx context.Context, apiKey *models.APIKey) error {
	return r.apiKeysRepo.Update(ctx, squirrel.And{
		squirrel.Eq{"id": apiKey.ID},
		squirrel.Eq{"organization_id": apiKey.OrganizationID},
	}, map[string]any{
		"name":        apiKey.Name,
		"permissions": apiKey.Permissions,
		"updated_at":  time.Now(),
	})
}

func (r *apiKeysRepository) DeleteAPIKey(ctx context.Context, organizationID, id string) error {
	return r.apiKeysRepo.Delete(ctx, squirrel.And{
		squirrel.Eq{"id": id},
		squirrel.Eq{"organization_id": organizationID},
	})
}

func (r *apiKeysRepository) GetAPIKeyByID(ctx context.Context, organizationID, id string) (*models.APIKey, error) {
	return r.apiKeysRepo.Get(ctx, squirrel.And{
		squirrel.Eq{"id": id},
		squirrel.Eq{"organization_id": organizationID},
	}, nil)
}

func (r *apiKeysRepository) GetAPIKeyByHash(ctx context.Context, hash string) (*models.APIKey, error) {
	return r.apiKeysRepo.Get(ctx, squirrel.Eq{"key_hash": hash}, nil)
}

func (r *apiKeysRepository) ListAPIKeysByOrganizationID(ctx context.Context, organizationID string) (*[]models.APIKey, error) {
	return r.apiKeysRepo.Select(ctx, squirrel.Eq{"organization_id": organizationID}, nil, "created_at DESC")
}
