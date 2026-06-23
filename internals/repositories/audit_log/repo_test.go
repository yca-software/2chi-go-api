//go:build integration

package audit_log_repository_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/suite"

	"github.com/yca-software/2chi-go-api/internals/models"
	audit_log_repository "github.com/yca-software/2chi-go-api/internals/repositories/audit_log"
	"github.com/yca-software/2chi-go-api/internals/packages/testutil"
	chi_repository "github.com/yca-software/2chi-go-repository"
)

const (
	seedAuditOrgID    = "22222222-2222-2222-2222-222222222501"
	seedAuditActorID  = "11111111-1111-1111-1111-111111111501"
	seedAuditLogID    = "99999999-9999-9999-9999-999999999501"
	seedAuditLogNewID = "99999999-9999-9999-9999-999999999503"
	seedAuditTxID     = "99999999-9999-9999-9999-999999999504"
)

var (
	seedAuditLogTime     = time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	seedAuditFilterStart = time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC)
	seedAuditFilterEnd   = time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC)
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
	repo audit_log_repository.Repository
	ctx  context.Context
}

func (s *RepositorySuite) SetupSuite() {
	testDB, err := testutil.GetIntegrationDB()
	s.Require().NoError(err)

	s.db, err = testDB.SQLx()
	s.Require().NoError(err)

	s.repo = audit_log_repository.NewRepository(s.db, nil)
	s.ctx = context.Background()
}

func (s *RepositorySuite) SetupTest() {
	_, err := s.db.ExecContext(s.ctx, `
INSERT INTO users (
	id, created_at, deleted_at, first_name, last_name, language, email, password
) VALUES (
	'11111111-1111-1111-1111-111111111501', '2024-01-01T00:00:00Z', NULL, 'Audit', 'Actor', 'en',
	'audit-actor@example.com', 'hash'
);
INSERT INTO organizations (id, created_at, deleted_at, name, address, city, zip, country, place_id, geo, timezone) VALUES (
	'22222222-2222-2222-2222-222222222501', '2024-01-01T00:00:00Z', NULL, 'Audit Org', '1 Main St', 'Oslo', '0001', 'NO', 'place_seed_audit', ST_SetSRID(ST_MakePoint(10.7, 59.9), 4326), 'Europe/Oslo'
);
INSERT INTO audit_logs (
	id, created_at, organization_id, actor_id, actor_info, impersonated_by_id, impersonated_by_email,
	action, resource_type, resource_id, resource_name, data
) VALUES
	('99999999-9999-9999-9999-999999999501', '2024-06-01T12:00:00Z', '22222222-2222-2222-2222-222222222501',
		'11111111-1111-1111-1111-111111111501', 'audit-actor@example.com', NULL, '', 'audit_log_repository.updated', 'user',
		'11111111-1111-1111-1111-111111111501', 'Audit Actor', '{"field":"name"}'::jsonb),
	('99999999-9999-9999-9999-999999999502', '2024-01-01T12:00:00Z', '22222222-2222-2222-2222-222222222501',
		'11111111-1111-1111-1111-111111111501', 'audit-actor@example.com', NULL, '', 'audit_log_repository.created', 'user',
		'11111111-1111-1111-1111-111111111501', 'Audit Actor', NULL)`)
	s.Require().NoError(err)
}

func (s *RepositorySuite) TearDownTest() {
	_, err := s.db.ExecContext(s.ctx, `TRUNCATE TABLE audit_logs, organizations, users CASCADE`)
	s.Require().NoError(err)
}

func (s *RepositorySuite) TestCreate() {
	raw := json.RawMessage(`{"ok":true}`)
	log := &models.AuditLog{
		ID:                  uuid.MustParse(seedAuditLogNewID),
		CreatedAt:           seedAuditLogTime,
		OrganizationID:      uuid.MustParse(seedAuditOrgID),
		ActorID:             uuid.MustParse(seedAuditActorID),
		ActorInfo:           "audit-actor@example.com",
		ImpersonatedByEmail: "",
		Action:              "audit_log_repository.updated",
		ResourceType:        "role",
		ResourceID:          uuid.MustParse(seedAuditActorID),
		Data:                &raw,
	}
	s.Require().NoError(s.repo.Create(s.ctx, log))

	rows, err := s.repo.ListByOrganizationID(s.ctx, seedAuditOrgID, nil, 10, 0)
	s.Require().NoError(err)
	s.GreaterOrEqual(len(*rows), 3)
}

func (s *RepositorySuite) TestListByOrganizationID() {
	rows, err := s.repo.ListByOrganizationID(s.ctx, seedAuditOrgID, nil, 10, 0)
	s.Require().NoError(err)
	s.Require().Len(*rows, 2)
	s.Equal(seedAuditLogID, (*rows)[0].ID.String())
}

func (s *RepositorySuite) TestListByOrganizationID_WithDateFilters() {
	rows, err := s.repo.ListByOrganizationID(s.ctx, seedAuditOrgID, &audit_log_repository.AuditLogFilters{
		StartDate: &seedAuditFilterStart,
		EndDate:   &seedAuditFilterEnd,
	}, 10, 0)
	s.Require().NoError(err)
	s.Require().Len(*rows, 1)
	s.Equal(seedAuditLogID, (*rows)[0].ID.String())
}

func (s *RepositorySuite) TestWithTx() {
	raw := json.RawMessage(`{"tx":true}`)
	log := &models.AuditLog{
		ID:                  uuid.MustParse(seedAuditTxID),
		CreatedAt:           seedAuditLogTime,
		OrganizationID:      uuid.MustParse(seedAuditOrgID),
		ActorID:             uuid.MustParse(seedAuditActorID),
		ActorInfo:           "audit-actor@example.com",
		ImpersonatedByEmail: "",
		Action:              "audit_log_repository.updated",
		ResourceType:        "user",
		ResourceID:          uuid.MustParse(seedAuditActorID),
		Data:                &raw,
	}
	err := chi_repository.RunInTx(s.ctx, s.db, nil, func(tx chi_repository.Tx) error {
		return s.repo.WithTx(tx).Create(s.ctx, log)
	})
	s.Require().NoError(err)

	rows, err := s.repo.ListByOrganizationID(s.ctx, seedAuditOrgID, nil, 10, 0)
	s.Require().NoError(err)
	found := false
	for _, row := range *rows {
		if row.ID.String() == seedAuditTxID {
			found = true
			break
		}
	}
	s.True(found)
}
