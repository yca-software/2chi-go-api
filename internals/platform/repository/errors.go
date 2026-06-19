package repository

import (
	"errors"
	"net/http"

	chi_error "github.com/yca-software/2chi-go-error"
)

func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	if apiErr, ok := errors.AsType[*chi_error.Error](err); ok {
		return apiErr.StatusCode == http.StatusNotFound
	}
	return false
}
