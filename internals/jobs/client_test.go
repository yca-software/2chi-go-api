package jobs_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
	chi_aws_sqs "github.com/yca-software/2chi-go-aws/sqs"
	"github.com/yca-software/2chi-go-api/internals/jobs"
)

type ClientConfigSuite struct {
	suite.Suite
}

func TestClientConfigSuite(t *testing.T) {
	suite.Run(t, new(ClientConfigSuite))
}

func (s *ClientConfigSuite) TestNewClient_MissingSQS() {
	_, err := jobs.NewClient(jobs.Config{
		CleanupQueueURL:            "https://sqs/cleanup",
		ApplyScheduledPlanQueueURL: "https://sqs/apply",
		Logger:                     testLogger(),
	})
	s.Error(err)
	s.Contains(err.Error(), "SQS")
}

func (s *ClientConfigSuite) TestNewClient_MissingQueueURLs() {
	_, err := jobs.NewClient(jobs.Config{
		SQS:    stubSQS{},
		Logger: testLogger(),
	})
	s.Error(err)
	s.Contains(err.Error(), "queue URLs")
}

type stubSQS struct{}

func (stubSQS) SendMessage(context.Context, string, []byte, map[string]string) error {
	return nil
}
func (stubSQS) ReceiveMessages(context.Context, chi_aws_sqs.ReceiveOptions) ([]chi_aws_sqs.QueueMessage, error) {
	return nil, nil
}
func (stubSQS) DeleteMessage(context.Context, string, string) error { return nil }
func (stubSQS) ChangeMessageVisibility(context.Context, string, string, int32) error {
	return nil
}
