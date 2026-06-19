package user_identity_repository

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/yca-software/2chi-go-api/internals/domains/core/models"
	chi_repository "github.com/yca-software/2chi-go-repository"
)

type MockUserIdentityRepository struct {
	mock.Mock
}

func (m *MockUserIdentityRepository) WithTx(_ chi_repository.Tx) UserIdentityRepository {
	return m
}

func (m *MockUserIdentityRepository) CreateUserIdentity(ctx context.Context, identity *models.UserIdentity) error {
	return m.Called(ctx, identity).Error(0)
}

func (m *MockUserIdentityRepository) UpdateUserIdentity(ctx context.Context, identity *models.UserIdentity) error {
	return m.Called(ctx, identity).Error(0)
}

func (m *MockUserIdentityRepository) GetUserIdentityByProviderAndProviderUserID(ctx context.Context, provider, providerUserID string) (*models.UserIdentity, error) {
	args := m.Called(ctx, provider, providerUserID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserIdentity), args.Error(1)
}

func (m *MockUserIdentityRepository) GetUserIdentityByUserIDAndProvider(ctx context.Context, userID, provider string) (*models.UserIdentity, error) {
	args := m.Called(ctx, userID, provider)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserIdentity), args.Error(1)
}
