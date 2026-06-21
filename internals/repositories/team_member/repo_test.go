//go:build integration

package team_member_repository_test

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
	team_member_repository "github.com/yca-software/2chi-go-api/internals/repositories/team_member"
	"github.com/yca-software/2chi-go-api/internals/packages/testutil"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_repository "github.com/yca-software/2chi-go-repository"
	chi_types "github.com/yca-software/2chi-go-types"
)

const (
	seedTeamsOrgID        = "22222222-2222-2222-2222-222222222201"
	seedTeamsUserID       = "11111111-1111-1111-1111-111111111201"
	seedTeamsActiveID     = "55555555-5555-5555-5555-555555555201"
	seedTeamsUpdateID     = "55555555-5555-5555-5555-555555555202"
	seedTeamsDeleteID     = "55555555-5555-5555-5555-555555555203"
	seedTeamsMemberID     = "66666666-6666-6666-6666-666666666201"
	seedTeamsDeleteMember = "66666666-6666-6666-6666-666666666202"
	seedTeamsNewMemberID  = "66666666-6666-6666-6666-666666666203"
	seedTeamsTxMemberID   = "66666666-6666-6666-6666-666666666204"
)

var seedCreatedAtTime = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

func TestMain(m *testing.M) {
	os.Exit(testutil.IntegrationTestMain(m))
}

func TestTeamMembersRepositorySuite(t *testing.T) {
	suite.Run(t, new(TeamMembersRepositorySuite))
}

type TeamMembersRepositorySuite struct {
	suite.Suite

	db   *sqlx.DB
	repo team_member_repository.TeamMembersRepository
	ctx  context.Context
}

func (s *TeamMembersRepositorySuite) SetupSuite() {
	testDB, err := testutil.GetIntegrationDB()
	s.Require().NoError(err)

	s.db, err = testDB.SQLx()
	s.Require().NoError(err)

	s.repo = team_member_repository.NewTeamMembersRepository(s.db, nil)
	s.ctx = context.Background()
}

func (s *TeamMembersRepositorySuite) SetupTest() {
	_, err := s.db.ExecContext(s.ctx, `
INSERT INTO users (
	id, created_at, deleted_at, first_name, last_name, language, email, password
) VALUES (
	'11111111-1111-1111-1111-111111111201', '2024-01-01T00:00:00Z', NULL, 'Team', 'User', 'en',
	'team-user@example.com', 'hash'
);
INSERT INTO organizations (id, created_at, deleted_at, name) VALUES (
	'22222222-2222-2222-2222-222222222201', '2024-01-01T00:00:00Z', NULL, 'Teams Org'
);
INSERT INTO teams (id, created_at, organization_id, name, description) VALUES
	('55555555-5555-5555-5555-555555555201', '2024-01-01T00:00:00Z', '22222222-2222-2222-2222-222222222201', 'Active Team', 'Active'),
	('55555555-5555-5555-5555-555555555202', '2024-01-01T00:00:00Z', '22222222-2222-2222-2222-222222222201', 'Update Team', 'Update'),
	('55555555-5555-5555-5555-555555555203', '2024-01-01T00:00:00Z', '22222222-2222-2222-2222-222222222201', 'Delete Team', 'Delete');
INSERT INTO team_members (id, created_at, organization_id, team_id, user_id) VALUES
	('66666666-6666-6666-6666-666666666201', '2024-01-01T00:00:00Z', '22222222-2222-2222-2222-222222222201', '55555555-5555-5555-5555-555555555201', '11111111-1111-1111-1111-111111111201'),
	('66666666-6666-6666-6666-666666666202', '2024-01-01T00:00:00Z', '22222222-2222-2222-2222-222222222201', '55555555-5555-5555-5555-555555555202', '11111111-1111-1111-1111-111111111201')`)
	s.Require().NoError(err)
}

func (s *TeamMembersRepositorySuite) TearDownTest() {
	_, err := s.db.ExecContext(s.ctx, `TRUNCATE TABLE team_members, teams, organizations, users CASCADE`)
	s.Require().NoError(err)
}

