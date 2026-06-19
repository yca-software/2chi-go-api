package location_service

import (
	"context"

	"github.com/stretchr/testify/mock"
	chi_google "github.com/yca-software/2chi-go-google/maps"
)

type MockService struct {
	mock.Mock
}

func (m *MockService) AutocompleteLocation(ctx context.Context, input string) (*chi_google.AutocompleteLocationResponse, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*chi_google.AutocompleteLocationResponse), args.Error(1)
}

func (m *MockService) GetLocationData(ctx context.Context, placeID string) (*chi_google.LocationData, error) {
	args := m.Called(ctx, placeID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*chi_google.LocationData), args.Error(1)
}
