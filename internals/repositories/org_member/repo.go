package organization_member_repository

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"

	"github.com/yca-software/2chi-go-api/internals/models"
	chi_observer "github.com/yca-software/2chi-go-observer"
	chi_repository "github.com/yca-software/2chi-go-repository"
)

const (
	TableName = "organization_members"
)

var (
	Columns = []string{
		"id", "created_at", "updated_at", "organization_id", "user_id", "role_id",
	}
)

type Repository interface {
	WithTx(tx chi_repository.Tx) Repository

	Create(ctx context.Context, member *models.OrganizationMember) error
	Update(ctx context.Context, member *models.OrganizationMember) error
	DeleteByUserID(ctx context.Context, organizationID, userID string) error
	DeleteByMemberID(ctx context.Context, organizationID, memberID string) error

	GetByUserID(ctx context.Context, organizationID, userID string) (*models.OrganizationMember, error)
	GetByMemberID(ctx context.Context, organizationID, memberID string) (*models.OrganizationMember, error)
	GetByUserIDWithUser(ctx context.Context, organizationID, userID string) (*models.OrganizationMemberWithUser, error)
	GetByMemberIDWithUser(ctx context.Context, organizationID, memberID string) (*models.OrganizationMemberWithUser, error)
	GetByUserIDWithOrganizationAndRole(ctx context.Context, organizationID, userID string) (*models.OrganizationMemberWithOrganizationAndRole, error)

	ListByUserID(ctx context.Context, userID string) (*[]models.OrganizationMemberWithOrganizationAndRole, error)
	ListByOrganizationID(ctx context.Context, organizationID string) (*[]models.OrganizationMemberWithUser, error)
	ListUserEmailsForRole(ctx context.Context, organizationID, roleID string) ([]string, error)
}

type repository struct {
	repo chi_repository.Repository[models.OrganizationMember]
}

func NewRepository(db *sqlx.DB, metricsHook chi_observer.QueryMetricsHook) Repository {
	return &repository{
		repo: chi_repository.NewRepository[models.OrganizationMember](db, TableName, Columns, metricsHook),
	}
}

func (r *repository) WithTx(tx chi_repository.Tx) Repository {
	return &repository{
		repo: r.repo.WithTx(tx),
	}
}

func organizationMemberSelectColumns(extra ...string) []string {
	columns := make([]string, 0, len(Columns)+len(extra))
	for _, column := range Columns {
		columns = append(columns, fmt.Sprintf("om.%s", column))
	}
	return append(columns, extra...)
}

func (r *repository) Create(ctx context.Context, member *models.OrganizationMember) error {
	now := time.Now()
	return r.repo.Create(ctx, map[string]any{
		"id":              member.ID,
		"created_at":      now,
		"updated_at":      now,
		"organization_id": member.OrganizationID,
		"user_id":         member.UserID,
		"role_id":         member.RoleID,
	})
}

func (r *repository) Update(ctx context.Context, member *models.OrganizationMember) error {
	return r.repo.Update(ctx, squirrel.And{
		squirrel.Eq{"id": member.ID},
		squirrel.Eq{"organization_id": member.OrganizationID},
	}, map[string]any{
		"role_id":    member.RoleID,
		"updated_at": time.Now(),
	})
}

func (r *repository) DeleteByUserID(ctx context.Context, organizationID, userID string) error {
	return r.repo.Delete(ctx, squirrel.And{
		squirrel.Eq{"organization_id": organizationID},
		squirrel.Eq{"user_id": userID},
	})
}

func (r *repository) DeleteByMemberID(ctx context.Context, organizationID, memberID string) error {
	return r.repo.Delete(ctx, squirrel.And{
		squirrel.Eq{"organization_id": organizationID},
		squirrel.Eq{"id": memberID},
	})
}

func (r *repository) GetByUserID(ctx context.Context, organizationID, userID string) (*models.OrganizationMember, error) {
	return r.repo.Get(ctx, squirrel.And{
		squirrel.Eq{"organization_id": organizationID},
		squirrel.Eq{"user_id": userID},
	}, nil)
}

func (r *repository) GetByMemberID(ctx context.Context, organizationID, memberID string) (*models.OrganizationMember, error) {
	return r.repo.Get(ctx, squirrel.And{
		squirrel.Eq{"organization_id": organizationID},
		squirrel.Eq{"id": memberID},
	}, nil)
}

func (r *repository) GetByUserIDWithUser(ctx context.Context, organizationID, userID string) (*models.OrganizationMemberWithUser, error) {
	columns := organizationMemberSelectColumns(
		"u.email as user_email",
		"u.first_name as user_first_name",
		"u.last_name as user_last_name",
	)
	query := r.repo.GetQueryBuilder().Select(columns...).
		From(fmt.Sprintf("%s AS om", TableName)).
		LeftJoin("users as u ON u.id = om.user_id").
		Where(squirrel.And{
			squirrel.Eq{"om.organization_id": organizationID},
			squirrel.Eq{"om.user_id": userID},
		})
	sqlStr, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	dest := new(models.OrganizationMemberWithUser)
	if err = r.repo.DB().GetContext(ctx, dest, sqlStr, args...); err != nil {
		return nil, chi_repository.WrapSQLError(err)
	}
	return dest, nil
}

