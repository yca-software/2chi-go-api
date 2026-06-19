package location

import (
	chi_types "github.com/yca-software/2chi-go-types"
)

type LocationModel struct {
	chi_types.ModelBase

	Address  string          `json:"address" db:"address"`
	City     string          `json:"city" db:"city"`
	Zip      string          `json:"zip" db:"zip"`
	Country  string          `json:"country" db:"country"`
	PlaceID  string          `json:"placeId" db:"place_id"`
	Geo      chi_types.Point `json:"geo" db:"geo"`
	Timezone string          `json:"timezone" db:"timezone"`
}
