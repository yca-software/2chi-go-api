//go:build integration

package user_repository_test

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
	user_repository "github.com/yca-software/2chi-go-api/internals/repositories/user"
	"github.com/yca-software/2chi-go-api/internals/packages/testutil"
	chi_archive "github.com/yca-software/2chi-go-archive"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_repository "github.com/yca-software/2chi-go-repository"
	chi_types "github.com/yca-software/2chi-go-types"
)

const (
	seedActiveUserID     = "11111111-1111-1111-1111-111111111001"
	seedUpdateID     = "11111111-1111-1111-1111-111111111002"
	seedArchiveTargetID  = "11111111-1111-1111-1111-111111111003"
	seedRestoreID    = "11111111-1111-1111-1111-111111111004"
	seedArchivedUserID   = "11111111-1111-1111-1111-111111111006"
	seedStaleArchivedID  = "11111111-1111-1111-1111-111111111007"
	seedSearchActiveID   = "11111111-1111-1111-1111-111111111008"
	seedSearchArchivedID = "11111111-1111-1111-1111-111111111009"

	seedActiveEmail = "active@example.com"

	seedNewUserID    = "11111111-1111-1111-1111-11111111100b"
	seedNewUserEmail = "new-user@example.com"
	seedTxUserID     = "11111111-1111-1111-1111-11111111100c"
	seedTxUserEmail  = "tx-user@example.com"
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
	repo user_repository.Repository
	ctx  context.Context
}

func (s *RepositorySuite) SetupSuite() {
	testDB, err := testutil.GetIntegrationDB()
	s.Require().NoError(err)

	s.db, err = testDB.SQLx()
	s.Require().NoError(err)

	s.repo = user_repository.NewRepository(s.db, nil)
	s.ctx = context.Background()
}

func (s *RepositorySuite) SetupTest() {
	_, err := s.db.ExecContext(s.ctx, `
INSERT INTO users (
	id, created_at, deleted_at, first_name, last_name, language, email, password
) VALUES
	('11111111-1111-1111-1111-111111111001', '2024-01-01T00:00:00Z', NULL, 'Active', 'User', 'en', 'active@example.com', 'hash'),
	('11111111-1111-1111-1111-111111111002', '2024-01-01T00:00:00Z', NULL, 'Update', 'User', 'en', 'update@example.com', 'hash'),
	('11111111-1111-1111-1111-111111111003', '2024-01-01T00:00:00Z', NULL, 'Archive', 'Target', 'en', 'archive-target@example.com', 'hash'),
	('11111111-1111-1111-1111-111111111004', '2024-01-01T00:00:00Z', '2026-06-06T00:00:00Z', 'Restore', 'User', 'en', 'restore@example.com', 'hash'),
	('11111111-1111-1111-1111-111111111006', '2024-01-01T00:00:00Z', '2026-06-06T00:00:00Z', 'Archived', 'User', 'en', 'archived@example.com', 'hash'),
	('11111111-1111-1111-1111-111111111007', '2024-01-01T00:00:00Z', '2020-01-01T00:00:00Z', 'Stale', 'User', 'en', 'stale@example.com', 'hash'),
	('11111111-1111-1111-1111-111111111008', '2024-01-01T00:00:00Z', NULL, 'FindMeActive', 'User', 'en', 'find-active@example.com', 'hash'),
	('11111111-1111-1111-1111-111111111009', '2024-01-01T00:00:00Z', '2026-06-06T00:00:00Z', 'FindMeArchived', 'User', 'en', 'find-archived@example.com', 'hash')`)
	s.Require().NoError(err)
}

func (s *RepositorySuite) TearDownTest() {
	_, err := s.db.ExecContext(s.ctx, `TRUNCATE TABLE users CASCADE`)
	s.Require().NoError(err)
}

func (s *RepositorySuite) TestCreate() {
	user := &models.User{
		ModelBaseWithArchive: chi_types.ModelBaseWithArchive{
			ModelBase: chi_types.ModelBase{
				ID:        uuid.MustParse(seedNewUserID),
				CreatedAt: seedCreatedAtTime,
			},
		},
		FirstName: "New",
		LastName:  "User",
		Language:  "en",
		Email:     seedNewUserEmail,
		Password:  "hash",
	}
	s.Require().NoError(s.repo.Create(s.ctx, user))

	got, err := s.repo.GetByID(s.ctx, seedNewUserID)
	s.Require().NoError(err)
	s.Equal(seedNewUserEmail, got.Email)
	s.False(got.CreatedAt.IsZero())
	s.False(got.UpdatedAt.IsZero())
}

