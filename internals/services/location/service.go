package location_service

import (
	"context"
	"errors"

	chi_google "github.com/yca-software/2chi-go-google/maps"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_logger "github.com/yca-software/2chi-go-logger"
)

type Dependencies struct {
	Logger chi_logger.Logger
	Maps   chi_google.Maps
}

type Service interface {
	AutocompleteLocation(ctx context.Context, input string) (*chi_google.AutocompleteLocationResponse, error)
	GetLocationData(ctx context.Context, placeID string) (*chi_google.LocationData, error)
}

type service struct {
	logger chi_logger.Logger
	maps   chi_google.Maps
}

func New(deps Dependencies) Service {
	return &service{
		logger: deps.Logger,
		maps:   deps.Maps,
	}
}

func (s *service) AutocompleteLocation(ctx context.Context, input string) (*chi_google.AutocompleteLocationResponse, error) {
	if s.maps == nil {
		return nil, chi_error.NewServiceUnavailableError(errors.New("maps not configured"), "LocationSearchUnavailable", nil)
	}
	return s.maps.AutocompleteLocation(ctx, input)
}

func (s *service) GetLocationData(ctx context.Context, placeID string) (*chi_google.LocationData, error) {
	if s.maps == nil {
		return nil, chi_error.NewServiceUnavailableError(errors.New("maps not configured"), "LocationSearchUnavailable", nil)
	}
	return s.maps.GetLocationData(ctx, placeID)
}
