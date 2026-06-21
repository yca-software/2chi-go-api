package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	user_handlers "github.com/yca-software/2chi-go-api/internals/handlers/user"
	user_service "github.com/yca-software/2chi-go-api/internals/services/user"
	chi_archive "github.com/yca-software/2chi-go-archive"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_logger "github.com/yca-software/2chi-go-logger"
	chi_types "github.com/yca-software/2chi-go-types"
)

const testUserID = "11111111-1111-4111-8111-111111111101"

type UsersHandlerSuite struct {
	suite.Suite
	echo         *echo.Echo
	usersService *user_service.MockService
	handler      *user_handlers.UsersHandler
	userAccess   *chi_types.AccessInfo
}

func TestUsersHandlerSuite(t *testing.T) {
	suite.Run(t, new(UsersHandlerSuite))
}

func (s *UsersHandlerSuite) SetupTest() {
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

	s.usersService = new(user_service.MockService)
	s.handler = user_handlers.NewUsersHandler(s.usersService, &chi_logger.MockLogger{})
	s.userAccess = &chi_types.AccessInfo{
		Type:      chi_types.AccessTypeUser,
		SubjectID: uuid.MustParse(testUserID),
		Email:     "user@example.com",
		IsAdmin:   true,
	}
}

func (s *UsersHandlerSuite) withAccess(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Set("accessInfo", s.userAccess)
		return next(c)
	}
}

func (s *UsersHandlerSuite) TestListUsers_RequiresAuth() {
	s.echo.GET("/api/v1/users", s.handler.ListUsers)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	rec := httptest.NewRecorder()
	s.echo.ServeHTTP(rec, req)

	s.Equal(http.StatusUnauthorized, rec.Code)
}

func (s *UsersHandlerSuite) TestListUsers_InvalidLimit() {
	s.echo.GET("/api/v1/users", s.handler.ListUsers, s.withAccess)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users?limit=0", nil)
	rec := httptest.NewRecorder()
	s.echo.ServeHTTP(rec, req)

	s.Equal(http.StatusBadRequest, rec.Code)
}

func (s *UsersHandlerSuite) TestListUsers_Success() {
	s.usersService.On("ListUsers", mock.Anything, mock.MatchedBy(func(req *user_service.ListUsersRequest) bool {
		return req.ArchiveFilter == chi_archive.ArchiveFilterActive &&
			req.Limit == 20 &&
			req.Offset == 0
	}), s.userAccess).Return(&user_service.ListUsersResponse{}, nil).Once()

	s.echo.GET("/api/v1/users", s.handler.ListUsers, s.withAccess)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	rec := httptest.NewRecorder()
	s.echo.ServeHTTP(rec, req)

	s.Equal(http.StatusOK, rec.Code)
	s.usersService.AssertExpectations(s.T())
}

func (s *UsersHandlerSuite) TestGetUser_Me() {
	s.usersService.On("GetUser", mock.Anything, mock.MatchedBy(func(req *user_service.GetUserRequest) bool {
		return req.UserID == testUserID
	}), s.userAccess).Return(&user_service.GetUserResponse{}, nil).Once()

	s.echo.GET("/api/v1/users/:id", s.handler.GetUser, s.withAccess)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	rec := httptest.NewRecorder()
	s.echo.ServeHTTP(rec, req)

	s.Equal(http.StatusOK, rec.Code)
	s.usersService.AssertExpectations(s.T())
}

func (s *UsersHandlerSuite) TestGetUser_InvalidID() {
	s.echo.GET("/api/v1/users/:id", s.handler.GetUser, s.withAccess)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/not-a-uuid", nil)
	rec := httptest.NewRecorder()
	s.echo.ServeHTTP(rec, req)

	s.Equal(http.StatusBadRequest, rec.Code)
}
