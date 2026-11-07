package expressioncore

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// SatResult is one SMT query's three-way outcome - the same three
// outcomes any SMT-LIB2 "(check-sat)" command produces.
type SatResult int

const (
	Unknown SatResult = iota
	Sat
	Unsat
)

func (r SatResult) String() string {
	switch r {
	case Sat:
		return "sat"
	case Unsat:
		return "unsat"
	default:
		return "unknown"
	}
}

// Solver runs an SMT-LIB2 query (as [BuildSatisfiabilityQuery]
// produces) through a real SMT solver binary - z3, invoked as a
// subprocess reading the query from stdin ("z3 -in"), the same
// external-binary pattern internal/sandbox already uses for Docker
// ("--network none" via a real docker daemon), not CGO bindings to
// libz3: no mature pure-Go SMT solver with arithmetic-theory support
// exists (researched, not assumed), and a subprocess keeps this
// package - and everything that imports it - buildable without a C
// toolchain or libz3 installed to even compile, only to run this one
// code path for real.
//
// The zero value is ready to use against a real local z3 install
// (Z3Binary defaults to "z3", resolved via PATH).
type Solver struct {
	// Z3Binary overrides the z3 executable to invoke - a test hook (a
	// real caller has z3 on its PATH; this package does not install or
	// locate it).
	Z3Binary string
}

// CheckSatisfiability runs query through z3 ("z3 -in", reading query
// from stdin) and reports its verdict.
func (s Solver) CheckSatisfiability(ctx context.Context, query string) (SatResult, error) {
	binary := s.Z3Binary
	if binary == "" {
		binary = "z3"
	}

	cmd := exec.CommandContext(ctx, binary, "-in")
	cmd.Stdin = strings.NewReader(query)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return Unknown, fmt.Errorf("expressioncore: running %s: %w (stderr: %s)", binary, err, stderr.String())
	}

	return parseSatResult(stdout.String())
}

// parseSatResult reads z3's own first line of output to "(check-sat)"
// - "sat", "unsat", or "unknown", exactly as the SMT-LIB2 standard
// names them - ignoring anything after it (a later "(get-model)" or
// similar this package never asks for, but a real z3 install's own
// default configuration might still echo).
func parseSatResult(output string) (SatResult, error) {
	line := strings.TrimSpace(strings.SplitN(output, "\n", 2)[0])
	switch line {
	case "sat":
		return Sat, nil
	case "unsat":
		return Unsat, nil
	case "unknown":
		return Unknown, nil
	default:
		return Unknown, fmt.Errorf("expressioncore: unrecognized solver output %q", line)
	}
}
