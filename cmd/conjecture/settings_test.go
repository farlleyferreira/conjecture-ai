package main

import (
	"bytes"
	"strings"
	"testing"
)

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

func TestRunSettingsCheckFloor_TwoPositionalArgsIsUsageError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runSettings([]string{"check-floor", "x", "y", "--local", "1", "--floor", "1", "--direction", "higher"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runSettings with two positional args: exit code = %d, want 1", code)
	}
}
