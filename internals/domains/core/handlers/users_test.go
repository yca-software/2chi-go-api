package handlers_test

import (
	"net/http"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	core_handlers "github.com/yca-software/2chi-go-api/internals/domains/core/handlers"
	user_service "github.com/yca-software/2chi-go-api/internals/domains/core/services/user"
)

type UsersHandlerSuite struct {
	suite.Suite
	echo         *echo.Echo
	usersService *user_service.MockService
	handler      *core_handlers.UsersHandler
}

func TestUsersHandlerSuite(t *testing.T) {
	suite.Run(t, new(UsersHandlerSuite))
}

func (s *UsersHandlerSuite) SetupTest() {
	s.echo = newEchoWithAPIErrors()
	s.usersService = new(user_service.MockService)
	s.handler = core_handlers.NewUsersHandler(s.usersService, testLogger())
}

func (s *UsersHandlerSuite) TestListUsers_RequiresAuth() {
	s.echo.GET("/api/v1/users", s.handler.ListUsers)
	rec := serve(s.echo, jsonRequest(http.MethodGet, "/api/v1/users", nil))

	s.Equal(http.StatusUnauthorized, rec.Code)
}

func (s *UsersHandlerSuite) TestListUsers_InvalidLimit() {
	s.echo.GET("/api/v1/users", s.handler.ListUsers, injectAccess(testUserAccess()))
	rec := serve(s.echo, jsonRequest(http.MethodGet, "/api/v1/users?limit=0", nil))

	s.Equal(http.StatusBadRequest, rec.Code)
}

func (s *UsersHandlerSuite) TestListUsers_Success() {
	s.usersService.On("ListUsers", mock.Anything, mock.Anything, mock.Anything).
		Return(&user_service.ListUsersResponse{}, nil).Once()

	s.echo.GET("/api/v1/users", s.handler.ListUsers, injectAccess(testUserAccess()))
	rec := serve(s.echo, jsonRequest(http.MethodGet, "/api/v1/users", nil))

	s.Equal(http.StatusOK, rec.Code)
	s.usersService.AssertExpectations(s.T())
}

func (s *UsersHandlerSuite) TestGetUser_Me() {
	s.usersService.On("GetUser", mock.Anything, mock.MatchedBy(func(req *user_service.GetUserRequest) bool {
		return req.UserID == testUserID.String()
	}), mock.Anything).Return(&user_service.GetUserResponse{}, nil).Once()

	s.echo.GET("/api/v1/users/:id", s.handler.GetUser, injectAccess(testUserAccess()))
	rec := serve(s.echo, jsonRequest(http.MethodGet, "/api/v1/users/me", nil))

	s.Equal(http.StatusOK, rec.Code)
	s.usersService.AssertExpectations(s.T())
}

func (s *UsersHandlerSuite) TestGetUser_InvalidID() {
	s.echo.GET("/api/v1/users/:id", s.handler.GetUser, injectAccess(testUserAccess()))
	rec := serve(s.echo, jsonRequest(http.MethodGet, "/api/v1/users/not-a-uuid", nil))

	s.Equal(http.StatusBadRequest, rec.Code)
}
