//go:build integration

package testutil

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	chi_test "github.com/yca-software/2chi-go-test"
)

const integrationLockFile = "2chi-go-api-integration.lock"

var integrationLock *os.File

func acquireIntegrationLock() error {
	path := filepath.Join(os.TempDir(), integrationLockFile)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return fmt.Errorf("integration lock open: %w", err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		_ = f.Close()
		return fmt.Errorf("integration lock acquire: %w", err)
	}
	integrationLock = f
	return nil
}

func releaseIntegrationLock() {
	if integrationLock == nil {
		return
	}
	_ = syscall.Flock(int(integrationLock.Fd()), syscall.LOCK_UN)
	_ = integrationLock.Close()
	integrationLock = nil
}

// IntegrationTestMain runs repository integration tests with a cross-process file lock
// so parallel packages share one reused PostGIS container without migration or data races.
// Use in TestMain: os.Exit(testutil.IntegrationTestMain(m))
func IntegrationTestMain(m *testing.M) int {
	if err := acquireIntegrationLock(); err != nil {
		fmt.Fprintf(os.Stderr, "testutil: %v\n", err)
		return 1
	}
	defer releaseIntegrationLock()

	code := m.Run()
	chi_test.Cleanup()
	return code
}

// GetIntegrationDB returns a migrated Postgres instance for repository integration tests.
func GetIntegrationDB() (*chi_test.DB, error) {
	return chi_test.Get(MigrationsDir())
}
