package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type AuditLog struct {
	ID        uuid.UUID `json:"id" db:"id"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`

	OrganizationID uuid.UUID `json:"organizationId" db:"organization_id"`

	ActorID             uuid.UUID     `db:"actor_id" json:"actorId"`
	ActorInfo           string        `db:"actor_info" json:"actorInfo"`
	ImpersonatedByID    uuid.NullUUID `db:"impersonated_by_id" json:"impersonatedById"`
	ImpersonatedByEmail string        `db:"impersonated_by_email" json:"impersonatedByEmail"`

	Action       string    `db:"action" json:"action"`
	ResourceType string    `db:"resource_type" json:"resourceType"`
	ResourceID   uuid.UUID `db:"resource_id" json:"resourceId"`
	ResourceName string    `db:"resource_name" json:"resourceName"`

	Data *json.RawMessage `db:"data" json:"data"`
}

// AuditLogPublic is the client-safe audit log shape (sanitized data, optional impersonation).
type AuditLogPublic struct {
	ID        uuid.UUID `json:"id" db:"id"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`

	OrganizationID uuid.UUID `json:"organizationId" db:"organization_id"`

	ActorID             uuid.UUID     `json:"actorId"`
	ActorInfo           string        `json:"actorInfo"`
	ImpersonatedByID    uuid.NullUUID `json:"-"`
	ImpersonatedByEmail string        `json:"-"`

	Action       string    `json:"action"`
	ResourceType string    `json:"resourceType"`
	ResourceID   uuid.UUID `json:"resourceId"`
	ResourceName string    `json:"resourceName"`

	Data json.RawMessage `json:"data"`
}

type ImpersonationSession struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	StartedAt time.Time  `json:"startedAt" db:"started_at"`
	EndedAt   *time.Time `json:"endedAt,omitempty" db:"ended_at"`
	EndReason *string    `json:"endReason,omitempty" db:"end_reason"`

	AdminID    uuid.UUID `json:"adminId" db:"admin_id"`
	AdminEmail string    `json:"adminEmail" db:"admin_email"`

	TargetUserID    uuid.UUID `json:"targetUserId" db:"target_user_id"`
	TargetUserEmail string    `json:"targetUserEmail" db:"target_user_email"`

	RefreshTokenID uuid.UUID `json:"refreshTokenId" db:"refresh_token_id"`

	IP        string `json:"ip" db:"ip"`
	UserAgent string `json:"userAgent" db:"user_agent"`
}
