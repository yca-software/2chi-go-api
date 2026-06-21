package billing_service

import "github.com/yca-software/2chi-go-api/internals/constants"

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
