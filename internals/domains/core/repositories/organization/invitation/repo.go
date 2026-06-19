package invitation_repository

import (
	"context"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"

	chi_archive "github.com/yca-software/2chi-go-archive"
	"github.com/yca-software/2chi-go-api/internals/domains/core/models"
	chi_observer "github.com/yca-software/2chi-go-observer"
	chi_repository "github.com/yca-software/2chi-go-repository"
)

const (
	InvitationsTableName = "invitations"
)

var (
	InvitationsColumns = []string{
		"id", "created_at", "updated_at", "expires_at", "accepted_at", "revoked_at",
		"organization_id", "role_id", "email", "invited_by_id", "invited_by_email", "token_hash",
	}
)

type InvitationsRepository interface {
	WithTx(tx chi_repository.Tx) InvitationsRepository

	CreateInvitation(ctx context.Context, invitation *models.Invitation) error
	UpdateInvitation(ctx context.Context, invitation *models.Invitation) error

	GetInvitationByID(ctx context.Context, organizationID, id string) (*models.Invitation, error)
	GetInvitationByTokenHash(ctx context.Context, tokenHash string) (*models.Invitation, error)
	ListInvitationsByOrganizationID(ctx context.Context, organizationID string) (*[]models.Invitation, error)

	CleanupStaleInvitations(ctx context.Context) error
}

type invitationsRepository struct {
	invitationsRepo chi_repository.Repository[models.Invitation]
}

func NewInvitationsRepository(db *sqlx.DB, metricsHook chi_observer.QueryMetricsHook) InvitationsRepository {
	return &invitationsRepository{
		invitationsRepo: chi_repository.NewRepository[models.Invitation](db, InvitationsTableName, InvitationsColumns, metricsHook),
	}
}

func (r *invitationsRepository) WithTx(tx chi_repository.Tx) InvitationsRepository {
	return &invitationsRepository{
		invitationsRepo: r.invitationsRepo.WithTx(tx),
	}
}

func (r *invitationsRepository) CreateInvitation(ctx context.Context, invitation *models.Invitation) error {
	now := time.Now()
	return r.invitationsRepo.Create(ctx, map[string]any{
		"id":               invitation.ID,
		"created_at":       now,
		"updated_at":       now,
		"expires_at":       invitation.ExpiresAt,
		"organization_id":  invitation.OrganizationID,
		"role_id":          invitation.RoleID,
		"email":            invitation.Email,
		"invited_by_id":    invitation.InvitedByID,
		"invited_by_email": invitation.InvitedByEmail,
		"token_hash":       invitation.TokenHash,
	})
}

func (r *invitationsRepository) UpdateInvitation(ctx context.Context, invitation *models.Invitation) error {
	return r.invitationsRepo.Update(ctx, squirrel.And{
		squirrel.Eq{"id": invitation.ID},
		squirrel.Eq{"organization_id": invitation.OrganizationID},
		squirrel.Eq{"accepted_at": nil},
		squirrel.Eq{"revoked_at": nil},
	}, map[string]any{
		"accepted_at": invitation.AcceptedAt,
		"revoked_at":  invitation.RevokedAt,
		"updated_at":  time.Now(),
	})
}

func (r *invitationsRepository) GetInvitationByID(ctx context.Context, organizationID, id string) (*models.Invitation, error) {
	return r.invitationsRepo.Get(ctx, squirrel.And{
		squirrel.Eq{"id": id},
		squirrel.Eq{"organization_id": organizationID},
	}, nil)
}

func (r *invitationsRepository) GetInvitationByTokenHash(ctx context.Context, tokenHash string) (*models.Invitation, error) {
	return r.invitationsRepo.Get(ctx, squirrel.Eq{"token_hash": tokenHash}, nil)
}

func (r *invitationsRepository) ListInvitationsByOrganizationID(ctx context.Context, organizationID string) (*[]models.Invitation, error) {
	return r.invitationsRepo.Select(ctx, squirrel.And{
		squirrel.Eq{"organization_id": organizationID},
		squirrel.Eq{"accepted_at": nil},
		squirrel.Eq{"revoked_at": nil},
	}, nil, "created_at DESC")
}

func (r *invitationsRepository) CleanupStaleInvitations(ctx context.Context) error {
	threshold := time.Now().Add(-chi_archive.ArchivedDataRetentionPeriod)
	return r.invitationsRepo.Delete(ctx, squirrel.Or{
		squirrel.And{
			squirrel.NotEq{"accepted_at": nil},
			squirrel.LtOrEq{"accepted_at": threshold},
		},
		squirrel.And{
			squirrel.NotEq{"revoked_at": nil},
			squirrel.LtOrEq{"revoked_at": threshold},
		},
		squirrel.And{
			squirrel.Eq{"accepted_at": nil},
			squirrel.Eq{"revoked_at": nil},
			squirrel.LtOrEq{"expires_at": threshold},
		},
	})
}
