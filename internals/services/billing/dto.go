package billing_service

import (
	"github.com/yca-software/2chi-go-api/internals/models"
)

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
