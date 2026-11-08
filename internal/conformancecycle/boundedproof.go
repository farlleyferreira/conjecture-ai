// boundedproof.go implements the bounded-proof rigor level: the same
// one-concept-at-a-time variation DirectedSamplingInputs already uses,
// but exhausting a boundable concept's entire finite domain instead of
// only sampling its boundary.
package conformancecycle

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/farlleyferreira/conjecture-ai/internal/cjtr"
	"github.com/farlleyferreira/conjecture-ai/internal/sandbox"
)

// maxBoundedProofValuesPerConcept caps how many values BoundedProofInputs
// will ever enumerate for a single input concept - this project's own
// disclosed limit (see openspec/changes/archive's conformance-cycle-
// rigor-levels design doc), not a count the specification gives.
// Exceeding it makes that concept count as not boundable for this
// purpose, the same posture [cjtr.ExhaustiveValues] itself takes.
const maxBoundedProofValuesPerConcept = 500

// BoundedProofInputs generates one [cjtr.Bindings] candidate per
// bounded-proof test case for h: starting from h's own first worked
// example as a baseline (the same starting point
// [DirectedSamplingInputs] uses), it varies exactly one input-variable
// concept at a time across its own [cjtr.ExhaustiveValues] - the
// concept's *entire* finite domain, not [cjtr.BoundaryValues]'s sample
// of its edges - holding every other concept at the baseline.
//
// found reports whether at least one input concept actually has a
// boundable domain. A concept with no declared domain, or a decimal
// range without a declared precision, contributes no cases - the same
// as [DirectedSamplingInputs] - but unlike that function, bounded
// proof is meaningless when *no* concept contributed anything: an
// empty case list here means this hypothesis has nothing bounded proof
// can actually strengthen, not that it passed one.
func BoundedProofInputs(h cjtr.Hypothesis, concepts map[string]cjtr.Concept) (cases []cjtr.Bindings, found bool, err error) {
	if len(h.Examples) == 0 {
		return nil, false, fmt.Errorf("conformancecycle: bounded proof needs at least one worked example in %q to use as its baseline", h.Name)
	}
	baseline, err := cjtr.BindingsFromExample(h.Examples[0], concepts)
	if err != nil {
		return nil, false, fmt.Errorf("conformancecycle: resolving %q's baseline example: %w", h.Name, err)
	}

	names := make([]string, 0, len(concepts))
	for name := range concepts {
		names = append(names, name)
	}
	sort.Strings(names) // deterministic case order, regardless of map iteration order

	for _, name := range names {
		c := concepts[name]
		if c.Kind != cjtr.VariableConcept || !strings.HasPrefix(c.Variable, "input.") {
			continue
		}
		values, ok, err := cjtr.ExhaustiveValues(c, maxBoundedProofValuesPerConcept)
		if err != nil {
			return nil, false, fmt.Errorf("conformancecycle: generating exhaustive values for %q: %w", name, err)
		}
		if !ok {
			continue
		}
		found = true
		for _, v := range values {
			variant := make(cjtr.Bindings, len(baseline))
			for k, bv := range baseline {
				variant[k] = bv
			}
			variant[c.Variable] = v
			cases = append(cases, variant)
		}
	}
	return cases, found, nil
}

// CheckHypothesisBoundedProof runs h's full bounded-proof check against
// a real implementation: one round per [BoundedProofInputs] candidate,
// aggregated into a verdict exactly like
// [CheckHypothesisDirectedSampling] (every candidate runs regardless of
// earlier failures; [MinimalCounterexample] picks the smallest failure
// found; an Inconclusive round without any failure reports Inconclusive
// overall).
//
// found reports whether [BoundedProofInputs] actually found a boundable
// concept to run against. When it is false, no round ever ran and the
// returned Verdict is meaningless - a caller selecting a rigor level
// falls back to a weaker one instead of trusting it.
func CheckHypothesisBoundedProof(
	ctx context.Context,
	sb *sandbox.Sandbox,
	opts sandbox.Options,
	h cjtr.Hypothesis,
	concepts map[string]cjtr.Concept,
	tables cjtr.Tables,
) (verdict Verdict, counterexample *Counterexample, found bool, err error) {
	cases, found, err := BoundedProofInputs(h, concepts)
	if err != nil {
		return 0, nil, false, err
	}
	if !found {
		return 0, nil, false, nil
	}

	var failures []Counterexample
	anyInconclusive := false
	for _, input := range cases {
		v, err := CheckHypothesisAgainstImplementation(ctx, sb, opts, h, concepts, tables, input)
		if err != nil {
			return 0, nil, false, err
		}
		switch v {
		case NonConformant:
			failures = append(failures, Counterexample{Size: bindingsSerializedLength(input), Value: input})
		case Inconclusive:
			anyInconclusive = true
		}
	}

	if best, ok := MinimalCounterexample(failures); ok {
		return NonConformant, &best, true, nil
	}
	if anyInconclusive {
		return Inconclusive, nil, true, nil
	}
	return Conformant, nil, true, nil
}
