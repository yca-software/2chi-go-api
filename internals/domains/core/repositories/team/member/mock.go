package team_member_repository

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/yca-software/2chi-go-api/internals/domains/core/models"
	chi_repository "github.com/yca-software/2chi-go-repository"
)

type MockTeamMembersRepository struct {
	mock.Mock
}

func (m *MockTeamMembersRepository) WithTx(_ chi_repository.Tx) TeamMembersRepository {
	return m
}

func (m *MockTeamMembersRepository) CreateTeamMember(ctx context.Context, member *models.TeamMember) error {
	return m.Called(ctx, member).Error(0)
}

func (m *MockTeamMembersRepository) DeleteTeamMember(ctx context.Context, organizationID, id string) error {
	return m.Called(ctx, organizationID, id).Error(0)
}

func (m *MockTeamMembersRepository) GetTeamMemberByID(ctx context.Context, organizationID, id string) (*models.TeamMember, error) {
	args := m.Called(ctx, organizationID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.TeamMember), args.Error(1)
}

func (m *MockTeamMembersRepository) GetTeamMemberByIDWithUser(ctx context.Context, organizationID, id string) (*models.TeamMemberWithUser, error) {
	args := m.Called(ctx, organizationID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.TeamMemberWithUser), args.Error(1)
}

func (m *MockTeamMembersRepository) ListTeamMembersByUserID(ctx context.Context, userID string) (*[]models.TeamMemberWithTeam, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*[]models.TeamMemberWithTeam), args.Error(1)
}

func (m *MockTeamMembersRepository) ListTeamMembersByTeamID(ctx context.Context, organizationID, teamID string) (*[]models.TeamMemberWithUser, error) {
	args := m.Called(ctx, organizationID, teamID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*[]models.TeamMemberWithUser), args.Error(1)
}

func (m *MockTeamMembersRepository) ListTeamMembersByOrganizationID(ctx context.Context, organizationID string) (*[]models.TeamMemberWithUser, error) {
	args := m.Called(ctx, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*[]models.TeamMemberWithUser), args.Error(1)
}
