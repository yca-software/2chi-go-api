package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	handler_helpers "github.com/yca-software/2chi-go-api/internals/handlers/helpers"
	api_key_handler "github.com/yca-software/2chi-go-api/internals/handlers/organization/api_key"
	invitations_handler "github.com/yca-software/2chi-go-api/internals/handlers/organization/invitation"
	roles_handler "github.com/yca-software/2chi-go-api/internals/handlers/organization/role"
	teams_handler "github.com/yca-software/2chi-go-api/internals/handlers/organization/team"
	platform_http "github.com/yca-software/2chi-go-api/internals/packages/http"
	api_key_service "github.com/yca-software/2chi-go-api/internals/services/api_key"
	audit_service "github.com/yca-software/2chi-go-api/internals/services/audit"
	billing_service "github.com/yca-software/2chi-go-api/internals/services/billing"
	invitation_service "github.com/yca-software/2chi-go-api/internals/services/invitation"
	organization_service "github.com/yca-software/2chi-go-api/internals/services/organization"
	role_service "github.com/yca-software/2chi-go-api/internals/services/role"
	team_service "github.com/yca-software/2chi-go-api/internals/services/team"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_logger "github.com/yca-software/2chi-go-logger"
	chi_ratelimit "github.com/yca-software/2chi-go-ratelimit"
)

type OrganizationsHandler struct {
	organizationsService organization_service.Service
	billingService       billing_service.Service
	auditLogsService     audit_service.Service
	rolesHandler         *roles_handler.RolesHandler
	teamsHandler         *teams_handler.TeamsHandler
	invitationsHandler   *invitations_handler.InvitationsHandler
	apiKeysHandler       *api_key_handler.APIKeysHandler
	logger               chi_logger.Logger
}

func NewOrganizationsHandler(
	organizationsService organization_service.Service,
	billingService billing_service.Service,
	auditLogsService audit_service.Service,
	rolesService role_service.Service,
	teamsService team_service.Service,
	invitationsService invitation_service.Service,
	apiKeysService api_key_service.Service,
	logger chi_logger.Logger,
) *OrganizationsHandler {
	return &OrganizationsHandler{
		organizationsService: organizationsService,
		billingService:       billingService,
		auditLogsService:     auditLogsService,
		rolesHandler:         roles_handler.NewRolesHandler(rolesService, logger),
		teamsHandler:         teams_handler.NewTeamsHandler(teamsService, logger),
		invitationsHandler:   invitations_handler.NewInvitationsHandler(invitationsService, logger),
		apiKeysHandler:       api_key_handler.NewAPIKeysHandler(apiKeysService, logger),
		logger:               logger,
	}
}

func (h *OrganizationsHandler) RegisterRoutes(e *echo.Echo, authMiddleware echo.MiddlewareFunc, rateLimiter *chi_ratelimit.RateLimiter) {
	orgV1 := e.Group("/api/v1/organization", authMiddleware)

	orgV1.POST("", h.CreateOrganization)

	detail := orgV1.Group("/:orgId")
	detail.GET("", h.GetOrganization)
	detail.PATCH("", h.UpdateOrganization)
	detail.POST("/archive", h.ArchiveOrganization)
	if h.billingService != nil {
		detail.POST("/subscription/checkout", h.CreateCheckoutSession)
		detail.POST("/subscription/change-plan", h.ChangePlan)
		detail.POST("/subscription/portal", h.CreateCustomerPortalSession)
		detail.POST("/subscription/process-transaction", h.ProcessTransaction)
	}
	detail.GET("/audit-log", h.ListAuditLogs)

	h.registerMemberRoutes(detail)
	h.rolesHandler.RegisterRoutes(detail)
	h.teamsHandler.RegisterRoutes(detail)
	h.invitationsHandler.RegisterRoutes(detail, rateLimiter)
	h.apiKeysHandler.RegisterRoutes(detail)
}

func (h *OrganizationsHandler) registerMemberRoutes(detail *echo.Group) {
	members := detail.Group("/member")
	members.GET("", h.ListOrganizationMembers)
	members.PATCH("/:memberId/role", h.UpdateOrganizationMemberRole)
	members.DELETE("/:memberId", h.RemoveOrganizationMember)
}

