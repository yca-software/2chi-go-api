package runtime

import (
	"context"
)

// RunWorkerConsumers starts configured SQS consumers until ctx is cancelled.
func RunWorkerConsumers(ctx context.Context, deps *WorkerDeps) error {
	return deps.JobClient.RunApplyScheduledPlanChangesConsumer(ctx, func() error {
		return deps.Services.Billing.ApplyScheduledPlanChanges(ctx)
	}, deps.Config.Jobs.ApplyScheduledPlanChanges.Concurrency)
}
