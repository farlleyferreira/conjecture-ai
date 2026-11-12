package omissiondetection

import (
	"testing"

	"github.com/farlleyferreira/conjecture-ai/internal/cjtr"
)

func shippingTableFixture() (cjtr.Concept, map[string]cjtr.Concept, *cjtr.Table) {
	tableConcept := cjtr.Concept{
		Name: "table shipping fee", Kind: cjtr.TableConcept,
		ByKeys: []string{"region", "weight"},
	}
	concepts := map[string]cjtr.Concept{
		"region": {Name: "region", Kind: cjtr.VariableConcept, Type: "text", OneOf: []string{"SP", "RJ"}},
		"weight": {Name: "weight", Kind: cjtr.VariableConcept, Type: "text", OneOf: []string{"light", "heavy"}},
	}
	table := &cjtr.Table{
		Header: []string{"region", "weight", "fee"},
		Rows: [][]string{
			{"SP", "light", "9.90"},
			{"SP", "heavy", "14.90"},
			{"RJ", "light", "12.50"},
			// RJ/heavy is deliberately missing.
		},
	}
	return tableConcept, concepts, table
}

// stateEventMatrixFixture is the same shape as shippingTableFixture,
// except its by-keys are named "state" and "event" instead of
// "region" and "weight" - CheckTypeC's own doc comment claims a
// state/event matrix needs no code path of its own, only a table
// concept whose by-keys happen to be named that way; this fixture
// exercises that claim directly rather than leaving it as prose.
func stateEventMatrixFixture() (cjtr.Concept, map[string]cjtr.Concept, *cjtr.Table) {
	tableConcept := cjtr.Concept{
		Name: "order lifecycle transitions", Kind: cjtr.TableConcept,
		ByKeys: []string{"state", "event"},
	}
	concepts := map[string]cjtr.Concept{
		"state": {Name: "state", Kind: cjtr.VariableConcept, Type: "text", OneOf: []string{"pending", "shipped"}},
		"event": {Name: "event", Kind: cjtr.VariableConcept, Type: "text", OneOf: []string{"pay", "cancel"}},
	}
	table := &cjtr.Table{
		Header: []string{"state", "event", "next state"},
		Rows: [][]string{
			{"pending", "pay", "shipped"},
			{"pending", "cancel", "pending"},
			{"shipped", "cancel", "shipped"},
			// shipped/pay is deliberately missing - no rule says what
			// happens on a second payment once already shipped.
		},
	}
	return tableConcept, concepts, table
}

func TestCheckTypeC_StateEventMatrixMissingCellFlagged(t *testing.T) {
	tableConcept, concepts, table := stateEventMatrixFixture()

	omissions, found, err := CheckTypeC(tableConcept, concepts, table, 100)
	if err != nil {
		t.Fatalf("CheckTypeC returned unexpected error: %v", err)
	}
	if !found {
		t.Fatal("found = false, want true (both by-keys are one_of, exhaustively enumerable)")
	}
	if len(omissions) != 1 {
		t.Fatalf("len(omissions) = %d, want 1; omissions: %+v", len(omissions), omissions)
	}
	got := omissions[0].Keys
	if got["state"] != "shipped" || got["event"] != "pay" {
		t.Errorf("omissions[0].Keys = %v, want state=shipped event=pay", got)
	}
}

func TestCheckTypeC_StateEventMatrixFullCoverageFindsNoOmission(t *testing.T) {
	tableConcept, concepts, table := stateEventMatrixFixture()
	table.Rows = append(table.Rows, []string{"shipped", "pay", "shipped"})

	omissions, found, err := CheckTypeC(tableConcept, concepts, table, 100)
	if err != nil {
		t.Fatalf("CheckTypeC returned unexpected error: %v", err)
	}
	if !found {
		t.Fatal("found = false, want true")
	}
	if len(omissions) != 0 {
		t.Errorf("CheckTypeC = %+v, want no omissions (every state/event combination has a row)", omissions)
	}
}

