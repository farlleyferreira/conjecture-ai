// Command conjecture is the conjecture CLI entrypoint.
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/farlleyferreira/conjecture-ai/internal/hardwarescheduler"
	"github.com/farlleyferreira/conjecture-ai/internal/version"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run executes the CLI with the given arguments and returns the process exit code.
func run(args []string, stdout, stderr io.Writer) int {
	// hardware-scheduler's "Minimum hardware requirement": "WHEN
	// conjecture starts on a machine with less than 6 GB of RAM THEN
	// the system reports that the minimum hardware requirement is not
	// met." The specification asks only that this be reported, not
	// that the command refuse to run, so this warns on stderr without
	// affecting the exit code.
	if err := hardwarescheduler.CheckMinimumRAM(); err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: warning: %v\n", err)
	}

	if len(args) == 0 {
		_, _ = fmt.Fprintln(stderr, "usage: conjecture <command>")
		return 1
	}

	switch args[0] {
	case "version":
		_, _ = fmt.Fprintln(stdout, version.String())
		return 0
	case "verify":
		return runVerify(args[1:], stdout, stderr)
	case "sign":
		return runSign(args[1:], stdout, stderr)
	case "base":
		return runBase(args[1:], stdout, stderr)
	case "context":
		return runContext(args[1:], stdout, stderr)
	case "draft":
		return runDraft(args[1:], stdout, stderr)
	case "check":
		return runCheck(args[1:], stdout, stderr)
	case "implement":
		return runImplement(args[1:], stdout, stderr)
	case "queue":
		return runQueue(args[1:], stdout, stderr)
	case "mcp":
		return runMCP(args[1:], stdout, stderr)
	case "work":
		return runWork(args[1:], stdout, stderr)
	case "results":
		return runResults(args[1:], stdout, stderr)
	default:
		_, _ = fmt.Fprintf(stderr, "unknown command: %s\n", args[0])
		return 1
	}
}
