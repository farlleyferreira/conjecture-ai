package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// settingsFixture is a real, complete settings.cjtr - every setting
// this project's CLI currently discloses as a fixed default
// (lawContestedThreshold in check.go, queueAgingRate/queueAgingUnit
// in queue.go, workMaxAttemptsPerItem in work.go,
// maxBoundedProofValuesPerConcept/maxCompleteProofTotalCases in
// conformancecycle), given a real floor and direction here instead of
// staying a Go constant with nothing to compare a local override
// against.
const settingsFixture = `
conjecture: 1.0.0

settings:
  law_contested_threshold:
    floor: 3
    direction: lower
  queue_aging_rate:
    floor: 1.0
    direction: higher
  queue_aging_unit_seconds:
    floor: 86400
    direction: lower
  work_max_attempts_per_item:
    floor: 3
    direction: higher
  max_bounded_proof_values_per_concept:
    floor: 500
    direction: higher
  max_complete_proof_total_cases:
    floor: 2000
    direction: higher
`

func writeSettingsFixture(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "settings.cjtr")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	return path
}

func TestRunSettings_NoSubcommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runSettings(nil, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runSettings with no subcommand: exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Errorf("stderr should contain a usage message, got %q", stderr.String())
	}
}

func TestRunSettings_UnknownSubcommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runSettings([]string{"bogus"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runSettings with an unknown subcommand: exit code = %d, want 1", code)
	}
}

