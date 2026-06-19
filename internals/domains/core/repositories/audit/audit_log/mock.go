package audit_log_repository

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/yca-software/2chi-go-api/internals/domains/core/models"
	chi_repository "github.com/yca-software/2chi-go-repository"
)

type MockAuditLogsRepository struct {
	mock.Mock
}

func (m *MockAuditLogsRepository) WithTx(_ chi_repository.Tx) AuditLogsRepository {
	return m
}

func (m *MockAuditLogsRepository) CreateAuditLog(ctx context.Context, log *models.AuditLog) error {
	return m.Called(ctx, log).Error(0)
}

func (m *MockAuditLogsRepository) ListAuditLogsByOrganizationID(ctx context.Context, organizationID string, filters *AuditLogFilters, limit, offset int) (*[]models.AuditLog, error) {
	args := m.Called(ctx, organizationID, filters, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*[]models.AuditLog), args.Error(1)
}
