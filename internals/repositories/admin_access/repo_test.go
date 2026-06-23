//go:build integration

package admin_access_repository_test

import (
	"context"
	"errors"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/suite"

	"github.com/yca-software/2chi-go-api/internals/packages/testutil"
	admin_access_repository "github.com/yca-software/2chi-go-api/internals/repositories/admin_access"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_repository "github.com/yca-software/2chi-go-repository"
)

const (
	seedAdminUserID    = "22222222-2222-2222-2222-222222222001"
	seedNonAdminUserID = "22222222-2222-2222-2222-222222222002"
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
	repo admin_access_repository.Repository
	ctx  context.Context
}

func (s *RepositorySuite) SetupSuite() {
	testDB, err := testutil.GetIntegrationDB()
	s.Require().NoError(err)

	s.db, err = testDB.SQLx()
	s.Require().NoError(err)

	s.repo = admin_access_repository.NewRepository(s.db, nil)
	s.ctx = context.Background()
}

func (s *RepositorySuite) SetupTest() {
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

func (s *RepositorySuite) TearDownTest() {
	_, err := s.db.ExecContext(s.ctx, `TRUNCATE TABLE users CASCADE`)
	s.Require().NoError(err)
}

func (s *RepositorySuite) TestGetByUserID() {
	got, err := s.repo.GetByUserID(s.ctx, seedAdminUserID)
	s.Require().NoError(err)
	s.Equal(seedAdminUserID, got.UserID.String())
	s.True(got.CreatedAt.Equal(seedCreatedAtTime))
}

func (s *RepositorySuite) TestGetByUserID_NotFound() {
	_, err := s.repo.GetByUserID(s.ctx, seedNonAdminUserID)
	s.requireNotFound(err)
}

func (s *RepositorySuite) TestDeleteByUserID() {
	s.Require().NoError(s.repo.DeleteByUserID(s.ctx, seedAdminUserID))

	_, err := s.repo.GetByUserID(s.ctx, seedAdminUserID)
	s.requireNotFound(err)
}

func (s *RepositorySuite) TestWithTx() {
	var gotUserID string
	err := chi_repository.RunInTx(s.ctx, s.db, nil, func(tx chi_repository.Tx) error {
		got, err := s.repo.WithTx(tx).GetByUserID(s.ctx, seedAdminUserID)
		if err != nil {
			return err
		}
		gotUserID = got.UserID.String()
		return nil
	})
	s.Require().NoError(err)
	s.Equal(seedAdminUserID, gotUserID)
}

func (s *RepositorySuite) requireNotFound(err error) {
	s.T().Helper()
	s.Require().Error(err)
	var apiErr *chi_error.Error
	s.Require().True(errors.As(err, &apiErr))
	s.Equal(http.StatusNotFound, apiErr.StatusCode)
}
