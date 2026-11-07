// Command fakez3 stands in for the real "z3" binary in solver_test.go,
// so Solver.CheckSatisfiability's own subprocess orchestration
// (stdin delivery, output parsing, error handling) can be exercised
// without a real Z3 install - not available in the environment this
// was written in. It is not built as part of the module's normal
// build (testdata is ignored by the go tool); solver_test.go compiles
// it once, on demand, with "go build".
package main

import (
	"fmt"
	"io"
	"os"
)

func main() {
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Lets a test assert exactly what query text CheckSatisfiability
	// actually sent over stdin, the same "write a marker file" pattern
	// internal/sandbox's own fakedocker uses for its kill marker.
	if marker := os.Getenv("FAKEZ3_INPUT_MARKER"); marker != "" {
		_ = os.WriteFile(marker, input, 0o644)
	}

	switch os.Getenv("FAKEZ3_RESULT") {
	case "error":
		fmt.Fprintln(os.Stderr, "fakez3: simulated failure")
		os.Exit(1)
	case "unsat":
		fmt.Println("unsat")
	case "unknown":
		fmt.Println("unknown")
	case "garbage":
		fmt.Println("not-a-real-sat-result")
	default:
		fmt.Println("sat")
	}
}
