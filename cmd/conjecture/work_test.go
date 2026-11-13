package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/farlleyferreira/conjecture-ai/internal/sandbox"
	"github.com/farlleyferreira/conjecture-ai/internal/workqueue"
)

func TestResolveQueueItem_FourSegments(t *testing.T) {
	path, ruleID, hypothesisID, err := resolveQueueItem("demo-area::demo::result-rule::result-matches", "/root")
	if err != nil {
		t.Fatalf("resolveQueueItem returned unexpected error: %v", err)
	}
	wantPath := filepath.Join("/root", "conjecture", "phenomena", "demo-area", "demo.cjtr")
	if path != wantPath {
		t.Errorf("path = %q, want %q", path, wantPath)
	}
	if ruleID != "result-rule" || hypothesisID != "result-matches" {
		t.Errorf("ruleID, hypothesisID = %q, %q, want %q, %q", ruleID, hypothesisID, "result-rule", "result-matches")
	}
}

func TestResolveQueueItem_ThreeSegmentsIsFileLevelLaw(t *testing.T) {
	path, ruleID, hypothesisID, err := resolveQueueItem("demo-area::demo::some-law", "/root")
	if err != nil {
		t.Fatalf("resolveQueueItem returned unexpected error: %v", err)
	}
	wantPath := filepath.Join("/root", "conjecture", "phenomena", "demo-area", "demo.cjtr")
	if path != wantPath {
		t.Errorf("path = %q, want %q", path, wantPath)
	}
	if ruleID != "" {
		t.Errorf("ruleID = %q, want empty (file-level law)", ruleID)
	}
	if hypothesisID != "some-law" {
		t.Errorf("hypothesisID = %q, want %q", hypothesisID, "some-law")
	}
}

func TestResolveQueueItem_MalformedID(t *testing.T) {
	if _, _, _, err := resolveQueueItem("not-a-valid-id", "/root"); err == nil {
		t.Error(`resolveQueueItem("not-a-valid-id") = nil error, want one`)
	}
}

// implementFixtureRoot returns the root directory implementFixture
// (written by writeImplementFixture) is rooted under - four levels
// above the .cjtr file itself (conjecture/phenomena/demo-area/
// demo.cjtr - joinPhenomenonPath's own real layout) - so a test can
// pass it to workLoop as root without workLoop or writeImplementFixture
// needing to agree on it any other way.
func implementFixtureRoot(path string) string {
	return filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(path))))
}

