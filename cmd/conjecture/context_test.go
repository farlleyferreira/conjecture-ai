package main

import (
	"bytes"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/farlleyferreira/conjecture-ai/internal/contextcomposition"
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
