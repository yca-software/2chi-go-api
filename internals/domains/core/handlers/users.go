package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	user_service "github.com/yca-software/2chi-go-api/internals/domains/core/services/user"
	platform_http "github.com/yca-software/2chi-go-api/internals/platform/http"
	chi_archive "github.com/yca-software/2chi-go-archive"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_logger "github.com/yca-software/2chi-go-logger"
	chi_ratelimit "github.com/yca-software/2chi-go-ratelimit"
	chi_server "github.com/yca-software/2chi-go-server"
	chi_types "github.com/yca-software/2chi-go-types"
)

type UsersHandler struct {
	usersService user_service.Service
	logger       chi_logger.Logger
}

func NewUsersHandler(usersService user_service.Service, logger chi_logger.Logger) *UsersHandler {
	return &UsersHandler{
		usersService: usersService,
		logger:       logger,
	}
}

func (h *UsersHandler) RegisterRoutes(e *echo.Echo, authMiddleware echo.MiddlewareFunc, rateLimiter *chi_ratelimit.RateLimiter) {
	usersV1 := e.Group("/api/v1/users", authMiddleware)
	usersV1.GET("", h.ListUsers)

	usersDetailV1 := usersV1.Group("/:id")
	usersDetailV1.GET("", h.GetUser)
	usersDetailV1.DELETE("", h.ArchiveUser)
	usersDetailV1.POST("/restore", h.RestoreUser)
	usersDetailV1.PATCH("/terms", h.AcceptTerms)
	usersDetailV1.PATCH("/password", h.ChangePassword)
	usersDetailV1.PATCH("/profile", h.UpdateProfile)
	usersDetailV1.PATCH("/language", h.UpdateLanguage)
	usersDetailV1.POST("/resend-verification-email", h.ResendVerificationEmail, rateLimiter.ScopedPrincipalRateLimit("6-H", "resend_verification"))

	usersRefreshTokensV1 := usersDetailV1.Group("/refresh-tokens")
	usersRefreshTokensV1.GET("", h.ListUserActiveRefreshTokens)
	usersRefreshTokensV1.DELETE("/:tokenId", h.RevokeUserRefreshToken)
	usersRefreshTokensV1.DELETE("", h.RevokeUserAllRefreshTokens)
}

/*
* Helpers
 */

func (h *UsersHandler) userContext(c echo.Context) (context.Context, *chi_types.AccessInfo, error) {
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

func (h *UsersHandler) resolveTargetID(c echo.Context, accessInfo *chi_types.AccessInfo) (uuid.UUID, error) {
	idParam := c.Param("id")

	if idParam == "me" {
		return accessInfo.SubjectID, nil
	}

	parsedID, err := uuid.Parse(idParam)
	if err != nil {
		return uuid.Nil, chi_error.NewBadRequestError(err, "InvalidUserIDFormat", nil)
	}

	return parsedID, nil
}

func (h *UsersHandler) userRequestContext(c echo.Context) (context.Context, *chi_types.AccessInfo, string, error) {
	ctx, accessInfo, err := h.userContext(c)
	if err != nil {
		return ctx, nil, "", err
	}

	targetID, err := h.resolveTargetID(c, accessInfo)
	if err != nil {
		return ctx, accessInfo, "", err
	}

	return ctx, accessInfo, targetID.String(), nil
}

/*
* Endpoints
 */

// ListUsers godoc
// @Summary      List users
// @Description  Paginated user search (active or archived scope). Caller must be a platform admin (enforced in service).
// @Tags         user
// @Accept       json
// @Produce      json
// @Param        search   query     string  false  "Search phrase (email, first name, last name)"
// @Param        archive  query     string  false  "Archive scope: active (default) or archived"
// @Param        limit    query     int     false  "Page size (1-100, default 20)"
// @Param        offset   query     int     false  "Offset (default 0)"
// @Success      200      {object}  user_service.ListUsersResponse
// @Failure      400      {object}  error.ErrorResponse
// @Failure      401      {object}  error.ErrorResponse
// @Failure      403      {object}  error.ErrorResponse
// @Failure      500      {object}  error.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/users [get]
func (h *UsersHandler) ListUsers(c echo.Context) error {
	ctx := c.Request().Context()
	accessInfo, err := chi_server.GetAccessInfo(c)
	if err != nil {
		return err
	}

	limit, offset, err := platform_http.ParseLimitOffsetStrict(c, 20, 100)
	if err != nil {
		return err
	}

	filter, err := chi_archive.ParseArchiveFilterQuery(c.QueryParam("archive"))
	if err != nil {
		return err
	}

	resp, err := h.usersService.ListUsers(ctx, &user_service.ListUsersRequest{
		SearchPhrase:  c.QueryParam("search"),
		ArchiveFilter: filter,
		Limit:         limit,
		Offset:        offset,
	}, accessInfo)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, resp)
}

