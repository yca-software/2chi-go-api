package gateway

import (
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
	"github.com/yca-software/2chi-go-api/internals/config"
	"github.com/yca-software/2chi-go-api/internals/constants"
	core_handlers "github.com/yca-software/2chi-go-api/internals/domains/core/handlers"
	core_repositories "github.com/yca-software/2chi-go-api/internals/domains/core/repositories"
	"github.com/yca-software/2chi-go-api/internals/domains/core/services"
	"github.com/yca-software/2chi-go-api/internals/domains/middleware"
	"github.com/yca-software/2chi-go-api/internals/platform/authz"
	"github.com/yca-software/2chi-go-api/internals/platform/datastores"
	"github.com/yca-software/2chi-go-api/internals/platform/observer"
	chi_logger "github.com/yca-software/2chi-go-logger"
	chi_ratelimit "github.com/yca-software/2chi-go-ratelimit"
	chi_token "github.com/yca-software/2chi-go-token"
)

func NewGateway(e *echo.Echo, datastores *datastores.Datastores, cfg *config.Config, appObserver *observer.Observer, logger chi_logger.Logger) {
	db := datastores.Postgres.GetClient().(*sqlx.DB)
	coreRepos := core_repositories.NewRepositories(db, appObserver.GetQueryMetricsHook())

	redisSession := datastores.RedisSession.GetClient().(*redis.Client)
	sessionCache := authz.NewSessionCache(redisSession, constants.ACCESS_TOKEN_TTL)
	tokenHasher := chi_token.NewHasher(cfg.Auth.TokenHashPepper)

	loadUserAccessDeps := authz.LoadUserAccessDeps{
		AdminAccessRepo:         coreRepos.AdminAccess,
		UsersRepo:               coreRepos.Users,
		UserRefreshTokensRepo:   coreRepos.UserRefreshTokens,
		OrganizationsRepo:       coreRepos.Organizations,
		OrganizationMembersRepo: coreRepos.OrganizationMembers,
	}

	permissionResolver := authz.NewPermissionResolver(authz.PermissionResolverDeps{
		SessionCache:                    sessionCache,
		LoadUserAccess:                  loadUserAccessDeps,
		APIKeysRepo:                     coreRepos.APIKeys,
		OrganizationBillingAccountsRepo: coreRepos.OrganizationBillingAccounts,
		HashToken:                       tokenHasher.Hash,
		Now:                             time.Now,
	})

	coreServices := services.NewCoreServices(datastores, cfg, logger, coreRepos, sessionCache)
	authMiddleware := middleware.RequireAuth(cfg.Auth.AccessTokenSecret, permissionResolver, loadUserAccessDeps)

	redisRateLimit := datastores.RedisRateLimit.GetClient().(*redis.Client)
	rateLimiter := chi_ratelimit.NewRateLimiter(redisRateLimit, appObserver.Base, logger)

	core_handlers.NewBillingWebhookHandler(coreServices.Billing, logger).RegisterRoutes(e)

	core_handlers.NewAuthHandler(coreServices.Auth, cfg, logger).RegisterRoutes(e, authMiddleware, rateLimiter)

	core_handlers.NewUsersHandler(coreServices.User, logger).RegisterRoutes(e, authMiddleware, rateLimiter)

	core_handlers.NewOrganizationsHandler(
		coreServices.Organization,
		coreServices.Billing,
		coreServices.Audit,
		coreServices.Role,
		coreServices.Team,
		coreServices.Invitation,
		coreServices.APIKey,
		logger,
	).RegisterRoutes(e, authMiddleware, rateLimiter)

	core_handlers.NewSupportHandler(coreServices.Support, logger).RegisterRoutes(e, authMiddleware, rateLimiter)

	core_handlers.NewLocationHandler(coreServices.Location, logger).RegisterRoutes(e, authMiddleware, rateLimiter)

	adminRateLimit := rateLimiter.ScopedPrincipalRateLimit("100-H", "admin")
	impersonateRateLimit := rateLimiter.ScopedPrincipalRateLimit("10-H", "admin-impersonate")
	core_handlers.NewAdminHandler(coreServices.Auth, coreServices.User, coreServices.Organization, coreServices.Audit, cfg, logger).
		RegisterRoutes(e, authMiddleware, middleware.RequirePlatformAdmin(), adminRateLimit, impersonateRateLimit)
}
