package middleware

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/yca-software/2chi-go-api/internals/platform/authz"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_types "github.com/yca-software/2chi-go-types"
)

func RequireAuth(jwtSecret string, resolver authz.PermissionResolver, loadUserAccess authz.LoadUserAccessDeps) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := c.Request().Context()

			var accessInfo *chi_types.AccessInfo
			var err error

			authHeader := c.Request().Header.Get("Authorization")
			apiKeyHeader := c.Request().Header.Get("X-API-Key")

			if apiKeyHeader != "" {
				accessInfo, err = resolver.ResolveAPIKeyAccess(ctx, apiKeyHeader)
				if err != nil {
					return err
				}
			} else if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
				tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
				token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
					if token.Method != jwt.SigningMethodHS256 {
						return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
					}
					return []byte(jwtSecret), nil
				})
				if err != nil || !token.Valid {
					return chi_error.NewUnauthorizedError(fmt.Errorf("invalid access token: %w", err), "InvalidToken", nil)
				}

				claims, _ := token.Claims.(jwt.MapClaims)
				if exp, ok := claims["exp"].(float64); ok && time.Now().Unix() > int64(exp) {
					return chi_error.NewUnauthorizedError(errors.New("access token has expired"), "ExpiredToken", nil)
				}

				subjectID, _ := claims["sub"].(string)

				accessInfo, err = resolver.ResolveUserAccess(ctx, subjectID)
				if err != nil {
					if apiErr, ok := chi_error.AsError(err); ok && apiErr.ErrorCode == "InvalidToken" {
						return err
					}
					return chi_error.NewUnauthorizedError(fmt.Errorf("invalid session: %w", err), "InvalidSession", nil)
				}

				if err := authz.ApplyValidatedAccessTokenImpersonationClaims(ctx, loadUserAccess, subjectID, accessInfo, claims); err != nil {
					return chi_error.NewUnauthorizedError(fmt.Errorf("invalid session: %w", err), "InvalidSession", nil)
				}
			} else {
				return chi_error.NewUnauthorizedError(errors.New("authorization is required"), "Unauthorized", nil)
			}

			accessInfo.IPAddress = c.RealIP()
			accessInfo.UserAgent = c.Request().UserAgent()
			accessInfo.RequestID = c.Response().Header().Get(echo.HeaderXRequestID)

			c.Set("accessInfo", accessInfo)

			return next(c)
		}
	}
}
