package jobs

import (
	"errors"
	"net/http"

	chi_error "github.com/yca-software/2chi-go-error"
)

type errRetryableMarker struct{ err error }

func (e *errRetryableMarker) Error() string { return e.err.Error() }
func (e *errRetryableMarker) Unwrap() error { return e.err }

// Retryable marks an error as infrastructure-retryable for SQS consumers.
func Retryable(err error) error {
	if err == nil {
		return nil
	}
	return &errRetryableMarker{err: err}
}

// IsRetryable reports whether err should be retried before dead-lettering.
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}
	if _, ok := errors.AsType[*errRetryableMarker](err); ok {
		return true
	}
	var apiErr *chi_error.Error
	if errors.As(err, &apiErr) && apiErr != nil {
		code := apiErr.StatusCode
		return code >= http.StatusInternalServerError && code <= 599
	}
	return false
}

// ClassifyJobError reports whether a failed job message should be dead-lettered (not retried).
func ClassifyJobError(err error, retryCount, infraMax int) bool {
	return classifyJobError(err, retryCount, infraMax)
}

func classifyJobError(err error, retryCount, infraMax int) (deadLetter bool) {
	if !IsRetryable(err) {
		return true
	}
	if retryCount >= infraMax {
		return true
	}
	return false
}
