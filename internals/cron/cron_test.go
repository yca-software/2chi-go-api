package cron_test

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/suite"
	chi_logger "github.com/yca-software/2chi-go-logger"
	appcron "github.com/yca-software/2chi-go-api/internals/cron"
)

type CronSuite struct {
	suite.Suite
}

func TestCronSuite(t *testing.T) {
	suite.Run(t, new(CronSuite))
}

type mockTasks struct {
	cleanupCalls   atomic.Int64
	applyPlanCalls atomic.Int64
	cleanupErr     error
	applyPlanErr   error
}

func (m *mockTasks) RunCleanup(context.Context) error {
	m.cleanupCalls.Add(1)
	return m.cleanupErr
}

func (m *mockTasks) PublishApplyScheduledPlanChanges(context.Context) error {
	m.applyPlanCalls.Add(1)
	return m.applyPlanErr
}

func (s *CronSuite) TestStart_NoTasksReturnsNilCancel() {
	cancel, err := appcron.Start(appcron.Deps{Logger: testLogger()})
	s.NoError(err)
	s.Nil(cancel)
}

func (s *CronSuite) TestStart_RunsTasksImmediately() {
	tasks := &mockTasks{}
	cancel, err := appcron.Start(appcron.Deps{
		Cleanup:                        tasks.RunCleanup,
		PublishApplyScheduledPlanChanges: tasks.PublishApplyScheduledPlanChanges,
		Logger:                         testLogger(),
	})
	s.NoError(err)
	s.NotNil(cancel)
	defer cancel()

	s.Equal(int64(1), tasks.cleanupCalls.Load())
	s.Equal(int64(1), tasks.applyPlanCalls.Load())
}

func testLogger() chi_logger.Logger {
	return chi_logger.New(chi_logger.LoggerConfig{OutputType: "json", ThresholdLevel: "error"})
}
