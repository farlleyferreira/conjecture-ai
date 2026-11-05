package optimizationsuite

// Individual is one candidate test input: a vector of real-valued genes
// (e.g. a hypothesis's decimal-typed concept values, in whatever order
// a caller assigns them). This package has no model of what a gene
// means - that mapping (which concept each index is, how it becomes a
// cjtr.Bindings value) is a caller's job, not built here.
type Individual []float64

// Fitness scores an individual on a continuous scale a caller defines;
// EvolveForBehaviorChange climbs toward higher values. This package has
// no notion of what "better" means for the caller's domain (a
// comparator's own margin, say, as a signal of how close an input is
// to crossing it) - Fitness is entirely the caller's own knowledge,
// supplied as a plain function.
type Fitness func(Individual) float64

// Changed reports whether candidate's observable behavior differs from
// seed's own baseline behavior (e.g. seed and candidate drive a
// hypothesis's Given/Then down different branches). This, not Fitness,
// is what EvolveForBehaviorChange is actually searching for - Fitness
// only steers the search toward more promising regions; Changed decides
// success.
type Changed func(seed, candidate Individual) bool

// EvolveForBehaviorChange implements optimization-suite's evolutionary
// input generator ("The system SHALL generate test inputs using an
// evolutionary (genetic) algorithm that targets meaningful behavior
// change deterministically, without requiring the generated inputs to
// be hand-written").
//
// It is a deterministic mutate-and-select search (a (1+2n)-style
// evolution strategy: one parent, 2*len(seed) deterministic mutants
// per generation, best-mutant replacement) rather than a full genetic
// algorithm with crossover and a stochastic population - the
// specification requires the search be deterministic, which a search
// built on an unseeded random crossover/mutation step would not be,
// and a crossover operator has no obvious, non-arbitrary shape for an
// arbitrary-length real-valued gene vector the specification gives no
// further structure for. Every candidate this produces is still
// "selected to target an observable change in behavior, not chosen
// arbitrarily" (the specification's own scenario): changed is checked
// on every mutant as soon as it is generated, before fitness is even
// consulted, and the search returns the instant one is found.
//
// Each generation, every gene is perturbed by +stepSize and -stepSize
// in turn (coordinate-wise, no other combination - no randomness
// anywhere in this function). The best-scoring mutant becomes the next
// parent when it improves on the current fitness; when none does,
// stepSize is halved rather than the search giving up immediately, so
// it can still close in on a change from a wider starting step.
//
// It returns the best individual found (seed itself if nothing ever
// improved on it) and whether a behavior change was actually found
// within maxGenerations.
func EvolveForBehaviorChange(seed Individual, fitness Fitness, changed Changed, stepSize float64, maxGenerations int) (best Individual, found bool) {
	current := cloneIndividual(seed)
	currentFitness := fitness(current)
	step := stepSize

	for gen := 0; gen < maxGenerations; gen++ {
		var bestMutant Individual
		bestMutantFitness := currentFitness

		for i := range current {
			for _, sign := range [2]float64{1, -1} {
				mutant := cloneIndividual(current)
				mutant[i] += sign * step

				if changed(seed, mutant) {
					return mutant, true
				}

				if f := fitness(mutant); f > bestMutantFitness {
					bestMutant = mutant
					bestMutantFitness = f
				}
			}
		}

		if bestMutant == nil {
			step /= 2
			continue
		}
		current = bestMutant
		currentFitness = bestMutantFitness
	}

	return current, false
}

func cloneIndividual(i Individual) Individual {
	clone := make(Individual, len(i))
	copy(clone, i)
	return clone
}
