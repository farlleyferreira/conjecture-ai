// Command conjecture is the conjecture CLI entrypoint.
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/farlleyferreira/conjecture-ai/internal/version"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run executes the CLI with the given arguments and returns the process exit code.
func run(args []string, stdout, stderr io.Writer) int {
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
	default:
		_, _ = fmt.Fprintf(stderr, "unknown command: %s\n", args[0])
		return 1
	}
}
