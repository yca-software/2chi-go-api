package organization_service

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/yca-software/2chi-go-api/internals/models"
	chi_types "github.com/yca-software/2chi-go-types"
)

type MockService struct {
	mock.Mock
}

func (m *MockService) CreateOrganization(ctx context.Context, req *CreateOrganizationRequest, access *chi_types.AccessInfo) (*CreateOrganizationResponse, error) {
	args := m.Called(ctx, req, access)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*CreateOrganizationResponse), args.Error(1)
}

func (m *MockService) UpdateOrganization(ctx context.Context, req *UpdateOrganizationRequest, access *chi_types.AccessInfo) (*models.Organization, error) {
	args := m.Called(ctx, req, access)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Organization), args.Error(1)
}

func (m *MockService) UpdateOrganizationSubscription(ctx context.Context, req *UpdateOrganizationSubscriptionRequest, access *chi_types.AccessInfo) (*models.OrganizationBillingAccount, error) {
	args := m.Called(ctx, req, access)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.OrganizationBillingAccount), args.Error(1)
}

func (m *MockService) UpdateOrganizationMember(ctx context.Context, req *UpdateOrganizationMemberRequest, access *chi_types.AccessInfo) (*models.OrganizationMemberWithUser, error) {
	args := m.Called(ctx, req, access)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.OrganizationMemberWithUser), args.Error(1)
}

func (m *MockService) ArchiveOrganization(ctx context.Context, req *ArchiveOrganizationRequest, access *chi_types.AccessInfo) error {
	args := m.Called(ctx, req, access)
	return args.Error(0)
}

func (m *MockService) RestoreOrganization(ctx context.Context, req *RestoreOrganizationRequest, access *chi_types.AccessInfo) (*models.Organization, error) {
	args := m.Called(ctx, req, access)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Organization), args.Error(1)
}

func (m *MockService) DeleteOrganizationMember(ctx context.Context, req *DeleteOrganizationMemberRequest, access *chi_types.AccessInfo) error {
	args := m.Called(ctx, req, access)
	return args.Error(0)
}

func (m *MockService) GetOrganization(ctx context.Context, req *GetOrganizationRequest, access *chi_types.AccessInfo) (*models.Organization, error) {
	args := m.Called(ctx, req, access)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Organization), args.Error(1)
}

func (m *MockService) GetOrganizationBillingAccount(ctx context.Context, req *GetOrganizationBillingAccountRequest, access *chi_types.AccessInfo) (*models.OrganizationBillingAccount, error) {
	args := m.Called(ctx, req, access)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.OrganizationBillingAccount), args.Error(1)
}

func (m *MockService) GetArchivedOrganization(ctx context.Context, req *GetOrganizationRequest, access *chi_types.AccessInfo) (*models.Organization, error) {
	args := m.Called(ctx, req, access)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Organization), args.Error(1)
}

func (m *MockService) ListOrganizations(ctx context.Context, req *ListOrganizationsRequest, access *chi_types.AccessInfo) (*ListOrganizationsResponse, error) {
	args := m.Called(ctx, req, access)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ListOrganizationsResponse), args.Error(1)
}

func (m *MockService) ListOrganizationMembers(ctx context.Context, req *ListOrganizationMembersRequest, access *chi_types.AccessInfo) (*[]models.OrganizationMemberWithUser, error) {
	args := m.Called(ctx, req, access)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*[]models.OrganizationMemberWithUser), args.Error(1)
}

func (m *MockService) ListOrganizationRolesForUser(ctx context.Context, req *ListOrganizationRolesForUserRequest, access *chi_types.AccessInfo) (*[]models.OrganizationMemberWithOrganizationAndRole, error) {
	args := m.Called(ctx, req, access)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*[]models.OrganizationMemberWithOrganizationAndRole), args.Error(1)
}

func (m *MockService) AdminCreateOrganization(ctx context.Context, req *AdminCreateOrganizationRequest, access *chi_types.AccessInfo) (*models.Organization, error) {
	args := m.Called(ctx, req, access)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Organization), args.Error(1)
}

func (m *MockService) CleanupArchivedOrganizations(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}
