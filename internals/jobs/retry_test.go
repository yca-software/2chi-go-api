package jobs_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
	chi_error "github.com/yca-software/2chi-go-error"
	"github.com/yca-software/2chi-go-api/internals/jobs"
)

type RetrySuite struct {
	suite.Suite
}

func TestRetrySuite(t *testing.T) {
	suite.Run(t, new(RetrySuite))
}

func (s *RetrySuite) TestIsRetryable_InfrastructureMarker() {
	s.True(jobs.IsRetryable(jobs.Retryable(errors.New("sqs down"))))
}

func (s *RetrySuite) TestIsRetryable_API5xx() {
	err := chi_error.NewInternalServerError(errors.New("db"), "InternalServerError", nil)
	s.True(jobs.IsRetryable(err))
}

func (s *RetrySuite) TestIsRetryable_API4xx() {
	err := chi_error.NewBadRequestError(errors.New("bad"), "BadRequest", nil)
	s.False(jobs.IsRetryable(err))
}

func (s *RetrySuite) TestClassifyJobError_RetryUntilMax() {
	err := jobs.Retryable(errors.New("timeout"))
	s.False(jobs.ClassifyJobError(err, 1, 3))
	s.True(jobs.ClassifyJobError(err, 3, 3))
}

func (s *RetrySuite) TestClassifyJobError_NonRetryableDeadLetter() {
	err := chi_error.NewBadRequestError(errors.New("bad"), "BadRequest", nil)
	s.True(jobs.ClassifyJobError(err, 0, 3))
}