func (s *TeamMembersRepositorySuite) TestCreateTeamMember() {
	member := &models.TeamMember{
		ModelBase: chi_types.ModelBase{
			ID:        uuid.MustParse(seedTeamsNewMemberID),
			CreatedAt: seedCreatedAtTime,
		},
		OrganizationID: uuid.MustParse(seedTeamsOrgID),
		TeamID:         uuid.MustParse(seedTeamsDeleteID),
		UserID:         uuid.MustParse(seedTeamsUserID),
	}
	s.Require().NoError(s.repo.CreateTeamMember(s.ctx, member))

	got, err := s.repo.GetTeamMemberByID(s.ctx, seedTeamsOrgID, seedTeamsNewMemberID)
	s.Require().NoError(err)
	s.Equal(seedTeamsUserID, got.UserID.String())
}

func (s *TeamMembersRepositorySuite) TestGetTeamMemberByIDWithUser() {
	got, err := s.repo.GetTeamMemberByIDWithUser(s.ctx, seedTeamsOrgID, seedTeamsMemberID)
	s.Require().NoError(err)
	s.Equal("team-user@example.com", got.UserEmail)
}

func (s *TeamMembersRepositorySuite) TestListTeamMembersByTeamID() {
	activeRows, err := s.repo.ListTeamMembersByTeamID(s.ctx, seedTeamsOrgID, seedTeamsActiveID)
	s.Require().NoError(err)
	s.Len(*activeRows, 1)

	updateRows, err := s.repo.ListTeamMembersByTeamID(s.ctx, seedTeamsOrgID, seedTeamsUpdateID)
	s.Require().NoError(err)
	s.Len(*updateRows, 1)
}

func (s *TeamMembersRepositorySuite) TestListTeamMembersByUserID() {
	rows, err := s.repo.ListTeamMembersByUserID(s.ctx, seedTeamsUserID)
	s.Require().NoError(err)
	s.GreaterOrEqual(len(*rows), 2)
}

func (s *TeamMembersRepositorySuite) TestListTeamMembersByOrganizationID() {
	rows, err := s.repo.ListTeamMembersByOrganizationID(s.ctx, seedTeamsOrgID)
	s.Require().NoError(err)
	s.GreaterOrEqual(len(*rows), 2)
}

func (s *TeamMembersRepositorySuite) TestDeleteTeamMember() {
	s.Require().NoError(s.repo.DeleteTeamMember(s.ctx, seedTeamsOrgID, seedTeamsDeleteMember))
	_, err := s.repo.GetTeamMemberByID(s.ctx, seedTeamsOrgID, seedTeamsDeleteMember)
	s.requireNotFound(err)
}

func (s *TeamMembersRepositorySuite) TestWithTx() {
	member := &models.TeamMember{
		ModelBase: chi_types.ModelBase{
			ID:        uuid.MustParse(seedTeamsTxMemberID),
			CreatedAt: seedCreatedAtTime,
		},
		OrganizationID: uuid.MustParse(seedTeamsOrgID),
		TeamID:         uuid.MustParse(seedTeamsDeleteID),
		UserID:         uuid.MustParse(seedTeamsUserID),
	}
	err := chi_repository.RunInTx(s.ctx, s.db, nil, func(tx chi_repository.Tx) error {
		return s.repo.WithTx(tx).CreateTeamMember(s.ctx, member)
	})
	s.Require().NoError(err)

	got, err := s.repo.GetTeamMemberByID(s.ctx, seedTeamsOrgID, seedTeamsTxMemberID)
	s.Require().NoError(err)
	s.Equal(seedTeamsUserID, got.UserID.String())
}

func (s *TeamMembersRepositorySuite) requireNotFound(err error) {
	s.T().Helper()
	s.Require().Error(err)
	var apiErr *chi_error.Error
	s.Require().True(errors.As(err, &apiErr))
	s.Equal(http.StatusNotFound, apiErr.StatusCode)
}
