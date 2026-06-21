package handler_helpers

import (
	"context"
	"errors"

	"github.com/labstack/echo/v4"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_server "github.com/yca-software/2chi-go-server"
	chi_types "github.com/yca-software/2chi-go-types"
)

func UserContext(c echo.Context) (context.Context, *chi_types.AccessInfo, error) {
	ctx := c.Request().Context()

	accessInfo, err := chi_server.GetAccessInfo(c)
	if err != nil {
		return ctx, nil, err
	}

	if accessInfo.Type != chi_types.AccessTypeUser {
		return ctx, nil, chi_error.NewForbiddenError(errors.New("user identity required"), "UserIdentityRequired", nil)
	}

	return ctx, accessInfo, nil
}

func OrgHandlerContext(c echo.Context) (context.Context, *chi_types.AccessInfo, string, error) {
	ctx := c.Request().Context()
	accessInfo, err := chi_server.GetAccessInfo(c)
	if err != nil {
		return ctx, nil, "", err
	}
	return ctx, accessInfo, c.Param("orgId"), nil
}
