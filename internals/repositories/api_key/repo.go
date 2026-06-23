package api_key_repository

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
	TableName = "api_keys"
)

var (
	Columns = []string{
		"id", "created_at", "updated_at", "organization_id", "expires_at", "name", "key_prefix", "key_hash", "permissions",
	}
)

type Repository interface {
	WithTx(tx chi_repository.Tx) Repository

	Create(ctx context.Context, apiKey *models.APIKey) error
	Update(ctx context.Context, apiKey *models.APIKey) error
	Delete(ctx context.Context, organizationID, id string) error

	GetByID(ctx context.Context, organizationID, id string) (*models.APIKey, error)
	GetByHash(ctx context.Context, hash string) (*models.APIKey, error)
	ListByOrganizationID(ctx context.Context, organizationID string) (*[]models.APIKey, error)
}

type repository struct {
	repo chi_repository.Repository[models.APIKey]
}

func NewRepository(db *sqlx.DB, metricsHook chi_observer.QueryMetricsHook) Repository {
	return &repository{
		repo: chi_repository.NewRepository[models.APIKey](db, TableName, Columns, metricsHook),
	}
}

func (r *repository) WithTx(tx chi_repository.Tx) Repository {
	return &repository{repo: r.repo.WithTx(tx)}
}

func (r *repository) Create(ctx context.Context, apiKey *models.APIKey) error {
	now := time.Now()
	return r.repo.Create(ctx, map[string]any{
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

func (r *repository) Update(ctx context.Context, apiKey *models.APIKey) error {
	return r.repo.Update(ctx, squirrel.And{
		squirrel.Eq{"id": apiKey.ID},
		squirrel.Eq{"organization_id": apiKey.OrganizationID},
	}, map[string]any{
		"name":        apiKey.Name,
		"permissions": apiKey.Permissions,
		"updated_at":  time.Now(),
	})
}

func (r *repository) Delete(ctx context.Context, organizationID, id string) error {
	return r.repo.Delete(ctx, squirrel.And{
		squirrel.Eq{"id": id},
		squirrel.Eq{"organization_id": organizationID},
	})
}

func (r *repository) GetByID(ctx context.Context, organizationID, id string) (*models.APIKey, error) {
	return r.repo.Get(ctx, squirrel.And{
		squirrel.Eq{"id": id},
		squirrel.Eq{"organization_id": organizationID},
	}, nil)
}

func (r *repository) GetByHash(ctx context.Context, hash string) (*models.APIKey, error) {
	return r.repo.Get(ctx, squirrel.Eq{"key_hash": hash}, nil)
}

func (r *repository) ListByOrganizationID(ctx context.Context, organizationID string) (*[]models.APIKey, error) {
	return r.repo.Select(ctx, squirrel.Eq{"organization_id": organizationID}, nil, "created_at DESC")
}
