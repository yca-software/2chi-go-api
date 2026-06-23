package api_key_service

import (
	"time"

	"github.com/yca-software/2chi-go-api/internals/models"
)

type CreateRequest struct {
	OrganizationID string                 `json:"organizationId" validate:"required,uuid"`
	Name           string                 `json:"name" validate:"required,min=1,max=255"`
	Permissions    models.RolePermissions `json:"permissions" validate:"required,min=1"`
	ExpiresAt      time.Time              `json:"expiresAt" validate:"required"`
}

type CreateResponse struct {
	APIKey *models.APIKey `json:"apiKey"`
	Secret string         `json:"secret"`
}

type UpdateRequest struct {
	OrganizationID string                 `json:"-" validate:"required,uuid"`
	APIKeyID       string                 `json:"-" validate:"required,uuid"`
	Name           string                 `json:"name" validate:"required,min=1,max=255"`
	Permissions    models.RolePermissions `json:"permissions" validate:"required,min=1"`
}

type DeleteRequest struct {
	OrganizationID string `json:"-" validate:"required,uuid"`
	APIKeyID       string `json:"-" validate:"required,uuid"`
}

type ListRequest struct {
	OrganizationID string `json:"-" validate:"required,uuid"`
}
