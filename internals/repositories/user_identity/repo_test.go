//go:build integration

package user_identity_repository_test

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
	user_identity_repository "github.com/yca-software/2chi-go-api/internals/repositories/user_identity"
	"github.com/yca-software/2chi-go-api/internals/packages/testutil"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_repository "github.com/yca-software/2chi-go-repository"
	chi_types "github.com/yca-software/2chi-go-types"
)

const (
	seedUserID          = "33333333-3333-3333-3333-333333333001"
	seedIdentityID      = "44444444-4444-4444-4444-444444444001"
	seedProvider        = "google"
	seedProviderUserID  = "google-user-123"
	seedUpdatedProvider = "google-user-456"
)

var seedCreatedAtTime = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

func TestMain(m *testing.M) {
	os.Exit(testutil.IntegrationTestMain(m))
}

func TestUserIdentityRepositorySuite(t *testing.T) {
	suite.Run(t, new(UserIdentityRepositorySuite))
}

type UserIdentityRepositorySuite struct {
	suite.Suite

	db   *sqlx.DB
	repo user_identity_repository.UserIdentityRepository
	ctx  context.Context
}

func (s *UserIdentityRepositorySuite) SetupSuite() {
	testDB, err := testutil.GetIntegrationDB()
	s.Require().NoError(err)

	s.db, err = testDB.SQLx()
	s.Require().NoError(err)

	s.repo = user_identity_repository.NewUserIdentityRepository(s.db, nil)
	s.ctx = context.Background()
}

func (s *UserIdentityRepositorySuite) SetupTest() {
	_, err := s.db.ExecContext(s.ctx, `
INSERT INTO users (
	id, created_at, deleted_at, first_name, last_name, language, email, password
) VALUES
	('33333333-3333-3333-3333-333333333001', '2024-01-01T00:00:00Z', NULL, 'OAuth', 'User', 'en', 'oauth@example.com', 'hash');
INSERT INTO user_identities (id, created_at, updated_at, user_id, provider, provider_user_id) VALUES
	('44444444-4444-4444-4444-444444444001', '2024-01-01T00:00:00Z', '2024-01-01T00:00:00Z', '33333333-3333-3333-3333-333333333001', 'google', 'google-user-123')`)
	s.Require().NoError(err)
}

func (s *UserIdentityRepositorySuite) TearDownTest() {
	_, err := s.db.ExecContext(s.ctx, `TRUNCATE TABLE users CASCADE`)
	s.Require().NoError(err)
}

func (s *UserIdentityRepositorySuite) TestCreateUserIdentity() {
	identity := &models.UserIdentity{
		ModelBase: chi_types.ModelBase{
			ID:        uuid.MustParse("44444444-4444-4444-4444-444444444002"),
			CreatedAt: seedCreatedAtTime,
			UpdatedAt: seedCreatedAtTime,
		},
		UserID:         uuid.MustParse(seedUserID),
		Provider:       "github",
		ProviderUserID: "github-user-123",
	}
	s.Require().NoError(s.repo.CreateUserIdentity(s.ctx, identity))

	got, err := s.repo.GetUserIdentityByProviderAndProviderUserID(s.ctx, "github", "github-user-123")
	s.Require().NoError(err)
	s.Equal("44444444-4444-4444-4444-444444444002", got.ID.String())
}

func (s *UserIdentityRepositorySuite) TestGetUserIdentityByProviderAndProviderUserID() {
	got, err := s.repo.GetUserIdentityByProviderAndProviderUserID(s.ctx, seedProvider, seedProviderUserID)
	s.Require().NoError(err)
	s.Equal(seedIdentityID, got.ID.String())
}

func (s *UserIdentityRepositorySuite) TestGetUserIdentityByUserIDAndProvider() {
	got, err := s.repo.GetUserIdentityByUserIDAndProvider(s.ctx, seedUserID, seedProvider)
	s.Require().NoError(err)
	s.Equal(seedProviderUserID, got.ProviderUserID)
}

func (s *UserIdentityRepositorySuite) TestGetUserIdentityByProviderAndProviderUserID_NotFound() {
	_, err := s.repo.GetUserIdentityByProviderAndProviderUserID(s.ctx, seedProvider, "missing")
	s.requireNotFound(err)
}

func (s *UserIdentityRepositorySuite) TestUpdateUserIdentity() {
	identity, err := s.repo.GetUserIdentityByUserIDAndProvider(s.ctx, seedUserID, seedProvider)
	s.Require().NoError(err)

	identity.ProviderUserID = seedUpdatedProvider
	identity.UpdatedAt = seedCreatedAtTime.Add(time.Hour)
	s.Require().NoError(s.repo.UpdateUserIdentity(s.ctx, identity))

	got, err := s.repo.GetUserIdentityByUserIDAndProvider(s.ctx, seedUserID, seedProvider)
	s.Require().NoError(err)
	s.Equal(seedUpdatedProvider, got.ProviderUserID)
}

func (s *UserIdentityRepositorySuite) TestWithTx() {
	identity := &models.UserIdentity{
		ModelBase: chi_types.ModelBase{
			ID:        uuid.MustParse("44444444-4444-4444-4444-444444444003"),
			CreatedAt: seedCreatedAtTime,
			UpdatedAt: seedCreatedAtTime,
		},
		UserID:         uuid.MustParse(seedUserID),
		Provider:       "linkedin",
		ProviderUserID: "linkedin-user-123",
	}
	err := chi_repository.RunInTx(s.ctx, s.db, nil, func(tx chi_repository.Tx) error {
		return s.repo.WithTx(tx).CreateUserIdentity(s.ctx, identity)
	})
	s.Require().NoError(err)

	got, err := s.repo.GetUserIdentityByProviderAndProviderUserID(s.ctx, "linkedin", "linkedin-user-123")
	s.Require().NoError(err)
	s.Equal("44444444-4444-4444-4444-444444444003", got.ID.String())
}

func (s *UserIdentityRepositorySuite) requireNotFound(err error) {
	s.T().Helper()
	s.Require().Error(err)
	var apiErr *chi_error.Error
	s.Require().True(errors.As(err, &apiErr))
	s.Equal(http.StatusNotFound, apiErr.StatusCode)
}
