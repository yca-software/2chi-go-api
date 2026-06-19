package audit_service

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/yca-software/2chi-go-api/internals/domains/core/models"
	chi_types "github.com/yca-software/2chi-go-types"
)

type MockService struct {
	mock.Mock
}

func (m *MockService) CreateAuditLog(ctx context.Context, req *CreateAuditLogRequest, access *chi_types.AccessInfo) (*models.AuditLog, error) {
	args := m.Called(ctx, req, access)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.AuditLog), args.Error(1)
}

func (m *MockService) ListAuditLogsForOrganization(ctx context.Context, req *ListAuditLogsForOrganizationRequest, access *chi_types.AccessInfo) (*ListAuditLogsForOrganizationResponse, error) {
	args := m.Called(ctx, req, access)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ListAuditLogsForOrganizationResponse), args.Error(1)
}
