package handlers

import (
	"io"
	"net/http"

	"github.com/labstack/echo/v4"
	billing_service "github.com/yca-software/2chi-go-api/internals/services/billing"
	chi_logger "github.com/yca-software/2chi-go-logger"
)

type BillingWebhookHandler struct {
	billingService billing_service.Service
	logger         chi_logger.Logger
}

func NewBillingWebhookHandler(billingService billing_service.Service, logger chi_logger.Logger) *BillingWebhookHandler {
	return &BillingWebhookHandler{billingService: billingService, logger: logger}
}

func (h *BillingWebhookHandler) RegisterRoutes(e *echo.Echo) {
	if h.billingService == nil {
		return
	}
	e.POST("/api/v1/webhooks/paddle", h.HandlePaddleWebhook)
}

// HandlePaddleWebhook godoc
// @Summary      Paddle webhook
// @Description  Receives Paddle Billing webhook events
// @Tags         webhooks
// @Accept       json
// @Produce      json
// @Success      200
// @Failure      401  {object}  error.ErrorResponse
// @Failure      422  {object}  error.ErrorResponse
// @Failure      500  {object}  error.ErrorResponse
// @Router       /api/v1/webhooks/paddle [post]
func (h *BillingWebhookHandler) HandlePaddleWebhook(c echo.Context) error {
	payload, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return err
	}
	signature := c.Request().Header.Get("Paddle-Signature")
	if err := h.billingService.HandleWebhook(c.Request().Context(), payload, signature); err != nil {
		return err
	}
	return c.NoContent(http.StatusOK)
}
