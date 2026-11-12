package omissiondetection

import (
	"fmt"

	"github.com/farlleyferreira/conjecture-ai/internal/cjtr"
)

// TypeCOmission is one combination of a decision table's own by-key
// values that has no matching row.
type TypeCOmission struct {
	Keys map[string]string
}

// CheckTypeC detects Type C omissions in a decision table or a
// state/event matrix - "Type C - case not specified": "The system
// SHALL detect a case missing from a decision table or a state/event
// matrix for a phenomenon... WHEN a state/event matrix for a
// phenomenon has an empty cell with no corresponding rule THEN the
// system flags it as a Type C omission."
//
// A "state/event matrix" is not a distinct grammar from a decision
// table in this codebase's own reading: nothing in the cjtr-language
// grammar (or in either omission-detection's or spec-judge's own
// specification text, which names both side by side with no worked
// example of either) declares "state" or "event" as dedicated
// concepts. A [cjtr.TableConcept]'s own by-key concepts are already
// unrestricted in what they may represent - a project models "state"
// and "event" the same way it would model "region" or "weight": as
// plain one_of-valued concepts named in tableConcept.ByKeys. Reading
// a state/event matrix this way - a decision table whose by-keys
// happen to be named "state" and "event" - is the only one that does
// not invent a second mechanism the specification gives no concrete
// shape for, so CheckTypeC implements both at once, by construction,
// rather than as two separate code paths: tableConcept (a [cjtr.
// TableConcept], backed by table) with its own by-key concepts each
// exhaustively enumerated via [cjtr.ExhaustiveValues], then every
// combination of those values checked against table via [cjtr.Table.
// MatchCount].
//
// found is false when at least one by-key concept's own domain is
// not exhaustively enumerable ([cjtr.ExhaustiveValues]'s own "found" -
// a range with no declared "places", or a one_of/range wider than
// maxCount) - the same "a category outside this catalog stays
// invisible" honesty [CheckTypeD] already applies: an empty result
// from CheckTypeC when found is false means nothing was actually
// checked, never "no omissions found." maxCount is passed straight
// through to ExhaustiveValues for each by-key - the caller's own
// disclosed cap, the same kind conformance-cycle's own bounded/
// complete-proof generation already declares for the same reason
// (ExhaustiveValues' own ceiling against an unbounded combinatorial
// blowup).
//
// A combination with more than one matching row is not reported: that
// is a different problem (an ambiguous table, the same thing [cjtr.
// Table.Lookup] itself refuses outright), not a missing one, and Type
// C is only about a cell nothing addresses.
func CheckTypeC(tableConcept cjtr.Concept, concepts map[string]cjtr.Concept, table *cjtr.Table, maxCount int) (omissions []TypeCOmission, found bool, err error) {
	if tableConcept.Kind != cjtr.TableConcept {
		return nil, false, fmt.Errorf("omissiondetection: %q is not a table concept", tableConcept.Name)
	}

	domains := make([]struct {
		name   string
		values []string
	}, len(tableConcept.ByKeys))
	for i, keyName := range tableConcept.ByKeys {
		keyConcept, ok := concepts[keyName]
		if !ok {
			return nil, false, fmt.Errorf("omissiondetection: by-key %q is not a declared concept", keyName)
		}
		values, keyFound, err := cjtr.ExhaustiveValues(keyConcept, maxCount)
		if err != nil {
			return nil, false, fmt.Errorf("omissiondetection: enumerating by-key %q: %w", keyName, err)
		}
		if !keyFound {
			return nil, false, nil
		}
		domains[i] = struct {
			name   string
			values []string
		}{keyName, values}
	}

	combos := []map[string]string{{}}
	for _, d := range domains {
		next := make([]map[string]string, 0, len(combos)*len(d.values))
		for _, combo := range combos {
			for _, v := range d.values {
				extended := make(map[string]string, len(combo)+1)
				for k, val := range combo {
					extended[k] = val
				}
				extended[d.name] = v
				next = append(next, extended)
			}
		}
		combos = next
	}

	for _, combo := range combos {
		count, err := table.MatchCount(combo, concepts)
		if err != nil {
			return nil, false, fmt.Errorf("omissiondetection: checking %v: %w", combo, err)
		}
		if count == 0 {
			omissions = append(omissions, TypeCOmission{Keys: combo})
		}
	}
	return omissions, true, nil
}
