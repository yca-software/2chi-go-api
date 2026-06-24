package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	user_handlers "github.com/yca-software/2chi-go-api/internals/handlers/user"
	"github.com/yca-software/2chi-go-api/internals/models"
	user_service "github.com/yca-software/2chi-go-api/internals/services/user"
	chi_archive "github.com/yca-software/2chi-go-archive"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_logger "github.com/yca-software/2chi-go-logger"
	chi_types "github.com/yca-software/2chi-go-types"
)

const (
	testUserID       = "11111111-1111-4111-8111-111111111101"
	testRefreshToken = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaa601"
)

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

func (s *UsersHandlerSuite) patchJSON(path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPatch, path, strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	s.echo.ServeHTTP(rec, req)
	return rec
}

func (s *UsersHandlerSuite) postJSON(path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	s.echo.ServeHTTP(rec, req)
	return rec
}

func (s *UsersHandlerSuite) delete(path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodDelete, path, nil)
	rec := httptest.NewRecorder()
	s.echo.ServeHTTP(rec, req)
	return rec
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

func (s *UsersHandlerSuite) TestArchiveUser_Success() {
	s.usersService.On("ArchiveUser", mock.Anything, mock.MatchedBy(func(req *user_service.ArchiveUserRequest) bool {
		return req.UserID == testUserID
	}), s.userAccess).Return(nil).Once()

	s.echo.DELETE("/api/v1/users/:id", s.handler.ArchiveUser, s.withAccess)
	rec := s.delete("/api/v1/users/me")

	s.Equal(http.StatusNoContent, rec.Code)
	s.usersService.AssertExpectations(s.T())
}

func (s *UsersHandlerSuite) TestRestoreUser_Success() {
	s.usersService.On("RestoreUser", mock.Anything, mock.MatchedBy(func(req *user_service.RestoreUserRequest) bool {
		return req.UserID == testUserID
	}), s.userAccess).Return(&models.User{}, nil).Once()

	s.echo.POST("/api/v1/users/:id/restore", s.handler.RestoreUser, s.withAccess)
	rec := s.postJSON("/api/v1/users/me/restore", `{}`)

	s.Equal(http.StatusOK, rec.Code)
	s.usersService.AssertExpectations(s.T())
}

func (s *UsersHandlerSuite) TestAcceptTerms_Success() {
	s.usersService.On("AcceptTerms", mock.Anything, mock.MatchedBy(func(req *user_service.AcceptTermsRequest) bool {
		return req.UserID == testUserID && req.TermsVersion == "1.0.0" && req.PrivacyPolicyVersion == "1.0.0"
	}), s.userAccess).Return(&user_service.UserProfile{}, nil).Once()

	s.echo.PATCH("/api/v1/users/:id/terms", s.handler.AcceptTerms, s.withAccess)
	rec := s.patchJSON("/api/v1/users/me/terms", `{"termsVersion":"1.0.0","privacyPolicyVersion":"1.0.0"}`)

	s.Equal(http.StatusOK, rec.Code)
	s.usersService.AssertExpectations(s.T())
}

func (s *UsersHandlerSuite) TestChangePassword_Success() {
	s.usersService.On("ChangePassword", mock.Anything, mock.MatchedBy(func(req *user_service.ChangePasswordRequest) bool {
		return req.UserID == testUserID
	}), s.userAccess).Return(nil).Once()

	s.echo.PATCH("/api/v1/users/:id/password", s.handler.ChangePassword, s.withAccess)
	rec := s.patchJSON("/api/v1/users/me/password", `{"oldPassword":"old","newPassword":"newpassword1"}`)

	s.Equal(http.StatusNoContent, rec.Code)
	s.usersService.AssertExpectations(s.T())
}

