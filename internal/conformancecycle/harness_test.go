package conformancecycle

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/farlleyferreira/conjecture-ai/internal/cjtr"
	"github.com/farlleyferreira/conjecture-ai/internal/sandbox"
)

var fakeImplPath string

// TestMain compiles testdata/fakeimpl once, shared by every test in
// this file, the same reason and technique internal/sandbox's own
// tests already use: no real Docker daemon exists in the environment
// this was written in.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "fakeimpl")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	fakeImplPath = filepath.Join(dir, "fakeimpl")
	if runtime.GOOS == "windows" {
		fakeImplPath += ".exe"
	}

	// Located via PATH, not runtime.GOROOT() (deprecated since Go
	// 1.24 - its own value reflects where this test binary was
	// compiled, not necessarily a real toolchain install): go test
	// itself only ever runs via the go tool, so "go" being
	// unresolvable here would mean a broken environment, not
	// something worth a fallback for.
	goBin, err := exec.LookPath("go")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	buildCmd := exec.Command(goBin, "build", "-o", fakeImplPath, "./testdata/fakeimpl")
	if out, err := buildCmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "building fakeimpl: %v\n%s", err, out)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func testOpts(t *testing.T, command ...string) sandbox.Options {
	t.Helper()
	return sandbox.Options{Image: "unused", Dir: t.TempDir(), Command: command}
}

func TestRunHarness_WritesInputAndReadsOutputBack(t *testing.T) {
	sb := &sandbox.Sandbox{DockerBinary: fakeImplPath}
	opts := testOpts(t, "WRITE", `{"output.result":"42.00"}`)

	output, result, err := RunHarness(context.Background(), sb, opts, cjtr.Bindings{"input.x": "1"})
	if err != nil {
		t.Fatalf("RunHarness returned unexpected error: %v", err)
	}
	if result.ExitCode != 0 || result.TimedOut {
		t.Fatalf("result = %+v, want ExitCode 0, TimedOut false", result)
	}
	if output["output.result"] != "42.00" {
		t.Errorf("output = %v, want output.result = 42.00", output)
	}
}

// TestRunHarness_FiltersOutputKeysOutOfTheInputFile guards against a
// real finding (see [RunHarness]'s own doc comment): every real
// caller's own "input" still carries the hypothesis's own worked
// example's output value too (holding "every other concept at the
// baseline" while it varies one input concept at a time) - that must
// never reach InputFileName, or a sandboxed implementation could
// trust/echo an unrecognized "output.*" key straight back and pass
// without doing any real work.
func TestRunHarness_FiltersOutputKeysOutOfTheInputFile(t *testing.T) {
	sb := &sandbox.Sandbox{DockerBinary: fakeImplPath}
	opts := testOpts(t, "ECHO_INPUT")

	output, _, err := RunHarness(context.Background(), sb, opts, cjtr.Bindings{
		"input.x":       "1",
		"output.leaked": "the-expected-answer",
	})
	if err != nil {
		t.Fatalf("RunHarness returned unexpected error: %v", err)
	}
	if output["input.x"] != "1" {
		t.Errorf("output = %v, want the real input.x to have reached the sandboxed file", output)
	}
	if _, leaked := output["output.leaked"]; leaked {
		t.Errorf("output = %v, want output.leaked NOT to have reached the sandboxed input file", output)
	}
}

func TestRunHarness_CleansUpInputAndOutputFiles(t *testing.T) {
	sb := &sandbox.Sandbox{DockerBinary: fakeImplPath}
	opts := testOpts(t, "WRITE", `{"output.result":"1"}`)

	if _, _, err := RunHarness(context.Background(), sb, opts, cjtr.Bindings{}); err != nil {
		t.Fatalf("RunHarness returned unexpected error: %v", err)
	}
	for _, name := range []string{InputFileName, OutputFileName} {
		if _, err := os.Stat(filepath.Join(opts.Dir, name)); !os.IsNotExist(err) {
			t.Errorf("%s still exists after RunHarness, want it cleaned up", name)
		}
	}
}

func TestRunHarness_NonZeroExitReturnsResultNotError(t *testing.T) {
	sb := &sandbox.Sandbox{DockerBinary: fakeImplPath}
	opts := testOpts(t, "FAIL")

	output, result, err := RunHarness(context.Background(), sb, opts, cjtr.Bindings{})
	if err != nil {
		t.Fatalf("RunHarness returned unexpected error: %v", err)
	}
	if result.ExitCode != 3 {
		t.Errorf("result.ExitCode = %d, want 3", result.ExitCode)
	}
	if output != nil {
		t.Errorf("output = %v, want nil on a non-zero exit", output)
	}
}

func TestRunHarness_TimedOutReturnsResultNotError(t *testing.T) {
	sb := &sandbox.Sandbox{DockerBinary: fakeImplPath}
	opts := testOpts(t, "SLEEP_FOREVER")

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_, result, err := RunHarness(ctx, sb, opts, cjtr.Bindings{})
	if err != nil {
		t.Fatalf("RunHarness returned unexpected error: %v", err)
	}
	if !result.TimedOut {
		t.Error("result.TimedOut = false, want true")
	}
}

