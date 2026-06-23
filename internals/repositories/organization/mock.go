package organization_repository

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/yca-software/2chi-go-api/internals/models"
	chi_archive "github.com/yca-software/2chi-go-archive"
	chi_repository "github.com/yca-software/2chi-go-repository"
)

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) WithTx(_ chi_repository.Tx) Repository {
	return m
}

func (m *MockRepository) Create(ctx context.Context, organization *models.Organization) error {
	return m.Called(ctx, organization).Error(0)
}

func (m *MockRepository) Update(ctx context.Context, organization *models.Organization) error {
	return m.Called(ctx, organization).Error(0)
}

func (m *MockRepository) Archive(ctx context.Context, organization *models.Organization) error {
	return m.Called(ctx, organization).Error(0)
}

func (m *MockRepository) Restore(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}

func (m *MockRepository) CleanupArchived(ctx context.Context) error {
	return m.Called(ctx).Error(0)
}

func (m *MockRepository) GetByID(ctx context.Context, id string) (*models.Organization, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Organization), args.Error(1)
}

func (m *MockRepository) GetByIDIncludeArchived(ctx context.Context, id string) (*models.Organization, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Organization), args.Error(1)
}

func (m *MockRepository) Search(ctx context.Context, searchPhrase string, filter chi_archive.ArchiveFilter, limit, offset int) (*[]models.Organization, error) {
	args := m.Called(ctx, searchPhrase, filter, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*[]models.Organization), args.Error(1)
}
