package team_service

type CreateTeamRequest struct {
	OrganizationID string `json:"-" validate:"required,uuid"`
	Name           string `json:"name" validate:"required,min=1,max=255"`
	Description    string `json:"description"`
}

type UpdateTeamRequest struct {
	OrganizationID string `json:"-" validate:"required,uuid"`
	TeamID         string `json:"-" validate:"required,uuid"`
	Name           string `json:"name" validate:"required,min=1,max=255"`
	Description    string `json:"description"`
}

type DeleteTeamRequest struct {
	OrganizationID string `json:"-" validate:"required,uuid"`
	TeamID         string `json:"-" validate:"required,uuid"`
}

type ListTeamsRequest struct {
	OrganizationID string `json:"-" validate:"required,uuid"`
}

type AddTeamMemberRequest struct {
	OrganizationID string `json:"-" validate:"required,uuid"`
	TeamID         string `json:"-" validate:"required,uuid"`
	UserID         string `json:"userId" validate:"required,uuid"`
}

type RemoveTeamMemberRequest struct {
	OrganizationID string `json:"-" validate:"required,uuid"`
	TeamID         string `json:"-" validate:"required,uuid"`
	MemberID       string `json:"-" validate:"required,uuid"`
}

type ListTeamMembersRequest struct {
	OrganizationID string `json:"-" validate:"required,uuid"`
	TeamID         string `json:"-" validate:"required,uuid"`
}
