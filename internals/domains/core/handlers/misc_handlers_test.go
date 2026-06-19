package handlers_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	core_handlers "github.com/yca-software/2chi-go-api/internals/domains/core/handlers"
	billing_service "github.com/yca-software/2chi-go-api/internals/domains/core/services/billing"
	location_service "github.com/yca-software/2chi-go-api/internals/domains/core/services/location"
	support_service "github.com/yca-software/2chi-go-api/internals/domains/core/services/support"
	chi_google "github.com/yca-software/2chi-go-google/maps"
)

type SupportHandlerSuite struct {
	suite.Suite
	echo           *echo.Echo
	supportService *support_service.MockService
	handler        *core_handlers.SupportHandler
}

func TestSupportHandlerSuite(t *testing.T) {
	suite.Run(t, new(SupportHandlerSuite))
}

func (s *SupportHandlerSuite) SetupTest() {
	s.echo = newEchoWithAPIErrors()
	s.supportService = new(support_service.MockService)
	s.handler = core_handlers.NewSupportHandler(s.supportService, testLogger())
}

func (s *SupportHandlerSuite) TestSubmit_Success() {
	s.supportService.On("Submit", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

	s.echo.POST("/api/v1/support", s.handler.Submit, injectAccess(testUserAccess()))
	rec := serve(s.echo, jsonRequest(http.MethodPost, "/api/v1/support", map[string]string{
		"message": "Need help",
	}))

	s.Equal(http.StatusNoContent, rec.Code)
	s.supportService.AssertExpectations(s.T())
}

func (s *SupportHandlerSuite) TestSubmit_DefaultSubject() {
	s.supportService.On("Submit", mock.Anything, mock.MatchedBy(func(req *support_service.SubmitSupportRequest) bool {
		return req.Subject == "(no subject)"
	}), mock.Anything).Return(nil).Once()

	s.echo.POST("/api/v1/support", s.handler.Submit, injectAccess(testUserAccess()))
	rec := serve(s.echo, jsonRequest(http.MethodPost, "/api/v1/support", map[string]string{
		"message": "Need help",
	}))

	s.Equal(http.StatusNoContent, rec.Code)
	s.supportService.AssertExpectations(s.T())
}

type LocationHandlerSuite struct {
	suite.Suite
	echo            *echo.Echo
	locationService *location_service.MockService
	handler         *core_handlers.LocationHandler
}

func TestLocationHandlerSuite(t *testing.T) {
	suite.Run(t, new(LocationHandlerSuite))
}

func (s *LocationHandlerSuite) SetupTest() {
	s.echo = newEchoWithAPIErrors()
	s.locationService = new(location_service.MockService)
	s.handler = core_handlers.NewLocationHandler(s.locationService, testLogger())
}

func (s *LocationHandlerSuite) TestAutocomplete_MissingInput() {
	s.echo.GET("/api/v1/location/autocomplete", s.handler.Autocomplete, injectAccess(testUserAccess()))
	rec := serve(s.echo, jsonRequest(http.MethodGet, "/api/v1/location/autocomplete", nil))

	s.Equal(http.StatusBadRequest, rec.Code)
}

func (s *LocationHandlerSuite) TestAutocomplete_Success() {
	s.locationService.On("AutocompleteLocation", mock.Anything, "London").
		Return(&chi_google.AutocompleteLocationResponse{}, nil).Once()

	s.echo.GET("/api/v1/location/autocomplete", s.handler.Autocomplete, injectAccess(testUserAccess()))
	rec := serve(s.echo, jsonRequest(http.MethodGet, "/api/v1/location/autocomplete?input=London", nil))

	s.Equal(http.StatusOK, rec.Code)
	s.locationService.AssertExpectations(s.T())
}

type BillingWebhookHandlerSuite struct {
	suite.Suite
	echo           *echo.Echo
	billingService *billing_service.MockService
	handler        *core_handlers.BillingWebhookHandler
}

func TestBillingWebhookHandlerSuite(t *testing.T) {
	suite.Run(t, new(BillingWebhookHandlerSuite))
}

func (s *BillingWebhookHandlerSuite) SetupTest() {
	s.echo = newEchoWithAPIErrors()
	s.billingService = new(billing_service.MockService)
	s.handler = core_handlers.NewBillingWebhookHandler(s.billingService, testLogger())
}

func (s *BillingWebhookHandlerSuite) TestHandlePaddleWebhook_Success() {
	payload := []byte(`{"event":"subscription.updated"}`)
	s.billingService.On("HandleWebhook", mock.Anything, payload, "sig").Return(nil).Once()

	s.echo.POST("/api/v1/webhooks/paddle", s.handler.HandlePaddleWebhook)
	req, err := http.NewRequest(http.MethodPost, "/api/v1/webhooks/paddle", strings.NewReader(string(payload)))
	s.Require().NoError(err)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("Paddle-Signature", "sig")
	rec := serve(s.echo, req)

	s.Equal(http.StatusOK, rec.Code)
	s.billingService.AssertExpectations(s.T())
}
