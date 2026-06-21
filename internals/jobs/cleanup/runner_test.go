package cleanup_runner_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/suite"
	cleanup_runner "github.com/yca-software/2chi-go-api/internals/jobs/cleanup"
	chi_logger "github.com/yca-software/2chi-go-logger"
)

type CleanupRunnerSuite struct {
	suite.Suite
}

func TestCleanupRunnerSuite(t *testing.T) {
	suite.Run(t, new(CleanupRunnerSuite))
}

func (s *CleanupRunnerSuite) TestRun_AllStepsSucceed() {
	var calls atomic.Int32
	runner := cleanup_runner.NewCleanupRunner(testLogger(),
		cleanup_runner.CleanupStep{Name: "a", Run: func(context.Context) error { calls.Add(1); return nil }},
		cleanup_runner.CleanupStep{Name: "b", Run: func(context.Context) error { calls.Add(1); return nil }},
	)
	s.NoError(runner.Run(context.Background()))
	s.Equal(int32(2), calls.Load())
}

func (s *CleanupRunnerSuite) TestRun_StepErrorDoesNotFailJob() {
	runner := cleanup_runner.NewCleanupRunner(testLogger(),
		cleanup_runner.CleanupStep{Name: "fail", Run: func(context.Context) error {
			return errors.New("boom")
		}},
		cleanup_runner.CleanupStep{Name: "ok", Run: func(context.Context) error { return nil }},
	)
	s.NoError(runner.Run(context.Background()))
}

func (s *CleanupRunnerSuite) TestRun_IgnoresNoRowsAffected() {
	runner := cleanup_runner.NewCleanupRunner(testLogger(),
		cleanup_runner.CleanupStep{Name: "empty", Run: func(context.Context) error {
			return errors.New("no rows affected")
		}},
	)
	s.NoError(runner.Run(context.Background()))
}

func testLogger() chi_logger.Logger {
	return chi_logger.New(chi_logger.LoggerConfig{OutputType: "json", ThresholdLevel: "error"})
}
