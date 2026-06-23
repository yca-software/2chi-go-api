package organization_member_repository

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

func (m *MockRepository) Create(ctx context.Context, member *models.OrganizationMember) error {
	return m.Called(ctx, member).Error(0)
}

func (m *MockRepository) Update(ctx context.Context, member *models.OrganizationMember) error {
	return m.Called(ctx, member).Error(0)
}

func (m *MockRepository) DeleteByUserID(ctx context.Context, organizationID, userID string) error {
	return m.Called(ctx, organizationID, userID).Error(0)
}

func (m *MockRepository) DeleteByMemberID(ctx context.Context, organizationID, memberID string) error {
	return m.Called(ctx, organizationID, memberID).Error(0)
}

func (m *MockRepository) GetByUserID(ctx context.Context, organizationID, userID string) (*models.OrganizationMember, error) {
	args := m.Called(ctx, organizationID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.OrganizationMember), args.Error(1)
}

func (m *MockRepository) GetByMemberID(ctx context.Context, organizationID, memberID string) (*models.OrganizationMember, error) {
	args := m.Called(ctx, organizationID, memberID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.OrganizationMember), args.Error(1)
}

func (m *MockRepository) GetByUserIDWithUser(ctx context.Context, organizationID, userID string) (*models.OrganizationMemberWithUser, error) {
	args := m.Called(ctx, organizationID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.OrganizationMemberWithUser), args.Error(1)
}

func (m *MockRepository) GetByMemberIDWithUser(ctx context.Context, organizationID, memberID string) (*models.OrganizationMemberWithUser, error) {
	args := m.Called(ctx, organizationID, memberID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.OrganizationMemberWithUser), args.Error(1)
}

func (m *MockRepository) GetByUserIDWithOrganizationAndRole(ctx context.Context, organizationID, userID string) (*models.OrganizationMemberWithOrganizationAndRole, error) {
	args := m.Called(ctx, organizationID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.OrganizationMemberWithOrganizationAndRole), args.Error(1)
}

func (m *MockRepository) ListByUserID(ctx context.Context, userID string) (*[]models.OrganizationMemberWithOrganizationAndRole, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*[]models.OrganizationMemberWithOrganizationAndRole), args.Error(1)
}

func (m *MockRepository) ListByOrganizationID(ctx context.Context, organizationID string) (*[]models.OrganizationMemberWithUser, error) {
	args := m.Called(ctx, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*[]models.OrganizationMemberWithUser), args.Error(1)
}

func (m *MockRepository) ListUserEmailsForRole(ctx context.Context, organizationID, roleID string) ([]string, error) {
	args := m.Called(ctx, organizationID, roleID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}
