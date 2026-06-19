package models

import (
	"github.com/google/uuid"
	chi_types "github.com/yca-software/2chi-go-types"
)

type Team struct {
	chi_types.ModelBase
	OrganizationID uuid.UUID `json:"organizationId" db:"organization_id"`

	Name        string `db:"name" json:"name"`
	Description string `db:"description" json:"description"`
}

type TeamMember struct {
	chi_types.ModelBase
	OrganizationID uuid.UUID `json:"organizationId" db:"organization_id"`
	UserID         uuid.UUID `json:"userId" db:"user_id"`
	TeamID         uuid.UUID `json:"teamId" db:"team_id"`
}

type TeamMemberWithTeam struct {
	TeamMember
	TeamName string `db:"team_name" json:"teamName"`
}

type TeamMemberWithUser struct {
	TeamMember
	UserEmail     string `db:"user_email" json:"userEmail"`
	UserFirstName string `db:"user_first_name" json:"userFirstName"`
	UserLastName  string `db:"user_last_name" json:"userLastName"`
}
