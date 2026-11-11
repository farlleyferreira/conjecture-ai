package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestRunWatch_NoArguments(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runWatch(nil, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runWatch with no arguments: exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Errorf("stderr should contain a usage message, got %q", stderr.String())
	}
}

func TestRunWatch_MissingDirectory(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runWatch([]string{filepath.Join(t.TempDir(), "nope")}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runWatch with a missing directory: exit code = %d, want 1", code)
	}
}

func TestRunWatch_PathIsAFileNotDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.cjtr")
	if err := os.WriteFile(path, []byte("conjecture: 1.0.0"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := runWatch([]string{path}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runWatch given a file instead of a directory: exit code = %d, want 1", code)
	}
}

// TestRunWatchLoop_DetectsChangeAndStopsOnContextCancel runs the real
// polling loop (runWatchLoop, the same code runWatch's own signal-
// wired wrapper calls) on a short interval: writes a .cjtr file after
// the loop has already started (so it is a genuine detected *change*,
// not the baseline), confirms the real "conjecture verify" output for
// it appears on stdout, then cancels the context - standing in for
// the interrupt signal runWatch itself listens for - and confirms the
// loop actually stops and returns 0, instead of just trusting
// scanOnce's own unit tests to imply the surrounding loop works.
func TestRunWatchLoop_DetectsChangeAndStopsOnContextCancel(t *testing.T) {
	dir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var stdout syncBuffer
	var stderr bytes.Buffer
	done := make(chan int, 1)
	go func() {
		done <- runWatchLoop(ctx, dir, 20*time.Millisecond, &stdout, &stderr)
	}()

	// Give the baseline scan a moment to run before writing the file,
	// so this is unambiguously a change, not part of the baseline.
	time.Sleep(60 * time.Millisecond)
	path := filepath.Join(dir, "demo.cjtr")
	if err := os.WriteFile(path, []byte(checkFixture), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for !strings.Contains(stdout.String(), "examples consistent") {
		if time.Now().After(deadline) {
			t.Fatalf("runWatchLoop did not report the change in time; stdout so far:\n%s", stdout.String())
		}
		time.Sleep(10 * time.Millisecond)
	}

	cancel()

	select {
	case code := <-done:
		if code != 0 {
			t.Errorf("runWatchLoop exit code = %d, want 0 (clean shutdown on cancellation)", code)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("runWatchLoop did not return after the context was cancelled")
	}
}

// syncBuffer is a bytes.Buffer safe to read from the test goroutine
// while runWatch's own goroutine writes to it concurrently.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

func TestParseWatchArgs_DefaultInterval(t *testing.T) {
	dir, interval, ok := parseWatchArgs([]string{"somedir"})
	if !ok {
		t.Fatal("parseWatchArgs returned ok = false, want true")
	}
	if dir != "somedir" {
		t.Errorf("dir = %q, want %q", dir, "somedir")
	}
	if interval != watchDefaultInterval {
		t.Errorf("interval = %v, want the default %v", interval, watchDefaultInterval)
	}
}

func TestParseWatchArgs_CustomInterval(t *testing.T) {
	dir, interval, ok := parseWatchArgs([]string{"somedir", "--interval", "500ms"})
	if !ok {
		t.Fatal("parseWatchArgs returned ok = false, want true")
	}
	if dir != "somedir" || interval != 500*time.Millisecond {
		t.Errorf("dir = %q, interval = %v, want %q, 500ms", dir, interval, "somedir")
	}
}

func TestParseWatchArgs_MissingDirIsUsageError(t *testing.T) {
	_, _, ok := parseWatchArgs([]string{"--interval", "1s"})
	if ok {
		t.Error("parseWatchArgs with no directory returned ok = true, want false")
	}
}

func TestParseWatchArgs_IntervalFlagMissingValueIsUsageError(t *testing.T) {
	_, _, ok := parseWatchArgs([]string{"somedir", "--interval"})
	if ok {
		t.Error("parseWatchArgs with a valueless --interval returned ok = true, want false")
	}
}

func TestParseWatchArgs_MalformedIntervalIsUsageError(t *testing.T) {
	_, _, ok := parseWatchArgs([]string{"somedir", "--interval", "not-a-duration"})
	if ok {
		t.Error("parseWatchArgs with a malformed --interval returned ok = true, want false")
	}
}

func TestParseWatchArgs_TwoPositionalArgsIsUsageError(t *testing.T) {
	_, _, ok := parseWatchArgs([]string{"dir-one", "dir-two"})
	if ok {
		t.Error("parseWatchArgs with two positional args returned ok = true, want false")
	}
}

func TestScanOnce_BaselineScanRecordsHashRegardlessOfWriter(t *testing.T) {
	// runWatch's own baseline scan passes io.Discard deliberately -
	// scanOnce itself still reports a first-ever sighting like any
	// other change, it is the caller's choice of writer that silences
	// it.
	dir := t.TempDir()
	path := filepath.Join(dir, "demo.cjtr")
	if err := os.WriteFile(path, []byte(checkFixture), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	seen := make(map[string][32]byte)
	if err := scanOnce(dir, seen, io.Discard, io.Discard); err != nil {
		t.Fatalf("scanOnce returned unexpected error: %v", err)
	}
	if _, ok := seen[path]; !ok {
		t.Error("seen should now record this file's hash")
	}
}

func TestScanOnce_ReportsAndVerifiesAChangedFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "demo.cjtr")
	if err := os.WriteFile(path, []byte(checkFixture), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	seen := make(map[string][32]byte)
	// Baseline scan, discarding output - same as runWatch's own first
	// scan.
	if err := scanOnce(dir, seen, io.Discard, io.Discard); err != nil {
		t.Fatalf("scanOnce returned unexpected error: %v", err)
	}

	// No change yet: a second scan should report nothing.
	var stdout, stderr bytes.Buffer
	if err := scanOnce(dir, seen, &stdout, &stderr); err != nil {
		t.Fatalf("scanOnce returned unexpected error: %v", err)
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty (nothing changed)", stdout.String())
	}

	// Change the file's content, then scan again.
	changed := strings.Replace(checkFixture, "7.50", "8.00", 1)
	if err := os.WriteFile(path, []byte(changed), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	stdout.Reset()
	if err := scanOnce(dir, seen, &stdout, &stderr); err != nil {
		t.Fatalf("scanOnce returned unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), "changed") {
		t.Errorf("stdout should report the change, got:\n%s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "examples consistent") {
		t.Errorf("stdout should include runVerify's own output, got:\n%s", stdout.String())
	}
}

func TestScanOnce_NonCjtrFilesIgnored(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	seen := make(map[string][32]byte)
	var stdout, stderr bytes.Buffer
	if err := scanOnce(dir, seen, &stdout, &stderr); err != nil {
		t.Fatalf("scanOnce returned unexpected error: %v", err)
	}
	if len(seen) != 0 {
		t.Errorf("seen = %v, want empty (no .cjtr files in this directory)", seen)
	}
}

func TestScanOnce_RemovedFileDroppedFromSeen(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "demo.cjtr")
	if err := os.WriteFile(path, []byte(checkFixture), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	seen := make(map[string][32]byte)
	if err := scanOnce(dir, seen, io.Discard, io.Discard); err != nil {
		t.Fatalf("scanOnce returned unexpected error: %v", err)
	}
	if len(seen) != 1 {
		t.Fatalf("len(seen) = %d, want 1", len(seen))
	}

	if err := os.Remove(path); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}
	if err := scanOnce(dir, seen, io.Discard, io.Discard); err != nil {
		t.Fatalf("scanOnce returned unexpected error: %v", err)
	}
	if len(seen) != 0 {
		t.Errorf("seen = %v, want empty (the file was removed)", seen)
	}
}
