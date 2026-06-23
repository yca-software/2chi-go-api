package api_key_repository

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

func (m *MockRepository) Create(ctx context.Context, apiKey *models.APIKey) error {
	return m.Called(ctx, apiKey).Error(0)
}

func (m *MockRepository) Update(ctx context.Context, apiKey *models.APIKey) error {
	return m.Called(ctx, apiKey).Error(0)
}

func (m *MockRepository) Delete(ctx context.Context, organizationID, id string) error {
	return m.Called(ctx, organizationID, id).Error(0)
}

func (m *MockRepository) GetByID(ctx context.Context, organizationID, id string) (*models.APIKey, error) {
	args := m.Called(ctx, organizationID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.APIKey), args.Error(1)
}

func (m *MockRepository) GetByHash(ctx context.Context, hash string) (*models.APIKey, error) {
	args := m.Called(ctx, hash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.APIKey), args.Error(1)
}

func (m *MockRepository) ListByOrganizationID(ctx context.Context, organizationID string) (*[]models.APIKey, error) {
	args := m.Called(ctx, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*[]models.APIKey), args.Error(1)
}
