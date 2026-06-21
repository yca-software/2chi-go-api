package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	handler_helpers "github.com/yca-software/2chi-go-api/internals/handlers/helpers"
	api_key_service "github.com/yca-software/2chi-go-api/internals/services/api_key"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_logger "github.com/yca-software/2chi-go-logger"
)

type APIKeysHandler struct {
	apiKeysService api_key_service.Service
	logger         chi_logger.Logger
}

func NewAPIKeysHandler(apiKeysService api_key_service.Service, logger chi_logger.Logger) *APIKeysHandler {
	return &APIKeysHandler{apiKeysService: apiKeysService, logger: logger}
}

func (h *APIKeysHandler) RegisterRoutes(detail *echo.Group) {
	group := detail.Group("/api-key")
	group.POST("", h.CreateAPIKey)
	group.GET("", h.ListAPIKeys)
	group.PATCH("/:apiKeyId", h.UpdateAPIKey)
	group.DELETE("/:apiKeyId", h.DeleteAPIKey)
}

// CreateAPIKey godoc
// @Summary      Create API key
// @Description  Creates a new API key for an organization (secret shown once)
// @Tags         organization api key
// @Accept       json
// @Produce      json
// @Param        orgId   path      string                         true  "Organization ID"
// @Param        apiKey  body      api_key_service.CreateAPIKeyRequest   true  "API key request"
// @Success      201     {object}  api_key_service.CreateAPIKeyResponse
// @Failure      400     {object}  error.ErrorResponse
// @Failure      401     {object}  error.ErrorResponse
// @Failure      403     {object}  error.ErrorResponse
// @Failure      402     {object}  error.ErrorResponse
// @Failure      404     {object}  error.ErrorResponse
// @Failure      422     {object}  error.ErrorResponse
// @Failure      500     {object}  error.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/organization/{orgId}/api-key [post]
func (h *APIKeysHandler) CreateAPIKey(c echo.Context) error {
	ctx, accessInfo, orgID, err := handler_helpers.OrgHandlerContext(c)
	if err != nil {
		return err
	}

	var req api_key_service.CreateAPIKeyRequest
	if err := c.Bind(&req); err != nil {
		return chi_error.NewBadRequestError(err, "InvalidRequestBody", nil)
	}
	req.OrganizationID = orgID

	resp, err := h.apiKeysService.CreateAPIKey(ctx, &req, accessInfo)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, resp)
}

// ListAPIKeys godoc
// @Summary      List API keys
// @Description  Retrieves API keys for an organization
// @Tags         organization api key
// @Accept       json
// @Produce      json
// @Param        orgId  path  string  true  "Organization ID"
// @Success      200    {array}   models.APIKey
// @Failure      400    {object}  error.ErrorResponse
// @Failure      401    {object}  error.ErrorResponse
// @Failure      403    {object}  error.ErrorResponse
// @Failure      402    {object}  error.ErrorResponse
// @Failure      404    {object}  error.ErrorResponse
// @Failure      500    {object}  error.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/organization/{orgId}/api-key [get]
func (h *APIKeysHandler) ListAPIKeys(c echo.Context) error {
	ctx, accessInfo, orgID, err := handler_helpers.OrgHandlerContext(c)
	if err != nil {
		return err
	}

	keys, err := h.apiKeysService.ListAPIKeys(ctx, &api_key_service.ListAPIKeysRequest{OrganizationID: orgID}, accessInfo)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, keys)
}

// UpdateAPIKey godoc
// @Summary      Update API key
// @Description  Updates an API key name and permissions
// @Tags         organization api key
// @Accept       json
// @Produce      json
// @Param        orgId      path      string                         true  "Organization ID"
// @Param        apiKeyId   path      string                         true  "API key ID"
// @Param        apiKey     body      api_key_service.UpdateAPIKeyRequest   true  "API key request"
// @Success      200        {object}  models.APIKey
// @Failure      400        {object}  error.ErrorResponse
// @Failure      401        {object}  error.ErrorResponse
// @Failure      403        {object}  error.ErrorResponse
// @Failure      402        {object}  error.ErrorResponse
// @Failure      404        {object}  error.ErrorResponse
// @Failure      422        {object}  error.ErrorResponse
// @Failure      500        {object}  error.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/organization/{orgId}/api-key/{apiKeyId} [patch]
func (h *APIKeysHandler) UpdateAPIKey(c echo.Context) error {
	ctx, accessInfo, orgID, err := handler_helpers.OrgHandlerContext(c)
	if err != nil {
		return err
	}

	var req api_key_service.UpdateAPIKeyRequest
	if err := c.Bind(&req); err != nil {
		return chi_error.NewBadRequestError(err, "InvalidRequestBody", nil)
	}
	req.OrganizationID = orgID
	req.APIKeyID = c.Param("apiKeyId")

	key, err := h.apiKeysService.UpdateAPIKey(ctx, &req, accessInfo)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, key)
}

// DeleteAPIKey godoc
// @Summary      Delete API key
// @Description  Permanently deletes an API key
// @Tags         organization api key
// @Accept       json
// @Produce      json
// @Param        orgId     path  string  true  "Organization ID"
// @Param        apiKeyId  path  string  true  "API key ID"
// @Success      204       "No Content"
// @Failure      400       {object}  error.ErrorResponse
// @Failure      401       {object}  error.ErrorResponse
// @Failure      403       {object}  error.ErrorResponse
// @Failure      402       {object}  error.ErrorResponse
// @Failure      404       {object}  error.ErrorResponse
// @Failure      500       {object}  error.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/organization/{orgId}/api-key/{apiKeyId} [delete]
func (h *APIKeysHandler) DeleteAPIKey(c echo.Context) error {
	ctx, accessInfo, orgID, err := handler_helpers.OrgHandlerContext(c)
	if err != nil {
		return err
	}

	if err := h.apiKeysService.DeleteAPIKey(ctx, &api_key_service.DeleteAPIKeyRequest{
		OrganizationID: orgID,
		APIKeyID:       c.Param("apiKeyId"),
	}, accessInfo); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}
