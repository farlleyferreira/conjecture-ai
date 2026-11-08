package expressioncore

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

var fakeZ3Path string

// TestMain compiles testdata/fakez3 once, into a temp directory
// shared by every test in this package, so Solver's own subprocess
// orchestration can be exercised without a real Z3 install - not
// available in the environment this was written in.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "fakez3")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	fakeZ3Path = filepath.Join(dir, "fakez3")
	if runtime.GOOS == "windows" {
		fakeZ3Path += ".exe"
	}

	buildCmd := exec.Command(goBinary(), "build", "-o", fakeZ3Path, "./testdata/fakez3")
	if out, err := buildCmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "building fakez3: %v\n%s", err, out)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

// goBinary locates the go toolchain via PATH, not runtime.GOROOT()
// (deprecated since Go 1.24 - its own value reflects where this test
// binary was compiled, not necessarily a real toolchain install): go
// test itself only ever runs via the go tool, so "go" being
// unresolvable here would mean a broken environment, not something
// worth a fallback for.
func goBinary() string {
	path, err := exec.LookPath("go")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	return path
}

func TestCheckSatisfiability_Sat(t *testing.T) {
	s := Solver{Z3Binary: fakeZ3Path}
	result, err := s.CheckSatisfiability(context.Background(), "(declare-const x Real)\n(assert (> x 0))\n(check-sat)\n")
	if err != nil {
		t.Fatalf("CheckSatisfiability returned unexpected error: %v", err)
	}
	if result != Sat {
		t.Errorf("CheckSatisfiability = %v, want Sat", result)
	}
}

func TestCheckSatisfiability_Unsat(t *testing.T) {
	t.Setenv("FAKEZ3_RESULT", "unsat")
	s := Solver{Z3Binary: fakeZ3Path}
	result, err := s.CheckSatisfiability(context.Background(), "(declare-const x Real)\n(assert (>= x 50))\n(assert (<= x 10))\n(check-sat)\n")
	if err != nil {
		t.Fatalf("CheckSatisfiability returned unexpected error: %v", err)
	}
	if result != Unsat {
		t.Errorf("CheckSatisfiability = %v, want Unsat", result)
	}
}

func TestCheckSatisfiability_Unknown(t *testing.T) {
	t.Setenv("FAKEZ3_RESULT", "unknown")
	s := Solver{Z3Binary: fakeZ3Path}
	result, err := s.CheckSatisfiability(context.Background(), "(check-sat)\n")
	if err != nil {
		t.Fatalf("CheckSatisfiability returned unexpected error: %v", err)
	}
	if result != Unknown {
		t.Errorf("CheckSatisfiability = %v, want Unknown", result)
	}
}

func TestCheckSatisfiability_SolverProcessError(t *testing.T) {
	t.Setenv("FAKEZ3_RESULT", "error")
	s := Solver{Z3Binary: fakeZ3Path}
	if _, err := s.CheckSatisfiability(context.Background(), "(check-sat)\n"); err == nil {
		t.Error("CheckSatisfiability with a failing solver process = nil error, want one")
	}
}

func TestCheckSatisfiability_UnrecognizedOutputIsError(t *testing.T) {
	t.Setenv("FAKEZ3_RESULT", "garbage")
	s := Solver{Z3Binary: fakeZ3Path}
	if _, err := s.CheckSatisfiability(context.Background(), "(check-sat)\n"); err == nil {
		t.Error("CheckSatisfiability with unrecognized solver output = nil error, want one")
	}
}

func TestCheckSatisfiability_QueryDeliveredViaStdin(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "received-query")
	t.Setenv("FAKEZ3_INPUT_MARKER", marker)
	s := Solver{Z3Binary: fakeZ3Path}

	query := BuildSatisfiabilityQuery("shipping_fee", []string{"(>= shipping_fee 50)"})
	if _, err := s.CheckSatisfiability(context.Background(), query); err != nil {
		t.Fatalf("CheckSatisfiability returned unexpected error: %v", err)
	}

	received, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("reading marker file: %v", err)
	}
	if string(received) != query {
		t.Errorf("solver received %q over stdin, want exactly the query %q", received, query)
	}
}

func TestSatResult_String(t *testing.T) {
	cases := map[SatResult]string{Sat: "sat", Unsat: "unsat", Unknown: "unknown"}
	for result, want := range cases {
		if got := result.String(); got != want {
			t.Errorf("%v.String() = %q, want %q", result, got, want)
		}
	}
}
