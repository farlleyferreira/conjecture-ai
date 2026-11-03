package cjtr

import (
	"fmt"
	"regexp"
	"strings"
)

// idSeparator joins the segments of a composite identifier.
const idSeparator = "::"

// segmentPattern matches one valid identifier segment: lowercase letters,
// digits, and hyphens, same as the kebab-case names used throughout .cjtr
// files (e.g. "free-shipping", "above-threshold").
var segmentPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// ValidSegment reports whether s is a valid single segment of a composite
// identifier (one area, phenomenon, rule, or hypothesis name).
func ValidSegment(s string) bool {
	return segmentPattern.MatchString(s)
}

// ComposeID builds a composite identifier from an ordered chain of names
// actually written in the file - e.g. ComposeID("shipping",
// "free-shipping", "free-above") = "shipping::free-shipping::free-above".
// Composite identifiers are derived from content, never freestanding
// generated IDs such as UUIDs: the same chain of names always produces the
// same ID, which is what lets a rename/move be recognized by content
// instead of treated as an unrelated new node.
func ComposeID(parts ...string) (string, error) {
	if len(parts) == 0 {
		return "", fmt.Errorf("cjtr: composite ID needs at least one segment")
	}
	for _, p := range parts {
		if !ValidSegment(p) {
			return "", fmt.Errorf("cjtr: invalid ID segment %q: want lowercase letters, digits, and hyphens", p)
		}
	}
	return strings.Join(parts, idSeparator), nil
}

// HasPrefix reports whether id is scoped by prefix: either id equals
// prefix, or id starts with prefix followed by a full "::" segment
// boundary. This is what lets verification be scoped to a prefix like
// "shipping::free-shipping" without accidentally also matching an unrelated
// id that merely starts with the same characters (e.g.
// "shipping::free-shipping-extra").
func HasPrefix(id, prefix string) bool {
	if id == prefix {
		return true
	}
	return strings.HasPrefix(id, prefix+idSeparator)
}
