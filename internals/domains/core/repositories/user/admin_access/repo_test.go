package admin_access_repository_test

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/suite"

	admin_access_repository "github.com/yca-software/2chi-go-api/internals/domains/core/repositories/user/admin_access"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_repository "github.com/yca-software/2chi-go-repository"
	chi_test "github.com/yca-software/2chi-go-test"
)

const (
	seedAdminUserID    = "22222222-2222-2222-2222-222222222001"
	seedNonAdminUserID = "22222222-2222-2222-2222-222222222002"
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

func TestAdminAccessRepositorySuite(t *testing.T) {
	suite.Run(t, new(AdminAccessRepositorySuite))
}

type AdminAccessRepositorySuite struct {
	suite.Suite

	db   *sqlx.DB
	repo admin_access_repository.AdminAccessRepository
	ctx  context.Context
}

func (s *AdminAccessRepositorySuite) SetupSuite() {
	testDB, err := chi_test.Get(moduleMigrationsDir())
	s.Require().NoError(err)

	s.db, err = testDB.SQLx()
	s.Require().NoError(err)

	s.repo = admin_access_repository.NewAdminAccessRepository(s.db, nil)
	s.ctx = context.Background()
}

func (s *AdminAccessRepositorySuite) SetupTest() {
	_, err := s.db.ExecContext(s.ctx, `
INSERT INTO users (
	id, created_at, deleted_at, first_name, last_name, language, email, password
) VALUES
	('22222222-2222-2222-2222-222222222001', '2024-01-01T00:00:00Z', NULL, 'Admin', 'User', 'en', 'admin@example.com', 'hash'),
	('22222222-2222-2222-2222-222222222002', '2024-01-01T00:00:00Z', NULL, 'Regular', 'User', 'en', 'regular@example.com', 'hash')`)
	s.Require().NoError(err)

	_, err = s.db.ExecContext(s.ctx, `
INSERT INTO admin_access (user_id, created_at) VALUES
	('22222222-2222-2222-2222-222222222001', '2024-01-01T00:00:00Z')`)
	s.Require().NoError(err)
}

func (s *AdminAccessRepositorySuite) TearDownTest() {
	_, err := s.db.ExecContext(s.ctx, `TRUNCATE TABLE users CASCADE`)
	s.Require().NoError(err)
}

func (s *AdminAccessRepositorySuite) TestGetAdminAccessByUserID() {
	got, err := s.repo.GetAdminAccessByUserID(s.ctx, seedAdminUserID)
	s.Require().NoError(err)
	s.Equal(seedAdminUserID, got.UserID.String())
	s.True(got.CreatedAt.Equal(seedCreatedAtTime))
}

func (s *AdminAccessRepositorySuite) TestGetAdminAccessByUserID_NotFound() {
	_, err := s.repo.GetAdminAccessByUserID(s.ctx, seedNonAdminUserID)
	s.requireNotFound(err)
}

func (s *AdminAccessRepositorySuite) TestWithTx() {
	var gotUserID string
	err := chi_repository.RunInTx(s.ctx, s.db, nil, func(tx chi_repository.Tx) error {
		got, err := s.repo.WithTx(tx).GetAdminAccessByUserID(s.ctx, seedAdminUserID)
		if err != nil {
			return err
		}
		gotUserID = got.UserID.String()
		return nil
	})
	s.Require().NoError(err)
	s.Equal(seedAdminUserID, gotUserID)
}

func (s *AdminAccessRepositorySuite) requireNotFound(err error) {
	s.T().Helper()
	s.Require().Error(err)
	var apiErr *chi_error.Error
	s.Require().True(errors.As(err, &apiErr))
	s.Equal(http.StatusNotFound, apiErr.StatusCode)
}
