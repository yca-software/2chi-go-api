package api_key_service

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/yca-software/2chi-go-api/internals/models"
	chi_types "github.com/yca-software/2chi-go-types"
)

type MockService struct {
	mock.Mock
}

func (m *MockService) CreateAPIKey(ctx context.Context, req *CreateAPIKeyRequest, access *chi_types.AccessInfo) (*CreateAPIKeyResponse, error) {
	args := m.Called(ctx, req, access)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*CreateAPIKeyResponse), args.Error(1)
}

func (m *MockService) UpdateAPIKey(ctx context.Context, req *UpdateAPIKeyRequest, access *chi_types.AccessInfo) (*models.APIKey, error) {
	args := m.Called(ctx, req, access)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.APIKey), args.Error(1)
}

func (m *MockService) DeleteAPIKey(ctx context.Context, req *DeleteAPIKeyRequest, access *chi_types.AccessInfo) error {
	args := m.Called(ctx, req, access)
	return args.Error(0)
}

func (m *MockService) ListAPIKeys(ctx context.Context, req *ListAPIKeysRequest, access *chi_types.AccessInfo) (*[]models.APIKey, error) {
	args := m.Called(ctx, req, access)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*[]models.APIKey), args.Error(1)
}
