package handlers

import (
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	audit_log_repository "github.com/yca-software/2chi-go-api/internals/domains/core/repositories/audit/audit_log"
	chi_error "github.com/yca-software/2chi-go-error"
)

func parseAuditLogFilters(c echo.Context) (*audit_log_repository.AuditLogFilters, error) {
	startDate := c.QueryParam("startDate")
	endDate := c.QueryParam("endDate")
	action := strings.TrimSpace(c.QueryParam("action"))
	resourceType := strings.TrimSpace(c.QueryParam("resourceType"))
	search := strings.TrimSpace(c.QueryParam("search"))

	if startDate == "" && endDate == "" && action == "" && resourceType == "" && search == "" {
		return nil, nil
	}

	f := &audit_log_repository.AuditLogFilters{}
	if startDate != "" {
		parsedStart, parseErr := time.Parse(time.RFC3339, startDate)
		if parseErr != nil {
			return nil, chi_error.NewUnprocessableEntityError(parseErr, "", map[string]any{
				"startDate": map[string]any{"format": "RFC3339"},
			})
		}
		f.StartDate = &parsedStart
	}
	if endDate != "" {
		parsedEnd, parseErr := time.Parse(time.RFC3339, endDate)
		if parseErr != nil {
			return nil, chi_error.NewUnprocessableEntityError(parseErr, "", map[string]any{
				"endDate": map[string]any{"format": "RFC3339"},
			})
		}
		f.EndDate = &parsedEnd
	}
	if action != "" {
		f.Action = &action
	}
	if resourceType != "" {
		f.ResourceType = &resourceType
	}
	if search != "" {
		f.Search = &search
	}
	return f, nil
}
