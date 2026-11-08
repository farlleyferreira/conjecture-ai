package cjtr

import (
	"fmt"
	"math/big"
)

// BoundaryValues returns candidate values for a concept's declared
// domain, concentrated on its boundaries and partitions - the input-value
// half of the conformance cycle's "directed sampling" level (the
// specification's rigor-level table: "Centenas de entradas do domínio,
// concentradas em fronteiras, partições, controles negativos e pares das
// relações de consequência" - hundreds of domain inputs, concentrated on
// boundaries, partitions, negative controls, and consequence-relation
// pairs).
//
// This only generates values; it does not run anything. The rest of
// directed sampling - calling a contract's declared observation
// operation with these inputs (cli-mcp-integration territory) inside a
// sandbox (hardware-scheduler territory) and checking the result against
// a hypothesis's predicate (already possible today via [Evaluate]) - is
// not implemented here. Neither are negative controls or
// consequence-relation pairs: both need a hypothesis's full given/then
// shape, not one concept's domain in isolation, and the specification
// gives no concrete rule for constructing either beyond their name, so
// this does not guess at one.
//
// A concept declaring "one_of" returns every declared value: a finite
// domain's partitions are already exhaustive. A concept declaring
// "range" returns its Min and Max, plus the value one declared "places"
// unit inside each end (e.g. range 0.00..100000.00 with places 2 yields
// 0.00, 0.01, 99999.99, 100000.00) - the inside-the-boundary values are
// skipped when "places" isn't declared, since there would be no way to
// pick a step size without guessing one. A concept declaring neither
// returns (nil, nil): nothing here invents a domain it was not given.
func BoundaryValues(c Concept) ([]string, error) {
	if len(c.OneOf) > 0 {
		return append([]string(nil), c.OneOf...), nil
	}

	if !c.HasRange {
		return nil, nil
	}

	values := []string{c.Min, c.Max}

	if c.HasPlaces {
		min, err := parseExactDecimal(c.Min)
		if err != nil {
			return nil, fmt.Errorf("cjtr: concept %q: range.min: %w", c.Name, err)
		}
		max, err := parseExactDecimal(c.Max)
		if err != nil {
			return nil, fmt.Errorf("cjtr: concept %q: range.max: %w", c.Name, err)
		}
		step := unitStep(c.Places)

		if justAboveMin := new(big.Rat).Add(min, step); justAboveMin.Cmp(max) < 0 {
			values = append(values, formatDecimal(justAboveMin, c.Places))
		}
		if justBelowMax := new(big.Rat).Sub(max, step); justBelowMax.Cmp(min) > 0 {
			values = append(values, formatDecimal(justBelowMax, c.Places))
		}
	}

	return values, nil
}

// ExhaustiveValues returns every value of c's own declared domain, when
// that domain is finite - conformance-cycle's "bounded proof"/"complete
// proof" rigor levels' own input: full coverage of a concept's domain,
// not [BoundaryValues]'s own sample of its edges.
//
// A "one_of" concept is already finite: every declared value is
// returned. A "range" concept is only finite when it also declares
// "places" - together they give an exact step count
// ((Max-Min)/10^-Places + 1) the specification author already committed
// to; a range without "places" has no declared step, so there is no
// grounded way to enumerate it without inventing a precision the
// specification never gave this concept, and found is false. A concept
// declaring neither is likewise not boundable.
//
// maxCount caps how many values this function will ever actually
// enumerate: a wide range at a fine precision could still mean millions
// of values, which this package will not silently produce - exceeding
// maxCount reports found=false, the same as a genuinely unbounded
// domain, rather than a value this caller did not ask for.
func ExhaustiveValues(c Concept, maxCount int) (values []string, found bool, err error) {
	if len(c.OneOf) > 0 {
		if len(c.OneOf) > maxCount {
			return nil, false, nil
		}
		return append([]string(nil), c.OneOf...), true, nil
	}

	if !c.HasRange || !c.HasPlaces {
		return nil, false, nil
	}

	min, err := parseExactDecimal(c.Min)
	if err != nil {
		return nil, false, fmt.Errorf("cjtr: concept %q: range.min: %w", c.Name, err)
	}
	max, err := parseExactDecimal(c.Max)
	if err != nil {
		return nil, false, fmt.Errorf("cjtr: concept %q: range.max: %w", c.Name, err)
	}
	step := unitStep(c.Places)

	span := new(big.Rat).Sub(max, min)
	if span.Sign() < 0 {
		return nil, false, fmt.Errorf("cjtr: concept %q: range.min is greater than range.max", c.Name)
	}
	countRat := new(big.Rat).Quo(span, step)
	if !countRat.IsInt() {
		return nil, false, fmt.Errorf("cjtr: concept %q: range is not an exact multiple of its own declared precision", c.Name)
	}
	// +1: both endpoints are inclusive.
	countPlusOne := new(big.Int).Add(countRat.Num(), big.NewInt(1))
	if !countPlusOne.IsInt64() || countPlusOne.Int64() > int64(maxCount) {
		return nil, false, nil
	}
	count := countPlusOne.Int64()

	values = make([]string, 0, count)
	v := new(big.Rat).Set(min)
	for i := int64(0); i < count; i++ {
		values = append(values, formatDecimal(v, c.Places))
		v = new(big.Rat).Add(v, step)
	}
	return values, true, nil
}

// unitStep returns 10^-places as an exact rational, e.g. places=2 gives
// 1/100 (0.01) - the smallest value a decimal concept declaring that
// many "places" can represent.
func unitStep(places int) *big.Rat {
	denom := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(places)), nil)
	return new(big.Rat).SetFrac(big.NewInt(1), denom)
}

// formatDecimal renders r as exact decimal text with exactly places
// digits after the point. Every value this package builds with
// [unitStep] is already exact at that precision, so FloatString never
// actually rounds here.
func formatDecimal(r *big.Rat, places int) string {
	return r.FloatString(places)
}
