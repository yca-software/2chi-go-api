package billing_service

import (
	"errors"
	"net/http"

	"github.com/yca-software/2chi-go-api/internals/constants"
	"github.com/yca-software/2chi-go-api/internals/models"
	chi_error "github.com/yca-software/2chi-go-error"
)

func normalizeTier(tier string) string {
	if tier == "" {
		return constants.TIER_FREE
	}
	return tier
}

func tierRank(tier string) int {
	switch normalizeTier(tier) {
	case constants.TIER_BASIC:
		return 1
	case constants.TIER_PRO:
		return 2
	case constants.TIER_ENTERPRISE:
		return 3
	default:
		return 0
	}
}

func seatsIncludedForTier(tier string) int {
	switch normalizeTier(tier) {
	case constants.TIER_BASIC:
		return constants.SUBSCRIPTION_TYPE_SEATS_INCLUDED_BASIC
	case constants.TIER_PRO:
		return constants.SUBSCRIPTION_TYPE_SEATS_INCLUDED_PRO
	case constants.TIER_ENTERPRISE:
		return constants.SUBSCRIPTION_TYPE_SEATS_INCLUDED_ENTERPRISE
	default:
		return constants.SUBSCRIPTION_TYPE_SEATS_INCLUDED_FREE
	}
}

func applySubscriptionFromPrice(account *models.OrganizationBillingAccount, catalog PriceCatalog, priceID string) {
	account.SubscriptionTier = catalog.TierFromPriceID(priceID)
	account.SubscriptionSeats = seatsIncludedForTier(account.SubscriptionTier)
	account.SubscriptionPaymentInterval = catalog.IntervalFromPriceID(priceID)
}

func webhookNotRelevant(err error) bool {
	var apiErr *chi_error.Error
	if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
		return true
	}
	return false
}
