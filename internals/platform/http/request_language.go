package http

import (
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/yca-software/2chi-go-api/internals/constants"
	platform_i18n "github.com/yca-software/2chi-go-api/internals/platform/i18n"
)

func RequestLanguage(c echo.Context) string {
	header := c.Request().Header.Get("Accept-Language")
	if header == "" {
		return constants.DEFAULT_LANGUAGE
	}

	tag := header
	if idx := strings.IndexAny(header, ",;"); idx >= 0 {
		tag = header[:idx]
	}
	tag = strings.TrimSpace(tag)
	if dash := strings.Index(tag, "-"); dash >= 0 {
		tag = tag[:dash]
	}
	return platform_i18n.NormalizeLanguage(tag)
}
