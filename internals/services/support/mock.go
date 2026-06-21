package support_service

import (
	"context"

	"github.com/stretchr/testify/mock"
	chi_types "github.com/yca-software/2chi-go-types"
)

type MockService struct {
	mock.Mock
}

func (m *MockService) Submit(ctx context.Context, req *SubmitSupportRequest, access *chi_types.AccessInfo) error {
	args := m.Called(ctx, req, access)
	return args.Error(0)
}
