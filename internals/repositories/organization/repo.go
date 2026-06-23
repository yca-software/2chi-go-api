package organization_repository

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
	TableName = "organizations"
)

var (
	Columns = []string{"id", "created_at", "updated_at", "deleted_at", "name", "address", "city", "zip", "country", "place_id", "geo", "timezone"}
)

type Repository interface {
	WithTx(tx chi_repository.Tx) Repository

	Create(ctx context.Context, organization *models.Organization) error
	Update(ctx context.Context, organization *models.Organization) error
	Archive(ctx context.Context, organization *models.Organization) error
	Restore(ctx context.Context, id string) error
	CleanupArchived(ctx context.Context) error

	GetByID(ctx context.Context, id string) (*models.Organization, error)
	GetByIDIncludeArchived(ctx context.Context, id string) (*models.Organization, error)
	Search(ctx context.Context, searchPhrase string, filter chi_archive.ArchiveFilter, limit, offset int) (*[]models.Organization, error)
}

type repository struct {
	repo chi_repository.Repository[models.Organization]
}

func NewRepository(db *sqlx.DB, metricsHook chi_observer.QueryMetricsHook) Repository {
	return &repository{
		repo: chi_repository.NewRepository[models.Organization](db, TableName, Columns, metricsHook),
	}
}

func (r *repository) WithTx(tx chi_repository.Tx) Repository {
	return &repository{
		repo: r.repo.WithTx(tx),
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

func (r *repository) Create(ctx context.Context, organization *models.Organization) error {
	now := time.Now()
	return r.repo.Create(ctx, map[string]any{
		"id":         organization.ID,
		"created_at": now,
		"updated_at": now,
		"name":       organization.Name,
		"address":    organization.Address,
		"city":       organization.City,
		"zip":        organization.Zip,
		"country":    organization.Country,
		"place_id":   organization.PlaceID,
		"geo":        organization.Geo,
		"timezone":   organization.Timezone,
	})
}

func (r *repository) Update(ctx context.Context, organization *models.Organization) error {
	return r.repo.Update(ctx, squirrel.Eq{"id": organization.ID, "deleted_at": nil}, map[string]any{
		"name":       organization.Name,
		"updated_at": time.Now(),
		"address":    organization.Address,
		"city":       organization.City,
		"zip":        organization.Zip,
		"country":    organization.Country,
		"place_id":   organization.PlaceID,
		"geo":        organization.Geo,
		"timezone":   organization.Timezone,
	})
}

func (r *repository) Archive(ctx context.Context, organization *models.Organization) error {
	return r.repo.Update(ctx, squirrel.Eq{"id": organization.ID, "deleted_at": nil}, map[string]any{
		"deleted_at": time.Now(),
		"updated_at": time.Now(),
	})
}

func (r *repository) Restore(ctx context.Context, id string) error {
	return r.repo.Update(ctx, squirrel.And{
		squirrel.Eq{"id": id},
		squirrel.NotEq{"deleted_at": nil},
	}, map[string]any{
		"deleted_at": nil,
		"updated_at": time.Now(),
	})
}

func (r *repository) CleanupArchived(ctx context.Context) error {
	threshold := time.Now().Add(-chi_archive.ArchivedDataRetentionPeriod)
	return r.repo.Delete(ctx, squirrel.And{
		squirrel.NotEq{"deleted_at": nil},
		squirrel.LtOrEq{"deleted_at": threshold},
	})
}

func (r *repository) GetByID(ctx context.Context, id string) (*models.Organization, error) {
	return r.repo.Get(ctx, squirrel.Eq{"id": id, "deleted_at": nil}, nil)
}

func (r *repository) GetByIDIncludeArchived(ctx context.Context, id string) (*models.Organization, error) {
	return r.repo.Get(ctx, squirrel.Eq{"id": id}, nil)
}

func (r *repository) Search(ctx context.Context, searchPhrase string, filter chi_archive.ArchiveFilter, limit, offset int) (*[]models.Organization, error) {
	filter, err := chi_archive.NormalizeArchiveFilter(filter)
	if err != nil {
		return nil, err
	}
	condition, sort := searchOrganizationsCondition(searchPhrase, filter)
	return r.repo.PaginatedSelect(ctx, condition, nil, sort, uint64(limit), uint64(offset))
}
