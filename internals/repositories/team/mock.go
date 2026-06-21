package team_repository

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/yca-software/2chi-go-api/internals/models"
	chi_repository "github.com/yca-software/2chi-go-repository"
)

type MockTeamsRepository struct {
	mock.Mock
}

func (m *MockTeamsRepository) WithTx(_ chi_repository.Tx) TeamsRepository {
	return m
}

func (m *MockTeamsRepository) CreateTeam(ctx context.Context, team *models.Team) error {
	return m.Called(ctx, team).Error(0)
}

func (m *MockTeamsRepository) UpdateTeam(ctx context.Context, team *models.Team) error {
	return m.Called(ctx, team).Error(0)
}

func (m *MockTeamsRepository) DeleteTeam(ctx context.Context, organizationID, id string) error {
	return m.Called(ctx, organizationID, id).Error(0)
}

func (m *MockTeamsRepository) GetTeamByID(ctx context.Context, organizationID, id string) (*models.Team, error) {
	args := m.Called(ctx, organizationID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Team), args.Error(1)
}

func (m *MockTeamsRepository) ListTeamsByOrganizationID(ctx context.Context, organizationID string) (*[]models.Team, error) {
	args := m.Called(ctx, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*[]models.Team), args.Error(1)
}
