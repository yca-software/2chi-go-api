package jobs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	chi_aws_sqs "github.com/yca-software/2chi-go-aws/sqs"
	chi_logger "github.com/yca-software/2chi-go-logger"
	chi_observer "github.com/yca-software/2chi-go-observer"
)

func TestProcessMessage_RecordsSuccessMetrics(t *testing.T) {
	t.Parallel()
	metrics := &recordingJobMetrics{}
	sqs := &recordingSQS{}
	client := &Client{sqs: sqs, infraMaxRetries: 3, logger: testJobsLogger(), metrics: metrics}
	msg := chi_aws_sqs.QueueMessage{Body: "1", ReceiptHandle: "rh-1"}

	client.processMessage(context.Background(), "https://sqs/apply", QueueApplyScheduledPlanChanges, msg, func() error {
		return nil
	}, "")

	require.Equal(t, [][2]string{{QueueApplyScheduledPlanChanges, chi_observer.JobOutcomeSuccess}}, metrics.outcomes)
	require.Len(t, metrics.durations, 1)
	require.Equal(t, QueueApplyScheduledPlanChanges, metrics.durations[0].job)
}

func TestProcessMessage_RetriesRetryableError(t *testing.T) {
	t.Parallel()
	metrics := &recordingJobMetrics{}
	sqs := &recordingSQS{}
	client := &Client{
		sqs:             sqs,
		applyPlanURL:    "https://sqs/apply",
		infraMaxRetries: 3,
		logger:          testJobsLogger(),
		metrics:         metrics,
	}
	msg := chi_aws_sqs.QueueMessage{Body: "1", ReceiptHandle: "rh-1"}

	client.processMessage(context.Background(), "https://sqs/apply", QueueApplyScheduledPlanChanges, msg, func() error {
		return Retryable(errors.New("timeout"))
	}, "")

	require.Equal(t, [][2]string{{QueueApplyScheduledPlanChanges, chi_observer.JobOutcomeRetryRepublished}}, metrics.outcomes)
	require.Len(t, sqs.sent, 1)
	require.Equal(t, "1", sqs.sent[0].attrs["retry_count"])
}

func TestProcessMessage_DropsPermanentError(t *testing.T) {
	t.Parallel()
	metrics := &recordingJobMetrics{}
	sqs := &recordingSQS{}
	client := &Client{sqs: sqs, infraMaxRetries: 3, logger: testJobsLogger(), metrics: metrics}
	msg := chi_aws_sqs.QueueMessage{Body: "1", ReceiptHandle: "rh-1"}

	client.processMessage(context.Background(), "https://sqs/apply", QueueApplyScheduledPlanChanges, msg, func() error {
		return errors.New("bad input")
	}, "")

	require.Equal(t, [][2]string{{QueueApplyScheduledPlanChanges, chi_observer.JobOutcomeDeadLetter}}, metrics.outcomes)
	require.Empty(t, sqs.sent)
}

func TestPublishApplyScheduledPlanChanges_RecordsPublishedMetric(t *testing.T) {
	t.Parallel()
	metrics := &recordingJobMetricsWithPublished{}
	sqs := &recordingSQS{}
	client := &Client{sqs: sqs, applyPlanURL: "https://sqs/apply", metrics: metrics}

	require.NoError(t, client.PublishApplyScheduledPlanChanges(context.Background()))
	require.Equal(t, []string{QueueApplyScheduledPlanChanges}, metrics.published)
}

type recordingJobMetrics struct {
	outcomes  [][2]string
	durations []struct {
		job string
		d   time.Duration
	}
}

func (r *recordingJobMetrics) RecordJobPublished(string) {}

func (r *recordingJobMetrics) RecordJobConsumerOutcome(job, outcome string) {
	r.outcomes = append(r.outcomes, [2]string{job, outcome})
}

func (r *recordingJobMetrics) RecordJobConsumerDuration(job string, duration time.Duration) {
	r.durations = append(r.durations, struct {
		job string
		d   time.Duration
	}{job, duration})
}

type recordingJobMetricsWithPublished struct {
	recordingJobMetrics
	published []string
}

func (r *recordingJobMetricsWithPublished) RecordJobPublished(job string) {
	r.published = append(r.published, job)
}

type recordingSQS struct {
	sent []sentMessage
}

type sentMessage struct {
	body  []byte
	attrs map[string]string
}

func (r *recordingSQS) SendMessage(_ context.Context, _ string, body []byte, attrs map[string]string) error {
	copied := make(map[string]string, len(attrs))
	for k, v := range attrs {
		copied[k] = v
	}
	r.sent = append(r.sent, sentMessage{body: body, attrs: copied})
	return nil
}

func (r *recordingSQS) ReceiveMessages(context.Context, chi_aws_sqs.ReceiveOptions) ([]chi_aws_sqs.QueueMessage, error) {
	return nil, nil
}

func (r *recordingSQS) DeleteMessage(context.Context, string, string) error { return nil }

func (r *recordingSQS) ChangeMessageVisibility(context.Context, string, string, int32) error {
	return nil
}

func testJobsLogger() chi_logger.Logger {
	return chi_logger.New(chi_logger.LoggerConfig{OutputType: "json", ThresholdLevel: "error"})
}
