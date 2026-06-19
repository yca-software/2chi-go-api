package jobs_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
	chi_logger "github.com/yca-software/2chi-go-logger"
	"github.com/yca-software/2chi-go-api/internals/jobs"
)

type CleanupRunnerSuite struct {
	suite.Suite
}

func TestCleanupRunnerSuite(t *testing.T) {
	suite.Run(t, new(CleanupRunnerSuite))
}

func (s *CleanupRunnerSuite) TestRun_AllStepsSucceed() {
	var calls int
	runner := jobs.NewCleanupRunner(testLogger(),
		jobs.CleanupStep{Name: "a", Run: func(context.Context) error { calls++; return nil }},
		jobs.CleanupStep{Name: "b", Run: func(context.Context) error { calls++; return nil }},
	)
	s.NoError(runner.Run(context.Background()))
	s.Equal(2, calls)
}

func (s *CleanupRunnerSuite) TestRun_StepErrorDoesNotFailJob() {
	runner := jobs.NewCleanupRunner(testLogger(),
		jobs.CleanupStep{Name: "fail", Run: func(context.Context) error {
			return errors.New("boom")
		}},
		jobs.CleanupStep{Name: "ok", Run: func(context.Context) error { return nil }},
	)
	s.NoError(runner.Run(context.Background()))
}

func (s *CleanupRunnerSuite) TestRun_IgnoresNoRowsAffected() {
	runner := jobs.NewCleanupRunner(testLogger(),
		jobs.CleanupStep{Name: "empty", Run: func(context.Context) error {
			return errors.New("no rows affected")
		}},
	)
	s.NoError(runner.Run(context.Background()))
}

func testLogger() chi_logger.Logger {
	return chi_logger.New(chi_logger.LoggerConfig{OutputType: "json", ThresholdLevel: "error"})
}
