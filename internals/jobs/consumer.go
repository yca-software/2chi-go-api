package jobs

import (
	"context"
	"strconv"
	"time"

	chi_aws_sqs "github.com/yca-software/2chi-go-aws/sqs"
	chi_observer "github.com/yca-software/2chi-go-observer"
	"golang.org/x/sync/errgroup"
)

const attrRetryCount = "retry_count"

func (c *Client) consume(ctx context.Context, queueURL, queueName string, concurrency int, handler func() error, errLog string) error {
	if concurrency < 1 {
		concurrency = 1
	}
	g, ctx := errgroup.WithContext(ctx)
	for w := 0; w < concurrency; w++ {
		g.Go(func() error {
			return c.consumeWorker(ctx, queueURL, queueName, handler, errLog)
		})
	}
	return g.Wait()
}

func (c *Client) consumeWorker(ctx context.Context, queueURL, queueName string, handler func() error, errLog string) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		msgs, err := c.sqs.ReceiveMessages(ctx, chi_aws_sqs.ReceiveOptions{
			QueueURL:              queueURL,
			MaxMessages:           1,
			WaitTimeSeconds:       20,
			VisibilityTimeout:     300,
			MessageAttributeNames: []string{attrRetryCount},
		})
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			c.logger.WithContext(ctx).Error(errLog, "error", Retryable(err), "queue", queueName)
			continue
		}

		for _, msg := range msgs {
			c.processMessage(ctx, queueURL, queueName, msg, handler, errLog)
		}
	}
}

func (c *Client) processMessage(ctx context.Context, queueURL, queueName string, msg chi_aws_sqs.QueueMessage, handler func() error, errLog string) {
	retryCount := parseRetryCount(msg.Attributes[attrRetryCount])
	start := time.Now()
	err := handler()
	c.jobMetrics().RecordJobConsumerDuration(queueName, time.Since(start))

	if err == nil {
		c.jobMetrics().RecordJobConsumerOutcome(queueName, chi_observer.JobOutcomeSuccess)
		_ = c.sqs.DeleteMessage(ctx, queueURL, msg.ReceiptHandle)
		return
	}

	if errLog != "" {
		c.logger.WithContext(ctx).Error(errLog, "error", err, "queue", queueName, "retryCount", retryCount)
	}

	if ClassifyJobError(err, retryCount, c.infraMaxRetries) {
		c.jobMetrics().RecordJobConsumerOutcome(queueName, chi_observer.JobOutcomeDeadLetter)
		_ = c.sqs.DeleteMessage(ctx, queueURL, msg.ReceiptHandle)
		return
	}

	next := retryCount + 1
	republishErr := c.sqs.SendMessage(ctx, queueURL, []byte(msg.Body), map[string]string{
		attrRetryCount: strconv.Itoa(next),
	})
	if republishErr != nil {
		c.logger.WithContext(ctx).Error("jobs: republish failed, deleting message", "error", republishErr, "queue", queueName)
	}
	c.jobMetrics().RecordJobConsumerOutcome(queueName, chi_observer.JobOutcomeRetryRepublished)
	_ = c.sqs.DeleteMessage(ctx, queueURL, msg.ReceiptHandle)
}

func parseRetryCount(raw string) int {
	if raw == "" {
		return 0
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return 0
	}
	return n
}
