// Package conformancecycle's harness.go closes the one connection
// this package's other files (budget.go, counterexample.go, rigor.go,
// verdict.go) deliberately left open: actually running implementation
// code inside the sandbox (internal/sandbox) against a generated
// input (internal/cjtr's BoundaryValues) and checking its real output
// against a hypothesis.
//
// Doing that needs a concrete way to deliver input values into a
// running implementation and read its output back out. Nothing in
// this specification or the product proposal document names one -
// unlike JUnit, TAP, and JSON Schema, explicitly named as the
// standard *result* formats (language-agnostic-contracts), nothing
// names a standard *input* delivery mechanism. The convention below
// is this project's own invention to make an end-to-end check
// actually runnable, not something any specification requires or
// describes - the same kind of judgment call this project has made
// before when a specification was silent (the table-concept path
// convention, knowledge-bases' local-only storage reading), picking
// the least committal, most standard-adjacent shape available (a flat
// JSON object, since internal/cjtr.Bindings already is one) rather
// than inventing nothing and leaving the connection unbuilt.
package conformancecycle

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/farlleyferreira/conjecture-ai/internal/cjtr"
	"github.com/farlleyferreira/conjecture-ai/internal/sandbox"
)

// InputFileName and OutputFileName are this package's own invented
// input/output convention. Implementation code under test (opts.
// Command) is expected to read a JSON object of variable name (the
// same dotted "input.x"/"output.y" form a concept's own "variable"
// field declares) to string value from InputFileName in its working
// directory, and write one in the same shape to OutputFileName before
// it exits. Translating between these variable-path keys and
// whatever the real implementation's own parameters or API actually
// look like is the one adapter a real integration still has to write
// for itself - this package does not attempt that translation, since
// nothing specifies what the real interface on the other side looks
// like either.
const (
	InputFileName  = "conjecture_input.json"
	OutputFileName = "conjecture_output.json"
)

// RunHarness runs opts.Command inside sb against input, following
// this package's own InputFileName/OutputFileName convention, and
// returns the resulting output bindings alongside the sandbox's own
// result.
//
// input is filtered to its own "input."-prefixed bindings before it
// is written to InputFileName, even though every real caller
// (directedsampling.go, boundedproof.go, completeproof.go) actually
// passes a full baseline-derived binding set that still carries the
// hypothesis's own worked example's *output* value too (each of those
// callers only ever varies one input concept at a time, holding
// "every other concept at the baseline" - including the baseline's
// own output). Marshaling that unfiltered would hand the sandboxed
// implementation the very answer the check exists to ask it for,
// under a key ("output.<x>") indistinguishable from any other input
// variable - found for real wrapping a genuine third-party library
// (python-slugify) for conformance checking, not by inspection: an
// implementation that happens to trust or echo back unrecognized
// input keys would otherwise pass without doing any real work at all.
//
// RunHarness returns a non-nil error only for an infrastructure
// failure of its own (encoding/writing the input file, the sandbox
// itself failing to even start, or - once the implementation reports
// success - a missing or malformed output file: a process that exits
// 0 without producing OutputFileName has not actually followed the
// convention, which is this package's failure to catch, not the
// implementation's conformance to judge). A non-zero exit or a
// timeout from the implementation itself is not an error here - the
// caller reads those from the returned sandbox.Result and decides
// what verdict follows, the same way [CheckHypothesisAgainstImplementation]
// does.
func RunHarness(ctx context.Context, sb *sandbox.Sandbox, opts sandbox.Options, input cjtr.Bindings) (cjtr.Bindings, sandbox.Result, error) {
	inputPath := filepath.Join(opts.Dir, InputFileName)
	inputJSON, err := json.Marshal(inputOnly(input))
	if err != nil {
		return nil, sandbox.Result{}, fmt.Errorf("conformancecycle: encoding harness input: %w", err)
	}
	if err := os.WriteFile(inputPath, inputJSON, 0o644); err != nil {
		return nil, sandbox.Result{}, fmt.Errorf("conformancecycle: writing harness input file: %w", err)
	}
	defer func() { _ = os.Remove(inputPath) }()

	outputPath := filepath.Join(opts.Dir, OutputFileName)
	_ = os.Remove(outputPath) // a stale file from a previous run must never be mistaken for this run's own output
	defer func() { _ = os.Remove(outputPath) }()

	result, err := sb.Run(ctx, opts)
	if err != nil {
		return nil, sandbox.Result{}, err
	}
	if result.TimedOut || result.ExitCode != 0 {
		return nil, result, nil
	}

	outputJSON, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, result, fmt.Errorf("conformancecycle: implementation exited 0 but did not write %s: %w", OutputFileName, err)
	}
	var output cjtr.Bindings
	if err := json.Unmarshal(outputJSON, &output); err != nil {
		return nil, result, fmt.Errorf("conformancecycle: decoding %s: %w", OutputFileName, err)
	}
	return output, result, nil
}

// inputOnly returns bindings's own "input."-prefixed entries only -
// see [RunHarness]'s own doc comment for why: a caller's bindings may
// still carry a baseline worked example's output value too, which
// must never reach the sandboxed implementation as if it were a
// genuine input.
func inputOnly(bindings cjtr.Bindings) cjtr.Bindings {
	filtered := make(cjtr.Bindings, len(bindings))
	for k, v := range bindings {
		if strings.HasPrefix(k, "input.") {
			filtered[k] = v
		}
	}
	return filtered
}

// CheckHypothesisAgainstImplementation runs one real conformance check
// for h: it runs the implementation via [RunHarness] against input,
// merges its output back into input, and checks the merged example
// against h via [cjtr.CheckExample] - the first actual end-to-end
// connection between a generated input, a sandboxed real
// implementation, and a hypothesis's own predicate.
//
// A sandbox timeout reports Inconclusive, per this package's own
// VerdictFromOutcome rule. A non-zero exit reports NonConformant: an
// implementation that cannot even produce a result for a given input
// is read here as failing to conform for it, not as a separate kind
// of failure the specification does not name.
func CheckHypothesisAgainstImplementation(
	ctx context.Context,
	sb *sandbox.Sandbox,
	opts sandbox.Options,
	h cjtr.Hypothesis,
	concepts map[string]cjtr.Concept,
	tables cjtr.Tables,
	input cjtr.Bindings,
) (Verdict, error) {
	output, result, err := RunHarness(ctx, sb, opts, input)
	if err != nil {
		return 0, err
	}
	if result.TimedOut {
		return Inconclusive, nil
	}
	if result.ExitCode != 0 {
		return NonConformant, nil
	}

	merged := make(cjtr.Bindings, len(input)+len(output))
	for k, v := range input {
		merged[k] = v
	}
	for k, v := range output {
		merged[k] = v
	}

	passed, err := cjtr.CheckExample(h, concepts, tables, bindingsToExample(merged, concepts))
	if err != nil {
		return 0, err
	}
	return VerdictFromOutcome(false, passed), nil
}

// bindingsToExample converts bindings (keyed by a concept's own
// "variable" path, cjtr.Bindings's own shape) into the concept-name-
// keyed map [cjtr.CheckExample] itself expects - the inverse of what
// that function already does internally for an example written by
// hand in a .cjtr file. Only VariableConcept entries are included: a
// value or table concept's own value never comes from a binding.
func bindingsToExample(bindings cjtr.Bindings, concepts map[string]cjtr.Concept) map[string]string {
	example := make(map[string]string, len(concepts))
	for name, c := range concepts {
		if c.Kind != cjtr.VariableConcept {
			continue
		}
		if v, ok := bindings[c.Variable]; ok {
			example[name] = v
		}
	}
	return example
}
