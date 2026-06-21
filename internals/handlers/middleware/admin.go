package middleware

import (
	"errors"

	"github.com/labstack/echo/v4"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_server "github.com/yca-software/2chi-go-server"
	chi_types "github.com/yca-software/2chi-go-types"
)

func RequirePlatformAdmin() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			accessInfo, err := chi_server.GetAccessInfo(c)
			if err != nil {
				return err
			}
			if accessInfo.Type != chi_types.AccessTypeUser || !accessInfo.IsAdmin {
				return chi_error.NewForbiddenError(errors.New("platform admin required"), "Forbidden", nil)
			}
			return next(c)
		}
	}
}
