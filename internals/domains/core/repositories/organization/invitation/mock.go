package invitation_repository

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/yca-software/2chi-go-api/internals/domains/core/models"
	chi_repository "github.com/yca-software/2chi-go-repository"
)

type MockInvitationsRepository struct {
	mock.Mock
}

func (m *MockInvitationsRepository) WithTx(_ chi_repository.Tx) InvitationsRepository {
	return m
}

func (m *MockInvitationsRepository) CreateInvitation(ctx context.Context, invitation *models.Invitation) error {
	return m.Called(ctx, invitation).Error(0)
}

func (m *MockInvitationsRepository) UpdateInvitation(ctx context.Context, invitation *models.Invitation) error {
	return m.Called(ctx, invitation).Error(0)
}

func (m *MockInvitationsRepository) GetInvitationByID(ctx context.Context, organizationID, id string) (*models.Invitation, error) {
	args := m.Called(ctx, organizationID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Invitation), args.Error(1)
}

func (m *MockInvitationsRepository) GetInvitationByTokenHash(ctx context.Context, tokenHash string) (*models.Invitation, error) {
	args := m.Called(ctx, tokenHash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Invitation), args.Error(1)
}

func (m *MockInvitationsRepository) ListInvitationsByOrganizationID(ctx context.Context, organizationID string) (*[]models.Invitation, error) {
	args := m.Called(ctx, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*[]models.Invitation), args.Error(1)
}

func (m *MockInvitationsRepository) CleanupStaleInvitations(ctx context.Context) error {
	return m.Called(ctx).Error(0)
}
