package organization_member_repository_test

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
	organization_member_repository "github.com/yca-software/2chi-go-api/internals/domains/core/repositories/organization/member"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_repository "github.com/yca-software/2chi-go-repository"
	chi_test "github.com/yca-software/2chi-go-test"
	chi_types "github.com/yca-software/2chi-go-types"
)

const (
	seedMemberOrgID         = "22222222-2222-2222-2222-222222222001"
	seedMemberUserID        = "11111111-1111-1111-1111-111111111101"
	seedMemberSecondUserID  = "11111111-1111-1111-1111-111111111102"
	seedMemberRoleID        = "33333333-3333-3333-3333-333333333001"
	seedMemberSecondRoleID  = "33333333-3333-3333-3333-333333333002"
	seedMemberID            = "44444444-4444-4444-4444-444444444001"
	seedMemberDeleteID      = "44444444-4444-4444-4444-444444444002"
	seedMemberNewOrgID      = "22222222-2222-2222-2222-22222222200b"
	seedMemberNewMemberID   = "44444444-4444-4444-4444-444444444003"
	seedMemberTxID          = "44444444-4444-4444-4444-444444444004"
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

func TestOrganizationMembersRepositorySuite(t *testing.T) {
	suite.Run(t, new(OrganizationMembersRepositorySuite))
}

type OrganizationMembersRepositorySuite struct {
	suite.Suite

	db   *sqlx.DB
	repo organization_member_repository.OrganizationMembersRepository
	ctx  context.Context
}

func (s *OrganizationMembersRepositorySuite) SetupSuite() {
	testDB, err := chi_test.Get(moduleMigrationsDir())
	s.Require().NoError(err)

	s.db, err = testDB.SQLx()
	s.Require().NoError(err)

	s.repo = organization_member_repository.NewOrganizationMembersRepository(s.db, nil)
	s.ctx = context.Background()
}

func (s *OrganizationMembersRepositorySuite) SetupTest() {
	_, err := s.db.ExecContext(s.ctx, `
INSERT INTO users (
	id, created_at, deleted_at, first_name, last_name, language, email, password
) VALUES
	('11111111-1111-1111-1111-111111111101', '2024-01-01T00:00:00Z', NULL, 'Org', 'Member', 'en',
		'member@example.com', 'hash'),
	('11111111-1111-1111-1111-111111111102', '2024-01-01T00:00:00Z', NULL, 'Second', 'Member', 'en',
		'second-member@example.com', 'hash');
INSERT INTO organizations (id, created_at, deleted_at, name) VALUES (
	'22222222-2222-2222-2222-222222222001', '2024-01-01T00:00:00Z', NULL, 'Active Org'
);
INSERT INTO roles (id, created_at, organization_id, name, description, permissions, locked) VALUES
	('33333333-3333-3333-3333-333333333001', '2024-01-01T00:00:00Z', '22222222-2222-2222-2222-222222222001', 'Admin', 'Admin role', '["org:read"]'::jsonb, false),
	('33333333-3333-3333-3333-333333333002', '2024-01-01T00:00:00Z', '22222222-2222-2222-2222-222222222001', 'Member', 'Member role', '["org:read"]'::jsonb, false);
INSERT INTO organization_members (id, created_at, organization_id, user_id, role_id) VALUES
	('44444444-4444-4444-4444-444444444001', '2024-01-01T00:00:00Z', '22222222-2222-2222-2222-222222222001', '11111111-1111-1111-1111-111111111101', '33333333-3333-3333-3333-333333333001'),
	('44444444-4444-4444-4444-444444444002', '2024-01-01T00:00:00Z', '22222222-2222-2222-2222-222222222001', '11111111-1111-1111-1111-111111111102', '33333333-3333-3333-3333-333333333002')`)
	s.Require().NoError(err)
}

func (s *OrganizationMembersRepositorySuite) TearDownTest() {
	_, err := s.db.ExecContext(s.ctx, `TRUNCATE TABLE organization_members, roles, organizations, users CASCADE`)
	s.Require().NoError(err)
}

func (s *OrganizationMembersRepositorySuite) TestGetOrganizationMemberByID() {
	got, err := s.repo.GetOrganizationMemberByID(s.ctx, seedMemberOrgID, seedMemberUserID)
	s.Require().NoError(err)
	s.Equal(seedMemberID, got.ID.String())
}

func (s *OrganizationMembersRepositorySuite) TestGetOrganizationMemberByIDWithUser() {
	got, err := s.repo.GetOrganizationMemberByIDWithUser(s.ctx, seedMemberOrgID, seedMemberUserID)
	s.Require().NoError(err)
	s.Equal("member@example.com", got.UserEmail)
}

func (s *OrganizationMembersRepositorySuite) TestGetOrganizationMemberByIDWithOrganizationAndRole() {
	got, err := s.repo.GetOrganizationMemberByIDWithOrganizationAndRole(s.ctx, seedMemberOrgID, seedMemberUserID)
	s.Require().NoError(err)
	s.Equal("Active Org", got.OrganizationName)
	s.Equal("Admin", got.RoleName)
}

func (s *OrganizationMembersRepositorySuite) TestListByUserID() {
	rows, err := s.repo.ListByUserID(s.ctx, seedMemberUserID)
	s.Require().NoError(err)
	s.Require().Len(*rows, 1)
	s.Equal(seedMemberOrgID, (*rows)[0].OrganizationID.String())
}

func (s *OrganizationMembersRepositorySuite) TestListByOrganizationID() {
	rows, err := s.repo.ListByOrganizationID(s.ctx, seedMemberOrgID)
	s.Require().NoError(err)
	s.GreaterOrEqual(len(*rows), 2)
}

func (s *OrganizationMembersRepositorySuite) TestListUserEmailsForRole() {
	emails, err := s.repo.ListUserEmailsForRole(s.ctx, seedMemberOrgID, seedMemberRoleID)
	s.Require().NoError(err)
	s.Equal([]string{"member@example.com"}, emails)
}

func (s *OrganizationMembersRepositorySuite) TestUpdateOrganizationMember() {
	member, err := s.repo.GetOrganizationMemberByMembershipID(s.ctx, seedMemberOrgID, seedMemberID)
	s.Require().NoError(err)
	member.RoleID = uuid.MustParse(seedMemberSecondRoleID)
	s.Require().NoError(s.repo.UpdateOrganizationMember(s.ctx, member))

	got, err := s.repo.GetOrganizationMemberByMembershipID(s.ctx, seedMemberOrgID, seedMemberID)
	s.Require().NoError(err)
	s.Equal(seedMemberSecondRoleID, got.RoleID.String())
}

func (s *OrganizationMembersRepositorySuite) TestDeleteOrganizationMember() {
	s.Require().NoError(s.repo.DeleteOrganizationMember(s.ctx, seedMemberOrgID, seedMemberUserID))
	_, err := s.repo.GetOrganizationMemberByID(s.ctx, seedMemberOrgID, seedMemberUserID)
	s.requireNotFound(err)
}

func (s *OrganizationMembersRepositorySuite) TestCreateOrganizationMember() {
	_, err := s.db.ExecContext(s.ctx, `INSERT INTO organizations (id, created_at, deleted_at, name) VALUES (
		'22222222-2222-2222-2222-22222222200b', '2024-01-01T00:00:00Z', NULL, 'Member Org')`)
	s.Require().NoError(err)

	member := &models.OrganizationMember{
		ModelBase: chi_types.ModelBase{
			ID:        uuid.MustParse(seedMemberNewMemberID),
			CreatedAt: seedCreatedAtTime,
		},
		OrganizationID: uuid.MustParse(seedMemberNewOrgID),
		UserID:         uuid.MustParse(seedMemberUserID),
		RoleID:         uuid.MustParse(seedMemberRoleID),
	}
	s.Require().NoError(s.repo.CreateOrganizationMember(s.ctx, member))

	got, err := s.repo.GetOrganizationMemberByMembershipID(s.ctx, seedMemberNewOrgID, seedMemberNewMemberID)
	s.Require().NoError(err)
	s.Equal(seedMemberUserID, got.UserID.String())
}

func (s *OrganizationMembersRepositorySuite) TestWithTx() {
	_, err := s.db.ExecContext(s.ctx, `INSERT INTO users (
		id, created_at, deleted_at, first_name, last_name, language, email, password
	) VALUES (
		'11111111-1111-1111-1111-111111111103', '2024-01-01T00:00:00Z', NULL, 'Tx', 'Member', 'en',
		'tx-member@example.com', 'hash')`)
	s.Require().NoError(err)

	member := &models.OrganizationMember{
		ModelBase: chi_types.ModelBase{
			ID:        uuid.MustParse(seedMemberTxID),
			CreatedAt: seedCreatedAtTime,
		},
		OrganizationID: uuid.MustParse(seedMemberOrgID),
		UserID:         uuid.MustParse("11111111-1111-1111-1111-111111111103"),
		RoleID:         uuid.MustParse(seedMemberRoleID),
	}
	err = chi_repository.RunInTx(s.ctx, s.db, nil, func(tx chi_repository.Tx) error {
		return s.repo.WithTx(tx).CreateOrganizationMember(s.ctx, member)
	})
	s.Require().NoError(err)

	got, err := s.repo.GetOrganizationMemberByMembershipID(s.ctx, seedMemberOrgID, seedMemberTxID)
	s.Require().NoError(err)
	s.Equal("11111111-1111-1111-1111-111111111103", got.UserID.String())
}

func (s *OrganizationMembersRepositorySuite) requireNotFound(err error) {
	s.T().Helper()
	s.Require().Error(err)
	var apiErr *chi_error.Error
	s.Require().True(errors.As(err, &apiErr))
	s.Equal(http.StatusNotFound, apiErr.StatusCode)
}
