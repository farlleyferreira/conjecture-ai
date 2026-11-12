// Package teamconfig implements the team-config capability's core rule
// (openspec/specs/team-config/spec.md): a setting in the shared,
// git-tracked settings.cjtr sets a floor, never a ceiling, for an
// individual contributor's local configuration - "A local configuration
// SHALL be allowed to exceed it, but SHALL NOT be allowed to fall below
// it."
//
// This package only implements that comparison, as a generic primitive
// over whatever ordered value a setting happens to be. It does not
// parse settings.cjtr itself, or apply its own signature/local-
// compilation requirements - that is cmd/conjecture/settings.go's own
// concern, reusing the same cjtr-wide signing pipeline (internal/cjtr/
// signing.go) every other signed .cjtr file already goes through.
// This package stays a generic comparison primitive, deliberately
// independent of any one concrete settings schema.
package teamconfig

import "fmt"

// Direction says which end of a setting's ordering counts as stricter.
// Higher is for a setting where a bigger value is stricter (e.g. a
// minimum required case count, a minimum risk weight); Lower is for one
// where a smaller value is stricter (e.g. a maximum tolerated violation
// rate, a maximum allowed timeout).
type Direction int

const (
	Higher Direction = iota
	Lower
)

func (d Direction) String() string {
	switch d {
	case Higher:
		return "higher"
	case Lower:
		return "lower"
	default:
		return "unknown direction"
	}
}

// orderedByUnderlying is any type whose comparison operators order it
// meaningfully as its own underlying numeric type - plain numeric types
// directly, and any named type defined over one (such as this
// project's own evidenceledger.Weight) by virtue of sharing it.
type orderedByUnderlying interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64
}

// MeetsFloor reports whether local meets or exceeds the team's required
// floor in direction - "floor, not ceiling": a local value that does
// not satisfy this is weaker than the team requires and must be
// rejected (see [CheckFloor]); one that does, whether equal to or
// stricter than floor, is accepted.
func MeetsFloor[T orderedByUnderlying](local, floor T, direction Direction) bool {
	if direction == Higher {
		return local >= floor
	}
	return local <= floor
}

// CheckFloor returns nil when local meets name's required floor (see
// [MeetsFloor]), or a descriptive error - the specification's "Local
// override below the floor rejected" scenario - otherwise.
func CheckFloor[T orderedByUnderlying](name string, local, floor T, direction Direction) error {
	if MeetsFloor(local, floor, direction) {
		return nil
	}
	word := "at least"
	if direction == Lower {
		word = "at most"
	}
	return fmt.Errorf("teamconfig: %q is weaker than the team's required floor: local value %v, want %s %v", name, local, word, floor)
}
