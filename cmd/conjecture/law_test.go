package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/farlleyferreira/conjecture-ai/internal/persistence"
	"github.com/farlleyferreira/conjecture-ai/internal/userdefinedlaws"
)

func TestRunLaw_NoSubcommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runLaw(nil, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runLaw with no subcommand: exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Errorf("stderr should contain a usage message, got %q", stderr.String())
	}
}

func TestRunLaw_UnknownSubcommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runLaw([]string{"bogus"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runLaw with an unknown subcommand: exit code = %d, want 1", code)
	}
}

func TestRunLawResolveConflict_MissingLawIDIsUsageError(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	var stdout, stderr bytes.Buffer
	code := runLaw([]string{"resolve-conflict", "--equal-authority"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("resolve-conflict with no law id: exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Errorf("stderr should contain a usage message, got %q", stderr.String())
	}
}

func TestRunLawResolveConflict_FormalProofContradictsInvalidates(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	var stdout, stderr bytes.Buffer
	code := runLaw([]string{"resolve-conflict", "demo-area::demo::some-rule::my-law", "--formal-proof-contradicts"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("resolve-conflict returned exit code %d, want 0; stderr:\n%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), userdefinedlaws.AutoInvalidated.String()) {
		t.Errorf("stdout should report the auto-invalidated outcome, got:\n%s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "status set to invalidated") {
		t.Errorf("stdout should confirm the status was set, got:\n%s", stdout.String())
	}

	db, err := persistence.OpenStore(dir)
	if err != nil {
		t.Fatalf("persistence.OpenStore returned unexpected error: %v", err)
	}
	status, err := userdefinedlaws.GetStatus(db, "demo-area::demo::some-rule::my-law")
	if err != nil {
		t.Fatalf("GetStatus returned unexpected error: %v", err)
	}
	if status != userdefinedlaws.Invalidated {
		t.Errorf("GetStatus = %v, want Invalidated", status)
	}
}

func TestRunLawResolveConflict_EqualAuthorityRaisesQuestion(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	var stdout, stderr bytes.Buffer
	code := runLaw([]string{"resolve-conflict", "demo-area::demo::some-rule::my-law", "--equal-authority"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("resolve-conflict returned exit code %d, want 0; stderr:\n%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "QUESTION") {
		t.Errorf("stdout should raise a flagged question, got:\n%s", stdout.String())
	}

	db, err := persistence.OpenStore(dir)
	if err != nil {
		t.Fatalf("persistence.OpenStore returned unexpected error: %v", err)
	}
	status, err := userdefinedlaws.GetStatus(db, "demo-area::demo::some-rule::my-law")
	if err != nil {
		t.Fatalf("GetStatus returned unexpected error: %v", err)
	}
	if status != userdefinedlaws.Unverified {
		t.Errorf("GetStatus = %v, want Unverified (raising a question changes no stored status)", status)
	}
}

func TestRunLawResolveConflict_NeitherFlagLawPrevails(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	var stdout, stderr bytes.Buffer
	code := runLaw([]string{"resolve-conflict", "demo-area::demo::some-rule::my-law"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("resolve-conflict returned exit code %d, want 0; stderr:\n%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), userdefinedlaws.LawPrevails.String()) {
		t.Errorf("stdout should report the law-prevails outcome, got:\n%s", stdout.String())
	}
}
