package models

import (
	"database/sql/driver"

	"github.com/google/uuid"
	chi_jsonb "github.com/yca-software/2chi-go-jsonb"
	chi_types "github.com/yca-software/2chi-go-types"
)

type Role struct {
	chi_types.ModelBase
	OrganizationID uuid.UUID `json:"organizationId" db:"organization_id"`

	Name        string          `db:"name" json:"name"`
	Description string          `db:"description" json:"description"`
	Permissions RolePermissions `db:"permissions" json:"permissions"`
	Locked      bool            `db:"locked" json:"locked"`
}

type RolePermissions []string

func (rp RolePermissions) Value() (driver.Value, error) {
	return chi_jsonb.JSONBValue(rp)
}

func (rp *RolePermissions) Scan(value any) error {
	return chi_jsonb.JSONBScan(value, rp)
}
