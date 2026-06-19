package handlers

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	location_service "github.com/yca-software/2chi-go-api/internals/domains/core/services/location"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_logger "github.com/yca-software/2chi-go-logger"
	chi_ratelimit "github.com/yca-software/2chi-go-ratelimit"
)

type LocationHandler struct {
	locationService location_service.Service
	logger          chi_logger.Logger
}

func NewLocationHandler(locationService location_service.Service, logger chi_logger.Logger) *LocationHandler {
	return &LocationHandler{locationService: locationService, logger: logger}
}

func (h *LocationHandler) RegisterRoutes(e *echo.Echo, authMiddleware echo.MiddlewareFunc, rateLimiter *chi_ratelimit.RateLimiter) {
	locationRL := rateLimiter.ScopedPrincipalRateLimit("100-H", "location_autocomplete")
	e.GET("/api/v1/location/autocomplete", h.Autocomplete, authMiddleware, locationRL)
}

// Autocomplete godoc
// @Summary      Location autocomplete
// @Description  Returns location suggestions for address input
// @Tags         location
// @Accept       json
// @Produce      json
// @Param        input  query     string  true  "Search input"
// @Success      200    {object}  google.AutocompleteLocationResponse
// @Failure      400    {object}  error.ErrorResponse
// @Failure      401    {object}  error.ErrorResponse
// @Failure      500    {object}  error.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/location/autocomplete [get]
func (h *LocationHandler) Autocomplete(c echo.Context) error {
	input := c.QueryParam("input")
	if input == "" {
		return chi_error.NewBadRequestError(errors.New("input parameter is required"), "InvalidRequestBody", nil)
	}

	resp, err := h.locationService.AutocompleteLocation(c.Request().Context(), input)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, resp)
}
