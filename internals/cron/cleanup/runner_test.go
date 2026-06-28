package cleanup_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/suite"
	croncleanup "github.com/yca-software/2chi-go-api/internals/cron/cleanup"
	chi_logger "github.com/yca-software/2chi-go-logger"
)

type RunnerSuite struct {
	suite.Suite
}

func TestRunnerSuite(t *testing.T) {
	suite.Run(t, new(RunnerSuite))
}

func (s *RunnerSuite) TestRun_AllStepsSucceed() {
	var calls atomic.Int32
	runner := croncleanup.NewRunner(testLogger(),
		croncleanup.Step{Name: "a", Run: func(context.Context) error { calls.Add(1); return nil }},
		croncleanup.Step{Name: "b", Run: func(context.Context) error { calls.Add(1); return nil }},
	)
	s.NoError(runner.Run(context.Background()))
	s.Equal(int32(2), calls.Load())
}

func (s *RunnerSuite) TestRun_StepErrorDoesNotFailTask() {
	runner := croncleanup.NewRunner(testLogger(),
		croncleanup.Step{Name: "fail", Run: func(context.Context) error {
			return errors.New("boom")
		}},
		croncleanup.Step{Name: "ok", Run: func(context.Context) error { return nil }},
	)
	s.NoError(runner.Run(context.Background()))
}

func (s *RunnerSuite) TestRun_IgnoresNoRowsAffected() {
	runner := croncleanup.NewRunner(testLogger(),
		croncleanup.Step{Name: "empty", Run: func(context.Context) error {
			return errors.New("no rows affected")
		}},
	)
	s.NoError(runner.Run(context.Background()))
}

func testLogger() chi_logger.Logger {
	return chi_logger.New(chi_logger.LoggerConfig{OutputType: "json", ThresholdLevel: "error"})
}
