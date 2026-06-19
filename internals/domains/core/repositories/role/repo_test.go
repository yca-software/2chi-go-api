package role_repository_test

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
	role_repository "github.com/yca-software/2chi-go-api/internals/domains/core/repositories/role"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_repository "github.com/yca-software/2chi-go-repository"
	chi_test "github.com/yca-software/2chi-go-test"
	chi_types "github.com/yca-software/2chi-go-types"
)

const (
	seedRolesOrgID    = "22222222-2222-2222-2222-222222222101"
	seedRolesActiveID = "33333333-3333-3333-3333-333333333101"
	seedRolesUpdateID = "33333333-3333-3333-3333-333333333102"
	seedRolesDeleteID = "33333333-3333-3333-3333-333333333104"
	seedRolesNewID    = "33333333-3333-3333-3333-333333333105"
	seedRolesBulkID   = "33333333-3333-3333-3333-333333333106"
	seedRolesTxID     = "33333333-3333-3333-3333-333333333107"
)

var seedCreatedAtTime = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

func moduleMigrationsDir() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", "..", "..", "migrations"))
}

func TestMain(m *testing.M) {
	code := m.Run()
	chi_test.Cleanup()
	os.Exit(code)
}

func TestRolesRepositorySuite(t *testing.T) {
	suite.Run(t, new(RolesRepositorySuite))
}

type RolesRepositorySuite struct {
	suite.Suite

	db   *sqlx.DB
	repo role_repository.RolesRepository
	ctx  context.Context
}

func (s *RolesRepositorySuite) SetupSuite() {
	testDB, err := chi_test.Get(moduleMigrationsDir())
	s.Require().NoError(err)

	s.db, err = testDB.SQLx()
	s.Require().NoError(err)

	s.repo = role_repository.NewRolesRepository(s.db, nil)
	s.ctx = context.Background()
}

func (s *RolesRepositorySuite) SetupTest() {
	_, err := s.db.ExecContext(s.ctx, `
INSERT INTO organizations (id, created_at, deleted_at, name) VALUES (
	'22222222-2222-2222-2222-222222222101', '2024-01-01T00:00:00Z', NULL, 'Roles Org'
);
INSERT INTO roles (id, created_at, organization_id, name, description, permissions, locked) VALUES
	('33333333-3333-3333-3333-333333333101', '2024-01-01T00:00:00Z', '22222222-2222-2222-2222-222222222101', 'Active', 'Active role', '["org:read"]'::jsonb, false),
	('33333333-3333-3333-3333-333333333102', '2024-01-01T00:00:00Z', '22222222-2222-2222-2222-222222222101', 'Update Me', 'Update role', '["org:read"]'::jsonb, false),
	('33333333-3333-3333-3333-333333333103', '2024-01-01T00:00:00Z', '22222222-2222-2222-2222-222222222101', 'Locked', 'Locked role', '["org:read"]'::jsonb, true),
	('33333333-3333-3333-3333-333333333104', '2024-01-01T00:00:00Z', '22222222-2222-2222-2222-222222222101', 'Delete Me', 'Delete role', '["org:read"]'::jsonb, false)`)
	s.Require().NoError(err)
}

func (s *RolesRepositorySuite) TearDownTest() {
	_, err := s.db.ExecContext(s.ctx, `TRUNCATE TABLE roles, organizations CASCADE`)
	s.Require().NoError(err)
}

func (s *RolesRepositorySuite) TestCreateRole() {
	role := &models.Role{
		ModelBase: chi_types.ModelBase{
			ID:        uuid.MustParse(seedRolesNewID),
			CreatedAt: seedCreatedAtTime,
		},
		OrganizationID: uuid.MustParse(seedRolesOrgID),
		Name:           "New Role",
		Description:    "New role",
		Permissions:    models.RolePermissions{"org:write"},
	}
	s.Require().NoError(s.repo.CreateRole(s.ctx, role))

	got, err := s.repo.GetRoleByID(s.ctx, seedRolesOrgID, seedRolesNewID)
	s.Require().NoError(err)
	s.Equal("New Role", got.Name)
}

func (s *RolesRepositorySuite) TestCreateRoles() {
	roles := []models.Role{{
		ModelBase: chi_types.ModelBase{
			ID:        uuid.MustParse(seedRolesBulkID),
			CreatedAt: seedCreatedAtTime,
		},
		OrganizationID: uuid.MustParse(seedRolesOrgID),
		Name:           "Bulk Role",
		Description:    "Bulk role",
		Permissions:    models.RolePermissions{"org:read"},
	}}
	s.Require().NoError(s.repo.CreateRoles(s.ctx, &roles))

	got, err := s.repo.GetRoleByID(s.ctx, seedRolesOrgID, seedRolesBulkID)
	s.Require().NoError(err)
	s.Equal("Bulk Role", got.Name)
}

func (s *RolesRepositorySuite) TestUpdateRole() {
	role, err := s.repo.GetRoleByID(s.ctx, seedRolesOrgID, seedRolesUpdateID)
	s.Require().NoError(err)
	role.Name = "Updated Role"
	role.Permissions = models.RolePermissions{"org:write"}
	s.Require().NoError(s.repo.UpdateRole(s.ctx, role))

	got, err := s.repo.GetRoleByID(s.ctx, seedRolesOrgID, seedRolesUpdateID)
	s.Require().NoError(err)
	s.Equal("Updated Role", got.Name)
}

func (s *RolesRepositorySuite) TestDeleteRole() {
	s.Require().NoError(s.repo.DeleteRole(s.ctx, seedRolesOrgID, seedRolesDeleteID))
	_, err := s.repo.GetRoleByID(s.ctx, seedRolesOrgID, seedRolesDeleteID)
	s.requireNotFound(err)
}

func (s *RolesRepositorySuite) TestListRolesByOrganizationID() {
	rows, err := s.repo.ListRolesByOrganizationID(s.ctx, seedRolesOrgID)
	s.Require().NoError(err)
	s.GreaterOrEqual(len(*rows), 4)
}

func (s *RolesRepositorySuite) TestWithTx() {
	role := &models.Role{
		ModelBase: chi_types.ModelBase{
			ID:        uuid.MustParse(seedRolesTxID),
			CreatedAt: seedCreatedAtTime,
		},
		OrganizationID: uuid.MustParse(seedRolesOrgID),
		Name:           "Tx Role",
		Description:    "Tx role",
		Permissions:    models.RolePermissions{"org:read"},
	}
	err := chi_repository.RunInTx(s.ctx, s.db, nil, func(tx chi_repository.Tx) error {
		return s.repo.WithTx(tx).CreateRole(s.ctx, role)
	})
	s.Require().NoError(err)

	got, err := s.repo.GetRoleByID(s.ctx, seedRolesOrgID, seedRolesTxID)
	s.Require().NoError(err)
	s.Equal("Tx Role", got.Name)
}

func (s *RolesRepositorySuite) requireNotFound(err error) {
	s.T().Helper()
	s.Require().Error(err)
	var apiErr *chi_error.Error
	s.Require().True(errors.As(err, &apiErr))
	s.Equal(http.StatusNotFound, apiErr.StatusCode)
}
