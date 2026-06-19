package api_key_service

import (
	"time"

	"github.com/yca-software/2chi-go-api/internals/domains/core/models"
)

type CreateAPIKeyRequest struct {
	OrganizationID string                 `json:"organizationId" validate:"required,uuid"`
	Name           string                 `json:"name" validate:"required,min=1,max=255"`
	Permissions    models.RolePermissions `json:"permissions" validate:"required,min=1"`
	ExpiresAt      time.Time              `json:"expiresAt" validate:"required"`
}

type CreateAPIKeyResponse struct {
	APIKey *models.APIKey `json:"apiKey"`
	Secret string         `json:"secret"`
}

type UpdateAPIKeyRequest struct {
	OrganizationID string                 `json:"-" validate:"required,uuid"`
	APIKeyID       string                 `json:"-" validate:"required,uuid"`
	Name           string                 `json:"name" validate:"required,min=1,max=255"`
	Permissions    models.RolePermissions `json:"permissions" validate:"required,min=1"`
}

type DeleteAPIKeyRequest struct {
	OrganizationID string `json:"-" validate:"required,uuid"`
	APIKeyID       string `json:"-" validate:"required,uuid"`
}

type ListAPIKeysRequest struct {
	OrganizationID string `json:"-" validate:"required,uuid"`
}