// CreateOrganization godoc
// @Summary      Create organization
// @Description  Creates a new organization with default roles and owner membership
// @Tags         organization
// @Accept       json
// @Produce      json
// @Param        organization  body      organization_service.CreateOrganizationRequest  true  "Organization request"
// @Success      201           {object}  organization_service.CreateOrganizationResponse
// @Failure      400           {object}  error.ErrorResponse
// @Failure      401           {object}  error.ErrorResponse
// @Failure      403           {object}  error.ErrorResponse
// @Failure      422           {object}  error.ErrorResponse
// @Failure      500           {object}  error.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/organization [post]
func (h *OrganizationsHandler) CreateOrganization(c echo.Context) error {
	ctx, accessInfo, err := handler_helpers.UserContext(c)
	if err != nil {
		return err
	}

	var req organization_service.CreateOrganizationRequest
	if err := c.Bind(&req); err != nil {
		return chi_error.NewBadRequestError(err, "InvalidRequestBody", nil)
	}

	resp, err := h.organizationsService.CreateOrganization(ctx, &req, accessInfo)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, resp)
}

// GetOrganization godoc
// @Summary      Get organization
// @Description  Retrieves an organization by ID
// @Tags         organization
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
// @Router       /api/v1/organization/{orgId} [get]
func (h *OrganizationsHandler) GetOrganization(c echo.Context) error {
	ctx, accessInfo, orgID, err := handler_helpers.OrgHandlerContext(c)
	if err != nil {
		return err
	}

	org, err := h.organizationsService.GetOrganization(ctx, &organization_service.GetOrganizationRequest{
		OrganizationID: orgID,
	}, accessInfo)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, org)
}

// UpdateOrganization godoc
// @Summary      Update organization
// @Description  Updates organization name and place ID
// @Tags         organization
// @Accept       json
// @Produce      json
// @Param        orgId         path      string  true  "Organization ID"
// @Param        organization  body      organization_service.UpdateOrganizationRequest  true  "Organization request"
// @Success      200           {object}  models.Organization
// @Failure      400           {object}  error.ErrorResponse
// @Failure      401           {object}  error.ErrorResponse
// @Failure      403           {object}  error.ErrorResponse
// @Failure      402           {object}  error.ErrorResponse
// @Failure      422           {object}  error.ErrorResponse
// @Failure      500           {object}  error.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/organization/{orgId} [patch]
func (h *OrganizationsHandler) UpdateOrganization(c echo.Context) error {
	ctx, accessInfo, orgID, err := handler_helpers.OrgHandlerContext(c)
	if err != nil {
		return err
	}

	var req organization_service.UpdateOrganizationRequest
	if err := c.Bind(&req); err != nil {
		return chi_error.NewBadRequestError(err, "InvalidRequestBody", nil)
	}
	req.OrganizationID = orgID

	org, err := h.organizationsService.UpdateOrganization(ctx, &req, accessInfo)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, org)
}

// ArchiveOrganization godoc
// @Summary      Archive organization
// @Description  Soft-deletes an organization
// @Tags         organization
// @Accept       json
// @Produce      json
// @Param        orgId  path  string  true  "Organization ID"
// @Success      204    "No Content"
// @Failure      400    {object}  error.ErrorResponse
// @Failure      401    {object}  error.ErrorResponse
// @Failure      403    {object}  error.ErrorResponse
// @Failure      404    {object}  error.ErrorResponse
// @Failure      500    {object}  error.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/organization/{orgId}/archive [post]
func (h *OrganizationsHandler) ArchiveOrganization(c echo.Context) error {
	ctx, accessInfo, orgID, err := handler_helpers.OrgHandlerContext(c)
	if err != nil {
		return err
	}

	if err := h.organizationsService.ArchiveOrganization(ctx, &organization_service.ArchiveOrganizationRequest{
		OrganizationID: orgID,
	}, accessInfo); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

// ListAuditLogs godoc
// @Summary      List organization audit logs
// @Description  Paginated audit logs for the organization (subject to plan retention)
// @Tags         organization
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
// @Router       /api/v1/organization/{orgId}/audit-log [get]
func (h *OrganizationsHandler) ListAuditLogs(c echo.Context) error {
	ctx, accessInfo, orgID, err := handler_helpers.OrgHandlerContext(c)
	if err != nil {
		return err
	}

	limit, offset, err := platform_http.ParseLimitOffsetStrict(c, 50, 100)
	if err != nil {
		return err
	}

	filters, err := handler_helpers.ParseAuditLogFilters(c)
	if err != nil {
		return err
	}

	resp, err := h.auditLogsService.ListForOrganization(ctx, &audit_service.ListForOrganizationRequest{
		OrganizationID: orgID,
		Limit:          limit,
		Offset:         offset,
		Filters:        filters,
	}, accessInfo)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, resp)
}

