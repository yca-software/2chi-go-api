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

const TeamMembersTableName = "team_members"

var TeamMembersColumns = []string{
	"id", "created_at", "updated_at", "organization_id", "team_id", "user_id",
}

type TeamMembersRepository interface {
	WithTx(tx chi_repository.Tx) TeamMembersRepository

	CreateTeamMember(ctx context.Context, member *models.TeamMember) error
	DeleteTeamMember(ctx context.Context, organizationID, id string) error

	GetTeamMemberByID(ctx context.Context, organizationID, id string) (*models.TeamMember, error)
	GetTeamMemberByIDWithUser(ctx context.Context, organizationID, id string) (*models.TeamMemberWithUser, error)

	ListTeamMembersByUserID(ctx context.Context, userID string) (*[]models.TeamMemberWithTeam, error)
	ListTeamMembersByTeamID(ctx context.Context, organizationID, teamID string) (*[]models.TeamMemberWithUser, error)
	ListTeamMembersByOrganizationID(ctx context.Context, organizationID string) (*[]models.TeamMemberWithUser, error)
}

type teamMembersRepository struct {
	teamMembersRepo chi_repository.Repository[models.TeamMember]
}

func NewTeamMembersRepository(db *sqlx.DB, metricsHook chi_observer.QueryMetricsHook) TeamMembersRepository {
	return &teamMembersRepository{
		teamMembersRepo: chi_repository.NewRepository[models.TeamMember](db, TeamMembersTableName, TeamMembersColumns, metricsHook),
	}
}

func (r *teamMembersRepository) WithTx(tx chi_repository.Tx) TeamMembersRepository {
	return &teamMembersRepository{
		teamMembersRepo: r.teamMembersRepo.WithTx(tx),
	}
}

func teamMemberSelectColumns(extra ...string) []string {
	columns := make([]string, 0, len(TeamMembersColumns)+len(extra))
	for _, column := range TeamMembersColumns {
		columns = append(columns, fmt.Sprintf("tm.%s", column))
	}
	return append(columns, extra...)
}

func (r *teamMembersRepository) CreateTeamMember(ctx context.Context, member *models.TeamMember) error {
	now := time.Now()
	return r.teamMembersRepo.Create(ctx, map[string]any{
		"id":              member.ID,
		"created_at":      now,
		"updated_at":      now,
		"organization_id": member.OrganizationID,
		"team_id":         member.TeamID,
		"user_id":         member.UserID,
	})
}

func (r *teamMembersRepository) DeleteTeamMember(ctx context.Context, organizationID, id string) error {
	return r.teamMembersRepo.Delete(ctx, squirrel.And{
		squirrel.Eq{"id": id},
		squirrel.Eq{"organization_id": organizationID},
	})
}

func (r *teamMembersRepository) GetTeamMemberByID(ctx context.Context, organizationID, id string) (*models.TeamMember, error) {
	return r.teamMembersRepo.Get(ctx, squirrel.And{
		squirrel.Eq{"id": id},
		squirrel.Eq{"organization_id": organizationID},
	}, nil)
}

func (r *teamMembersRepository) GetTeamMemberByIDWithUser(ctx context.Context, organizationID, id string) (*models.TeamMemberWithUser, error) {
	columns := teamMemberSelectColumns(
		"u.email as user_email",
		"u.first_name as user_first_name",
		"u.last_name as user_last_name",
	)
	query := r.teamMembersRepo.GetQueryBuilder().Select(columns...).
		From(fmt.Sprintf("%s AS tm", TeamMembersTableName)).
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
	if err = r.teamMembersRepo.DB().GetContext(ctx, dest, sqlStr, args...); err != nil {
		return nil, chi_repository.WrapSQLError(err)
	}
	return dest, nil
}

func (r *teamMembersRepository) ListTeamMembersByUserID(ctx context.Context, userID string) (*[]models.TeamMemberWithTeam, error) {
	columns := teamMemberSelectColumns("t.name as team_name")
	query := r.teamMembersRepo.GetQueryBuilder().Select(columns...).
		From(fmt.Sprintf("%s AS tm", TeamMembersTableName)).
		LeftJoin("teams as t ON t.id = tm.team_id").
		Where(squirrel.Eq{"tm.user_id": userID})
	sqlStr, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	dest := new([]models.TeamMemberWithTeam)
	if err = r.teamMembersRepo.DB().SelectContext(ctx, dest, sqlStr, args...); err != nil {
		return nil, chi_repository.WrapSQLError(err)
	}
	return dest, nil
}

func (r *teamMembersRepository) ListTeamMembersByTeamID(ctx context.Context, organizationID, teamID string) (*[]models.TeamMemberWithUser, error) {
	columns := teamMemberSelectColumns(
		"u.email as user_email",
		"u.first_name as user_first_name",
		"u.last_name as user_last_name",
	)
	query := r.teamMembersRepo.GetQueryBuilder().Select(columns...).
		From(fmt.Sprintf("%s AS tm", TeamMembersTableName)).
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
	if err = r.teamMembersRepo.DB().SelectContext(ctx, dest, sqlStr, args...); err != nil {
		return nil, chi_repository.WrapSQLError(err)
	}
	return dest, nil
}

func (r *teamMembersRepository) ListTeamMembersByOrganizationID(ctx context.Context, organizationID string) (*[]models.TeamMemberWithUser, error) {
	columns := teamMemberSelectColumns(
		"u.email as user_email",
		"u.first_name as user_first_name",
		"u.last_name as user_last_name",
	)
	query := r.teamMembersRepo.GetQueryBuilder().Select(columns...).
		From(fmt.Sprintf("%s AS tm", TeamMembersTableName)).
		LeftJoin("users as u ON u.id = tm.user_id").
		Where(squirrel.Eq{"tm.organization_id": organizationID})
	sqlStr, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	dest := new([]models.TeamMemberWithUser)
	if err = r.teamMembersRepo.DB().SelectContext(ctx, dest, sqlStr, args...); err != nil {
		return nil, chi_repository.WrapSQLError(err)
	}
	return dest, nil
}