func (s *UsersHandlerSuite) TestUpdateProfile_Success() {
	s.usersService.On("UpdateProfile", mock.Anything, mock.MatchedBy(func(req *user_service.UpdateProfileRequest) bool {
		return req.UserID == testUserID && req.FirstName == "Jane"
	}), s.userAccess).Return(&models.User{FirstName: "Jane"}, nil).Once()

	s.echo.PATCH("/api/v1/users/:id/profile", s.handler.UpdateProfile, s.withAccess)
	rec := s.patchJSON("/api/v1/users/me/profile", `{"firstName":"Jane","lastName":"Doe"}`)

	s.Equal(http.StatusOK, rec.Code)
	s.usersService.AssertExpectations(s.T())
}

func (s *UsersHandlerSuite) TestUpdateLanguage_Success() {
	s.usersService.On("UpdateLanguage", mock.Anything, mock.MatchedBy(func(req *user_service.UpdateLanguageRequest) bool {
		return req.UserID == testUserID && req.Language == "nb"
	}), s.userAccess).Return(&models.User{Language: "nb"}, nil).Once()

	s.echo.PATCH("/api/v1/users/:id/language", s.handler.UpdateLanguage, s.withAccess)
	rec := s.patchJSON("/api/v1/users/me/language", `{"language":"nb"}`)

	s.Equal(http.StatusOK, rec.Code)
	s.usersService.AssertExpectations(s.T())
}

func (s *UsersHandlerSuite) TestResendVerificationEmail_Success() {
	s.usersService.On("ResendVerificationEmail", mock.Anything, mock.MatchedBy(func(req *user_service.ResendVerificationEmailRequest) bool {
		return req.UserID == testUserID
	}), s.userAccess).Return(nil).Once()

	s.echo.POST("/api/v1/users/:id/resend-verification-email", s.handler.ResendVerificationEmail, s.withAccess)
	rec := s.postJSON("/api/v1/users/me/resend-verification-email", `{}`)

	s.Equal(http.StatusNoContent, rec.Code)
	s.usersService.AssertExpectations(s.T())
}

func (s *UsersHandlerSuite) TestListUserActiveRefreshTokens_Success() {
	tokens := []models.UserRefreshToken{{TokenHash: "hash"}}
	s.usersService.On("ListUserActiveRefreshTokens", mock.Anything, mock.MatchedBy(func(req *user_service.ListUserActiveRefreshTokensRequest) bool {
		return req.UserID == testUserID
	}), s.userAccess).Return(&tokens, nil).Once()

	s.echo.GET("/api/v1/users/:id/refresh-tokens", s.handler.ListUserActiveRefreshTokens, s.withAccess)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me/refresh-tokens", nil)
	rec := httptest.NewRecorder()
	s.echo.ServeHTTP(rec, req)

	s.Equal(http.StatusOK, rec.Code)
	s.usersService.AssertExpectations(s.T())
}

func (s *UsersHandlerSuite) TestRevokeUserRefreshToken_Success() {
	s.usersService.On("RevokeUserRefreshToken", mock.Anything, mock.MatchedBy(func(req *user_service.RevokeUserRefreshTokenRequest) bool {
		return req.UserID == testUserID && req.RefreshTokenID == testRefreshToken
	}), s.userAccess).Return(nil).Once()

	s.echo.DELETE("/api/v1/users/:id/refresh-tokens/:tokenId", s.handler.RevokeUserRefreshToken, s.withAccess)
	rec := s.delete("/api/v1/users/me/refresh-tokens/" + testRefreshToken)

	s.Equal(http.StatusNoContent, rec.Code)
	s.usersService.AssertExpectations(s.T())
}

func (s *UsersHandlerSuite) TestRevokeUserAllRefreshTokens_Success() {
	s.usersService.On("RevokeUserAllRefreshTokens", mock.Anything, mock.MatchedBy(func(req *user_service.RevokeUserAllRefreshTokensRequest) bool {
		return req.UserID == testUserID
	}), s.userAccess).Return(nil).Once()

	s.echo.DELETE("/api/v1/users/:id/refresh-tokens", s.handler.RevokeUserAllRefreshTokens, s.withAccess)
	rec := s.delete("/api/v1/users/me/refresh-tokens")

	s.Equal(http.StatusNoContent, rec.Code)
	s.usersService.AssertExpectations(s.T())
}
