package audit_service

import (
	"encoding/json"

	"github.com/yca-software/2chi-go-api/internals/models"
	audit_log_repository "github.com/yca-software/2chi-go-api/internals/repositories/audit_log"
	chi_types "github.com/yca-software/2chi-go-types"
)

type CreateAuditLogRequest struct {
	OrganizationID string           `json:"organizationId" validate:"required,uuid"`
	Action         string           `json:"action" validate:"required"`
	ResourceType   string           `json:"resourceType" validate:"required"`
	ResourceID     string           `json:"resourceId" validate:"required,uuid"`
	ResourceName   string           `json:"resourceName" validate:"required,max=512"`
	Data           *json.RawMessage `json:"data" validate:"omitempty"`
}

type ListAuditLogsForOrganizationRequest struct {
	OrganizationID string                                `json:"-" validate:"required,uuid"`
	Limit          int                                   `json:"-" validate:"required,min=1,max=100"`
	Offset         int                                   `json:"-" validate:"min=0"`
	Filters        *audit_log_repository.AuditLogFilters `json:"-" validate:"omitempty"`
}

type ListAuditLogsForOrganizationResponse chi_types.PaginatedListResponse[models.AuditLogPublic]
