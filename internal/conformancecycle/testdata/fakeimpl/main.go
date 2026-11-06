// Command fakeimpl stands in for both "docker" and the implementation
// code running inside it, for harness_test.go: it is not containerized
// (there is no real Docker here), so it finds the host directory a
// real "docker run -v HOSTDIR:/workspace" would have mounted from its
// own "-v" argument, and reads/writes conjecture_input.json/
// conjecture_output.json directly there, exactly where a real
// containerized process would see them at /workspace.
//
// Its own command sentinels (the trailing positional arguments a test
// passes as sandbox.Options.Command) are this test fixture's own
// convention, not conjecture's harness convention:
//   - "FAIL": exits non-zero.
//   - "NO_OUTPUT": exits 0 without writing an output file.
//   - "SLEEP_FOREVER": blocks, for a test to time out against.
//   - "WRITE", "<json>": writes <json> as conjecture_output.json
//     verbatim, ignoring the input file - full control over the
//     implementation's own "output" for a test, independent of input.
package main

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	args := os.Args[1:]

	if len(args) > 0 && args[0] == "kill" {
		os.Exit(0)
	}

	var hostDir string
	for i, a := range args {
		if a == "-v" && i+1 < len(args) {
			// Split on the known ":/workspace" suffix, not the first
			// colon: a Windows host path already has one right after
			// its drive letter (e.g. "C:\Users\...\t\001"), which
			// would otherwise be mistaken for the host:container
			// separator.
			if host, ok := strings.CutSuffix(args[i+1], ":/workspace"); ok {
				hostDir = host
			}
		}
	}

	last := args[len(args)-1]
	switch {
	case last == "FAIL":
		os.Stderr.WriteString("implementation crashed\n")
		os.Exit(3)
	case last == "NO_OUTPUT":
		os.Exit(0)
	case last == "SLEEP_FOREVER":
		time.Sleep(10 * time.Second)
		os.Exit(0)
	case len(args) >= 2 && args[len(args)-2] == "WRITE":
		if err := os.WriteFile(filepath.Join(hostDir, "conjecture_output.json"), []byte(last), 0o644); err != nil {
			os.Stderr.WriteString(err.Error())
			os.Exit(1)
		}
		os.Exit(0)
	default:
		os.Stderr.WriteString("fakeimpl: unrecognized command " + last + "\n")
		os.Exit(1)
	}
}
