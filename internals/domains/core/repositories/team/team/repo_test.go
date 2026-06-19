package team_repository_test

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/suite"

	"github.com/yca-software/2chi-go-api/internals/domains/core/models"
	team_repository "github.com/yca-software/2chi-go-api/internals/domains/core/repositories/team/team"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_repository "github.com/yca-software/2chi-go-repository"
	chi_test "github.com/yca-software/2chi-go-test"
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

func moduleMigrationsDir() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", "..", "..", "..", "migrations"))
}

func TestMain(m *testing.M) {
	code := m.Run()
	chi_test.Cleanup()
	os.Exit(code)
}

func TestTeamsRepositorySuite(t *testing.T) {
	suite.Run(t, new(TeamsRepositorySuite))
}

type TeamsRepositorySuite struct {
	suite.Suite

	db   *sqlx.DB
	repo team_repository.TeamsRepository
	ctx  context.Context
}

func (s *TeamsRepositorySuite) SetupSuite() {
	testDB, err := chi_test.Get(moduleMigrationsDir())
	s.Require().NoError(err)

	s.db, err = testDB.SQLx()
	s.Require().NoError(err)

	s.repo = team_repository.NewTeamsRepository(s.db, nil)
	s.ctx = context.Background()
}

func (s *TeamsRepositorySuite) SetupTest() {
	_, err := s.db.ExecContext(s.ctx, `
INSERT INTO organizations (id, created_at, deleted_at, name) VALUES (
	'22222222-2222-2222-2222-222222222201', '2024-01-01T00:00:00Z', NULL, 'Teams Org'
);
INSERT INTO teams (id, created_at, organization_id, name, description) VALUES
	('55555555-5555-5555-5555-555555555201', '2024-01-01T00:00:00Z', '22222222-2222-2222-2222-222222222201', 'Active Team', 'Active'),
	('55555555-5555-5555-5555-555555555202', '2024-01-01T00:00:00Z', '22222222-2222-2222-2222-222222222201', 'Update Team', 'Update'),
	('55555555-5555-5555-5555-555555555203', '2024-01-01T00:00:00Z', '22222222-2222-2222-2222-222222222201', 'Delete Team', 'Delete')`)
	s.Require().NoError(err)
}

func (s *TeamsRepositorySuite) TearDownTest() {
	_, err := s.db.ExecContext(s.ctx, `TRUNCATE TABLE teams, organizations CASCADE`)
	s.Require().NoError(err)
}

func (s *TeamsRepositorySuite) TestCreateTeam() {
	team := &models.Team{
		ModelBase: chi_types.ModelBase{
			ID:        uuid.MustParse(seedTeamsNewID),
			CreatedAt: seedCreatedAtTime,
		},
		OrganizationID: uuid.MustParse(seedTeamsOrgID),
		Name:           "New Team",
		Description:    "New",
	}
	s.Require().NoError(s.repo.CreateTeam(s.ctx, team))

	got, err := s.repo.GetTeamByID(s.ctx, seedTeamsOrgID, seedTeamsNewID)
	s.Require().NoError(err)
	s.Equal("New Team", got.Name)
}

func (s *TeamsRepositorySuite) TestUpdateTeam() {
	team, err := s.repo.GetTeamByID(s.ctx, seedTeamsOrgID, seedTeamsUpdateID)
	s.Require().NoError(err)
	team.Name = "Updated Team"
	s.Require().NoError(s.repo.UpdateTeam(s.ctx, team))

	got, err := s.repo.GetTeamByID(s.ctx, seedTeamsOrgID, seedTeamsUpdateID)
	s.Require().NoError(err)
	s.Equal("Updated Team", got.Name)
}

func (s *TeamsRepositorySuite) TestDeleteTeam() {
	s.Require().NoError(s.repo.DeleteTeam(s.ctx, seedTeamsOrgID, seedTeamsDeleteID))
	_, err := s.repo.GetTeamByID(s.ctx, seedTeamsOrgID, seedTeamsDeleteID)
	s.requireNotFound(err)
}

func (s *TeamsRepositorySuite) TestListTeamsByOrganizationID() {
	rows, err := s.repo.ListTeamsByOrganizationID(s.ctx, seedTeamsOrgID)
	s.Require().NoError(err)
	s.GreaterOrEqual(len(*rows), 3)
}

func (s *TeamsRepositorySuite) TestWithTx() {
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
		return s.repo.WithTx(tx).CreateTeam(s.ctx, team)
	})
	s.Require().NoError(err)

	got, err := s.repo.GetTeamByID(s.ctx, seedTeamsOrgID, seedTeamsTxID)
	s.Require().NoError(err)
	s.Equal("Tx Team", got.Name)
}

func (s *TeamsRepositorySuite) requireNotFound(err error) {
	s.T().Helper()
	s.Require().Error(err)
	var apiErr *chi_error.Error
	s.Require().True(errors.As(err, &apiErr))
	s.Equal(http.StatusNotFound, apiErr.StatusCode)
}
