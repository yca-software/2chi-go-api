package billing_service

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yca-software/2chi-go-api/internals/constants"
)

func TestPriceCatalog_OurPriceIDFromWebhookItems(t *testing.T) {
	catalog := PriceCatalog{
		PriceIDs: PriceIDs{
			BasicMonthly: "pri_basic_monthly",
			ProAnnual:    "pri_pro_annual",
		},
	}

	items := []any{
		map[string]any{
			"price": map[string]any{"id": "pri_unknown"},
		},
		map[string]any{
			"price": map[string]any{"id": "pri_basic_monthly"},
		},
	}

	require.Equal(t, "pri_basic_monthly", catalog.OurPriceIDFromWebhookItems(items))
}

func TestPriceCatalog_TierFromPriceID(t *testing.T) {
	catalog := PriceCatalog{
		PriceIDs: PriceIDs{
			BasicMonthly: "pri_basic_monthly",
			ProAnnual:    "pri_pro_annual",
		},
	}

	require.Equal(t, constants.TIER_BASIC, catalog.TierFromPriceID("pri_basic_monthly"))
	require.Equal(t, constants.TIER_PRO, catalog.TierFromPriceID("pri_pro_annual"))
	require.Equal(t, constants.TIER_FREE, catalog.TierFromPriceID("pri_unknown"))
}
