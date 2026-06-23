package location_service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
	location_service "github.com/yca-software/2chi-go-api/internals/services/location"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_google "github.com/yca-software/2chi-go-google/maps"
	chi_logger "github.com/yca-software/2chi-go-logger"
)

type LocationServiceSuite struct {
	suite.Suite
	ctx    context.Context
	maps   *chi_google.MockMaps
	svc    location_service.Service
	logger chi_logger.Logger
}

func TestLocationServiceSuite(t *testing.T) {
	suite.Run(t, new(LocationServiceSuite))
}

func (s *LocationServiceSuite) SetupTest() {
	s.ctx = context.Background()
	s.maps = &chi_google.MockMaps{}
	s.logger = &chi_logger.MockLogger{}
	s.svc = location_service.New(location_service.Dependencies{
		Logger: s.logger,
		Maps:   s.maps,
	})
}

func (s *LocationServiceSuite) TestAutocompleteLocation_NotConfigured() {
	svc := location_service.New(location_service.Dependencies{
		Logger: s.logger,
	})
	_, err := svc.AutocompleteLocation(s.ctx, "oslo")
	s.Error(err)
	if apiErr, ok := chi_error.AsError(err); ok {
		s.Equal("LocationSearchUnavailable", apiErr.ErrorCode)
	}
}

func (s *LocationServiceSuite) TestAutocompleteLocation_Success() {
	expected := &chi_google.AutocompleteLocationResponse{}
	s.maps.On("AutocompleteLocation", s.ctx, "oslo").Return(expected, nil).Once()

	resp, err := s.svc.AutocompleteLocation(s.ctx, "oslo")
	s.Require().NoError(err)
	s.Equal(expected, resp)
}

func (s *LocationServiceSuite) TestGetLocationData_E2EStubWhenMapsNotConfigured() {
	svc := location_service.New(location_service.Dependencies{
		Logger: s.logger,
	})
	result, err := svc.GetLocationData(s.ctx, "e2e-local-place")
	s.Require().NoError(err)
	s.Equal("e2e-local-place", result.PlaceID)
	s.Equal("Test City", result.City)
}

func (s *LocationServiceSuite) TestGetLocationData_E2EStubWhenMapsConfigured() {
	result, err := s.svc.GetLocationData(s.ctx, "e2e-local-place")
	s.Require().NoError(err)
	s.Equal("e2e-local-place", result.PlaceID)
	s.maps.AssertNotCalled(s.T(), "GetLocationData")
}

func (s *LocationServiceSuite) TestGetLocationData_NotConfigured() {
	svc := location_service.New(location_service.Dependencies{
		Logger: s.logger,
	})
	_, err := svc.GetLocationData(s.ctx, "place_1")
	s.Error(err)
	if apiErr, ok := chi_error.AsError(err); ok {
		s.Equal("LocationSearchUnavailable", apiErr.ErrorCode)
	}
}

func (s *LocationServiceSuite) TestGetLocationData_PropagatesError() {
	s.maps.On("GetLocationData", s.ctx, "place_1").Return(nil, errors.New("maps down")).Once()

	_, err := s.svc.GetLocationData(s.ctx, "place_1")
	s.Error(err)
}

func (s *LocationServiceSuite) TestGetLocationData_Success() {
	expected := &chi_google.LocationData{PlaceID: "place_1", City: "Oslo"}
	s.maps.On("GetLocationData", s.ctx, "place_1").Return(expected, nil).Once()

	result, err := s.svc.GetLocationData(s.ctx, "place_1")
	s.Require().NoError(err)
	s.Equal(expected, result)
}
