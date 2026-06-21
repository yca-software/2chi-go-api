package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/suite"
	"github.com/yca-software/2chi-go-api/internals/handlers/middleware"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_types "github.com/yca-software/2chi-go-types"
)

type AdminMiddlewareSuite struct {
	suite.Suite
	echo *echo.Echo
}

func TestAdminMiddlewareSuite(t *testing.T) {
	suite.Run(t, new(AdminMiddlewareSuite))
}

func (s *AdminMiddlewareSuite) SetupTest() {
	s.echo = echo.New()
	s.echo.HTTPErrorHandler = func(err error, c echo.Context) {
		if apiErr, ok := chi_error.AsError(err); ok {
			_ = c.JSON(apiErr.StatusCode, apiErr)
			return
		}
		_ = c.JSON(http.StatusInternalServerError, map[string]any{"message": err.Error()})
	}
}

func (s *AdminMiddlewareSuite) TestRequirePlatformAdmin_AllowsAdmin() {
	s.echo.GET("/", func(c echo.Context) error { return c.NoContent(http.StatusOK) },
		func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				c.Set("accessInfo", &chi_types.AccessInfo{
					Type:      chi_types.AccessTypeUser,
					SubjectID: uuid.New(),
					IsAdmin:   true,
				})
				return next(c)
			}
		},
		middleware.RequirePlatformAdmin(),
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	s.echo.ServeHTTP(rec, req)

	s.Equal(http.StatusOK, rec.Code)
}

func (s *AdminMiddlewareSuite) TestRequirePlatformAdmin_ForbidsNonAdmin() {
	s.echo.GET("/", func(c echo.Context) error { return c.NoContent(http.StatusOK) },
		func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				c.Set("accessInfo", &chi_types.AccessInfo{
					Type:      chi_types.AccessTypeUser,
					SubjectID: uuid.New(),
					IsAdmin:   false,
				})
				return next(c)
			}
		},
		middleware.RequirePlatformAdmin(),
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	s.echo.ServeHTTP(rec, req)

	s.Equal(http.StatusForbidden, rec.Code)
}
