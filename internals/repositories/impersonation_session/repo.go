package impersonation_session_repository

import (
	"context"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/yca-software/2chi-go-api/internals/models"
	chi_observer "github.com/yca-software/2chi-go-observer"
	chi_repository "github.com/yca-software/2chi-go-repository"
)

const TableName = "impersonation_sessions"

var Columns = []string{
	"id", "started_at", "ended_at", "end_reason",
	"admin_id", "admin_email", "target_user_id", "target_user_email",
	"refresh_token_id", "ip", "user_agent",
}

type Repository interface {
	WithTx(tx chi_repository.Tx) Repository

	Create(ctx context.Context, session *models.ImpersonationSession) error
	EndByRefreshTokenID(ctx context.Context, refreshTokenID uuid.UUID, endedAt time.Time, reason string) error
	EndExpired(ctx context.Context, now time.Time, expiredReason string) (int64, error)
}

type repository struct {
	repo chi_repository.Repository[models.ImpersonationSession]
}

func NewRepository(db *sqlx.DB, metricsHook chi_observer.QueryMetricsHook) Repository {
	return &repository{
		repo: chi_repository.NewRepository[models.ImpersonationSession](db, TableName, Columns, metricsHook),
	}
}

func (r *repository) WithTx(tx chi_repository.Tx) Repository {
	return &repository{repo: r.repo.WithTx(tx)}
}

func (r *repository) Create(ctx context.Context, session *models.ImpersonationSession) error {
	data := map[string]any{
		"id":                session.ID,
		"admin_id":          session.AdminID,
		"admin_email":       session.AdminEmail,
		"target_user_id":    session.TargetUserID,
		"target_user_email": session.TargetUserEmail,
		"refresh_token_id":  session.RefreshTokenID,
		"ip":                session.IP,
		"user_agent":        session.UserAgent,
	}
	if !session.StartedAt.IsZero() {
		data["started_at"] = session.StartedAt
	}
	return r.repo.Create(ctx, data)
}

func (r *repository) EndByRefreshTokenID(ctx context.Context, refreshTokenID uuid.UUID, endedAt time.Time, reason string) error {
	return r.repo.Update(ctx, squirrel.And{
		squirrel.Eq{"refresh_token_id": refreshTokenID},
		squirrel.Eq{"ended_at": nil},
	}, map[string]any{
		"ended_at":   endedAt,
		"end_reason": reason,
	})
}

func (r *repository) EndExpired(ctx context.Context, now time.Time, expiredReason string) (int64, error) {
	const query = `
UPDATE impersonation_sessions s
SET
    ended_at = COALESCE(
        (SELECT COALESCE(rt.revoked_at, rt.expires_at) FROM user_refresh_tokens rt WHERE rt.id = s.refresh_token_id),
        s.started_at + interval '1 hour'
    ),
    end_reason = $2
WHERE s.ended_at IS NULL
AND (
    EXISTS (
        SELECT 1 FROM user_refresh_tokens rt
        WHERE rt.id = s.refresh_token_id
          AND (rt.expires_at <= $1 OR rt.revoked_at IS NOT NULL)
    )
    OR (
        NOT EXISTS (SELECT 1 FROM user_refresh_tokens rt WHERE rt.id = s.refresh_token_id)
        AND s.started_at + interval '1 hour' <= $1
    )
)`
	result, err := r.repo.DB().ExecContext(ctx, query, now, expiredReason)
	if err != nil {
		return 0, chi_repository.WrapSQLError(err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, chi_repository.WrapSQLError(err)
	}
	return rowsAffected, nil
}
