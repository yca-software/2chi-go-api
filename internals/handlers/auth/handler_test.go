package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/yca-software/2chi-go-api/internals/config"
	auth_handlers "github.com/yca-software/2chi-go-api/internals/handlers/auth"
	auth_service "github.com/yca-software/2chi-go-api/internals/services/auth"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_logger "github.com/yca-software/2chi-go-logger"
	chi_types "github.com/yca-software/2chi-go-types"
)

type AuthHandlerSuite struct {
	suite.Suite
	echo        *echo.Echo
	authService *auth_service.MockService
	handler     *auth_handlers.AuthHandler
	userAccess  *chi_types.AccessInfo
}

func TestAuthHandlerSuite(t *testing.T) {
	suite.Run(t, new(AuthHandlerSuite))
}

func (s *AuthHandlerSuite) SetupTest() {
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

	s.authService = new(auth_service.MockService)
	cfg := &config.Config{
		App: config.AppConfig{
			Name:         "2chi",
			Environment:  "local",
			CookieDomain: "localhost",
		},
	}
	s.handler = auth_handlers.NewAuthHandler(s.authService, cfg, &chi_logger.MockLogger{})
	s.userAccess = &chi_types.AccessInfo{
		Type:      chi_types.AccessTypeUser,
		SubjectID: uuid.MustParse("11111111-1111-4111-8111-111111111101"),
		Email:     "user@example.com",
	}
}

func (s *AuthHandlerSuite) withAccess(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Set("accessInfo", s.userAccess)
		return next(c)
	}
}

func (s *AuthHandlerSuite) postJSON(path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	s.echo.ServeHTTP(rec, req)
	return rec
}

func (s *AuthHandlerSuite) TestAuthenticateWithPassword_Success() {
	s.authService.On("AuthenticateWithPassword", mock.Anything, mock.Anything).
		Return(&auth_service.AuthenticateResponse{AccessToken: "at", RefreshToken: "rt"}, nil).Once()

	s.echo.POST("/api/v1/auth/login", s.handler.AuthenticateWithPassword)
	rec := s.postJSON("/api/v1/auth/login", `{"email":"user@example.com","password":"secret"}`)

	s.Equal(http.StatusOK, rec.Code)
	var resp map[string]string
	s.Require().NoError(json.Unmarshal(rec.Body.Bytes(), &resp))
	s.Equal("at", resp["accessToken"])
	s.Equal("rt", resp["refreshToken"])
	s.authService.AssertExpectations(s.T())
}

func (s *AuthHandlerSuite) TestAuthenticateWithPassword_InvalidBody() {
	s.echo.POST("/api/v1/auth/login", s.handler.AuthenticateWithPassword)
	rec := s.postJSON("/api/v1/auth/login", "not-json")

	s.Equal(http.StatusBadRequest, rec.Code)
}

func (s *AuthHandlerSuite) TestSignUp_Success() {
	s.authService.On("SignUp", mock.Anything, mock.Anything).
		Return(&auth_service.SignUpResponse{AccessToken: "at", RefreshToken: "rt"}, nil).Once()

	s.echo.POST("/api/v1/auth/signup", s.handler.SignUp)
	rec := s.postJSON("/api/v1/auth/signup", `{"email":"new@example.com","password":"secret","firstName":"New","lastName":"User"}`)

	s.Equal(http.StatusCreated, rec.Code)
	s.authService.AssertExpectations(s.T())
}

func (s *AuthHandlerSuite) TestLogout_RequiresAuth() {
	s.echo.POST("/api/v1/auth/logout", s.handler.Logout)
	rec := s.postJSON("/api/v1/auth/logout", `{"refreshToken":"rt"}`)

	s.Equal(http.StatusUnauthorized, rec.Code)
}

func (s *AuthHandlerSuite) TestLogout_Success() {
	s.authService.On("Logout", mock.Anything, mock.Anything, s.userAccess).Return(nil).Once()

	s.echo.POST("/api/v1/auth/logout", s.handler.Logout, s.withAccess)
	rec := s.postJSON("/api/v1/auth/logout", `{"refreshToken":"rt"}`)

	s.Equal(http.StatusNoContent, rec.Code)
	s.authService.AssertExpectations(s.T())
}

func (s *AuthHandlerSuite) TestRefreshAccessToken_MissingToken() {
	s.echo.POST("/api/v1/auth/refresh", s.handler.RefreshAccessToken)
	rec := s.postJSON("/api/v1/auth/refresh", `{}`)

	s.Equal(http.StatusBadRequest, rec.Code)
}

func (s *AuthHandlerSuite) TestVerifyEmail_Success() {
	s.authService.On("VerifyEmail", mock.Anything, mock.Anything).Return(nil).Once()

	s.echo.POST("/api/v1/auth/verify-email", s.handler.VerifyEmail)
	rec := s.postJSON("/api/v1/auth/verify-email", `{"token":"verify-token"}`)

	s.Equal(http.StatusNoContent, rec.Code)
	s.authService.AssertExpectations(s.T())
}

func (s *AuthHandlerSuite) TestAuthenticateWithGoogle_Success() {
	s.authService.On("AuthenticateWithGoogle", mock.Anything, mock.Anything).
		Return(&auth_service.AuthenticateResponse{AccessToken: "at", RefreshToken: "rt"}, nil).Once()

	s.echo.POST("/api/v1/auth/oauth/google", s.handler.AuthenticateWithGoogle)
	rec := s.postJSON("/api/v1/auth/oauth/google", `{"code":"google-auth-code","termsVersion":"1.0.0","privacyPolicyVersion":"1.0.0","language":"en"}`)

	s.Equal(http.StatusOK, rec.Code)
	s.authService.AssertExpectations(s.T())
}

func (s *AuthHandlerSuite) TestForgotPassword_Success() {
	s.authService.On("ForgotPassword", mock.Anything, mock.Anything).Return(nil).Once()

	s.echo.POST("/api/v1/auth/forgot-password", s.handler.ForgotPassword)
	rec := s.postJSON("/api/v1/auth/forgot-password", `{"email":"user@example.com","language":"en"}`)

	s.Equal(http.StatusNoContent, rec.Code)
	s.authService.AssertExpectations(s.T())
}

func (s *AuthHandlerSuite) TestResetPassword_Success() {
	s.authService.On("ResetPassword", mock.Anything, mock.Anything).Return(nil).Once()

	s.echo.POST("/api/v1/auth/reset-password", s.handler.ResetPassword)
	rec := s.postJSON("/api/v1/auth/reset-password", `{"token":"reset-token","password":"newpassword1"}`)

	s.Equal(http.StatusNoContent, rec.Code)
	s.authService.AssertExpectations(s.T())
}

func (s *AuthHandlerSuite) TestRefreshAccessToken_Success() {
	s.authService.On("RefreshAccessToken", mock.Anything, mock.Anything).
		Return(&auth_service.RefreshAccessTokenResponse{AccessToken: "at"}, nil).Once()

	s.echo.POST("/api/v1/auth/refresh", s.handler.RefreshAccessToken)
	rec := s.postJSON("/api/v1/auth/refresh", `{"refreshToken":"rt"}`)

	s.Equal(http.StatusOK, rec.Code)
	s.authService.AssertExpectations(s.T())
}
