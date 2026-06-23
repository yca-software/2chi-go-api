package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/yca-software/2chi-go-api/internals/config"
	handler_helpers "github.com/yca-software/2chi-go-api/internals/handlers/helpers"
	platform_http "github.com/yca-software/2chi-go-api/internals/packages/http"
	audit_service "github.com/yca-software/2chi-go-api/internals/services/audit"
	auth_service "github.com/yca-software/2chi-go-api/internals/services/auth"
	organization_service "github.com/yca-software/2chi-go-api/internals/services/organization"
	user_service "github.com/yca-software/2chi-go-api/internals/services/user"
	chi_archive "github.com/yca-software/2chi-go-archive"
	chi_logger "github.com/yca-software/2chi-go-logger"
)

type AdminHandler struct {
	authService          auth_service.Service
	usersService         user_service.Service
	organizationsService organization_service.Service
	auditLogsService     audit_service.Service
	cfg                  *config.Config
	logger               chi_logger.Logger
}

func NewAdminHandler(
	authService auth_service.Service,
	usersService user_service.Service,
	organizationsService organization_service.Service,
	auditLogsService audit_service.Service,
	cfg *config.Config,
	logger chi_logger.Logger,
) *AdminHandler {
	return &AdminHandler{
		authService:          authService,
		usersService:         usersService,
		organizationsService: organizationsService,
		auditLogsService:     auditLogsService,
		cfg:                  cfg,
		logger:               logger,
	}
}

