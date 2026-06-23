package role_service

import "github.com/yca-software/2chi-go-api/internals/models"

type CreateRequest struct {
	OrganizationID string                 `json:"-" validate:"required,uuid"`
	Name           string                 `json:"name" validate:"required,min=1,max=255"`
	Description    string                 `json:"description"`
	Permissions    models.RolePermissions `json:"permissions" validate:"required,min=1"`
}

type UpdateRequest struct {
	OrganizationID string                 `json:"-" validate:"required,uuid"`
	RoleID         string                 `json:"-" validate:"required,uuid"`
	Name           string                 `json:"name" validate:"required,min=1,max=255"`
	Description    string                 `json:"description"`
	Permissions    models.RolePermissions `json:"permissions" validate:"required,min=1"`
}

type DeleteRequest struct {
	OrganizationID string `json:"-" validate:"required,uuid"`
	RoleID         string `json:"-" validate:"required,uuid"`
}

type ListRequest struct {
	OrganizationID string `json:"-" validate:"required,uuid"`
}
