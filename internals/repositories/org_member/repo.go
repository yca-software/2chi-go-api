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
	OrganizationMembersTableName = "organization_members"
)

var (
	OrganizationMembersColumns = []string{
		"id", "created_at", "updated_at", "organization_id", "user_id", "role_id",
	}
)

type OrganizationMembersRepository interface {
	WithTx(tx chi_repository.Tx) OrganizationMembersRepository

	CreateOrganizationMember(ctx context.Context, member *models.OrganizationMember) error
	UpdateOrganizationMember(ctx context.Context, member *models.OrganizationMember) error
	DeleteOrganizationMember(ctx context.Context, organizationID, userID string) error
	DeleteOrganizationMemberByMembershipID(ctx context.Context, organizationID, memberID string) error

	GetOrganizationMemberByID(ctx context.Context, organizationID, userID string) (*models.OrganizationMember, error)
	GetOrganizationMemberByMembershipID(ctx context.Context, organizationID, memberID string) (*models.OrganizationMember, error)
	GetOrganizationMemberByIDWithUser(ctx context.Context, organizationID, userID string) (*models.OrganizationMemberWithUser, error)
	GetOrganizationMemberByMembershipIDWithUser(ctx context.Context, organizationID, memberID string) (*models.OrganizationMemberWithUser, error)
	GetOrganizationMemberByIDWithOrganizationAndRole(ctx context.Context, organizationID, userID string) (*models.OrganizationMemberWithOrganizationAndRole, error)
	GetOrganizationMemberByUserIDAndOrganizationID(ctx context.Context, userID, organizationID string) (*models.OrganizationMember, error)

	ListByUserID(ctx context.Context, userID string) (*[]models.OrganizationMemberWithOrganization, error)
	ListByUserIDWithRole(ctx context.Context, userID string) (*[]models.OrganizationMemberWithOrganizationAndRole, error)
	ListByOrganizationID(ctx context.Context, organizationID string) (*[]models.OrganizationMemberWithUser, error)
	ListUserEmailsForRole(ctx context.Context, organizationID, roleID string) ([]string, error)
}

type organizationMembersRepository struct {
	organizationMembersRepo chi_repository.Repository[models.OrganizationMember]
}

func NewOrganizationMembersRepository(db *sqlx.DB, metricsHook chi_observer.QueryMetricsHook) OrganizationMembersRepository {
	return &organizationMembersRepository{
		organizationMembersRepo: chi_repository.NewRepository[models.OrganizationMember](db, OrganizationMembersTableName, OrganizationMembersColumns, metricsHook),
	}
}

func (r *organizationMembersRepository) WithTx(tx chi_repository.Tx) OrganizationMembersRepository {
	return &organizationMembersRepository{
		organizationMembersRepo: r.organizationMembersRepo.WithTx(tx),
	}
}

func organizationMemberSelectColumns(extra ...string) []string {
	columns := make([]string, 0, len(OrganizationMembersColumns)+len(extra))
	for _, column := range OrganizationMembersColumns {
		columns = append(columns, fmt.Sprintf("om.%s", column))
	}
	return append(columns, extra...)
}

func (r *organizationMembersRepository) CreateOrganizationMember(ctx context.Context, member *models.OrganizationMember) error {
	now := time.Now()
	return r.organizationMembersRepo.Create(ctx, map[string]any{
		"id":              member.ID,
		"created_at":      now,
		"updated_at":      now,
		"organization_id": member.OrganizationID,
		"user_id":         member.UserID,
		"role_id":         member.RoleID,
	})
}

func (r *organizationMembersRepository) UpdateOrganizationMember(ctx context.Context, member *models.OrganizationMember) error {
	return r.organizationMembersRepo.Update(ctx, squirrel.And{
		squirrel.Eq{"id": member.ID},
		squirrel.Eq{"organization_id": member.OrganizationID},
	}, map[string]any{
		"role_id":    member.RoleID,
		"updated_at": time.Now(),
	})
}

func (r *organizationMembersRepository) DeleteOrganizationMember(ctx context.Context, organizationID, userID string) error {
	return r.organizationMembersRepo.Delete(ctx, squirrel.And{
		squirrel.Eq{"organization_id": organizationID},
		squirrel.Eq{"user_id": userID},
	})
}

func (r *organizationMembersRepository) DeleteOrganizationMemberByMembershipID(ctx context.Context, organizationID, memberID string) error {
	return r.organizationMembersRepo.Delete(ctx, squirrel.And{
		squirrel.Eq{"organization_id": organizationID},
		squirrel.Eq{"id": memberID},
	})
}

func (r *organizationMembersRepository) GetOrganizationMemberByID(ctx context.Context, organizationID, userID string) (*models.OrganizationMember, error) {
	return r.organizationMembersRepo.Get(ctx, squirrel.And{
		squirrel.Eq{"organization_id": organizationID},
		squirrel.Eq{"user_id": userID},
	}, nil)
}

func (r *organizationMembersRepository) GetOrganizationMemberByMembershipID(ctx context.Context, organizationID, memberID string) (*models.OrganizationMember, error) {
	return r.organizationMembersRepo.Get(ctx, squirrel.And{
		squirrel.Eq{"organization_id": organizationID},
		squirrel.Eq{"id": memberID},
	}, nil)
}

