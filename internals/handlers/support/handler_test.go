package handlers_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	support_handlers "github.com/yca-software/2chi-go-api/internals/handlers/support"
	support_service "github.com/yca-software/2chi-go-api/internals/services/support"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_logger "github.com/yca-software/2chi-go-logger"
	chi_types "github.com/yca-software/2chi-go-types"
)

const testUserAgent = "TestAgent/1.0"

type SupportHandlerSuite struct {
	suite.Suite
	echo    *echo.Echo
	mockSvc *support_service.MockService
	handler *support_handlers.SupportHandler
	access  *chi_types.AccessInfo
}

func TestSupportHandlerSuite(t *testing.T) {
	suite.Run(t, new(SupportHandlerSuite))
}

func (s *SupportHandlerSuite) SetupTest() {
	s.echo = echo.New()
	s.echo.HTTPErrorHandler = func(err error, c echo.Context) {
		if apiErr, ok := chi_error.AsError(err); ok {
			_ = c.JSON(apiErr.StatusCode, apiErr)
			return
		}
		if httpErr, ok := err.(*echo.HTTPError); ok {
			_ = c.JSON(httpErr.Code, map[string]any{"message": httpErr.Message})
			return
		}
		_ = c.JSON(http.StatusInternalServerError, map[string]any{"message": err.Error()})
	}

	s.mockSvc = &support_service.MockService{}
	s.handler = support_handlers.NewSupportHandler(s.mockSvc, &chi_logger.MockLogger{})
	s.access = &chi_types.AccessInfo{
		Type:      chi_types.AccessTypeUser,
		SubjectID: uuid.MustParse("11111111-1111-4111-8111-111111111101"),
		Email:     "user@example.com",
		IPAddress: "127.0.0.1",
	}
}

func (s *SupportHandlerSuite) registerSubmit(withAccess *chi_types.AccessInfo) {
	s.echo.POST("/api/v1/support", s.handler.Submit, func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if withAccess != nil {
				c.Set("accessInfo", withAccess)
			}
			return next(c)
		}
	})
}

func (s *SupportHandlerSuite) postSupport(body string, userAgent string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/support", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	}
	rec := httptest.NewRecorder()
	s.echo.ServeHTTP(rec, req)
	return rec
}

func (s *SupportHandlerSuite) TestSubmit_MissingAccessInfo() {
	s.registerSubmit(nil)

	rec := s.postSupport(`{"message":"help"}`, testUserAgent)

	s.Equal(http.StatusUnauthorized, rec.Code)
}

func (s *SupportHandlerSuite) TestSubmit_NonUserAccess() {
	s.registerSubmit(&chi_types.AccessInfo{
		Type:      chi_types.AccessTypeAPIKey,
		SubjectID: uuid.MustParse("22222222-2222-4222-8222-222222222202"),
	})

	rec := s.postSupport(`{"message":"help"}`, testUserAgent)

	s.Equal(http.StatusForbidden, rec.Code)

	var body chi_error.Error
	s.Require().NoError(json.Unmarshal(rec.Body.Bytes(), &body))
	s.Equal("UserIdentityRequired", body.ErrorCode)
}

func (s *SupportHandlerSuite) TestSubmit_InvalidBody() {
	s.registerSubmit(s.access)

	rec := s.postSupport(`{invalid`, testUserAgent)

	s.Equal(http.StatusBadRequest, rec.Code)
}

func (s *SupportHandlerSuite) TestSubmit_Success() {
	s.registerSubmit(s.access)
	s.mockSvc.On("Submit", mock.Anything, mock.MatchedBy(func(req *support_service.SubmitSupportRequest) bool {
		return req.Subject == "Billing" &&
			req.Message == "Need help" &&
			req.PageURL == "https://app.example.com/settings" &&
			req.UserAgent == testUserAgent
	}), s.access).Return(nil).Once()

	rec := s.postSupport(
		`{"subject":"Billing","message":"Need help","pageUrl":"https://app.example.com/settings"}`,
		testUserAgent,
	)

	s.Equal(http.StatusNoContent, rec.Code)
	s.Empty(rec.Body.Bytes())
	s.mockSvc.AssertExpectations(s.T())
}

func (s *SupportHandlerSuite) TestSubmit_Success_DefaultSubject() {
	s.registerSubmit(s.access)
	s.mockSvc.On("Submit", mock.Anything, mock.MatchedBy(func(req *support_service.SubmitSupportRequest) bool {
		return req.Subject == "(no subject)" && req.Message == "Need help"
	}), s.access).Return(nil).Once()

	rec := s.postSupport(`{"message":"Need help"}`, testUserAgent)

	s.Equal(http.StatusNoContent, rec.Code)
	s.mockSvc.AssertExpectations(s.T())
}

func (s *SupportHandlerSuite) TestSubmit_ServiceError() {
	s.registerSubmit(s.access)
	s.mockSvc.On("Submit", mock.Anything, mock.Anything, s.access).
		Return(chi_error.NewUnprocessableEntityError(errors.New("validation failed"), "", nil)).
		Once()

	rec := s.postSupport(`{"message":"help"}`, testUserAgent)

	s.Equal(http.StatusUnprocessableEntity, rec.Code)
	s.mockSvc.AssertExpectations(s.T())
}
