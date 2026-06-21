//go:build integration

package impersonation_session_repository_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/suite"

	"github.com/yca-software/2chi-go-api/internals/models"
	impersonation_session_repository "github.com/yca-software/2chi-go-api/internals/repositories/impersonation_session"
	"github.com/yca-software/2chi-go-api/internals/packages/testutil"
	chi_repository "github.com/yca-software/2chi-go-repository"
)

const (
	seedImpersonationAdminID        = "11111111-1111-1111-1111-111111111701"
	seedImpersonationTargetID       = "11111111-1111-1111-1111-111111111702"
	seedImpersonationRefreshID      = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa701"
	seedImpersonationExpiredRefresh = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa702"
	seedImpersonationActiveSession  = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb701"
	seedImpersonationNewSession     = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb702"
	seedImpersonationOrphanSession  = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb703"
	seedImpersonationTxSession      = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb704"
)

var seedCreatedAtTime = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

func TestMain(m *testing.M) {
	os.Exit(testutil.IntegrationTestMain(m))
}

func TestImpersonationSessionsRepositorySuite(t *testing.T) {
	suite.Run(t, new(ImpersonationSessionsRepositorySuite))
}

type ImpersonationSessionsRepositorySuite struct {
	suite.Suite

	db   *sqlx.DB
	repo impersonation_session_repository.ImpersonationSessionsRepository
	ctx  context.Context
}

func (s *ImpersonationSessionsRepositorySuite) SetupSuite() {
	testDB, err := testutil.GetIntegrationDB()
	s.Require().NoError(err)

	s.db, err = testDB.SQLx()
	s.Require().NoError(err)

	s.repo = impersonation_session_repository.NewImpersonationSessionsRepository(s.db, nil)
	s.ctx = context.Background()
}

func (s *ImpersonationSessionsRepositorySuite) SetupTest() {
	_, err := s.db.ExecContext(s.ctx, `
INSERT INTO users (
	id, created_at, deleted_at, first_name, last_name, language, email, password
) VALUES
	('11111111-1111-1111-1111-111111111701', '2024-01-01T00:00:00Z', NULL, 'Admin', 'User', 'en',
		'admin-impersonate@example.com', 'hash'),
	('11111111-1111-1111-1111-111111111702', '2024-01-01T00:00:00Z', NULL, 'Target', 'User', 'en',
		'target-impersonate@example.com', 'hash');
INSERT INTO user_refresh_tokens (
	id, user_id, created_at, expires_at, revoked_at, ip, user_agent, token_hash, impersonated_by
) VALUES
	('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa701', '11111111-1111-1111-1111-111111111702', '2024-01-01T00:00:00Z', '2030-01-01T00:00:00Z', NULL, '127.0.0.1', 'agent', 'refresh-impersonation-active', '11111111-1111-1111-1111-111111111701'),
	('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa702', '11111111-1111-1111-1111-111111111702', '2020-01-01T00:00:00Z', '2020-01-02T00:00:00Z', NULL, '127.0.0.1', 'agent', 'refresh-impersonation-expired', '11111111-1111-1111-1111-111111111701');
INSERT INTO impersonation_sessions (
	id, started_at, ended_at, end_reason, admin_id, admin_email, target_user_id, target_user_email,
	refresh_token_id, ip, user_agent
) VALUES
	('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb701', '2024-01-01T00:00:00Z', NULL, NULL,
		'11111111-1111-1111-1111-111111111701', 'admin-impersonate@example.com',
		'11111111-1111-1111-1111-111111111702', 'target-impersonate@example.com',
		'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa701', '127.0.0.1', 'agent'),
	('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbb703', '2020-01-01T00:00:00Z', NULL, NULL,
		'11111111-1111-1111-1111-111111111701', 'admin-impersonate@example.com',
		'11111111-1111-1111-1111-111111111702', 'target-impersonate@example.com',
		'00000000-0000-0000-0000-000000000099', '127.0.0.1', 'agent')`)
	s.Require().NoError(err)
}

func (s *ImpersonationSessionsRepositorySuite) TearDownTest() {
	_, err := s.db.ExecContext(s.ctx, `TRUNCATE TABLE impersonation_sessions, user_refresh_tokens, users CASCADE`)
	s.Require().NoError(err)
}

func (s *ImpersonationSessionsRepositorySuite) TestCreateSession() {
	session := &models.ImpersonationSession{
		ID:              uuid.MustParse(seedImpersonationNewSession),
		StartedAt:       seedCreatedAtTime,
		AdminID:         uuid.MustParse(seedImpersonationAdminID),
		AdminEmail:      "admin-impersonate@example.com",
		TargetUserID:    uuid.MustParse(seedImpersonationTargetID),
		TargetUserEmail: "target-impersonate@example.com",
		RefreshTokenID:  uuid.MustParse(seedImpersonationRefreshID),
		IP:              "127.0.0.1",
		UserAgent:       "agent",
	}
	s.Require().NoError(s.repo.CreateSession(s.ctx, session))

	var endedAt *time.Time
	err := s.db.GetContext(s.ctx, &endedAt, `SELECT ended_at FROM impersonation_sessions WHERE id = $1`, seedImpersonationNewSession)
	s.Require().NoError(err)
	s.Nil(endedAt)
}

func (s *ImpersonationSessionsRepositorySuite) TestEndSessionByRefreshTokenID() {
	endedAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	s.Require().NoError(s.repo.EndSessionByRefreshTokenID(s.ctx, uuid.MustParse(seedImpersonationRefreshID), endedAt, "logout"))

	var endReason *string
	err := s.db.GetContext(s.ctx, &endReason, `
SELECT end_reason FROM impersonation_sessions WHERE id = $1`, seedImpersonationActiveSession)
	s.Require().NoError(err)
	s.Require().NotNil(endReason)
	s.Equal("logout", *endReason)
}

func (s *ImpersonationSessionsRepositorySuite) TestEndExpiredSessions() {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	count, err := s.repo.EndExpiredSessions(s.ctx, now, "expired")
	s.Require().NoError(err)
	s.GreaterOrEqual(count, int64(1))

	var endedAt *time.Time
	err = s.db.GetContext(s.ctx, &endedAt, `
SELECT ended_at FROM impersonation_sessions WHERE id = $1`, seedImpersonationOrphanSession)
	s.Require().NoError(err)
	s.NotNil(endedAt)
}

func (s *ImpersonationSessionsRepositorySuite) TestWithTx() {
	session := &models.ImpersonationSession{
		ID:              uuid.MustParse(seedImpersonationTxSession),
		StartedAt:       seedCreatedAtTime,
		AdminID:         uuid.MustParse(seedImpersonationAdminID),
		AdminEmail:      "admin-impersonate@example.com",
		TargetUserID:    uuid.MustParse(seedImpersonationTargetID),
		TargetUserEmail: "target-impersonate@example.com",
		RefreshTokenID:  uuid.MustParse(seedImpersonationExpiredRefresh),
		IP:              "127.0.0.1",
		UserAgent:       "agent",
	}
	err := chi_repository.RunInTx(s.ctx, s.db, nil, func(tx chi_repository.Tx) error {
		return s.repo.WithTx(tx).CreateSession(s.ctx, session)
	})
	s.Require().NoError(err)

	var count int
	err = s.db.GetContext(s.ctx, &count, `SELECT COUNT(*) FROM impersonation_sessions WHERE id = $1`, seedImpersonationTxSession)
	s.Require().NoError(err)
	s.Equal(1, count)
}
