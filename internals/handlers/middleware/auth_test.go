package middleware_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/yca-software/2chi-go-api/internals/handlers/middleware"
	"github.com/yca-software/2chi-go-api/internals/packages/authz"
	user_refresh_token_repository "github.com/yca-software/2chi-go-api/internals/repositories/user_refresh_token"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_types "github.com/yca-software/2chi-go-types"
)

func emptyLoadUserAccessDeps() authz.LoadUserAccessDeps {
	return authz.LoadUserAccessDeps{}
}

type mockPermissionResolver struct {
	userAccess   *chi_types.AccessInfo
	userErr      error
	apiKeyAccess *chi_types.AccessInfo
	apiKeyErr    error
}

func (m *mockPermissionResolver) ResolveUserAccess(_ context.Context, _ string) (*chi_types.AccessInfo, error) {
	return m.userAccess, m.userErr
}

func (m *mockPermissionResolver) ResolveAPIKeyAccess(_ context.Context, _ string) (*chi_types.AccessInfo, error) {
	return m.apiKeyAccess, m.apiKeyErr
}

type AuthMiddlewareSuite struct {
	suite.Suite
	echo   *echo.Echo
	secret string
}

func TestAuthMiddlewareSuite(t *testing.T) {
	suite.Run(t, new(AuthMiddlewareSuite))
}

func (s *AuthMiddlewareSuite) SetupTest() {
	s.echo = echo.New()
	s.echo.HTTPErrorHandler = func(err error, c echo.Context) {
		if apiErr, ok := chi_error.AsError(err); ok {
			_ = c.JSON(apiErr.StatusCode, apiErr)
			return
		}
		_ = c.JSON(http.StatusInternalServerError, map[string]any{"message": err.Error()})
	}
	s.secret = "test-secret-key-at-least-32-bytes-long"
}

func (s *AuthMiddlewareSuite) TestRequireAuth_MissingAuthorization() {
	s.echo.GET("/", func(c echo.Context) error { return c.NoContent(http.StatusOK) },
		middleware.RequireAuth(s.secret, &mockPermissionResolver{}, emptyLoadUserAccessDeps()))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	s.echo.ServeHTTP(rec, req)

	s.Equal(http.StatusUnauthorized, rec.Code)
}

func (s *AuthMiddlewareSuite) TestRequireAuth_ValidBearerToken() {
	userID := uuid.New()
	resolver := &mockPermissionResolver{
		userAccess: &chi_types.AccessInfo{
			Type:      chi_types.AccessTypeUser,
			SubjectID: userID,
		},
	}
	s.echo.GET("/", func(c echo.Context) error {
		access := c.Get("accessInfo").(*chi_types.AccessInfo)
		s.Equal(userID, access.SubjectID)
		return c.NoContent(http.StatusOK)
	}, middleware.RequireAuth(s.secret, resolver, emptyLoadUserAccessDeps()))

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": userID.String(),
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	tokenStr, err := token.SignedString([]byte(s.secret))
	s.Require().NoError(err)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rec := httptest.NewRecorder()
	s.echo.ServeHTTP(rec, req)

	s.Equal(http.StatusOK, rec.Code)
}

func (s *AuthMiddlewareSuite) TestRequireAuth_APIKey() {
	resolver := &mockPermissionResolver{
		apiKeyAccess: &chi_types.AccessInfo{Type: chi_types.AccessTypeAPIKey},
	}
	s.echo.GET("/", func(c echo.Context) error { return c.NoContent(http.StatusOK) },
		middleware.RequireAuth(s.secret, resolver, emptyLoadUserAccessDeps()))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-API-Key", "plain-key")
	rec := httptest.NewRecorder()
	s.echo.ServeHTTP(rec, req)

	s.Equal(http.StatusOK, rec.Code)
}

