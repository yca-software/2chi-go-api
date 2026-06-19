package team_repository

import (
	"context"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"

	"github.com/yca-software/2chi-go-api/internals/domains/core/models"
	chi_observer "github.com/yca-software/2chi-go-observer"
	chi_repository "github.com/yca-software/2chi-go-repository"
)

const TeamsTableName = "teams"

var TeamsColumns = []string{
	"id", "created_at", "updated_at", "organization_id", "name", "description",
}

type TeamsRepository interface {
	WithTx(tx chi_repository.Tx) TeamsRepository

	CreateTeam(ctx context.Context, team *models.Team) error
	UpdateTeam(ctx context.Context, team *models.Team) error
	DeleteTeam(ctx context.Context, organizationID, id string) error

	GetTeamByID(ctx context.Context, organizationID, id string) (*models.Team, error)
	ListTeamsByOrganizationID(ctx context.Context, organizationID string) (*[]models.Team, error)
}

type teamsRepository struct {
	teamsRepo chi_repository.Repository[models.Team]
}

func NewTeamsRepository(db *sqlx.DB, metricsHook chi_observer.QueryMetricsHook) TeamsRepository {
	return &teamsRepository{
		teamsRepo: chi_repository.NewRepository[models.Team](db, TeamsTableName, TeamsColumns, metricsHook),
	}
}

func (r *teamsRepository) WithTx(tx chi_repository.Tx) TeamsRepository {
	return &teamsRepository{
		teamsRepo: r.teamsRepo.WithTx(tx),
	}
}

func (r *teamsRepository) CreateTeam(ctx context.Context, team *models.Team) error {
	now := time.Now()
	return r.teamsRepo.Create(ctx, map[string]any{
		"id":              team.ID,
		"created_at":      now,
		"updated_at":      now,
		"organization_id": team.OrganizationID,
		"name":            team.Name,
		"description":     team.Description,
	})
}

func (r *teamsRepository) UpdateTeam(ctx context.Context, team *models.Team) error {
	return r.teamsRepo.Update(ctx, squirrel.And{
		squirrel.Eq{"id": team.ID},
		squirrel.Eq{"organization_id": team.OrganizationID},
	}, map[string]any{
		"name":        team.Name,
		"description": team.Description,
		"updated_at":  time.Now(),
	})
}

func (r *teamsRepository) DeleteTeam(ctx context.Context, organizationID, id string) error {
	return r.teamsRepo.Delete(ctx, squirrel.And{
		squirrel.Eq{"id": id},
		squirrel.Eq{"organization_id": organizationID},
	})
}

func (r *teamsRepository) GetTeamByID(ctx context.Context, organizationID, id string) (*models.Team, error) {
	return r.teamsRepo.Get(ctx, squirrel.And{
		squirrel.Eq{"id": id},
		squirrel.Eq{"organization_id": organizationID},
	}, nil)
}

func (r *teamsRepository) ListTeamsByOrganizationID(ctx context.Context, organizationID string) (*[]models.Team, error) {
	return r.teamsRepo.Select(ctx, squirrel.Eq{"organization_id": organizationID}, nil, "name ASC")
}
