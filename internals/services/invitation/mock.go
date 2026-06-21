package invitation_service

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/yca-software/2chi-go-api/internals/models"
	chi_types "github.com/yca-software/2chi-go-types"
)

type MockService struct {
	mock.Mock
}

func (m *MockService) CreateInvitation(ctx context.Context, req *CreateInvitationRequest, access *chi_types.AccessInfo) (*CreateInvitationResponse, error) {
	args := m.Called(ctx, req, access)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*CreateInvitationResponse), args.Error(1)
}

func (m *MockService) RevokeInvitation(ctx context.Context, req *RevokeInvitationRequest, access *chi_types.AccessInfo) error {
	args := m.Called(ctx, req, access)
	return args.Error(0)
}

func (m *MockService) ListInvitations(ctx context.Context, req *ListInvitationsRequest, access *chi_types.AccessInfo) (*[]models.Invitation, error) {
	args := m.Called(ctx, req, access)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*[]models.Invitation), args.Error(1)
}

func (m *MockService) CleanupStaleInvitations(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}
