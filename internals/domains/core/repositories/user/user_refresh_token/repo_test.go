package user_refresh_token_repository_test

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
	user_refresh_token_repository "github.com/yca-software/2chi-go-api/internals/domains/core/repositories/user/user_refresh_token"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_repository "github.com/yca-software/2chi-go-repository"
	chi_test "github.com/yca-software/2chi-go-test"
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

func TestUserRefreshTokenRepositorySuite(t *testing.T) {
	suite.Run(t, new(UserRefreshTokenRepositorySuite))
}

type UserRefreshTokenRepositorySuite struct {
	suite.Suite

	db   *sqlx.DB
	repo user_refresh_token_repository.UserRefreshTokenRepository
	ctx  context.Context
}

func (s *UserRefreshTokenRepositorySuite) SetupSuite() {
	testDB, err := chi_test.Get(moduleMigrationsDir())
	s.Require().NoError(err)

	s.db, err = testDB.SQLx()
	s.Require().NoError(err)

	s.repo = user_refresh_token_repository.NewUserRefreshTokenRepository(s.db, nil)
	s.ctx = context.Background()
}

func (s *UserRefreshTokenRepositorySuite) SetupTest() {
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

func (s *UserRefreshTokenRepositorySuite) TearDownTest() {
	_, err := s.db.ExecContext(s.ctx, `TRUNCATE TABLE users CASCADE`)
	s.Require().NoError(err)
}

func (s *UserRefreshTokenRepositorySuite) TestCreateRefreshToken() {
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
	s.Require().NoError(s.repo.CreateRefreshToken(s.ctx, token))

	got, err := s.repo.GetRefreshTokenByHash(s.ctx, seedRefreshHashNew)
	s.Require().NoError(err)
	s.Equal(seedRefreshNewID, got.ID.String())
}

func (s *UserRefreshTokenRepositorySuite) TestGetActiveRefreshTokensByUserID() {
	rows, err := s.repo.GetActiveRefreshTokensByUserID(s.ctx, seedTokenUserID)
	s.Require().NoError(err)
	s.GreaterOrEqual(len(*rows), 3)
}

func (s *UserRefreshTokenRepositorySuite) TestGetActiveImpersonationRefreshTokenByUserID() {
	got, err := s.repo.GetActiveImpersonationRefreshTokenByUserID(s.ctx, seedTokenUserID)
	s.Require().NoError(err)
	s.Equal(seedRefreshImpersonateID, got.ID.String())
}

func (s *UserRefreshTokenRepositorySuite) TestRevokeRefreshTokenByID() {
	s.Require().NoError(s.repo.RevokeRefreshTokenByID(s.ctx, seedTokenUserID, seedRefreshRevokeID))
	got, err := s.repo.GetRefreshTokenByHash(s.ctx, seedRefreshHashRevoke)
	s.Require().NoError(err)
	s.NotNil(got.RevokedAt)
}

func (s *UserRefreshTokenRepositorySuite) TestRevokeRefreshTokenByHash() {
	s.Require().NoError(s.repo.RevokeRefreshTokenByHash(s.ctx, seedRefreshHashActive))
	got, err := s.repo.GetRefreshTokenByHash(s.ctx, seedRefreshHashActive)
	s.Require().NoError(err)
	s.NotNil(got.RevokedAt)
}

func (s *UserRefreshTokenRepositorySuite) TestRevokeAllRefreshTokensByUserID() {
	exclude := seedRefreshExcludeID
	s.Require().NoError(s.repo.RevokeAllRefreshTokensByUserID(s.ctx, seedTokenUserID, &exclude))
	got, err := s.repo.GetRefreshTokenByHash(s.ctx, "refresh-hash-exclude")
	s.Require().NoError(err)
	s.Nil(got.RevokedAt)
}

func (s *UserRefreshTokenRepositorySuite) TestCleanupStaleUnusedRefreshTokens() {
	s.Require().NoError(s.repo.CleanupStaleUnusedRefreshTokens(s.ctx))
	_, err := s.repo.GetRefreshTokenByHash(s.ctx, "refresh-hash-stale")
	s.requireNotFound(err)
}

func (s *UserRefreshTokenRepositorySuite) TestWithTx() {
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
		return s.repo.WithTx(tx).CreateRefreshToken(s.ctx, token)
	})
	s.Require().NoError(err)

	got, err := s.repo.GetRefreshTokenByHash(s.ctx, "refresh-hash-tx")
	s.Require().NoError(err)
	s.Equal("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa607", got.ID.String())
}

func (s *UserRefreshTokenRepositorySuite) requireNotFound(err error) {
	s.T().Helper()
	s.Require().Error(err)
	var apiErr *chi_error.Error
	s.Require().True(errors.As(err, &apiErr))
	s.Equal(http.StatusNotFound, apiErr.StatusCode)
}
