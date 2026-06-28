package jobs

import (
	"errors"
	"fmt"
)

// Retryable marks an error as eligible for SQS consumer retry.
func Retryable(err error) error {
	if err == nil {
		return nil
	}
	return &retryableError{err: err}
}

type retryableError struct {
	err error
}

func (e *retryableError) Error() string {
	return e.err.Error()
}

func (e *retryableError) Unwrap() error {
	return e.err
}

// IsRetryable reports whether err (or its chain) was marked retryable.
func IsRetryable(err error) bool {
	var retryable *retryableError
	return errors.As(err, &retryable)
}

// ClassifyJobError returns true when the message should be dropped (dead-letter behavior).
func ClassifyJobError(err error, retryCount, maxRetries int) bool {
	if IsRetryable(err) {
		return retryCount >= maxRetries
	}
	return true
}

// PermanentJobError marks handler failures that should not be retried.
type PermanentJobError struct {
	Err error
}

func (e *PermanentJobError) Error() string {
	if e.Err == nil {
		return "permanent job error"
	}
	return fmt.Sprintf("permanent job error: %v", e.Err)
}

func (e *PermanentJobError) Unwrap() error {
	return e.Err
}
