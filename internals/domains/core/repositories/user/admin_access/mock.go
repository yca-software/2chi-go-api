package admin_access_repository

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/yca-software/2chi-go-api/internals/domains/core/models"
	chi_repository "github.com/yca-software/2chi-go-repository"
)

type MockAdminAccessRepository struct {
	mock.Mock
}

func (m *MockAdminAccessRepository) WithTx(_ chi_repository.Tx) AdminAccessRepository {
	return m
}

func (m *MockAdminAccessRepository) GetAdminAccessByUserID(ctx context.Context, userID string) (*models.AdminAccess, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.AdminAccess), args.Error(1)
}

func (m *MockAdminAccessRepository) DeleteAdminAccessByUserID(ctx context.Context, userID string) error {
	return m.Called(ctx, userID).Error(0)
}
