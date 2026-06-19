package user_repository

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
	UsersTableName = "users"
)

var (
	UsersColumns = []string{
		"id", "created_at", "updated_at", "deleted_at",
		"first_name", "last_name", "language", "email", "email_verified_at",
		"password", "avatar_url",
	}
)

type UsersRepository interface {
	WithTx(tx chi_repository.Tx) UsersRepository

	CreateUser(ctx context.Context, user *models.User) error
	UpdateUser(ctx context.Context, user *models.User) error
	ArchiveUser(ctx context.Context, user *models.User) error
	RestoreUser(ctx context.Context, userID string) error
	CleanupArchivedUsers(ctx context.Context) error

	GetUserByID(ctx context.Context, id string) (*models.User, error)
	GetUserByIDIncludeArchived(ctx context.Context, id string) (*models.User, error)
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	SearchUsers(ctx context.Context, searchPhrase string, filter chi_archive.ArchiveFilter, limit, offset int) (*[]models.User, error)
}

type usersRepository struct {
	usersRepo chi_repository.Repository[models.User]
}

func NewUsersRepository(db *sqlx.DB, metricsHook chi_observer.QueryMetricsHook) UsersRepository {
	return &usersRepository{
		usersRepo: chi_repository.NewRepository[models.User](db, UsersTableName, UsersColumns, metricsHook),
	}
}

func (r *usersRepository) WithTx(tx chi_repository.Tx) UsersRepository {
	return &usersRepository{
		usersRepo: r.usersRepo.WithTx(tx),
	}
}

func searchUsersCondition(searchPhrase string, filter chi_archive.ArchiveFilter) (squirrel.Sqlizer, string) {
	var archiveCond squirrel.Sqlizer = squirrel.Eq{"deleted_at": nil}
	sort := "created_at DESC"
	if filter == chi_archive.ArchiveFilterArchived {
		archiveCond = squirrel.NotEq{"deleted_at": nil}
		sort = "deleted_at DESC"
	}

	if searchPhrase == "" {
		return archiveCond, sort
	}

	return squirrel.And{
		archiveCond,
		squirrel.Or{
			squirrel.ILike{"email": "%" + searchPhrase + "%"},
			squirrel.ILike{"first_name": "%" + searchPhrase + "%"},
			squirrel.ILike{"last_name": "%" + searchPhrase + "%"},
		},
	}, sort
}

func (r *usersRepository) CreateUser(ctx context.Context, user *models.User) error {
	now := time.Now()
	return r.usersRepo.Create(ctx, map[string]any{
		"id":                user.ID,
		"created_at":        now,
		"updated_at":        now,
		"first_name":        user.FirstName,
		"last_name":         user.LastName,
		"language":          user.Language,
		"email":             user.Email,
		"email_verified_at": user.EmailVerifiedAt,
		"password":          user.Password,
		"avatar_url":        user.AvatarURL,
	})
}

func (r *usersRepository) UpdateUser(ctx context.Context, user *models.User) error {
	return r.usersRepo.Update(ctx, squirrel.Eq{"id": user.ID, "deleted_at": nil}, map[string]any{
		"first_name":        user.FirstName,
		"last_name":         user.LastName,
		"language":          user.Language,
		"avatar_url":        user.AvatarURL,
		"email_verified_at": user.EmailVerifiedAt,
		"password":          user.Password,
		"updated_at":        time.Now(),
	})
}

func (r *usersRepository) ArchiveUser(ctx context.Context, user *models.User) error {
	return r.usersRepo.Update(ctx, squirrel.Eq{"id": user.ID, "deleted_at": nil}, map[string]any{
		"deleted_at": time.Now(),
		"updated_at": time.Now(),
	})
}

func (r *usersRepository) RestoreUser(ctx context.Context, userID string) error {
	return r.usersRepo.Update(ctx, squirrel.And{
		squirrel.Eq{"id": userID},
		squirrel.NotEq{"deleted_at": nil},
	}, map[string]any{
		"deleted_at": nil,
		"updated_at": time.Now(),
	})
}

func (r *usersRepository) CleanupArchivedUsers(ctx context.Context) error {
	threshold := time.Now().Add(-chi_archive.ArchivedDataRetentionPeriod)
	return r.usersRepo.Delete(ctx, squirrel.And{
		squirrel.NotEq{"deleted_at": nil},
		squirrel.LtOrEq{"deleted_at": threshold},
	})
}

func (r *usersRepository) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	return r.usersRepo.Get(ctx, squirrel.Eq{"id": id, "deleted_at": nil}, nil)
}

func (r *usersRepository) GetUserByIDIncludeArchived(ctx context.Context, id string) (*models.User, error) {
	return r.usersRepo.Get(ctx, squirrel.Eq{"id": id}, nil)
}

func (r *usersRepository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	return r.usersRepo.Get(ctx, squirrel.Eq{"email": email, "deleted_at": nil}, nil)
}

func (r *usersRepository) SearchUsers(ctx context.Context, searchPhrase string, filter chi_archive.ArchiveFilter, limit, offset int) (*[]models.User, error) {
	filter, err := chi_archive.NormalizeArchiveFilter(filter)
	if err != nil {
		return nil, err
	}
	condition, sort := searchUsersCondition(searchPhrase, filter)
	return r.usersRepo.PaginatedSelect(ctx, condition, nil, sort, uint64(limit), uint64(offset))
}
