package admin_access_repository

import (
	"context"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"

	"github.com/yca-software/2chi-go-api/internals/models"
	chi_observer "github.com/yca-software/2chi-go-observer"
	chi_repository "github.com/yca-software/2chi-go-repository"
)

const (
	AdminAccessTableName = "admin_access"
)

var (
	AdminAccessColumns = []string{"user_id", "created_at"}
)

type AdminAccessRepository interface {
	WithTx(tx chi_repository.Tx) AdminAccessRepository

	GetAdminAccessByUserID(ctx context.Context, userID string) (*models.AdminAccess, error)
	DeleteAdminAccessByUserID(ctx context.Context, userID string) error
}

type adminAccessRepository struct {
	adminAccessRepo chi_repository.Repository[models.AdminAccess]
}

func NewAdminAccessRepository(db *sqlx.DB, metricsHook chi_observer.QueryMetricsHook) AdminAccessRepository {
	return &adminAccessRepository{
		adminAccessRepo: chi_repository.NewRepository[models.AdminAccess](db, AdminAccessTableName, AdminAccessColumns, metricsHook),
	}
}

func (r *adminAccessRepository) WithTx(tx chi_repository.Tx) AdminAccessRepository {
	return &adminAccessRepository{
		adminAccessRepo: r.adminAccessRepo.WithTx(tx),
	}
}

func (r *adminAccessRepository) GetAdminAccessByUserID(ctx context.Context, userID string) (*models.AdminAccess, error) {
	return r.adminAccessRepo.Get(ctx, squirrel.Eq{"user_id": userID}, nil)
}

func (r *adminAccessRepository) DeleteAdminAccessByUserID(ctx context.Context, userID string) error {
	return r.adminAccessRepo.Delete(ctx, squirrel.Eq{"user_id": userID})
}
