package models

import (
	"time"

	"github.com/google/uuid"
)

type OrganizationBillingAccount struct {
	OrganizationID uuid.UUID `json:"organizationId" db:"organization_id"`
	CreatedAt      time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt      time.Time `json:"updatedAt" db:"updated_at"`

	BillingEmail string `db:"billing_email" json:"billingEmail"`

	Provider           string `db:"provider" json:"provider"` // paddle, stripe, customer etc.
	ProviderCustomerID string `db:"provider_customer_id" json:"providerCustomerId"`

	ProviderSubscriptionID string `db:"provider_subscription_id" json:"providerSubscriptionId"`

	SubscriptionExpiresAt       *time.Time `db:"subscription_expires_at" json:"subscriptionExpiresAt"`
	SubscriptionPaymentInterval string     `db:"subscription_payment_interval" json:"subscriptionPaymentInterval"`
	SubscriptionTier            string     `db:"subscription_tier" json:"subscriptionTier"`
	SubscriptionSeats           int        `db:"subscription_seats" json:"subscriptionSeats"`
	SubscriptionInTrial         bool       `db:"subscription_in_trial" json:"subscriptionInTrial"`

	// When set: switch to this price & tier at end of current period (SubscriptionExpiresAt). Used for annual→monthly.
	SubscriptionScheduledPlanPriceID string `db:"subscription_scheduled_plan_price_id" json:"subscriptionScheduledPlanPriceId"`
}
