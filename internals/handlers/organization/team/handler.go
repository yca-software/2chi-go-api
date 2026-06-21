package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	handler_helpers "github.com/yca-software/2chi-go-api/internals/handlers/helpers"
	team_service "github.com/yca-software/2chi-go-api/internals/services/team"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_logger "github.com/yca-software/2chi-go-logger"
)

type TeamsHandler struct {
	teamsService team_service.Service
	logger       chi_logger.Logger
}

func NewTeamsHandler(teamsService team_service.Service, logger chi_logger.Logger) *TeamsHandler {
	return &TeamsHandler{teamsService: teamsService, logger: logger}
}

func (h *TeamsHandler) RegisterRoutes(detail *echo.Group) {
	group := detail.Group("/team")
	group.POST("", h.CreateTeam)
	group.GET("", h.ListTeams)
	group.PATCH("/:teamId", h.UpdateTeam)
	group.DELETE("/:teamId", h.DeleteTeam)

	members := group.Group("/:teamId/member")
	members.GET("", h.ListTeamMembers)
	members.POST("", h.AddTeamMember)
	members.DELETE("/:memberId", h.RemoveTeamMember)
}

// CreateTeam godoc
// @Summary      Create team
// @Description  Creates a new team for an organization
// @Tags         organization team
// @Accept       json
// @Produce      json
// @Param        orgId  path      string                      true  "Organization ID"
// @Param        team   body      team_service.CreateTeamRequest  true  "Team request"
// @Success      201    {object}  models.Team
// @Failure      400    {object}  error.ErrorResponse
// @Failure      401    {object}  error.ErrorResponse
// @Failure      403    {object}  error.ErrorResponse
// @Failure      402    {object}  error.ErrorResponse
// @Failure      404    {object}  error.ErrorResponse
// @Failure      422    {object}  error.ErrorResponse
// @Failure      500    {object}  error.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/organization/{orgId}/team [post]
func (h *TeamsHandler) CreateTeam(c echo.Context) error {
	ctx, accessInfo, orgID, err := handler_helpers.OrgHandlerContext(c)
	if err != nil {
		return err
	}

	var req team_service.CreateTeamRequest
	if err := c.Bind(&req); err != nil {
		return chi_error.NewBadRequestError(err, "InvalidRequestBody", nil)
	}
	req.OrganizationID = orgID

	team, err := h.teamsService.CreateTeam(ctx, &req, accessInfo)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, team)
}

// ListTeams godoc
// @Summary      List teams
// @Description  Retrieves teams for an organization
// @Tags         organization team
// @Accept       json
// @Produce      json
// @Param        orgId  path  string  true  "Organization ID"
// @Success      200    {array}   models.Team
// @Failure      400    {object}  error.ErrorResponse
// @Failure      401    {object}  error.ErrorResponse
// @Failure      403    {object}  error.ErrorResponse
// @Failure      404    {object}  error.ErrorResponse
// @Failure      500    {object}  error.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/organization/{orgId}/team [get]
func (h *TeamsHandler) ListTeams(c echo.Context) error {
	ctx, accessInfo, orgID, err := handler_helpers.OrgHandlerContext(c)
	if err != nil {
		return err
	}

	teams, err := h.teamsService.ListTeams(ctx, &team_service.ListTeamsRequest{OrganizationID: orgID}, accessInfo)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, teams)
}

// UpdateTeam godoc
// @Summary      Update team
// @Description  Updates a team for an organization
// @Tags         organization team
// @Accept       json
// @Produce      json
// @Param        orgId   path      string                      true  "Organization ID"
// @Param        teamId  path      string                      true  "Team ID"
// @Param        team    body      team_service.UpdateTeamRequest  true  "Team request"
// @Success      200     {object}  models.Team
// @Failure      400     {object}  error.ErrorResponse
// @Failure      401     {object}  error.ErrorResponse
// @Failure      403     {object}  error.ErrorResponse
// @Failure      402     {object}  error.ErrorResponse
// @Failure      404     {object}  error.ErrorResponse
// @Failure      422     {object}  error.ErrorResponse
// @Failure      500     {object}  error.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/organization/{orgId}/team/{teamId} [patch]
func (h *TeamsHandler) UpdateTeam(c echo.Context) error {
	ctx, accessInfo, orgID, err := handler_helpers.OrgHandlerContext(c)
	if err != nil {
		return err
	}

	var req team_service.UpdateTeamRequest
	if err := c.Bind(&req); err != nil {
		return chi_error.NewBadRequestError(err, "InvalidRequestBody", nil)
	}
	req.OrganizationID = orgID
	req.TeamID = c.Param("teamId")

	team, err := h.teamsService.UpdateTeam(ctx, &req, accessInfo)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, team)
}

