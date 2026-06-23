//go:build integration

package team_repository_test

import (
	"context"
	"errors"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/suite"

	"github.com/yca-software/2chi-go-api/internals/models"
	team_repository "github.com/yca-software/2chi-go-api/internals/repositories/team"
	"github.com/yca-software/2chi-go-api/internals/packages/testutil"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_repository "github.com/yca-software/2chi-go-repository"
	chi_types "github.com/yca-software/2chi-go-types"
)

const (
	seedTeamsOrgID    = "22222222-2222-2222-2222-222222222201"
	seedTeamsActiveID = "55555555-5555-5555-5555-555555555201"
	seedTeamsUpdateID = "55555555-5555-5555-5555-555555555202"
	seedTeamsDeleteID = "55555555-5555-5555-5555-555555555203"
	seedTeamsNewID    = "55555555-5555-5555-5555-555555555204"
	seedTeamsTxID     = "55555555-5555-5555-5555-555555555205"
)

var seedCreatedAtTime = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

func TestMain(m *testing.M) {
	os.Exit(testutil.IntegrationTestMain(m))
}

func TestRepositorySuite(t *testing.T) {
	suite.Run(t, new(RepositorySuite))
}

type RepositorySuite struct {
	suite.Suite

	db   *sqlx.DB
	repo team_repository.Repository
	ctx  context.Context
}

func (s *RepositorySuite) SetupSuite() {
	testDB, err := testutil.GetIntegrationDB()
	s.Require().NoError(err)

	s.db, err = testDB.SQLx()
	s.Require().NoError(err)

	s.repo = team_repository.NewRepository(s.db, nil)
	s.ctx = context.Background()
}

func (s *RepositorySuite) SetupTest() {
	_, err := s.db.ExecContext(s.ctx, `
INSERT INTO organizations (id, created_at, deleted_at, name, address, city, zip, country, place_id, geo, timezone) VALUES (
	'22222222-2222-2222-2222-222222222201', '2024-01-01T00:00:00Z', NULL, 'Teams Org', '1 Main St', 'Oslo', '0001', 'NO', 'place_seed_team', ST_SetSRID(ST_MakePoint(10.7, 59.9), 4326), 'Europe/Oslo'
);
INSERT INTO teams (id, created_at, organization_id, name, description) VALUES
	('55555555-5555-5555-5555-555555555201', '2024-01-01T00:00:00Z', '22222222-2222-2222-2222-222222222201', 'Active Team', 'Active'),
	('55555555-5555-5555-5555-555555555202', '2024-01-01T00:00:00Z', '22222222-2222-2222-2222-222222222201', 'Update Team', 'Update'),
	('55555555-5555-5555-5555-555555555203', '2024-01-01T00:00:00Z', '22222222-2222-2222-2222-222222222201', 'Delete Team', 'Delete')`)
	s.Require().NoError(err)
}

func (s *RepositorySuite) TearDownTest() {
	_, err := s.db.ExecContext(s.ctx, `TRUNCATE TABLE teams, organizations CASCADE`)
	s.Require().NoError(err)
}

func (s *RepositorySuite) TestCreate() {
	team := &models.Team{
		ModelBase: chi_types.ModelBase{
			ID:        uuid.MustParse(seedTeamsNewID),
			CreatedAt: seedCreatedAtTime,
		},
		OrganizationID: uuid.MustParse(seedTeamsOrgID),
		Name:           "New Team",
		Description:    "New",
	}
	s.Require().NoError(s.repo.Create(s.ctx, team))

	got, err := s.repo.GetByID(s.ctx, seedTeamsOrgID, seedTeamsNewID)
	s.Require().NoError(err)
	s.Equal("New Team", got.Name)
}

func (s *RepositorySuite) TestUpdate() {
	team, err := s.repo.GetByID(s.ctx, seedTeamsOrgID, seedTeamsUpdateID)
	s.Require().NoError(err)
	team.Name = "Updated Team"
	s.Require().NoError(s.repo.Update(s.ctx, team))

	got, err := s.repo.GetByID(s.ctx, seedTeamsOrgID, seedTeamsUpdateID)
	s.Require().NoError(err)
	s.Equal("Updated Team", got.Name)
}

func (s *RepositorySuite) TestDelete() {
	s.Require().NoError(s.repo.Delete(s.ctx, seedTeamsOrgID, seedTeamsDeleteID))
	_, err := s.repo.GetByID(s.ctx, seedTeamsOrgID, seedTeamsDeleteID)
	s.requireNotFound(err)
}

func (s *RepositorySuite) TestListByOrganizationID() {
	rows, err := s.repo.ListByOrganizationID(s.ctx, seedTeamsOrgID)
	s.Require().NoError(err)
	s.GreaterOrEqual(len(*rows), 3)
}

func (s *RepositorySuite) TestWithTx() {
	team := &models.Team{
		ModelBase: chi_types.ModelBase{
			ID:        uuid.MustParse(seedTeamsTxID),
			CreatedAt: seedCreatedAtTime,
		},
		OrganizationID: uuid.MustParse(seedTeamsOrgID),
		Name:           "Tx Team",
		Description:    "Tx",
	}
	err := chi_repository.RunInTx(s.ctx, s.db, nil, func(tx chi_repository.Tx) error {
		return s.repo.WithTx(tx).Create(s.ctx, team)
	})
	s.Require().NoError(err)

	got, err := s.repo.GetByID(s.ctx, seedTeamsOrgID, seedTeamsTxID)
	s.Require().NoError(err)
	s.Equal("Tx Team", got.Name)
}

func (s *RepositorySuite) requireNotFound(err error) {
	s.T().Helper()
	s.Require().Error(err)
	var apiErr *chi_error.Error
	s.Require().True(errors.As(err, &apiErr))
	s.Equal(http.StatusNotFound, apiErr.StatusCode)
}
