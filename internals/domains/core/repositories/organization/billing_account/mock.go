package billing_account_repository

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/yca-software/2chi-go-api/internals/domains/core/models"
	chi_repository "github.com/yca-software/2chi-go-repository"
)

type MockOrganizationBillingAccountsRepository struct {
	mock.Mock
}

func (m *MockOrganizationBillingAccountsRepository) WithTx(_ chi_repository.Tx) OrganizationBillingAccountsRepository {
	return m
}

func (m *MockOrganizationBillingAccountsRepository) CreateOrganizationBillingAccount(ctx context.Context, account *models.OrganizationBillingAccount) error {
	return m.Called(ctx, account).Error(0)
}

func (m *MockOrganizationBillingAccountsRepository) UpdateOrganizationBillingAccount(ctx context.Context, account *models.OrganizationBillingAccount) error {
	return m.Called(ctx, account).Error(0)
}

func (m *MockOrganizationBillingAccountsRepository) GetOrganizationBillingAccountByID(ctx context.Context, organizationID, id string) (*models.OrganizationBillingAccount, error) {
	args := m.Called(ctx, organizationID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.OrganizationBillingAccount), args.Error(1)
}

func (m *MockOrganizationBillingAccountsRepository) GetOrganizationBillingAccountByOrganizationID(ctx context.Context, organizationID string) (*models.OrganizationBillingAccount, error) {
	args := m.Called(ctx, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.OrganizationBillingAccount), args.Error(1)
}

func (m *MockOrganizationBillingAccountsRepository) GetOrganizationBillingAccountByProviderAndProviderCustomerID(ctx context.Context, provider, providerCustomerID string) (*models.OrganizationBillingAccount, error) {
	args := m.Called(ctx, provider, providerCustomerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.OrganizationBillingAccount), args.Error(1)
}

func (m *MockOrganizationBillingAccountsRepository) GetOrganizationBillingAccountByProviderAndProviderSubscriptionID(ctx context.Context, provider, providerSubscriptionID string) (*models.OrganizationBillingAccount, error) {
	args := m.Called(ctx, provider, providerSubscriptionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.OrganizationBillingAccount), args.Error(1)
}

func (m *MockOrganizationBillingAccountsRepository) ListOrganizationBillingAccountsWithScheduledPlanChangeDue(ctx context.Context) (*[]models.OrganizationBillingAccount, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*[]models.OrganizationBillingAccount), args.Error(1)
}