// DeleteTeam godoc
// @Summary      Delete team
// @Description  Permanently deletes a team by its ID
// @Tags         organization team
// @Accept       json
// @Produce      json
// @Param        orgId   path  string  true  "Organization ID"
// @Param        teamId  path  string  true  "Team ID"
// @Success      204     "No Content"
// @Failure      400     {object}  error.ErrorResponse
// @Failure      401     {object}  error.ErrorResponse
// @Failure      403     {object}  error.ErrorResponse
// @Failure      402     {object}  error.ErrorResponse
// @Failure      404     {object}  error.ErrorResponse
// @Failure      500     {object}  error.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/organization/{orgId}/team/{teamId} [delete]
func (h *TeamsHandler) DeleteTeam(c echo.Context) error {
	ctx, accessInfo, orgID, err := handler_helpers.OrgHandlerContext(c)
	if err != nil {
		return err
	}

	if err := h.teamsService.DeleteTeam(ctx, &team_service.DeleteTeamRequest{
		OrganizationID: orgID,
		TeamID:         c.Param("teamId"),
	}, accessInfo); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

// ListTeamMembers godoc
// @Summary      List team members
// @Description  Retrieves members for a team
// @Tags         organization team member
// @Accept       json
// @Produce      json
// @Param        orgId   path  string  true  "Organization ID"
// @Param        teamId  path  string  true  "Team ID"
// @Success      200     {array}   models.TeamMemberWithUser
// @Failure      400     {object}  error.ErrorResponse
// @Failure      401     {object}  error.ErrorResponse
// @Failure      403     {object}  error.ErrorResponse
// @Failure      402     {object}  error.ErrorResponse
// @Failure      404     {object}  error.ErrorResponse
// @Failure      500     {object}  error.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/organization/{orgId}/team/{teamId}/member [get]
func (h *TeamsHandler) ListTeamMembers(c echo.Context) error {
	ctx, accessInfo, orgID, err := handler_helpers.OrgHandlerContext(c)
	if err != nil {
		return err
	}

	members, err := h.teamsService.ListTeamMembers(ctx, &team_service.ListTeamMembersRequest{
		OrganizationID: orgID,
		TeamID:         c.Param("teamId"),
	}, accessInfo)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, members)
}

// AddTeamMember godoc
// @Summary      Add team member
// @Description  Adds an organization member to a team
// @Tags         organization team member
// @Accept       json
// @Produce      json
// @Param        orgId   path      string                           true  "Organization ID"
// @Param        teamId  path      string                           true  "Team ID"
// @Param        body    body      team_service.AddTeamMemberRequest    true  "Team member request"
// @Success      201     {object}  models.TeamMemberWithUser
// @Failure      400     {object}  error.ErrorResponse
// @Failure      401     {object}  error.ErrorResponse
// @Failure      403     {object}  error.ErrorResponse
// @Failure      402     {object}  error.ErrorResponse
// @Failure      404     {object}  error.ErrorResponse
// @Failure      422     {object}  error.ErrorResponse
// @Failure      500     {object}  error.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/organization/{orgId}/team/{teamId}/member [post]
func (h *TeamsHandler) AddTeamMember(c echo.Context) error {
	ctx, accessInfo, orgID, err := handler_helpers.OrgHandlerContext(c)
	if err != nil {
		return err
	}

	var req team_service.AddTeamMemberRequest
	if err := c.Bind(&req); err != nil {
		return chi_error.NewBadRequestError(err, "InvalidRequestBody", nil)
	}
	req.OrganizationID = orgID
	req.TeamID = c.Param("teamId")

	member, err := h.teamsService.AddTeamMember(ctx, &req, accessInfo)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, member)
}

// RemoveTeamMember godoc
// @Summary      Remove team member
// @Description  Removes a member from a team
// @Tags         organization team member
// @Accept       json
// @Produce      json
// @Param        orgId     path  string  true  "Organization ID"
// @Param        teamId    path  string  true  "Team ID"
// @Param        memberId  path  string  true  "Team member ID"
// @Success      204       "No Content"
// @Failure      400       {object}  error.ErrorResponse
// @Failure      401       {object}  error.ErrorResponse
// @Failure      403       {object}  error.ErrorResponse
// @Failure      402       {object}  error.ErrorResponse
// @Failure      404       {object}  error.ErrorResponse
// @Failure      500       {object}  error.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/organization/{orgId}/team/{teamId}/member/{memberId} [delete]
func (h *TeamsHandler) RemoveTeamMember(c echo.Context) error {
	ctx, accessInfo, orgID, err := handler_helpers.OrgHandlerContext(c)
	if err != nil {
		return err
	}

	if err := h.teamsService.RemoveTeamMember(ctx, &team_service.RemoveTeamMemberRequest{
		OrganizationID: orgID,
		TeamID:         c.Param("teamId"),
		MemberID:       c.Param("memberId"),
	}, accessInfo); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}
