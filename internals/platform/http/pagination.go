package http

import (
	"strconv"

	"github.com/labstack/echo/v4"
	chi_error "github.com/yca-software/2chi-go-error"
)

// ParseLimitOffset parses limit and offset query params. Invalid values are ignored;
// limit is capped at maxLimit.
func ParseLimitOffset(c echo.Context, defaultLimit, maxLimit int) (limit, offset int) {
	limit = defaultLimit
	if v := c.QueryParam("limit"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	offset = 0
	if v := c.QueryParam("offset"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed >= 0 {
			offset = parsed
		}
	}
	return limit, offset
}

// ParseLimitOffsetStrict parses limit and offset query params and returns BadRequest
// errors for invalid values.
func ParseLimitOffsetStrict(c echo.Context, defaultLimit, maxLimit int) (limit, offset int, err error) {
	limit = defaultLimit
	if v := c.QueryParam("limit"); v != "" {
		parsed, parseErr := strconv.Atoi(v)
		if parseErr != nil || parsed < 1 || parsed > maxLimit {
			return 0, 0, chi_error.NewBadRequestError(parseErr, "InvalidLimitFormat", nil)
		}
		limit = parsed
	}

	offset = 0
	if v := c.QueryParam("offset"); v != "" {
		parsed, parseErr := strconv.Atoi(v)
		if parseErr != nil || parsed < 0 {
			return 0, 0, chi_error.NewBadRequestError(parseErr, "InvalidOffsetFormat", nil)
		}
		offset = parsed
	}
	return limit, offset, nil
}
