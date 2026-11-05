// Package teamconfig implements the team-config capability's core rule
// (openspec/specs/team-config/spec.md): a setting in the shared,
// git-tracked settings.cjtr sets a floor, never a ceiling, for an
// individual contributor's local configuration - "A local configuration
// SHALL be allowed to exceed it, but SHALL NOT be allowed to fall below
// it."
//
// This package only implements that comparison, as a generic primitive
// over whatever ordered value a setting happens to be. It does not parse
// settings.cjtr itself: the specification gives no worked example of
// that file's content, so this does not invent a concrete settings
// schema to go with the comparison rule. It also does not implement
// settings.cjtr's signature or local-compilation requirements, both of
// which depend on a general .cjtr signing/compilation mechanism this
// codebase has not built for any .cjtr file yet, team settings included.
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
