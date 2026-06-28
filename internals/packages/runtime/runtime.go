package runtime

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	"github.com/yca-software/2chi-go-api/internals/config"
	"github.com/yca-software/2chi-go-api/internals/constants"
	"github.com/yca-software/2chi-go-api/internals/jobs"
	"github.com/yca-software/2chi-go-api/internals/packages/authz"
	"github.com/yca-software/2chi-go-api/internals/packages/datastores"
	"github.com/yca-software/2chi-go-api/internals/packages/observer"
	core_repositories "github.com/yca-software/2chi-go-api/internals/repositories"
	"github.com/yca-software/2chi-go-api/internals/services"
	chi_aws "github.com/yca-software/2chi-go-aws"
	chi_aws_sqs "github.com/yca-software/2chi-go-aws/sqs"
	chi_logger "github.com/yca-software/2chi-go-logger"
)

// WorkerDeps bundles dependencies for the worker and cron processes.
type WorkerDeps struct {
	Config     *config.Config
	Logger     chi_logger.Logger
	Datastores *datastores.Datastores
	Services   *services.Services
	JobClient  *jobs.Client
	Observer   *observer.Observer
}

// BootstrapWorker wires datastores, services, and the SQS job client.
func BootstrapWorker(ctx context.Context, cfg *config.Config, logger chi_logger.Logger, appName string) (*WorkerDeps, error) {
	ds, err := datastores.New(cfg, logger)
	if err != nil {
		return nil, err
	}

	db := ds.Postgres.GetClient().(*sqlx.DB)
	appObserver := observer.NewForApp(cfg, appName)
	repos := core_repositories.NewRepositories(db, appObserver.GetQueryMetricsHook())

	redisSession := ds.RedisSession.GetClient().(*redis.Client)
	sessionCache := authz.NewSessionCache(redisSession, constants.ACCESS_TOKEN_TTL)
	srvs := services.NewServices(ds, cfg, logger, repos, sessionCache)

	awsModule, err := chi_aws.New(ctx, chi_aws.Config{
		Region:   cfg.AWS.DefaultRegion,
		Endpoint: cfg.AWS.DefaultEndpoint,
		SQS: &chi_aws_sqs.Config{
			Region:   cfg.AWS.DefaultRegion,
			Endpoint: cfg.AWS.DefaultEndpoint,
		},
	})
	if err != nil {
		return nil, err
	}
	if awsModule.SQS == nil {
		return nil, fmt.Errorf("SQS is not configured")
	}

	jobClient, err := jobs.NewClient(jobs.Config{
		SQS:                        awsModule.SQS,
		ApplyScheduledPlanQueueURL: cfg.Jobs.ApplyScheduledPlanChanges.QueueURL,
		Logger:                     logger,
		Metrics:                    appObserver.GetJobMetricsHook(),
		InfraMaxRetries:            cfg.Jobs.MaxRetries,
		ApplyScheduledConcurrency:  cfg.Jobs.ApplyScheduledPlanChanges.Concurrency,
	})
	if err != nil {
		return nil, err
	}

	return &WorkerDeps{
		Config:     cfg,
		Logger:     logger,
		Datastores: ds,
		Services:   srvs,
		JobClient:  jobClient,
		Observer:   appObserver,
	}, nil
}

// CloseDatastores closes process datastores.
func CloseDatastores(deps *WorkerDeps) {
	if deps == nil || deps.Datastores == nil {
		return
	}
	deps.Datastores.Close()
}
