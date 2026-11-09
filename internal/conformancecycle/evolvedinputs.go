// evolvedinputs.go wires optimization-suite's own evolutionary input
// generator (optimizationsuite.EvolveForBehaviorChange) into
// conformance-cycle: a case-generation source for exactly the one gap
// cjtr.BoundaryValues and cjtr.ExhaustiveValues both leave untouched -
// an input-variable concept declaring neither a one_of list nor a
// range, where this package's other generators (DirectedSamplingInputs,
// BoundedProofInputs, CompleteProofInputs) all contribute nothing at
// all for that concept.
package conformancecycle

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/farlleyferreira/conjecture-ai/internal/cjtr"
	"github.com/farlleyferreira/conjecture-ai/internal/optimizationsuite"
)

// evolveStepSize and evolveMaxGenerations are this package's own
// disclosed defaults for EvolveForBehaviorChange's own two tuning
// parameters - the specification names no concrete step size or
// generation count for the evolutionary input generator, so these are
// this project's own chosen values (the same "disclosed default, not
// a value the specification gives" posture every other fixed constant
// in this codebase already takes), not something a caller can
// override yet.
const (
	evolveStepSize       = 1.0
	evolveMaxGenerations = 20
)

// EvolvedInputs generates at most one [cjtr.Bindings] candidate per
// input-variable concept that [cjtr.BoundaryValues] and
// [cjtr.ExhaustiveValues] both have nothing to offer for - a decimal
// concept declaring neither "one_of" nor "range" - by searching from
// h's own baseline worked example for a value that flips whether h's
// own "given" conditions hold, via optimization-suite's deterministic
// evolutionary search.
//
// This only ever applies to a non-universal hypothesis (h.Given non-
// empty): a universal hypothesis's "always" clause has no given-
// condition branch to search for a crossing of, so this returns no
// cases for one, the same honest "nothing to invent" posture
// [DirectedSamplingInputs] already takes for a concept with no
// declared domain. A "text"-typed concept is skipped too - there is
// no continuous space for an evolutionary search over a bareword
// vocabulary to explore.
//
// found reports whether at least one concept actually got an evolved
// case - a hypothesis where every undeclared-domain concept is text,
// or where none exist at all, contributes nothing, which is not the
// same as this search having tried and failed to find a crossing.
func EvolvedInputs(h cjtr.Hypothesis, concepts map[string]cjtr.Concept, tables cjtr.Tables) (cases []cjtr.Bindings, found bool, err error) {
	if h.IsUniversal || len(h.Given) == 0 {
		return nil, false, nil
	}
	if len(h.Examples) == 0 {
		return nil, false, fmt.Errorf("conformancecycle: evolved inputs need at least one worked example in %q to use as its baseline", h.Name)
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
		if c.Type != "decimal" || c.HasRange || len(c.OneOf) > 0 {
			continue // already handled by BoundaryValues/ExhaustiveValues, or not a continuous value
		}

		seedText, ok := baseline[c.Variable]
		if !ok {
			continue
		}
		seedValue, err := strconv.ParseFloat(seedText, 64)
		if err != nil {
			return nil, false, fmt.Errorf("conformancecycle: concept %q: baseline value %q is not a decimal: %w", name, seedText, err)
		}

		value, ok, err := evolveValueFor(h, c, baseline, seedValue, concepts, tables)
		if err != nil {
			return nil, false, err
		}
		if !ok {
			continue
		}
		found = true
		variant := make(cjtr.Bindings, len(baseline))
		for k, v := range baseline {
			variant[k] = v
		}
		variant[c.Variable] = formatEvolvedValue(value)
		cases = append(cases, variant)
	}
	return cases, found, nil
}

// evolveValueFor runs EvolveForBehaviorChange starting from seedValue,
// searching for a value of c that flips whether h's own "given"
// conditions hold against baseline.
func evolveValueFor(h cjtr.Hypothesis, c cjtr.Concept, baseline cjtr.Bindings, seedValue float64, concepts map[string]cjtr.Concept, tables cjtr.Tables) (float64, bool, error) {
	seedGiven, err := evaluateGiven(h, concepts, tables, baseline)
	if err != nil {
		return 0, false, err
	}

	var evalErr error
	changed := func(_, candidate optimizationsuite.Individual) bool {
		variant := make(cjtr.Bindings, len(baseline))
		for k, v := range baseline {
			variant[k] = v
		}
		variant[c.Variable] = formatEvolvedValue(candidate[0])
		candidateGiven, err := evaluateGiven(h, concepts, tables, variant)
		if err != nil {
			evalErr = err
			return false
		}
		return candidateGiven != seedGiven
	}

	fitness := evolveFitnessFor(h, c, seedValue)

	best, found := optimizationsuite.EvolveForBehaviorChange(
		optimizationsuite.Individual{seedValue}, fitness, changed, evolveStepSize, evolveMaxGenerations,
	)
	if evalErr != nil {
		return 0, false, evalErr
	}
	if !found {
		return 0, false, nil
	}
	return best[0], true, nil
}

// evolveFitnessFor steers the search toward whichever of h's own
// "given" statements compares c against a literal decimal value -
// that statement's own Value (or, for "is between"/"is strictly
// between", its Value, the lower bound) is the one concrete number
// this package already has grounds to treat as a likely boundary,
// since it is exactly the value [cjtr.BoundaryValues] itself would
// use if c declared a range ending there. When no such statement
// exists (c is never compared to a literal in h's own "given", only
// to another concept, or not mentioned there at all), this falls back
// to a flat fitness: the search still explores a geometric sequence
// of offsets out from seedValue, honestly reflecting that no boundary
// is known to aim for.
func evolveFitnessFor(h cjtr.Hypothesis, c cjtr.Concept, seedValue float64) optimizationsuite.Fitness {
	for _, stmt := range h.Given {
		if stmt.Term != c.Name || stmt.ValueIsConcept {
			continue
		}
		target, err := strconv.ParseFloat(stmt.Value, 64)
		if err != nil {
			continue
		}
		return func(ind optimizationsuite.Individual) float64 {
			d := ind[0] - target
			if d < 0 {
				d = -d
			}
			return -d
		}
	}
	return func(optimizationsuite.Individual) float64 { return 0 }
}

// evaluateGiven reports whether every one of h's own "given"
// statements holds against bindings - "and across given phrases is
// the only combination allowed" ([cjtr.Hypothesis.Given]'s own doc
// comment). An empty Given is vacuously true, matching how a
// universal hypothesis's own absent Given is read everywhere else in
// this codebase - though [EvolvedInputs] itself never calls this for
// one, since it returns early when Given is empty.
func evaluateGiven(h cjtr.Hypothesis, concepts map[string]cjtr.Concept, tables cjtr.Tables, bindings cjtr.Bindings) (bool, error) {
	for _, stmt := range h.Given {
		ok, err := cjtr.Evaluate(stmt, concepts, tables, bindings)
		if err != nil {
			return false, err
		}
		if !ok {
			return false, nil
		}
	}
	return true, nil
}

// formatEvolvedValue renders an evolved decimal value as text, at
// whatever precision float64 arithmetic actually produced - unlike
// [formatDecimal] elsewhere in this codebase, there is no declared
// "places" for an undeclared-domain concept to format to, so this
// does not invent one.
func formatEvolvedValue(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}
