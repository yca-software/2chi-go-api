//go:build integration

package user_password_reset_token_repository_test

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
	user_password_reset_token_repository "github.com/yca-software/2chi-go-api/internals/repositories/user_password_reset_token"
	"github.com/yca-software/2chi-go-api/internals/packages/testutil"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_repository "github.com/yca-software/2chi-go-repository"
	chi_types "github.com/yca-software/2chi-go-types"
)

const (
	seedTokenUserID       = "11111111-1111-1111-1111-111111111601"
	seedPasswordResetID   = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb601"
	seedPasswordResetHash = "password-reset-hash"
)

var (
	seedCreatedAtTime      = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	seedTokenExpiresFuture = time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
)

func TestMain(m *testing.M) {
	os.Exit(testutil.IntegrationTestMain(m))
}

func TestRepositorySuite(t *testing.T) {
	suite.Run(t, new(RepositorySuite))
}

type RepositorySuite struct {
	suite.Suite

	db   *sqlx.DB
	repo user_password_reset_token_repository.Repository
	ctx  context.Context
}

func (s *RepositorySuite) SetupSuite() {
	testDB, err := testutil.GetIntegrationDB()
	s.Require().NoError(err)

	s.db, err = testDB.SQLx()
	s.Require().NoError(err)

	s.repo = user_password_reset_token_repository.NewRepository(s.db, nil)
	s.ctx = context.Background()
}

func (s *RepositorySuite) SetupTest() {
	_, err := s.db.ExecContext(s.ctx, `
INSERT INTO users (
	id, created_at, deleted_at, first_name, last_name, language, email, password
) VALUES
	('11111111-1111-1111-1111-111111111601', '2024-01-01T00:00:00Z', NULL, 'Token', 'User', 'en', 'token-user@example.com', 'hash');
INSERT INTO user_password_reset_tokens (id, user_id, created_at, expires_at, used_at, token_hash) VALUES
	('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb601', '11111111-1111-1111-1111-111111111601', '2024-01-01T00:00:00Z', '2030-01-01T00:00:00Z', NULL, 'password-reset-hash'),
	('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb602', '11111111-1111-1111-1111-111111111601', '2020-01-01T00:00:00Z', '2020-01-02T00:00:00Z', NULL, 'password-reset-hash-stale')`)
	s.Require().NoError(err)
}

func (s *RepositorySuite) TearDownTest() {
	_, err := s.db.ExecContext(s.ctx, `TRUNCATE TABLE users CASCADE`)
	s.Require().NoError(err)
}

func (s *RepositorySuite) TestCreate() {
	token := &models.UserPasswordResetToken{
		ModelBase: chi_types.ModelBase{
			ID:        uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb603"),
			CreatedAt: seedCreatedAtTime,
		},
		UserID:    uuid.MustParse(seedTokenUserID),
		ExpiresAt: seedTokenExpiresFuture,
		TokenHash: "password-reset-hash-new",
	}
	s.Require().NoError(s.repo.Create(s.ctx, token))

	got, err := s.repo.GetByHash(s.ctx, "password-reset-hash-new")
	s.Require().NoError(err)
	s.Equal("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb603", got.ID.String())
}

func (s *RepositorySuite) TestGetByHash() {
	got, err := s.repo.GetByHash(s.ctx, seedPasswordResetHash)
	s.Require().NoError(err)
	s.Equal(seedPasswordResetID, got.ID.String())
}

func (s *RepositorySuite) TestMarkAsUsed() {
	s.Require().NoError(s.repo.MarkAsUsed(s.ctx, seedPasswordResetID))
	got, err := s.repo.GetByHash(s.ctx, seedPasswordResetHash)
	s.Require().NoError(err)
	s.NotNil(got.UsedAt)
}

func (s *RepositorySuite) TestCleanupStaleUnused() {
	s.Require().NoError(s.repo.CleanupStaleUnused(s.ctx))
	_, err := s.repo.GetByHash(s.ctx, "password-reset-hash-stale")
	s.requireNotFound(err)
}

func (s *RepositorySuite) TestWithTx() {
	token := &models.UserPasswordResetToken{
		ModelBase: chi_types.ModelBase{
			ID:        uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb604"),
			CreatedAt: seedCreatedAtTime,
		},
		UserID:    uuid.MustParse(seedTokenUserID),
		ExpiresAt: seedTokenExpiresFuture,
		TokenHash: "password-reset-hash-tx",
	}
	err := chi_repository.RunInTx(s.ctx, s.db, nil, func(tx chi_repository.Tx) error {
		return s.repo.WithTx(tx).Create(s.ctx, token)
	})
	s.Require().NoError(err)

	got, err := s.repo.GetByHash(s.ctx, "password-reset-hash-tx")
	s.Require().NoError(err)
	s.Equal("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb604", got.ID.String())
}

func (s *RepositorySuite) requireNotFound(err error) {
	s.T().Helper()
	s.Require().Error(err)
	var apiErr *chi_error.Error
	s.Require().True(errors.As(err, &apiErr))
	s.Equal(http.StatusNotFound, apiErr.StatusCode)
}
