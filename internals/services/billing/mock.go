package billing_service

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/yca-software/2chi-go-api/internals/models"
	chi_types "github.com/yca-software/2chi-go-types"
)

type MockService struct {
	mock.Mock
}

func (m *MockService) CreateCustomer(ctx context.Context, input *CreateCustomerInput) (string, error) {
	args := m.Called(ctx, input)
	return args.String(0), args.Error(1)
}

func (m *MockService) UpdateCustomer(ctx context.Context, input *UpdateCustomerInput) error {
	args := m.Called(ctx, input)
	return args.Error(0)
}

func (m *MockService) ReleaseProvisionedCustomer(ctx context.Context, organizationID string, billingAccount *models.OrganizationBillingAccount) error {
	args := m.Called(ctx, organizationID, billingAccount)
	return args.Error(0)
}

func (m *MockService) CancelSubscription(ctx context.Context, providerSubscriptionID string) error {
	args := m.Called(ctx, providerSubscriptionID)
	return args.Error(0)
}

func (m *MockService) CreateCheckoutSession(ctx context.Context, req *CreateCheckoutSessionRequest, access *chi_types.AccessInfo) (*CheckoutSessionResponse, error) {
	args := m.Called(ctx, req, access)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*CheckoutSessionResponse), args.Error(1)
}

func (m *MockService) CreateCustomerPortalSession(ctx context.Context, req *CreateCustomerPortalSessionRequest, access *chi_types.AccessInfo) (*CustomerPortalSessionResponse, error) {
	args := m.Called(ctx, req, access)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*CustomerPortalSessionResponse), args.Error(1)
}

func (m *MockService) ProcessTransaction(ctx context.Context, req *ProcessTransactionRequest, access *chi_types.AccessInfo) (*models.OrganizationBillingAccount, error) {
	args := m.Called(ctx, req, access)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.OrganizationBillingAccount), args.Error(1)
}

func (m *MockService) ChangePlan(ctx context.Context, req *ChangePlanRequest, access *chi_types.AccessInfo) (*ChangePlanResponse, error) {
	args := m.Called(ctx, req, access)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ChangePlanResponse), args.Error(1)
}

func (m *MockService) HandleWebhook(ctx context.Context, payload []byte, signature string) error {
	args := m.Called(ctx, payload, signature)
	return args.Error(0)
}

func (m *MockService) ApplyScheduledPlanChanges(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}
