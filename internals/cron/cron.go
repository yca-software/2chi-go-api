package cron

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/robfig/cron/v3"
	chi_logger "github.com/yca-software/2chi-go-logger"
)

const (
	cleanupArchivedInterval           = 24 * time.Hour
	applyScheduledPlanChangesInterval = time.Hour
)

// Deps wires cron scheduling dependencies.
type Deps struct {
	Cleanup                        func(context.Context) error
	PublishApplyScheduledPlanChanges func(context.Context) error
	Logger                         chi_logger.Logger
}

// Start registers interval tasks and returns a cancel func for shutdown.
func Start(deps Deps) (context.CancelFunc, error) {
	if deps.Cleanup == nil && deps.PublishApplyScheduledPlanChanges == nil {
		deps.Logger.Warn("no cron tasks configured; cron scheduler disabled")
		return nil, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	scheduler := cron.New()
	logger := deps.Logger

	if deps.Cleanup != nil {
		scheduleTask(ctx, scheduler, logger, cleanupArchivedInterval, "cleanup_archived", deps.Cleanup)
	}
	if deps.PublishApplyScheduledPlanChanges != nil {
		scheduleTask(ctx, scheduler, logger, applyScheduledPlanChangesInterval, "apply_scheduled_plan_changes", deps.PublishApplyScheduledPlanChanges)
	}

	scheduler.Start()
	go func() {
		<-ctx.Done()
		stopCtx := scheduler.Stop()
		<-stopCtx.Done()
	}()

	return cancel, nil
}

func scheduleTask(
	ctx context.Context,
	scheduler *cron.Cron,
	logger chi_logger.Logger,
	interval time.Duration,
	jobName string,
	run func(context.Context) error,
) {
	if err := run(ctx); err != nil {
		logger.Error("cron task failed", "job", jobName, "error", err)
	}

	var running atomic.Bool
	spec := "@every " + interval.String()
	if _, err := scheduler.AddFunc(spec, func() {
		if running.Swap(true) {
			return
		}
		defer running.Store(false)
		if ctx.Err() != nil {
			return
		}
		if err := run(ctx); err != nil && ctx.Err() == nil {
			logger.Error("cron task failed", "job", jobName, "error", err)
		}
	}); err != nil {
		logger.Error("failed to register cron job", "job", jobName, "error", err, "interval", interval.String())
	}
}
