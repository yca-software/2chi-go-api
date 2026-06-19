package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/yca-software/2chi-go-api/internals/config"
	"github.com/yca-software/2chi-go-api/internals/gateway"
	"github.com/yca-software/2chi-go-api/internals/platform/datastores"
	"github.com/yca-software/2chi-go-api/internals/platform/logger"
	"github.com/yca-software/2chi-go-api/internals/platform/observer"
	chi_logger "github.com/yca-software/2chi-go-logger"
	chi_ratelimit "github.com/yca-software/2chi-go-ratelimit"
	chi_server "github.com/yca-software/2chi-go-server"
)

// @title           2Chi Go API
// @version         1.0
// @description     2Chi Go API.
//
// @host      https://api.yourdomain.com
// @BasePath  /
//
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT access token.
//
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-API-Key
// @description Organization API key (alternative to JWT Bearer token).
func main() {
	cfg, err := config.Init()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	appLogger := chi_logger.New(chi_logger.LoggerConfig{
		OutputType:       cfg.Logger.Format,
		ThresholdLevel:   cfg.Logger.Level,
		ContextExtractor: logger.ContextExtractor,
	})
	appObserver := observer.New(cfg)

	appDatastores, err := datastores.New(cfg, appLogger)
	if err != nil {
		log.Fatalf("failed to create datastores: %v", err)
	}

	readinessDeps := []chi_server.ReadinessDependency{appDatastores.Postgres, appDatastores.RedisSession, appDatastores.RedisRateLimit}
	cleanupDeps := []chi_server.CleanupDependency{appDatastores.Postgres, appDatastores.RedisSession, appDatastores.RedisRateLimit}

	metricsCtx, metricsCancel := context.WithCancel(context.Background())
	defer metricsCancel()
	chi_server.StartDedicatedMetricsServer(metricsCtx, cfg.Server.MetricsPort, readinessDeps)
	defer func() {
		chi_server.CleanupDependenciesConcurrently(cleanupDeps)
	}()

	appServer := chi_server.New(chi_server.ServerConfig{
		Port:               cfg.Server.Port,
		CORSAllowOrigins:   cfg.Server.CORSAllowOrigins,
		BodyLimit:          cfg.Server.BodyLimit,
		ServerReadTimeout:  cfg.Server.ServerReadTimeout,
		ServerWriteTimeout: cfg.Server.ServerWriteTimeout,
		ServerIdleTimeout:  cfg.Server.ServerIdleTimeout,
		Logger:             appLogger,
		Observer:           appObserver,
		RegisterRoutes: func(e *echo.Echo) {
			e.Use(chi_ratelimit.EnsureDeviceID(cfg.App.Environment))

			chi_server.RegisterHealthHandlers(e, readinessDeps)

			gateway.NewGateway(e, appDatastores, cfg, appObserver, appLogger)
		},
	})

	serverErr := make(chan error, 1)
	go func() {
		log.Printf("Starting server on port %d", cfg.Server.Port)
		if err := appServer.Start(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		log.Printf("Received signal: %v", sig)
	case err := <-serverErr:
		log.Printf("Server error: %v", err)
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		_ = appServer.Shutdown(shutdownCtx)
		shutdownCancel()
		return
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := appServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("Failed to shutdown server: %v", err)
		return
	}
}
