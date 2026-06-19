package organization_service

import (
	"time"

	chi_archive "github.com/yca-software/2chi-go-archive"
	"github.com/yca-software/2chi-go-api/internals/domains/core/models"
	chi_types "github.com/yca-software/2chi-go-types"
)

type CreateOrganizationRequest struct {
	Name         string `json:"name" validate:"required,min=1,max=255"`
	PlaceID      string `json:"placeId" validate:"required,min=1,max=255"`
	BillingEmail string `json:"billingEmail" validate:"required,email"`
}

type CreateOrganizationResponse struct {
	Organization *models.Organization       `json:"organization"`
	Roles        *[]models.Role             `json:"roles"`
	Member       *models.OrganizationMember `json:"member"`
}

type UpdateOrganizationRequest struct {
	OrganizationID string `json:"-" validate:"required,uuid"`
	Name           string `json:"name" validate:"required,min=1,max=255"`
	PlaceID        string `json:"placeId" validate:"required,min=1,max=255"`
}

type UpdateOrganizationSubscriptionRequest struct {
	OrganizationID        string    `json:"-" validate:"required,uuid"`
	CustomSubscription    bool      `json:"customSubscription" validate:"required"`
	SubscriptionType      string    `json:"subscriptionType" validate:"required,oneof=basic pro enterprise"`
	SubscriptionSeats     int       `json:"subscriptionSeats" validate:"required,min=-1"`
	SubscriptionExpiresAt time.Time `json:"subscriptionExpiresAt" validate:"required"`
}

type ArchiveOrganizationRequest struct {
	OrganizationID string `json:"-" validate:"required,uuid"`
}

type RestoreOrganizationRequest struct {
	OrganizationID string `json:"-" validate:"required,uuid"`
}

type GetOrganizationRequest struct {
	OrganizationID string `json:"-" validate:"required,uuid"`
}

type ListOrganizationsRequest struct {
	SearchPhrase  string                  `json:"-" validate:"omitempty,min=1,max=255"`
	ArchiveFilter chi_archive.ArchiveFilter `json:"-" validate:"omitempty,oneof=active archived all"`
	Limit         int                     `json:"-" validate:"required,min=1,max=100"`
	Offset        int                     `json:"-" validate:"gte=0"`
}

type ListOrganizationsResponse chi_types.PaginatedListResponse[models.Organization]

type ListOrganizationMembersRequest struct {
	OrganizationID string `json:"-" validate:"required,uuid"`
}

type ListOrganizationRolesForUserRequest struct {
	UserID string `json:"-" validate:"required,uuid"`
}

type UpdateOrganizationMemberRequest struct {
	OrganizationID string `json:"-" validate:"required,uuid"`
	MemberID       string `json:"-" validate:"required,uuid"`
	RoleID         string `json:"roleId" validate:"required,uuid"`
}

type DeleteOrganizationMemberRequest struct {
	OrganizationID string `json:"-" validate:"required,uuid"`
	MemberID       string `json:"-" validate:"required,uuid"`
}

type AdminCreateOrganizationRequest struct {
	Name                  string     `json:"name" validate:"required,min=1,max=255"`
	PlaceID               string     `json:"placeId" validate:"required,min=1,max=255"`
	BillingEmail          string     `json:"billingEmail" validate:"required,email"`
	OwnerEmail            string     `json:"ownerEmail" validate:"required,email"`
	SubscriptionType      string     `json:"subscriptionType" validate:"required,oneof=basic pro enterprise"`
	SubscriptionSeats     int        `json:"subscriptionSeats" validate:"required,min=-1"`
	SubscriptionExpiresAt *time.Time `json:"subscriptionExpiresAt"`
	Language              string     `json:"language" validate:"required"`
}
