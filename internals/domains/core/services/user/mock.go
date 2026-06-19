package user_service

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/yca-software/2chi-go-api/internals/domains/core/models"
	chi_types "github.com/yca-software/2chi-go-types"
)

type MockService struct {
	mock.Mock
}

func (m *MockService) AcceptTerms(ctx context.Context, req *AcceptTermsRequest, access *chi_types.AccessInfo) (*models.User, error) {
	args := m.Called(ctx, req, access)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockService) ChangePassword(ctx context.Context, req *ChangePasswordRequest, access *chi_types.AccessInfo) error {
	args := m.Called(ctx, req, access)
	return args.Error(0)
}

func (m *MockService) UpdateProfile(ctx context.Context, req *UpdateProfileRequest, access *chi_types.AccessInfo) (*models.User, error) {
	args := m.Called(ctx, req, access)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockService) UpdateLanguage(ctx context.Context, req *UpdateLanguageRequest, access *chi_types.AccessInfo) (*models.User, error) {
	args := m.Called(ctx, req, access)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockService) ArchiveUser(ctx context.Context, req *ArchiveUserRequest, access *chi_types.AccessInfo) error {
	args := m.Called(ctx, req, access)
	return args.Error(0)
}

func (m *MockService) RestoreUser(ctx context.Context, req *RestoreUserRequest, access *chi_types.AccessInfo) (*models.User, error) {
	args := m.Called(ctx, req, access)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockService) RevokeUserRefreshToken(ctx context.Context, req *RevokeUserRefreshTokenRequest, access *chi_types.AccessInfo) error {
	args := m.Called(ctx, req, access)
	return args.Error(0)
}

func (m *MockService) RevokeUserAllRefreshTokens(ctx context.Context, req *RevokeUserAllRefreshTokensRequest, access *chi_types.AccessInfo) error {
	args := m.Called(ctx, req, access)
	return args.Error(0)
}

func (m *MockService) RevokeUserAdminAccess(ctx context.Context, req *RevokeUserAdminAccessRequest, access *chi_types.AccessInfo) error {
	args := m.Called(ctx, req, access)
	return args.Error(0)
}

func (m *MockService) GetUser(ctx context.Context, req *GetUserRequest, access *chi_types.AccessInfo) (*GetUserResponse, error) {
	args := m.Called(ctx, req, access)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*GetUserResponse), args.Error(1)
}

func (m *MockService) ListUsers(ctx context.Context, req *ListUsersRequest, access *chi_types.AccessInfo) (*ListUsersResponse, error) {
	args := m.Called(ctx, req, access)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ListUsersResponse), args.Error(1)
}

func (m *MockService) ListUserActiveRefreshTokens(ctx context.Context, req *ListUserActiveRefreshTokensRequest, access *chi_types.AccessInfo) (*[]models.UserRefreshToken, error) {
	args := m.Called(ctx, req, access)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*[]models.UserRefreshToken), args.Error(1)
}

func (m *MockService) ResendVerificationEmail(ctx context.Context, req *ResendVerificationEmailRequest, access *chi_types.AccessInfo) error {
	args := m.Called(ctx, req, access)
	return args.Error(0)
}

func (m *MockService) CleanupArchivedUsers(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockService) CleanupStaleUnusedUserTokens(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}
