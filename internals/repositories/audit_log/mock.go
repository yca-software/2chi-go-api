package audit_log_repository

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

func (m *MockRepository) Create(ctx context.Context, log *models.AuditLog) error {
	return m.Called(ctx, log).Error(0)
}

func (m *MockRepository) ListByOrganizationID(ctx context.Context, organizationID string, filters *AuditLogFilters, limit, offset int) (*[]models.AuditLog, error) {
	args := m.Called(ctx, organizationID, filters, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*[]models.AuditLog), args.Error(1)
}
