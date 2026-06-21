package role_service

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/yca-software/2chi-go-api/internals/models"
	chi_types "github.com/yca-software/2chi-go-types"
)

type MockService struct {
	mock.Mock
}

func (m *MockService) CreateRole(ctx context.Context, req *CreateRoleRequest, access *chi_types.AccessInfo) (*models.Role, error) {
	args := m.Called(ctx, req, access)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Role), args.Error(1)
}

func (m *MockService) UpdateRole(ctx context.Context, req *UpdateRoleRequest, access *chi_types.AccessInfo) (*models.Role, error) {
	args := m.Called(ctx, req, access)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Role), args.Error(1)
}

func (m *MockService) DeleteRole(ctx context.Context, req *DeleteRoleRequest, access *chi_types.AccessInfo) error {
	args := m.Called(ctx, req, access)
	return args.Error(0)
}

func (m *MockService) ListRoles(ctx context.Context, req *ListRolesRequest, access *chi_types.AccessInfo) (*[]models.Role, error) {
	args := m.Called(ctx, req, access)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*[]models.Role), args.Error(1)
}
