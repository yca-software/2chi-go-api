package audit_log_repository

import (
	"context"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"

	"github.com/yca-software/2chi-go-api/internals/models"
	chi_observer "github.com/yca-software/2chi-go-observer"
	chi_repository "github.com/yca-software/2chi-go-repository"
)

const (
	TableName = "audit_logs"
)

var (
	Columns = []string{
		"id", "created_at", "organization_id",
		"actor_id", "actor_info", "impersonated_by_id", "impersonated_by_email",
		"action", "resource_type", "resource_id", "resource_name", "data",
	}
)

type AuditLogFilters struct {
	StartDate    *time.Time
	EndDate      *time.Time
	Action       *string
	ResourceType *string
	Search       *string
}

type Repository interface {
	WithTx(tx chi_repository.Tx) Repository

	Create(ctx context.Context, log *models.AuditLog) error
	ListByOrganizationID(ctx context.Context, organizationID string, filters *AuditLogFilters, limit, offset int) (*[]models.AuditLog, error)
}

type repository struct {
	repo chi_repository.Repository[models.AuditLog]
}

func NewRepository(db *sqlx.DB, metricsHook chi_observer.QueryMetricsHook) Repository {
	return &repository{
		repo: chi_repository.NewRepository[models.AuditLog](db, TableName, Columns, metricsHook),
	}
}

func (r *repository) WithTx(tx chi_repository.Tx) Repository {
	return &repository{repo: r.repo.WithTx(tx)}
}

func applyAuditLogFilters(condition squirrel.And, filters *AuditLogFilters) squirrel.And {
	if filters == nil {
		return condition
	}
	if filters.StartDate != nil {
		condition = append(condition, squirrel.GtOrEq{"created_at": filters.StartDate})
	}
	if filters.EndDate != nil {
		condition = append(condition, squirrel.Lt{"created_at": filters.EndDate})
	}
	if filters.Action != nil && *filters.Action != "" {
		condition = append(condition, squirrel.Eq{"action": *filters.Action})
	}
	if filters.ResourceType != nil && *filters.ResourceType != "" {
		condition = append(condition, squirrel.Eq{"resource_type": *filters.ResourceType})
	}
	if filters.Search != nil && *filters.Search != "" {
		pattern := auditLogSearchPattern(*filters.Search)
		condition = append(condition, squirrel.Or{
			squirrel.ILike{"actor_info": pattern},
			squirrel.ILike{"resource_name": pattern},
			squirrel.Expr("resource_id::text ILIKE ?", pattern),
		})
	}
	return condition
}

func auditLogSearchPattern(search string) string {
	search = strings.ReplaceAll(search, `\`, `\\`)
	search = strings.ReplaceAll(search, `%`, `\%`)
	search = strings.ReplaceAll(search, `_`, `\_`)
	return "%" + search + "%"
}

func (r *repository) Create(ctx context.Context, log *models.AuditLog) error {
	return r.repo.Create(ctx, map[string]any{
		"id":                    log.ID,
		"created_at":            time.Now(),
		"organization_id":       log.OrganizationID,
		"actor_id":              log.ActorID,
		"actor_info":            log.ActorInfo,
		"impersonated_by_id":    log.ImpersonatedByID,
		"impersonated_by_email": log.ImpersonatedByEmail,
		"action":                log.Action,
		"resource_type":         log.ResourceType,
		"resource_id":           log.ResourceID,
		"resource_name":         log.ResourceName,
		"data":                  log.Data,
	})
}

func (r *repository) ListByOrganizationID(ctx context.Context, organizationID string, filters *AuditLogFilters, limit, offset int) (*[]models.AuditLog, error) {
	condition := applyAuditLogFilters(squirrel.And{squirrel.Eq{"organization_id": organizationID}}, filters)
	return r.repo.PaginatedSelect(ctx, condition, nil, "created_at DESC", uint64(limit), uint64(offset))
}