func (r *repository) GetByMemberIDWithUser(ctx context.Context, organizationID, memberID string) (*models.OrganizationMemberWithUser, error) {
	columns := organizationMemberSelectColumns(
		"u.email as user_email",
		"u.first_name as user_first_name",
		"u.last_name as user_last_name",
	)
	query := r.repo.GetQueryBuilder().Select(columns...).
		From(fmt.Sprintf("%s AS om", TableName)).
		LeftJoin("users as u ON u.id = om.user_id").
		Where(squirrel.And{
			squirrel.Eq{"om.organization_id": organizationID},
			squirrel.Eq{"om.id": memberID},
		})
	sqlStr, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	dest := new(models.OrganizationMemberWithUser)
	if err = r.repo.DB().GetContext(ctx, dest, sqlStr, args...); err != nil {
		return nil, chi_repository.WrapSQLError(err)
	}
	return dest, nil
}

func (r *repository) GetByUserIDWithOrganizationAndRole(ctx context.Context, organizationID, userID string) (*models.OrganizationMemberWithOrganizationAndRole, error) {
	columns := organizationMemberSelectColumns(
		"o.name as organization_name",
		"r.name as role_name",
		"r.permissions as role_permissions",
	)
	query := r.repo.GetQueryBuilder().Select(columns...).
		From(fmt.Sprintf("%s AS om", TableName)).
		LeftJoin("organizations as o ON o.id = om.organization_id").
		LeftJoin("roles as r ON r.id = om.role_id").
		Where(squirrel.And{
			squirrel.Eq{"om.organization_id": organizationID},
			squirrel.Eq{"om.user_id": userID},
		})
	sqlStr, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	dest := new(models.OrganizationMemberWithOrganizationAndRole)
	if err = r.repo.DB().GetContext(ctx, dest, sqlStr, args...); err != nil {
		return nil, chi_repository.WrapSQLError(err)
	}
	return dest, nil
}

func (r *repository) ListByUserID(ctx context.Context, userID string) (*[]models.OrganizationMemberWithOrganizationAndRole, error) {
	columns := organizationMemberSelectColumns(
		"o.name as organization_name",
		"r.name as role_name",
		"r.permissions as role_permissions",
	)
	query := r.repo.GetQueryBuilder().Select(columns...).
		From(fmt.Sprintf("%s AS om", TableName)).
		LeftJoin("organizations as o ON o.id = om.organization_id").
		LeftJoin("roles as r ON r.id = om.role_id").
		Where(squirrel.And{
			squirrel.Eq{"om.user_id": userID},
			squirrel.Expr("o.deleted_at IS NULL"),
		})
	sqlStr, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	dest := new([]models.OrganizationMemberWithOrganizationAndRole)
	if err = r.repo.DB().SelectContext(ctx, dest, sqlStr, args...); err != nil {
		return nil, chi_repository.WrapSQLError(err)
	}
	return dest, nil
}

func (r *repository) ListByOrganizationID(ctx context.Context, organizationID string) (*[]models.OrganizationMemberWithUser, error) {
	columns := organizationMemberSelectColumns(
		"u.email as user_email",
		"u.first_name as user_first_name",
		"u.last_name as user_last_name",
	)
	query := r.repo.GetQueryBuilder().Select(columns...).
		From(fmt.Sprintf("%s AS om", TableName)).
		LeftJoin("users as u ON u.id = om.user_id").
		Where(squirrel.Eq{"om.organization_id": organizationID})
	sqlStr, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	dest := new([]models.OrganizationMemberWithUser)
	if err = r.repo.DB().SelectContext(ctx, dest, sqlStr, args...); err != nil {
		return nil, chi_repository.WrapSQLError(err)
	}
	return dest, nil
}

func (r *repository) ListUserEmailsForRole(ctx context.Context, organizationID, roleID string) ([]string, error) {
	query := r.repo.GetQueryBuilder().Select("u.email").
		From(fmt.Sprintf("%s AS om", TableName)).
		InnerJoin("users AS u ON u.id = om.user_id").
		Where(squirrel.And{
			squirrel.Eq{"om.organization_id": organizationID},
			squirrel.Eq{"om.role_id": roleID},
		}).
		OrderBy("u.email")
	sqlStr, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}
	var emails []string
	if err := r.repo.DB().SelectContext(ctx, &emails, sqlStr, args...); err != nil {
		return nil, chi_repository.WrapSQLError(err)
	}
	sort.Strings(emails)
	return emails, nil
}