// ListOrganizationMembers godoc
// @Summary      List organization members
// @Description  Lists members for an organization
// @Tags         organization member
// @Accept       json
// @Produce      json
// @Param        orgId  path  string  true  "Organization ID"
// @Success      200    {array}   models.OrganizationMemberWithUser
// @Failure      400    {object}  error.ErrorResponse
// @Failure      401    {object}  error.ErrorResponse
// @Failure      403    {object}  error.ErrorResponse
// @Failure      404    {object}  error.ErrorResponse
// @Failure      500    {object}  error.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/organization/{orgId}/member [get]
func (h *OrganizationsHandler) ListOrganizationMembers(c echo.Context) error {
	ctx, accessInfo, orgID, err := handler_helpers.OrgHandlerContext(c)
	if err != nil {
		return err
	}

	resp, err := h.organizationsService.ListOrganizationMembers(ctx, &organization_service.ListOrganizationMembersRequest{
		OrganizationID: orgID,
	}, accessInfo)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, resp)
}

// UpdateOrganizationMemberRole godoc
// @Summary      Update organization member role
// @Description  Updates the role of a member in an organization
// @Tags         organization member
// @Accept       json
// @Produce      json
// @Param        orgId     path  string  true  "Organization ID"
// @Param        memberId  path  string  true  "Member ID"
// @Param        role      body  organization_service.UpdateOrganizationMemberRequest  true  "Role request"
// @Success      200       {object}  models.OrganizationMemberWithUser
// @Failure      400       {object}  error.ErrorResponse
// @Failure      401       {object}  error.ErrorResponse
// @Failure      403       {object}  error.ErrorResponse
// @Failure      404       {object}  error.ErrorResponse
// @Failure      422       {object}  error.ErrorResponse
// @Failure      500       {object}  error.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/organization/{orgId}/member/{memberId}/role [patch]
func (h *OrganizationsHandler) UpdateOrganizationMemberRole(c echo.Context) error {
	ctx, accessInfo, orgID, err := handler_helpers.OrgHandlerContext(c)
	if err != nil {
		return err
	}

	var req organization_service.UpdateOrganizationMemberRequest
	if err := c.Bind(&req); err != nil {
		return chi_error.NewBadRequestError(err, "InvalidRequestBody", nil)
	}
	req.OrganizationID = orgID
	req.MemberID = c.Param("memberId")

	resp, err := h.organizationsService.UpdateOrganizationMember(ctx, &req, accessInfo)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, resp)
}

// RemoveOrganizationMember godoc
// @Summary      Remove organization member
// @Description  Removes a member from an organization
// @Tags         organization member
// @Accept       json
// @Produce      json
// @Param        orgId     path  string  true  "Organization ID"
// @Param        memberId  path  string  true  "Member ID"
// @Success      204       "No Content"
// @Failure      400       {object}  error.ErrorResponse
// @Failure      401       {object}  error.ErrorResponse
// @Failure      403       {object}  error.ErrorResponse
// @Failure      404       {object}  error.ErrorResponse
// @Failure      500       {object}  error.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/organization/{orgId}/member/{memberId} [delete]
func (h *OrganizationsHandler) RemoveOrganizationMember(c echo.Context) error {
	ctx, accessInfo, orgID, err := handler_helpers.OrgHandlerContext(c)
	if err != nil {
		return err
	}

	if err := h.organizationsService.DeleteOrganizationMember(ctx, &organization_service.DeleteOrganizationMemberRequest{
		OrganizationID: orgID,
		MemberID:       c.Param("memberId"),
	}, accessInfo); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

// CreateCheckoutSession godoc
// @Summary      Create checkout session
// @Description  Starts a Paddle checkout for a subscription plan
// @Tags         organization
// @Accept       json
// @Produce      json
// @Param        orgId         path      string                                true  "Organization ID"
// @Param        checkout      body      billing_service.CreateCheckoutSessionRequest  true  "Checkout request"
// @Success      200           {object}  billing_service.CheckoutSessionResponse
// @Failure      400           {object}  error.ErrorResponse
// @Failure      401           {object}  error.ErrorResponse
// @Failure      403           {object}  error.ErrorResponse
// @Failure      402           {object}  error.ErrorResponse
// @Failure      422           {object}  error.ErrorResponse
// @Failure      500           {object}  error.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/organization/{orgId}/subscription/checkout [post]
func (h *OrganizationsHandler) CreateCheckoutSession(c echo.Context) error {
	ctx, accessInfo, orgID, err := handler_helpers.OrgHandlerContext(c)
	if err != nil {
		return err
	}
	var req billing_service.CreateCheckoutSessionRequest
	if err := c.Bind(&req); err != nil {
		return err
	}
	req.OrganizationID = orgID
	resp, err := h.billingService.CreateCheckoutSession(ctx, &req, accessInfo)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, resp)
}

