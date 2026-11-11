package main

import (
	"bytes"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	"github.com/farlleyferreira/conjecture-ai/internal/contextengine"
	"github.com/farlleyferreira/conjecture-ai/internal/evidenceledger"
	"github.com/farlleyferreira/conjecture-ai/internal/persistence"
)

func openGraphTestStore(t *testing.T) *sql.DB {
	t.Helper()
	db, err := persistence.OpenStoreAt(filepath.Join(t.TempDir(), "data"))
	if err != nil {
		t.Fatalf("OpenStoreAt returned unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := contextengine.EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema returned unexpected error: %v", err)
	}
	return db
}

func TestRunGraph_NoSubcommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runGraph(nil, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runGraph with no subcommand: exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Errorf("stderr should contain a usage message, got %q", stderr.String())
	}
}

func TestRunGraph_UnknownSubcommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runGraph([]string{"bogus"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runGraph with an unknown subcommand: exit code = %d, want 1", code)
	}
}

func TestRunGraphSet_WrongArgCount(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runGraph([]string{"set", "n1"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("set with one argument: exit code = %d, want 1", code)
	}
}

func TestRunGraphSet_UnrecognizedWeightIsError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runGraph([]string{"set", "n1", "--weight", "extreme"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("set with an unrecognized weight: exit code = %d, want 1", code)
	}
}

func TestRunGraphSet_UnrecognizedFlagIsUsageError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runGraph([]string{"set", "n1", "--bogus", "high"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("set with an unrecognized flag: exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Errorf("stderr should contain a usage message, got %q", stderr.String())
	}
}

func TestGraphSet_ReportsTheAxisSet(t *testing.T) {
	db := openGraphTestStore(t)

	var stdout, stderr bytes.Buffer
	code := graphSet(db, contextengine.SetWeight, "weight", "n1", evidenceledger.High, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("graphSet returned exit code %d, want 0; stderr:\n%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "weight set to high") {
		t.Errorf("stdout should report the axis set, got %q", stdout.String())
	}

	axes, found, err := contextengine.GetAxes(db, "n1")
	if err != nil {
		t.Fatalf("GetAxes returned unexpected error: %v", err)
	}
	if !found || axes.Weight == nil || *axes.Weight != evidenceledger.High {
		t.Errorf("GetAxes = %+v, %v, want Weight=High, true", axes, found)
	}
}

func TestRunGraphInspect_WrongArgCount(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runGraph([]string{"inspect"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("inspect with no node id: exit code = %d, want 1", code)
	}
}

func TestGraphInspect_UnknownNodeReportsNoAxesRecorded(t *testing.T) {
	db := openGraphTestStore(t)

	var stdout, stderr bytes.Buffer
	code := graphInspect(db, "nope", &stdout, &stderr)
	if code != 0 {
		t.Fatalf("graphInspect returned exit code %d, want 0; stderr:\n%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "no axes recorded") {
		t.Errorf("stdout should report no axes recorded, got %q", stdout.String())
	}
}

func TestGraphInspect_ReportsEachAxisSeparately(t *testing.T) {
	db := openGraphTestStore(t)
	if err := contextengine.SetWeight(db, "n1", evidenceledger.Critical); err != nil {
		t.Fatalf("SetWeight returned unexpected error: %v", err)
	}
	if err := contextengine.SetRelevance(db, "n1", evidenceledger.Low); err != nil {
		t.Fatalf("SetRelevance returned unexpected error: %v", err)
	}
	// Confidence deliberately left unset.

	var stdout, stderr bytes.Buffer
	code := graphInspect(db, "n1", &stdout, &stderr)
	if code != 0 {
		t.Fatalf("graphInspect returned exit code %d, want 0; stderr:\n%s", code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "weight:     critical") {
		t.Errorf("stdout should report weight critical, got:\n%s", out)
	}
	if !strings.Contains(out, "relevance:  low") {
		t.Errorf("stdout should report relevance low, got:\n%s", out)
	}
	if !strings.Contains(out, "confidence: not set") {
		t.Errorf("stdout should report confidence as not set, got:\n%s", out)
	}
}
