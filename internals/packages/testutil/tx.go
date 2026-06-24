package testutil

import (
	"context"

	chi_repository "github.com/yca-software/2chi-go-repository"
)

// InlineRunInTx runs fn with a nil tx — sufficient for service unit tests using mocked repos.
func InlineRunInTx(_ context.Context, fn func(chi_repository.Tx) error) error {
	return fn(nil)
}
