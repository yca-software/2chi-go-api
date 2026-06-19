package invitation_service

import "github.com/yca-software/2chi-go-api/internals/domains/core/models"

type CreateInvitationRequest struct {
	OrganizationID string `json:"-" validate:"required,uuid"`
	Email          string `json:"email" validate:"required,email"`
	RoleID         string `json:"roleId" validate:"required,uuid"`
	InvitedByID    string `json:"-" validate:"required,uuid"`
	InvitedByEmail string `json:"-" validate:"required,email"`
	Language       string `json:"-" validate:"required"`
}

type CreateInvitationResponse struct {
	Invitation *models.Invitation                 `json:"invitation,omitempty"`
	Member     *models.OrganizationMemberWithUser `json:"member,omitempty"`
}

type RevokeInvitationRequest struct {
	OrganizationID string `json:"-" validate:"required,uuid"`
	InvitationID   string `json:"-" validate:"required,uuid"`
}

type ListInvitationsRequest struct {
	OrganizationID string `json:"-" validate:"required,uuid"`
}
