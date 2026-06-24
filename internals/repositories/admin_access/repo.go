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
	TableName = "admin_access"
)

var (
	Columns = []string{"user_id", "created_at"}
)

type Repository interface {
	WithTx(tx chi_repository.Tx) Repository

	GetByUserID(ctx context.Context, userID string) (*models.AdminAccess, error)
}

type repository struct {
	repo chi_repository.Repository[models.AdminAccess]
}

func NewRepository(db *sqlx.DB, metricsHook chi_observer.QueryMetricsHook) Repository {
	return &repository{
		repo: chi_repository.NewRepository[models.AdminAccess](db, TableName, Columns, metricsHook),
	}
}

func (r *repository) WithTx(tx chi_repository.Tx) Repository {
	return &repository{
		repo: r.repo.WithTx(tx),
	}
}

func (r *repository) GetByUserID(ctx context.Context, userID string) (*models.AdminAccess, error) {
	return r.repo.Get(ctx, squirrel.Eq{"user_id": userID}, nil)
}
