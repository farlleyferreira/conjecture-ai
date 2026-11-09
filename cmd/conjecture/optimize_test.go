package main

import (
	"bytes"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	"github.com/farlleyferreira/conjecture-ai/internal/optimizationsuite"
	"github.com/farlleyferreira/conjecture-ai/internal/persistence"
	"github.com/farlleyferreira/conjecture-ai/internal/teamconfig"
)

func openOptimizeTestStore(t *testing.T) *sql.DB {
	t.Helper()
	db, err := persistence.OpenStoreAt(filepath.Join(t.TempDir(), "data"))
	if err != nil {
		t.Fatalf("OpenStoreAt returned unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := optimizationsuite.EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema returned unexpected error: %v", err)
	}
	return db
}

func TestRunOptimize_NoSubcommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runOptimize(nil, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runOptimize with no subcommand: exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Errorf("stderr should contain a usage message, got %q", stderr.String())
	}
}

func TestRunOptimize_UnknownSubcommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runOptimize([]string{"bogus"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runOptimize with an unknown subcommand: exit code = %d, want 1", code)
	}
}

func TestRunOptimizeSetCurrent_WrongArgCount(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runOptimize([]string{"set-current", "x"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("set-current with one argument: exit code = %d, want 1", code)
	}
}

func TestSetCurrent_ReportsTheValueSet(t *testing.T) {
	db := openOptimizeTestStore(t)

	var stdout, stderr bytes.Buffer
	code := setCurrent(db, "work_max_attempts_per_item", 3, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("setCurrent returned exit code %d, want 0; stderr:\n%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "current set to 3") {
		t.Errorf("stdout should report the value set, got:\n%s", stdout.String())
	}

	value, found, err := optimizationsuite.GetCurrent(db, "work_max_attempts_per_item")
	if err != nil {
		t.Fatalf("GetCurrent returned unexpected error: %v", err)
	}
	if !found || value != 3 {
		t.Errorf("GetCurrent = %v, %v, want 3, true", value, found)
	}
}

func TestCalibrate_ApprovedWithinFloorApplies(t *testing.T) {
	db := openOptimizeTestStore(t)
	if err := optimizationsuite.SetCurrent(db, "work_max_attempts_per_item", 3); err != nil {
		t.Fatalf("SetCurrent returned unexpected error: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := calibrate(db, "work_max_attempts_per_item", 5, true, 3, teamconfig.Higher, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("calibrate returned exit code %d, want 0; stderr:\n%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "current is now 5") {
		t.Errorf("stdout should report the new current value 5 (5 >= floor 3, Higher), got:\n%s", stdout.String())
	}

	value, found, err := optimizationsuite.GetCurrent(db, "work_max_attempts_per_item")
	if err != nil {
		t.Fatalf("GetCurrent returned unexpected error: %v", err)
	}
	if !found || value != 5 {
		t.Errorf("GetCurrent = %v, %v, want 5, true", value, found)
	}
}

func TestCalibrate_ApprovedBelowFloorClamped(t *testing.T) {
	db := openOptimizeTestStore(t)
	if err := optimizationsuite.SetCurrent(db, "work_max_attempts_per_item", 3); err != nil {
		t.Fatalf("SetCurrent returned unexpected error: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := calibrate(db, "work_max_attempts_per_item", 1, true, 3, teamconfig.Higher, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("calibrate returned exit code %d, want 0; stderr:\n%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "clamped at the floor") {
		t.Errorf("stdout should report clamping (proposal 1 < floor 3, Higher), got:\n%s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "current is now 3") {
		t.Errorf("stdout should report the clamped current value 3, got:\n%s", stdout.String())
	}
}

func TestCalibrate_NotApprovedLeavesCurrentUnchanged(t *testing.T) {
	db := openOptimizeTestStore(t)
	if err := optimizationsuite.SetCurrent(db, "work_max_attempts_per_item", 3); err != nil {
		t.Fatalf("SetCurrent returned unexpected error: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := calibrate(db, "work_max_attempts_per_item", 10, false, 3, teamconfig.Higher, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("calibrate returned exit code %d, want 0; stderr:\n%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "not approved") {
		t.Errorf("stdout should report the proposal as not approved, got:\n%s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "stays 3") {
		t.Errorf("stdout should report current staying at 3, got:\n%s", stdout.String())
	}
}

func TestCalibrate_NoCurrentYetIsError(t *testing.T) {
	db := openOptimizeTestStore(t)

	var stdout, stderr bytes.Buffer
	code := calibrate(db, "work_max_attempts_per_item", 5, true, 3, teamconfig.Higher, &stdout, &stderr)
	if code != 1 {
		t.Errorf("calibrate with no seeded current: exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "set-current") {
		t.Errorf("stderr should point at \"set-current\", got %q", stderr.String())
	}
}

func TestRunOptimizeCalibrate_UndeclaredFieldIsError(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	settingsPath := writeSettingsFixture(t, settingsFixture)

	var stdout, stderr bytes.Buffer
	code := runOptimize([]string{"calibrate", settingsPath, "no_such_field", "--proposal", "5", "--approved"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("calibrate with an undeclared field: exit code = %d, want 1", code)
	}
}

func TestRunOptimizeCalibrate_MissingProposalIsUsageError(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	settingsPath := writeSettingsFixture(t, settingsFixture)

	var stdout, stderr bytes.Buffer
	code := runOptimize([]string{"calibrate", settingsPath, "work_max_attempts_per_item"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("calibrate with no --proposal: exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Errorf("stderr should contain a usage message, got %q", stderr.String())
	}
}

func TestRunOptimizeCalibrate_MissingFilePathIsError(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	var stdout, stderr bytes.Buffer
	code := runOptimize([]string{"calibrate", "this-file-does-not-exist.cjtr", "work_max_attempts_per_item", "--proposal", "5", "--approved"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("calibrate with a missing settings file: exit code = %d, want 1", code)
	}
}
