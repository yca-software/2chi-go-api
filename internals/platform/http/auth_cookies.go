package http

import (
	"errors"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/yca-software/2chi-go-api/internals/constants"
	chi_error "github.com/yca-software/2chi-go-error"
)

type tokenCookieNames struct {
	accessToken  string
	refreshToken string
}

func tokenCookieNamesFor(appName, env string) tokenCookieNames {
	prefix := "@" + appName + "-" + env
	return tokenCookieNames{
		accessToken:  prefix + "/access-token",
		refreshToken: prefix + "/refresh-token",
	}
}

func useHttpOnlyRefreshToken(env string) bool {
	return env != "local" && env != "development"
}

func isSecureCookieEnv(env string) bool {
	return env != "local" && env != "development"
}

func SetRefreshTokenCookie(c echo.Context, appName, env, domain, token string) {
	names := tokenCookieNamesFor(appName, env)
	cookie := &http.Cookie{
		Name:     names.refreshToken,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   isSecureCookieEnv(env),
		MaxAge:   int(constants.REFRESH_TOKEN_TTL / time.Second),
		SameSite: http.SameSiteLaxMode,
	}
	if domain != "" {
		cookie.Domain = domain
	}
	c.SetCookie(cookie)
}

func ClearRefreshTokenCookie(c echo.Context, appName, env, domain string) {
	names := tokenCookieNamesFor(appName, env)
	cookie := &http.Cookie{
		Name:     names.refreshToken,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   isSecureCookieEnv(env),
		MaxAge:   -1,
		SameSite: http.SameSiteLaxMode,
	}
	if domain != "" {
		cookie.Domain = domain
	}
	c.SetCookie(cookie)
}

func setAccessTokenCookie(c echo.Context, appName, env, domain, token string) {
	names := tokenCookieNamesFor(appName, env)
	cookie := &http.Cookie{
		Name:     names.accessToken,
		Value:    token,
		Path:     "/",
		HttpOnly: false,
		Secure:   isSecureCookieEnv(env),
		MaxAge:   int(constants.ACCESS_TOKEN_TTL / time.Second),
		SameSite: http.SameSiteLaxMode,
	}
	if domain != "" {
		cookie.Domain = domain
	}
	c.SetCookie(cookie)
}

func WriteAuthTokenResponse(c echo.Context, appName, env, domain string, status int, accessToken, refreshToken string) error {
	setAccessTokenCookie(c, appName, env, domain, accessToken)
	SetRefreshTokenCookie(c, appName, env, domain, refreshToken)

	body := map[string]string{"accessToken": accessToken}
	if !useHttpOnlyRefreshToken(env) {
		body["refreshToken"] = refreshToken
	}
	return c.JSON(status, body)
}

func ResolveRefreshTokenFromRequest(c echo.Context, appName, env, bodyToken string) (string, error) {
	names := tokenCookieNamesFor(appName, env)

	cookieToken := ""
	if cookie, err := c.Cookie(names.refreshToken); err == nil {
		cookieToken = cookie.Value
	}

	if useHttpOnlyRefreshToken(env) {
		if cookieToken == "" {
			return "", chi_error.NewBadRequestError(errors.New("refresh token is required"), "InvalidRequestBody", nil)
		}
		return cookieToken, nil
	}

	if bodyToken != "" {
		return bodyToken, nil
	}
	return cookieToken, nil
}
