package models

import (
	"github.com/google/uuid"
	"github.com/yca-software/2chi-go-api/internals/platform/location"
	chi_types "github.com/yca-software/2chi-go-types"
)

type Organization struct {
	chi_types.ModelBaseWithArchive

	Name string `json:"name" db:"name"`
}

type OrganizationLocation struct {
	location.LocationModel
	OrganizationID uuid.UUID `json:"organizationId" db:"organization_id"`
}