func TestRunHarness_MissingOutputFileIsError(t *testing.T) {
	sb := &sandbox.Sandbox{DockerBinary: fakeImplPath}
	opts := testOpts(t, "NO_OUTPUT")

	_, _, err := RunHarness(context.Background(), sb, opts, cjtr.Bindings{})
	if err == nil {
		t.Fatal("RunHarness with no output file written = nil error, want one")
	}
}

// outputValueHypothesis returns an "always" hypothesis - "the output
// value is 42.00" - and its one concept, for
// TestCheckHypothesisAgainstImplementation's cases.
func outputValueHypothesis(t *testing.T) (cjtr.Hypothesis, map[string]cjtr.Concept) {
	t.Helper()
	concepts := map[string]cjtr.Concept{
		"output value": {Name: "output value", Kind: cjtr.VariableConcept, Variable: "output.result", Type: "decimal"},
	}
	then, err := cjtr.ParseComparison("the output value is 42.00", concepts)
	if err != nil {
		t.Fatalf("ParseComparison returned unexpected error: %v", err)
	}
	return cjtr.Hypothesis{Name: "test", IsUniversal: true, Then: then}, concepts
}

func TestCheckHypothesisAgainstImplementation_Conformant(t *testing.T) {
	h, concepts := outputValueHypothesis(t)
	sb := &sandbox.Sandbox{DockerBinary: fakeImplPath}
	opts := testOpts(t, "WRITE", `{"output.result":"42.00"}`)

	verdict, err := CheckHypothesisAgainstImplementation(context.Background(), sb, opts, h, concepts, nil, cjtr.Bindings{})
	if err != nil {
		t.Fatalf("CheckHypothesisAgainstImplementation returned unexpected error: %v", err)
	}
	if verdict != Conformant {
		t.Errorf("verdict = %v, want Conformant", verdict)
	}
}

func TestCheckHypothesisAgainstImplementation_NonConformantOnWrongValue(t *testing.T) {
	h, concepts := outputValueHypothesis(t)
	sb := &sandbox.Sandbox{DockerBinary: fakeImplPath}
	opts := testOpts(t, "WRITE", `{"output.result":"99.00"}`)

	verdict, err := CheckHypothesisAgainstImplementation(context.Background(), sb, opts, h, concepts, nil, cjtr.Bindings{})
	if err != nil {
		t.Fatalf("CheckHypothesisAgainstImplementation returned unexpected error: %v", err)
	}
	if verdict != NonConformant {
		t.Errorf("verdict = %v, want NonConformant", verdict)
	}
}

func TestCheckHypothesisAgainstImplementation_NonConformantOnCrash(t *testing.T) {
	h, concepts := outputValueHypothesis(t)
	sb := &sandbox.Sandbox{DockerBinary: fakeImplPath}
	opts := testOpts(t, "FAIL")

	verdict, err := CheckHypothesisAgainstImplementation(context.Background(), sb, opts, h, concepts, nil, cjtr.Bindings{})
	if err != nil {
		t.Fatalf("CheckHypothesisAgainstImplementation returned unexpected error: %v", err)
	}
	if verdict != NonConformant {
		t.Errorf("verdict = %v, want NonConformant", verdict)
	}
}

func TestCheckHypothesisAgainstImplementation_InconclusiveOnTimeout(t *testing.T) {
	h, concepts := outputValueHypothesis(t)
	sb := &sandbox.Sandbox{DockerBinary: fakeImplPath}
	opts := testOpts(t, "SLEEP_FOREVER")

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	verdict, err := CheckHypothesisAgainstImplementation(ctx, sb, opts, h, concepts, nil, cjtr.Bindings{})
	if err != nil {
		t.Fatalf("CheckHypothesisAgainstImplementation returned unexpected error: %v", err)
	}
	if verdict != Inconclusive {
		t.Errorf("verdict = %v, want Inconclusive", verdict)
	}
}

func TestBindingsToExample_OnlyIncludesVariableConcepts(t *testing.T) {
	concepts := map[string]cjtr.Concept{
		"x":     {Name: "x", Kind: cjtr.VariableConcept, Variable: "input.x"},
		"fixed": {Name: "fixed", Kind: cjtr.ConstantConcept, Value: "100.00"},
	}
	bindings := cjtr.Bindings{"input.x": "5"}

	example := bindingsToExample(bindings, concepts)

	if example["x"] != "5" {
		t.Errorf("example[\"x\"] = %q, want \"5\"", example["x"])
	}
	if _, ok := example["fixed"]; ok {
		t.Error("example includes a non-variable concept, want it excluded")
	}
}

func TestBindingsToExample_IgnoresUnboundConcepts(t *testing.T) {
	concepts := map[string]cjtr.Concept{
		"x": {Name: "x", Kind: cjtr.VariableConcept, Variable: "input.x"},
	}
	example := bindingsToExample(cjtr.Bindings{}, concepts)
	if _, ok := example["x"]; ok {
		t.Error("example includes an unbound concept, want it excluded")
	}
}
