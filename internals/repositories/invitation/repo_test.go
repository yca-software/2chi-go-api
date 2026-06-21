package invitation_repository_test

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
	invitation_repository "github.com/yca-software/2chi-go-api/internals/repositories/invitation"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_repository "github.com/yca-software/2chi-go-repository"
	chi_test "github.com/yca-software/2chi-go-test"
	chi_types "github.com/yca-software/2chi-go-types"
)

const (
	seedInvOrgID        = "22222222-2222-2222-2222-222222222301"
	seedInvRoleID       = "33333333-3333-3333-3333-333333333301"
	seedInvUserID       = "11111111-1111-1111-1111-111111111301"
	seedInvActiveID     = "77777777-7777-7777-7777-777777777301"
	seedInvTokenHash    = "invite-token-hash-active"
	seedInvUpdateID     = "77777777-7777-7777-7777-777777777302"
	seedInvNewID        = "77777777-7777-7777-7777-777777777306"
	seedInvNewTokenHash = "invite-token-hash-new"
	seedInvTxID         = "77777777-7777-7777-7777-777777777307"
	seedInvTxTokenHash  = "invite-token-hash-tx"
)

var (
	seedCreatedAtTime    = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	seedInvExpiresFuture = time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
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

func TestInvitationsRepositorySuite(t *testing.T) {
	suite.Run(t, new(InvitationsRepositorySuite))
}

type InvitationsRepositorySuite struct {
	suite.Suite

	db   *sqlx.DB
	repo invitation_repository.InvitationsRepository
	ctx  context.Context
}

func (s *InvitationsRepositorySuite) SetupSuite() {
	testDB, err := chi_test.Get(moduleMigrationsDir())
	s.Require().NoError(err)

	s.db, err = testDB.SQLx()
	s.Require().NoError(err)

	s.repo = invitation_repository.NewInvitationsRepository(s.db, nil)
	s.ctx = context.Background()
}

func (s *InvitationsRepositorySuite) SetupTest() {
	_, err := s.db.ExecContext(s.ctx, `
INSERT INTO users (
	id, created_at, deleted_at, first_name, last_name, language, email, password
) VALUES (
	'11111111-1111-1111-1111-111111111301', '2024-01-01T00:00:00Z', NULL, 'Inviter', 'User', 'en',
	'inviter@example.com', 'hash'
);
INSERT INTO organizations (id, created_at, deleted_at, name) VALUES (
	'22222222-2222-2222-2222-222222222301', '2024-01-01T00:00:00Z', NULL, 'Inv Org'
);
INSERT INTO roles (id, created_at, organization_id, name, description, permissions, locked) VALUES (
	'33333333-3333-3333-3333-333333333301', '2024-01-01T00:00:00Z', '22222222-2222-2222-2222-222222222301',
	'Member', 'Member role', '["org:read"]'::jsonb, false
);
INSERT INTO invitations (
	id, created_at, expires_at, accepted_at, revoked_at,
	organization_id, role_id, email, invited_by_id, invited_by_email, token_hash
) VALUES
	('77777777-7777-7777-7777-777777777301', '2024-01-01T00:00:00Z', '2030-01-01T00:00:00Z', NULL, NULL,
		'22222222-2222-2222-2222-222222222301', '33333333-3333-3333-3333-333333333301', 'pending@example.com',
		'11111111-1111-1111-1111-111111111301', 'inviter@example.com', 'invite-token-hash-active'),
	('77777777-7777-7777-7777-777777777302', '2024-01-01T00:00:00Z', '2030-01-01T00:00:00Z', NULL, NULL,
		'22222222-2222-2222-2222-222222222301', '33333333-3333-3333-3333-333333333301', 'update@example.com',
		'11111111-1111-1111-1111-111111111301', 'inviter@example.com', 'invite-token-hash-update'),
	('77777777-7777-7777-7777-777777777303', '2024-01-01T00:00:00Z', '2030-01-01T00:00:00Z', '2020-01-01T00:00:00Z', NULL,
		'22222222-2222-2222-2222-222222222301', '33333333-3333-3333-3333-333333333301', 'accepted@example.com',
		'11111111-1111-1111-1111-111111111301', 'inviter@example.com', 'invite-token-hash-accepted'),
	('77777777-7777-7777-7777-777777777304', '2024-01-01T00:00:00Z', '2030-01-01T00:00:00Z', NULL, '2020-01-01T00:00:00Z',
		'22222222-2222-2222-2222-222222222301', '33333333-3333-3333-3333-333333333301', 'revoked@example.com',
		'11111111-1111-1111-1111-111111111301', 'inviter@example.com', 'invite-token-hash-revoked'),
	('77777777-7777-7777-7777-777777777305', '2020-01-01T00:00:00Z', '2020-01-02T00:00:00Z', NULL, NULL,
		'22222222-2222-2222-2222-222222222301', '33333333-3333-3333-3333-333333333301', 'stale@example.com',
		'11111111-1111-1111-1111-111111111301', 'inviter@example.com', 'invite-token-hash-stale')`)
	s.Require().NoError(err)
}

func (s *InvitationsRepositorySuite) TearDownTest() {
	_, err := s.db.ExecContext(s.ctx, `TRUNCATE TABLE invitations, roles, organizations, users CASCADE`)
	s.Require().NoError(err)
}

func (s *InvitationsRepositorySuite) TestCreateInvitation() {
	inv := s.newInvitation(seedInvNewID, seedInvNewTokenHash, "new@example.com")
	s.Require().NoError(s.repo.CreateInvitation(s.ctx, inv))

	got, err := s.repo.GetInvitationByID(s.ctx, seedInvOrgID, seedInvNewID)
	s.Require().NoError(err)
	s.Equal("new@example.com", got.Email)
}

func (s *InvitationsRepositorySuite) TestGetInvitationByTokenHash() {
	got, err := s.repo.GetInvitationByTokenHash(s.ctx, seedInvTokenHash)
	s.Require().NoError(err)
	s.Equal(seedInvActiveID, got.ID.String())
}

func (s *InvitationsRepositorySuite) TestUpdateInvitation() {
	inv, err := s.repo.GetInvitationByID(s.ctx, seedInvOrgID, seedInvUpdateID)
	s.Require().NoError(err)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	inv.RevokedAt = &now
	s.Require().NoError(s.repo.UpdateInvitation(s.ctx, inv))

	got, err := s.repo.GetInvitationByID(s.ctx, seedInvOrgID, seedInvUpdateID)
	s.Require().NoError(err)
	s.NotNil(got.RevokedAt)
}

func (s *InvitationsRepositorySuite) TestListInvitationsByOrganizationID() {
	rows, err := s.repo.ListInvitationsByOrganizationID(s.ctx, seedInvOrgID)
	s.Require().NoError(err)
	s.Len(*rows, 3)
}

func (s *InvitationsRepositorySuite) TestCleanupStaleInvitations() {
	s.Require().NoError(s.repo.CleanupStaleInvitations(s.ctx))

	_, err := s.repo.GetInvitationByTokenHash(s.ctx, "invite-token-hash-stale")
	s.requireNotFound(err)

	got, err := s.repo.GetInvitationByTokenHash(s.ctx, seedInvTokenHash)
	s.Require().NoError(err)
	s.Equal(seedInvActiveID, got.ID.String())
}

func (s *InvitationsRepositorySuite) TestWithTx() {
	inv := s.newInvitation(seedInvTxID, seedInvTxTokenHash, "tx@example.com")
	err := chi_repository.RunInTx(s.ctx, s.db, nil, func(tx chi_repository.Tx) error {
		return s.repo.WithTx(tx).CreateInvitation(s.ctx, inv)
	})
	s.Require().NoError(err)

	got, err := s.repo.GetInvitationByID(s.ctx, seedInvOrgID, seedInvTxID)
	s.Require().NoError(err)
	s.Equal("tx@example.com", got.Email)
}

func (s *InvitationsRepositorySuite) newInvitation(id, tokenHash, email string) *models.Invitation {
	return &models.Invitation{
		ModelBase: chi_types.ModelBase{
			ID:        uuid.MustParse(id),
			CreatedAt: seedCreatedAtTime,
		},
		ExpiresAt:      seedInvExpiresFuture,
		OrganizationID: uuid.MustParse(seedInvOrgID),
		RoleID:         uuid.MustParse(seedInvRoleID),
		Email:          email,
		InvitedByID:    uuid.NullUUID{UUID: uuid.MustParse(seedInvUserID), Valid: true},
		InvitedByEmail: "inviter@example.com",
		TokenHash:      tokenHash,
	}
}

func (s *InvitationsRepositorySuite) requireNotFound(err error) {
	s.T().Helper()
	s.Require().Error(err)
	var apiErr *chi_error.Error
	s.Require().True(errors.As(err, &apiErr))
	s.Equal(http.StatusNotFound, apiErr.StatusCode)
}
