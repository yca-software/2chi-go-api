package role_repository

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/yca-software/2chi-go-api/internals/domains/core/models"
	chi_repository "github.com/yca-software/2chi-go-repository"
)

type MockRolesRepository struct {
	mock.Mock
}

func (m *MockRolesRepository) WithTx(_ chi_repository.Tx) RolesRepository {
	return m
}

func (m *MockRolesRepository) CreateRole(ctx context.Context, role *models.Role) error {
	return m.Called(ctx, role).Error(0)
}

func (m *MockRolesRepository) CreateRoles(ctx context.Context, roles *[]models.Role) error {
	return m.Called(ctx, roles).Error(0)
}

func (m *MockRolesRepository) UpdateRole(ctx context.Context, role *models.Role) error {
	return m.Called(ctx, role).Error(0)
}

func (m *MockRolesRepository) DeleteRole(ctx context.Context, organizationID, id string) error {
	return m.Called(ctx, organizationID, id).Error(0)
}

func (m *MockRolesRepository) GetRoleByID(ctx context.Context, organizationID, id string) (*models.Role, error) {
	args := m.Called(ctx, organizationID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Role), args.Error(1)
}

func (m *MockRolesRepository) ListRolesByOrganizationID(ctx context.Context, organizationID string) (*[]models.Role, error) {
	args := m.Called(ctx, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*[]models.Role), args.Error(1)
}
