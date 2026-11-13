package main

import (
	"bytes"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/farlleyferreira/conjecture-ai/internal/contextcomposition"
	"github.com/farlleyferreira/conjecture-ai/internal/decisionledger"
	"github.com/farlleyferreira/conjecture-ai/internal/persistence"
	"github.com/farlleyferreira/conjecture-ai/internal/workqueue"
)

func openTestContextStore(t *testing.T) *sql.DB {
	t.Helper()
	db, err := persistence.OpenStoreAt(filepath.Join(t.TempDir(), "data"))
	if err != nil {
		t.Fatalf("OpenStoreAt returned unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := contextcomposition.EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema returned unexpected error: %v", err)
	}
	return db
}

func TestContextList_AreaFlagFiltersByArea(t *testing.T) {
	db := openTestContextStore(t)
	if _, err := contextcomposition.CreateContext(db, "checkout", "web"); err != nil {
		t.Fatalf("CreateContext(checkout) returned unexpected error: %v", err)
	}
	if _, err := contextcomposition.CreateContext(db, "cart", "mobile"); err != nil {
		t.Fatalf("CreateContext(cart) returned unexpected error: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := contextList(db, []string{"--area", "web"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("contextList returned exit code %d, stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "checkout") {
		t.Errorf("stdout = %q, want it to mention \"checkout\"", stdout.String())
	}
	if strings.Contains(stdout.String(), "cart") {
		t.Errorf("stdout = %q, want it to NOT mention \"cart\" (different area)", stdout.String())
	}
}

func TestContextList_MalformedSinceIsError(t *testing.T) {
	db := openTestContextStore(t)
	var stdout, stderr bytes.Buffer
	code := contextList(db, []string{"--since", "not-a-date"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("contextList with a malformed --since: exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "--since") {
		t.Errorf("stderr should mention --since, got %q", stderr.String())
	}
}

func TestContextList_UnknownFlagIsUsageError(t *testing.T) {
	db := openTestContextStore(t)
	var stdout, stderr bytes.Buffer
	code := contextList(db, []string{"--bogus", "x"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("contextList with an unknown flag: exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Errorf("stderr should contain a usage message, got %q", stderr.String())
	}
}

func TestRunContext_NoArguments(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runContext(nil, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runContext with no arguments: exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Errorf("stderr should contain a usage message, got %q", stderr.String())
	}
}

func TestRunContext_UnknownSubcommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runContext([]string{"delete", "checkout"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runContext with an unknown subcommand: exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Errorf("stderr should contain a usage message, got %q", stderr.String())
	}
}

func TestContextList_NoContexts(t *testing.T) {
	db := openTestContextStore(t)
	var stdout, stderr bytes.Buffer
	code := contextList(db, nil, &stdout, &stderr)
	if code != 0 {
		t.Errorf("contextList on an empty store: exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "no contexts") {
		t.Errorf("stdout = %q, want it to say there are no contexts", stdout.String())
	}
}

func TestContextList_ReportsStateAndSupersededBy(t *testing.T) {
	db := openTestContextStore(t)
	if _, err := contextcomposition.CreateContext(db, "checkout", ""); err != nil {
		t.Fatalf("CreateContext returned unexpected error: %v", err)
	}
	if _, err := contextcomposition.ArchiveContext(db, "checkout", "new-checkout"); err != nil {
		t.Fatalf("ArchiveContext returned unexpected error: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := contextList(db, nil, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("contextList returned exit code %d, stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "checkout") || !strings.Contains(stdout.String(), "Archived") {
		t.Errorf("stdout = %q, want it to mention \"checkout\" and \"Archived\"", stdout.String())
	}
	if !strings.Contains(stdout.String(), "new-checkout") {
		t.Errorf("stdout = %q, want it to mention the superseding context \"new-checkout\"", stdout.String())
	}
}

func TestContextResume_ReturnsLatestVersionContent(t *testing.T) {
	db := openTestContextStore(t)
	if _, err := contextcomposition.CreateContext(db, "checkout", ""); err != nil {
		t.Fatalf("CreateContext returned unexpected error: %v", err)
	}
	if _, err := contextcomposition.AddVersion(db, "checkout", "old state"); err != nil {
		t.Fatalf("AddVersion returned unexpected error: %v", err)
	}
	if _, err := contextcomposition.AddVersion(db, "checkout", "latest state"); err != nil {
		t.Fatalf("AddVersion returned unexpected error: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := contextResume(db, []string{"checkout"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("contextResume returned exit code %d, stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "latest state") {
		t.Errorf("stdout = %q, want it to contain the latest version's content", stdout.String())
	}
	if strings.Contains(stdout.String(), "old state") {
		t.Errorf("stdout = %q, should not contain the older version's content", stdout.String())
	}
}

func TestContextResume_ReportsEmptyQueue(t *testing.T) {
	db := openTestContextStore(t)
	if _, err := contextcomposition.CreateContext(db, "checkout", ""); err != nil {
		t.Fatalf("CreateContext returned unexpected error: %v", err)
	}
	if _, err := contextcomposition.AddVersion(db, "checkout", "state"); err != nil {
		t.Fatalf("AddVersion returned unexpected error: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := contextResume(db, []string{"checkout"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("contextResume returned exit code %d, stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "next work: queue is empty") {
		t.Errorf("stdout = %q, want it to report the queue is empty", stdout.String())
	}
}

func TestContextResume_ReportsHighestPriorityQueueItem(t *testing.T) {
	db := openTestContextStore(t)
	if _, err := contextcomposition.CreateContext(db, "checkout", ""); err != nil {
		t.Fatalf("CreateContext returned unexpected error: %v", err)
	}
	if _, err := contextcomposition.AddVersion(db, "checkout", "state"); err != nil {
		t.Fatalf("AddVersion returned unexpected error: %v", err)
	}
	if err := workqueue.EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema returned unexpected error: %v", err)
	}
	now := time.Now().UTC()
	if err := workqueue.Add(db, workqueue.Item{ID: "background-item", Tier: workqueue.Background, CreatedAt: now}); err != nil {
		t.Fatalf("Add returned unexpected error: %v", err)
	}
	if err := workqueue.Add(db, workqueue.Item{ID: "blocker-item", Tier: workqueue.Blocker, CreatedAt: now}); err != nil {
		t.Fatalf("Add returned unexpected error: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := contextResume(db, []string{"checkout"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("contextResume returned exit code %d, stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "next work: blocker-item") {
		t.Errorf("stdout = %q, want it to report blocker-item as the next work", stdout.String())
	}

	items, err := workqueue.List(db)
	if err != nil {
		t.Fatalf("List returned unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("after contextResume, queue has %d items, want 2 (resume must not pop)", len(items))
	}
}

func TestContextResume_UnknownContext(t *testing.T) {
	db := openTestContextStore(t)
	var stdout, stderr bytes.Buffer
	code := contextResume(db, []string{"nonexistent"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("contextResume on an unknown context: exit code = %d, want 1", code)
	}
}

func TestContextResume_WrongArgCount(t *testing.T) {
	db := openTestContextStore(t)
	var stdout, stderr bytes.Buffer
	code := contextResume(db, nil, &stdout, &stderr)
	if code != 1 {
		t.Errorf("contextResume with no arguments: exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Errorf("stderr should contain a usage message, got %q", stderr.String())
	}
}

func TestContextOpen_ReturnsTheExactVersionAsked(t *testing.T) {
	db := openTestContextStore(t)
	if _, err := contextcomposition.CreateContext(db, "checkout", ""); err != nil {
		t.Fatalf("CreateContext returned unexpected error: %v", err)
	}
	if _, err := contextcomposition.AddVersion(db, "checkout", "v1 state"); err != nil {
		t.Fatalf("AddVersion returned unexpected error: %v", err)
	}
	if _, err := contextcomposition.AddVersion(db, "checkout", "v2 state"); err != nil {
		t.Fatalf("AddVersion returned unexpected error: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := contextOpen(db, []string{"checkout@v1"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("contextOpen returned exit code %d, stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "v1 state") {
		t.Errorf("stdout = %q, want it to contain v1's content", stdout.String())
	}
	if strings.Contains(stdout.String(), "v2 state") {
		t.Errorf("stdout = %q, should not contain v2's content", stdout.String())
	}
}

func TestContextOpen_MalformedRef(t *testing.T) {
	db := openTestContextStore(t)
	var stdout, stderr bytes.Buffer
	code := contextOpen(db, []string{"checkout@latest"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("contextOpen with a malformed ref: exit code = %d, want 1", code)
	}
}

func TestContextOpen_NoVersionGiven(t *testing.T) {
	db := openTestContextStore(t)
	if _, err := contextcomposition.CreateContext(db, "checkout", ""); err != nil {
		t.Fatalf("CreateContext returned unexpected error: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := contextOpen(db, []string{"checkout"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("contextOpen with no explicit version: exit code = %d, want 1", code)
	}
}

func TestContextRestore_ArchivedContextBecomesActive(t *testing.T) {
	db := openTestContextStore(t)
	if _, err := contextcomposition.CreateContext(db, "checkout", ""); err != nil {
		t.Fatalf("CreateContext returned unexpected error: %v", err)
	}
	if _, err := contextcomposition.ArchiveContext(db, "checkout", "new-checkout"); err != nil {
		t.Fatalf("ArchiveContext returned unexpected error: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := contextRestore(db, []string{"checkout"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("contextRestore returned exit code %d, stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "restored checkout") {
		t.Errorf("stdout = %q, want confirmation it restored \"checkout\"", stdout.String())
	}

	reloaded, err := contextcomposition.GetContext(db, "checkout")
	if err != nil {
		t.Fatalf("GetContext returned unexpected error: %v", err)
	}
	if reloaded.State != contextcomposition.Active {
		t.Errorf("after contextRestore, GetContext = %+v, want State Active", reloaded)
	}
}

func TestContextRestore_NotArchivedIsError(t *testing.T) {
	db := openTestContextStore(t)
	if _, err := contextcomposition.CreateContext(db, "checkout", ""); err != nil {
		t.Fatalf("CreateContext returned unexpected error: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := contextRestore(db, []string{"checkout"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("contextRestore on an Active context: exit code = %d, want 1", code)
	}
}

func TestContextResolveConflict_ReferenceOutranksEmbedding(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runContext([]string{"resolve-conflict", "--mode-a", "ByReference", "--mode-b", "ByEmbedding"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("resolve-conflict returned exit code %d, want 0; stderr:\n%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "ByReference outranks ByEmbedding") {
		t.Errorf("stdout = %q, want it to report ByReference outranking ByEmbedding", stdout.String())
	}
}

func TestContextResolveConflict_OrderOfFlagsDoesNotMatter(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runContext([]string{"resolve-conflict", "--mode-b", "ByView", "--mode-a", "ByInheritance"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("resolve-conflict returned exit code %d, want 0; stderr:\n%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "ByInheritance outranks ByView") {
		t.Errorf("stdout = %q, want it to report ByInheritance outranking ByView", stdout.String())
	}
}

func TestContextResolveConflict_SameModeIsUndecided(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runContext([]string{"resolve-conflict", "--mode-a", "ByReference", "--mode-b", "ByReference"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("resolve-conflict returned exit code %d, want 0; stderr:\n%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "undecided") {
		t.Errorf("stdout = %q, want it to report the outcome as undecided for two sources of the same mode", stdout.String())
	}
}

func TestContextResolveConflict_UnrecognizedModeIsError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runContext([]string{"resolve-conflict", "--mode-a", "ByReference", "--mode-b", "Bogus"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("resolve-conflict with an unrecognized mode: exit code = %d, want 1", code)
	}
}

func TestContextResolveConflict_MissingFlagIsUsageError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runContext([]string{"resolve-conflict", "--mode-a", "ByReference"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("resolve-conflict with only --mode-a: exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Errorf("stderr should contain a usage message, got %q", stderr.String())
	}
}

func TestSplitPhenomenonID(t *testing.T) {
	tests := []struct {
		id, wantArea, wantPhenomenon string
		wantOK                       bool
	}{
		{"shipping::checkout", "shipping", "checkout", true},
		{"shipping::checkout::extra", "", "", false},
		{"no-separator", "", "", false},
		{"::checkout", "", "", false},
		{"shipping::", "", "", false},
	}
	for _, tt := range tests {
		area, phenomenon, ok := splitPhenomenonID(tt.id)
		if ok != tt.wantOK || (ok && (area != tt.wantArea || phenomenon != tt.wantPhenomenon)) {
			t.Errorf("splitPhenomenonID(%q) = (%q, %q, %v), want (%q, %q, %v)",
				tt.id, area, phenomenon, ok, tt.wantArea, tt.wantPhenomenon, tt.wantOK)
		}
	}
}

func TestContextOrigin_ReportsSignedBlockAndDecisions(t *testing.T) {
	repoRoot := t.TempDir()
	dir := filepath.Join(repoRoot, "phenomena", "shipping")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned unexpected error: %v", err)
	}
	path := filepath.Join(dir, "checkout.cjtr")
	if err := os.WriteFile(path, []byte(checkFixture), 0o644); err != nil {
		t.Fatalf("WriteFile returned unexpected error: %v", err)
	}
	if err := resignInPlace(path, "alice", true); err != nil {
		t.Fatalf("resignInPlace returned unexpected error: %v", err)
	}

	db := openTestContextStore(t)
	if _, err := contextcomposition.CreateContext(db, "shipping::checkout", "shipping"); err != nil {
		t.Fatalf("CreateContext returned unexpected error: %v", err)
	}
	if err := decisionledger.EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema returned unexpected error: %v", err)
	}
	if _, err := decisionledger.Append(db, decisionledger.Decision{
		Kind: "candidate-review", Subject: path, Determination: "accepted", DecidedBy: "alice",
	}); err != nil {
		t.Fatalf("Append returned unexpected error: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := contextOrigin(db, repoRoot, []string{"shipping::checkout"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("contextOrigin returned exit code %d, stderr: %s", code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "signed by alice") || !strings.Contains(out, "confirmed true") {
		t.Errorf("stdout = %q, want it to report the real signed block", out)
	}
	if !strings.Contains(out, "candidate-review") || !strings.Contains(out, "accepted") {
		t.Errorf("stdout = %q, want it to report the recorded decision", out)
	}
}

func TestContextOrigin_UnsignedFileAndNoDecisionsReportsHonestly(t *testing.T) {
	repoRoot := t.TempDir()
	dir := filepath.Join(repoRoot, "phenomena", "shipping")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned unexpected error: %v", err)
	}
	path := filepath.Join(dir, "checkout.cjtr")
	if err := os.WriteFile(path, []byte(checkFixture), 0o644); err != nil {
		t.Fatalf("WriteFile returned unexpected error: %v", err)
	}

	db := openTestContextStore(t)
	if _, err := contextcomposition.CreateContext(db, "shipping::checkout", "shipping"); err != nil {
		t.Fatalf("CreateContext returned unexpected error: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := contextOrigin(db, repoRoot, []string{"shipping::checkout"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("contextOrigin returned exit code %d, stderr: %s", code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "not signed") {
		t.Errorf("stdout = %q, want it to report the file is not signed", out)
	}
	if !strings.Contains(out, "no decisions recorded") {
		t.Errorf("stdout = %q, want it to report no decisions were found", out)
	}
}

func TestContextOrigin_UnknownContextIsError(t *testing.T) {
	db := openTestContextStore(t)
	var stdout, stderr bytes.Buffer
	code := contextOrigin(db, t.TempDir(), []string{"nope::nope"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("contextOrigin with an unknown context: exit code = %d, want 1", code)
	}
}

func TestContextOrigin_WrongArgCountIsUsageError(t *testing.T) {
	db := openTestContextStore(t)
	var stdout, stderr bytes.Buffer
	code := contextOrigin(db, t.TempDir(), nil, &stdout, &stderr)
	if code != 1 {
		t.Errorf("contextOrigin with no name: exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Errorf("stderr should contain a usage message, got %q", stderr.String())
	}
}

func TestContextDiff_ShowsUnifiedDiffBetweenTwoVersions(t *testing.T) {
	db := openTestContextStore(t)
	if _, err := contextcomposition.CreateContext(db, "checkout", ""); err != nil {
		t.Fatalf("CreateContext returned unexpected error: %v", err)
	}
	if _, err := contextcomposition.AddVersion(db, "checkout", "line one\nline two\n"); err != nil {
		t.Fatalf("AddVersion returned unexpected error: %v", err)
	}
	if _, err := contextcomposition.AddVersion(db, "checkout", "line one\nline TWO changed\n"); err != nil {
		t.Fatalf("AddVersion returned unexpected error: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := contextDiff(db, []string{"checkout@v1", "checkout@v2"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("contextDiff returned exit code %d, stderr: %s", code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "-line two") || !strings.Contains(out, "+line TWO changed") {
		t.Errorf("stdout = %q, want a unified diff showing the changed line", out)
	}
}

func TestContextDiff_IdenticalContentReportsNoDiff(t *testing.T) {
	db := openTestContextStore(t)
	if _, err := contextcomposition.CreateContext(db, "checkout", ""); err != nil {
		t.Fatalf("CreateContext returned unexpected error: %v", err)
	}
	if _, err := contextcomposition.AddVersion(db, "checkout", "same content\n"); err != nil {
		t.Fatalf("AddVersion returned unexpected error: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := contextDiff(db, []string{"checkout", "checkout@v1"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("contextDiff returned exit code %d, stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "identical") {
		t.Errorf("stdout = %q, want it to report the two versions as identical", stdout.String())
	}
}

func TestContextDiff_WrongArgCountIsUsageError(t *testing.T) {
	db := openTestContextStore(t)
	var stdout, stderr bytes.Buffer
	code := contextDiff(db, []string{"checkout"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("contextDiff with one argument: exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Errorf("stderr should contain a usage message, got %q", stderr.String())
	}
}

func TestContextDiff_UnknownContextIsError(t *testing.T) {
	db := openTestContextStore(t)
	var stdout, stderr bytes.Buffer
	code := contextDiff(db, []string{"nope", "alsonope"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("contextDiff with unknown contexts: exit code = %d, want 1", code)
	}
}
