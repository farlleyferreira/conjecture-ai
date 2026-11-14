package conformancecycle

import (
	"context"
	"fmt"
	"math/big"
	"math/rand/v2"
	"sort"
	"strings"

	"github.com/farlleyferreira/conjecture-ai/internal/cjtr"
	"github.com/farlleyferreira/conjecture-ai/internal/sandbox"
)

// HiddenSamplingInputs draws n cases for h, each one an independent
// uniform draw of EVERY input concept from that concept's own
// declared domain - unlike directed sampling, which holds a worked
// example fixed and varies one concept at a time over its boundary
// values, the same handful of cases on every run.
//
// These are the cases evidence-ledger's numeric promotion criteria
// are about ("100 failure-free cases ... at 95% confidence"): that
// arithmetic only means anything for independent draws, and running
// the same deterministic boundary cases a hundred times is one
// observation, not a hundred. They are "hidden" in the plain sense
// that nothing reading the specification can know them in advance:
// they exist only for one run, from a seed chosen at that run.
//
// sampleable is false, with reason naming the concept, when some
// input concept has no declared domain (a decimal with no range, a
// text with no one_of): there is then nothing to draw from, and the
// honest result is no sample at all rather than values invented for
// it.
func HiddenSamplingInputs(concepts map[string]cjtr.Concept, n int, seed uint64) (cases []cjtr.Bindings, sampleable bool, reason string, err error) {
	names := make([]string, 0, len(concepts))
	for name, c := range concepts {
		if c.Kind == cjtr.VariableConcept && strings.HasPrefix(c.Variable, "input.") {
			names = append(names, name)
		}
	}
	sort.Strings(names) // the same seed must always give the same cases
	if len(names) == 0 {
		return nil, false, "the specification declares no input concept", nil
	}

	rng := rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))
	pick := func(limit *big.Int) *big.Int {
		if limit.IsUint64() && limit.Uint64() > 0 {
			return new(big.Int).SetUint64(rng.Uint64N(limit.Uint64()))
		}
		// A range wider than 2^64 steps: draw enough random bits and
		// reduce. The slight modulo bias is irrelevant at that width.
		v := new(big.Int)
		for v.BitLen() < limit.BitLen()+64 {
			v.Lsh(v, 64).Or(v, new(big.Int).SetUint64(rng.Uint64()))
		}
		return v.Mod(v, limit)
	}

	for i := 0; i < n; i++ {
		input := make(cjtr.Bindings, len(names))
		for _, name := range names {
			c := concepts[name]
			value, ok, err := cjtr.RandomValue(c, pick)
			if err != nil {
				return nil, false, "", fmt.Errorf("conformancecycle: sampling %q: %w", name, err)
			}
			if !ok {
				kind := "a one_of list"
				if c.Type != "text" {
					kind = "a range"
				}
				return nil, false, fmt.Sprintf("concept %q declares no domain to draw from (give it %s)", name, kind), nil
			}
			input[c.Variable] = value
		}
		cases = append(cases, input)
	}
	return cases, true, "", nil
}

// CheckHypothesisHiddenSampling runs h against every one of cases and
// reports the same three-way verdict every other rigor level does: a
// minimal counterexample when any case fails, Inconclusive when none
// failed but at least one timed out, Conformant otherwise.
func CheckHypothesisHiddenSampling(
	ctx context.Context,
	sb *sandbox.Sandbox,
	opts sandbox.Options,
	h cjtr.Hypothesis,
	concepts map[string]cjtr.Concept,
	tables cjtr.Tables,
	cases []cjtr.Bindings,
) (Verdict, *Counterexample, error) {
	var failures []Counterexample
	anyInconclusive := false
	for _, input := range cases {
		verdict, observed, err := checkAndCapture(ctx, sb, opts, h, concepts, tables, input)
		if err != nil {
			return 0, nil, err
		}
		switch verdict {
		case NonConformant:
			failures = append(failures, Counterexample{Size: bindingsSerializedLength(observed), Value: observed})
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

// checkAndCapture is [CheckHypothesisAgainstImplementation] that also
// hands back the inputs merged with what the implementation actually
// answered, so a counterexample from a hidden case can show the real
// output. (A hidden case carries no worked example's output to fall
// back on, the way a directed case does.)
func checkAndCapture(
	ctx context.Context,
	sb *sandbox.Sandbox,
	opts sandbox.Options,
	h cjtr.Hypothesis,
	concepts map[string]cjtr.Concept,
	tables cjtr.Tables,
	input cjtr.Bindings,
) (Verdict, cjtr.Bindings, error) {
	output, result, err := RunHarness(ctx, sb, opts, input)
	if err != nil {
		return 0, nil, err
	}
	// A directed case's input still carries its baseline worked
	// example's output (see RunHarness). Whatever is handed back here
	// is shown to a person as "what happened", so that example output
	// is dropped first: when the implementation timed out or crashed
	// there is no real output to show, and showing the example's as if
	// the implementation had produced it would be false.
	sent := inputOnly(input)
	if result.TimedOut {
		return Inconclusive, sent, nil
	}
	if result.ExitCode != 0 {
		return NonConformant, sent, nil
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
		return 0, nil, err
	}
	if !passed {
		return NonConformant, merged, nil
	}
	return Conformant, merged, nil
}
