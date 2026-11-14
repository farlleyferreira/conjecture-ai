package cjtr

import (
	"fmt"
	"math/big"
	"strings"
)

// RandomValue draws one value for c uniformly from its own declared
// domain: one of its one_of values, or a decimal inside its range at
// the granularity its declared places give (or, with no places
// declared, the finest granularity range.min/range.max are themselves
// written at).
//
// pick is the randomness: called with n > 0, it returns a uniform
// integer in [0, n). It is a parameter, not a *rand.Rand, so a caller
// chooses and records the seed, and a test can supply a fixed
// sequence.
//
// ok is false when c has no declared domain to draw from - a text
// concept with no one_of, or a decimal with no range. That is not an
// error: it is the honest answer that nothing here says which values
// are possible, so no draw could claim to be a sample of them.
func RandomValue(c Concept, pick func(n *big.Int) *big.Int) (value string, ok bool, err error) {
	if len(c.OneOf) > 0 {
		i := pick(big.NewInt(int64(len(c.OneOf))))
		return c.OneOf[i.Int64()], true, nil
	}
	if !c.HasRange {
		return "", false, nil
	}

	min, err := parseExactDecimal(c.Min)
	if err != nil {
		return "", false, fmt.Errorf("cjtr: concept %q: range.min: %w", c.Name, err)
	}
	max, err := parseExactDecimal(c.Max)
	if err != nil {
		return "", false, fmt.Errorf("cjtr: concept %q: range.max: %w", c.Name, err)
	}
	if min.Cmp(max) > 0 {
		return "", false, fmt.Errorf("cjtr: concept %q: range.min is above range.max", c.Name)
	}

	places := c.Places
	if !c.HasPlaces {
		places = decimalPlaces(c.Min)
		if p := decimalPlaces(c.Max); p > places {
			places = p
		}
	}
	step := unitStep(places)

	// The number of representable values in [min, max] at this step.
	span := new(big.Rat).Sub(max, min)
	steps := new(big.Rat).Quo(span, step)
	count := new(big.Int).Quo(steps.Num(), steps.Denom())
	count.Add(count, big.NewInt(1))

	k := pick(count)
	offset := new(big.Rat).Mul(new(big.Rat).SetInt(k), step)
	return formatDecimal(new(big.Rat).Add(min, offset), places), true, nil
}

// decimalPlaces counts the digits after the decimal point in a
// decimal literal as written ("24.00" has 2, "24" has 0).
func decimalPlaces(s string) int {
	_, frac, found := strings.Cut(strings.TrimSpace(s), ".")
	if !found {
		return 0
	}
	return len(frac)
}
