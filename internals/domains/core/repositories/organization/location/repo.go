package organization_location_repository

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
	OrganizationLocationsTableName = "organization_locations"
)

var (
	OrganizationLocationsColumns = []string{
		"id", "created_at", "updated_at", "organization_id",
		"address", "city", "zip", "country", "place_id", "geo", "timezone",
	}
)

type OrganizationLocationsRepository interface {
	WithTx(tx chi_repository.Tx) OrganizationLocationsRepository

	CreateOrganizationLocation(ctx context.Context, location *models.OrganizationLocation) error
	UpdateOrganizationLocation(ctx context.Context, location *models.OrganizationLocation) error

	GetOrganizationLocationByID(ctx context.Context, organizationID, id string) (*models.OrganizationLocation, error)
	GetOrganizationLocationByOrganizationID(ctx context.Context, organizationID string) (*models.OrganizationLocation, error)
}

type organizationLocationsRepository struct {
	organizationLocationsRepo chi_repository.Repository[models.OrganizationLocation]
}

func NewOrganizationLocationsRepository(db *sqlx.DB, metricsHook chi_observer.QueryMetricsHook) OrganizationLocationsRepository {
	return &organizationLocationsRepository{
		organizationLocationsRepo: chi_repository.NewRepository[models.OrganizationLocation](db, OrganizationLocationsTableName, OrganizationLocationsColumns, metricsHook),
	}
}

func (r *organizationLocationsRepository) WithTx(tx chi_repository.Tx) OrganizationLocationsRepository {
	return &organizationLocationsRepository{
		organizationLocationsRepo: r.organizationLocationsRepo.WithTx(tx),
	}
}

func (r *organizationLocationsRepository) CreateOrganizationLocation(ctx context.Context, location *models.OrganizationLocation) error {
	now := time.Now()
	return r.organizationLocationsRepo.Create(ctx, map[string]any{
		"id":              location.ID,
		"created_at":      now,
		"updated_at":      now,
		"organization_id": location.OrganizationID,
		"address":         location.Address,
		"city":            location.City,
		"zip":             location.Zip,
		"country":         location.Country,
		"place_id":        location.PlaceID,
		"geo":             location.Geo,
		"timezone":        location.Timezone,
	})
}

func (r *organizationLocationsRepository) UpdateOrganizationLocation(ctx context.Context, location *models.OrganizationLocation) error {
	return r.organizationLocationsRepo.Update(ctx, squirrel.And{
		squirrel.Eq{"id": location.ID},
		squirrel.Eq{"organization_id": location.OrganizationID},
	}, map[string]any{
		"address":    location.Address,
		"city":       location.City,
		"zip":        location.Zip,
		"country":    location.Country,
		"place_id":   location.PlaceID,
		"geo":        location.Geo,
		"timezone":   location.Timezone,
		"updated_at": time.Now(),
	})
}

func (r *organizationLocationsRepository) GetOrganizationLocationByID(ctx context.Context, organizationID, id string) (*models.OrganizationLocation, error) {
	return r.organizationLocationsRepo.Get(ctx, squirrel.And{
		squirrel.Eq{"id": id},
		squirrel.Eq{"organization_id": organizationID},
	}, nil)
}

func (r *organizationLocationsRepository) GetOrganizationLocationByOrganizationID(ctx context.Context, organizationID string) (*models.OrganizationLocation, error) {
	return r.organizationLocationsRepo.Get(ctx, squirrel.Eq{"organization_id": organizationID}, nil)
}
