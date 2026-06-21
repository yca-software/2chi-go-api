package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	handler_helpers "github.com/yca-software/2chi-go-api/internals/handlers/helpers"
	role_service "github.com/yca-software/2chi-go-api/internals/services/role"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_logger "github.com/yca-software/2chi-go-logger"
)

type RolesHandler struct {
	rolesService role_service.Service
	logger       chi_logger.Logger
}

func NewRolesHandler(rolesService role_service.Service, logger chi_logger.Logger) *RolesHandler {
	return &RolesHandler{rolesService: rolesService, logger: logger}
}

func (h *RolesHandler) RegisterRoutes(detail *echo.Group) {
	group := detail.Group("/role")
	group.POST("", h.CreateRole)
	group.GET("", h.ListRoles)
	group.PATCH("/:roleId", h.UpdateRole)
	group.DELETE("/:roleId", h.DeleteRole)
}

// CreateRole godoc
// @Summary      Create role
// @Description  Creates a new role for an organization
// @Tags         organization role
// @Accept       json
// @Produce      json
// @Param        orgId  path      string                         true  "Organization ID"
// @Param        role   body      role_service.CreateRoleRequest     true  "Role request"
// @Success      201    {object}  models.Role
// @Failure      400    {object}  error.ErrorResponse
// @Failure      401    {object}  error.ErrorResponse
// @Failure      403    {object}  error.ErrorResponse
// @Failure      402    {object}  error.ErrorResponse
// @Failure      404    {object}  error.ErrorResponse
// @Failure      422    {object}  error.ErrorResponse
// @Failure      500    {object}  error.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/organization/{orgId}/role [post]
func (h *RolesHandler) CreateRole(c echo.Context) error {
	ctx, accessInfo, orgID, err := handler_helpers.OrgHandlerContext(c)
	if err != nil {
		return err
	}

	var req role_service.CreateRoleRequest
	if err := c.Bind(&req); err != nil {
		return chi_error.NewBadRequestError(err, "InvalidRequestBody", nil)
	}
	req.OrganizationID = orgID

	role, err := h.rolesService.CreateRole(ctx, &req, accessInfo)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, role)
}

// ListRoles godoc
// @Summary      List roles
// @Description  Retrieves roles for an organization
// @Tags         organization role
// @Accept       json
// @Produce      json
// @Param        orgId  path  string  true  "Organization ID"
// @Success      200    {array}   models.Role
// @Failure      400    {object}  error.ErrorResponse
// @Failure      401    {object}  error.ErrorResponse
// @Failure      403    {object}  error.ErrorResponse
// @Failure      404    {object}  error.ErrorResponse
// @Failure      500    {object}  error.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/organization/{orgId}/role [get]
func (h *RolesHandler) ListRoles(c echo.Context) error {
	ctx, accessInfo, orgID, err := handler_helpers.OrgHandlerContext(c)
	if err != nil {
		return err
	}

	roles, err := h.rolesService.ListRoles(ctx, &role_service.ListRolesRequest{OrganizationID: orgID}, accessInfo)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, roles)
}

// UpdateRole godoc
// @Summary      Update role
// @Description  Updates a role for an organization
// @Tags         organization role
// @Accept       json
// @Produce      json
// @Param        orgId   path      string                         true  "Organization ID"
// @Param        roleId  path      string                         true  "Role ID"
// @Param        role    body      role_service.UpdateRoleRequest     true  "Role request"
// @Success      200     {object}  models.Role
// @Failure      400     {object}  error.ErrorResponse
// @Failure      401     {object}  error.ErrorResponse
// @Failure      403     {object}  error.ErrorResponse
// @Failure      402     {object}  error.ErrorResponse
// @Failure      404     {object}  error.ErrorResponse
// @Failure      422     {object}  error.ErrorResponse
// @Failure      500     {object}  error.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/organization/{orgId}/role/{roleId} [patch]
func (h *RolesHandler) UpdateRole(c echo.Context) error {
	ctx, accessInfo, orgID, err := handler_helpers.OrgHandlerContext(c)
	if err != nil {
		return err
	}

	var req role_service.UpdateRoleRequest
	if err := c.Bind(&req); err != nil {
		return chi_error.NewBadRequestError(err, "InvalidRequestBody", nil)
	}
	req.OrganizationID = orgID
	req.RoleID = c.Param("roleId")

	role, err := h.rolesService.UpdateRole(ctx, &req, accessInfo)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, role)
}

// DeleteRole godoc
// @Summary      Delete role
// @Description  Permanently deletes a role by its ID
// @Tags         organization role
// @Accept       json
// @Produce      json
// @Param        orgId   path  string  true  "Organization ID"
// @Param        roleId  path  string  true  "Role ID"
// @Success      204     "No Content"
// @Failure      400     {object}  error.ErrorResponse
// @Failure      401     {object}  error.ErrorResponse
// @Failure      403     {object}  error.ErrorResponse
// @Failure      402     {object}  error.ErrorResponse
// @Failure      404     {object}  error.ErrorResponse
// @Failure      409     {object}  error.ErrorResponse
// @Failure      500     {object}  error.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/organization/{orgId}/role/{roleId} [delete]
func (h *RolesHandler) DeleteRole(c echo.Context) error {
	ctx, accessInfo, orgID, err := handler_helpers.OrgHandlerContext(c)
	if err != nil {
		return err
	}

	if err := h.rolesService.DeleteRole(ctx, &role_service.DeleteRoleRequest{
		OrganizationID: orgID,
		RoleID:         c.Param("roleId"),
	}, accessInfo); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}
