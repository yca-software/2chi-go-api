package team_member_repository

import (
	"context"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"

	"github.com/yca-software/2chi-go-api/internals/models"
	chi_observer "github.com/yca-software/2chi-go-observer"
	chi_repository "github.com/yca-software/2chi-go-repository"
)

const TableName = "team_members"

var Columns = []string{
	"id", "created_at", "updated_at", "organization_id", "team_id", "user_id",
}

type Repository interface {
	WithTx(tx chi_repository.Tx) Repository

	Create(ctx context.Context, member *models.TeamMember) error
	Delete(ctx context.Context, organizationID, id string) error

	GetByID(ctx context.Context, organizationID, id string) (*models.TeamMember, error)
	GetByIDWithUser(ctx context.Context, organizationID, id string) (*models.TeamMemberWithUser, error)

	ListByUserID(ctx context.Context, userID string) (*[]models.TeamMemberWithTeam, error)
	ListByTeamID(ctx context.Context, organizationID, teamID string) (*[]models.TeamMemberWithUser, error)
	ListByOrganizationID(ctx context.Context, organizationID string) (*[]models.TeamMemberWithUser, error)
}

type repository struct {
	repo chi_repository.Repository[models.TeamMember]
}

func NewRepository(db *sqlx.DB, metricsHook chi_observer.QueryMetricsHook) Repository {
	return &repository{
		repo: chi_repository.NewRepository[models.TeamMember](db, TableName, Columns, metricsHook),
	}
}

func (r *repository) WithTx(tx chi_repository.Tx) Repository {
	return &repository{
		repo: r.repo.WithTx(tx),
	}
}

func teamMemberSelectColumns(extra ...string) []string {
	columns := make([]string, 0, len(Columns)+len(extra))
	for _, column := range Columns {
		columns = append(columns, fmt.Sprintf("tm.%s", column))
	}
	return append(columns, extra...)
}

func (r *repository) Create(ctx context.Context, member *models.TeamMember) error {
	now := time.Now()
	return r.repo.Create(ctx, map[string]any{
		"id":              member.ID,
		"created_at":      now,
		"updated_at":      now,
		"organization_id": member.OrganizationID,
		"team_id":         member.TeamID,
		"user_id":         member.UserID,
	})
}

func (r *repository) Delete(ctx context.Context, organizationID, id string) error {
	return r.repo.Delete(ctx, squirrel.And{
		squirrel.Eq{"id": id},
		squirrel.Eq{"organization_id": organizationID},
	})
}

func (r *repository) GetByID(ctx context.Context, organizationID, id string) (*models.TeamMember, error) {
	return r.repo.Get(ctx, squirrel.And{
		squirrel.Eq{"id": id},
		squirrel.Eq{"organization_id": organizationID},
	}, nil)
}

func (r *repository) GetByIDWithUser(ctx context.Context, organizationID, id string) (*models.TeamMemberWithUser, error) {
	columns := teamMemberSelectColumns(
		"u.email as user_email",
		"u.first_name as user_first_name",
		"u.last_name as user_last_name",
	)
	query := r.repo.GetQueryBuilder().Select(columns...).
		From(fmt.Sprintf("%s AS tm", TableName)).
		LeftJoin("users as u ON u.id = tm.user_id").
		Where(squirrel.And{
			squirrel.Eq{"tm.id": id},
			squirrel.Eq{"tm.organization_id": organizationID},
		})
	sqlStr, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	dest := new(models.TeamMemberWithUser)
	if err = r.repo.DB().GetContext(ctx, dest, sqlStr, args...); err != nil {
		return nil, chi_repository.WrapSQLError(err)
	}
	return dest, nil
}

func (r *repository) ListByUserID(ctx context.Context, userID string) (*[]models.TeamMemberWithTeam, error) {
	columns := teamMemberSelectColumns("t.name as team_name")
	query := r.repo.GetQueryBuilder().Select(columns...).
		From(fmt.Sprintf("%s AS tm", TableName)).
		LeftJoin("teams as t ON t.id = tm.team_id").
		Where(squirrel.Eq{"tm.user_id": userID})
	sqlStr, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	dest := new([]models.TeamMemberWithTeam)
	if err = r.repo.DB().SelectContext(ctx, dest, sqlStr, args...); err != nil {
		return nil, chi_repository.WrapSQLError(err)
	}
	return dest, nil
}

func (r *repository) ListByTeamID(ctx context.Context, organizationID, teamID string) (*[]models.TeamMemberWithUser, error) {
	columns := teamMemberSelectColumns(
		"u.email as user_email",
		"u.first_name as user_first_name",
		"u.last_name as user_last_name",
	)
	query := r.repo.GetQueryBuilder().Select(columns...).
		From(fmt.Sprintf("%s AS tm", TableName)).
		LeftJoin("users as u ON u.id = tm.user_id").
		Where(squirrel.And{
			squirrel.Eq{"tm.organization_id": organizationID},
			squirrel.Eq{"tm.team_id": teamID},
		})
	sqlStr, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	dest := new([]models.TeamMemberWithUser)
	if err = r.repo.DB().SelectContext(ctx, dest, sqlStr, args...); err != nil {
		return nil, chi_repository.WrapSQLError(err)
	}
	return dest, nil
}

func (r *repository) ListByOrganizationID(ctx context.Context, organizationID string) (*[]models.TeamMemberWithUser, error) {
	columns := teamMemberSelectColumns(
		"u.email as user_email",
		"u.first_name as user_first_name",
		"u.last_name as user_last_name",
	)
	query := r.repo.GetQueryBuilder().Select(columns...).
		From(fmt.Sprintf("%s AS tm", TableName)).
		LeftJoin("users as u ON u.id = tm.user_id").
		Where(squirrel.Eq{"tm.organization_id": organizationID})
	sqlStr, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	dest := new([]models.TeamMemberWithUser)
	if err = r.repo.DB().SelectContext(ctx, dest, sqlStr, args...); err != nil {
		return nil, chi_repository.WrapSQLError(err)
	}
	return dest, nil
}
