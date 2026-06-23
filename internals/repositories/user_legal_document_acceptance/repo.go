package user_legal_document_acceptance_repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"

	"github.com/yca-software/2chi-go-api/internals/models"
	chi_observer "github.com/yca-software/2chi-go-observer"
	chi_repository "github.com/yca-software/2chi-go-repository"
)

const (
	TableName = "user_legal_document_acceptances"
)

var (
	Columns = []string{"id", "created_at", "updated_at", "user_id", "document_type", "document_version"}
)

type Repository interface {
	WithTx(tx chi_repository.Tx) Repository

	Create(ctx context.Context, acceptance *models.UserLegalDocumentAcceptance) error
	ListByUserID(ctx context.Context, userID string) (*[]models.UserLegalDocumentAcceptance, error)
	GetLatestByUserIDAndDocumentType(ctx context.Context, userID, documentType string) (*models.UserLegalDocumentAcceptance, error)
	ListLatestByUserID(ctx context.Context, userID string) (*[]models.UserLegalDocumentAcceptance, error)
}

type repository struct {
	legalDocumentAcceptancesRepo chi_repository.Repository[models.UserLegalDocumentAcceptance]
}

func NewRepository(db *sqlx.DB, metricsHook chi_observer.QueryMetricsHook) Repository {
	return &repository{
		legalDocumentAcceptancesRepo: chi_repository.NewRepository[models.UserLegalDocumentAcceptance](db, TableName, Columns, metricsHook),
	}
}

func (r *repository) WithTx(tx chi_repository.Tx) Repository {
	return &repository{
		legalDocumentAcceptancesRepo: r.legalDocumentAcceptancesRepo.WithTx(tx),
	}
}

func (r *repository) Create(ctx context.Context, acceptance *models.UserLegalDocumentAcceptance) error {
	now := time.Now()
	return r.legalDocumentAcceptancesRepo.Create(ctx, map[string]any{
		"id":               acceptance.ID,
		"created_at":       now,
		"updated_at":       now,
		"user_id":          acceptance.UserID,
		"document_type":    acceptance.DocumentType,
		"document_version": acceptance.DocumentVersion,
	})
}

func (r *repository) ListByUserID(ctx context.Context, userID string) (*[]models.UserLegalDocumentAcceptance, error) {
	return r.legalDocumentAcceptancesRepo.Select(ctx, squirrel.Eq{"user_id": userID}, nil, "document_type ASC, created_at DESC")
}

func (r *repository) GetLatestByUserIDAndDocumentType(ctx context.Context, userID, documentType string) (*models.UserLegalDocumentAcceptance, error) {
	rows, err := r.legalDocumentAcceptancesRepo.PaginatedSelect(ctx, squirrel.Eq{
		"user_id":       userID,
		"document_type": documentType,
	}, nil, "created_at DESC", 1, 0)
	if err != nil {
		return nil, err
	}
	if rows == nil || len(*rows) == 0 {
		return nil, chi_repository.ErrNotFoundNoRowsAffected()
	}
	return &(*rows)[0], nil
}

func (r *repository) ListLatestByUserID(ctx context.Context, userID string) (*[]models.UserLegalDocumentAcceptance, error) {
	columns := strings.Join(Columns, ", ")
	sqlStr := fmt.Sprintf(`
SELECT DISTINCT ON (document_type) %s
FROM %s
WHERE user_id = $1
ORDER BY document_type, created_at DESC`, columns, TableName)

	var results []models.UserLegalDocumentAcceptance
	if err := r.legalDocumentAcceptancesRepo.DB().SelectContext(ctx, &results, sqlStr, userID); err != nil {
		return nil, chi_repository.WrapSQLError(err)
	}
	return &results, nil
}
