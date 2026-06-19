package billing_service

import (
	"github.com/yca-software/2chi-go-api/internals/constants"
	"github.com/yca-software/2chi-go-api/internals/domains/core/models"
)

type PriceIDs struct {
	BasicMonthly string
	BasicAnnual  string
	ProMonthly   string
	ProAnnual    string
}

type PriceCatalog struct {
	PriceIDs PriceIDs
}

func (c PriceCatalog) IsOurPriceID(priceID string) bool {
	if priceID == "" {
		return false
	}
	return priceID == c.PriceIDs.BasicMonthly ||
		priceID == c.PriceIDs.BasicAnnual ||
		priceID == c.PriceIDs.ProMonthly ||
		priceID == c.PriceIDs.ProAnnual
}

func (c PriceCatalog) TierFromPriceID(priceID string) string {
	switch priceID {
	case c.PriceIDs.BasicMonthly, c.PriceIDs.BasicAnnual:
		return constants.TIER_BASIC
	case c.PriceIDs.ProMonthly, c.PriceIDs.ProAnnual:
		return constants.TIER_PRO
	default:
		return constants.TIER_FREE
	}
}

func (c PriceCatalog) IntervalFromPriceID(priceID string) string {
	switch priceID {
	case c.PriceIDs.BasicAnnual, c.PriceIDs.ProAnnual:
		return constants.PAYMENT_INTERVAL_ANNUAL
	default:
		return constants.PAYMENT_INTERVAL_MONTHLY
	}
}

func (c PriceCatalog) OurPriceIDFromWebhookItems(items []any) string {
	for _, item := range items {
		itemMap, ok := item.(map[string]any)
		if !ok {
			continue
		}
		price, ok := itemMap["price"].(map[string]any)
		if !ok {
			continue
		}
		priceID, _ := price["id"].(string)
		if c.IsOurPriceID(priceID) {
			return priceID
		}
	}
	return ""
}

type CreateCustomerInput struct {
	OrganizationID   string
	OrganizationName string
	BillingEmail     string
	Location         *models.OrganizationLocation
}

type UpdateCustomerInput struct {
	OrganizationID   string
	OrganizationName string
	BillingAccount   *models.OrganizationBillingAccount
	Location         *models.OrganizationLocation
}

type CreateCheckoutSessionRequest struct {
	OrganizationID string `json:"organizationId" validate:"required,uuid"`
	PlanID         string `json:"planId" validate:"required"`
}

type CheckoutSessionResponse struct {
	TransactionID string `json:"transactionId"`
}

type CreateCustomerPortalSessionRequest struct {
	OrganizationID string `json:"organizationId" validate:"required,uuid"`
}

type CustomerPortalSessionResponse struct {
	PortalURL string `json:"portalUrl"`
}

type ProcessTransactionRequest struct {
	OrganizationID string `json:"organizationId" validate:"required,uuid"`
	TransactionID  string `json:"transactionId" validate:"required"`
	PriceID        string `json:"priceId" validate:"required"`
}

type ChangePlanRequest struct {
	OrganizationID string `json:"organizationId" validate:"required,uuid"`
	PlanID         string `json:"planId" validate:"required"`
}

type ChangePlanResponse struct {
	BillingAccount *models.OrganizationBillingAccount `json:"billingAccount"`
	EffectiveAt    string                             `json:"effectiveAt"`
}

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
