package team_member_repository

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

func (m *MockRepository) Create(ctx context.Context, member *models.TeamMember) error {
	return m.Called(ctx, member).Error(0)
}

func (m *MockRepository) Delete(ctx context.Context, organizationID, id string) error {
	return m.Called(ctx, organizationID, id).Error(0)
}

func (m *MockRepository) GetByID(ctx context.Context, organizationID, id string) (*models.TeamMember, error) {
	args := m.Called(ctx, organizationID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.TeamMember), args.Error(1)
}

func (m *MockRepository) GetByIDWithUser(ctx context.Context, organizationID, id string) (*models.TeamMemberWithUser, error) {
	args := m.Called(ctx, organizationID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.TeamMemberWithUser), args.Error(1)
}

func (m *MockRepository) ListByUserID(ctx context.Context, userID string) (*[]models.TeamMemberWithTeam, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*[]models.TeamMemberWithTeam), args.Error(1)
}

func (m *MockRepository) ListByTeamID(ctx context.Context, organizationID, teamID string) (*[]models.TeamMemberWithUser, error) {
	args := m.Called(ctx, organizationID, teamID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*[]models.TeamMemberWithUser), args.Error(1)
}

func (m *MockRepository) ListByOrganizationID(ctx context.Context, organizationID string) (*[]models.TeamMemberWithUser, error) {
	args := m.Called(ctx, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*[]models.TeamMemberWithUser), args.Error(1)
}
