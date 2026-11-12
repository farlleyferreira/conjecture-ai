package cjtr

import (
	"os"
	"path/filepath"
	"testing"
)

func writeCSV(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	return path
}

func TestLoadTable(t *testing.T) {
	path := writeCSV(t, t.TempDir(), "shipping.csv", "region,weight,fee\nSP,1.00,9.90\nSP,5.00,14.90\nRJ,1.00,12.50\n")

	table, err := LoadTable(path)
	if err != nil {
		t.Fatalf("LoadTable returned unexpected error: %v", err)
	}
	if got := table.Header; len(got) != 3 || got[0] != "region" || got[1] != "weight" || got[2] != "fee" {
		t.Errorf("Header = %v, want [region weight fee]", got)
	}
	if len(table.Rows) != 3 {
		t.Errorf("len(Rows) = %d, want 3", len(table.Rows))
	}
}

func TestLoadTable_MissingFile(t *testing.T) {
	_, err := LoadTable(filepath.Join(t.TempDir(), "nope.csv"))
	if err == nil {
		t.Fatal("LoadTable with a missing file = nil error, want an error")
	}
}

func TestLoadTable_Empty(t *testing.T) {
	path := writeCSV(t, t.TempDir(), "empty.csv", "")

	_, err := LoadTable(path)
	if err == nil {
		t.Fatal("LoadTable with an empty file = nil error, want an error")
	}
}

func TestLoadTable_RaggedRows(t *testing.T) {
	path := writeCSV(t, t.TempDir(), "ragged.csv", "region,fee\nSP,9.90,extra\n")

	_, err := LoadTable(path)
	if err == nil {
		t.Fatal("LoadTable with a row that has a different field count than the header = nil error, want an error")
	}
}

func TestTable_Lookup(t *testing.T) {
	table := &Table{
		Header: []string{"region", "weight", "fee"},
		Rows: [][]string{
			{"SP", "1.00", "9.90"},
			{"SP", "5.00", "14.90"},
			{"RJ", "1.00", "12.50"},
		},
	}
	keyConcepts := map[string]Concept{
		"region": {Name: "region", Kind: VariableConcept, Type: "text"},
		"weight": {Name: "weight", Kind: VariableConcept, Type: "decimal"},
	}

	got, err := table.Lookup(map[string]string{"region": "SP", "weight": "1.0"}, keyConcepts)
	if err != nil {
		t.Fatalf("Lookup returned unexpected error: %v", err)
	}
	if got != "9.90" {
		t.Errorf(`Lookup = %q, want %q ("1.0" should decimal-match the row's "1.00")`, got, "9.90")
	}
}

func TestTable_Lookup_NoMatch(t *testing.T) {
	table := &Table{Header: []string{"region", "fee"}, Rows: [][]string{{"SP", "9.90"}}}
	keyConcepts := map[string]Concept{"region": {Name: "region", Kind: VariableConcept, Type: "text"}}

	_, err := table.Lookup(map[string]string{"region": "RJ"}, keyConcepts)
	if err == nil {
		t.Fatal("Lookup with no matching row = nil error, want an error")
	}
}

func TestTable_Lookup_AmbiguousRows(t *testing.T) {
	table := &Table{Header: []string{"region", "fee"}, Rows: [][]string{{"SP", "9.90"}, {"SP", "14.90"}}}
	keyConcepts := map[string]Concept{"region": {Name: "region", Kind: VariableConcept, Type: "text"}}

	_, err := table.Lookup(map[string]string{"region": "SP"}, keyConcepts)
	if err == nil {
		t.Fatal("Lookup with more than one matching row = nil error, want an error")
	}
}

func TestTable_Lookup_AmbiguousValueColumn(t *testing.T) {
	table := &Table{Header: []string{"region", "fee", "extra"}, Rows: [][]string{{"SP", "9.90", "x"}}}
	keyConcepts := map[string]Concept{"region": {Name: "region", Kind: VariableConcept, Type: "text"}}

	_, err := table.Lookup(map[string]string{"region": "SP"}, keyConcepts)
	if err == nil {
		t.Fatal("Lookup with more than one column besides the by-keys = nil error, want an error")
	}
}

