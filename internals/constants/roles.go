package constants

import "github.com/yca-software/2chi-go-api/internals/domains/core/models"

var DefaultOwnerPermissions = models.RolePermissions{
	PERMISSION_ORG_READ,
	PERMISSION_ORG_WRITE,
	PERMISSION_ORG_DELETE,

	PERMISSION_MEMBERS_READ,
	PERMISSION_MEMBERS_WRITE,
	PERMISSION_MEMBERS_DELETE,

	PERMISSION_SUBSCRIPTION_READ,
	PERMISSION_SUBSCRIPTION_WRITE,

	PERMISSION_ROLE_READ,
	PERMISSION_ROLE_WRITE,
	PERMISSION_ROLE_DELETE,

	PERMISSION_TEAM_READ,
	PERMISSION_TEAM_WRITE,
	PERMISSION_TEAM_DELETE,

	PERMISSION_TEAM_MEMBER_READ,
	PERMISSION_TEAM_MEMBER_WRITE,
	PERMISSION_TEAM_MEMBER_DELETE,

	PERMISSION_API_KEY_READ,
	PERMISSION_API_KEY_WRITE,
	PERMISSION_API_KEY_DELETE,

	PERMISSION_AUDIT_READ,
}

var DefaultTeamMemberPermissions = models.RolePermissions{
	PERMISSION_ORG_READ,
	PERMISSION_MEMBERS_READ,
	PERMISSION_TEAM_READ,
	PERMISSION_TEAM_MEMBER_READ,
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
