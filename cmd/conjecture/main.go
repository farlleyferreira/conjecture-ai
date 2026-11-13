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

	return dispatch(args, stdout, stderr)
}

// dispatch routes to the named subcommand, recovering from any panic
// inside it via [runRecovered]. A bug inside one of the 20+
// subcommands below should never crash the whole process with a raw
// Go stack trace on stderr - jarring for anyone who isn't the
// developer reading it, and in a partner demo actively embarrassing.
func dispatch(args []string, stdout, stderr io.Writer) int {
	return runRecovered(stderr, func() int {
		return route(args, stdout, stderr)
	})
}

// runRecovered calls fn, converting any panic into exit code 1 and a
// plain, single-line error on stderr - the same way every other
// failure in this CLI already reports - instead of letting it crash
// the process with a raw Go stack trace. It does not retry fn or
// otherwise treat the panic as recoverable mid-command.
func runRecovered(stderr io.Writer, fn func() int) (exitCode int) {
	defer func() {
		if r := recover(); r != nil {
			_, _ = fmt.Fprintf(stderr, "conjecture: internal error: %v\n", r)
			exitCode = 1
		}
	}()
	return fn()
}

// route is dispatch's own actual command table, split out so
// [runRecovered]'s panic handling wraps it without the switch itself
// needing to know anything about recovery.
func route(args []string, stdout, stderr io.Writer) int {
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
	case "law":
		return runLaw(args[1:], stdout, stderr)
	case "translate":
		return runTranslate(args[1:], stdout, stderr)
	case "settings":
		return runSettings(args[1:], stdout, stderr)
	case "optimize":
		return runOptimize(args[1:], stdout, stderr)
	case "watch":
		return runWatch(args[1:], stdout, stderr)
	case "graph":
		return runGraph(args[1:], stdout, stderr)
	case "attack":
		return runAttack(args[1:], stdout, stderr)
	case "deps":
		return runDeps(args[1:], stdout, stderr)
	case "promote":
		return runPromote(args[1:], stdout, stderr)
	case "swarm":
		return runSwarm(args[1:], stdout, stderr)
	case "init":
		return runInit(args[1:], stdout, stderr)
	case "status":
		return runStatus(args[1:], stdout, stderr)
	default:
		_, _ = fmt.Fprintf(stderr, "unknown command: %s\n", args[0])
		return 1
	}
}