func (s *RepositorySuite) TestUpdate() {
	user, err := s.repo.GetByID(s.ctx, seedUpdateID)
	s.Require().NoError(err)
	originalUpdatedAt := user.UpdatedAt

	user.FirstName = "UpdatedFirst"
	user.LastName = "UpdatedLast"
	user.Language = "de"
	s.Require().NoError(s.repo.Update(s.ctx, user))

	got, err := s.repo.GetByID(s.ctx, seedUpdateID)
	s.Require().NoError(err)
	s.Equal("UpdatedFirst", got.FirstName)
	s.Equal("UpdatedLast", got.LastName)
	s.Equal("de", got.Language)
	s.True(got.UpdatedAt.After(originalUpdatedAt))
}

func (s *RepositorySuite) TestGetByID() {
	got, err := s.repo.GetByID(s.ctx, seedActiveUserID)
	s.Require().NoError(err)
	s.Equal(seedActiveUserID, got.ID.String())
	s.Equal(seedActiveEmail, got.Email)
}

func (s *RepositorySuite) TestGetByID_NotFoundWhenArchived() {
	_, err := s.repo.GetByID(s.ctx, seedArchivedUserID)
	s.requireNotFound(err)
}

func (s *RepositorySuite) TestGetByIDIncludeArchived() {
	got, err := s.repo.GetByIDIncludeArchived(s.ctx, seedArchivedUserID)
	s.Require().NoError(err)
	s.NotNil(got.DeletedAt)
}

func (s *RepositorySuite) TestGetByEmail() {
	got, err := s.repo.GetByEmail(s.ctx, seedActiveEmail)
	s.Require().NoError(err)
	s.Equal(seedActiveUserID, got.ID.String())
}

func (s *RepositorySuite) TestArchive() {
	user, err := s.repo.GetByID(s.ctx, seedArchiveTargetID)
	s.Require().NoError(err)
	s.Require().NoError(s.repo.Archive(s.ctx, user))

	got, err := s.repo.GetByIDIncludeArchived(s.ctx, seedArchiveTargetID)
	s.Require().NoError(err)
	s.NotNil(got.DeletedAt)
}

func (s *RepositorySuite) TestRestore() {
	s.Require().NoError(s.repo.Restore(s.ctx, seedRestoreID))

	got, err := s.repo.GetByID(s.ctx, seedRestoreID)
	s.Require().NoError(err)
	s.Nil(got.DeletedAt)
}

func (s *RepositorySuite) TestSearch_ActiveAndArchived() {
	activeRows, err := s.repo.Search(s.ctx, "FindMe", chi_archive.ArchiveFilterActive, 10, 0)
	s.Require().NoError(err)
	s.Require().Len(*activeRows, 1)
	s.Equal(seedSearchActiveID, (*activeRows)[0].ID.String())

	archivedRows, err := s.repo.Search(s.ctx, "FindMe", chi_archive.ArchiveFilterArchived, 10, 0)
	s.Require().NoError(err)
	s.Require().Len(*archivedRows, 1)
	s.Equal(seedSearchArchivedID, (*archivedRows)[0].ID.String())
}

func (s *RepositorySuite) TestSearch_InvalidFilter() {
	_, err := s.repo.Search(s.ctx, "", chi_archive.ArchiveFilter("nope"), 10, 0)
	s.Require().Error(err)
}

func (s *RepositorySuite) TestCleanupArchived() {
	s.Require().NoError(s.repo.CleanupArchived(s.ctx))

	_, err := s.repo.GetByIDIncludeArchived(s.ctx, seedStaleArchivedID)
	s.requireNotFound(err)

	got, err := s.repo.GetByIDIncludeArchived(s.ctx, seedArchivedUserID)
	s.Require().NoError(err)
	s.NotNil(got.DeletedAt)
}

func (s *RepositorySuite) TestWithTx() {
	user := &models.User{
		ModelBaseWithArchive: chi_types.ModelBaseWithArchive{
			ModelBase: chi_types.ModelBase{
				ID:        uuid.MustParse(seedTxUserID),
				CreatedAt: seedCreatedAtTime,
			},
		},
		FirstName: "Tx",
		LastName:  "User",
		Language:  "en",
		Email:     seedTxUserEmail,
		Password:  "hash",
	}
	err := chi_repository.RunInTx(s.ctx, s.db, nil, func(tx chi_repository.Tx) error {
		return s.repo.WithTx(tx).Create(s.ctx, user)
	})
	s.Require().NoError(err)

	got, err := s.repo.GetByID(s.ctx, seedTxUserID)
	s.Require().NoError(err)
	s.Equal(seedTxUserEmail, got.Email)
}

func (s *RepositorySuite) requireNotFound(err error) {
	s.T().Helper()
	s.Require().Error(err)
	var apiErr *chi_error.Error
	s.Require().True(errors.As(err, &apiErr))
	s.Equal(http.StatusNotFound, apiErr.StatusCode)
}
