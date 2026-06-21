package user_email_verification_token_repository_test

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

	"github.com/yca-software/2chi-go-api/internals/models"
	user_email_verification_token_repository "github.com/yca-software/2chi-go-api/internals/repositories/user_email_verification_token"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_repository "github.com/yca-software/2chi-go-repository"
	chi_test "github.com/yca-software/2chi-go-test"
	chi_types "github.com/yca-software/2chi-go-types"
)

const (
	seedTokenUserID      = "11111111-1111-1111-1111-111111111601"
	seedEmailVerifyID    = "cccccccc-cccc-cccc-cccc-cccccccccc01"
	seedEmailVerifyHash  = "email-verify-hash"
	seedEmailVerifyStale = "cccccccc-cccc-cccc-cccc-cccccccccc02"
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

func TestUserEmailVerificationTokenRepositorySuite(t *testing.T) {
	suite.Run(t, new(UserEmailVerificationTokenRepositorySuite))
}

type UserEmailVerificationTokenRepositorySuite struct {
	suite.Suite

	db   *sqlx.DB
	repo user_email_verification_token_repository.UserEmailVerificationTokenRepository
	ctx  context.Context
}

func (s *UserEmailVerificationTokenRepositorySuite) SetupSuite() {
	testDB, err := chi_test.Get(moduleMigrationsDir())
	s.Require().NoError(err)

	s.db, err = testDB.SQLx()
	s.Require().NoError(err)

	s.repo = user_email_verification_token_repository.NewUserEmailVerificationTokenRepository(s.db, nil)
	s.ctx = context.Background()
}

func (s *UserEmailVerificationTokenRepositorySuite) SetupTest() {
	_, err := s.db.ExecContext(s.ctx, `
INSERT INTO users (
	id, created_at, deleted_at, first_name, last_name, language, email, password
) VALUES
	('11111111-1111-1111-1111-111111111601', '2024-01-01T00:00:00Z', NULL, 'Token', 'User', 'en', 'token-user@example.com', 'hash');
INSERT INTO user_email_verification_tokens (id, user_id, created_at, expires_at, used_at, token_hash) VALUES
	('cccccccc-cccc-cccc-cccc-cccccccccc01', '11111111-1111-1111-1111-111111111601', '2024-01-01T00:00:00Z', '2030-01-01T00:00:00Z', NULL, 'email-verify-hash'),
	('cccccccc-cccc-cccc-cccc-cccccccccc02', '11111111-1111-1111-1111-111111111601', '2020-01-01T00:00:00Z', '2020-01-02T00:00:00Z', '2020-01-02T00:00:00Z', 'email-verify-hash-stale')`)
	s.Require().NoError(err)
}

func (s *UserEmailVerificationTokenRepositorySuite) TearDownTest() {
	_, err := s.db.ExecContext(s.ctx, `TRUNCATE TABLE users CASCADE`)
	s.Require().NoError(err)
}

func (s *UserEmailVerificationTokenRepositorySuite) TestCreateEmailVerificationToken() {
	token := &models.UserEmailVerificationToken{
		ModelBase: chi_types.ModelBase{
			ID: uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccc03"),
		},
		UserID:    uuid.MustParse(seedTokenUserID),
		CreatedAt: seedCreatedAtTime,
		ExpiresAt: seedTokenExpiresFuture,
		TokenHash: "email-verify-hash-new",
	}
	s.Require().NoError(s.repo.CreateEmailVerificationToken(s.ctx, token))

	got, err := s.repo.GetEmailVerificationTokenByHash(s.ctx, "email-verify-hash-new")
	s.Require().NoError(err)
	s.Equal("cccccccc-cccc-cccc-cccc-cccccccccc03", got.ID.String())
}

func (s *UserEmailVerificationTokenRepositorySuite) TestGetEmailVerificationTokenByHash() {
	got, err := s.repo.GetEmailVerificationTokenByHash(s.ctx, seedEmailVerifyHash)
	s.Require().NoError(err)
	s.Equal(seedEmailVerifyID, got.ID.String())
}

func (s *UserEmailVerificationTokenRepositorySuite) TestMarkEmailVerificationTokenAsUsed() {
	s.Require().NoError(s.repo.MarkEmailVerificationTokenAsUsed(s.ctx, seedEmailVerifyID))
	got, err := s.repo.GetEmailVerificationTokenByHash(s.ctx, seedEmailVerifyHash)
	s.Require().NoError(err)
	s.NotNil(got.UsedAt)
}

func (s *UserEmailVerificationTokenRepositorySuite) TestCleanupStaleUnusedEmailVerificationTokens() {
	s.Require().NoError(s.repo.CleanupStaleUnusedEmailVerificationTokens(s.ctx))
	_, err := s.repo.GetEmailVerificationTokenByHash(s.ctx, "email-verify-hash-stale")
	s.requireNotFound(err)
}

func (s *UserEmailVerificationTokenRepositorySuite) TestWithTx() {
	token := &models.UserEmailVerificationToken{
		ModelBase: chi_types.ModelBase{
			ID: uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccc04"),
		},
		UserID:    uuid.MustParse(seedTokenUserID),
		CreatedAt: seedCreatedAtTime,
		ExpiresAt: seedTokenExpiresFuture,
		TokenHash: "email-verify-hash-tx",
	}
	err := chi_repository.RunInTx(s.ctx, s.db, nil, func(tx chi_repository.Tx) error {
		return s.repo.WithTx(tx).CreateEmailVerificationToken(s.ctx, token)
	})
	s.Require().NoError(err)

	got, err := s.repo.GetEmailVerificationTokenByHash(s.ctx, "email-verify-hash-tx")
	s.Require().NoError(err)
	s.Equal("cccccccc-cccc-cccc-cccc-cccccccccc04", got.ID.String())
}

func (s *UserEmailVerificationTokenRepositorySuite) requireNotFound(err error) {
	s.T().Helper()
	s.Require().Error(err)
	var apiErr *chi_error.Error
	s.Require().True(errors.As(err, &apiErr))
	s.Equal(http.StatusNotFound, apiErr.StatusCode)
}
