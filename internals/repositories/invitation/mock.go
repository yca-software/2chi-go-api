package invitation_repository

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

func (m *MockRepository) Create(ctx context.Context, invitation *models.Invitation) error {
	return m.Called(ctx, invitation).Error(0)
}

func (m *MockRepository) Update(ctx context.Context, invitation *models.Invitation) error {
	return m.Called(ctx, invitation).Error(0)
}

func (m *MockRepository) GetByID(ctx context.Context, organizationID, id string) (*models.Invitation, error) {
	args := m.Called(ctx, organizationID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Invitation), args.Error(1)
}

func (m *MockRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*models.Invitation, error) {
	args := m.Called(ctx, tokenHash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Invitation), args.Error(1)
}

func (m *MockRepository) ListByOrganizationID(ctx context.Context, organizationID string) (*[]models.Invitation, error) {
	args := m.Called(ctx, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*[]models.Invitation), args.Error(1)
}

func (m *MockRepository) CleanupStale(ctx context.Context) error {
	return m.Called(ctx).Error(0)
}
