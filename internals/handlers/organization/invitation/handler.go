package handlers

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	handler_helpers "github.com/yca-software/2chi-go-api/internals/handlers/helpers"
	platform_http "github.com/yca-software/2chi-go-api/internals/packages/http"
	invitation_service "github.com/yca-software/2chi-go-api/internals/services/invitation"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_logger "github.com/yca-software/2chi-go-logger"
	chi_ratelimit "github.com/yca-software/2chi-go-ratelimit"
	chi_types "github.com/yca-software/2chi-go-types"
)

type InvitationsHandler struct {
	invitationsService invitation_service.Service
	logger             chi_logger.Logger
}

func NewInvitationsHandler(invitationsService invitation_service.Service, logger chi_logger.Logger) *InvitationsHandler {
	return &InvitationsHandler{invitationsService: invitationsService, logger: logger}
}

func (h *InvitationsHandler) RegisterRoutes(detail *echo.Group, rateLimiter *chi_ratelimit.RateLimiter) {
	group := detail.Group("/invitation")
	inviteRL := rateLimiter.ScopedPrincipalRateLimit("100-D", "invitation_create")
	group.POST("", h.CreateInvitation, inviteRL)
	group.GET("", h.ListInvitations)
	group.DELETE("/:invitationId", h.RevokeInvitation)
}

// CreateInvitation godoc
// @Summary      Create invitation
// @Description  Invites a user to join an organization (adds existing users immediately; others receive email)
// @Tags         organization invitation
// @Accept       json
// @Produce      json
// @Param        orgId       path      string                              true  "Organization ID"
// @Param        invitation  body      invitation_service.CreateRequest    true  "Invitation request"
// @Success      201         {object}  invitation_service.CreateResponse
// @Failure      400         {object}  error.ErrorResponse
// @Failure      401         {object}  error.ErrorResponse
// @Failure      403         {object}  error.ErrorResponse
// @Failure      404         {object}  error.ErrorResponse
// @Failure      409         {object}  error.ErrorResponse
// @Failure      422         {object}  error.ErrorResponse
// @Failure      429         {object}  error.ErrorResponse
// @Failure      500         {object}  error.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/organization/{orgId}/invitation [post]
func (h *InvitationsHandler) CreateInvitation(c echo.Context) error {
	ctx, accessInfo, orgID, err := handler_helpers.OrgHandlerContext(c)
	if err != nil {
		return err
	}
	if accessInfo.Type != chi_types.AccessTypeUser {
		return chi_error.NewUnauthorizedError(errors.New("user required"), "Unauthorized", nil)
	}

	var req invitation_service.CreateRequest
	if err := c.Bind(&req); err != nil {
		return chi_error.NewBadRequestError(err, "InvalidRequestBody", nil)
	}
	req.OrganizationID = orgID
	req.InvitedByID = accessInfo.SubjectID.String()
	req.InvitedByEmail = accessInfo.Email
	req.Language = platform_http.RequestLanguage(c)

	resp, err := h.invitationsService.Create(ctx, &req, accessInfo)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, resp)
}

// ListInvitations godoc
// @Summary      List invitations
// @Description  Retrieves pending invitations for an organization
// @Tags         organization invitation
// @Accept       json
// @Produce      json
// @Param        orgId  path  string  true  "Organization ID"
// @Success      200    {array}   models.Invitation
// @Failure      400    {object}  error.ErrorResponse
// @Failure      401    {object}  error.ErrorResponse
// @Failure      403    {object}  error.ErrorResponse
// @Failure      404    {object}  error.ErrorResponse
// @Failure      500    {object}  error.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/organization/{orgId}/invitation [get]
func (h *InvitationsHandler) ListInvitations(c echo.Context) error {
	ctx, accessInfo, orgID, err := handler_helpers.OrgHandlerContext(c)
	if err != nil {
		return err
	}

	invitations, err := h.invitationsService.List(ctx, &invitation_service.ListRequest{
		OrganizationID: orgID,
	}, accessInfo)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, invitations)
}

// RevokeInvitation godoc
// @Summary      Revoke invitation
// @Description  Revokes a pending invitation
// @Tags         organization invitation
// @Accept       json
// @Produce      json
// @Param        orgId          path  string  true  "Organization ID"
// @Param        invitationId   path  string  true  "Invitation ID"
// @Success      204            "No Content"
// @Failure      400            {object}  error.ErrorResponse
// @Failure      401            {object}  error.ErrorResponse
// @Failure      403            {object}  error.ErrorResponse
// @Failure      404            {object}  error.ErrorResponse
// @Failure      422            {object}  error.ErrorResponse
// @Failure      500            {object}  error.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/organization/{orgId}/invitation/{invitationId} [delete]
func (h *InvitationsHandler) RevokeInvitation(c echo.Context) error {
	ctx, accessInfo, orgID, err := handler_helpers.OrgHandlerContext(c)
	if err != nil {
		return err
	}

	if err := h.invitationsService.Revoke(ctx, &invitation_service.RevokeRequest{
		OrganizationID: orgID,
		InvitationID:   c.Param("invitationId"),
	}, accessInfo); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}
