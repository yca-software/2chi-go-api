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
	RolesTableName = "roles"
)

var (
	RolesColumns = []string{
		"id", "created_at", "updated_at", "organization_id", "name", "description", "permissions", "locked",
	}
)

type RolesRepository interface {
	WithTx(tx chi_repository.Tx) RolesRepository

	CreateRole(ctx context.Context, role *models.Role) error
	CreateRoles(ctx context.Context, roles *[]models.Role) error
	UpdateRole(ctx context.Context, role *models.Role) error
	DeleteRole(ctx context.Context, organizationID, id string) error

	GetRoleByID(ctx context.Context, organizationID, id string) (*models.Role, error)
	ListRolesByOrganizationID(ctx context.Context, organizationID string) (*[]models.Role, error)
}

type rolesRepository struct {
	rolesRepo chi_repository.Repository[models.Role]
}

func NewRolesRepository(db *sqlx.DB, metricsHook chi_observer.QueryMetricsHook) RolesRepository {
	return &rolesRepository{
		rolesRepo: chi_repository.NewRepository[models.Role](db, RolesTableName, RolesColumns, metricsHook),
	}
}

func (r *rolesRepository) WithTx(tx chi_repository.Tx) RolesRepository {
	return &rolesRepository{
		rolesRepo: r.rolesRepo.WithTx(tx),
	}
}

func (r *rolesRepository) CreateRole(ctx context.Context, role *models.Role) error {
	now := time.Now()
	return r.rolesRepo.Create(ctx, map[string]any{
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

func (r *rolesRepository) CreateRoles(ctx context.Context, roles *[]models.Role) error {
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
	return r.rolesRepo.CreateMany(ctx, RolesColumns, data, false)
}

func (r *rolesRepository) UpdateRole(ctx context.Context, role *models.Role) error {
	return r.rolesRepo.Update(ctx, squirrel.And{
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

func (r *rolesRepository) DeleteRole(ctx context.Context, organizationID, id string) error {
	return r.rolesRepo.Delete(ctx, squirrel.And{
		squirrel.Eq{"id": id},
		squirrel.Eq{"organization_id": organizationID},
		squirrel.Eq{"locked": false},
	})
}

func (r *rolesRepository) GetRoleByID(ctx context.Context, organizationID, id string) (*models.Role, error) {
	return r.rolesRepo.Get(ctx, squirrel.And{
		squirrel.Eq{"id": id},
		squirrel.Eq{"organization_id": organizationID},
	}, nil)
}

func (r *rolesRepository) ListRolesByOrganizationID(ctx context.Context, organizationID string) (*[]models.Role, error) {
	return r.rolesRepo.Select(ctx, squirrel.Eq{"organization_id": organizationID}, nil, "name ASC")
}
