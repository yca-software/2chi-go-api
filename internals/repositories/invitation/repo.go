package invitation_repository

import (
	"context"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"

	"github.com/yca-software/2chi-go-api/internals/models"
	chi_archive "github.com/yca-software/2chi-go-archive"
	chi_observer "github.com/yca-software/2chi-go-observer"
	chi_repository "github.com/yca-software/2chi-go-repository"
)

const (
	TableName = "invitations"
)

var (
	Columns = []string{
		"id", "created_at", "updated_at", "expires_at", "accepted_at", "revoked_at",
		"organization_id", "role_id", "email", "invited_by_id", "invited_by_email", "token_hash",
	}
)

type Repository interface {
	WithTx(tx chi_repository.Tx) Repository

	Create(ctx context.Context, invitation *models.Invitation) error
	Update(ctx context.Context, invitation *models.Invitation) error

	GetByID(ctx context.Context, organizationID, id string) (*models.Invitation, error)
	GetByTokenHash(ctx context.Context, tokenHash string) (*models.Invitation, error)
	ListByOrganizationID(ctx context.Context, organizationID string) (*[]models.Invitation, error)

	CleanupStale(ctx context.Context) error
}

type repository struct {
	repo chi_repository.Repository[models.Invitation]
}

func NewRepository(db *sqlx.DB, metricsHook chi_observer.QueryMetricsHook) Repository {
	return &repository{
		repo: chi_repository.NewRepository[models.Invitation](db, TableName, Columns, metricsHook),
	}
}

func (r *repository) WithTx(tx chi_repository.Tx) Repository {
	return &repository{
		repo: r.repo.WithTx(tx),
	}
}

func (r *repository) Create(ctx context.Context, invitation *models.Invitation) error {
	now := time.Now()
	return r.repo.Create(ctx, map[string]any{
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

func (r *repository) Update(ctx context.Context, invitation *models.Invitation) error {
	return r.repo.Update(ctx, squirrel.And{
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

func (r *repository) GetByID(ctx context.Context, organizationID, id string) (*models.Invitation, error) {
	return r.repo.Get(ctx, squirrel.And{
		squirrel.Eq{"id": id},
		squirrel.Eq{"organization_id": organizationID},
	}, nil)
}

func (r *repository) GetByTokenHash(ctx context.Context, tokenHash string) (*models.Invitation, error) {
	return r.repo.Get(ctx, squirrel.Eq{"token_hash": tokenHash}, nil)
}

func (r *repository) ListByOrganizationID(ctx context.Context, organizationID string) (*[]models.Invitation, error) {
	return r.repo.Select(ctx, squirrel.And{
		squirrel.Eq{"organization_id": organizationID},
		squirrel.Eq{"accepted_at": nil},
		squirrel.Eq{"revoked_at": nil},
	}, nil, "created_at DESC")
}

func (r *repository) CleanupStale(ctx context.Context) error {
	threshold := time.Now().Add(-chi_archive.ArchivedDataRetentionPeriod)
	return r.repo.Delete(ctx, squirrel.Or{
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