func (s *AuthMiddlewareSuite) TestRequireAuth_ExpiredToken() {
	s.echo.GET("/", func(c echo.Context) error { return c.NoContent(http.StatusOK) },
		middleware.RequireAuth(s.secret, &mockPermissionResolver{}, emptyLoadUserAccessDeps()))

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": uuid.New().String(),
		"exp": time.Now().Add(-time.Hour).Unix(),
	})
	tokenStr, err := token.SignedString([]byte(s.secret))
	s.Require().NoError(err)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rec := httptest.NewRecorder()
	s.echo.ServeHTTP(rec, req)

	s.Equal(http.StatusUnauthorized, rec.Code)
}

func (s *AuthMiddlewareSuite) TestRequireAuth_RejectsNonHS256Token() {
	s.echo.GET("/", func(c echo.Context) error { return c.NoContent(http.StatusOK) },
		middleware.RequireAuth(s.secret, &mockPermissionResolver{}, emptyLoadUserAccessDeps()))

	token := jwt.NewWithClaims(jwt.SigningMethodHS384, jwt.MapClaims{
		"sub": uuid.New().String(),
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	tokenStr, err := token.SignedString([]byte(s.secret))
	s.Require().NoError(err)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rec := httptest.NewRecorder()
	s.echo.ServeHTTP(rec, req)

	s.Equal(http.StatusUnauthorized, rec.Code)
}

func (s *AuthMiddlewareSuite) TestRequireAuth_PassthroughInvalidTokenFromResolver() {
	resolver := &mockPermissionResolver{
		userErr: chi_error.NewUnauthorizedError(errors.New("session revoked"), "InvalidToken", nil),
	}
	s.echo.GET("/", func(c echo.Context) error { return c.NoContent(http.StatusOK) },
		middleware.RequireAuth(s.secret, resolver, emptyLoadUserAccessDeps()))

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": uuid.New().String(),
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	tokenStr, err := token.SignedString([]byte(s.secret))
	s.Require().NoError(err)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rec := httptest.NewRecorder()
	s.echo.ServeHTTP(rec, req)

	s.Equal(http.StatusUnauthorized, rec.Code)
	var body chi_error.Error
	s.Require().NoError(json.Unmarshal(rec.Body.Bytes(), &body))
	s.Equal("InvalidToken", body.ErrorCode)
}

func (s *AuthMiddlewareSuite) TestRequireAuth_RejectsForgedImpersonationClaims() {
	userID := uuid.MustParse("11111111-1111-4111-8111-111111111101")
	forgedAdminID := uuid.MustParse("44444444-4444-4444-8444-444444444401")

	userRefreshTokensRepo := &user_refresh_token_repository.MockUserRefreshTokenRepository{}
	userRefreshTokensRepo.On("GetActiveImpersonationRefreshTokenByUserID", mock.Anything, userID.String()).
		Return(nil, chi_error.NewNotFoundError(nil, "NotFound", nil))

	resolver := &mockPermissionResolver{
		userAccess: &chi_types.AccessInfo{
			Type:      chi_types.AccessTypeUser,
			SubjectID: userID,
		},
	}
	s.echo.GET("/", func(c echo.Context) error {
		access := c.Get("accessInfo").(*chi_types.AccessInfo)
		s.False(access.ImpersonatedBy.Valid)
		s.Empty(access.ImpersonatedByEmail)
		return c.NoContent(http.StatusOK)
	}, middleware.RequireAuth(s.secret, resolver, authz.LoadUserAccessDeps{
		UserRefreshTokensRepo: userRefreshTokensRepo,
	}))

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":                 userID.String(),
		"exp":                 time.Now().Add(time.Hour).Unix(),
		"impersonatedBy":      forgedAdminID.String(),
		"impersonatedByEmail": "forged@example.com",
	})
	tokenStr, err := token.SignedString([]byte(s.secret))
	s.Require().NoError(err)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rec := httptest.NewRecorder()
	s.echo.ServeHTTP(rec, req)

	s.Equal(http.StatusOK, rec.Code)
	userRefreshTokensRepo.AssertExpectations(s.T())
}

var _ authz.PermissionResolver = (*mockPermissionResolver)(nil)
