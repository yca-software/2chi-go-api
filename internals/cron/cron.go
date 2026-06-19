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

// Publisher enqueues background job triggers (implemented by jobs.Client).
type Publisher interface {
	PublishCleanup(context.Context) error
	PublishApplyScheduledPlanChanges(context.Context) error
}

// Deps wires cron scheduling dependencies.
type Deps struct {
	Publisher Publisher
	Logger    chi_logger.Logger
}

// Start registers interval publishers and returns a cancel func for shutdown.
func Start(deps Deps) (context.CancelFunc, error) {
	if deps.Publisher == nil {
		deps.Logger.Warn("job publisher not configured; cron dispatcher disabled")
		return nil, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	scheduler := cron.New()
	logger := deps.Logger
	publisher := deps.Publisher

	schedulePublish(ctx, scheduler, logger, cleanupArchivedInterval, "cleanup_archived", publisher.PublishCleanup)
	schedulePublish(ctx, scheduler, logger, applyScheduledPlanChangesInterval, "apply_scheduled_plan_changes", publisher.PublishApplyScheduledPlanChanges)

	scheduler.Start()
	go func() {
		<-ctx.Done()
		stopCtx := scheduler.Stop()
		<-stopCtx.Done()
	}()

	return cancel, nil
}

// SchedulePublishForTest exposes schedule registration for unit tests.
func SchedulePublishForTest(
	ctx context.Context,
	scheduler *cron.Cron,
	logger chi_logger.Logger,
	interval time.Duration,
	jobName string,
	publish func(context.Context) error,
) {
	schedulePublish(ctx, scheduler, logger, interval, jobName, publish)
}

func schedulePublish(
	ctx context.Context,
	scheduler *cron.Cron,
	logger chi_logger.Logger,
	interval time.Duration,
	jobName string,
	publish func(context.Context) error,
) {
	if err := publish(ctx); err != nil {
		logger.Error("cron publish failed", "job", jobName, "error", err)
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
		if err := publish(ctx); err != nil && ctx.Err() == nil {
			logger.Error("cron publish failed", "job", jobName, "error", err)
		}
	}); err != nil {
		logger.Error("failed to register cron job", "job", jobName, "error", err, "interval", interval.String())
	}
}
