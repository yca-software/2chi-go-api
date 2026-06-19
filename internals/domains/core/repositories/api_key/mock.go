package api_key_repository

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/yca-software/2chi-go-api/internals/domains/core/models"
	chi_repository "github.com/yca-software/2chi-go-repository"
)

type MockAPIKeysRepository struct {
	mock.Mock
}

func (m *MockAPIKeysRepository) WithTx(_ chi_repository.Tx) APIKeysRepository {
	return m
}

func (m *MockAPIKeysRepository) CreateAPIKey(ctx context.Context, apiKey *models.APIKey) error {
	return m.Called(ctx, apiKey).Error(0)
}

func (m *MockAPIKeysRepository) UpdateAPIKey(ctx context.Context, apiKey *models.APIKey) error {
	return m.Called(ctx, apiKey).Error(0)
}

func (m *MockAPIKeysRepository) DeleteAPIKey(ctx context.Context, organizationID, id string) error {
	return m.Called(ctx, organizationID, id).Error(0)
}

func (m *MockAPIKeysRepository) GetAPIKeyByID(ctx context.Context, organizationID, id string) (*models.APIKey, error) {
	args := m.Called(ctx, organizationID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.APIKey), args.Error(1)
}

func (m *MockAPIKeysRepository) GetAPIKeyByHash(ctx context.Context, hash string) (*models.APIKey, error) {
	args := m.Called(ctx, hash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.APIKey), args.Error(1)
}

func (m *MockAPIKeysRepository) ListAPIKeysByOrganizationID(ctx context.Context, organizationID string) (*[]models.APIKey, error) {
	args := m.Called(ctx, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*[]models.APIKey), args.Error(1)
}
