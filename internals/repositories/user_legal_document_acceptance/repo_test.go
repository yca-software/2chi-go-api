//go:build integration

package user_legal_document_acceptance_repository_test

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
	user_legal_document_acceptance_repository "github.com/yca-software/2chi-go-api/internals/repositories/user_legal_document_acceptance"
	"github.com/yca-software/2chi-go-api/internals/packages/testutil"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_repository "github.com/yca-software/2chi-go-repository"
	chi_types "github.com/yca-software/2chi-go-types"
)

const (
	seedUserID              = "55555555-5555-5555-5555-555555555001"
	seedTermsV1ID           = "66666666-6666-6666-6666-666666666001"
	seedTermsV2ID           = "66666666-6666-6666-6666-666666666002"
	seedPrivacyID           = "66666666-6666-6666-6666-666666666003"
	seedDocumentTypeTerms   = "terms_of_service"
	seedDocumentTypePrivacy = "privacy_policy"
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
	repo user_legal_document_acceptance_repository.Repository
	ctx  context.Context
}

func (s *RepositorySuite) SetupSuite() {
	testDB, err := testutil.GetIntegrationDB()
	s.Require().NoError(err)

	s.db, err = testDB.SQLx()
	s.Require().NoError(err)

	s.repo = user_legal_document_acceptance_repository.NewRepository(s.db, nil)
	s.ctx = context.Background()
}

func (s *RepositorySuite) SetupTest() {
	_, err := s.db.ExecContext(s.ctx, `
INSERT INTO users (
	id, created_at, deleted_at, first_name, last_name, language, email, password
) VALUES
	('55555555-5555-5555-5555-555555555001', '2024-01-01T00:00:00Z', NULL, 'Legal', 'User', 'en', 'legal@example.com', 'hash');
INSERT INTO user_legal_document_acceptances (id, created_at, updated_at, user_id, document_type, document_version) VALUES
	('66666666-6666-6666-6666-666666666001', '2024-01-01T00:00:00Z', '2024-01-01T00:00:00Z', '55555555-5555-5555-5555-555555555001', 'terms_of_service', '1.0.0'),
	('66666666-6666-6666-6666-666666666002', '2024-06-01T00:00:00Z', '2024-06-01T00:00:00Z', '55555555-5555-5555-5555-555555555001', 'terms_of_service', '2.0.0'),
	('66666666-6666-6666-6666-666666666003', '2024-03-01T00:00:00Z', '2024-03-01T00:00:00Z', '55555555-5555-5555-5555-555555555001', 'privacy_policy', '1.0.0')`)
	s.Require().NoError(err)
}

func (s *RepositorySuite) TearDownTest() {
	_, err := s.db.ExecContext(s.ctx, `TRUNCATE TABLE users CASCADE`)
	s.Require().NoError(err)
}

func (s *RepositorySuite) TestCreate() {
	acceptance := &models.UserLegalDocumentAcceptance{
		ModelBase: chi_types.ModelBase{
			ID:        uuid.MustParse("66666666-6666-6666-6666-666666666004"),
			CreatedAt: time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC),
		},
		UserID:          uuid.MustParse(seedUserID),
		DocumentType:    "privacy_policy",
		DocumentVersion: "2.0.0",
	}
	s.Require().NoError(s.repo.Create(s.ctx, acceptance))

	got, err := s.repo.GetLatestByUserIDAndDocumentType(s.ctx, seedUserID, "privacy_policy")
	s.Require().NoError(err)
	s.Equal("66666666-6666-6666-6666-666666666004", got.ID.String())
}

func (s *RepositorySuite) TestListByUserID() {
	rows, err := s.repo.ListByUserID(s.ctx, seedUserID)
	s.Require().NoError(err)
	s.Len(*rows, 3)
	s.Equal(seedPrivacyID, (*rows)[0].ID.String())
	s.Equal(seedTermsV2ID, (*rows)[1].ID.String())
	s.Equal(seedTermsV1ID, (*rows)[2].ID.String())
}

func (s *RepositorySuite) TestGetLatestByUserIDAndDocumentType() {
	got, err := s.repo.GetLatestByUserIDAndDocumentType(s.ctx, seedUserID, seedDocumentTypeTerms)
	s.Require().NoError(err)
	s.Equal(seedTermsV2ID, got.ID.String())
	s.Equal("2.0.0", got.DocumentVersion)
}

func (s *RepositorySuite) TestGetLatestByUserIDAndDocumentType_NotFound() {
	_, err := s.repo.GetLatestByUserIDAndDocumentType(s.ctx, seedUserID, "cookie_policy")
	s.requireNotFound(err)
}

func (s *RepositorySuite) TestListLatestByUserID() {
	rows, err := s.repo.ListLatestByUserID(s.ctx, seedUserID)
	s.Require().NoError(err)
	s.Len(*rows, 2)

	byType := make(map[string]models.UserLegalDocumentAcceptance, len(*rows))
	for _, row := range *rows {
		byType[row.DocumentType] = row
	}
	s.Equal(seedPrivacyID, byType[seedDocumentTypePrivacy].ID.String())
	s.Equal(seedTermsV2ID, byType[seedDocumentTypeTerms].ID.String())
}

func (s *RepositorySuite) TestWithTx() {
	acceptance := &models.UserLegalDocumentAcceptance{
		ModelBase: chi_types.ModelBase{
			ID:        uuid.MustParse("66666666-6666-6666-6666-666666666005"),
			CreatedAt: time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC),
		},
		UserID:          uuid.MustParse(seedUserID),
		DocumentType:    seedDocumentTypeTerms,
		DocumentVersion: "3.0.0",
	}
	err := chi_repository.RunInTx(s.ctx, s.db, nil, func(tx chi_repository.Tx) error {
		return s.repo.WithTx(tx).Create(s.ctx, acceptance)
	})
	s.Require().NoError(err)

	got, err := s.repo.GetLatestByUserIDAndDocumentType(s.ctx, seedUserID, seedDocumentTypeTerms)
	s.Require().NoError(err)
	s.Equal("66666666-6666-6666-6666-666666666005", got.ID.String())
}

func (s *RepositorySuite) requireNotFound(err error) {
	s.T().Helper()
	s.Require().Error(err)
	var apiErr *chi_error.Error
	s.Require().True(errors.As(err, &apiErr))
	s.Equal(http.StatusNotFound, apiErr.StatusCode)
}
