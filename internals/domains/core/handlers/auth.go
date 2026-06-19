package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/yca-software/2chi-go-api/internals/config"
	auth_service "github.com/yca-software/2chi-go-api/internals/domains/core/services/auth"
	platform_http "github.com/yca-software/2chi-go-api/internals/platform/http"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_logger "github.com/yca-software/2chi-go-logger"
	chi_ratelimit "github.com/yca-software/2chi-go-ratelimit"
	chi_server "github.com/yca-software/2chi-go-server"
	chi_types "github.com/yca-software/2chi-go-types"
)

type AuthHandler struct {
	authService auth_service.Service
	cfg         *config.Config
	logger      chi_logger.Logger
}

func NewAuthHandler(authService auth_service.Service, cfg *config.Config, logger chi_logger.Logger) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		cfg:         cfg,
		logger:      logger,
	}
}

func (h *AuthHandler) writeAuthResponse(c echo.Context, status int, accessToken, refreshToken string) error {
	return platform_http.WriteAuthTokenResponse(c, h.cfg.App.Name, h.cfg.App.Environment, h.cfg.App.CookieDomain, status, accessToken, refreshToken)
}

func (h *AuthHandler) RegisterRoutes(e *echo.Echo, authMiddleware echo.MiddlewareFunc, rateLimiter *chi_ratelimit.RateLimiter) {
	authV1 := e.Group("/api/v1/auth")

	authRateLimit := rateLimiter.IPRateLimit("8-M")
	passwordResetRateLimit := rateLimiter.IPRateLimit("6-H")

	authV1.POST("/oauth/google", h.AuthenticateWithGoogle, authRateLimit)
	authV1.POST("/login", h.AuthenticateWithPassword, authRateLimit)
	authV1.POST("/logout", h.Logout, authRateLimit, authMiddleware)
	authV1.POST("/forgot-password", h.ForgotPassword, passwordResetRateLimit)
	authV1.POST("/refresh", h.RefreshAccessToken, authRateLimit)
	authV1.POST("/reset-password", h.ResetPassword, passwordResetRateLimit)
	authV1.POST("/signup", h.SignUp, authRateLimit)
	authV1.POST("/verify-email", h.VerifyEmail, authRateLimit)
}

// AuthenticateWithGoogle godoc
// @Summary      Authenticate user with Google OAuth
// @Description  Login or register with Google OAuth, returning access and refresh tokens
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      auth_service.AuthenticateWithGoogleRequest  true  "Google OAuth request"
// @Success      200      {object}  auth_service.AuthenticateResponse
// @Failure      400      {object}  error.ErrorResponse
// @Failure      401      {object}  error.ErrorResponse
// @Failure      422      {object}  error.ErrorResponse
// @Failure      429      {object}  error.ErrorResponse
// @Failure      500      {object}  error.ErrorResponse
// @Router       /api/v1/auth/oauth/google [post]
func (h *AuthHandler) AuthenticateWithGoogle(c echo.Context) error {
	var req auth_service.AuthenticateWithGoogleRequest
	if err := c.Bind(&req); err != nil {
		return chi_error.NewBadRequestError(err, "InvalidRequestBody", nil)
	}

	req.UserAgent = c.Request().UserAgent()
	req.IPAddress = c.RealIP()
	if req.Language == "" {
		req.Language = platform_http.RequestLanguage(c)
	}

	resp, err := h.authService.AuthenticateWithGoogle(c.Request().Context(), &req)
	if err != nil {
		return err
	}
	return h.writeAuthResponse(c, http.StatusOK, resp.AccessToken, resp.RefreshToken)
}