// GetUser godoc
// @Summary      Get user
// @Description  Get user by ID. Use "me" for the authenticated user. Platform admins may pass any user UUID.
// @Tags         user
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "User ID or me"
// @Success      200      {object}  user_service.GetUserResponse
// @Failure      400      {object}  error.ErrorResponse
// @Failure      401      {object}  error.ErrorResponse
// @Failure      403      {object}  error.ErrorResponse
// @Failure      404      {object}  error.ErrorResponse
// @Failure      500      {object}  error.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/users/{id} [get]
func (h *UsersHandler) GetUser(c echo.Context) error {
	ctx, accessInfo, userID, err := h.userRequestContext(c)
	if err != nil {
		return err
	}

	resp, err := h.usersService.GetUser(ctx, &user_service.GetUserRequest{UserID: userID}, accessInfo)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, resp)
}

// ArchiveUser godoc
// @Summary      Archive user account
// @Description  Soft-archives a user account (use "me" for the authenticated user)
// @Tags         user
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "User ID or me"
// @Success      204      {object}  nil
// @Failure      400      {object}  error.ErrorResponse
// @Failure      401      {object}  error.ErrorResponse
// @Failure      403      {object}  error.ErrorResponse
// @Failure      404      {object}  error.ErrorResponse
// @Failure      500      {object}  error.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/users/{id} [delete]
func (h *UsersHandler) ArchiveUser(c echo.Context) error {
	ctx, accessInfo, userID, err := h.userRequestContext(c)
	if err != nil {
		return err
	}

	if err := h.usersService.ArchiveUser(ctx, &user_service.ArchiveUserRequest{UserID: userID}, accessInfo); err != nil {
		return err
	}

	return c.NoContent(http.StatusNoContent)
}

// RestoreUser godoc
// @Summary      Restore archived user account
// @Description  Restores a soft-archived user account (use "me" for the authenticated user)
// @Tags         user
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "User ID or me"
// @Success      200      {object}  models.User
// @Failure      400      {object}  error.ErrorResponse
// @Failure      401      {object}  error.ErrorResponse
// @Failure      403      {object}  error.ErrorResponse
// @Failure      404      {object}  error.ErrorResponse
// @Failure      500      {object}  error.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/users/{id}/restore [post]
func (h *UsersHandler) RestoreUser(c echo.Context) error {
	ctx, accessInfo, userID, err := h.userRequestContext(c)
	if err != nil {
		return err
	}

	user, err := h.usersService.RestoreUser(ctx, &user_service.RestoreUserRequest{UserID: userID}, accessInfo)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, user)
}

// AcceptTerms godoc
// @Summary      Accept terms of service
// @Description  Records acceptance of the current terms of service for a user (use "me" for the authenticated user)
// @Tags         user
// @Accept       json
// @Produce      json
// @Param        id     path      string              true  "User ID or me"
// @Param        body   body      user_service.AcceptTermsRequest  true  "Terms acceptance (termsVersion)"
// @Success      200   {object}  models.User
// @Failure      400   {object}  error.ErrorResponse
// @Failure      401   {object}  error.ErrorResponse
// @Failure      403   {object}  error.ErrorResponse
// @Failure      422   {object}  error.ErrorResponse
// @Failure      500   {object}  error.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/users/{id}/terms [patch]
func (h *UsersHandler) AcceptTerms(c echo.Context) error {
	ctx, accessInfo, userID, err := h.userRequestContext(c)
	if err != nil {
		return err
	}

	var req user_service.AcceptTermsRequest
	if err := c.Bind(&req); err != nil {
		return chi_error.NewBadRequestError(err, "InvalidRequestBody", nil)
	}
	req.UserID = userID

	res, err := h.usersService.AcceptTerms(ctx, &req, accessInfo)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, res)
}

