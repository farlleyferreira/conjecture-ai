package cjtr

import (
	"fmt"
	"strings"
)

// TranslateComparator returns the exact Portuguese reading [Vocabulary]
// declares for one specific comparator phrase. A vocabulary entry that
// groups several synonymous phrases under one Phrases list holds its
// PortugueseReading as one comma-separated reading per phrase, in the
// same order as Phrases (true of every comparator entry's own data, not
// assumed) - this reads that existing structure out rather than adding
// any translation the project's single controlled-vocabulary source
// does not already carry.
func TranslateComparator(phrase string) (string, error) {
	entry, ok := LookupPhrase(phrase)
	if !ok || entry.Category != ComparatorCategory {
		return "", fmt.Errorf("cjtr: %q is not a known comparator phrase", phrase)
	}
	if len(entry.Phrases) == 1 {
		return entry.PortugueseReading, nil
	}

	readings := strings.Split(entry.PortugueseReading, ", ")
	if len(readings) != len(entry.Phrases) {
		return "", fmt.Errorf("cjtr: vocabulary entry for %q has %d phrases but %d Portuguese readings",
			phrase, len(entry.Phrases), len(readings))
	}
	for i, p := range entry.Phrases {
		if p == phrase {
			return readings[i], nil
		}
	}
	return "", fmt.Errorf("cjtr: %q not found among its own vocabulary entry's phrases", phrase)
}

// GeneratePTComment renders stmt as a deterministic "# pt:" comment -
// spec-authoring's "Bilingual comments carry deterministic translation"
// requirement, generated from [Vocabulary] alone, without invoking a
// model.
//
// Only stmt's comparator is translated. A concept's own name and a
// literal value are never part of the controlled vocabulary - this
// package has no source to translate either from, and inventing one
// (guessing a concept name's Portuguese gender or wording) is exactly
// the kind of assumption this project avoids making without a
// grounding source. This matches the vocabulary's own PortugueseReading
// doc comment: it exists "so a human reviewer can read a statement in
// Portuguese", not as a general-purpose full-sentence translator.
func GeneratePTComment(stmt Statement) (string, error) {
	reading, err := TranslateComparator(stmt.Comparator)
	if err != nil {
		return "", err
	}

	if twoValueComparators[stmt.Comparator] {
		// "is between"/"is strictly between" are the vocabulary's only
		// two-value comparators, and the only readings containing the
		// literal placeholders "X" and "Y" - substituted via
		// strings.Replacer so Value's own substitution can never be
		// re-matched by Value2's (unlike two sequential strings.Replace
		// calls would risk).
		replacer := strings.NewReplacer("X", stmt.Value, "Y", stmt.Value2)
		return fmt.Sprintf("# pt: %s %s", stmt.Term, replacer.Replace(reading)), nil
	}

	return fmt.Sprintf("# pt: %s %s %s", stmt.Term, reading, stmt.Value), nil
}