func TestRunSettingsCheckFloor_HigherStricterAccepted(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runSettings([]string{"check-floor", "required_cases", "--local", "1000", "--floor", "300", "--direction", "higher"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("runSettings returned exit code %d, want 0; stderr:\n%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "accepted") {
		t.Errorf("stdout should report acceptance, got:\n%s", stdout.String())
	}
}

func TestRunSettingsCheckFloor_HigherWeakerRejected(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runSettings([]string{"check-floor", "required_cases", "--local", "100", "--floor", "300", "--direction", "higher"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("runSettings returned exit code %d, want 1", code)
	}
	if !strings.Contains(stdout.String(), "rejected") {
		t.Errorf("stdout should report rejection, got:\n%s", stdout.String())
	}
}

func TestRunSettingsCheckFloor_LowerStricterAccepted(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runSettings([]string{"check-floor", "violation_rate", "--local", "0.3", "--floor", "1.0", "--direction", "lower"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("runSettings returned exit code %d, want 0; stderr:\n%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "accepted") {
		t.Errorf("stdout should report acceptance, got:\n%s", stdout.String())
	}
}

func TestRunSettingsCheckFloor_LowerWeakerRejected(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runSettings([]string{"check-floor", "violation_rate", "--local", "3.0", "--floor", "1.0", "--direction", "lower"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("runSettings returned exit code %d, want 1", code)
	}
	if !strings.Contains(stdout.String(), "rejected") {
		t.Errorf("stdout should report rejection, got:\n%s", stdout.String())
	}
}

func TestRunSettingsCheckFloor_InvalidDirectionIsError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runSettings([]string{"check-floor", "x", "--local", "1", "--floor", "1", "--direction", "sideways"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runSettings with an invalid direction: exit code = %d, want 1", code)
	}
}

func TestRunSettingsCheckFloor_NonNumericLocalIsError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runSettings([]string{"check-floor", "x", "--local", "not-a-number", "--floor", "1", "--direction", "higher"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runSettings with a non-numeric --local: exit code = %d, want 1", code)
	}
}

func TestRunSettingsCheckFloor_MissingFlagIsUsageError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runSettings([]string{"check-floor", "x", "--local", "1", "--floor", "1"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runSettings with a missing --direction: exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Errorf("stderr should contain a usage message, got %q", stderr.String())
	}
}

func TestRunSettingsVerify_ListsEveryDeclaredSetting(t *testing.T) {
	path := writeSettingsFixture(t, settingsFixture)

	var stdout, stderr bytes.Buffer
	code := runSettings([]string{"verify", path}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("runSettings verify returned exit code %d, want 0; stderr:\n%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "NOTICE") {
		t.Errorf("stdout should carry the unconfirmed-settings notice (the fixture has no \"signed\" block), got:\n%s", stdout.String())
	}
	for _, name := range []string{
		"law_contested_threshold", "queue_aging_rate", "queue_aging_unit_seconds",
		"work_max_attempts_per_item", "max_bounded_proof_values_per_concept", "max_complete_proof_total_cases",
	} {
		if !strings.Contains(stdout.String(), name) {
			t.Errorf("stdout should list %q, got:\n%s", name, stdout.String())
		}
	}
	if !strings.Contains(stdout.String(), "law_contested_threshold: floor=3 direction=lower") {
		t.Errorf("stdout should report law_contested_threshold's floor and direction, got:\n%s", stdout.String())
	}
}

func TestRunSettingsVerify_MissingFile(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runSettings([]string{"verify", "this-file-does-not-exist.cjtr"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runSettings verify with a missing file: exit code = %d, want 1", code)
	}
}

func TestRunSettingsVerify_WrongArgCount(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runSettings([]string{"verify"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runSettings verify with no path: exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Errorf("stderr should contain a usage message, got %q", stderr.String())
	}
}

func TestRunSettingsCheck_HigherStricterAcceptedFromFile(t *testing.T) {
	path := writeSettingsFixture(t, settingsFixture)

	var stdout, stderr bytes.Buffer
	code := runSettings([]string{"check", path, "work_max_attempts_per_item", "--local", "5"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("runSettings check returned exit code %d, want 0; stderr:\n%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "accepted") {
		t.Errorf("stdout should report acceptance (5 >= floor 3, Higher), got:\n%s", stdout.String())
	}
}

func TestRunSettingsCheck_HigherWeakerRejectedFromFile(t *testing.T) {
	path := writeSettingsFixture(t, settingsFixture)

	var stdout, stderr bytes.Buffer
	code := runSettings([]string{"check", path, "work_max_attempts_per_item", "--local", "1"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("runSettings check returned exit code %d, want 1", code)
	}
	if !strings.Contains(stdout.String(), "rejected") {
		t.Errorf("stdout should report rejection (1 < floor 3, Higher), got:\n%s", stdout.String())
	}
}

func TestRunSettingsCheck_LowerStricterAcceptedFromFile(t *testing.T) {
	path := writeSettingsFixture(t, settingsFixture)

	var stdout, stderr bytes.Buffer
	code := runSettings([]string{"check", path, "law_contested_threshold", "--local", "1"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("runSettings check returned exit code %d, want 0; stderr:\n%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "accepted") {
		t.Errorf("stdout should report acceptance (1 <= floor 3, Lower), got:\n%s", stdout.String())
	}
}

func TestRunSettingsCheck_UnknownSettingNameIsError(t *testing.T) {
	path := writeSettingsFixture(t, settingsFixture)

	var stdout, stderr bytes.Buffer
	code := runSettings([]string{"check", path, "no_such_setting", "--local", "1"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runSettings check with an undeclared setting name: exit code = %d, want 1", code)
	}
}

func TestRunSettingsCheck_MissingLocalIsUsageError(t *testing.T) {
	path := writeSettingsFixture(t, settingsFixture)

	var stdout, stderr bytes.Buffer
	code := runSettings([]string{"check", path, "law_contested_threshold"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runSettings check with no --local: exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Errorf("stderr should contain a usage message, got %q", stderr.String())
	}
}

func TestRunSettingsCheckFloor_TwoPositionalArgsIsUsageError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runSettings([]string{"check-floor", "x", "y", "--local", "1", "--floor", "1", "--direction", "higher"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runSettings with two positional args: exit code = %d, want 1", code)
	}
}
