package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/yca-software/2chi-go-api/internals/config"
	handler_helpers "github.com/yca-software/2chi-go-api/internals/handlers/helpers"
	platform_http "github.com/yca-software/2chi-go-api/internals/packages/http"
	auth_service "github.com/yca-software/2chi-go-api/internals/services/auth"
	user_service "github.com/yca-software/2chi-go-api/internals/services/user"
	chi_archive "github.com/yca-software/2chi-go-archive"
	chi_logger "github.com/yca-software/2chi-go-logger"
)

type UsersHandler struct {
	authService  auth_service.Service
	usersService user_service.Service
	cfg          *config.Config
	logger       chi_logger.Logger
}

func NewUsersHandler(
	authService auth_service.Service,
	usersService user_service.Service,
	cfg *config.Config,
	logger chi_logger.Logger,
) *UsersHandler {
	return &UsersHandler{
		authService:  authService,
		usersService: usersService,
		cfg:          cfg,
		logger:       logger,
	}
}

func (h *UsersHandler) RegisterRoutes(admin *echo.Group, impersonateRateLimit echo.MiddlewareFunc) {
	user := admin.Group("/user")
	user.GET("", h.ListUsers)
	user.GET("/:userId", h.GetUser)
	user.DELETE("/:userId", h.DeleteUser)
	user.POST("/:userId/impersonate", h.ImpersonateUser, impersonateRateLimit)
}

// ListUsers godoc
// @Summary      List users (admin)
// @Description  Paginated user search for platform admins (active users only)
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        search  query     string  false  "Search phrase (email, first name, last name)"
// @Param        limit   query     int     false  "Page size (1-100, default 20)"
// @Param        offset  query     int     false  "Offset (default 0)"
// @Success      200     {object}  user_service.ListUsersResponse
// @Failure      400     {object}  error.ErrorResponse
// @Failure      401     {object}  error.ErrorResponse
// @Failure      403     {object}  error.ErrorResponse
// @Failure      500     {object}  error.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/admin/user [get]
func (h *UsersHandler) ListUsers(c echo.Context) error {
	ctx, accessInfo, err := handler_helpers.UserContext(c)
	if err != nil {
		return err
	}
	limit, offset := platform_http.ParseLimitOffset(c, 20, 100)
	resp, err := h.usersService.ListUsers(ctx, &user_service.ListUsersRequest{
		SearchPhrase:  c.QueryParam("search"),
		ArchiveFilter: chi_archive.ArchiveFilterActive,
		Limit:         limit,
		Offset:        offset,
	}, accessInfo)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, resp)
}

// GetUser godoc
// @Summary      Get user (admin)
// @Description  Returns a user by ID (platform admin only)
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        userId  path      string  true  "User ID"
// @Success      200     {object}  user_service.GetUserResponse
// @Failure      400     {object}  error.ErrorResponse
// @Failure      401     {object}  error.ErrorResponse
// @Failure      403     {object}  error.ErrorResponse
// @Failure      404     {object}  error.ErrorResponse
// @Failure      500     {object}  error.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/admin/user/{userId} [get]
func (h *UsersHandler) GetUser(c echo.Context) error {
	ctx, accessInfo, err := handler_helpers.UserContext(c)
	if err != nil {
		return err
	}
	resp, err := h.usersService.GetUser(ctx, &user_service.GetUserRequest{UserID: c.Param("userId")}, accessInfo)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, resp)
}

// DeleteUser godoc
// @Summary      Archive user (admin)
// @Description  Archives a user account (platform admin only)
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        userId  path  string  true  "User ID"
// @Success      204
// @Failure      400  {object}  error.ErrorResponse
// @Failure      401  {object}  error.ErrorResponse
// @Failure      403  {object}  error.ErrorResponse
// @Failure      404  {object}  error.ErrorResponse
// @Failure      500  {object}  error.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/admin/user/{userId} [delete]
func (h *UsersHandler) DeleteUser(c echo.Context) error {
	ctx, accessInfo, err := handler_helpers.UserContext(c)
	if err != nil {
		return err
	}
	if err := h.usersService.ArchiveUser(ctx, &user_service.ArchiveUserRequest{UserID: c.Param("userId")}, accessInfo); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

// ImpersonateUser godoc
// @Summary      Impersonate user (admin)
// @Description  Issues access and refresh tokens for the target user (platform admin only)
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        userId  path      string  true  "User ID"
// @Success      200     {object}  auth_service.AuthenticateResponse
// @Failure      400     {object}  error.ErrorResponse
// @Failure      401     {object}  error.ErrorResponse
// @Failure      403     {object}  error.ErrorResponse
// @Failure      404     {object}  error.ErrorResponse
// @Failure      500     {object}  error.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/admin/user/{userId}/impersonate [post]
func (h *UsersHandler) ImpersonateUser(c echo.Context) error {
	ctx, accessInfo, err := handler_helpers.UserContext(c)
	if err != nil {
		return err
	}
	resp, err := h.authService.Impersonate(ctx, &auth_service.ImpersonateRequest{
		UserID:    c.Param("userId"),
		IPAddress: c.RealIP(),
		UserAgent: c.Request().UserAgent(),
		RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
	}, accessInfo)
	if err != nil {
		return err
	}
	return platform_http.WriteAuthTokenResponse(
		c, h.cfg.App.Name, h.cfg.App.Environment, h.cfg.App.CookieDomain,
		http.StatusOK, resp.AccessToken, resp.RefreshToken,
	)
}
