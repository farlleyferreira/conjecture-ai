package cjtr

import "fmt"

// Falsifiable reports whether stmt could, in principle, be
// contradicted by some concrete value drawn from its own term's
// declared domain - the spec-judge capability's "Falsifiability
// check" (openspec/specs/spec-judge/spec.md): "The judge SHALL reject
// any hypothesis that is not written in a way that could be disproven
// by a concrete case."
//
// This only detects the one case checkable from a term's own declared
// domain (its "range" or "one_of") without a general solver: a
// comparator whose threshold already holds for every value the term's
// own declaration allows is a tautology - no concrete case of that
// term could ever disagree with it, so a hypothesis built from it
// could never be disproven. For example, "the shipping fee is at
// least -1" is unfalsifiable when shipping fee's declared range
// starts at 0.00: no value the concept could ever take would violate
// it. It does not attempt the general case (reasoning across several
// concepts' statements at once, which spec-judge's own separate
// "Consistency check via solver" requirement names and which needs an
// SMT solver this codebase does not have) - a statement this function
// cannot prove tautological is reported falsifiable, not assumed
// sound by some stronger proof it never ran.
//
// A statement whose Value is a reference to another concept
// (ValueIsConcept) is always reported falsifiable: that concept's own
// value can vary at runtime, so stmt's truth cannot be pinned down
// from term's domain alone.
func Falsifiable(stmt Statement, concepts map[string]Concept) (bool, error) {
	term, ok := concepts[stmt.Term]
	if !ok {
		return false, fmt.Errorf("cjtr: statement's term %q is not a declared concept", stmt.Term)
	}
	if stmt.ValueIsConcept {
		return true, nil
	}

	if stmt.Comparator == "is" || stmt.Comparator == "equals" {
		if len(term.OneOf) == 1 && term.OneOf[0] == stmt.Value {
			return false, nil // the term can never be anything else
		}
		return true, nil
	}

	if !term.HasRange {
		return true, nil // no declared domain to check a numeric tautology against
	}
	termMin, err := parseExactDecimal(term.Min)
	if err != nil {
		return false, fmt.Errorf("cjtr: concept %q: range.min: %w", term.Name, err)
	}
	termMax, err := parseExactDecimal(term.Max)
	if err != nil {
		return false, fmt.Errorf("cjtr: concept %q: range.max: %w", term.Name, err)
	}

	switch stmt.Comparator {
	case "is at least":
		val, err := parseExactDecimal(stmt.Value)
		if err != nil {
			return false, err
		}
		return termMin.Cmp(val) < 0, nil // tautology (not falsifiable) when termMin >= val
	case "is at most":
		val, err := parseExactDecimal(stmt.Value)
		if err != nil {
			return false, err
		}
		return termMax.Cmp(val) > 0, nil // tautology when termMax <= val
	case "is greater than":
		val, err := parseExactDecimal(stmt.Value)
		if err != nil {
			return false, err
		}
		return termMin.Cmp(val) <= 0, nil // tautology when termMin > val
	case "is less than":
		val, err := parseExactDecimal(stmt.Value)
		if err != nil {
			return false, err
		}
		return termMax.Cmp(val) >= 0, nil // tautology when termMax < val
	case "is between":
		if stmt.Value2IsConcept {
			return true, nil
		}
		lower, err := parseExactDecimal(stmt.Value)
		if err != nil {
			return false, err
		}
		upper, err := parseExactDecimal(stmt.Value2)
		if err != nil {
			return false, err
		}
		return !(termMin.Cmp(lower) >= 0 && termMax.Cmp(upper) <= 0), nil
	case "is strictly between":
		if stmt.Value2IsConcept {
			return true, nil
		}
		lower, err := parseExactDecimal(stmt.Value)
		if err != nil {
			return false, err
		}
		upper, err := parseExactDecimal(stmt.Value2)
		if err != nil {
			return false, err
		}
		return !(termMin.Cmp(lower) > 0 && termMax.Cmp(upper) < 0), nil
	default:
		return true, nil // not a comparator this check knows how to prove tautological
	}
}
