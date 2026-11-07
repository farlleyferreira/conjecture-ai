// directedsampling.go implements the directed-sampling half of "Three
// levels of universal ('for all') rigor" for real, against a running
// implementation: generating candidate inputs from a hypothesis's own
// boundary data, and checking every one of them, so [MinimalCounterexample]
// has an actual multi-input run to draw from instead of only ever
// seeing one caller-supplied input (what [CheckHypothesisAgainstImplementation]
// alone provides).
package conformancecycle

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/farlleyferreira/conjecture-ai/internal/cjtr"
	"github.com/farlleyferreira/conjecture-ai/internal/sandbox"
)

// DirectedSamplingInputs generates one [cjtr.Bindings] candidate per
// directed-sampling test case for h - the rigor-level table's own
// wording, "centenas de entradas do domínio, concentradas em
// fronteiras, partições..." (hundreds of domain inputs, concentrated
// on boundaries and partitions). Starting from a baseline - h's own
// first worked example, the one concrete row the specification already
// requires a hypothesis to carry - it varies exactly one input-
// variable concept at a time across its own [cjtr.BoundaryValues],
// holding every other concept at the baseline.
//
// This does not attempt a cross-product of every input concept's own
// boundaries together ("pares das relações de consequência" -
// consequence-relation pairs): the same as [cjtr.BoundaryValues]'s own
// doc comment already says of its single-concept scope, the
// specification gives no concrete rule for combining more than one
// concept's boundaries into a single test case, so this does not guess
// at one.
func DirectedSamplingInputs(h cjtr.Hypothesis, concepts map[string]cjtr.Concept) ([]cjtr.Bindings, error) {
	if len(h.Examples) == 0 {
		return nil, fmt.Errorf("conformancecycle: directed sampling needs at least one worked example in %q to use as its baseline", h.Name)
	}
	baseline, err := cjtr.BindingsFromExample(h.Examples[0], concepts)
	if err != nil {
		return nil, fmt.Errorf("conformancecycle: resolving %q's baseline example: %w", h.Name, err)
	}

	names := make([]string, 0, len(concepts))
	for name := range concepts {
		names = append(names, name)
	}
	sort.Strings(names) // deterministic case order, regardless of map iteration order

	var cases []cjtr.Bindings
	for _, name := range names {
		c := concepts[name]
		if c.Kind != cjtr.VariableConcept || !strings.HasPrefix(c.Variable, "input.") {
			continue
		}
		values, err := cjtr.BoundaryValues(c)
		if err != nil {
			return nil, fmt.Errorf("conformancecycle: generating boundary values for %q: %w", name, err)
		}
		for _, v := range values {
			variant := make(cjtr.Bindings, len(baseline))
			for k, bv := range baseline {
				variant[k] = bv
			}
			variant[c.Variable] = v
			cases = append(cases, variant)
		}
	}
	return cases, nil
}

// CheckHypothesisDirectedSampling runs h's full directed-sampling check
// against a real implementation: one round per [DirectedSamplingInputs]
// candidate. It does not stop at the first NonConformant round -
// "Minimal counterexample returned on violation" ("WHEN the
// conformance cycle finds multiple failing inputs for the same
// violation THEN it returns the smallest/simplest one it found")
// presumes the cycle keeps sampling to actually have more than one to
// compare - so every candidate runs, and [MinimalCounterexample] picks
// the smallest failure found, by total serialized length of its input
// bindings: [Counterexample]'s own doc comment names "serialized
// length" as one acceptable reading of "simplest" when the
// specification gives no concrete metric, which is the choice made
// here.
//
// An Inconclusive round (a sandboxed timeout) decides nothing on its
// own - it is neither a pass nor a counterexample - but it is not
// silently ignored either: if no round is ever NonConformant and at
// least one round was Inconclusive, the whole check reports
// Inconclusive, since a hypothesis is not demonstrated conformant when
// part of its own directed-sampling domain was never actually
// exercised to a definitive result.
func CheckHypothesisDirectedSampling(
	ctx context.Context,
	sb *sandbox.Sandbox,
	opts sandbox.Options,
	h cjtr.Hypothesis,
	concepts map[string]cjtr.Concept,
	tables cjtr.Tables,
) (Verdict, *Counterexample, error) {
	cases, err := DirectedSamplingInputs(h, concepts)
	if err != nil {
		return 0, nil, err
	}

	var failures []Counterexample
	anyInconclusive := false
	for _, input := range cases {
		verdict, err := CheckHypothesisAgainstImplementation(ctx, sb, opts, h, concepts, tables, input)
		if err != nil {
			return 0, nil, err
		}
		switch verdict {
		case NonConformant:
			failures = append(failures, Counterexample{Size: bindingsSerializedLength(input), Value: input})
		case Inconclusive:
			anyInconclusive = true
		}
	}

	if best, found := MinimalCounterexample(failures); found {
		return NonConformant, &best, nil
	}
	if anyInconclusive {
		return Inconclusive, nil, nil
	}
	return Conformant, nil, nil
}

// bindingsSerializedLength sums the length of every key and value in
// b - this function's own chosen "serialized length" measure for
// [Counterexample.Size] (see [CheckHypothesisDirectedSampling]'s doc
// comment).
func bindingsSerializedLength(b cjtr.Bindings) int {
	total := 0
	for k, v := range b {
		total += len(k) + len(v)
	}
	return total
}