func TestWorkLoop_EmptyQueueExitsCleanly(t *testing.T) {
	db := openCheckTestStore(t)
	var stdout bytes.Buffer
	code := workLoop(context.Background(), db, &sandbox.Sandbox{DockerBinary: fakeImplementPath}, &fakeImplementerClient{}, "/unused", 3, &stdout)
	if code != 0 {
		t.Errorf("workLoop on an empty queue = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "empty") {
		t.Errorf("stdout = %q, want it to report the queue is empty", stdout.String())
	}
}

func TestWorkLoop_PopsAndImplementsUntilConformant(t *testing.T) {
	t.Setenv("FAKEIMPLEMENT_SUCCEED_AFTER", "2")
	path := writeImplementFixture(t)
	root := implementFixtureRoot(path)

	db := openCheckTestStore(t)
	now := time.Now().UTC()
	if err := workqueue.Add(db, workqueue.Item{ID: "demo-area::demo::result-rule::result-matches", Tier: workqueue.Blocker, CreatedAt: now}); err != nil {
		t.Fatalf("Add returned unexpected error: %v", err)
	}

	var stdout bytes.Buffer
	code := workLoop(context.Background(), db, &sandbox.Sandbox{DockerBinary: fakeImplementPath}, &fakeImplementerClient{}, root, 3, &stdout)
	if code != 0 {
		t.Errorf("workLoop = %d, want 0 (the item should reach Conformant); stdout:\n%s", code, stdout.String())
	}
	if !strings.Contains(stdout.String(), "result-matches: conformant") {
		t.Errorf("stdout = %q, want it to report the item's own final verdict as conformant", stdout.String())
	}

	items, err := workqueue.List(db)
	if err != nil {
		t.Fatalf("List returned unexpected error: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("queue after workLoop = %+v, want empty (the item conformed, so it was never re-enqueued)", items)
	}
}

func TestWorkLoop_ExhaustsAttemptsAndLeavesItemForLaterRun(t *testing.T) {
	t.Setenv("FAKEIMPLEMENT_SUCCEED_AFTER", "0") // never succeeds
	path := writeImplementFixture(t)
	root := implementFixtureRoot(path)

	db := openCheckTestStore(t)
	itemID := "demo-area::demo::result-rule::result-matches"
	now := time.Now().UTC()
	if err := workqueue.Add(db, workqueue.Item{ID: itemID, Tier: workqueue.Blocker, CreatedAt: now}); err != nil {
		t.Fatalf("Add returned unexpected error: %v", err)
	}

	var stdout bytes.Buffer
	code := workLoop(context.Background(), db, &sandbox.Sandbox{DockerBinary: fakeImplementPath}, &fakeImplementerClient{}, root, 1, &stdout)
	if code != 1 {
		t.Errorf("workLoop = %d, want 1 (the item never conformed); stdout:\n%s", code, stdout.String())
	}
	if !strings.Contains(stdout.String(), "exhausted") {
		t.Errorf("stdout = %q, want it to report the attempt budget was exhausted", stdout.String())
	}

	items, err := workqueue.List(db)
	if err != nil {
		t.Fatalf("List returned unexpected error: %v", err)
	}
	if len(items) != 1 || items[0].ID != itemID {
		t.Errorf("queue after workLoop = %+v, want the item restored for a later run", items)
	}
}

func TestWorkLoop_RefusedBlockerItemMovesToDeadLetterAndStopsRun(t *testing.T) {
	repoRoot := t.TempDir()
	complexDir := filepath.Join(repoRoot, "conjecture", "phenomena", "demo-area")
	if err := os.MkdirAll(complexDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned unexpected error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(complexDir, "demo.cjtr"), []byte(implementFixtureUnboundedDomain), 0o644); err != nil {
		t.Fatalf("WriteFile returned unexpected error: %v", err)
	}
	simpleDir := filepath.Join(repoRoot, "conjecture", "phenomena", "other-area")
	if err := os.MkdirAll(simpleDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned unexpected error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(simpleDir, "demo.cjtr"), []byte(implementFixture), 0o644); err != nil {
		t.Fatalf("WriteFile returned unexpected error: %v", err)
	}

	db := openCheckTestStore(t)
	complexID := "demo-area::demo::result-rule::result-matches"
	simpleID := "other-area::demo::result-rule::result-matches"
	now := time.Now().UTC()
	if err := workqueue.Add(db, workqueue.Item{ID: complexID, Tier: workqueue.Blocker, CreatedAt: now}); err != nil {
		t.Fatalf("Add returned unexpected error: %v", err)
	}
	if err := workqueue.Add(db, workqueue.Item{ID: simpleID, Tier: workqueue.Background, CreatedAt: now}); err != nil {
		t.Fatalf("Add returned unexpected error: %v", err)
	}

	var stdout bytes.Buffer
	code := workLoop(context.Background(), db, &sandbox.Sandbox{DockerBinary: fakeImplementPath}, &fakeImplementerClient{}, repoRoot, 3, &stdout)
	if code != 1 {
		t.Errorf("workLoop = %d, want 1 (a blocker item was refused); stdout:\n%s", code, stdout.String())
	}
	if !strings.Contains(stdout.String(), "blocker item moved to the dead letter") {
		t.Errorf("stdout = %q, want it to report the blocker item moved to the dead letter and the run stopping", stdout.String())
	}
	if strings.Contains(stdout.String(), simpleID) {
		t.Errorf("stdout = %q, want the lower-priority item never reached (the blocker halted the run)", stdout.String())
	}

	deadLetter, err := workqueue.ListDeadLetter(db)
	if err != nil {
		t.Fatalf("ListDeadLetter returned unexpected error: %v", err)
	}
	if len(deadLetter) != 1 || deadLetter[0].ID != complexID || deadLetter[0].Tier != workqueue.Blocker {
		t.Errorf("deadLetter = %+v, want exactly the blocker item", deadLetter)
	}

	remaining, err := workqueue.List(db)
	if err != nil {
		t.Fatalf("List returned unexpected error: %v", err)
	}
	if len(remaining) != 1 || remaining[0].ID != simpleID {
		t.Errorf("queue after workLoop = %+v, want the untouched lower-priority item still queued", remaining)
	}
}

func TestWorkLoop_RefusedNonBlockerItemMovesToDeadLetterAndRunContinues(t *testing.T) {
	t.Setenv("FAKEIMPLEMENT_SUCCEED_AFTER", "2")
	repoRoot := t.TempDir()
	complexDir := filepath.Join(repoRoot, "conjecture", "phenomena", "demo-area")
	if err := os.MkdirAll(complexDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned unexpected error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(complexDir, "demo.cjtr"), []byte(implementFixtureUnboundedDomain), 0o644); err != nil {
		t.Fatalf("WriteFile returned unexpected error: %v", err)
	}
	simpleDir := filepath.Join(repoRoot, "conjecture", "phenomena", "other-area")
	if err := os.MkdirAll(simpleDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned unexpected error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(simpleDir, "demo.cjtr"), []byte(implementFixture), 0o644); err != nil {
		t.Fatalf("WriteFile returned unexpected error: %v", err)
	}

	db := openCheckTestStore(t)
	complexID := "demo-area::demo::result-rule::result-matches"
	simpleID := "other-area::demo::result-rule::result-matches"
	now := time.Now().UTC()
	if err := workqueue.Add(db, workqueue.Item{ID: complexID, Tier: workqueue.Background, CreatedAt: now}); err != nil {
		t.Fatalf("Add returned unexpected error: %v", err)
	}
	if err := workqueue.Add(db, workqueue.Item{ID: simpleID, Tier: workqueue.CurrentTask, CreatedAt: now}); err != nil {
		t.Fatalf("Add returned unexpected error: %v", err)
	}

	var stdout bytes.Buffer
	code := workLoop(context.Background(), db, &sandbox.Sandbox{DockerBinary: fakeImplementPath}, &fakeImplementerClient{}, repoRoot, 3, &stdout)
	if code != 1 {
		t.Errorf("workLoop = %d, want 1 (a non-blocker item was refused); stdout:\n%s", code, stdout.String())
	}
	if !strings.Contains(stdout.String(), complexID+": moved to the dead letter") {
		t.Errorf("stdout = %q, want the non-blocker item reported as moved to the dead letter", stdout.String())
	}
	if !strings.Contains(stdout.String(), simpleID+": conformant") {
		t.Errorf("stdout = %q, want the run to have continued and conformed the other item", stdout.String())
	}

	deadLetter, err := workqueue.ListDeadLetter(db)
	if err != nil {
		t.Fatalf("ListDeadLetter returned unexpected error: %v", err)
	}
	if len(deadLetter) != 1 || deadLetter[0].ID != complexID || deadLetter[0].Tier != workqueue.Background {
		t.Errorf("deadLetter = %+v, want exactly the non-blocker item", deadLetter)
	}
}

func TestWorkLoop_MalformedItemIsReportedAndDiscarded(t *testing.T) {
	db := openCheckTestStore(t)
	now := time.Now().UTC()
	if err := workqueue.Add(db, workqueue.Item{ID: "not-a-valid-id", Tier: workqueue.Blocker, CreatedAt: now}); err != nil {
		t.Fatalf("Add returned unexpected error: %v", err)
	}

	var stdout bytes.Buffer
	code := workLoop(context.Background(), db, &sandbox.Sandbox{DockerBinary: fakeImplementPath}, &fakeImplementerClient{}, "/unused", 3, &stdout)
	if code != 1 {
		t.Errorf("workLoop = %d, want 1 (an unresolvable item)", code)
	}
	if !strings.Contains(stdout.String(), "ERROR") {
		t.Errorf("stdout = %q, want it to report the error", stdout.String())
	}

	items, err := workqueue.List(db)
	if err != nil {
		t.Fatalf("List returned unexpected error: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("queue after workLoop = %+v, want empty (a malformed id can never be resolved, so it is discarded, not restored)", items)
	}
}

// implementFixtureWithFileLevelLaw declares "user_defined_laws" at
// the file's top level (sibling to "rules") with its own declared
// "id" - resolveQueueItem's own 3-segment form
// (area::phenomenon::hypothesis) is only ever reachable for a law
// declared this way, not one nested under a rule.
const implementFixtureWithFileLevelLaw = `
conjecture: 1.0.0

phenomenon: demo implement file law

concepts:
  flag:
    variable: input.flag
    one_of: [x]
  result:
    variable: output.result
    type: decimal

user_defined_laws:
  result law:
    id: result-law
    always: the result is 7.50
    examples:
      - {flag: x, result: 7.50}

rules:
  placeholder rule:
    id: placeholder-rule
    hypotheses:
      placeholder:
        id: placeholder-hyp
        always: the result is at least 0.00
`

func writeImplementFixtureWithFileLevelLaw(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "conjecture", "phenomena", "demo-area")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	path := filepath.Join(dir, "demo.cjtr")
	if err := os.WriteFile(path, []byte(implementFixtureWithFileLevelLaw), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	return path
}

func TestFindHypothesis_FileLevelLawReportsIsLaw(t *testing.T) {
	path := writeImplementFixtureWithFileLevelLaw(t)
	f, err := loadFicha(path)
	if err != nil {
		t.Fatalf("loadFicha returned unexpected error: %v", err)
	}

	rule, h, isLaw, err := findHypothesis(f, "", "result-law")
	if err != nil {
		t.Fatalf("findHypothesis returned unexpected error: %v", err)
	}
	if h.ID != "result-law" {
		t.Errorf("findHypothesis returned hypothesis %+v", h)
	}
	if rule.CompositeID != f.phenomenonID {
		t.Errorf("findHypothesis returned rule.CompositeID = %q, want %q (the phenomenon's own id)", rule.CompositeID, f.phenomenonID)
	}
	if !isLaw {
		t.Error("findHypothesis reported isLaw = false for a file-level law, want true")
	}
}

func TestFindHypothesis_EmptyRuleIDUnknownLawID(t *testing.T) {
	path := writeImplementFixtureWithFileLevelLaw(t)
	f, err := loadFicha(path)
	if err != nil {
		t.Fatalf("loadFicha returned unexpected error: %v", err)
	}

	if _, _, _, err := findHypothesis(f, "", "does-not-exist"); err == nil {
		t.Fatal("findHypothesis with an unknown file-level law id = nil error, want one")
	}
}

func TestWorkLoop_FileLevelLawResolvedAndImplemented(t *testing.T) {
	t.Setenv("FAKEIMPLEMENT_SUCCEED_AFTER", "2")
	path := writeImplementFixtureWithFileLevelLaw(t)
	root := implementFixtureRoot(path)

	db := openCheckTestStore(t)
	now := time.Now().UTC()
	if err := workqueue.Add(db, workqueue.Item{ID: "demo-area::demo::result-law", Tier: workqueue.Blocker, CreatedAt: now}); err != nil {
		t.Fatalf("Add returned unexpected error: %v", err)
	}

	var stdout bytes.Buffer
	code := workLoop(context.Background(), db, &sandbox.Sandbox{DockerBinary: fakeImplementPath}, &fakeImplementerClient{}, root, 3, &stdout)
	if code != 0 {
		t.Errorf("workLoop = %d, want 0; stdout:\n%s", code, stdout.String())
	}
	if !strings.Contains(stdout.String(), "result-law: conformant") {
		t.Errorf("stdout = %q, want the file-level law's own final verdict to be conformant", stdout.String())
	}
}

func TestRunWork_TooManyUnknownFlags(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runWork([]string{"--bogus-flag"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runWork with an unrecognized flag = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), workUsage) {
		t.Errorf("stderr = %q, want it to contain the usage string", stderr.String())
	}
}
