package optimizationsuite

import "testing"

// thresholdBehavior mimics a hypothesis whose branch flips at
// x >= threshold - the realistic case this generator targets (finding
// an input that crosses a comparator's own boundary).
func thresholdBehavior(threshold float64) func(Individual) bool {
	return func(ind Individual) bool { return ind[0] >= threshold }
}

func TestEvolveForBehaviorChange_FindsCandidateAcrossThreshold(t *testing.T) {
	behavior := thresholdBehavior(100)
	seed := Individual{50}

	fitness := func(ind Individual) float64 {
		d := ind[0] - 100
		if d < 0 {
			d = -d
		}
		return -d // higher (less negative) the closer ind[0] is to 100
	}
	changed := func(seed, candidate Individual) bool {
		return behavior(seed) != behavior(candidate)
	}

	best, found := EvolveForBehaviorChange(seed, fitness, changed, 32, 20)

	if !found {
		t.Fatalf("EvolveForBehaviorChange did not find a behavior change; best = %v", best)
	}
	if behavior(seed) == behavior(best) {
		t.Errorf("best = %v has the same behavior as seed = %v, want different", best, seed)
	}
}

func TestEvolveForBehaviorChange_NoImprovementShrinksStepAndGivesUp(t *testing.T) {
	seed := Individual{0}
	flatFitness := func(Individual) float64 { return 0 } // no gradient anywhere
	neverChanges := func(seed, candidate Individual) bool { return false }

	best, found := EvolveForBehaviorChange(seed, flatFitness, neverChanges, 1, 10)

	if found {
		t.Error("EvolveForBehaviorChange reported found=true with a Changed function that always returns false")
	}
	if best[0] != 0 {
		t.Errorf("best = %v, want seed unchanged (%v) - nothing ever improved on it", best, seed)
	}
}

func TestEvolveForBehaviorChange_DeterministicAcrossRuns(t *testing.T) {
	behavior := thresholdBehavior(100)
	seed := Individual{50}
	fitness := func(ind Individual) float64 {
		d := ind[0] - 100
		if d < 0 {
			d = -d
		}
		return -d
	}
	changed := func(seed, candidate Individual) bool { return behavior(seed) != behavior(candidate) }

	best1, found1 := EvolveForBehaviorChange(seed, fitness, changed, 32, 20)
	best2, found2 := EvolveForBehaviorChange(seed, fitness, changed, 32, 20)

	if found1 != found2 || best1[0] != best2[0] {
		t.Errorf("two runs with identical inputs diverged: (%v, %v) vs (%v, %v)", best1, found1, best2, found2)
	}
}

func TestEvolveForBehaviorChange_MultiDimensionalGenesMutatedCoordinateWise(t *testing.T) {
	// Behavior flips once EITHER gene crosses its own threshold -
	// exercises that every gene gets its own +/- mutant each
	// generation, not just the first.
	seed := Individual{0, 0}
	behavior := func(ind Individual) bool { return ind[0] >= 10 || ind[1] >= 10 }
	fitness := func(ind Individual) float64 {
		// Reward whichever gene is closer to its own threshold.
		best := ind[0]
		if ind[1] > best {
			best = ind[1]
		}
		return best
	}
	changed := func(seed, candidate Individual) bool { return behavior(seed) != behavior(candidate) }

	best, found := EvolveForBehaviorChange(seed, fitness, changed, 8, 10)

	if !found {
		t.Fatalf("EvolveForBehaviorChange did not find a behavior change; best = %v", best)
	}
	if !behavior(best) {
		t.Errorf("best = %v does not actually trigger the changed behavior", best)
	}
}

func TestEvolveForBehaviorChange_SeedItselfIsNotMutated(t *testing.T) {
	seed := Individual{50}
	behavior := thresholdBehavior(100)
	fitness := func(ind Individual) float64 {
		d := ind[0] - 100
		if d < 0 {
			d = -d
		}
		return -d
	}
	changed := func(seed, candidate Individual) bool { return behavior(seed) != behavior(candidate) }

	_, _ = EvolveForBehaviorChange(seed, fitness, changed, 32, 20)

	if seed[0] != 50 {
		t.Errorf("seed was mutated in place: seed = %v, want unchanged [50]", seed)
	}
}
