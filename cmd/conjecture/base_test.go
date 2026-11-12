package main

import (
	"bytes"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/farlleyferreira/conjecture-ai/internal/knowledgebases"
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

func TestRunBaseSearch_WrongArgCount(t *testing.T) {
	// No query at all: args fail validation before the real per-OS
	// store is ever touched, same as TestRunBase_UnknownSubcommand
	// already relies on for runBase itself - safe to call directly.
	var stdout, stderr bytes.Buffer
	code := runBaseSearch([]string{}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runBaseSearch with no query = %d, want 1", code)
	}
}

func TestSearchBases_NoMatchesReportsPlainly(t *testing.T) {
	db := openTestKnowledgeStore(t)
	var stdout, stderr bytes.Buffer
	code := searchBases(db, "nothing indexed matches this", "", false, &stdout, &stderr)
	if code != 0 {
		t.Errorf("searchBases with no matches = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "no matches") {
		t.Errorf("stdout = %q, want it to report no matches", stdout.String())
	}
}

func TestSearchBases_FindsWhatWasRegisteredAndIndexed(t *testing.T) {
	// Registration and indexing tested at their own store-independent
	// boundary (registerBase, knowledgebases.IndexBase - already
	// covered in depth by internal/knowledgebases/search_test.go), not
	// through the CLI dispatcher, same as every other "Add" assertion
	// in this file.
	db := openTestKnowledgeStore(t)
	path := filepath.Join(t.TempDir(), "lgpd.txt")
	if err := os.WriteFile(path, []byte("o titular pode solicitar a exclusao dos dados"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	stored, err := registerBase(db, path, "reference", "project", "")
	if err != nil {
		t.Fatalf("registerBase returned unexpected error: %v", err)
	}
	if err := knowledgebases.EnsureSearchSchema(db); err != nil {
		t.Fatalf("EnsureSearchSchema returned unexpected error: %v", err)
	}
	if err := knowledgebases.EnsureEmbeddingSchema(db); err != nil {
		t.Fatalf("EnsureEmbeddingSchema returned unexpected error: %v", err)
	}
	if err := knowledgebases.IndexBase(db, stored); err != nil {
		t.Fatalf("IndexBase returned unexpected error: %v", err)
	}

	var searchOut, searchErr bytes.Buffer
	code := searchBases(db, "exclusao", "", false, &searchOut, &searchErr)
	if code != 0 {
		t.Fatalf("searchBases = %d, want 0 (stderr: %q)", code, searchErr.String())
	}
	if !strings.Contains(searchOut.String(), path) {
		t.Errorf("search output = %q, want it to mention %q", searchOut.String(), path)
	}
}

func TestSearchBases_AreaExcludesBaseFromADifferentArea(t *testing.T) {
	db := openTestKnowledgeStore(t)

	shippingPath := filepath.Join(t.TempDir(), "shipping.txt")
	if err := os.WriteFile(shippingPath, []byte("limite de peso para o frete"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	shippingBase, err := registerBase(db, shippingPath, "reference", "area:shipping", "")
	if err != nil {
		t.Fatalf("registerBase returned unexpected error: %v", err)
	}
	if err := knowledgebases.EnsureSearchSchema(db); err != nil {
		t.Fatalf("EnsureSearchSchema returned unexpected error: %v", err)
	}
	if err := knowledgebases.EnsureEmbeddingSchema(db); err != nil {
		t.Fatalf("EnsureEmbeddingSchema returned unexpected error: %v", err)
	}
	if err := knowledgebases.IndexBase(db, shippingBase); err != nil {
		t.Fatalf("IndexBase returned unexpected error: %v", err)
	}

	billingPath := filepath.Join(t.TempDir(), "billing.txt")
	if err := os.WriteFile(billingPath, []byte("limite de peso para a cobranca"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	billingBase, err := registerBase(db, billingPath, "reference", "area:billing", "")
	if err != nil {
		t.Fatalf("registerBase returned unexpected error: %v", err)
	}
	if err := knowledgebases.IndexBase(db, billingBase); err != nil {
		t.Fatalf("IndexBase returned unexpected error: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := searchBases(db, "limite de peso", "shipping", false, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("searchBases = %d, want 0 (stderr: %q)", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), shippingPath) {
		t.Errorf("stdout = %q, want it to mention the shipping-scoped base %q", stdout.String(), shippingPath)
	}
	if strings.Contains(stdout.String(), billingPath) {
		t.Errorf("stdout = %q, should not mention the billing-scoped base %q", stdout.String(), billingPath)
	}
}

func TestRunBaseSearch_AreaFlagParsed(t *testing.T) {
	// --area with no value fails validation before the real per-OS
	// store is ever touched, safe to call directly.
	var stdout, stderr bytes.Buffer
	code := runBaseSearch([]string{"--area"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runBaseSearch with --area and no value = %d, want 1", code)
	}
}

func TestSearchBases_SemanticFindsWhatWasRegisteredAndIndexed(t *testing.T) {
	db := openTestKnowledgeStore(t)
	path := filepath.Join(t.TempDir(), "lgpd.txt")
	if err := os.WriteFile(path, []byte("o titular pode solicitar a exclusao dos dados"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	stored, err := registerBase(db, path, "reference", "project", "")
	if err != nil {
		t.Fatalf("registerBase returned unexpected error: %v", err)
	}
	if err := knowledgebases.EnsureSearchSchema(db); err != nil {
		t.Fatalf("EnsureSearchSchema returned unexpected error: %v", err)
	}
	if err := knowledgebases.EnsureEmbeddingSchema(db); err != nil {
		t.Fatalf("EnsureEmbeddingSchema returned unexpected error: %v", err)
	}
	if err := knowledgebases.IndexBase(db, stored); err != nil {
		t.Fatalf("IndexBase returned unexpected error: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := searchBases(db, "o titular pode solicitar a exclusao dos dados", "", true, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("searchBases(semantic) = %d, want 0 (stderr: %q)", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), path) {
		t.Errorf("search output = %q, want it to mention %q", stdout.String(), path)
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
