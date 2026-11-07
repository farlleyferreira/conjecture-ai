package main

import (
	"bytes"
	"database/sql"
	"path/filepath"
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