// AuthenticateWithPassword godoc
// @Summary      Authenticate user with email and password
// @Description  Login with email and password to receive access and refresh tokens
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      auth_service.AuthenticateWithPasswordRequest  true  "Login request"
// @Success      200      {object}  auth_service.AuthenticateResponse
// @Failure      400      {object}  error.ErrorResponse
// @Failure      401      {object}  error.ErrorResponse
// @Failure      404      {object}  error.ErrorResponse
// @Failure      429      {object}  error.ErrorResponse
// @Failure      422      {object}  error.ErrorResponse
// @Failure      500      {object}  error.ErrorResponse
// @Router       /api/v1/auth/login [post]
func (h *AuthHandler) AuthenticateWithPassword(c echo.Context) error {
	var req auth_service.AuthenticateWithPasswordRequest
	if err := c.Bind(&req); err != nil {
		return chi_error.NewBadRequestError(err, "InvalidRequestBody", nil)
	}

	req.UserAgent = c.Request().UserAgent()
	req.IPAddress = c.RealIP()

	resp, err := h.authService.AuthenticateWithPassword(c.Request().Context(), &req)
	if err != nil {
		return err
	}
	return h.writeAuthResponse(c, http.StatusOK, resp.AccessToken, resp.RefreshToken)
}

// ForgotPassword godoc
// @Summary      Request password reset
// @Description  Send a password reset email to the user
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      auth_service.ForgotPasswordRequest  true  "Forgot password request"
// @Success      204      "No Content"
// @Failure      400      {object}  error.ErrorResponse
// @Failure      404      {object}  error.ErrorResponse
// @Failure      429      {object}  error.ErrorResponse
// @Failure      500      {object}  error.ErrorResponse
// @Router       /api/v1/auth/forgot-password [post]
func (h *AuthHandler) ForgotPassword(c echo.Context) error {
	var req auth_service.ForgotPasswordRequest
	if err := c.Bind(&req); err != nil {
		return chi_error.NewBadRequestError(err, "InvalidRequestBody", nil)
	}

	req.Language = platform_http.RequestLanguage(c)

	if err := h.authService.ForgotPassword(c.Request().Context(), &req); err != nil {
		return err
	}

	return c.NoContent(http.StatusNoContent)
}

// Logout godoc
// @Summary      Logout user
// @Description  Invalidate the refresh token to log out the user
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      auth_service.LogoutRequest  true  "Logout request"
// @Success      204      "No Content"
// @Failure      400      {object}  error.ErrorResponse
// @Failure      401      {object}  error.ErrorResponse
// @Failure      403      {object}  error.ErrorResponse
// @Failure      404      {object}  error.ErrorResponse
// @Failure      500      {object}  error.ErrorResponse
// @Router       /api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c echo.Context) error {
	ctx, accessInfo, err := userContext(c)
	if err != nil {
		return err
	}

	var req auth_service.LogoutRequest
	_ = c.Bind(&req)
	refreshToken, err := platform_http.ResolveRefreshTokenFromRequest(c, h.cfg.App.Name, h.cfg.App.Environment, req.RefreshToken)
	if err != nil {
		return err
	}
	req.RefreshToken = refreshToken
	if req.RefreshToken == "" {
		return chi_error.NewBadRequestError(errors.New("refresh token is required"), "InvalidRequestBody", nil)
	}

	if err := h.authService.Logout(ctx, &req, accessInfo); err != nil {
		return err
	}

	platform_http.ClearRefreshTokenCookie(c, h.cfg.App.Name, h.cfg.App.Environment, h.cfg.App.CookieDomain)
	return c.NoContent(http.StatusNoContent)
}

// RefreshAccessToken godoc
// @Summary      Refresh access token
// @Description  Get a new access token using a valid refresh token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      auth_service.RefreshAccessTokenRequest  true  "Refresh token request"
// @Success      200      {object}  auth_service.RefreshAccessTokenResponse
// @Failure      400      {object}  error.ErrorResponse
// @Failure      401      {object}  error.ErrorResponse
// @Failure      404      {object}  error.ErrorResponse
// @Failure      429      {object}  error.ErrorResponse
// @Failure      500      {object}  error.ErrorResponse
// @Router       /api/v1/auth/refresh [post]
func (h *AuthHandler) RefreshAccessToken(c echo.Context) error {
	var req auth_service.RefreshAccessTokenRequest
	_ = c.Bind(&req)
	refreshToken, err := platform_http.ResolveRefreshTokenFromRequest(c, h.cfg.App.Name, h.cfg.App.Environment, req.RefreshToken)
	if err != nil {
		return err
	}
	req.RefreshToken = refreshToken
	if req.RefreshToken == "" {
		return chi_error.NewBadRequestError(errors.New("refresh token is required"), "InvalidRequestBody", nil)
	}

	req.UserAgent = c.Request().UserAgent()
	req.IPAddress = c.RealIP()

	resp, err := h.authService.RefreshAccessToken(c.Request().Context(), &req)
	if err != nil {
		return err
	}
	platform_http.SetRefreshTokenCookie(c, h.cfg.App.Name, h.cfg.App.Environment, h.cfg.App.CookieDomain, req.RefreshToken)
	return c.JSON(http.StatusOK, resp)
}

