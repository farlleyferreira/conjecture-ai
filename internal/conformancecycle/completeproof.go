// completeproof.go implements the complete-proof rigor level: the full
// combinatorial cross-product of every boundable input concept's own
// exhaustive domain, rather than bounded proof's one-concept-at-a-time
// variation.
package conformancecycle

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/farlleyferreira/conjecture-ai/internal/cjtr"
	"github.com/farlleyferreira/conjecture-ai/internal/sandbox"
)

// maxCompleteProofTotalCases caps the total number of cases
// CompleteProofInputs will ever actually build - the full cross-
// product of several finite domains can grow far faster than any one
// of them alone. Exceeding it makes the whole hypothesis count as not
// boundable for complete proof, the same "disclosed cap, not an
// unbounded one" posture [maxBoundedProofValuesPerConcept] already
// takes for a single concept.
const maxCompleteProofTotalCases = 2000

// CompleteProofInputs generates one [cjtr.Bindings] candidate per
// complete-proof test case for h: the full cross-product across every
// input-variable concept's own [cjtr.ExhaustiveValues], starting from
// h's own first worked example as the baseline for any concept that is
// not itself an input (the same baseline [DirectedSamplingInputs] and
// [BoundedProofInputs] use).
//
// found reports whether h actually qualifies for complete proof: every
// one of its input concepts must have a boundable domain, and the
// total case count (the product of each concept's own value count)
// must stay within [maxCompleteProofTotalCases]. If even one input
// concept is not boundable, or the cross-product would be too large,
// found is false and cases is nil - complete proof does not claim a
// completeness it did not actually cover (design decision 2), rather
// than silently proving less than "complete" implies.
//
// A hypothesis with no input-variable concepts at all has nothing to
// cross - found is false, the same degenerate case
// [BoundedProofInputs] treats as not found.
func CompleteProofInputs(h cjtr.Hypothesis, concepts map[string]cjtr.Concept) (cases []cjtr.Bindings, found bool, err error) {
	if len(h.Examples) == 0 {
		return nil, false, fmt.Errorf("conformancecycle: complete proof needs at least one worked example in %q to use as its baseline", h.Name)
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

	type inputDomain struct {
		variable string
		values   []string
	}
	var domains []inputDomain
	total := 1
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
			return nil, false, nil
		}
		domains = append(domains, inputDomain{variable: c.Variable, values: values})
		total *= len(values)
		if total > maxCompleteProofTotalCases {
			return nil, false, nil
		}
	}
	if len(domains) == 0 {
		return nil, false, nil
	}

	cases = []cjtr.Bindings{cloneBindings(baseline)}
	for _, d := range domains {
		next := make([]cjtr.Bindings, 0, len(cases)*len(d.values))
		for _, existing := range cases {
			for _, v := range d.values {
				variant := cloneBindings(existing)
				variant[d.variable] = v
				next = append(next, variant)
			}
		}
		cases = next
	}
	return cases, true, nil
}

// cloneBindings returns a shallow copy of b, so building the cross-
// product in [CompleteProofInputs] never mutates a binding set another
// in-progress case still shares.
func cloneBindings(b cjtr.Bindings) cjtr.Bindings {
	clone := make(cjtr.Bindings, len(b))
	for k, v := range b {
		clone[k] = v
	}
	return clone
}

// CheckHypothesisCompleteProof runs h's full complete-proof check
// against a real implementation: one round per [CompleteProofInputs]
// candidate, aggregated into a verdict exactly like
// [CheckHypothesisDirectedSampling] and [CheckHypothesisBoundedProof]
// (every candidate runs regardless of earlier failures;
// [MinimalCounterexample] picks the smallest failure found; an
// Inconclusive round without any failure reports Inconclusive
// overall).
//
// found reports whether [CompleteProofInputs] actually qualified h for
// complete proof. When it is false, no round ever ran and the returned
// Verdict is meaningless - a caller selecting a rigor level falls back
// to a weaker one instead of trusting it.
func CheckHypothesisCompleteProof(
	ctx context.Context,
	sb *sandbox.Sandbox,
	opts sandbox.Options,
	h cjtr.Hypothesis,
	concepts map[string]cjtr.Concept,
	tables cjtr.Tables,
) (verdict Verdict, counterexample *Counterexample, found bool, err error) {
	cases, found, err := CompleteProofInputs(h, concepts)
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
