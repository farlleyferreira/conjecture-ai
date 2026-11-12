package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/farlleyferreira/conjecture-ai/internal/hardwarescheduler"
)

const watchUsage = "usage: conjecture watch <dir> [--interval <duration>]"

// watchDefaultInterval is this command's own disclosed default poll
// interval - the specification names no concrete latency for folder-
// watch observation, so this is this project's own chosen value, the
// same "disclosed default" posture every other unspecified constant
// in this codebase already takes.
const watchDefaultInterval = 2 * time.Second

// runWatch implements "conjecture watch <dir>" - cli-mcp-integration's
// own "automatic folder-watch observation" channel: its purpose text
// names MCP, CLI, and folder-watch side by side as the three
// integration channels, and "conjecture mcp"/"conjecture verify"
// already cover the first two.
//
// "Folder-watch observation is read-only": this only ever reads a
// .cjtr file's own content to hash it and, on a change, calls the
// exact same runVerify this CLI's own "verify" command already uses -
// which itself never writes to the file it checks. Nothing here adds
// a write path.
//
// "Verdicts delivered back through the same channel": the request
// (an automatically detected file change) and the response
// (runVerify's own stdout/stderr output) both happen within this same
// "conjecture watch" invocation - there is no other channel for a
// folder-watch-triggered notification to go through, so printing to
// this process's own stdout/stderr is that channel.
//
// Honesty note: this polls for changes (hashing every .cjtr file
// under dir on a fixed interval) rather than using an OS-specific
// file-event API (inotify, ReadDirectoryChangesW, FSEvents) - a
// deliberately simpler, dependency-free choice the specification does
// not require against, consistent with "Optional runtime adapters for
// introspection... a rule's conformance verdict SHALL NOT depend on
// any such adapter being present" (language-agnostic-contracts' own
// posture toward making a real-time, platform-specific mechanism
// optional rather than load-bearing).
func runWatch(args []string, stdout, stderr io.Writer) int {
	dir, interval, ok := parseWatchArgs(args)
	if !ok {
		_, _ = fmt.Fprintln(stderr, watchUsage)
		return 1
	}
	if info, err := os.Stat(dir); err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	} else if !info.IsDir() {
		_, _ = fmt.Fprintf(stderr, "conjecture: %q is not a directory\n", dir)
		return 1
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	return runWatchLoop(ctx, dir, interval, hardwarescheduler.CurrentCPUSample, stdout, stderr)
}

// runWatchLoop is runWatch's own real polling loop, factored out so a
// test can drive it with a context it controls directly (cancelling
// it to simulate the interrupt signal runWatch itself listens for)
// instead of having to deliver a real OS signal to the test process.
// cpuSample is likewise factored out - runWatch's own call passes
// [hardwarescheduler.CurrentCPUSample], the real per-OS reading; a
// test passes a deterministic stub instead, so its own timing never
// depends on the actual machine's CPU load at the moment it runs.
func runWatchLoop(ctx context.Context, dir string, interval time.Duration, cpuSample func() (hardwarescheduler.CPUSample, error), stdout, stderr io.Writer) int {
	seen := make(map[string][32]byte)
	// The first scan establishes a baseline silently - "folder-watch
	// observation" is about detecting a *change*, not re-verifying
	// every file that already existed before watching started.
	if err := scanOnce(dir, seen, io.Discard, io.Discard); err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}
	prevCPU, prevErr := cpuSample()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return 0
		case <-ticker.C:
			curCPU, curErr := cpuSample()
			shouldDefer := shouldDeferThisTick(prevCPU, prevErr, curCPU, curErr)
			prevCPU, prevErr = curCPU, curErr

			if shouldDefer {
				_, _ = fmt.Fprintln(stdout, "watch: deferring this scan - machine is busy")
				continue
			}
			if err := scanOnce(dir, seen, stdout, stderr); err != nil {
				_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
			}
		}
	}
}

// shouldDeferThisTick reports whether runWatchLoop should skip this
// tick's own scan - hardware-scheduler's own "Heavy work only on an
// idle machine" ("the system SHALL NOT make the user's interactive
// work feel slower as a result of background verification"), applied
// via [hardwarescheduler.ShouldDeferHeavyWork] with this loop's own
// CPU-busyness reading (see cpuload.go's own doc comment for why
// overall CPU load, not literal input-idle detection, is this
// package's chosen proxy for "userActive") as that policy's
// userActive signal. systemUnderTestLoad is passed as false: nothing
// in this command measures that fact, and false is the honest "not
// known to be true" default, not a guess that it never applies.
//
// A measurement error on either sample (prevErr or curErr) never
// defers: an unmeasurable load is read as "not busy," not as a
// reason to stall watch's own verification over what is, even when
// it works, only a best-effort signal.
func shouldDeferThisTick(prevCPU hardwarescheduler.CPUSample, prevErr error, curCPU hardwarescheduler.CPUSample, curErr error) bool {
	if prevErr != nil || curErr != nil {
		return false
	}
	busy := hardwarescheduler.BusyFraction(prevCPU, curCPU) >= hardwarescheduler.CPUBusyThreshold
	return hardwarescheduler.ShouldDeferHeavyWork(false, busy)
}

// parseWatchArgs parses runWatch's own arguments: the required
// directory, plus the optional --interval flag (a Go duration string,
// e.g. "500ms", "5s"). ok is false on any malformed invocation.
func parseWatchArgs(args []string) (dir string, interval time.Duration, ok bool) {
	interval = watchDefaultInterval
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--interval":
			i++
			if i >= len(args) {
				return "", 0, false
			}
			d, err := time.ParseDuration(args[i])
			if err != nil {
				return "", 0, false
			}
			interval = d
		default:
			if dir != "" {
				return "", 0, false
			}
			dir = args[i]
		}
	}
	if dir == "" {
		return "", 0, false
	}
	return dir, interval, true
}

// scanOnce hashes every ".cjtr" file under dir and, for any whose
// content hash differs from (or is new since) what seen already
// records, runs the real "conjecture verify" against it - the one
// read, and the one action, "folder-watch observation" takes. seen is
// updated in place with every file's current hash, including
// unchanged ones, and a file seen on a previous scan but no longer
// present is dropped from it, so the next call only ever reports what
// actually changed between the two scans.
func scanOnce(dir string, seen map[string][32]byte, stdout, stderr io.Writer) error {
	current := make(map[string]bool)
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".cjtr") {
			return nil
		}
		current[path] = true

		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		hash := sha256.Sum256(raw)

		if previous, ok := seen[path]; ok && previous == hash {
			return nil
		}
		seen[path] = hash

		_, _ = fmt.Fprintf(stdout, "watch: %s changed\n", path)
		_ = runVerify([]string{path}, stdout, stderr)
		return nil
	})
	if err != nil {
		return err
	}
	for path := range seen {
		if !current[path] {
			delete(seen, path)
		}
	}
	return nil
}
