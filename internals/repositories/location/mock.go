package organization_location_repository

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/yca-software/2chi-go-api/internals/models"
	chi_repository "github.com/yca-software/2chi-go-repository"
)

type MockOrganizationLocationsRepository struct {
	mock.Mock
}

func (m *MockOrganizationLocationsRepository) WithTx(_ chi_repository.Tx) OrganizationLocationsRepository {
	return m
}

func (m *MockOrganizationLocationsRepository) CreateOrganizationLocation(ctx context.Context, location *models.OrganizationLocation) error {
	return m.Called(ctx, location).Error(0)
}

func (m *MockOrganizationLocationsRepository) UpdateOrganizationLocation(ctx context.Context, location *models.OrganizationLocation) error {
	return m.Called(ctx, location).Error(0)
}

func (m *MockOrganizationLocationsRepository) GetOrganizationLocationByID(ctx context.Context, organizationID, id string) (*models.OrganizationLocation, error) {
	args := m.Called(ctx, organizationID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.OrganizationLocation), args.Error(1)
}

func (m *MockOrganizationLocationsRepository) GetOrganizationLocationByOrganizationID(ctx context.Context, organizationID string) (*models.OrganizationLocation, error) {
	args := m.Called(ctx, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.OrganizationLocation), args.Error(1)
}
