package http_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
	platform_http "github.com/yca-software/2chi-go-api/internals/platform/http"
	chi_error "github.com/yca-software/2chi-go-error"
)

func newEchoContext(query string) echo.Context {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/?"+query, nil)
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec)
}

func TestParseLimitOffset_Defaults(t *testing.T) {
	c := newEchoContext("")
	limit, offset := platform_http.ParseLimitOffset(c, 20, 100)
	require.Equal(t, 20, limit)
	require.Equal(t, 0, offset)
}

func TestParseLimitOffset_ClampsLimit(t *testing.T) {
	c := newEchoContext("limit=500&offset=10")
	limit, offset := platform_http.ParseLimitOffset(c, 20, 100)
	require.Equal(t, 100, limit)
	require.Equal(t, 10, offset)
}

func TestParseLimitOffset_IgnoresInvalid(t *testing.T) {
	c := newEchoContext("limit=abc&offset=-1")
	limit, offset := platform_http.ParseLimitOffset(c, 20, 100)
	require.Equal(t, 20, limit)
	require.Equal(t, 0, offset)
}

func TestParseLimitOffsetStrict_Valid(t *testing.T) {
	c := newEchoContext("limit=50&offset=5")
	limit, offset, err := platform_http.ParseLimitOffsetStrict(c, 20, 100)
	require.NoError(t, err)
	require.Equal(t, 50, limit)
	require.Equal(t, 5, offset)
}

func TestParseLimitOffsetStrict_InvalidLimit(t *testing.T) {
	c := newEchoContext("limit=0")
	_, _, err := platform_http.ParseLimitOffsetStrict(c, 20, 100)
	requireAPIErrorCode(t, err, "InvalidLimitFormat")
}

func TestParseLimitOffsetStrict_InvalidOffset(t *testing.T) {
	c := newEchoContext("offset=-1")
	_, _, err := platform_http.ParseLimitOffsetStrict(c, 20, 100)
	requireAPIErrorCode(t, err, "InvalidOffsetFormat")
}

func requireAPIErrorCode(t *testing.T, err error, code string) {
	t.Helper()
	apiErr, ok := chi_error.AsError(err)
	require.True(t, ok)
	require.Equal(t, code, apiErr.ErrorCode)
}
