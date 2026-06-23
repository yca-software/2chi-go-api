package billing_account_repository

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/yca-software/2chi-go-api/internals/models"
	chi_repository "github.com/yca-software/2chi-go-repository"
)

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) WithTx(_ chi_repository.Tx) Repository {
	return m
}

func (m *MockRepository) Create(ctx context.Context, account *models.OrganizationBillingAccount) error {
	return m.Called(ctx, account).Error(0)
}

func (m *MockRepository) Update(ctx context.Context, account *models.OrganizationBillingAccount) error {
	return m.Called(ctx, account).Error(0)
}

func (m *MockRepository) GetByOrganizationID(ctx context.Context, organizationID string) (*models.OrganizationBillingAccount, error) {
	args := m.Called(ctx, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.OrganizationBillingAccount), args.Error(1)
}

func (m *MockRepository) GetByProviderAndProviderCustomerID(ctx context.Context, provider, providerCustomerID string) (*models.OrganizationBillingAccount, error) {
	args := m.Called(ctx, provider, providerCustomerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.OrganizationBillingAccount), args.Error(1)
}

func (m *MockRepository) GetByProviderAndProviderSubscriptionID(ctx context.Context, provider, providerSubscriptionID string) (*models.OrganizationBillingAccount, error) {
	args := m.Called(ctx, provider, providerSubscriptionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.OrganizationBillingAccount), args.Error(1)
}

func (m *MockRepository) ListWithScheduledPlanChangeDue(ctx context.Context) (*[]models.OrganizationBillingAccount, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*[]models.OrganizationBillingAccount), args.Error(1)
}
