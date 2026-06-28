package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/yca-software/2chi-go-api/internals/config"
	"github.com/yca-software/2chi-go-api/internals/packages/logger"
	"github.com/yca-software/2chi-go-api/internals/packages/runtime"
	chi_logger "github.com/yca-software/2chi-go-logger"
	"golang.org/x/sync/errgroup"
)

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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	deps, err := runtime.BootstrapWorker(ctx, cfg, appLogger, "cron")
	if err != nil {
		log.Fatalf("failed to bootstrap cron: %v", err)
	}
	defer runtime.CloseDatastores(deps)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	var g errgroup.Group
	g.Go(func() error {
		return runtime.StartObservabilityServer(ctx, deps)
	})
	g.Go(func() error {
		appLogger.Info("starting cron scheduler", "env", cfg.App.Environment, "metricsPort", cfg.Server.MetricsPort)
		return runtime.RunCron(ctx, deps)
	})

	go func() {
		<-sigChan
		appLogger.Info("cron shutdown signal received")
		cancel()
	}()

	if err := g.Wait(); err != nil && err != context.Canceled {
		log.Fatalf("cron error: %v", err)
	}
}
