package role_repository

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

func (m *MockRepository) Create(ctx context.Context, role *models.Role) error {
	return m.Called(ctx, role).Error(0)
}

func (m *MockRepository) CreateMany(ctx context.Context, roles *[]models.Role) error {
	return m.Called(ctx, roles).Error(0)
}

func (m *MockRepository) Update(ctx context.Context, role *models.Role) error {
	return m.Called(ctx, role).Error(0)
}

func (m *MockRepository) Delete(ctx context.Context, organizationID, id string) error {
	return m.Called(ctx, organizationID, id).Error(0)
}

func (m *MockRepository) GetByID(ctx context.Context, organizationID, id string) (*models.Role, error) {
	args := m.Called(ctx, organizationID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Role), args.Error(1)
}

func (m *MockRepository) ListByOrganizationID(ctx context.Context, organizationID string) (*[]models.Role, error) {
	args := m.Called(ctx, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*[]models.Role), args.Error(1)
}
