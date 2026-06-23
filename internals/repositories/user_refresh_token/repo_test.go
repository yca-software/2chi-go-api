//go:build integration

package user_refresh_token_repository_test

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
	user_refresh_token_repository "github.com/yca-software/2chi-go-api/internals/repositories/user_refresh_token"
	"github.com/yca-software/2chi-go-api/internals/packages/testutil"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_repository "github.com/yca-software/2chi-go-repository"
	chi_types "github.com/yca-software/2chi-go-types"
)

const (
	seedTokenUserID          = "11111111-1111-1111-1111-111111111601"
	seedImpersonatorUserID   = "11111111-1111-1111-1111-111111111602"
	seedRefreshActiveID      = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa601"
	seedRefreshRevokeID      = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa602"
	seedRefreshImpersonateID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa603"
	seedRefreshExcludeID     = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa605"
	seedRefreshNewID         = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa606"
	seedRefreshHashActive    = "refresh-hash-active"
	seedRefreshHashRevoke    = "refresh-hash-revoke"
	seedRefreshHashNew       = "refresh-hash-new"
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
	repo user_refresh_token_repository.Repository
	ctx  context.Context
}

func (s *RepositorySuite) SetupSuite() {
	testDB, err := testutil.GetIntegrationDB()
	s.Require().NoError(err)

	s.db, err = testDB.SQLx()
	s.Require().NoError(err)

	s.repo = user_refresh_token_repository.NewRepository(s.db, nil)
	s.ctx = context.Background()
}

func (s *RepositorySuite) SetupTest() {
	_, err := s.db.ExecContext(s.ctx, `
INSERT INTO users (
	id, created_at, deleted_at, first_name, last_name, language, email, password
) VALUES
	('11111111-1111-1111-1111-111111111601', '2024-01-01T00:00:00Z', NULL, 'Token', 'User', 'en', 'token-user@example.com', 'hash'),
	('11111111-1111-1111-1111-111111111602', '2024-01-01T00:00:00Z', NULL, 'Admin', 'User', 'en', 'admin-user@example.com', 'hash');
INSERT INTO user_refresh_tokens (
	id, user_id, created_at, expires_at, revoked_at, ip, user_agent, token_hash, impersonated_by
) VALUES
	('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa601', '11111111-1111-1111-1111-111111111601', '2024-01-01T00:00:00Z', '2030-01-01T00:00:00Z', NULL, '127.0.0.1', 'agent', 'refresh-hash-active', NULL),
	('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa602', '11111111-1111-1111-1111-111111111601', '2024-01-01T00:00:00Z', '2030-01-01T00:00:00Z', NULL, '127.0.0.1', 'agent', 'refresh-hash-revoke', NULL),
	('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa603', '11111111-1111-1111-1111-111111111601', '2024-01-01T00:00:00Z', '2030-01-01T00:00:00Z', NULL, '127.0.0.1', 'agent', 'refresh-hash-impersonate', '11111111-1111-1111-1111-111111111602'),
	('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa604', '11111111-1111-1111-1111-111111111601', '2020-01-01T00:00:00Z', '2020-01-02T00:00:00Z', NULL, '127.0.0.1', 'agent', 'refresh-hash-stale', NULL),
	('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa605', '11111111-1111-1111-1111-111111111601', '2024-01-01T00:00:00Z', '2030-01-01T00:00:00Z', NULL, '127.0.0.1', 'agent', 'refresh-hash-exclude', NULL)`)
	s.Require().NoError(err)
}

func (s *RepositorySuite) TearDownTest() {
	_, err := s.db.ExecContext(s.ctx, `TRUNCATE TABLE users CASCADE`)
	s.Require().NoError(err)
}

func (s *RepositorySuite) TestCreate() {
	token := &models.UserRefreshToken{
		ModelBase: chi_types.ModelBase{
			ID:        uuid.MustParse(seedRefreshNewID),
			CreatedAt: seedCreatedAtTime,
		},
		UserID:    uuid.MustParse(seedTokenUserID),
		ExpiresAt: seedTokenExpiresFuture,
		IP:        "127.0.0.1",
		UserAgent: "agent",
		TokenHash: seedRefreshHashNew,
	}
	s.Require().NoError(s.repo.Create(s.ctx, token))

	got, err := s.repo.GetByHash(s.ctx, seedRefreshHashNew)
	s.Require().NoError(err)
	s.Equal(seedRefreshNewID, got.ID.String())
}

func (s *RepositorySuite) TestListActiveByUserID() {
	rows, err := s.repo.ListActiveByUserID(s.ctx, seedTokenUserID)
	s.Require().NoError(err)
	s.GreaterOrEqual(len(*rows), 3)
}

func (s *RepositorySuite) TestGetActiveImpersonationByUserID() {
	got, err := s.repo.GetActiveImpersonationByUserID(s.ctx, seedTokenUserID)
	s.Require().NoError(err)
	s.Equal(seedRefreshImpersonateID, got.ID.String())
}

func (s *RepositorySuite) TestRevokeByID() {
	s.Require().NoError(s.repo.RevokeByID(s.ctx, seedTokenUserID, seedRefreshRevokeID))
	got, err := s.repo.GetByHash(s.ctx, seedRefreshHashRevoke)
	s.Require().NoError(err)
	s.NotNil(got.RevokedAt)
}

func (s *RepositorySuite) TestRevokeByHash() {
	s.Require().NoError(s.repo.RevokeByHash(s.ctx, seedRefreshHashActive))
	got, err := s.repo.GetByHash(s.ctx, seedRefreshHashActive)
	s.Require().NoError(err)
	s.NotNil(got.RevokedAt)
}

func (s *RepositorySuite) TestRevokeAllByUserID() {
	exclude := seedRefreshExcludeID
	s.Require().NoError(s.repo.RevokeAllByUserID(s.ctx, seedTokenUserID, &exclude))
	got, err := s.repo.GetByHash(s.ctx, "refresh-hash-exclude")
	s.Require().NoError(err)
	s.Nil(got.RevokedAt)
}

func (s *RepositorySuite) TestCleanupStaleUnused() {
	s.Require().NoError(s.repo.CleanupStaleUnused(s.ctx))
	_, err := s.repo.GetByHash(s.ctx, "refresh-hash-stale")
	s.requireNotFound(err)
}

func (s *RepositorySuite) TestWithTx() {
	token := &models.UserRefreshToken{
		ModelBase: chi_types.ModelBase{
			ID:        uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa607"),
			CreatedAt: seedCreatedAtTime,
		},
		UserID:    uuid.MustParse(seedTokenUserID),
		ExpiresAt: seedTokenExpiresFuture,
		IP:        "127.0.0.1",
		UserAgent: "agent",
		TokenHash: "refresh-hash-tx",
	}
	err := chi_repository.RunInTx(s.ctx, s.db, nil, func(tx chi_repository.Tx) error {
		return s.repo.WithTx(tx).Create(s.ctx, token)
	})
	s.Require().NoError(err)

	got, err := s.repo.GetByHash(s.ctx, "refresh-hash-tx")
	s.Require().NoError(err)
	s.Equal("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa607", got.ID.String())
}

func (s *RepositorySuite) requireNotFound(err error) {
	s.T().Helper()
	s.Require().Error(err)
	var apiErr *chi_error.Error
	s.Require().True(errors.As(err, &apiErr))
	s.Equal(http.StatusNotFound, apiErr.StatusCode)
}
