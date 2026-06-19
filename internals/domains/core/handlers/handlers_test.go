package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/yca-software/2chi-go-api/internals/config"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_logger "github.com/yca-software/2chi-go-logger"
	chi_types "github.com/yca-software/2chi-go-types"
)

var (
	testUserID       = uuid.MustParse("018f1234-5678-7abc-8def-012345678901")
	testOrgID        = uuid.MustParse("018f1234-5678-7abc-8def-012345678902")
	testRoleID       = uuid.MustParse("018f1234-5678-7abc-8def-012345678903")
	testTeamID       = uuid.MustParse("018f1234-5678-7abc-8def-012345678904")
	testMemberID     = uuid.MustParse("018f1234-5678-7abc-8def-012345678905")
	testInvitationID = uuid.MustParse("018f1234-5678-7abc-8def-012345678906")
	testAPIKeyID     = uuid.MustParse("018f1234-5678-7abc-8def-012345678907")
	testTokenID      = uuid.MustParse("018f1234-5678-7abc-8def-012345678908")
)

func testLogger() chi_logger.Logger {
	return &chi_logger.MockLogger{}
}

func testConfig() *config.Config {
	return &config.Config{
		App: config.AppConfig{
			Name:        "2chi",
			Environment: "local",
		},
	}
}

func testUserAccess() *chi_types.AccessInfo {
	return &chi_types.AccessInfo{
		Type:      chi_types.AccessTypeUser,
		SubjectID: testUserID,
		Email:     "user@example.com",
	}
}

func testAdminAccess() *chi_types.AccessInfo {
	return &chi_types.AccessInfo{
		Type:      chi_types.AccessTypeUser,
		SubjectID: testUserID,
		Email:     "admin@example.com",
		IsAdmin:   true,
	}
}

func newEchoWithAPIErrors() *echo.Echo {
	e := echo.New()
	e.HTTPErrorHandler = func(err error, c echo.Context) {
		if apiErr, ok := chi_error.AsError(err); ok {
			_ = c.JSON(apiErr.StatusCode, apiErr)
			return
		}
		if he, ok := err.(*echo.HTTPError); ok {
			_ = c.JSON(he.Code, map[string]any{"message": he.Message})
			return
		}
		internal := chi_error.NewInternalServerError(err, "", nil)
		_ = c.JSON(internal.StatusCode, internal)
	}
	return e
}

func injectAccess(access *chi_types.AccessInfo) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("accessInfo", access)
			return next(c)
		}
	}
}

func jsonRequest(method, target string, body any) *http.Request {
	var reader *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reader = bytes.NewReader(b)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, target, reader)
	if body != nil {
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	}
	return req
}

func serve(e *echo.Echo, req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}
