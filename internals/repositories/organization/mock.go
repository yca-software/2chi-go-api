package organization_repository

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/yca-software/2chi-go-api/internals/models"
	chi_archive "github.com/yca-software/2chi-go-archive"
	chi_repository "github.com/yca-software/2chi-go-repository"
)

type MockOrganizationsRepository struct {
	mock.Mock
}

func (m *MockOrganizationsRepository) WithTx(_ chi_repository.Tx) OrganizationsRepository {
	return m
}

func (m *MockOrganizationsRepository) CreateOrganization(ctx context.Context, organization *models.Organization) error {
	return m.Called(ctx, organization).Error(0)
}

func (m *MockOrganizationsRepository) UpdateOrganization(ctx context.Context, organization *models.Organization) error {
	return m.Called(ctx, organization).Error(0)
}

func (m *MockOrganizationsRepository) ArchiveOrganization(ctx context.Context, organization *models.Organization) error {
	return m.Called(ctx, organization).Error(0)
}

func (m *MockOrganizationsRepository) RestoreOrganization(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}

func (m *MockOrganizationsRepository) CleanupArchivedOrganizations(ctx context.Context) error {
	return m.Called(ctx).Error(0)
}

func (m *MockOrganizationsRepository) GetOrganizationByID(ctx context.Context, id string) (*models.Organization, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Organization), args.Error(1)
}

func (m *MockOrganizationsRepository) GetOrganizationByIDIncludeArchived(ctx context.Context, id string) (*models.Organization, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Organization), args.Error(1)
}

func (m *MockOrganizationsRepository) SearchOrganizations(ctx context.Context, searchPhrase string, filter chi_archive.ArchiveFilter, limit, offset int) (*[]models.Organization, error) {
	args := m.Called(ctx, searchPhrase, filter, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*[]models.Organization), args.Error(1)
}
