package runtime

import (
	"context"

	chi_server "github.com/yca-software/2chi-go-server"
)

// StartObservabilityServer exposes /health, /ready, and /metrics for worker/cron processes.
func StartObservabilityServer(ctx context.Context, deps *WorkerDeps) error {
	readiness := []chi_server.ReadinessDependency{deps.Datastores.Postgres}
	return chi_server.StartObservabilityServer(ctx, chi_server.ObservabilityConfig{
		Port:                  deps.Config.Server.MetricsPort,
		ReadinessDependencies: readiness,
	})
}
