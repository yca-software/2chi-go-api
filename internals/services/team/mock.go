package team_service

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/yca-software/2chi-go-api/internals/models"
	chi_types "github.com/yca-software/2chi-go-types"
)

type MockService struct {
	mock.Mock
}

func (m *MockService) CreateTeam(ctx context.Context, req *CreateTeamRequest, access *chi_types.AccessInfo) (*models.Team, error) {
	args := m.Called(ctx, req, access)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Team), args.Error(1)
}

func (m *MockService) UpdateTeam(ctx context.Context, req *UpdateTeamRequest, access *chi_types.AccessInfo) (*models.Team, error) {
	args := m.Called(ctx, req, access)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Team), args.Error(1)
}

func (m *MockService) DeleteTeam(ctx context.Context, req *DeleteTeamRequest, access *chi_types.AccessInfo) error {
	args := m.Called(ctx, req, access)
	return args.Error(0)
}

func (m *MockService) ListTeams(ctx context.Context, req *ListTeamsRequest, access *chi_types.AccessInfo) (*[]models.Team, error) {
	args := m.Called(ctx, req, access)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*[]models.Team), args.Error(1)
}

func (m *MockService) AddTeamMember(ctx context.Context, req *AddTeamMemberRequest, access *chi_types.AccessInfo) (*models.TeamMemberWithUser, error) {
	args := m.Called(ctx, req, access)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.TeamMemberWithUser), args.Error(1)
}

func (m *MockService) RemoveTeamMember(ctx context.Context, req *RemoveTeamMemberRequest, access *chi_types.AccessInfo) error {
	args := m.Called(ctx, req, access)
	return args.Error(0)
}

func (m *MockService) ListTeamMembers(ctx context.Context, req *ListTeamMembersRequest, access *chi_types.AccessInfo) (*[]models.TeamMemberWithUser, error) {
	args := m.Called(ctx, req, access)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*[]models.TeamMemberWithUser), args.Error(1)
}
