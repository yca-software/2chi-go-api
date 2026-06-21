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
	UserLegalDocumentAcceptancesTableName = "user_legal_document_acceptances"
)

var (
	UserLegalDocumentAcceptancesColumns = []string{"id", "created_at", "updated_at", "user_id", "document_type", "document_version"}
)

type UserLegalDocumentAcceptanceRepository interface {
	WithTx(tx chi_repository.Tx) UserLegalDocumentAcceptanceRepository

	CreateUserLegalDocumentAcceptance(ctx context.Context, acceptance *models.UserLegalDocumentAcceptance) error
	ListUserLegalDocumentAcceptancesByUserID(ctx context.Context, userID string) (*[]models.UserLegalDocumentAcceptance, error)
	GetLatestUserLegalDocumentAcceptanceByUserIDAndDocumentType(ctx context.Context, userID, documentType string) (*models.UserLegalDocumentAcceptance, error)
	ListLatestUserLegalDocumentAcceptancesByUserID(ctx context.Context, userID string) (*[]models.UserLegalDocumentAcceptance, error)
}

type userLegalDocumentAcceptanceRepository struct {
	legalDocumentAcceptancesRepo chi_repository.Repository[models.UserLegalDocumentAcceptance]
}

func NewUserLegalDocumentAcceptanceRepository(db *sqlx.DB, metricsHook chi_observer.QueryMetricsHook) UserLegalDocumentAcceptanceRepository {
	return &userLegalDocumentAcceptanceRepository{
		legalDocumentAcceptancesRepo: chi_repository.NewRepository[models.UserLegalDocumentAcceptance](db, UserLegalDocumentAcceptancesTableName, UserLegalDocumentAcceptancesColumns, metricsHook),
	}
}

func (r *userLegalDocumentAcceptanceRepository) WithTx(tx chi_repository.Tx) UserLegalDocumentAcceptanceRepository {
	return &userLegalDocumentAcceptanceRepository{
		legalDocumentAcceptancesRepo: r.legalDocumentAcceptancesRepo.WithTx(tx),
	}
}

func (r *userLegalDocumentAcceptanceRepository) CreateUserLegalDocumentAcceptance(ctx context.Context, acceptance *models.UserLegalDocumentAcceptance) error {
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

func (r *userLegalDocumentAcceptanceRepository) ListUserLegalDocumentAcceptancesByUserID(ctx context.Context, userID string) (*[]models.UserLegalDocumentAcceptance, error) {
	return r.legalDocumentAcceptancesRepo.Select(ctx, squirrel.Eq{"user_id": userID}, nil, "document_type ASC, created_at DESC")
}

func (r *userLegalDocumentAcceptanceRepository) GetLatestUserLegalDocumentAcceptanceByUserIDAndDocumentType(ctx context.Context, userID, documentType string) (*models.UserLegalDocumentAcceptance, error) {
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

func (r *userLegalDocumentAcceptanceRepository) ListLatestUserLegalDocumentAcceptancesByUserID(ctx context.Context, userID string) (*[]models.UserLegalDocumentAcceptance, error) {
	columns := strings.Join(UserLegalDocumentAcceptancesColumns, ", ")
	sqlStr := fmt.Sprintf(`
SELECT DISTINCT ON (document_type) %s
FROM %s
WHERE user_id = $1
ORDER BY document_type, created_at DESC`, columns, UserLegalDocumentAcceptancesTableName)

	var results []models.UserLegalDocumentAcceptance
	if err := r.legalDocumentAcceptancesRepo.DB().SelectContext(ctx, &results, sqlStr, userID); err != nil {
		return nil, chi_repository.WrapSQLError(err)
	}
	return &results, nil
}
