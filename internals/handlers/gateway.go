package handlers

import (
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
	"github.com/yca-software/2chi-go-api/internals/config"
	"github.com/yca-software/2chi-go-api/internals/constants"
	admin_handlers "github.com/yca-software/2chi-go-api/internals/handlers/admin"
	auth_handlers "github.com/yca-software/2chi-go-api/internals/handlers/auth"
	location_handlers "github.com/yca-software/2chi-go-api/internals/handlers/location"
	"github.com/yca-software/2chi-go-api/internals/handlers/middleware"
	organization_handlers "github.com/yca-software/2chi-go-api/internals/handlers/organization"
	support_handlers "github.com/yca-software/2chi-go-api/internals/handlers/support"
	user_handlers "github.com/yca-software/2chi-go-api/internals/handlers/user"
	webhook_handlers "github.com/yca-software/2chi-go-api/internals/handlers/webhook"
	"github.com/yca-software/2chi-go-api/internals/packages/authz"
	"github.com/yca-software/2chi-go-api/internals/packages/datastores"
	"github.com/yca-software/2chi-go-api/internals/packages/observer"
	"github.com/yca-software/2chi-go-api/internals/repositories"
	"github.com/yca-software/2chi-go-api/internals/services"
	chi_logger "github.com/yca-software/2chi-go-logger"
	chi_ratelimit "github.com/yca-software/2chi-go-ratelimit"
	chi_token "github.com/yca-software/2chi-go-token"
)

func NewGateway(e *echo.Echo, datastores *datastores.Datastores, cfg *config.Config, appObserver *observer.Observer, logger chi_logger.Logger) {
	db := datastores.Postgres.GetClient().(*sqlx.DB)

	repos := repositories.NewRepositories(db, appObserver.GetQueryMetricsHook())

	redisSession := datastores.RedisSession.GetClient().(*redis.Client)
	sessionCache := authz.NewSessionCache(redisSession, constants.ACCESS_TOKEN_TTL)
	tokenHasher := chi_token.NewHasher(cfg.Auth.TokenHashPepper)

	loadUserAccessDeps := authz.LoadUserAccessDeps{
		AdminAccessRepo:         repos.AdminAccess,
		UsersRepo:               repos.Users,
		UserRefreshTokensRepo:   repos.UserRefreshTokens,
		OrganizationsRepo:       repos.Organizations,
		OrganizationMembersRepo: repos.OrganizationMembers,
	}
	permissionResolver := authz.NewPermissionResolver(authz.PermissionResolverDeps{
		SessionCache:                    sessionCache,
		LoadUserAccess:                  loadUserAccessDeps,
		APIKeysRepo:                     repos.APIKeys,
		OrganizationsRepo:               repos.Organizations,
		OrganizationBillingAccountsRepo: repos.OrganizationBillingAccounts,
		HashToken:                       tokenHasher.Hash,
		Now:                             time.Now,
	})

	services := services.NewServices(datastores, cfg, logger, repos, sessionCache)
	authMiddleware := middleware.RequireAuth(cfg.Auth.AccessTokenSecret, permissionResolver, loadUserAccessDeps)

	redisRateLimit := datastores.RedisRateLimit.GetClient().(*redis.Client)
	rateLimiter := chi_ratelimit.NewRateLimiter(redisRateLimit, appObserver.Base, logger)

	webhook_handlers.NewBillingWebhookHandler(services.Billing, logger).RegisterRoutes(e)

	auth_handlers.NewAuthHandler(services.Auth, cfg, logger).RegisterRoutes(e, authMiddleware, rateLimiter)

	user_handlers.NewUsersHandler(services.User, logger).RegisterRoutes(e, authMiddleware, rateLimiter)

	organization_handlers.NewOrganizationsHandler(
		services.Organization,
		services.Billing,
		services.Audit,
		services.Role,
		services.Team,
		services.Invitation,
		services.APIKey,
		logger,
	).RegisterRoutes(e, authMiddleware, rateLimiter)

	support_handlers.NewSupportHandler(services.Support, logger).RegisterRoutes(e, authMiddleware, rateLimiter)

	location_handlers.NewLocationHandler(services.Location, logger).RegisterRoutes(e, authMiddleware, rateLimiter)

	adminRateLimit := rateLimiter.ScopedPrincipalRateLimit("100-H", "admin")
	impersonateRateLimit := rateLimiter.ScopedPrincipalRateLimit("10-H", "admin-impersonate")
	admin_handlers.NewAdminHandler(services.Auth, services.User, services.Organization, services.Audit, cfg, logger).
		RegisterRoutes(e, authMiddleware, middleware.RequirePlatformAdmin(), adminRateLimit, impersonateRateLimit)
}