// ChangePassword godoc
// @Summary      Change password
// @Description  Changes the password for a user (use "me" for the authenticated user)
// @Tags         user
// @Accept       json
// @Produce      json
// @Param        id     path      string                 true  "User ID or me"
// @Param        body   body      user_service.ChangePasswordRequest  true  "Password change request"
// @Success      204      {object}  nil
// @Failure      400      {object}  error.ErrorResponse
// @Failure      401      {object}  error.ErrorResponse
// @Failure      403      {object}  error.ErrorResponse
// @Failure      422      {object}  error.ErrorResponse
// @Failure      500      {object}  error.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/users/{id}/password [patch]
func (h *UsersHandler) ChangePassword(c echo.Context) error {
	ctx, accessInfo, userID, err := h.userRequestContext(c)
	if err != nil {
		return err
	}

	var req user_service.ChangePasswordRequest
	if err := c.Bind(&req); err != nil {
		return chi_error.NewBadRequestError(err, "InvalidRequestBody", nil)
	}
	req.UserID = userID

	if err := h.usersService.ChangePassword(ctx, &req, accessInfo); err != nil {
		return err
	}

	return c.NoContent(http.StatusNoContent)
}

// UpdateProfile godoc
// @Summary      Update profile
// @Description  Updates the profile for a user (use "me" for the authenticated user)
// @Tags         user
// @Accept       json
// @Produce      json
// @Param        id     path      string                true  "User ID or me"
// @Param        body   body      user_service.UpdateProfileRequest  true  "Profile update request"
// @Success      200      {object}  models.User
// @Failure      400      {object}  error.ErrorResponse
// @Failure      401      {object}  error.ErrorResponse
// @Failure      403      {object}  error.ErrorResponse
// @Failure      422      {object}  error.ErrorResponse
// @Failure      500      {object}  error.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/users/{id}/profile [patch]
func (h *UsersHandler) UpdateProfile(c echo.Context) error {
	ctx, accessInfo, userID, err := h.userRequestContext(c)
	if err != nil {
		return err
	}

	var req user_service.UpdateProfileRequest
	if err := c.Bind(&req); err != nil {
		return chi_error.NewBadRequestError(err, "InvalidRequestBody", nil)
	}
	req.UserID = userID

	res, err := h.usersService.UpdateProfile(ctx, &req, accessInfo)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, res)
}

// UpdateLanguage godoc
// @Summary      Update profile language
// @Description  Updates the language for a user (use "me" for the authenticated user)
// @Tags         user
// @Accept       json
// @Produce      json
// @Param        id     path      string                 true  "User ID or me"
// @Param        body   body      user_service.UpdateLanguageRequest  true  "Language update request"
// @Success      200      {object}  models.User
// @Failure      400      {object}  error.ErrorResponse
// @Failure      401      {object}  error.ErrorResponse
// @Failure      403      {object}  error.ErrorResponse
// @Failure      422      {object}  error.ErrorResponse
// @Failure      500      {object}  error.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/users/{id}/language [patch]
func (h *UsersHandler) UpdateLanguage(c echo.Context) error {
	ctx, accessInfo, userID, err := h.userRequestContext(c)
	if err != nil {
		return err
	}

	var req user_service.UpdateLanguageRequest
	if err := c.Bind(&req); err != nil {
		return chi_error.NewBadRequestError(err, "InvalidRequestBody", nil)
	}
	req.UserID = userID

	res, err := h.usersService.UpdateLanguage(ctx, &req, accessInfo)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, res)
}

