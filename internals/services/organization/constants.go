package organization_service

import (
	"github.com/yca-software/2chi-go-api/internals/constants"
	"github.com/yca-software/2chi-go-api/internals/models"
)

var DefaultOwnerPermissions = models.RolePermissions{
	constants.PERMISSION_ORG_READ,
	constants.PERMISSION_ORG_WRITE,
	constants.PERMISSION_ORG_DELETE,

	constants.PERMISSION_MEMBERS_READ,
	constants.PERMISSION_MEMBERS_WRITE,
	constants.PERMISSION_MEMBERS_DELETE,

	constants.PERMISSION_SUBSCRIPTION_READ,
	constants.PERMISSION_SUBSCRIPTION_WRITE,

	constants.PERMISSION_ROLE_READ,
	constants.PERMISSION_ROLE_WRITE,
	constants.PERMISSION_ROLE_DELETE,

	constants.PERMISSION_TEAM_READ,
	constants.PERMISSION_TEAM_WRITE,
	constants.PERMISSION_TEAM_DELETE,

	constants.PERMISSION_TEAM_MEMBER_READ,
	constants.PERMISSION_TEAM_MEMBER_WRITE,
	constants.PERMISSION_TEAM_MEMBER_DELETE,

	constants.PERMISSION_API_KEY_READ,
	constants.PERMISSION_API_KEY_WRITE,
	constants.PERMISSION_API_KEY_DELETE,

	constants.PERMISSION_AUDIT_READ,
}

var DefaultTeamMemberPermissions = models.RolePermissions{
	constants.PERMISSION_ORG_READ,
	constants.PERMISSION_MEMBERS_READ,
	constants.PERMISSION_TEAM_READ,
	constants.PERMISSION_TEAM_MEMBER_READ,
}

var DefaultRolesToCreateForOrganization = []models.Role{
	{
		Name:        "Owner",
		Description: "Full access to the organization",
		Permissions: DefaultOwnerPermissions,
		Locked:      true,
	},
	{
		Name:        "Member",
		Description: "Default member role",
		Permissions: DefaultTeamMemberPermissions,
		Locked:      false,
	},
}
