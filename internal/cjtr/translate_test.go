package cjtr

import "testing"

func TestTranslateComparator_SinglePhraseEntry(t *testing.T) {
	got, err := TranslateComparator("is greater than")
	if err != nil {
		t.Fatalf("TranslateComparator returned unexpected error: %v", err)
	}
	if got != "é maior que" {
		t.Errorf("TranslateComparator(%q) = %q, want %q", "is greater than", got, "é maior que")
	}
}

func TestTranslateComparator_PicksTheRightSynonymReading(t *testing.T) {
	cases := map[string]string{
		"is":             "é",
		"equals":         "é igual a",
		"is not":         "não é",
		"does not equal": "é diferente de",
	}
	for phrase, want := range cases {
		got, err := TranslateComparator(phrase)
		if err != nil {
			t.Errorf("TranslateComparator(%q) returned unexpected error: %v", phrase, err)
			continue
		}
		if got != want {
			t.Errorf("TranslateComparator(%q) = %q, want %q", phrase, got, want)
		}
	}
}

func TestTranslateComparator_TwoValueComparatorKeepsPlaceholders(t *testing.T) {
	got, err := TranslateComparator("is between")
	if err != nil {
		t.Fatalf("TranslateComparator returned unexpected error: %v", err)
	}
	if got != "está entre X e Y, inclusive" {
		t.Errorf("TranslateComparator(%q) = %q, want the raw X/Y reading", "is between", got)
	}
}

func TestTranslateComparator_UnknownPhraseIsError(t *testing.T) {
	_, err := TranslateComparator("is somehow related to")
	if err == nil {
		t.Fatal("TranslateComparator with an unknown phrase = nil error, want one")
	}
}

func TestTranslateComparator_NonComparatorPhraseIsError(t *testing.T) {
	_, err := TranslateComparator("plus") // ArithmeticCategory, not ComparatorCategory
	if err == nil {
		t.Fatal("TranslateComparator with a non-comparator phrase = nil error, want one")
	}
}

func TestGeneratePTComment_SingleValueComparator(t *testing.T) {
	stmt := Statement{Term: "discounted subtotal", Comparator: "is greater than", Value: "100.00"}
	got, err := GeneratePTComment(stmt)
	if err != nil {
		t.Fatalf("GeneratePTComment returned unexpected error: %v", err)
	}
	want := "# pt: discounted subtotal é maior que 100.00"
	if got != want {
		t.Errorf("GeneratePTComment = %q, want %q", got, want)
	}
}

func TestGeneratePTComment_TwoValueComparatorSubstitutesBothBounds(t *testing.T) {
	stmt := Statement{Term: "age", Comparator: "is between", Value: "18", Value2: "65"}
	got, err := GeneratePTComment(stmt)
	if err != nil {
		t.Fatalf("GeneratePTComment returned unexpected error: %v", err)
	}
	want := "# pt: age está entre 18 e 65, inclusive"
	if got != want {
		t.Errorf("GeneratePTComment = %q, want %q", got, want)
	}
}

func TestGeneratePTComment_TwoValueComparatorBoundsDoNotCrossContaminate(t *testing.T) {
	// Value2 containing the literal letter "Y" must not be affected by the
	// earlier "X" substitution picking it up - strings.Replacer (not two
	// sequential strings.Replace calls) is what guarantees this.
	stmt := Statement{Term: "code", Comparator: "is strictly between", Value: "A", Value2: "Y"}
	got, err := GeneratePTComment(stmt)
	if err != nil {
		t.Fatalf("GeneratePTComment returned unexpected error: %v", err)
	}
	want := "# pt: code está estritamente entre A e Y"
	if got != want {
		t.Errorf("GeneratePTComment = %q, want %q", got, want)
	}
}

func TestGeneratePTComment_UnknownComparatorIsError(t *testing.T) {
	_, err := GeneratePTComment(Statement{Term: "x", Comparator: "is somehow related to", Value: "y"})
	if err == nil {
		t.Fatal("GeneratePTComment with an unrecognized comparator = nil error, want one")
	}
}