func TestTable_MatchCount(t *testing.T) {
	table := &Table{
		Header: []string{"region", "fee"},
		Rows:   [][]string{{"SP", "9.90"}, {"SP", "14.90"}, {"RJ", "12.50"}},
	}
	keyConcepts := map[string]Concept{"region": {Name: "region", Kind: VariableConcept, Type: "text"}}

	tests := []struct {
		region string
		want   int
	}{
		{"SP", 2},
		{"RJ", 1},
		{"BA", 0},
	}
	for _, tt := range tests {
		got, err := table.MatchCount(map[string]string{"region": tt.region}, keyConcepts)
		if err != nil {
			t.Fatalf("MatchCount(%q) returned unexpected error: %v", tt.region, err)
		}
		if got != tt.want {
			t.Errorf("MatchCount(%q) = %d, want %d", tt.region, got, tt.want)
		}
	}
}

func TestTable_MatchCount_UnknownByKeyIsError(t *testing.T) {
	table := &Table{Header: []string{"region", "fee"}, Rows: [][]string{{"SP", "9.90"}}}
	keyConcepts := map[string]Concept{"region": {Name: "region", Kind: VariableConcept, Type: "text"}}

	_, err := table.MatchCount(map[string]string{"nope": "x"}, keyConcepts)
	if err == nil {
		t.Fatal("MatchCount with a by-key not in the table's header = nil error, want an error")
	}
}

func TestLoadTables(t *testing.T) {
	dir := t.TempDir()
	writeCSV(t, dir, "shipping.csv", "region,fee\nSP,9.90\n")

	concepts := map[string]Concept{
		"table shipping fee": {Name: "table shipping fee", Kind: TableConcept, TablePath: "shipping.csv", ByKeys: []string{"region"}},
		"region":             {Name: "region", Kind: VariableConcept, Variable: "input.region", Type: "text"},
	}

	tables, err := LoadTables(concepts, dir)
	if err != nil {
		t.Fatalf("LoadTables returned unexpected error: %v", err)
	}
	if _, ok := tables["table shipping fee"]; !ok {
		t.Fatal(`LoadTables did not load "table shipping fee"`)
	}
}

func TestLoadTables_MissingFile(t *testing.T) {
	concepts := map[string]Concept{
		"table shipping fee": {Name: "table shipping fee", Kind: TableConcept, TablePath: "nope.csv"},
	}

	_, err := LoadTables(concepts, t.TempDir())
	if err == nil {
		t.Fatal("LoadTables with a missing CSV file = nil error, want an error")
	}
}

// TestEvaluate_TableConcept runs a table concept through the real chain
// (Concepts -> LoadTables -> Evaluate), not just Table.Lookup in
// isolation, to check the pieces actually fit together.
func TestEvaluate_TableConcept(t *testing.T) {
	dir := t.TempDir()
	writeCSV(t, dir, "shipping-table.csv", "region,weight,fee\nSP,1.00,9.90\nRJ,1.00,12.50\n")

	src := `
concepts:
  shipping fee:
    variable: output.shipping_fee
    type: decimal
  table shipping fee:
    table: shipping-table.csv
    by:
      region: {variable: input.region, type: text}
      weight: {variable: input.weight, type: decimal}
`
	root := loadNode(t, src)
	concepts, err := Concepts(root)
	if err != nil {
		t.Fatalf("Concepts returned unexpected error: %v", err)
	}

	tables, err := LoadTables(concepts, dir)
	if err != nil {
		t.Fatalf("LoadTables returned unexpected error: %v", err)
	}

	stmt, err := ParseComparison("the shipping fee equals the table shipping fee", concepts)
	if err != nil {
		t.Fatalf("ParseComparison returned unexpected error: %v", err)
	}

	bindings := Bindings{
		"output.shipping_fee": "9.90",
		"input.region":        "SP",
		"input.weight":        "1.0",
	}

	got, err := Evaluate(stmt, concepts, tables, bindings)
	if err != nil {
		t.Fatalf("Evaluate returned unexpected error: %v", err)
	}
	if !got {
		t.Error("Evaluate(shipping fee equals table shipping fee) = false, want true (9.90 matches the SP/1.00 row)")
	}

	bindings["output.shipping_fee"] = "12.50"
	got, err = Evaluate(stmt, concepts, tables, bindings)
	if err != nil {
		t.Fatalf("Evaluate returned unexpected error: %v", err)
	}
	if got {
		t.Error("Evaluate(shipping fee equals table shipping fee) = true, want false (12.50 does not match the SP/1.00 row's 9.90)")
	}
}
