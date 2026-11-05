package main

import (
	"bytes"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/farlleyferreira/conjecture-ai/internal/persistence"
)

func TestRunBase_MissingAddSubcommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runBase([]string{}, &stdout, &stderr)

	if code != 1 {
		t.Errorf("runBase with no subcommand exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Errorf("stderr should contain a usage message, got %q", stderr.String())
	}
}

func TestRunBase_UnknownSubcommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runBase([]string{"remove", "x"}, &stdout, &stderr)

	if code != 1 {
		t.Errorf("runBase with an unknown subcommand exit code = %d, want 1", code)
	}
}

func TestRunBase_MissingRequiredFlags(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runBase([]string{"add", "./normas/"}, &stdout, &stderr)

	if code != 1 {
		t.Errorf("runBase with no --authority/--scope exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Errorf("stderr should contain a usage message, got %q", stderr.String())
	}
}

func TestRunBase_FlagMissingValue(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runBase([]string{"add", "./normas/", "--authority"}, &stdout, &stderr)

	if code != 1 {
		t.Errorf("runBase with \"--authority\" missing its value exit code = %d, want 1", code)
	}
}

func TestRegisterBase_CompleteNormAuthority(t *testing.T) {
	db := openTestKnowledgeStore(t)

	path := filepath.Join(t.TempDir(), "lgpd.txt")
	if err := os.WriteFile(path, []byte("art. 1 ..."), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	stored, err := registerBase(db, path, "norm", "project", "2026-01-01")
	if err != nil {
		t.Fatalf("registerBase returned unexpected error: %v", err)
	}
	if stored.Incomplete {
		t.Error("stored.Incomplete = true, want false (a review date was given)")
	}
	if len(stored.ContentHash) == 0 {
		t.Error("stored.ContentHash is empty, want a computed hash")
	}
}

func TestRegisterBase_IncompleteNormAuthority(t *testing.T) {
	db := openTestKnowledgeStore(t)

	path := filepath.Join(t.TempDir(), "lgpd.txt")
	if err := os.WriteFile(path, []byte("art. 1 ..."), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	stored, err := registerBase(db, path, "norm", "project", "")
	if err != nil {
		t.Fatalf("registerBase returned unexpected error: %v", err)
	}
	if !stored.Incomplete {
		t.Error("stored.Incomplete = false, want true (no review date was given)")
	}
}

func TestRegisterBase_UnrecognizedAuthority(t *testing.T) {
	db := openTestKnowledgeStore(t)

	path := filepath.Join(t.TempDir(), "x.txt")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	_, err := registerBase(db, path, "opinion", "project", "")
	if err == nil {
		t.Fatal("registerBase with an unrecognized authority = nil error, want an error")
	}
}

func TestRegisterBase_MissingPath(t *testing.T) {
	db := openTestKnowledgeStore(t)

	_, err := registerBase(db, filepath.Join(t.TempDir(), "nope"), "reference", "project", "")
	if err == nil {
		t.Fatal("registerBase with a missing path = nil error, want an error")
	}
}

func openTestKnowledgeStore(t *testing.T) *sql.DB {
	t.Helper()
	db, err := persistence.OpenStoreAt(filepath.Join(t.TempDir(), "data"))
	if err != nil {
		t.Fatalf("OpenStoreAt returned unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}
