package role_repository

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
	TableName = "roles"
)

var (
	Columns = []string{
		"id", "created_at", "updated_at", "organization_id", "name", "description", "permissions", "locked",
	}
)

type Repository interface {
	WithTx(tx chi_repository.Tx) Repository

	Create(ctx context.Context, role *models.Role) error
	CreateMany(ctx context.Context, roles *[]models.Role) error
	Update(ctx context.Context, role *models.Role) error
	Delete(ctx context.Context, organizationID, id string) error

	GetByID(ctx context.Context, organizationID, id string) (*models.Role, error)
	ListByOrganizationID(ctx context.Context, organizationID string) (*[]models.Role, error)
}

type repository struct {
	repo chi_repository.Repository[models.Role]
}

func NewRepository(db *sqlx.DB, metricsHook chi_observer.QueryMetricsHook) Repository {
	return &repository{
		repo: chi_repository.NewRepository[models.Role](db, TableName, Columns, metricsHook),
	}
}

func (r *repository) WithTx(tx chi_repository.Tx) Repository {
	return &repository{
		repo: r.repo.WithTx(tx),
	}
}

func (r *repository) Create(ctx context.Context, role *models.Role) error {
	now := time.Now()
	return r.repo.Create(ctx, map[string]any{
		"id":              role.ID,
		"created_at":      now,
		"updated_at":      now,
		"organization_id": role.OrganizationID,
		"name":            role.Name,
		"description":     role.Description,
		"permissions":     role.Permissions,
		"locked":          role.Locked,
	})
}

func (r *repository) CreateMany(ctx context.Context, roles *[]models.Role) error {
	now := time.Now()
	data := make([]map[string]any, len(*roles))
	for i, role := range *roles {
		data[i] = map[string]any{
			"id":              role.ID,
			"created_at":      now,
			"updated_at":      now,
			"organization_id": role.OrganizationID,
			"name":            role.Name,
			"description":     role.Description,
			"permissions":     role.Permissions,
			"locked":          role.Locked,
		}
	}
	return r.repo.CreateMany(ctx, Columns, data, false)
}

func (r *repository) Update(ctx context.Context, role *models.Role) error {
	return r.repo.Update(ctx, squirrel.And{
		squirrel.Eq{"id": role.ID},
		squirrel.Eq{"organization_id": role.OrganizationID},
		squirrel.Eq{"locked": false},
	}, map[string]any{
		"name":        role.Name,
		"description": role.Description,
		"permissions": role.Permissions,
		"updated_at":  time.Now(),
	})
}

func (r *repository) Delete(ctx context.Context, organizationID, id string) error {
	return r.repo.Delete(ctx, squirrel.And{
		squirrel.Eq{"id": id},
		squirrel.Eq{"organization_id": organizationID},
		squirrel.Eq{"locked": false},
	})
}

func (r *repository) GetByID(ctx context.Context, organizationID, id string) (*models.Role, error) {
	return r.repo.Get(ctx, squirrel.And{
		squirrel.Eq{"id": id},
		squirrel.Eq{"organization_id": organizationID},
	}, nil)
}

func (r *repository) ListByOrganizationID(ctx context.Context, organizationID string) (*[]models.Role, error) {
	return r.repo.Select(ctx, squirrel.Eq{"organization_id": organizationID}, nil, "name ASC")
}