func (h *AdminHandler) RegisterRoutes(e *echo.Echo, authMiddleware, adminMiddleware, adminRateLimit, impersonateRateLimit echo.MiddlewareFunc) {
	admin := e.Group("/api/v1/admin", authMiddleware, adminMiddleware, adminRateLimit)

	user := admin.Group("/user")
	user.GET("", h.ListUsers)
	user.GET("/:userId", h.GetUser)
	user.DELETE("/:userId", h.DeleteUser)
	user.DELETE("/:userId/admin-access", h.RevokeUserAdminAccess)
	user.POST("/:userId/impersonate", h.ImpersonateUser, impersonateRateLimit)

	org := admin.Group("/organization")
	org.GET("", h.ListOrganizations)
	org.POST("", h.CreateOrganization)
	org.GET("/archived", h.ListArchivedOrganizations)
	org.GET("/archived/:orgId", h.GetArchivedOrganization)
	org.POST("/archived/:orgId/restore", h.RestoreOrganization)
	org.GET("/:orgId", h.GetOrganization)
	org.GET("/:orgId/billing", h.GetOrganizationBillingAccount)
	org.PATCH("/:orgId/subscription", h.UpdateOrganizationSubscription)
	org.GET("/:orgId/audit-log", h.ListOrganizationAuditLogs)
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
func (h *AdminHandler) ListUsers(c echo.Context) error {
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
func (h *AdminHandler) GetUser(c echo.Context) error {
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
func (h *AdminHandler) DeleteUser(c echo.Context) error {
	ctx, accessInfo, err := handler_helpers.UserContext(c)
	if err != nil {
		return err
	}
	if err := h.usersService.ArchiveUser(ctx, &user_service.ArchiveUserRequest{UserID: c.Param("userId")}, accessInfo); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

// RevokeUserAdminAccess godoc
// @Summary      Revoke platform admin access
// @Description  Removes platform admin access for a user and invalidates their cached session (platform admin only)
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
// @Router       /api/v1/admin/user/{userId}/admin-access [delete]
func (h *AdminHandler) RevokeUserAdminAccess(c echo.Context) error {
	ctx, accessInfo, err := handler_helpers.UserContext(c)
	if err != nil {
		return err
	}
	if err := h.usersService.RevokeUserAdminAccess(ctx, &user_service.RevokeUserAdminAccessRequest{
		UserID: c.Param("userId"),
	}, accessInfo); err != nil {
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
func (h *AdminHandler) ImpersonateUser(c echo.Context) error {
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

// ListOrganizations godoc
// @Summary      List organizations (admin)
// @Description  Paginated organization search for platform admins (active organizations only)
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        search  query     string  false  "Search phrase"
// @Param        limit   query     int     false  "Page size (1-100, default 20)"
// @Param        offset  query     int     false  "Offset (default 0)"
// @Success      200     {object}  organization_service.ListOrganizationsResponse
// @Failure      400     {object}  error.ErrorResponse
// @Failure      401     {object}  error.ErrorResponse
// @Failure      403     {object}  error.ErrorResponse
// @Failure      500     {object}  error.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/admin/organization [get]
func (h *AdminHandler) ListOrganizations(c echo.Context) error {
	ctx, accessInfo, err := handler_helpers.UserContext(c)
	if err != nil {
		return err
	}
	limit, offset := platform_http.ParseLimitOffset(c, 20, 100)
	resp, err := h.organizationsService.ListOrganizations(ctx, &organization_service.ListOrganizationsRequest{
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

// ListArchivedOrganizations godoc
// @Summary      List archived organizations (admin)
// @Description  Paginated search of archived organizations (platform admin only)
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        search  query     string  false  "Search phrase"
// @Param        limit   query     int     false  "Page size (1-100, default 20)"
// @Param        offset  query     int     false  "Offset (default 0)"
// @Success      200     {object}  organization_service.ListOrganizationsResponse
// @Failure      400     {object}  error.ErrorResponse
// @Failure      401     {object}  error.ErrorResponse
// @Failure      403     {object}  error.ErrorResponse
// @Failure      500     {object}  error.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/admin/organization/archived [get]
func (h *AdminHandler) ListArchivedOrganizations(c echo.Context) error {
	ctx, accessInfo, err := handler_helpers.UserContext(c)
	if err != nil {
		return err
	}
	limit, offset := platform_http.ParseLimitOffset(c, 20, 100)
	resp, err := h.organizationsService.ListOrganizations(ctx, &organization_service.ListOrganizationsRequest{
		SearchPhrase:  c.QueryParam("search"),
		ArchiveFilter: chi_archive.ArchiveFilterArchived,
		Limit:         limit,
		Offset:        offset,
	}, accessInfo)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, resp)
}

// GetOrganization godoc
// @Summary      Get organization (admin)
// @Description  Returns an active organization by ID (platform admin only)
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        orgId  path      string  true  "Organization ID"
// @Success      200    {object}  models.Organization
// @Failure      400    {object}  error.ErrorResponse
// @Failure      401    {object}  error.ErrorResponse
// @Failure      403    {object}  error.ErrorResponse
// @Failure      404    {object}  error.ErrorResponse
// @Failure      500    {object}  error.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/admin/organization/{orgId} [get]
func (h *AdminHandler) GetOrganization(c echo.Context) error {
	ctx, accessInfo, err := handler_helpers.UserContext(c)
	if err != nil {
		return err
	}
	org, err := h.organizationsService.GetOrganization(ctx, &organization_service.GetOrganizationRequest{
		OrganizationID: c.Param("orgId"),
	}, accessInfo)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, org)
}

// GetOrganizationBillingAccount godoc
// @Summary      Get organization billing account (admin)
// @Description  Returns billing and subscription state for an organization (platform admin only)
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        orgId  path      string  true  "Organization ID"
// @Success      200    {object}  models.OrganizationBillingAccount
// @Failure      400    {object}  error.ErrorResponse
// @Failure      401    {object}  error.ErrorResponse
// @Failure      403    {object}  error.ErrorResponse
// @Failure      404    {object}  error.ErrorResponse
// @Failure      500    {object}  error.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/admin/organization/{orgId}/billing [get]
func (h *AdminHandler) GetOrganizationBillingAccount(c echo.Context) error {
	ctx, accessInfo, err := handler_helpers.UserContext(c)
	if err != nil {
		return err
	}
	account, err := h.organizationsService.GetOrganizationBillingAccount(ctx, &organization_service.GetOrganizationBillingAccountRequest{
		OrganizationID:  c.Param("orgId"),
		IncludeArchived: true,
	}, accessInfo)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, account)
}

// GetArchivedOrganization godoc
// @Summary      Get archived organization (admin)
// @Description  Returns an archived organization by ID (platform admin only)
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        orgId  path      string  true  "Organization ID"
// @Success      200    {object}  models.Organization
// @Failure      400    {object}  error.ErrorResponse
// @Failure      401    {object}  error.ErrorResponse
// @Failure      403    {object}  error.ErrorResponse
// @Failure      404    {object}  error.ErrorResponse
// @Failure      500    {object}  error.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/admin/organization/archived/{orgId} [get]
func (h *AdminHandler) GetArchivedOrganization(c echo.Context) error {
	ctx, accessInfo, err := handler_helpers.UserContext(c)
	if err != nil {
		return err
	}
	org, err := h.organizationsService.GetArchivedOrganization(ctx, &organization_service.GetOrganizationRequest{
		OrganizationID: c.Param("orgId"),
	}, accessInfo)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, org)
}

// CreateOrganization godoc
// @Summary      Create organization (admin)
// @Description  Creates a custom-subscription organization with owner (platform admin only)
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        organization  body      organization_service.AdminCreateOrganizationRequest  true  "Organization request"
// @Success      201           {object}  models.Organization
// @Failure      400           {object}  error.ErrorResponse
// @Failure      401           {object}  error.ErrorResponse
// @Failure      403           {object}  error.ErrorResponse
// @Failure      422           {object}  error.ErrorResponse
// @Failure      500           {object}  error.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/admin/organization [post]
func (h *AdminHandler) CreateOrganization(c echo.Context) error {
	ctx, accessInfo, err := handler_helpers.UserContext(c)
	if err != nil {
		return err
	}
	var req organization_service.AdminCreateOrganizationRequest
	if err := c.Bind(&req); err != nil {
		return err
	}
	org, err := h.organizationsService.AdminCreateOrganization(ctx, &req, accessInfo)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, org)
}

// RestoreOrganization godoc
// @Summary      Restore archived organization (admin)
// @Description  Restores an archived organization (platform admin only)
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        orgId  path      string  true  "Organization ID"
// @Success      200    {object}  models.Organization
// @Failure      400    {object}  error.ErrorResponse
// @Failure      401    {object}  error.ErrorResponse
// @Failure      403    {object}  error.ErrorResponse
// @Failure      404    {object}  error.ErrorResponse
// @Failure      500    {object}  error.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/admin/organization/archived/{orgId}/restore [post]
func (h *AdminHandler) RestoreOrganization(c echo.Context) error {
	ctx, accessInfo, err := handler_helpers.UserContext(c)
	if err != nil {
		return err
	}
	org, err := h.organizationsService.RestoreOrganization(ctx, &organization_service.RestoreOrganizationRequest{
		OrganizationID: c.Param("orgId"),
	}, accessInfo)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, org)
}

// UpdateOrganizationSubscription godoc
// @Summary      Update organization subscription (admin)
// @Description  Updates custom subscription fields for an organization (platform admin only)
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        orgId         path      string                                          true  "Organization ID"
// @Param        subscription  body      organization_service.UpdateOrganizationSubscriptionRequest  true  "Subscription update"
// @Success      200           {object}  models.OrganizationBillingAccount
// @Failure      400           {object}  error.ErrorResponse
// @Failure      401           {object}  error.ErrorResponse
// @Failure      403           {object}  error.ErrorResponse
// @Failure      404           {object}  error.ErrorResponse
// @Failure      422           {object}  error.ErrorResponse
// @Failure      500           {object}  error.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/admin/organization/{orgId}/subscription [patch]
func (h *AdminHandler) UpdateOrganizationSubscription(c echo.Context) error {
	ctx, accessInfo, err := handler_helpers.UserContext(c)
	if err != nil {
		return err
	}
	var req organization_service.UpdateOrganizationSubscriptionRequest
	if err := c.Bind(&req); err != nil {
		return err
	}
	req.OrganizationID = c.Param("orgId")
	billingAccount, err := h.organizationsService.UpdateOrganizationSubscription(ctx, &req, accessInfo)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, billingAccount)
}

// ListOrganizationAuditLogs godoc
// @Summary      List organization audit logs (admin)
// @Description  Paginated audit logs for any organization (platform admin only)
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        orgId      path   string  true   "Organization ID"
// @Param        limit      query  int     false  "Items per page (1-100, default 50)"
// @Param        offset     query  int     false  "Offset (default 0)"
// @Param        startDate  query  string  false  "Start date (RFC3339)"
// @Param        endDate    query  string  false  "End date (RFC3339)"
// @Success      200        {object}  audit_service.ListForOrganizationResponse
// @Failure      400        {object}  error.ErrorResponse
// @Failure      401        {object}  error.ErrorResponse
// @Failure      403        {object}  error.ErrorResponse
// @Failure      422        {object}  error.ErrorResponse
// @Failure      500        {object}  error.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/admin/organization/{orgId}/audit-log [get]
func (h *AdminHandler) ListOrganizationAuditLogs(c echo.Context) error {
	ctx, accessInfo, err := handler_helpers.UserContext(c)
	if err != nil {
		return err
	}
	limit, offset := platform_http.ParseLimitOffset(c, 50, 100)

	filters, err := handler_helpers.ParseAuditLogFilters(c)
	if err != nil {
		return err
	}

	resp, err := h.auditLogsService.ListForOrganization(ctx, &audit_service.ListForOrganizationRequest{
		OrganizationID: c.Param("orgId"),
		Limit:          limit,
		Offset:         offset,
		Filters:        filters,
	}, accessInfo)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, resp)
}
