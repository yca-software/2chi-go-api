package handlers_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	location_handlers "github.com/yca-software/2chi-go-api/internals/handlers/location"
	location_service "github.com/yca-software/2chi-go-api/internals/services/location"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_google "github.com/yca-software/2chi-go-google/maps"
	chi_logger "github.com/yca-software/2chi-go-logger"
)

type LocationHandlerSuite struct {
	suite.Suite
	echo    *echo.Echo
	mockSvc *location_service.MockService
	handler *location_handlers.LocationHandler
}

func TestLocationHandlerSuite(t *testing.T) {
	suite.Run(t, new(LocationHandlerSuite))
}

func (s *LocationHandlerSuite) SetupTest() {
	s.echo = echo.New()
	s.echo.HTTPErrorHandler = func(err error, c echo.Context) {
		if apiErr, ok := chi_error.AsError(err); ok {
			_ = c.JSON(apiErr.StatusCode, apiErr)
			return
		}
		_ = c.JSON(http.StatusInternalServerError, map[string]any{"message": err.Error()})
	}

	s.mockSvc = &location_service.MockService{}
	s.handler = location_handlers.NewLocationHandler(s.mockSvc, &chi_logger.MockLogger{})
	s.echo.GET("/api/v1/location/autocomplete", s.handler.Autocomplete)
}

func (s *LocationHandlerSuite) TestAutocomplete_MissingInput() {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/location/autocomplete", nil)
	rec := httptest.NewRecorder()
	s.echo.ServeHTTP(rec, req)

	s.Equal(http.StatusBadRequest, rec.Code)

	var body chi_error.Error
	s.Require().NoError(json.Unmarshal(rec.Body.Bytes(), &body))
	s.Equal("InvalidRequestBody", body.ErrorCode)
}

func (s *LocationHandlerSuite) TestAutocomplete_Success() {
	expected := &chi_google.AutocompleteLocationResponse{}
	s.mockSvc.On("AutocompleteLocation", mock.Anything, "oslo").Return(expected, nil).Once()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/location/autocomplete?input=oslo", nil)
	rec := httptest.NewRecorder()
	s.echo.ServeHTTP(rec, req)

	s.Equal(http.StatusOK, rec.Code)

	var body chi_google.AutocompleteLocationResponse
	s.Require().NoError(json.Unmarshal(rec.Body.Bytes(), &body))
	s.Equal(*expected, body)
	s.mockSvc.AssertExpectations(s.T())
}

func (s *LocationHandlerSuite) TestAutocomplete_ServiceError() {
	s.mockSvc.On("AutocompleteLocation", mock.Anything, "oslo").
		Return(nil, chi_error.NewServiceUnavailableError(errors.New("maps down"), "LocationSearchUnavailable", nil)).
		Once()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/location/autocomplete?input=oslo", nil)
	rec := httptest.NewRecorder()
	s.echo.ServeHTTP(rec, req)

	s.Equal(http.StatusServiceUnavailable, rec.Code)

	var body chi_error.Error
	s.Require().NoError(json.Unmarshal(rec.Body.Bytes(), &body))
	s.Equal("LocationSearchUnavailable", body.ErrorCode)
	s.mockSvc.AssertExpectations(s.T())
}
