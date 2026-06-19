package jobs

import (
	"context"
	"fmt"

	chi_aws_sqs "github.com/yca-software/2chi-go-aws/sqs"
	chi_logger "github.com/yca-software/2chi-go-logger"
	chi_observer "github.com/yca-software/2chi-go-observer"
)

const (
	QueueCleanup                   = "cleanup"
	QueueApplyScheduledPlanChanges = "apply_scheduled_plan_changes"
)

const triggerBody = "1"

// Client publishes job triggers and runs SQS consumers.
type Client struct {
	sqs             chi_aws_sqs.SQS
	cleanupURL      string
	applyPlanURL    string
	logger          chi_logger.Logger
	metrics         chi_observer.JobMetricsHook
	infraMaxRetries int
}

// Config wires the jobs client and consumer concurrency (concurrency is passed per Run* call).
type Config struct {
	SQS                        chi_aws_sqs.SQS
	CleanupQueueURL            string
	ApplyScheduledPlanQueueURL string
	Logger                     chi_logger.Logger
	Metrics                    chi_observer.JobMetricsHook
	InfraMaxRetries            int
	CleanupConcurrency         int
	ApplyScheduledConcurrency  int
}

// NewClient validates config and returns an SQS jobs client.
func NewClient(cfg Config) (*Client, error) {
	if cfg.SQS == nil {
		return nil, fmt.Errorf("jobs: SQS client is required")
	}
	if cfg.CleanupQueueURL == "" || cfg.ApplyScheduledPlanQueueURL == "" {
		return nil, fmt.Errorf("jobs: cleanup and apply_scheduled_plan_changes queue URLs are required")
	}
	maxRetries := cfg.InfraMaxRetries
	if maxRetries < 1 {
		maxRetries = 3
	}
	metrics := cfg.Metrics
	if metrics == nil {
		metrics = chi_observer.NoopJobMetricsHook
	}

	return &Client{
		sqs:             cfg.SQS,
		cleanupURL:      cfg.CleanupQueueURL,
		applyPlanURL:    cfg.ApplyScheduledPlanQueueURL,
		logger:          cfg.Logger,
		metrics:         metrics,
		infraMaxRetries: maxRetries,
	}, nil
}

// PublishCleanup enqueues a cleanup job trigger.
func (c *Client) PublishCleanup(ctx context.Context) error {
	if err := c.publishTrigger(ctx, c.cleanupURL); err != nil {
		return err
	}
	c.jobMetrics().RecordJobPublished(QueueCleanup)
	return nil
}

// PublishApplyScheduledPlanChanges enqueues a scheduled plan apply trigger.
func (c *Client) PublishApplyScheduledPlanChanges(ctx context.Context) error {
	if err := c.publishTrigger(ctx, c.applyPlanURL); err != nil {
		return err
	}
	c.jobMetrics().RecordJobPublished(QueueApplyScheduledPlanChanges)
	return nil
}

func (c *Client) publishTrigger(ctx context.Context, queueURL string) error {
	return c.sqs.SendMessage(ctx, queueURL, []byte(triggerBody), nil)
}

// RunCleanupConsumer processes messages from the cleanup queue.
func (c *Client) RunCleanupConsumer(ctx context.Context, handler func() error, concurrency int) error {
	return c.consume(ctx, c.cleanupURL, QueueCleanup, concurrency, handler, "cleanup job failed")
}

// RunApplyScheduledPlanChangesConsumer processes messages from the apply-scheduled-plan queue.
func (c *Client) RunApplyScheduledPlanChangesConsumer(ctx context.Context, handler func() error, concurrency int) error {
	return c.consume(ctx, c.applyPlanURL, QueueApplyScheduledPlanChanges, concurrency, handler, "apply scheduled plan changes job failed")
}
