package cjtr

import (
	"fmt"
	"sort"
	"strings"
)

// Statement is a parsed .cjtr comparison of the form
// "the <term> <comparator> the <concept>" or
// "the <term> <comparator> <literal>" - the shape the specification's
// formal grammar table gives for "given", "then", "otherwise", and
// "always" (not for "when" or "steps", which reference an observation
// operation name instead and are not statements in this sense).
//
// "is between" and "is strictly between" are the vocabulary's only
// two-value comparators (their own table row spells the shape as "is
// between X and Y"), so they additionally populate Value2/Value2IsConcept;
// every other comparator leaves those two fields zero.
type Statement struct {
	// Term is the left-hand concept name, e.g. "discounted subtotal".
	Term string
	// Comparator is the fixed vocabulary phrase joining Term and Value,
	// e.g. "is greater than".
	Comparator string
	// Value is the right-hand side: either another concept's name (when
	// ValueIsConcept) or a literal exactly as written (a number, or a
	// bareword like "subscription"). For "is between"/"is strictly
	// between" this is the lower bound (X).
	Value          string
	ValueIsConcept bool
	// Value2 is the upper bound (Y), set only for "is between"/"is
	// strictly between".
	Value2          string
	Value2IsConcept bool
}

// supportedComparators are the vocabulary's comparator phrases this parser
// currently handles. The vocabulary (see vocabulary.go) also declares
// comparators this parser deliberately does not support yet: "is
// empty"/"is not empty" and "remains"/"does not change" (zero values), and
// "is one of"/"contains"/"does not contain" (value is a list or needs
// different handling) - none of those appear in the specification's one
// worked example, so this parser does not guess at how to handle them.
var supportedComparators = []string{
	"is", "equals",
	"is not", "does not equal",
	"is greater than", "is less than",
	"is at least", "is at most",
	"is between", "is strictly between",
}

// twoValueComparators are the comparators whose value is a pair "X and Y"
// rather than a single value - exactly the vocabulary entries whose own
// table row spells the phrase with two placeholders.
var twoValueComparators = map[string]bool{
	"is between":          true,
	"is strictly between": true,
}

// ParseComparison parses a .cjtr comparison statement against the given
// set of declared concepts (as returned by [Concepts]).
func ParseComparison(text string, concepts map[string]Concept) (Statement, error) {
	const article = "the "

	rest, ok := strings.CutPrefix(text, article)
	if !ok {
		return Statement{}, fmt.Errorf("cjtr: statement %q must start with %q", text, article)
	}

	term, rest, err := consumeConceptName(rest, concepts)
	if err != nil {
		return Statement{}, fmt.Errorf("cjtr: statement %q: %w", text, err)
	}

	comparator, rest, err := consumeComparator(rest)
	if err != nil {
		return Statement{}, fmt.Errorf("cjtr: statement %q: %w", text, err)
	}

	if rest == "" {
		return Statement{}, fmt.Errorf("cjtr: statement %q is missing a value after %q", text, comparator)
	}

	if twoValueComparators[comparator] {
		lower, upper, ok := strings.Cut(rest, " and ")
		if !ok {
			return Statement{}, fmt.Errorf("cjtr: statement %q: %q needs two values joined by \" and \"", text, comparator)
		}
		lowerValue, lowerIsConcept := resolveStatementSide(lower, concepts)
		upperValue, upperIsConcept := resolveStatementSide(upper, concepts)
		return Statement{
			Term: term, Comparator: comparator,
			Value: lowerValue, ValueIsConcept: lowerIsConcept,
			Value2: upperValue, Value2IsConcept: upperIsConcept,
		}, nil
	}

	value, isConcept := resolveStatementSide(rest, concepts)
	return Statement{Term: term, Comparator: comparator, Value: value, ValueIsConcept: isConcept}, nil
}

// resolveStatementSide reads one value on the right-hand side of a
// comparator: a concept reference ("the <concept>") when one matches, or
// the text as a literal otherwise.
func resolveStatementSide(s string, concepts map[string]Concept) (value string, isConcept bool) {
	const article = "the "
	if valueText, ok := strings.CutPrefix(s, article); ok {
		if _, isConcept := concepts[valueText]; isConcept {
			return valueText, true
		}
	}
	return s, false
}

// consumeConceptName finds the longest declared concept name that is a
// whole-word prefix of s, and returns it along with what follows it (with
// exactly one separating space, if any, removed).
func consumeConceptName(s string, concepts map[string]Concept) (name, rest string, err error) {
	names := make([]string, 0, len(concepts))
	for name := range concepts {
		names = append(names, name)
	}
	// Longest first, so a concept name that is itself a prefix of a longer
	// one (e.g. "shipping fee" vs. a hypothetical "shipping fee rate")
	// never wins by accident.
	sort.Slice(names, func(i, j int) bool { return len(names[i]) > len(names[j]) })

	for _, name := range names {
		if s == name {
			return name, "", nil
		}
		if rest, ok := strings.CutPrefix(s, name+" "); ok {
			return name, rest, nil
		}
	}

	return "", "", fmt.Errorf("no declared concept matches the term at %q", s)
}

// consumeComparator finds the longest comparator phrase from the FULL
// vocabulary (not just the supported subset) that is a whole-word prefix
// of s. Matching against every comparator the vocabulary knows - not only
// the ones this parser supports - matters because some supported phrases
// are themselves a prefix of an unsupported one (e.g. "is" is a prefix of
// "is empty"): matching only the supported list would silently misread
// "the X is empty" as comparator "is" with value "empty".
func consumeComparator(s string) (comparator, rest string, err error) {
	var allComparators []string
	for _, entry := range Vocabulary {
		if entry.Category == ComparatorCategory {
			allComparators = append(allComparators, entry.Phrases...)
		}
	}
	sort.Slice(allComparators, func(i, j int) bool { return len(allComparators[i]) > len(allComparators[j]) })

	for _, phrase := range allComparators {
		r, ok := strings.CutPrefix(s, phrase+" ")
		if !ok && s == phrase {
			r, ok = "", true
		}
		if !ok {
			continue
		}
		if !isSupportedComparator(phrase) {
			return "", "", fmt.Errorf("comparator %q in %q is recognized by the vocabulary but not supported by this parser yet", phrase, s)
		}
		return phrase, r, nil
	}

	return "", "", fmt.Errorf("no recognized comparator at %q", s)
}

func isSupportedComparator(phrase string) bool {
	for _, p := range supportedComparators {
		if p == phrase {
			return true
		}
	}
	return false
}
