package runtime

import (
	"context"

	"github.com/yca-software/2chi-go-api/internals/cron"
)

// RunCronPublisher starts interval job publishers until ctx is cancelled.
func RunCronPublisher(ctx context.Context, deps *WorkerDeps) error {
	cronCancel, err := cron.Start(cron.Deps{
		Publisher: deps.JobClient,
		Logger:    deps.Logger,
	})
	if err != nil {
		return err
	}
	if cronCancel == nil {
		<-ctx.Done()
		return ctx.Err()
	}
	defer cronCancel()
	<-ctx.Done()
	return ctx.Err()
}
