package handlers_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	webhook_handlers "github.com/yca-software/2chi-go-api/internals/handlers/webhook"
	billing_service "github.com/yca-software/2chi-go-api/internals/services/billing"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_logger "github.com/yca-software/2chi-go-logger"
)

const testPaddleSignature = "ts=123;h1=abc"

type BillingWebhookHandlerSuite struct {
	suite.Suite
	echo           *echo.Echo
	billingService *billing_service.MockService
	handler        *webhook_handlers.BillingWebhookHandler
}

func TestBillingWebhookHandlerSuite(t *testing.T) {
	suite.Run(t, new(BillingWebhookHandlerSuite))
}

func (s *BillingWebhookHandlerSuite) SetupTest() {
	s.echo = echo.New()
	s.echo.HTTPErrorHandler = func(err error, c echo.Context) {
		if apiErr, ok := chi_error.AsError(err); ok {
			_ = c.JSON(apiErr.StatusCode, apiErr)
			return
		}
		_ = c.JSON(http.StatusInternalServerError, map[string]any{"message": err.Error()})
	}

	s.billingService = new(billing_service.MockService)
	s.handler = webhook_handlers.NewBillingWebhookHandler(s.billingService, &chi_logger.MockLogger{})
	s.echo.POST("/api/v1/webhooks/paddle", s.handler.HandlePaddleWebhook)
}

func (s *BillingWebhookHandlerSuite) postWebhook(body, signature string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/paddle", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	if signature != "" {
		req.Header.Set("Paddle-Signature", signature)
	}
	rec := httptest.NewRecorder()
	s.echo.ServeHTTP(rec, req)
	return rec
}

func (s *BillingWebhookHandlerSuite) TestHandlePaddleWebhook_Success() {
	payload := `{"event_type":"subscription.created","data":{"id":"sub_123"}}`
	s.billingService.On("HandleWebhook", mock.Anything, []byte(payload), testPaddleSignature).Return(nil).Once()

	rec := s.postWebhook(payload, testPaddleSignature)

	s.Equal(http.StatusOK, rec.Code)
	s.Empty(rec.Body.Bytes())
	s.billingService.AssertExpectations(s.T())
}

func (s *BillingWebhookHandlerSuite) TestHandlePaddleWebhook_ServiceError() {
	payload := `{"event_type":"subscription.created"}`
	s.billingService.On("HandleWebhook", mock.Anything, []byte(payload), testPaddleSignature).
		Return(chi_error.NewUnauthorizedError(errors.New("invalid signature"), "InvalidWebhookSignature", nil)).
		Once()

	rec := s.postWebhook(payload, testPaddleSignature)

	s.Equal(http.StatusUnauthorized, rec.Code)
	s.billingService.AssertExpectations(s.T())
}

func (s *BillingWebhookHandlerSuite) TestRegisterRoutes_SkipsWhenBillingNil() {
	e := echo.New()
	webhook_handlers.NewBillingWebhookHandler(nil, &chi_logger.MockLogger{}).RegisterRoutes(e)

	routes := e.Routes()
	for _, route := range routes {
		s.NotEqual("/api/v1/webhooks/paddle", route.Path)
	}
}