func (r *organizationMembersRepository) GetOrganizationMemberByIDWithUser(ctx context.Context, organizationID, userID string) (*models.OrganizationMemberWithUser, error) {
	columns := organizationMemberSelectColumns(
		"u.email as user_email",
		"u.first_name as user_first_name",
		"u.last_name as user_last_name",
	)
	query := r.organizationMembersRepo.GetQueryBuilder().Select(columns...).
		From(fmt.Sprintf("%s AS om", OrganizationMembersTableName)).
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
	if err = r.organizationMembersRepo.DB().GetContext(ctx, dest, sqlStr, args...); err != nil {
		return nil, chi_repository.WrapSQLError(err)
	}
	return dest, nil
}

func (r *organizationMembersRepository) GetOrganizationMemberByMembershipIDWithUser(ctx context.Context, organizationID, memberID string) (*models.OrganizationMemberWithUser, error) {
	columns := organizationMemberSelectColumns(
		"u.email as user_email",
		"u.first_name as user_first_name",
		"u.last_name as user_last_name",
	)
	query := r.organizationMembersRepo.GetQueryBuilder().Select(columns...).
		From(fmt.Sprintf("%s AS om", OrganizationMembersTableName)).
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
	if err = r.organizationMembersRepo.DB().GetContext(ctx, dest, sqlStr, args...); err != nil {
		return nil, chi_repository.WrapSQLError(err)
	}
	return dest, nil
}

func (r *organizationMembersRepository) GetOrganizationMemberByIDWithOrganizationAndRole(ctx context.Context, organizationID, userID string) (*models.OrganizationMemberWithOrganizationAndRole, error) {
	columns := organizationMemberSelectColumns(
		"o.name as organization_name",
		"r.name as role_name",
		"r.permissions as role_permissions",
	)
	query := r.organizationMembersRepo.GetQueryBuilder().Select(columns...).
		From(fmt.Sprintf("%s AS om", OrganizationMembersTableName)).
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
	if err = r.organizationMembersRepo.DB().GetContext(ctx, dest, sqlStr, args...); err != nil {
		return nil, chi_repository.WrapSQLError(err)
	}
	return dest, nil
}

func (r *organizationMembersRepository) GetOrganizationMemberByUserIDAndOrganizationID(ctx context.Context, userID, organizationID string) (*models.OrganizationMember, error) {
	return r.organizationMembersRepo.Get(ctx, squirrel.And{
		squirrel.Eq{"user_id": userID},
		squirrel.Eq{"organization_id": organizationID},
	}, nil)
}

func (r *organizationMembersRepository) ListByUserID(ctx context.Context, userID string) (*[]models.OrganizationMemberWithOrganization, error) {
	columns := organizationMemberSelectColumns("o.name as organization_name")
	query := r.organizationMembersRepo.GetQueryBuilder().Select(columns...).
		From(fmt.Sprintf("%s AS om", OrganizationMembersTableName)).
		LeftJoin("organizations as o ON o.id = om.organization_id").
		Where(squirrel.And{
			squirrel.Eq{"om.user_id": userID},
			squirrel.Expr("o.deleted_at IS NULL"),
		})
	sqlStr, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	dest := new([]models.OrganizationMemberWithOrganization)
	if err = r.organizationMembersRepo.DB().SelectContext(ctx, dest, sqlStr, args...); err != nil {
		return nil, chi_repository.WrapSQLError(err)
	}
	return dest, nil
}

func (r *organizationMembersRepository) ListByUserIDWithRole(ctx context.Context, userID string) (*[]models.OrganizationMemberWithOrganizationAndRole, error) {
	columns := organizationMemberSelectColumns(
		"o.name as organization_name",
		"r.name as role_name",
		"r.permissions as role_permissions",
	)
	query := r.organizationMembersRepo.GetQueryBuilder().Select(columns...).
		From(fmt.Sprintf("%s AS om", OrganizationMembersTableName)).
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
	if err = r.organizationMembersRepo.DB().SelectContext(ctx, dest, sqlStr, args...); err != nil {
		return nil, chi_repository.WrapSQLError(err)
	}
	return dest, nil
}

func (r *organizationMembersRepository) ListByOrganizationID(ctx context.Context, organizationID string) (*[]models.OrganizationMemberWithUser, error) {
	columns := organizationMemberSelectColumns(
		"u.email as user_email",
		"u.first_name as user_first_name",
		"u.last_name as user_last_name",
	)
	query := r.organizationMembersRepo.GetQueryBuilder().Select(columns...).
		From(fmt.Sprintf("%s AS om", OrganizationMembersTableName)).
		LeftJoin("users as u ON u.id = om.user_id").
		Where(squirrel.Eq{"om.organization_id": organizationID})
	sqlStr, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	dest := new([]models.OrganizationMemberWithUser)
	if err = r.organizationMembersRepo.DB().SelectContext(ctx, dest, sqlStr, args...); err != nil {
		return nil, chi_repository.WrapSQLError(err)
	}
	return dest, nil
}

func (r *organizationMembersRepository) ListUserEmailsForRole(ctx context.Context, organizationID, roleID string) ([]string, error) {
	query := r.organizationMembersRepo.GetQueryBuilder().Select("u.email").
		From(fmt.Sprintf("%s AS om", OrganizationMembersTableName)).
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
	if err := r.organizationMembersRepo.DB().SelectContext(ctx, &emails, sqlStr, args...); err != nil {
		return nil, chi_repository.WrapSQLError(err)
	}
	sort.Strings(emails)
	return emails, nil
}
