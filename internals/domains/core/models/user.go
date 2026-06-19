package models

import (
	"time"

	"github.com/google/uuid"
	chi_types "github.com/yca-software/2chi-go-types"
)

type User struct {
	chi_types.ModelBaseWithArchive

	Email           string     `json:"email" db:"email"`
	EmailVerifiedAt *time.Time `json:"emailVerifiedAt" db:"email_verified_at"`

	FirstName string `json:"firstName" db:"first_name"`
	LastName  string `json:"lastName" db:"last_name"`
	AvatarURL string `json:"avatarURL" db:"avatar_url"`

	Language string `json:"language" db:"language"`
	Password string `json:"password" db:"password"`
}

type UserLegalDocumentAcceptance struct {
	chi_types.ModelBase
	UserID uuid.UUID `json:"userId" db:"user_id"`

	DocumentType    string `json:"documentType" db:"document_type"` // e.g. "terms_of_service", "privacy_policy"
	DocumentVersion string `json:"documentVersion" db:"document_version"`
}

type AdminAccess struct {
	UserID    uuid.UUID `json:"userId" db:"user_id"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
}

type UserIdentity struct {
	chi_types.ModelBase
	UserID uuid.UUID `json:"userId" db:"user_id"`

	Provider       string `json:"provider" db:"provider"`               // e.g. "google", "github", "linkedin"
	ProviderUserID string `json:"providerUserId" db:"provider_user_id"` // e.g. "google_id", "github_id", "linkedin_id"
}

type UserRefreshToken struct {
	chi_types.ModelBase
	UserID uuid.UUID `json:"userId" db:"user_id"`

	ExpiresAt time.Time  `json:"expiresAt" db:"expires_at"`
	RevokedAt *time.Time `json:"revokedAt" db:"revoked_at"`

	IP             string        `json:"ip" db:"ip"`
	UserAgent      string        `json:"userAgent" db:"user_agent"`
	TokenHash      string        `json:"-" db:"token_hash"`
	ImpersonatedBy uuid.NullUUID `json:"-" db:"impersonated_by"`
}

type UserPasswordResetToken struct {
	chi_types.ModelBase
	UserID uuid.UUID `json:"userId" db:"user_id"`

	ExpiresAt time.Time  `json:"expiresAt" db:"expires_at"`
	UsedAt    *time.Time `json:"usedAt" db:"used_at"`

	TokenHash string `json:"-" db:"token_hash"`
}

type UserEmailVerificationToken struct {
	chi_types.ModelBase
	UserID uuid.UUID `json:"userId" db:"user_id"`

	CreatedAt time.Time  `json:"createdAt" db:"created_at"`
	ExpiresAt time.Time  `json:"expiresAt" db:"expires_at"`
	UsedAt    *time.Time `json:"usedAt" db:"used_at"`

	TokenHash string `json:"-" db:"token_hash"`
}
