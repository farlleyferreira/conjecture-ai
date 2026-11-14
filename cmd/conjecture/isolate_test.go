package main

import (
	"runtime"
	"testing"
)

// isolateLocalStore points the per-user local data directory
// ([persistence.LocalDataDir]) at a fresh temporary directory for the
// rest of this test, on whichever OS it runs: a test that only set
// XDG_DATA_HOME was isolated on Linux alone, and on Windows and macOS
// read and wrote the developer's real store - sharing state across
// tests and across runs, and leaving test rows behind in it.
func isolateLocalStore(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	switch runtime.GOOS {
	case "windows":
		t.Setenv("LOCALAPPDATA", dir)
	case "darwin":
		t.Setenv("HOME", dir)
	default:
		t.Setenv("XDG_DATA_HOME", dir)
	}
}