// ChangePlan godoc
// @Summary      Change subscription plan
// @Description  Changes the organization subscription plan via Paddle
// @Tags         organization
// @Accept       json
// @Produce      json
// @Param        orgId  path      string                      true  "Organization ID"
// @Param        plan   body      billing_service.ChangePlanRequest  true  "Plan change request"
// @Success      200    {object}  billing_service.ChangePlanResponse
// @Failure      400    {object}  error.ErrorResponse
// @Failure      401    {object}  error.ErrorResponse
// @Failure      403    {object}  error.ErrorResponse
// @Failure      402    {object}  error.ErrorResponse
// @Failure      422    {object}  error.ErrorResponse
// @Failure      500    {object}  error.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/organization/{orgId}/subscription/change-plan [post]
func (h *OrganizationsHandler) ChangePlan(c echo.Context) error {
	ctx, accessInfo, orgID, err := handler_helpers.OrgHandlerContext(c)
	if err != nil {
		return err
	}
	var req billing_service.ChangePlanRequest
	if err := c.Bind(&req); err != nil {
		return err
	}
	req.OrganizationID = orgID
	resp, err := h.billingService.ChangePlan(ctx, &req, accessInfo)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, resp)
}

// CreateCustomerPortalSession godoc
// @Summary      Create customer portal session
// @Description  Returns a Paddle customer portal URL for billing self-service
// @Tags         organization
// @Accept       json
// @Produce      json
// @Param        orgId   path      string                                         true  "Organization ID"
// @Param        portal  body      billing_service.CreateCustomerPortalSessionRequest  true  "Portal session request"
// @Success      200     {object}  billing_service.CustomerPortalSessionResponse
// @Failure      400     {object}  error.ErrorResponse
// @Failure      401     {object}  error.ErrorResponse
// @Failure      403     {object}  error.ErrorResponse
// @Failure      402     {object}  error.ErrorResponse
// @Failure      422     {object}  error.ErrorResponse
// @Failure      500     {object}  error.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/organization/{orgId}/subscription/portal [post]
func (h *OrganizationsHandler) CreateCustomerPortalSession(c echo.Context) error {
	ctx, accessInfo, orgID, err := handler_helpers.OrgHandlerContext(c)
	if err != nil {
		return err
	}
	var req billing_service.CreateCustomerPortalSessionRequest
	if err := c.Bind(&req); err != nil {
		return err
	}
	req.OrganizationID = orgID
	resp, err := h.billingService.CreateCustomerPortalSession(ctx, &req, accessInfo)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, resp)
}

// ProcessTransaction godoc
// @Summary      Process Paddle transaction
// @Description  Applies a completed Paddle transaction to the organization subscription
// @Tags         organization
// @Accept       json
// @Produce      json
// @Param        orgId        path      string                              true  "Organization ID"
// @Param        transaction  body      billing_service.ProcessTransactionRequest  true  "Transaction request"
// @Success      200          {object}  models.Organization
// @Failure      400          {object}  error.ErrorResponse
// @Failure      401          {object}  error.ErrorResponse
// @Failure      403          {object}  error.ErrorResponse
// @Failure      402          {object}  error.ErrorResponse
// @Failure      422          {object}  error.ErrorResponse
// @Failure      500          {object}  error.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/organization/{orgId}/subscription/process-transaction [post]
func (h *OrganizationsHandler) ProcessTransaction(c echo.Context) error {
	ctx, accessInfo, orgID, err := handler_helpers.OrgHandlerContext(c)
	if err != nil {
		return err
	}
	var req billing_service.ProcessTransactionRequest
	if err := c.Bind(&req); err != nil {
		return err
	}
	req.OrganizationID = orgID
	org, err := h.billingService.ProcessTransaction(ctx, &req, accessInfo)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, org)
}
