package conformancecycle

// Counterexample is one input that violated a rule, together with
// Size: a caller-supplied measure of how small or simple it is. The
// specification's own wording for comparing them is "the
// smallest/simplest," with no concrete metric given - serialized
// length, number of bound variables, anything the caller's own domain
// makes "simplest" mean - so this package takes that measure as given
// rather than inventing one of its own.
type Counterexample struct {
	Size  int
	Value any
}

// MinimalCounterexample returns the smallest of found, by Size -
// "Minimal counterexample returned on violation": "WHEN the
// conformance cycle finds multiple failing inputs for the same
// violation THEN it returns the smallest/simplest one it found as the
// counterexample." On a tie, the first one found (earliest in found)
// is kept, making this deterministic regardless of which equally-small
// counterexample a caller happens to list first.
//
// ok is false when found is empty - there is no counterexample to
// return because none was found, not because this function failed.
func MinimalCounterexample(found []Counterexample) (_ Counterexample, ok bool) {
	if len(found) == 0 {
		return Counterexample{}, false
	}

	best := found[0]
	for _, c := range found[1:] {
		if c.Size < best.Size {
			best = c
		}
	}
	return best, true
}
