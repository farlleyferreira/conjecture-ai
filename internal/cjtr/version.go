// Package cjtr loads and interprets .cjtr files: the conjecture-ai
// declarative specification format. It builds on [yamlstrict] for parsing
// and adds .cjtr-specific semantics (format versioning, composite
// identifiers, vocabulary and grammar, and so on, added incrementally).
package cjtr

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/farlleyferreira/conjecture-ai/internal/yamlstrict"
)

// SupportedMajor is the highest .cjtr format MAJOR version this build
// understands. A file declaring any other MAJOR version is rejected: a
// higher MAJOR may depend on guarantees this build does not implement, and
// a lower MAJOR may have since had a breaking change made to it.
const SupportedMajor = 1

// Version is a .cjtr file's declared format version
// (`conjecture: MAJOR.MINOR.PATCH`). MAJOR changes alter which guarantees,
// verdicts, or decisions the file depends on; MINOR is additive and
// non-breaking; PATCH is a wording-only correction.
type Version struct {
	Major, Minor, Patch int
}

func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// ParseVersion parses a "MAJOR.MINOR.PATCH" string.
func ParseVersion(s string) (Version, error) {
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return Version{}, fmt.Errorf("cjtr: invalid version %q: want MAJOR.MINOR.PATCH", s)
	}

	nums := make([]int, 3)
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return Version{}, fmt.Errorf("cjtr: invalid version %q: component %q is not a non-negative integer", s, p)
		}
		nums[i] = n
	}

	return Version{Major: nums[0], Minor: nums[1], Patch: nums[2]}, nil
}

// CheckSupported reports an error if v's MAJOR version is not one this
// build understands, rather than guessing at compatibility.
func CheckSupported(v Version) error {
	if v.Major != SupportedMajor {
		return fmt.Errorf(
			"cjtr: format version %s is not supported (this build understands MAJOR version %d)",
			v, SupportedMajor,
		)
	}
	return nil
}

// FormatVersion reads and parses the top-level `conjecture:` key of a
// strictly-loaded .cjtr document.
func FormatVersion(root *yamlstrict.Node) (Version, error) {
	node, ok := root.Get("conjecture")
	if !ok {
		return Version{}, fmt.Errorf("cjtr: missing required top-level \"conjecture\" version key")
	}
	if node.Kind != yamlstrict.ScalarKind {
		return Version{}, fmt.Errorf("cjtr: \"conjecture\" key must be a scalar version string, got %v", node.Kind)
	}
	return ParseVersion(node.Value)
}
