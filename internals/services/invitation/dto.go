package invitation_service

import "github.com/yca-software/2chi-go-api/internals/models"

type CreateRequest struct {
	OrganizationID string `json:"-" validate:"required,uuid"`
	Email          string `json:"email" validate:"required,email"`
	RoleID         string `json:"roleId" validate:"required,uuid"`
	InvitedByID    string `json:"-" validate:"required,uuid"`
	InvitedByEmail string `json:"-" validate:"required,email"`
	Language       string `json:"-" validate:"required"`
}

type CreateResponse struct {
	Invitation *models.Invitation                 `json:"invitation,omitempty"`
	Member     *models.OrganizationMemberWithUser `json:"member,omitempty"`
}

type RevokeRequest struct {
	OrganizationID string `json:"-" validate:"required,uuid"`
	InvitationID   string `json:"-" validate:"required,uuid"`
}

type ListRequest struct {
	OrganizationID string `json:"-" validate:"required,uuid"`
}
