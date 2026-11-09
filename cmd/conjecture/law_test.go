package main

import (
	"bytes"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	"github.com/farlleyferreira/conjecture-ai/internal/persistence"
	"github.com/farlleyferreira/conjecture-ai/internal/userdefinedlaws"
)

func openLawTestStore(t *testing.T) *sql.DB {
	t.Helper()
	db, err := persistence.OpenStoreAt(filepath.Join(t.TempDir(), "data"))
	if err != nil {
		t.Fatalf("OpenStoreAt returned unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

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
	var stdout, stderr bytes.Buffer
	code := runLaw([]string{"resolve-conflict", "--equal-authority"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("resolve-conflict with no law id: exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Errorf("stderr should contain a usage message, got %q", stderr.String())
	}
}

func TestRunLawResolveConflict_TwoPositionalArgsIsUsageError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runLaw([]string{"resolve-conflict", "law-one", "law-two"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("resolve-conflict with two positional args: exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Errorf("stderr should contain a usage message, got %q", stderr.String())
	}
}

func TestResolveLawConflict_FormalProofContradictsInvalidates(t *testing.T) {
	db := openLawTestStore(t)
	lawID := "demo-area::demo::some-rule::my-law"

	var stdout, stderr bytes.Buffer
	code := resolveLawConflict(db, lawID, true, false, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("resolveLawConflict returned exit code %d, want 0; stderr:\n%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), userdefinedlaws.AutoInvalidated.String()) {
		t.Errorf("stdout should report the auto-invalidated outcome, got:\n%s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "status set to invalidated") {
		t.Errorf("stdout should confirm the status was set, got:\n%s", stdout.String())
	}

	status, err := userdefinedlaws.GetStatus(db, lawID)
	if err != nil {
		t.Fatalf("GetStatus returned unexpected error: %v", err)
	}
	if status != userdefinedlaws.Invalidated {
		t.Errorf("GetStatus = %v, want Invalidated", status)
	}
}

func TestResolveLawConflict_EqualAuthorityRaisesQuestion(t *testing.T) {
	db := openLawTestStore(t)
	lawID := "demo-area::demo::some-rule::my-law"

	var stdout, stderr bytes.Buffer
	code := resolveLawConflict(db, lawID, false, true, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("resolveLawConflict returned exit code %d, want 0; stderr:\n%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "QUESTION") {
		t.Errorf("stdout should raise a flagged question, got:\n%s", stdout.String())
	}

	status, err := userdefinedlaws.GetStatus(db, lawID)
	if err != nil {
		t.Fatalf("GetStatus returned unexpected error: %v", err)
	}
	if status != userdefinedlaws.Unverified {
		t.Errorf("GetStatus = %v, want Unverified (raising a question changes no stored status)", status)
	}
}

func TestResolveLawConflict_NeitherFlagLawPrevails(t *testing.T) {
	db := openLawTestStore(t)
	lawID := "demo-area::demo::some-rule::my-law"

	var stdout, stderr bytes.Buffer
	code := resolveLawConflict(db, lawID, false, false, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("resolveLawConflict returned exit code %d, want 0; stderr:\n%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), userdefinedlaws.LawPrevails.String()) {
		t.Errorf("stdout should report the law-prevails outcome, got:\n%s", stdout.String())
	}

	status, err := userdefinedlaws.GetStatus(db, lawID)
	if err != nil {
		t.Fatalf("GetStatus returned unexpected error: %v", err)
	}
	if status != userdefinedlaws.Unverified {
		t.Errorf("GetStatus = %v, want Unverified (law-prevails changes no stored status)", status)
	}
}

func TestResolveLawConflict_FormalProofWinsEvenWithEqualAuthority(t *testing.T) {
	db := openLawTestStore(t)
	lawID := "demo-area::demo::some-rule::my-law"

	var stdout, stderr bytes.Buffer
	code := resolveLawConflict(db, lawID, true, true, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("resolveLawConflict returned exit code %d, want 0; stderr:\n%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), userdefinedlaws.AutoInvalidated.String()) {
		t.Errorf("stdout should report the auto-invalidated outcome regardless of authority, got:\n%s", stdout.String())
	}
}
