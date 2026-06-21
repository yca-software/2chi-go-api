package organization_member_repository

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/yca-software/2chi-go-api/internals/models"
	chi_repository "github.com/yca-software/2chi-go-repository"
)

type MockOrganizationMembersRepository struct {
	mock.Mock
}

func (m *MockOrganizationMembersRepository) WithTx(_ chi_repository.Tx) OrganizationMembersRepository {
	return m
}

func (m *MockOrganizationMembersRepository) CreateOrganizationMember(ctx context.Context, member *models.OrganizationMember) error {
	return m.Called(ctx, member).Error(0)
}

func (m *MockOrganizationMembersRepository) UpdateOrganizationMember(ctx context.Context, member *models.OrganizationMember) error {
	return m.Called(ctx, member).Error(0)
}

func (m *MockOrganizationMembersRepository) DeleteOrganizationMember(ctx context.Context, organizationID, userID string) error {
	return m.Called(ctx, organizationID, userID).Error(0)
}

func (m *MockOrganizationMembersRepository) DeleteOrganizationMemberByMembershipID(ctx context.Context, organizationID, memberID string) error {
	return m.Called(ctx, organizationID, memberID).Error(0)
}

func (m *MockOrganizationMembersRepository) GetOrganizationMemberByID(ctx context.Context, organizationID, userID string) (*models.OrganizationMember, error) {
	args := m.Called(ctx, organizationID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.OrganizationMember), args.Error(1)
}

func (m *MockOrganizationMembersRepository) GetOrganizationMemberByMembershipID(ctx context.Context, organizationID, memberID string) (*models.OrganizationMember, error) {
	args := m.Called(ctx, organizationID, memberID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.OrganizationMember), args.Error(1)
}

func (m *MockOrganizationMembersRepository) GetOrganizationMemberByIDWithUser(ctx context.Context, organizationID, userID string) (*models.OrganizationMemberWithUser, error) {
	args := m.Called(ctx, organizationID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.OrganizationMemberWithUser), args.Error(1)
}

func (m *MockOrganizationMembersRepository) GetOrganizationMemberByMembershipIDWithUser(ctx context.Context, organizationID, memberID string) (*models.OrganizationMemberWithUser, error) {
	args := m.Called(ctx, organizationID, memberID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.OrganizationMemberWithUser), args.Error(1)
}

func (m *MockOrganizationMembersRepository) GetOrganizationMemberByIDWithOrganizationAndRole(ctx context.Context, organizationID, userID string) (*models.OrganizationMemberWithOrganizationAndRole, error) {
	args := m.Called(ctx, organizationID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.OrganizationMemberWithOrganizationAndRole), args.Error(1)
}

func (m *MockOrganizationMembersRepository) GetOrganizationMemberByUserIDAndOrganizationID(ctx context.Context, userID, organizationID string) (*models.OrganizationMember, error) {
	args := m.Called(ctx, userID, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.OrganizationMember), args.Error(1)
}

func (m *MockOrganizationMembersRepository) ListByUserID(ctx context.Context, userID string) (*[]models.OrganizationMemberWithOrganization, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*[]models.OrganizationMemberWithOrganization), args.Error(1)
}

func (m *MockOrganizationMembersRepository) ListByUserIDWithRole(ctx context.Context, userID string) (*[]models.OrganizationMemberWithOrganizationAndRole, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*[]models.OrganizationMemberWithOrganizationAndRole), args.Error(1)
}

func (m *MockOrganizationMembersRepository) ListByOrganizationID(ctx context.Context, organizationID string) (*[]models.OrganizationMemberWithUser, error) {
	args := m.Called(ctx, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*[]models.OrganizationMemberWithUser), args.Error(1)
}

func (m *MockOrganizationMembersRepository) ListUserEmailsForRole(ctx context.Context, organizationID, roleID string) ([]string, error) {
	args := m.Called(ctx, organizationID, roleID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}
