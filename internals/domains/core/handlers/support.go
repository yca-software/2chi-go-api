package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	support_service "github.com/yca-software/2chi-go-api/internals/domains/core/services/support"
	chi_logger "github.com/yca-software/2chi-go-logger"
	chi_ratelimit "github.com/yca-software/2chi-go-ratelimit"
)

type SupportHandler struct {
	supportService support_service.Service
	logger         chi_logger.Logger
}

func NewSupportHandler(supportService support_service.Service, logger chi_logger.Logger) *SupportHandler {
	return &SupportHandler{supportService: supportService, logger: logger}
}

func (h *SupportHandler) RegisterRoutes(e *echo.Echo, authMiddleware echo.MiddlewareFunc, rateLimiter *chi_ratelimit.RateLimiter) {
	group := e.Group("/api/v1/support", authMiddleware)
	supportRL := rateLimiter.ScopedPrincipalRateLimit("10-H", "support_submit")
	group.POST("", h.Submit, supportRL)
}

// Submit godoc
// @Summary      Submit support request
// @Description  Sends a support message to the configured support inbox
// @Tags         support
// @Accept       json
// @Produce      json
// @Param        body  body      support_service.SubmitSupportRequest  true  "Support request"
// @Success      204
// @Failure      401  {object}  error.ErrorResponse
// @Failure      403  {object}  error.ErrorResponse
// @Failure      422  {object}  error.ErrorResponse
// @Failure      429  {object}  error.ErrorResponse
// @Failure      500  {object}  error.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/support [post]
func (h *SupportHandler) Submit(c echo.Context) error {
	ctx, accessInfo, err := userContext(c)
	if err != nil {
		return err
	}

	var req support_service.SubmitSupportRequest
	if err := c.Bind(&req); err != nil {
		return err
	}
	if req.Subject == "" {
		req.Subject = "(no subject)"
	}
	req.UserAgent = c.Request().UserAgent()

	if err := h.supportService.Submit(ctx, &req, accessInfo); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}
