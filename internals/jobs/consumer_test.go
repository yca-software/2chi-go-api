package jobs

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	chi_aws_sqs "github.com/yca-software/2chi-go-aws/sqs"
	chi_error "github.com/yca-software/2chi-go-error"
	chi_logger "github.com/yca-software/2chi-go-logger"
)

func testJobsLogger() chi_logger.Logger {
	return chi_logger.New(chi_logger.LoggerConfig{OutputType: "json", ThresholdLevel: "error"})
}

func TestParseRetryCount(t *testing.T) {
	t.Parallel()
	require.Equal(t, 0, parseRetryCount(""))
	require.Equal(t, 2, parseRetryCount("2"))
	require.Equal(t, 0, parseRetryCount("bad"))
	require.Equal(t, 0, parseRetryCount("-1"))
}

func TestProcessMessage_SuccessDeletes(t *testing.T) {
	t.Parallel()
	sqs := &recordingSQS{}
	client := &Client{sqs: sqs, infraMaxRetries: 3, logger: testJobsLogger()}
	msg := chi_aws_sqs.QueueMessage{Body: "1", ReceiptHandle: "rh-1"}

	client.processMessage(context.Background(), "https://sqs/cleanup", QueueCleanup, msg, func() error {
		return nil
	}, "")

	require.Equal(t, []string{"rh-1"}, sqs.deleted)
	require.Empty(t, sqs.sentBodies)
}

func TestProcessMessage_NonRetryableDeletesWithoutRepublish(t *testing.T) {
	t.Parallel()
	sqs := &recordingSQS{}
	client := &Client{sqs: sqs, infraMaxRetries: 3, logger: testJobsLogger()}
	msg := chi_aws_sqs.QueueMessage{Body: "1", ReceiptHandle: "rh-2"}

	err := chi_error.NewBadRequestError(errors.New("bad"), "BadRequest", nil)
	client.processMessage(context.Background(), "https://sqs/cleanup", QueueCleanup, msg, func() error {
		return err
	}, "cleanup job failed")

	require.Equal(t, []string{"rh-2"}, sqs.deleted)
	require.Empty(t, sqs.sentBodies)
}

func TestProcessMessage_RetryableRepublishesWithIncrementedCount(t *testing.T) {
	t.Parallel()
	sqs := &recordingSQS{}
	client := &Client{sqs: sqs, infraMaxRetries: 3, logger: testJobsLogger()}
	msg := chi_aws_sqs.QueueMessage{
		Body:          "1",
		ReceiptHandle: "rh-3",
		Attributes:    map[string]string{attrRetryCount: "1"},
	}

	client.processMessage(context.Background(), "https://sqs/cleanup", QueueCleanup, msg, func() error {
		return Retryable(errors.New("timeout"))
	}, "cleanup job failed")

	require.Equal(t, []string{"rh-3"}, sqs.deleted)
	require.Len(t, sqs.sentBodies, 1)
	require.Equal(t, "2", sqs.sentAttrs[0][attrRetryCount])
}

type recordingSQS struct {
	deleted    []string
	sentBodies [][]byte
	sentAttrs  []map[string]string
}

func (r *recordingSQS) SendMessage(_ context.Context, _ string, body []byte, attrs map[string]string) error {
	r.sentBodies = append(r.sentBodies, body)
	r.sentAttrs = append(r.sentAttrs, attrs)
	return nil
}

func (r *recordingSQS) ReceiveMessages(context.Context, chi_aws_sqs.ReceiveOptions) ([]chi_aws_sqs.QueueMessage, error) {
	return nil, nil
}

func (r *recordingSQS) DeleteMessage(_ context.Context, _, receiptHandle string) error {
	r.deleted = append(r.deleted, receiptHandle)
	return nil
}

func (r *recordingSQS) ChangeMessageVisibility(context.Context, string, string, int32) error {
	return nil
}
