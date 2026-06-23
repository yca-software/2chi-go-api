package team_repository

import (
	"context"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"

	"github.com/yca-software/2chi-go-api/internals/models"
	chi_observer "github.com/yca-software/2chi-go-observer"
	chi_repository "github.com/yca-software/2chi-go-repository"
)

const TableName = "teams"

var Columns = []string{
	"id", "created_at", "updated_at", "organization_id", "name", "description",
}

type Repository interface {
	WithTx(tx chi_repository.Tx) Repository

	Create(ctx context.Context, team *models.Team) error
	Update(ctx context.Context, team *models.Team) error
	Delete(ctx context.Context, organizationID, id string) error

	GetByID(ctx context.Context, organizationID, id string) (*models.Team, error)
	ListByOrganizationID(ctx context.Context, organizationID string) (*[]models.Team, error)
}

type repository struct {
	repo chi_repository.Repository[models.Team]
}

func NewRepository(db *sqlx.DB, metricsHook chi_observer.QueryMetricsHook) Repository {
	return &repository{
		repo: chi_repository.NewRepository[models.Team](db, TableName, Columns, metricsHook),
	}
}

func (r *repository) WithTx(tx chi_repository.Tx) Repository {
	return &repository{
		repo: r.repo.WithTx(tx),
	}
}

func (r *repository) Create(ctx context.Context, team *models.Team) error {
	now := time.Now()
	return r.repo.Create(ctx, map[string]any{
		"id":              team.ID,
		"created_at":      now,
		"updated_at":      now,
		"organization_id": team.OrganizationID,
		"name":            team.Name,
		"description":     team.Description,
	})
}

func (r *repository) Update(ctx context.Context, team *models.Team) error {
	return r.repo.Update(ctx, squirrel.And{
		squirrel.Eq{"id": team.ID},
		squirrel.Eq{"organization_id": team.OrganizationID},
	}, map[string]any{
		"name":        team.Name,
		"description": team.Description,
		"updated_at":  time.Now(),
	})
}

func (r *repository) Delete(ctx context.Context, organizationID, id string) error {
	return r.repo.Delete(ctx, squirrel.And{
		squirrel.Eq{"id": id},
		squirrel.Eq{"organization_id": organizationID},
	})
}

func (r *repository) GetByID(ctx context.Context, organizationID, id string) (*models.Team, error) {
	return r.repo.Get(ctx, squirrel.And{
		squirrel.Eq{"id": id},
		squirrel.Eq{"organization_id": organizationID},
	}, nil)
}

func (r *repository) ListByOrganizationID(ctx context.Context, organizationID string) (*[]models.Team, error) {
	return r.repo.Select(ctx, squirrel.Eq{"organization_id": organizationID}, nil, "name ASC")
}
