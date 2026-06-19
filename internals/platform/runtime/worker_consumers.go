package runtime

import (
	"context"

	"github.com/yca-software/2chi-go-api/internals/jobs"
	"golang.org/x/sync/errgroup"
)

// RunWorkerConsumers starts all configured SQS consumers until ctx is cancelled.
func RunWorkerConsumers(ctx context.Context, deps *WorkerDeps) error {
	cleanupRunner := jobs.NewCleanupRunner(deps.Logger,
		jobs.CleanupStep{Name: "organizations.CleanupArchived", Run: deps.Services.Organization.CleanupArchivedOrganizations},
		jobs.CleanupStep{Name: "users.CleanupStaleUnusedUserTokens", Run: deps.Services.User.CleanupStaleUnusedUserTokens},
		jobs.CleanupStep{Name: "users.CleanupArchivedUsers", Run: deps.Services.User.CleanupArchivedUsers},
		jobs.CleanupStep{Name: "invitations.CleanupStale", Run: deps.Services.Invitation.CleanupStaleInvitations},
	)

	var workers errgroup.Group
	workers.Go(func() error {
		return deps.JobClient.RunCleanupConsumer(ctx, func() error {
			return cleanupRunner.Run(ctx)
		}, deps.Config.Jobs.Cleanup.Concurrency)
	})
	workers.Go(func() error {
		return deps.JobClient.RunApplyScheduledPlanChangesConsumer(ctx, func() error {
			return deps.Services.Billing.ApplyScheduledPlanChanges(ctx)
		}, deps.Config.Jobs.ApplyScheduledPlanChanges.Concurrency)
	})
	return workers.Wait()
}