// ResetPassword godoc
// @Summary      Reset password
// @Description  Reset user password using the reset token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      auth_service.ResetPasswordRequest  true  "Reset password request"
// @Success      204      "No Content"
// @Failure      400      {object}  error.ErrorResponse
// @Failure      401      {object}  error.ErrorResponse
// @Failure      404      {object}  error.ErrorResponse
// @Failure      422      {object}  error.ErrorResponse
// @Failure      429      {object}  error.ErrorResponse
// @Failure      500      {object}  error.ErrorResponse
// @Router       /api/v1/auth/reset-password [post]
func (h *AuthHandler) ResetPassword(c echo.Context) error {
	var req auth_service.ResetPasswordRequest
	if err := c.Bind(&req); err != nil {
		return chi_error.NewBadRequestError(err, "InvalidRequestBody", nil)
	}

	if err := h.authService.ResetPassword(c.Request().Context(), &req); err != nil {
		return err
	}

	return c.NoContent(http.StatusNoContent)
}

// SignUp godoc
// @Summary      Register a new user
// @Description  Create a new user account with email and password
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      auth_service.SignUpRequest  true  "Sign up request"
// @Success      201      {object}  auth_service.SignUpResponse
// @Failure      400      {object}  error.ErrorResponse
// @Failure      409      {object}  error.ErrorResponse
// @Failure      422      {object}  error.ErrorResponse
// @Failure      429      {object}  error.ErrorResponse
// @Failure      500      {object}  error.ErrorResponse
// @Router       /api/v1/auth/signup [post]
func (h *AuthHandler) SignUp(c echo.Context) error {
	var req auth_service.SignUpRequest
	if err := c.Bind(&req); err != nil {
		return chi_error.NewBadRequestError(err, "InvalidRequestBody", nil)
	}

	req.Language = platform_http.RequestLanguage(c)
	req.UserAgent = c.Request().UserAgent()
	req.IPAddress = c.RealIP()

	resp, err := h.authService.SignUp(c.Request().Context(), &req)
	if err != nil {
		return err
	}
	return h.writeAuthResponse(c, http.StatusCreated, resp.AccessToken, resp.RefreshToken)
}

// VerifyEmail godoc
// @Summary      Verify email address
// @Description  Verify user email address using verification token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      auth_service.VerifyEmailRequest  true  "Verify email request"
// @Success      204      "No Content"
// @Failure      400      {object}  error.ErrorResponse
// @Failure      401      {object}  error.ErrorResponse
// @Failure      404      {object}  error.ErrorResponse
// @Failure      422      {object}  error.ErrorResponse
// @Failure      429      {object}  error.ErrorResponse
// @Failure      500      {object}  error.ErrorResponse
// @Router       /api/v1/auth/verify-email [post]
func (h *AuthHandler) VerifyEmail(c echo.Context) error {
	var req auth_service.VerifyEmailRequest
	if err := c.Bind(&req); err != nil {
		return chi_error.NewBadRequestError(err, "InvalidRequestBody", nil)
	}

	if err := h.authService.VerifyEmail(c.Request().Context(), &req); err != nil {
		return err
	}

	return c.NoContent(http.StatusNoContent)
}

func userContext(c echo.Context) (context.Context, *chi_types.AccessInfo, error) {
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
