package jobs

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	chi_aws_sqs "github.com/yca-software/2chi-go-aws/sqs"
	chi_observer "github.com/yca-software/2chi-go-observer"
)

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

func TestProcessMessage_RecordsSuccessMetrics(t *testing.T) {
	t.Parallel()
	metrics := &recordingJobMetrics{}
	sqs := &recordingSQS{}
	client := &Client{sqs: sqs, infraMaxRetries: 3, logger: testJobsLogger(), metrics: metrics}
	msg := chi_aws_sqs.QueueMessage{Body: "1", ReceiptHandle: "rh-1"}

	client.processMessage(context.Background(), "https://sqs/cleanup", QueueCleanup, msg, func() error {
		return nil
	}, "")

	require.Equal(t, [][2]string{{QueueCleanup, chi_observer.JobOutcomeSuccess}}, metrics.outcomes)
	require.Len(t, metrics.durations, 1)
	require.Equal(t, QueueCleanup, metrics.durations[0].job)
}

func TestPublishCleanup_RecordsPublishedMetric(t *testing.T) {
	t.Parallel()
	metrics := &recordingJobMetricsWithPublished{}
	sqs := &recordingSQS{}
	client := &Client{sqs: sqs, cleanupURL: "https://sqs/cleanup", metrics: metrics}

	require.NoError(t, client.PublishCleanup(context.Background()))
	require.Equal(t, []string{QueueCleanup}, metrics.published)
}

type recordingJobMetricsWithPublished struct {
	recordingJobMetrics
	published []string
}

func (r *recordingJobMetricsWithPublished) RecordJobPublished(job string) {
	r.published = append(r.published, job)
}
