// Package omissiondetection implements pieces of the omission-
// detection capability (openspec/specs/omission-detection/spec.md):
// the four classes of specification omission conjecture looks for.
// Type D ("requirement nobody thought of", typed.go), Type A
// ("asserted but not implemented", typea.go - conformance-cycle's own
// directed-sampling run now provides real implementation evidence to
// check against), and Type B ("case forgotten within the domain",
// typeb.go - reusing conformance-cycle's own directed-sampling
// boundary generation as its property-based test generator) are
// implemented. Type C still needs machinery this codebase does not
// have yet: a decision-table/state-event-matrix grammar.
package omissiondetection

// Catalog is a configurable list of requirement categories to check
// for - Type D's own wording: "a configurable catalog of requirement
// categories (for example, concurrency, localization, accessibility)."
type Catalog []string

// TypeDOmission is one catalog category found with no rule addressing
// it in a given area.
type TypeDOmission struct {
	Area     string
	Category string
}

// CheckTypeD reports every category in catalog that area's own
// addressedCategories does not mark true - "The system SHALL... check
// whether each category is addressed by at least one rule in the
// relevant area." addressedCategories is supplied by the caller, not
// determined here: nothing in the cjtr-language grammar this codebase
// parses lets a rule declare which requirement category it addresses,
// so this package cannot discover that on its own; it only applies the
// catalog once the caller already knows.
//
// A category never named in catalog is never checked and never
// reported either way, by construction - "a category outside this
// catalog cannot be detected" / "the system does not claim to have
// checked for it, and does not report its absence as 'no omissions
// found.'" An empty result from CheckTypeD therefore only means "no
// configured category was found missing," never "no omissions exist":
// a caller presenting this result must carry that distinction forward,
// not collapse it into an unqualified "no omissions found."
func CheckTypeD(catalog Catalog, area string, addressedCategories map[string]bool) []TypeDOmission {
	var omissions []TypeDOmission
	for _, category := range catalog {
		if !addressedCategories[category] {
			omissions = append(omissions, TypeDOmission{Area: area, Category: category})
		}
	}
	return omissions
}
