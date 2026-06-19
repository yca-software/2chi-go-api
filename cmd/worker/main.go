package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/yca-software/2chi-go-api/internals/config"
	"github.com/yca-software/2chi-go-api/internals/platform/logger"
	"github.com/yca-software/2chi-go-api/internals/platform/runtime"
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

	deps, err := runtime.BootstrapWorker(ctx, cfg, appLogger, "worker")
	if err != nil {
		log.Fatalf("failed to bootstrap worker: %v", err)
	}
	defer runtime.CloseDatastores(deps)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	var g errgroup.Group
	g.Go(func() error {
		return runtime.StartObservabilityServer(ctx, deps)
	})
	g.Go(func() error {
		appLogger.Info("starting SQS worker", "env", cfg.App.Environment, "metricsPort", cfg.Server.MetricsPort)
		return runtime.RunWorkerConsumers(ctx, deps)
	})

	go func() {
		<-sigChan
		appLogger.Info("worker shutdown signal received")
		cancel()
	}()

	if err := g.Wait(); err != nil && err != context.Canceled {
		log.Fatalf("worker error: %v", err)
	}
}