func TestCheckTypeC_MissingCellFlagged(t *testing.T) {
	tableConcept, concepts, table := shippingTableFixture()

	omissions, found, err := CheckTypeC(tableConcept, concepts, table, 100)
	if err != nil {
		t.Fatalf("CheckTypeC returned unexpected error: %v", err)
	}
	if !found {
		t.Fatal("found = false, want true (both by-keys are one_of, exhaustively enumerable)")
	}
	if len(omissions) != 1 {
		t.Fatalf("len(omissions) = %d, want 1; omissions: %+v", len(omissions), omissions)
	}
	got := omissions[0].Keys
	if got["region"] != "RJ" || got["weight"] != "heavy" {
		t.Errorf("omissions[0].Keys = %v, want region=RJ weight=heavy", got)
	}
}

func TestCheckTypeC_FullTableFindsNoOmission(t *testing.T) {
	tableConcept, concepts, table := shippingTableFixture()
	table.Rows = append(table.Rows, []string{"RJ", "heavy", "19.90"})

	omissions, found, err := CheckTypeC(tableConcept, concepts, table, 100)
	if err != nil {
		t.Fatalf("CheckTypeC returned unexpected error: %v", err)
	}
	if !found {
		t.Fatal("found = false, want true")
	}
	if len(omissions) != 0 {
		t.Errorf("CheckTypeC = %+v, want no omissions (every combination has a row)", omissions)
	}
}

func TestCheckTypeC_AmbiguousRowIsNotReportedAsMissing(t *testing.T) {
	tableConcept, concepts, table := shippingTableFixture()
	table.Rows = append(table.Rows, []string{"RJ", "heavy", "19.90"}, []string{"RJ", "heavy", "21.00"})

	omissions, found, err := CheckTypeC(tableConcept, concepts, table, 100)
	if err != nil {
		t.Fatalf("CheckTypeC returned unexpected error: %v", err)
	}
	if !found {
		t.Fatal("found = false, want true")
	}
	if len(omissions) != 0 {
		t.Errorf("CheckTypeC = %+v, want no omissions (RJ/heavy has rows, just more than one - a different problem)", omissions)
	}
}

func TestCheckTypeC_UnboundedByKeyIsNotFound(t *testing.T) {
	tableConcept := cjtr.Concept{Name: "table x", Kind: cjtr.TableConcept, ByKeys: []string{"unbounded"}}
	concepts := map[string]cjtr.Concept{
		"unbounded": {Name: "unbounded", Kind: cjtr.VariableConcept, Type: "decimal"},
	}
	table := &cjtr.Table{Header: []string{"unbounded", "value"}, Rows: [][]string{{"1", "x"}}}

	omissions, found, err := CheckTypeC(tableConcept, concepts, table, 100)
	if err != nil {
		t.Fatalf("CheckTypeC returned unexpected error: %v", err)
	}
	if found {
		t.Error("found = true, want false (the by-key declares neither one_of nor a bounded range)")
	}
	if len(omissions) != 0 {
		t.Errorf("omissions = %+v, want nil when found is false", omissions)
	}
}

func TestCheckTypeC_NotATableConceptIsError(t *testing.T) {
	notATable := cjtr.Concept{Name: "x", Kind: cjtr.VariableConcept}
	_, _, err := CheckTypeC(notATable, nil, nil, 100)
	if err == nil {
		t.Fatal("CheckTypeC with a non-table concept = nil error, want an error")
	}
}

func TestCheckTypeC_UndeclaredByKeyIsError(t *testing.T) {
	tableConcept := cjtr.Concept{Name: "table x", Kind: cjtr.TableConcept, ByKeys: []string{"nope"}}
	_, _, err := CheckTypeC(tableConcept, map[string]cjtr.Concept{}, &cjtr.Table{}, 100)
	if err == nil {
		t.Fatal("CheckTypeC with an undeclared by-key = nil error, want an error")
	}
}
