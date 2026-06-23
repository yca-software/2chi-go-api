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

func (m *MockService) Create(ctx context.Context, req *CreateRequest, access *chi_types.AccessInfo) (*CreateResponse, error) {
	args := m.Called(ctx, req, access)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*CreateResponse), args.Error(1)
}

func (m *MockService) Update(ctx context.Context, req *UpdateRequest, access *chi_types.AccessInfo) (*models.APIKey, error) {
	args := m.Called(ctx, req, access)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.APIKey), args.Error(1)
}

func (m *MockService) Delete(ctx context.Context, req *DeleteRequest, access *chi_types.AccessInfo) error {
	args := m.Called(ctx, req, access)
	return args.Error(0)
}

func (m *MockService) List(ctx context.Context, req *ListRequest, access *chi_types.AccessInfo) (*[]models.APIKey, error) {
	args := m.Called(ctx, req, access)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*[]models.APIKey), args.Error(1)
}
