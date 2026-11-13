package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/farlleyferreira/conjecture-ai/internal/persistence"
	"github.com/farlleyferreira/conjecture-ai/internal/workqueue"
)

func openTestQueueStore(t *testing.T) *sql.DB {
	t.Helper()
	db, err := persistence.OpenStoreAt(filepath.Join(t.TempDir(), "data"))
	if err != nil {
		t.Fatalf("OpenStoreAt returned unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := workqueue.EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema returned unexpected error: %v", err)
	}
	return db
}

func TestRunQueue_NoSubcommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runQueue(nil, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runQueue with no subcommand = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Errorf("stderr = %q, want a usage message", stderr.String())
	}
}

func TestRunQueue_UnknownSubcommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runQueue([]string{"remove", "x"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runQueue with an unknown subcommand = %d, want 1", code)
	}
}

func TestRunQueueAdd_UnrecognizedTier(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runQueueAdd([]string{"x", "urgent"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runQueueAdd with an unrecognized tier = %d, want 1", code)
	}
}

func TestRunQueueAdd_WrongArgCount(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runQueueAdd([]string{"only-one-arg"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runQueueAdd with one positional arg = %d, want 1", code)
	}
}

func TestQueueItemPhenomenonID(t *testing.T) {
	cases := []struct {
		id     string
		want   string
		wantOK bool
	}{
		{"demo-area::demo::rule::hyp", "demo-area::demo", true},
		{"demo-area::demo::file-law", "demo-area::demo", true},
		{"not-enough-segments", "", false},
		{"too::many::segments::here::for-this", "", false},
	}
	for _, c := range cases {
		got, ok := queueItemPhenomenonID(c.id)
		if ok != c.wantOK || (ok && got != c.want) {
			t.Errorf("queueItemPhenomenonID(%q) = %q, %v, want %q, %v", c.id, got, ok, c.want, c.wantOK)
		}
	}
}

func TestQueueSummaries_GroupsByPhenomenonSortedAndCountsByTier(t *testing.T) {
	now := time.Now().UTC()
	items := []workqueue.Item{
		{ID: "b-area::demo::rule::hyp1", Tier: workqueue.Blocker, CreatedAt: now},
		{ID: "b-area::demo::rule::hyp2", Tier: workqueue.Background, CreatedAt: now},
		{ID: "a-area::demo::rule::hyp3", Tier: workqueue.CurrentTask, CreatedAt: now},
	}
	summaries := queueSummaries(items)
	if len(summaries) != 2 {
		t.Fatalf("len(summaries) = %d, want 2", len(summaries))
	}
	if summaries[0].PhenomenonID != "a-area::demo" {
		t.Errorf("summaries[0].PhenomenonID = %q, want %q (sorted first)", summaries[0].PhenomenonID, "a-area::demo")
	}
	second := summaries[1]
	if second.PhenomenonID != "b-area::demo" || second.Total != 2 || second.Blocker != 1 || second.Background != 1 {
		t.Errorf("summaries[1] = %+v, want b-area::demo with total 2 (1 blocker, 1 background)", second)
	}
}

func TestQueueSummaries_MalformedIDGroupedUnderItself(t *testing.T) {
	items := []workqueue.Item{{ID: "not-a-valid-id", Tier: workqueue.Blocker, CreatedAt: time.Now().UTC()}}
	summaries := queueSummaries(items)
	if len(summaries) != 1 || summaries[0].PhenomenonID != "not-a-valid-id" {
		t.Errorf("summaries = %+v, want the malformed id disclosed as its own group, not dropped", summaries)
	}
}

func TestComputeQueueList_PhenomenonWithoutAreaIsError(t *testing.T) {
	if _, _, _, err := computeQueueList(nil, "", "checkout"); err == nil {
		t.Error("computeQueueList with phenomenon and no area = nil error, want one")
	}
}

func TestComputeQueueList_AreaAndPhenomenonDrillsToItems(t *testing.T) {
	now := time.Now().UTC()
	items := []workqueue.Item{
		{ID: "demo-area::checkout::rule::hyp", Tier: workqueue.Blocker, Weight: 1, Severity: 1, CreatedAt: now},
		{ID: "demo-area::billing::rule::hyp", Tier: workqueue.Blocker, Weight: 1, Severity: 1, CreatedAt: now},
	}
	summaries, queueItems, warning, err := computeQueueList(items, "demo-area", "checkout")
	if err != nil {
		t.Fatalf("computeQueueList returned unexpected error: %v", err)
	}
	if summaries != nil {
		t.Errorf("summaries = %+v, want nil when drilling into one phenomenon", summaries)
	}
	if len(queueItems) != 1 || queueItems[0].ID != "demo-area::checkout::rule::hyp" {
		t.Errorf("queueItems = %+v, want exactly demo-area::checkout's own item", queueItems)
	}
	if warning != "" {
		t.Errorf("warning = %q, want empty (1 item vs. 1 other phenomenon's own 1 item is not 3x)", warning)
	}
}

func TestComputeQueueList_WarnsWhenFarAboveTheProjectsOwnAverage(t *testing.T) {
	now := time.Now().UTC()
	var items []workqueue.Item
	// Two "other" phenomena averaging 1 item each; "checkout" has 4 -
	// more than 3x that average (3).
	items = append(items, workqueue.Item{ID: "demo-area::billing::rule::hyp", Tier: workqueue.Blocker, CreatedAt: now})
	items = append(items, workqueue.Item{ID: "demo-area::invoicing::rule::hyp", Tier: workqueue.Blocker, CreatedAt: now})
	for i := 0; i < 4; i++ {
		items = append(items, workqueue.Item{ID: fmt.Sprintf("demo-area::checkout::rule::hyp%d", i), Tier: workqueue.Blocker, CreatedAt: now})
	}

	_, queueItems, warning, err := computeQueueList(items, "demo-area", "checkout")
	if err != nil {
		t.Fatalf("computeQueueList returned unexpected error: %v", err)
	}
	if len(queueItems) != 4 {
		t.Fatalf("len(queueItems) = %d, want 4", len(queueItems))
	}
	if warning == "" {
		t.Error("warning is empty, want one (4 items vs. the other phenomena's own average of 1 is more than 3x)")
	}
	if !strings.Contains(warning, "demo-area::checkout") || !strings.Contains(warning, "4") {
		t.Errorf("warning = %q, want it to name demo-area::checkout and its own count (4)", warning)
	}
}

func TestComputeQueueList_NoOtherPhenomenonMeansNoWarning(t *testing.T) {
	now := time.Now().UTC()
	var items []workqueue.Item
	for i := 0; i < 50; i++ {
		items = append(items, workqueue.Item{ID: fmt.Sprintf("demo-area::checkout::rule::hyp%d", i), Tier: workqueue.Blocker, CreatedAt: now})
	}

	_, _, warning, err := computeQueueList(items, "demo-area", "checkout")
	if err != nil {
		t.Fatalf("computeQueueList returned unexpected error: %v", err)
	}
	if warning != "" {
		t.Errorf("warning = %q, want empty - there is no other phenomenon to compare against, even with 50 items", warning)
	}
}

func TestRunQueueList_ReportsGroupedSummary(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	db, err := persistence.OpenStore(".")
	if err != nil {
		t.Fatalf("OpenStore returned unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := workqueue.EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema returned unexpected error: %v", err)
	}
	if err := workqueue.Add(db, workqueue.Item{ID: "demo-area::checkout::rule::hyp", Tier: workqueue.Blocker, Weight: 1, CreatedAt: time.Now().UTC()}); err != nil {
		t.Fatalf("workqueue.Add returned unexpected error: %v", err)
	}
	_ = db.Close()

	var stdout, stderr bytes.Buffer
	code := runQueueList(nil, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("runQueueList exit code = %d, stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "demo-area::checkout: 1 items (blocker 1, current-task 0, background 0)") {
		t.Errorf("stdout = %q, want the grouped summary line", stdout.String())
	}
}

func TestRunQueueList_PhenomenonWithoutAreaIsUsageError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runQueueList([]string{"--phenomenon", "checkout"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runQueueList with --phenomenon and no --area = %d, want 1", code)
	}
}

func TestRunQueueReview_EmptyReportsEmpty(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	var stdout, stderr bytes.Buffer
	code := runQueueReview(nil, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("runQueueReview exit code = %d, stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "dead letter is empty") {
		t.Errorf("stdout = %q, want it to report the dead letter is empty", stdout.String())
	}
}

func TestRunQueueReview_ReportsDeadLetteredItems(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	db, err := persistence.OpenStore(".")
	if err != nil {
		t.Fatalf("OpenStore returned unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := workqueue.EnsureDeadLetterSchema(db); err != nil {
		t.Fatalf("EnsureDeadLetterSchema returned unexpected error: %v", err)
	}
	if err := workqueue.AddDeadLetter(db, workqueue.DeadLetterItem{
		ID: "demo-area::demo::rule::hyp", Tier: workqueue.Blocker, Weight: 1,
		Reason: "domain not boundable", CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("AddDeadLetter returned unexpected error: %v", err)
	}
	_ = db.Close()

	var stdout, stderr bytes.Buffer
	code := runQueueReview(nil, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("runQueueReview exit code = %d, stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "demo-area::demo::rule::hyp") || !strings.Contains(stdout.String(), "domain not boundable") {
		t.Errorf("stdout = %q, want the dead-lettered item and its reason", stdout.String())
	}
}

func TestRunQueueReview_WrongArgCountIsUsageError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runQueueReview([]string{"extra"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runQueueReview with an arg = %d, want 1", code)
	}
}

func TestRunQueueReview_ResolveRemovesItemFromDefaultListingButAllStillShowsIt(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	db, err := persistence.OpenStore(".")
	if err != nil {
		t.Fatalf("OpenStore returned unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := workqueue.EnsureDeadLetterSchema(db); err != nil {
		t.Fatalf("EnsureDeadLetterSchema returned unexpected error: %v", err)
	}
	if err := workqueue.AddDeadLetter(db, workqueue.DeadLetterItem{
		ID: "demo-area::demo::rule::hyp", Tier: workqueue.Blocker, Weight: 1,
		Reason: "domain not boundable", CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("AddDeadLetter returned unexpected error: %v", err)
	}
	items, err := workqueue.ListDeadLetter(db, false)
	if err != nil {
		t.Fatalf("ListDeadLetter returned unexpected error: %v", err)
	}
	rowID := items[0].RowID
	_ = db.Close()

	var resolveOut, resolveErr bytes.Buffer
	code := runQueueReview([]string{"--resolve", strconv.FormatInt(rowID, 10)}, &resolveOut, &resolveErr)
	if code != 0 {
		t.Fatalf("runQueueReview --resolve exit code = %d, stderr: %s", code, resolveErr.String())
	}

	var defaultOut, defaultErr bytes.Buffer
	if code := runQueueReview(nil, &defaultOut, &defaultErr); code != 0 {
		t.Fatalf("runQueueReview exit code = %d, stderr: %s", code, defaultErr.String())
	}
	if !strings.Contains(defaultOut.String(), "dead letter is empty") {
		t.Errorf("default listing after resolving the only row = %q, want it to report empty", defaultOut.String())
	}

	var allOut, allErr bytes.Buffer
	if code := runQueueReview([]string{"--all"}, &allOut, &allErr); code != 0 {
		t.Fatalf("runQueueReview --all exit code = %d, stderr: %s", code, allErr.String())
	}
	if !strings.Contains(allOut.String(), "demo-area::demo::rule::hyp") || !strings.Contains(allOut.String(), "resolved") {
		t.Errorf("--all listing = %q, want the resolved item reported as resolved", allOut.String())
	}
}

func TestRunQueueReview_ResolveUnknownRowIDErrors(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	var stdout, stderr bytes.Buffer
	code := runQueueReview([]string{"--resolve", "999"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runQueueReview --resolve on an unknown row id = %d, want 1", code)
	}
}

func TestRunQueueReview_ResolveMissingValueIsUsageError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runQueueReview([]string{"--resolve"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runQueueReview --resolve with no value = %d, want 1", code)
	}
}

func TestRunQueueReview_ResolveNonIntegerValueIsUsageError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runQueueReview([]string{"--resolve", "not-a-number"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runQueueReview --resolve with a non-integer value = %d, want 1", code)
	}
}

func TestRunQueueReview_ResolveAndAllTogetherIsUsageError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runQueueReview([]string{"--resolve", "1", "--all"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runQueueReview --resolve with --all = %d, want 1", code)
	}
}

func TestParseTier(t *testing.T) {
	cases := map[string]workqueue.Tier{
		"blocker":      workqueue.Blocker,
		"current-task": workqueue.CurrentTask,
		"background":   workqueue.Background,
	}
	for s, want := range cases {
		got, err := parseTier(s)
		if err != nil {
			t.Errorf("parseTier(%q) returned unexpected error: %v", s, err)
			continue
		}
		if got != want {
			t.Errorf("parseTier(%q) = %v, want %v", s, got, want)
		}
	}
	if _, err := parseTier("urgent"); err == nil {
		t.Error("parseTier(\"urgent\") = nil error, want one")
	}
}

func TestPopNext_EmptyQueue(t *testing.T) {
	db := openTestQueueStore(t)
	var stdout bytes.Buffer
	code := popNext(db, &stdout)
	if code != 0 {
		t.Errorf("popNext on an empty queue = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "empty") {
		t.Errorf("stdout = %q, want it to report the queue is empty", stdout.String())
	}
}

func TestPopNext_ReturnsAndRemovesHighestPriorityItem(t *testing.T) {
	db := openTestQueueStore(t)
	now := time.Now().UTC()
	if err := workqueue.Add(db, workqueue.Item{ID: "background-item", Tier: workqueue.Background, Weight: 100, CreatedAt: now}); err != nil {
		t.Fatalf("Add returned unexpected error: %v", err)
	}
	if err := workqueue.Add(db, workqueue.Item{ID: "blocker-item", Tier: workqueue.Blocker, Weight: 1, CreatedAt: now}); err != nil {
		t.Fatalf("Add returned unexpected error: %v", err)
	}

	var stdout bytes.Buffer
	code := popNext(db, &stdout)
	if code != 0 {
		t.Fatalf("popNext = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "blocker-item") {
		t.Errorf("stdout = %q, want it to report the blocker-tier item", stdout.String())
	}

	items, err := workqueue.List(db)
	if err != nil {
		t.Fatalf("List returned unexpected error: %v", err)
	}
	if len(items) != 1 || items[0].ID != "background-item" {
		t.Errorf("remaining items = %+v, want only background-item left", items)
	}
}
