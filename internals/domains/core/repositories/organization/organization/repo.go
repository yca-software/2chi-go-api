package organization_repository

import (
	"context"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"

	chi_archive "github.com/yca-software/2chi-go-archive"
	"github.com/yca-software/2chi-go-api/internals/domains/core/models"
	chi_observer "github.com/yca-software/2chi-go-observer"
	chi_repository "github.com/yca-software/2chi-go-repository"
)

const (
	OrganizationsTableName = "organizations"
)

var (
	OrganizationsColumns = []string{"id", "created_at", "updated_at", "deleted_at", "name"}
)

type OrganizationsRepository interface {
	WithTx(tx chi_repository.Tx) OrganizationsRepository

	CreateOrganization(ctx context.Context, organization *models.Organization) error
	UpdateOrganization(ctx context.Context, organization *models.Organization) error
	ArchiveOrganization(ctx context.Context, organization *models.Organization) error
	RestoreOrganization(ctx context.Context, id string) error
	CleanupArchivedOrganizations(ctx context.Context) error

	GetOrganizationByID(ctx context.Context, id string) (*models.Organization, error)
	GetOrganizationByIDIncludeArchived(ctx context.Context, id string) (*models.Organization, error)
	SearchOrganizations(ctx context.Context, searchPhrase string, filter chi_archive.ArchiveFilter, limit, offset int) (*[]models.Organization, error)
}

type organizationsRepository struct {
	organizationsRepo chi_repository.Repository[models.Organization]
}

func NewOrganizationsRepository(db *sqlx.DB, metricsHook chi_observer.QueryMetricsHook) OrganizationsRepository {
	return &organizationsRepository{
		organizationsRepo: chi_repository.NewRepository[models.Organization](db, OrganizationsTableName, OrganizationsColumns, metricsHook),
	}
}

func (r *organizationsRepository) WithTx(tx chi_repository.Tx) OrganizationsRepository {
	return &organizationsRepository{
		organizationsRepo: r.organizationsRepo.WithTx(tx),
	}
}

func searchOrganizationsCondition(searchPhrase string, filter chi_archive.ArchiveFilter) (squirrel.Sqlizer, string) {
	var archiveCond squirrel.Sqlizer = squirrel.Eq{"deleted_at": nil}
	sort := "created_at DESC"
	if filter == chi_archive.ArchiveFilterArchived {
		archiveCond = squirrel.NotEq{"deleted_at": nil}
		sort = "deleted_at DESC"
	}

	if searchPhrase == "" {
		return archiveCond, sort
	}

	return squirrel.And{
		archiveCond,
		squirrel.ILike{"name": "%" + searchPhrase + "%"},
	}, sort
}

func (r *organizationsRepository) CreateOrganization(ctx context.Context, organization *models.Organization) error {
	now := time.Now()
	return r.organizationsRepo.Create(ctx, map[string]any{
		"id":         organization.ID,
		"created_at": now,
		"updated_at": now,
		"name":       organization.Name,
	})
}

func (r *organizationsRepository) UpdateOrganization(ctx context.Context, organization *models.Organization) error {
	return r.organizationsRepo.Update(ctx, squirrel.Eq{"id": organization.ID, "deleted_at": nil}, map[string]any{
		"name":       organization.Name,
		"updated_at": time.Now(),
	})
}

func (r *organizationsRepository) ArchiveOrganization(ctx context.Context, organization *models.Organization) error {
	return r.organizationsRepo.Update(ctx, squirrel.Eq{"id": organization.ID, "deleted_at": nil}, map[string]any{
		"deleted_at": time.Now(),
		"updated_at": time.Now(),
	})
}

func (r *organizationsRepository) RestoreOrganization(ctx context.Context, id string) error {
	return r.organizationsRepo.Update(ctx, squirrel.And{
		squirrel.Eq{"id": id},
		squirrel.NotEq{"deleted_at": nil},
	}, map[string]any{
		"deleted_at": nil,
		"updated_at": time.Now(),
	})
}

func (r *organizationsRepository) CleanupArchivedOrganizations(ctx context.Context) error {
	threshold := time.Now().Add(-chi_archive.ArchivedDataRetentionPeriod)
	return r.organizationsRepo.Delete(ctx, squirrel.And{
		squirrel.NotEq{"deleted_at": nil},
		squirrel.LtOrEq{"deleted_at": threshold},
	})
}

func (r *organizationsRepository) GetOrganizationByID(ctx context.Context, id string) (*models.Organization, error) {
	return r.organizationsRepo.Get(ctx, squirrel.Eq{"id": id, "deleted_at": nil}, nil)
}

func (r *organizationsRepository) GetOrganizationByIDIncludeArchived(ctx context.Context, id string) (*models.Organization, error) {
	return r.organizationsRepo.Get(ctx, squirrel.Eq{"id": id}, nil)
}

func (r *organizationsRepository) SearchOrganizations(ctx context.Context, searchPhrase string, filter chi_archive.ArchiveFilter, limit, offset int) (*[]models.Organization, error) {
	filter, err := chi_archive.NormalizeArchiveFilter(filter)
	if err != nil {
		return nil, err
	}
	condition, sort := searchOrganizationsCondition(searchPhrase, filter)
	return r.organizationsRepo.PaginatedSelect(ctx, condition, nil, sort, uint64(limit), uint64(offset))
}
