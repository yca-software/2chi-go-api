package models

import (
	"time"

	"github.com/google/uuid"
	chi_types "github.com/yca-software/2chi-go-types"
)

type APIKey struct {
	chi_types.ModelBase
	OrganizationID uuid.UUID `json:"organizationId" db:"organization_id"`

	ExpiresAt time.Time `db:"expires_at" json:"expiresAt"`

	Name        string          `db:"name" json:"name"`
	KeyPrefix   string          `db:"key_prefix" json:"keyPrefix"`
	KeyHash     string          `db:"key_hash" json:"-"`
	Permissions RolePermissions `db:"permissions" json:"permissions"`
}
