package jobs_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/yca-software/2chi-go-api/internals/jobs"
)

type RetrySuite struct {
	suite.Suite
}

func TestRetrySuite(t *testing.T) {
	suite.Run(t, new(RetrySuite))
}

func (s *RetrySuite) TestIsRetryable() {
	s.True(jobs.IsRetryable(jobs.Retryable(errors.New("sqs down"))))
}

func (s *RetrySuite) TestClassifyJobError_RetryableWithinLimit() {
	err := jobs.Retryable(errors.New("timeout"))
	s.False(jobs.ClassifyJobError(err, 1, 3))
}

func (s *RetrySuite) TestClassifyJobError_RetryableExhausted() {
	err := jobs.Retryable(errors.New("timeout"))
	s.True(jobs.ClassifyJobError(err, 3, 3))
}

func (s *RetrySuite) TestClassifyJobError_Permanent() {
	s.True(jobs.ClassifyJobError(errors.New("bad input"), 0, 3))
}
