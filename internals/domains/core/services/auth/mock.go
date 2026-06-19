package auth_service

import (
	"context"

	"github.com/stretchr/testify/mock"
	chi_types "github.com/yca-software/2chi-go-types"
)

type MockService struct {
	mock.Mock
}

func (m *MockService) AuthenticateWithGoogle(ctx context.Context, req *AuthenticateWithGoogleRequest) (*AuthenticateResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*AuthenticateResponse), args.Error(1)
}

func (m *MockService) AuthenticateWithPassword(ctx context.Context, req *AuthenticateWithPasswordRequest) (*AuthenticateResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*AuthenticateResponse), args.Error(1)
}

func (m *MockService) ForgotPassword(ctx context.Context, req *ForgotPasswordRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *MockService) Logout(ctx context.Context, req *LogoutRequest, access *chi_types.AccessInfo) error {
	args := m.Called(ctx, req, access)
	return args.Error(0)
}

func (m *MockService) RefreshAccessToken(ctx context.Context, req *RefreshAccessTokenRequest) (*RefreshAccessTokenResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RefreshAccessTokenResponse), args.Error(1)
}

func (m *MockService) ResetPassword(ctx context.Context, req *ResetPasswordRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *MockService) SignUp(ctx context.Context, req *SignUpRequest) (*SignUpResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*SignUpResponse), args.Error(1)
}

func (m *MockService) VerifyEmail(ctx context.Context, req *VerifyEmailRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *MockService) Impersonate(ctx context.Context, req *ImpersonateRequest, access *chi_types.AccessInfo) (*AuthenticateResponse, error) {
	args := m.Called(ctx, req, access)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*AuthenticateResponse), args.Error(1)
}
