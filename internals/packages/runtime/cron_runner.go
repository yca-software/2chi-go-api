package runtime

import (
	"context"

	"github.com/yca-software/2chi-go-api/internals/cron"
	croncleanup "github.com/yca-software/2chi-go-api/internals/cron/cleanup"
)

// RunCron starts scheduled tasks until ctx is cancelled.
func RunCron(ctx context.Context, deps *WorkerDeps) error {
	cleanupRunner := croncleanup.NewRunner(deps.Logger,
		croncleanup.Step{Name: "organizations.CleanupArchived", Run: deps.Services.Organization.CleanupArchivedOrganizations},
		croncleanup.Step{Name: "users.CleanupStaleUnusedUserTokens", Run: deps.Services.User.CleanupStaleUnusedUserTokens},
		croncleanup.Step{Name: "users.CleanupArchivedUsers", Run: deps.Services.User.CleanupArchivedUsers},
		croncleanup.Step{Name: "invitations.CleanupStale", Run: deps.Services.Invitation.CleanupStale},
	)

	cronDeps := cron.Deps{
		Cleanup: cleanupRunner.Run,
		Logger:  deps.Logger,
	}
	if deps.JobClient != nil {
		cronDeps.PublishApplyScheduledPlanChanges = deps.JobClient.PublishApplyScheduledPlanChanges
	}

	cronCancel, err := cron.Start(cronDeps)
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
