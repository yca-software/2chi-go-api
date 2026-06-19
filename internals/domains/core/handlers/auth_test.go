package handlers_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	core_handlers "github.com/yca-software/2chi-go-api/internals/domains/core/handlers"
	auth_service "github.com/yca-software/2chi-go-api/internals/domains/core/services/auth"
)

type AuthHandlerSuite struct {
	suite.Suite
	echo        *echo.Echo
	authService *auth_service.MockService
	handler     *core_handlers.AuthHandler
}

func TestAuthHandlerSuite(t *testing.T) {
	suite.Run(t, new(AuthHandlerSuite))
}

func (s *AuthHandlerSuite) SetupTest() {
	s.echo = newEchoWithAPIErrors()
	s.authService = new(auth_service.MockService)
	s.handler = core_handlers.NewAuthHandler(s.authService, testConfig(), testLogger())
}

func (s *AuthHandlerSuite) TestAuthenticateWithPassword_Success() {
	s.authService.On("AuthenticateWithPassword", mock.Anything, mock.Anything).
		Return(&auth_service.AuthenticateResponse{AccessToken: "at", RefreshToken: "rt"}, nil).Once()

	s.echo.POST("/api/v1/auth/login", s.handler.AuthenticateWithPassword)
	rec := serve(s.echo, jsonRequest(http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email":    "user@example.com",
		"password": "secret",
	}))

	s.Equal(http.StatusOK, rec.Code)
	var body map[string]string
	s.Require().NoError(json.Unmarshal(rec.Body.Bytes(), &body))
	s.Equal("at", body["accessToken"])
	s.Equal("rt", body["refreshToken"])
	s.authService.AssertExpectations(s.T())
}

func (s *AuthHandlerSuite) TestAuthenticateWithPassword_InvalidBody() {
	s.echo.POST("/api/v1/auth/login", s.handler.AuthenticateWithPassword)
	rec := serve(s.echo, jsonRequest(http.MethodPost, "/api/v1/auth/login", "not-json"))

	s.Equal(http.StatusBadRequest, rec.Code)
}

func (s *AuthHandlerSuite) TestSignUp_Success() {
	s.authService.On("SignUp", mock.Anything, mock.Anything).
		Return(&auth_service.SignUpResponse{AccessToken: "at", RefreshToken: "rt"}, nil).Once()

	s.echo.POST("/api/v1/auth/signup", s.handler.SignUp)
	rec := serve(s.echo, jsonRequest(http.MethodPost, "/api/v1/auth/signup", map[string]string{
		"email":     "new@example.com",
		"password":  "secret",
		"firstName": "New",
		"lastName":  "User",
	}))

	s.Equal(http.StatusCreated, rec.Code)
	s.authService.AssertExpectations(s.T())
}

func (s *AuthHandlerSuite) TestLogout_RequiresAuth() {
	s.echo.POST("/api/v1/auth/logout", s.handler.Logout)
	rec := serve(s.echo, jsonRequest(http.MethodPost, "/api/v1/auth/logout", map[string]string{
		"refreshToken": "rt",
	}))

	s.Equal(http.StatusUnauthorized, rec.Code)
}

func (s *AuthHandlerSuite) TestLogout_Success() {
	s.authService.On("Logout", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

	s.echo.POST("/api/v1/auth/logout", s.handler.Logout, injectAccess(testUserAccess()))
	rec := serve(s.echo, jsonRequest(http.MethodPost, "/api/v1/auth/logout", map[string]string{
		"refreshToken": "rt",
	}))

	s.Equal(http.StatusNoContent, rec.Code)
	s.authService.AssertExpectations(s.T())
}

func (s *AuthHandlerSuite) TestRefreshAccessToken_MissingToken() {
	s.echo.POST("/api/v1/auth/refresh", s.handler.RefreshAccessToken)
	rec := serve(s.echo, jsonRequest(http.MethodPost, "/api/v1/auth/refresh", map[string]string{}))

	s.Equal(http.StatusBadRequest, rec.Code)
}

func (s *AuthHandlerSuite) TestVerifyEmail_Success() {
	s.authService.On("VerifyEmail", mock.Anything, mock.Anything).Return(nil).Once()

	s.echo.POST("/api/v1/auth/verify-email", s.handler.VerifyEmail)
	rec := serve(s.echo, jsonRequest(http.MethodPost, "/api/v1/auth/verify-email", map[string]string{
		"token": "verify-token",
	}))

	s.Equal(http.StatusNoContent, rec.Code)
	s.authService.AssertExpectations(s.T())
}