// ResendVerificationEmail godoc
// @Summary      Resend verification email
// @Description  Sends a new email verification link (use "me" for the authenticated user)
// @Tags         user
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "User ID or me"
// @Success      204
// @Failure      400      {object}  error.ErrorResponse
// @Failure      401      {object}  error.ErrorResponse
// @Failure      403      {object}  error.ErrorResponse
// @Failure      409      {object}  error.ErrorResponse
// @Failure      429      {object}  error.ErrorResponse
// @Failure      500      {object}  error.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/users/{id}/resend-verification-email [post]
func (h *UsersHandler) ResendVerificationEmail(c echo.Context) error {
	ctx, accessInfo, userID, err := h.userRequestContext(c)
	if err != nil {
		return err
	}

	if err := h.usersService.ResendVerificationEmail(ctx, &user_service.ResendVerificationEmailRequest{
		UserID:   userID,
		Language: platform_http.RequestLanguage(c),
	}, accessInfo); err != nil {
		return err
	}

	return c.NoContent(http.StatusNoContent)
}

// ListUserActiveRefreshTokens godoc
// @Summary      List active refresh tokens
// @Description  Lists all active refresh tokens for a user (use "me" for the authenticated user)
// @Tags         user, refresh-token
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "User ID or me"
// @Success      200      {array}   models.UserRefreshToken
// @Failure      400      {object}  error.ErrorResponse
// @Failure      401      {object}  error.ErrorResponse
// @Failure      403      {object}  error.ErrorResponse
// @Failure      500      {object}  error.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/users/{id}/refresh-tokens [get]
func (h *UsersHandler) ListUserActiveRefreshTokens(c echo.Context) error {
	ctx, accessInfo, userID, err := h.userRequestContext(c)
	if err != nil {
		return err
	}

	resp, err := h.usersService.ListUserActiveRefreshTokens(ctx, &user_service.ListUserActiveRefreshTokensRequest{UserID: userID}, accessInfo)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, resp)
}

// RevokeUserRefreshToken godoc
// @Summary      Revoke refresh token
// @Description  Revokes a single refresh token for a user (use "me" for the authenticated user)
// @Tags         user, refresh-token
// @Accept       json
// @Produce      json
// @Param        id       path      string  true  "User ID or me"
// @Param        tokenId  path      string  true  "Refresh token ID"
// @Success      204
// @Failure      400      {object}  error.ErrorResponse
// @Failure      401      {object}  error.ErrorResponse
// @Failure      403      {object}  error.ErrorResponse
// @Failure      404      {object}  error.ErrorResponse
// @Failure      500      {object}  error.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/users/{id}/refresh-tokens/{tokenId} [delete]
func (h *UsersHandler) RevokeUserRefreshToken(c echo.Context) error {
	ctx, accessInfo, userID, err := h.userRequestContext(c)
	if err != nil {
		return err
	}

	if err := h.usersService.RevokeUserRefreshToken(ctx, &user_service.RevokeUserRefreshTokenRequest{
		UserID:         userID,
		RefreshTokenID: c.Param("tokenId"),
	}, accessInfo); err != nil {
		return err
	}

	return c.NoContent(http.StatusNoContent)
}

// RevokeUserAllRefreshTokens godoc
// @Summary      Revoke all refresh tokens
// @Description  Revokes all refresh tokens for a user (use "me" for the authenticated user)
// @Tags         user, refresh-token
// @Accept       json
// @Produce      json
// @Param        id     path      string  true  "User ID or me"
// @Param        body   body      user_service.RevokeUserAllRefreshTokensRequest  false  "Optional keepRefreshToken"
// @Success      204
// @Failure      400      {object}  error.ErrorResponse
// @Failure      401      {object}  error.ErrorResponse
// @Failure      403      {object}  error.ErrorResponse
// @Failure      422      {object}  error.ErrorResponse
// @Failure      500      {object}  error.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/users/{id}/refresh-tokens [delete]
func (h *UsersHandler) RevokeUserAllRefreshTokens(c echo.Context) error {
	ctx, accessInfo, userID, err := h.userRequestContext(c)
	if err != nil {
		return err
	}

	var body user_service.RevokeUserAllRefreshTokensRequest
	_ = c.Bind(&body)
	body.UserID = userID

	if err := h.usersService.RevokeUserAllRefreshTokens(ctx, &body, accessInfo); err != nil {
		return err
	}

	return c.NoContent(http.StatusNoContent)
}
