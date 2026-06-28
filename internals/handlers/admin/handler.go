package handlers

import (
	"github.com/labstack/echo/v4"
	"github.com/yca-software/2chi-go-api/internals/config"
	admin_organization_handler "github.com/yca-software/2chi-go-api/internals/handlers/admin/organization"
	admin_user_handler "github.com/yca-software/2chi-go-api/internals/handlers/admin/user"
	audit_service "github.com/yca-software/2chi-go-api/internals/services/audit"
	auth_service "github.com/yca-software/2chi-go-api/internals/services/auth"
	organization_service "github.com/yca-software/2chi-go-api/internals/services/organization"
	user_service "github.com/yca-software/2chi-go-api/internals/services/user"
	chi_logger "github.com/yca-software/2chi-go-logger"
)

type AdminHandler struct {
	usersHandler         *admin_user_handler.UsersHandler
	organizationsHandler *admin_organization_handler.OrganizationsHandler
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
		usersHandler: admin_user_handler.NewUsersHandler(authService, usersService, cfg, logger),
		organizationsHandler: admin_organization_handler.NewOrganizationsHandler(
			organizationsService,
			auditLogsService,
			logger,
		),
		logger: logger,
	}
}

func (h *AdminHandler) RegisterRoutes(e *echo.Echo, authMiddleware, adminMiddleware, adminRateLimit, impersonateRateLimit echo.MiddlewareFunc) {
	admin := e.Group("/api/v1/admin", authMiddleware, adminMiddleware, adminRateLimit)

	h.usersHandler.RegisterRoutes(admin, impersonateRateLimit)
	h.organizationsHandler.RegisterRoutes(admin)
}
