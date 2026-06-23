//go:build integration

package api_key_repository_test

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
	api_key_repository "github.com/yca-software/2chi-go-api/internals/repositories/api_key"
	"github.com/yca-software/2chi-go-api/internals/packages/testutil"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_repository "github.com/yca-software/2chi-go-repository"
	chi_types "github.com/yca-software/2chi-go-types"
)

const (
	seedAPIKeyOrgID    = "22222222-2222-2222-2222-222222222401"
	seedAPIKeyActiveID = "88888888-8888-8888-8888-888888888401"
	seedAPIKeyHash     = "api-key-hash-active"
	seedAPIKeyUpdateID = "88888888-8888-8888-8888-888888888402"
	seedAPIKeyDeleteID = "88888888-8888-8888-8888-888888888403"
	seedAPIKeyNewID    = "88888888-8888-8888-8888-888888888404"
	seedAPIKeyNewHash  = "api-key-hash-new"
	seedAPIKeyTxID     = "88888888-8888-8888-8888-888888888405"
	seedAPIKeyTxHash   = "api-key-hash-tx"
)

var (
	seedCreatedAtTime   = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	seedAPIKeyExpiresAt = time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
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
	repo api_key_repository.Repository
	ctx  context.Context
}

func (s *RepositorySuite) SetupSuite() {
	testDB, err := testutil.GetIntegrationDB()
	s.Require().NoError(err)

	s.db, err = testDB.SQLx()
	s.Require().NoError(err)

	s.repo = api_key_repository.NewRepository(s.db, nil)
	s.ctx = context.Background()
}

func (s *RepositorySuite) SetupTest() {
	_, err := s.db.ExecContext(s.ctx, `
INSERT INTO organizations (id, created_at, deleted_at, name, address, city, zip, country, place_id, geo, timezone) VALUES (
	'22222222-2222-2222-2222-222222222401', '2024-01-01T00:00:00Z', NULL, 'API Org', '1 Main St', 'Oslo', '0001', 'NO', 'place_seed_api', ST_SetSRID(ST_MakePoint(10.7, 59.9), 4326), 'Europe/Oslo'
);
INSERT INTO api_keys (
	id, created_at, expires_at, name, key_prefix, key_hash, organization_id, permissions
) VALUES
	('88888888-8888-8888-8888-888888888401', '2024-01-01T00:00:00Z', '2030-01-01T00:00:00Z', 'Active Key', 'ak_act', 'api-key-hash-active', '22222222-2222-2222-2222-222222222401', '["org:read"]'::jsonb),
	('88888888-8888-8888-8888-888888888402', '2024-01-01T00:00:00Z', '2030-01-01T00:00:00Z', 'Update Key', 'ak_upd', 'api-key-hash-update', '22222222-2222-2222-2222-222222222401', '["org:read"]'::jsonb),
	('88888888-8888-8888-8888-888888888403', '2024-01-01T00:00:00Z', '2030-01-01T00:00:00Z', 'Delete Key', 'ak_del', 'api-key-hash-delete', '22222222-2222-2222-2222-222222222401', '["org:read"]'::jsonb)`)
	s.Require().NoError(err)
}

func (s *RepositorySuite) TearDownTest() {
	_, err := s.db.ExecContext(s.ctx, `TRUNCATE TABLE api_keys, organizations CASCADE`)
	s.Require().NoError(err)
}

func (s *RepositorySuite) TestCreate() {
	key := s.newAPIKey(seedAPIKeyNewID, seedAPIKeyNewHash, "New Key", "ak_new")
	s.Require().NoError(s.repo.Create(s.ctx, key))

	got, err := s.repo.GetByID(s.ctx, seedAPIKeyOrgID, seedAPIKeyNewID)
	s.Require().NoError(err)
	s.Equal("New Key", got.Name)
}

func (s *RepositorySuite) TestUpdate() {
	key, err := s.repo.GetByID(s.ctx, seedAPIKeyOrgID, seedAPIKeyUpdateID)
	s.Require().NoError(err)
	key.Name = "Updated Key"
	key.Permissions = models.RolePermissions{"org:write"}
	s.Require().NoError(s.repo.Update(s.ctx, key))

	got, err := s.repo.GetByID(s.ctx, seedAPIKeyOrgID, seedAPIKeyUpdateID)
	s.Require().NoError(err)
	s.Equal("Updated Key", got.Name)
}

func (s *RepositorySuite) TestDelete() {
	s.Require().NoError(s.repo.Delete(s.ctx, seedAPIKeyOrgID, seedAPIKeyDeleteID))
	_, err := s.repo.GetByID(s.ctx, seedAPIKeyOrgID, seedAPIKeyDeleteID)
	s.requireNotFound(err)
}

func (s *RepositorySuite) TestGetByHash() {
	got, err := s.repo.GetByHash(s.ctx, seedAPIKeyHash)
	s.Require().NoError(err)
	s.Equal(seedAPIKeyActiveID, got.ID.String())
}

func (s *RepositorySuite) TestListByOrganizationID() {
	rows, err := s.repo.ListByOrganizationID(s.ctx, seedAPIKeyOrgID)
	s.Require().NoError(err)
	s.GreaterOrEqual(len(*rows), 3)
}

func (s *RepositorySuite) TestWithTx() {
	key := s.newAPIKey(seedAPIKeyTxID, seedAPIKeyTxHash, "Tx Key", "ak_tx")
	err := chi_repository.RunInTx(s.ctx, s.db, nil, func(tx chi_repository.Tx) error {
		return s.repo.WithTx(tx).Create(s.ctx, key)
	})
	s.Require().NoError(err)

	got, err := s.repo.GetByHash(s.ctx, seedAPIKeyTxHash)
	s.Require().NoError(err)
	s.Equal("Tx Key", got.Name)
}

func (s *RepositorySuite) newAPIKey(id, hash, name, prefix string) *models.APIKey {
	return &models.APIKey{
		ModelBase: chi_types.ModelBase{
			ID:        uuid.MustParse(id),
			CreatedAt: seedCreatedAtTime,
		},
		ExpiresAt:      seedAPIKeyExpiresAt,
		Name:           name,
		KeyPrefix:      prefix,
		KeyHash:        hash,
		OrganizationID: uuid.MustParse(seedAPIKeyOrgID),
		Permissions:    models.RolePermissions{"org:read"},
	}
}

func (s *RepositorySuite) requireNotFound(err error) {
	s.T().Helper()
	s.Require().Error(err)
	var apiErr *chi_error.Error
	s.Require().True(errors.As(err, &apiErr))
	s.Equal(http.StatusNotFound, apiErr.StatusCode)
}
