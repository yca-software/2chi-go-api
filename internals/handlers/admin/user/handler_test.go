package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/yca-software/2chi-go-api/internals/config"
	admin_user_handlers "github.com/yca-software/2chi-go-api/internals/handlers/admin/user"
	auth_service "github.com/yca-software/2chi-go-api/internals/services/auth"
	user_service "github.com/yca-software/2chi-go-api/internals/services/user"
	chi_archive "github.com/yca-software/2chi-go-archive"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_logger "github.com/yca-software/2chi-go-logger"
	chi_types "github.com/yca-software/2chi-go-types"
)

const (
	testAdminUserID  = "11111111-1111-4111-8111-111111111101"
	testTargetUserID = "22222222-2222-4222-8222-222222222202"
)

type AdminUsersHandlerSuite struct {
	suite.Suite
	echo         *echo.Echo
	authService  *auth_service.MockService
	usersService *user_service.MockService
	handler      *admin_user_handlers.UsersHandler
	adminAccess  *chi_types.AccessInfo
}

func TestAdminUsersHandlerSuite(t *testing.T) {
	suite.Run(t, new(AdminUsersHandlerSuite))
}

func (s *AdminUsersHandlerSuite) SetupTest() {
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
	s.usersService = new(user_service.MockService)

	cfg := &config.Config{
		App: config.AppConfig{
			Name:         "2chi",
			Environment:  "local",
			CookieDomain: "localhost",
		},
	}

	s.handler = admin_user_handlers.NewUsersHandler(
		s.authService,
		s.usersService,
		cfg,
		&chi_logger.MockLogger{},
	)

	s.adminAccess = &chi_types.AccessInfo{
		Type:      chi_types.AccessTypeUser,
		SubjectID: uuid.MustParse(testAdminUserID),
		Email:     "admin@example.com",
		IsAdmin:   true,
		IPAddress: "127.0.0.1",
	}
}

func (s *AdminUsersHandlerSuite) withAccess(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Set("accessInfo", s.adminAccess)
		return next(c)
	}
}

func (s *AdminUsersHandlerSuite) delete(path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodDelete, path, nil)
	rec := httptest.NewRecorder()
	s.echo.ServeHTTP(rec, req)
	return rec
}

func (s *AdminUsersHandlerSuite) TestListUsers_Success() {
	s.usersService.On("ListUsers", mock.Anything, mock.MatchedBy(func(req *user_service.ListUsersRequest) bool {
		return req.SearchPhrase == "" &&
			req.ArchiveFilter == chi_archive.ArchiveFilterActive &&
			req.Limit == 20 &&
			req.Offset == 0
	}), s.adminAccess).Return(&user_service.ListUsersResponse{}, nil).Once()

	s.echo.GET("/api/v1/admin/user", s.handler.ListUsers, s.withAccess)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/user", nil)
	rec := httptest.NewRecorder()
	s.echo.ServeHTTP(rec, req)

	s.Equal(http.StatusOK, rec.Code)
	s.usersService.AssertExpectations(s.T())
}

func (s *AdminUsersHandlerSuite) TestGetUser_Success() {
	s.usersService.On("GetUser", mock.Anything, mock.MatchedBy(func(req *user_service.GetUserRequest) bool {
		return req.UserID == testTargetUserID
	}), s.adminAccess).Return(&user_service.GetUserResponse{}, nil).Once()

	s.echo.GET("/api/v1/admin/user/:userId", s.handler.GetUser, s.withAccess)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/user/"+testTargetUserID, nil)
	rec := httptest.NewRecorder()
	s.echo.ServeHTTP(rec, req)

	s.Equal(http.StatusOK, rec.Code)
	s.usersService.AssertExpectations(s.T())
}

func (s *AdminUsersHandlerSuite) TestImpersonateUser_Success() {
	s.authService.On("Impersonate", mock.Anything, mock.MatchedBy(func(req *auth_service.ImpersonateRequest) bool {
		return req.UserID == testTargetUserID
	}), s.adminAccess).Return(&auth_service.AuthenticateResponse{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
	}, nil).Once()

	s.echo.POST("/api/v1/admin/user/:userId/impersonate", s.handler.ImpersonateUser, s.withAccess)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/user/"+testTargetUserID+"/impersonate", nil)
	rec := httptest.NewRecorder()
	s.echo.ServeHTTP(rec, req)

	s.Equal(http.StatusOK, rec.Code)

	var body map[string]string
	s.Require().NoError(json.Unmarshal(rec.Body.Bytes(), &body))
	s.Equal("access-token", body["accessToken"])
	s.Equal("refresh-token", body["refreshToken"])
	s.authService.AssertExpectations(s.T())
}

func (s *AdminUsersHandlerSuite) TestDeleteUser_Success() {
	s.usersService.On("ArchiveUser", mock.Anything, mock.MatchedBy(func(req *user_service.ArchiveUserRequest) bool {
		return req.UserID == testTargetUserID
	}), s.adminAccess).Return(nil).Once()

	s.echo.DELETE("/api/v1/admin/user/:userId", s.handler.DeleteUser, s.withAccess)
	rec := s.delete("/api/v1/admin/user/" + testTargetUserID)

	s.Equal(http.StatusNoContent, rec.Code)
	s.usersService.AssertExpectations(s.T())
}
