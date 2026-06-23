package user_repository

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
	TableName = "users"
)

var (
	Columns = []string{
		"id", "created_at", "updated_at", "deleted_at",
		"first_name", "last_name", "language", "email", "email_verified_at",
		"password", "avatar_url",
	}
)

type Repository interface {
	WithTx(tx chi_repository.Tx) Repository

	Create(ctx context.Context, user *models.User) error
	Update(ctx context.Context, user *models.User) error
	Archive(ctx context.Context, user *models.User) error
	Restore(ctx context.Context, userID string) error
	CleanupArchived(ctx context.Context) error

	GetByID(ctx context.Context, id string) (*models.User, error)
	GetByIDIncludeArchived(ctx context.Context, id string) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	Search(ctx context.Context, searchPhrase string, filter chi_archive.ArchiveFilter, limit, offset int) (*[]models.User, error)
}

type repository struct {
	repo chi_repository.Repository[models.User]
}

func NewRepository(db *sqlx.DB, metricsHook chi_observer.QueryMetricsHook) Repository {
	return &repository{
		repo: chi_repository.NewRepository[models.User](db, TableName, Columns, metricsHook),
	}
}

func (r *repository) WithTx(tx chi_repository.Tx) Repository {
	return &repository{
		repo: r.repo.WithTx(tx),
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

func (r *repository) Create(ctx context.Context, user *models.User) error {
	now := time.Now()
	return r.repo.Create(ctx, map[string]any{
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

func (r *repository) Update(ctx context.Context, user *models.User) error {
	return r.repo.Update(ctx, squirrel.Eq{"id": user.ID, "deleted_at": nil}, map[string]any{
		"first_name":        user.FirstName,
		"last_name":         user.LastName,
		"language":          user.Language,
		"avatar_url":        user.AvatarURL,
		"email_verified_at": user.EmailVerifiedAt,
		"password":          user.Password,
		"updated_at":        time.Now(),
	})
}

func (r *repository) Archive(ctx context.Context, user *models.User) error {
	return r.repo.Update(ctx, squirrel.Eq{"id": user.ID, "deleted_at": nil}, map[string]any{
		"deleted_at": time.Now(),
		"updated_at": time.Now(),
	})
}

func (r *repository) Restore(ctx context.Context, userID string) error {
	return r.repo.Update(ctx, squirrel.And{
		squirrel.Eq{"id": userID},
		squirrel.NotEq{"deleted_at": nil},
	}, map[string]any{
		"deleted_at": nil,
		"updated_at": time.Now(),
	})
}

func (r *repository) CleanupArchived(ctx context.Context) error {
	threshold := time.Now().Add(-chi_archive.ArchivedDataRetentionPeriod)
	return r.repo.Delete(ctx, squirrel.And{
		squirrel.NotEq{"deleted_at": nil},
		squirrel.LtOrEq{"deleted_at": threshold},
	})
}

func (r *repository) GetByID(ctx context.Context, id string) (*models.User, error) {
	return r.repo.Get(ctx, squirrel.Eq{"id": id, "deleted_at": nil}, nil)
}

func (r *repository) GetByIDIncludeArchived(ctx context.Context, id string) (*models.User, error) {
	return r.repo.Get(ctx, squirrel.Eq{"id": id}, nil)
}

func (r *repository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	return r.repo.Get(ctx, squirrel.Eq{"email": email, "deleted_at": nil}, nil)
}

func (r *repository) Search(ctx context.Context, searchPhrase string, filter chi_archive.ArchiveFilter, limit, offset int) (*[]models.User, error) {
	filter, err := chi_archive.NormalizeArchiveFilter(filter)
	if err != nil {
		return nil, err
	}
	condition, sort := searchUsersCondition(searchPhrase, filter)
	return r.repo.PaginatedSelect(ctx, condition, nil, sort, uint64(limit), uint64(offset))
}
