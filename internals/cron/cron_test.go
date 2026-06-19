package cron_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	robfigcron "github.com/robfig/cron/v3"
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

type mockPublisher struct {
	cleanupCalls   atomic.Int64
	applyPlanCalls atomic.Int64
	cleanupErr     error
	applyPlanErr   error
}

func (m *mockPublisher) PublishCleanup(context.Context) error {
	m.cleanupCalls.Add(1)
	return m.cleanupErr
}

func (m *mockPublisher) PublishApplyScheduledPlanChanges(context.Context) error {
	m.applyPlanCalls.Add(1)
	return m.applyPlanErr
}

func (s *CronSuite) TestStart_NilPublisherReturnsNilCancel() {
	cancel, err := appcron.Start(appcron.Deps{Logger: testLogger()})
	s.NoError(err)
	s.Nil(cancel)
}

func (s *CronSuite) TestSchedulePublish_ImmediateAndPeriodic() {
	pub := &mockPublisher{}
	scheduler := robfigcron.New()
	ctx := context.Background()

	appcron.SchedulePublishForTest(ctx, scheduler, testLogger(), 50*time.Millisecond, "cleanup_archived", pub.PublishCleanup)

	s.Equal(int64(1), pub.cleanupCalls.Load())
	s.Len(scheduler.Entries(), 1)

	scheduler.Entries()[0].Job.Run()
	s.Equal(int64(2), pub.cleanupCalls.Load())
}

func testLogger() chi_logger.Logger {
	return chi_logger.New(chi_logger.LoggerConfig{OutputType: "json", ThresholdLevel: "error"})
}
