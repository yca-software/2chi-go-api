package models

import (
	"time"

	"github.com/google/uuid"
	chi_types "github.com/yca-software/2chi-go-types"
)

type OrganizationMember struct {
	chi_types.ModelBase
	OrganizationID uuid.UUID `json:"organizationId" db:"organization_id"`
	UserID         uuid.UUID `json:"userId" db:"user_id"`
	RoleID         uuid.UUID `json:"roleId" db:"role_id"`
}

type OrganizationMemberWithOrganization struct {
	OrganizationMember
	OrganizationName string `db:"organization_name" json:"organizationName"`
}

type OrganizationMemberWithOrganizationAndRole struct {
	OrganizationMemberWithOrganization
	RoleName        string          `db:"role_name" json:"roleName"`
	RolePermissions RolePermissions `db:"role_permissions" json:"rolePermissions"`
}

type OrganizationMemberWithUser struct {
	OrganizationMember
	UserEmail     string `db:"user_email" json:"userEmail"`
	UserFirstName string `db:"user_first_name" json:"userFirstName"`
	UserLastName  string `db:"user_last_name" json:"userLastName"`
}

type Invitation struct {
	chi_types.ModelBase
	OrganizationID uuid.UUID `json:"organizationId" db:"organization_id"`
	RoleID         uuid.UUID `json:"roleId" db:"role_id"`

	ExpiresAt  time.Time  `db:"expires_at" json:"expiresAt"`
	AcceptedAt *time.Time `db:"accepted_at" json:"acceptedAt"`
	RevokedAt  *time.Time `db:"revoked_at" json:"revokedAt"`

	Email string `db:"email" json:"email"`

	InvitedByID    uuid.NullUUID `db:"invited_by_id" json:"invitedById"`
	InvitedByEmail string        `db:"invited_by_email" json:"invitedByEmail"`
	TokenHash      string        `db:"token_hash" json:"-"`
}
