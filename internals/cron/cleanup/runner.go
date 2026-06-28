package cleanup

import (
	"context"
	"strings"
	"sync"

	chi_error "github.com/yca-software/2chi-go-error"
	chi_logger "github.com/yca-software/2chi-go-logger"
)

// Step is a single parallel cleanup task.
type Step struct {
	Name string
	Run  func(context.Context) error
}

// Runner runs cleanup steps in parallel.
type Runner struct {
	wg     sync.WaitGroup
	logger chi_logger.Logger
	steps  []Step
}

// NewRunner builds a runner for the given steps.
func NewRunner(logger chi_logger.Logger, steps ...Step) *Runner {
	return &Runner{logger: logger, steps: steps}
}

// Run executes all steps concurrently; step errors are logged but do not fail the task.
func (r *Runner) Run(ctx context.Context) error {
	for _, step := range r.steps {
		r.wg.Add(1)
		go func(s Step) {
			defer r.wg.Done()
			if err := s.Run(ctx); err != nil {
				if isNoRowsAffected(err) {
					return
				}
				r.logger.WithContext(ctx).Error("cleanup step failed", "step", s.Name, "error", err)
			}
		}(step)
	}
	r.wg.Wait()
	return nil
}

func isNoRowsAffected(err error) bool {
	if err == nil {
		return false
	}
	if strings.Contains(strings.ToLower(err.Error()), "no rows affected") {
		return true
	}
	if apiErr, ok := chi_error.AsError(err); ok && apiErr.Err != nil {
		return strings.Contains(strings.ToLower(apiErr.Err.Error()), "no rows affected")
	}
	return false
}
