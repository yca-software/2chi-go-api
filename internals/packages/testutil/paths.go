package testutil

import (
	"path/filepath"
	"runtime"
)

// ModuleRoot returns the go-api module root (directory containing go.mod).
func ModuleRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}

// MigrationsDir returns the path to the SQL migrations directory.
func MigrationsDir() string {
	return filepath.Join(ModuleRoot(), "migrations")
}

// TemplatesDir returns the path to the HTML email templates directory.
func TemplatesDir() string {
	return filepath.Join(ModuleRoot(), "templates")
}
