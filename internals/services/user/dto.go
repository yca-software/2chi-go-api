package user_service

import (
	"time"

	"github.com/yca-software/2chi-go-api/internals/models"
	chi_archive "github.com/yca-software/2chi-go-archive"
	chi_types "github.com/yca-software/2chi-go-types"
)

type AcceptTermsRequest struct {
	UserID               string `json:"-" validate:"required,uuid"`
	TermsVersion         string `json:"termsVersion" validate:"required,semver"`
	PrivacyPolicyVersion string `json:"privacyPolicyVersion" validate:"required,semver"`
}

type ChangePasswordRequest struct {
	UserID      string `json:"-" validate:"required,uuid"`
	OldPassword string `json:"oldPassword" validate:"required"`
	NewPassword string `json:"newPassword" validate:"required,min=8"`
}

type ArchiveUserRequest struct {
	UserID string `json:"-" validate:"required,uuid"`
}

type GetUserRequest struct {
	UserID string `json:"-" validate:"required,uuid"`
}

type UserProfile struct {
	models.User
	TermsVersion            string     `json:"termsVersion"`
	TermsAcceptedAt         *time.Time `json:"termsAcceptedAt"`
	PrivacyPolicyVersion    string     `json:"privacyPolicyVersion"`
	PrivacyPolicyAcceptedAt *time.Time `json:"privacyPolicyAcceptedAt"`
}

type GetUserResponse struct {
	User        UserProfile                                        `json:"user"`
	AdminAccess *models.AdminAccess                                `json:"adminAccess,omitempty"`
	Roles       []models.OrganizationMemberWithOrganizationAndRole `json:"roles"`
}

type ListUsersRequest struct {
	SearchPhrase  string                    `json:"-"`
	ArchiveFilter chi_archive.ArchiveFilter `json:"-" validate:"omitempty,oneof=active archived"`
	Limit         int                       `json:"-" validate:"required,min=1,max=100"`
	Offset        int                       `json:"-" validate:"gte=0"`
}

type ListUsersResponse chi_types.PaginatedListResponse[models.User]

type RestoreUserRequest struct {
	UserID string `json:"-" validate:"required,uuid"`
}

type UpdateProfileRequest struct {
	UserID    string `json:"-" validate:"required,uuid"`
	FirstName string `json:"firstName" validate:"required,min=1,max=255"`
	LastName  string `json:"lastName" validate:"required,min=1,max=255"`
}

type UpdateLanguageRequest struct {
	UserID   string `json:"-" validate:"required,uuid"`
	Language string `json:"language" validate:"required,len=2"`
}

type ListUserActiveRefreshTokensRequest struct {
	UserID string `json:"-" validate:"required,uuid"`
}

type RevokeUserRefreshTokenRequest struct {
	UserID         string `json:"-" validate:"required,uuid"`
	RefreshTokenID string `json:"-" validate:"required,uuid"`
}

type RevokeUserAllRefreshTokensRequest struct {
	UserID           string `json:"-" validate:"required,uuid"`
	KeepRefreshToken string `json:"keepRefreshToken,omitempty"`
}

type RevokeUserAdminAccessRequest struct {
	UserID string `json:"-" validate:"required,uuid"`
}

type ResendVerificationEmailRequest struct {
	UserID   string `json:"-" validate:"required,uuid"`
	Language string `json:"-" validate:"omitempty,len=2"`
}
