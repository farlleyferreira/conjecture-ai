package cjtr

import "testing"

func axiomsTestConcepts(t *testing.T) map[string]Concept {
	t.Helper()
	root := loadNode(t, "concepts:\n  shipping fee:\n    variable: output.shipping_fee\n    type: decimal\n")
	concepts, err := Concepts(root)
	if err != nil {
		t.Fatalf("Concepts returned unexpected error: %v", err)
	}
	return concepts
}

func TestAxioms_ObservablePremise(t *testing.T) {
	concepts := axiomsTestConcepts(t)
	root := loadNode(t, "axioms: the shipping fee is at least 0.00\n")

	axioms, err := Axioms(root, concepts)
	if err != nil {
		t.Fatalf("Axioms returned unexpected error: %v", err)
	}
	if len(axioms) != 1 {
		t.Fatalf("Axioms returned %d entries, want 1", len(axioms))
	}
	if axioms[0].Statement == nil {
		t.Fatal("axioms[0].Statement = nil, want a parsed Statement (the term is a declared concept)")
	}
	if axioms[0].Statement.Term != "shipping fee" || axioms[0].Statement.Comparator != "is at least" {
		t.Errorf("axioms[0].Statement = %+v", axioms[0].Statement)
	}
	if axioms[0].Text != "the shipping fee is at least 0.00" {
		t.Errorf("axioms[0].Text = %q", axioms[0].Text)
	}
}

func TestAxioms_UnobservablePremiseStaysAsText(t *testing.T) {
	concepts := axiomsTestConcepts(t)
	root := loadNode(t, "axioms: the balance never goes negative\n")

	axioms, err := Axioms(root, concepts)
	if err != nil {
		t.Fatalf("Axioms returned unexpected error: %v", err)
	}
	if len(axioms) != 1 {
		t.Fatalf("Axioms returned %d entries, want 1", len(axioms))
	}
	if axioms[0].Statement != nil {
		t.Errorf("axioms[0].Statement = %+v, want nil (\"balance\" is not a declared concept)", axioms[0].Statement)
	}
	if axioms[0].Text != "the balance never goes negative" {
		t.Errorf("axioms[0].Text = %q", axioms[0].Text)
	}
}

func TestAxioms_List(t *testing.T) {
	concepts := axiomsTestConcepts(t)
	root := loadNode(t, "axioms:\n  - the shipping fee is at least 0.00\n  - the balance never goes negative\n")

	axioms, err := Axioms(root, concepts)
	if err != nil {
		t.Fatalf("Axioms returned unexpected error: %v", err)
	}
	if len(axioms) != 2 {
		t.Fatalf("Axioms returned %d entries, want 2", len(axioms))
	}
	if axioms[0].Statement == nil || axioms[1].Statement != nil {
		t.Errorf("axioms = %+v, want [observable, unobservable]", axioms)
	}
}

func TestAxioms_AssumesSynonym(t *testing.T) {
	concepts := axiomsTestConcepts(t)
	root := loadNode(t, "assumes: the shipping fee is at least 0.00\n")

	axioms, err := Axioms(root, concepts)
	if err != nil {
		t.Fatalf("Axioms returned unexpected error: %v", err)
	}
	if len(axioms) != 1 || axioms[0].Statement == nil {
		t.Errorf("Axioms(\"assumes\") = %+v, want one observable entry", axioms)
	}
}

func TestAxioms_BothAxiomsAndAssumes(t *testing.T) {
	concepts := axiomsTestConcepts(t)
	root := loadNode(t, "axioms: the shipping fee is at least 0.00\nassumes: the shipping fee is at least 0.00\n")

	_, err := Axioms(root, concepts)
	if err == nil {
		t.Fatal("Axioms with both \"axioms\" and \"assumes\" declared = nil error, want an error")
	}
}

func TestAxioms_NoKey(t *testing.T) {
	root := loadNode(t, "conjecture: 1.0.0\n")

	axioms, err := Axioms(root, map[string]Concept{})
	if err != nil {
		t.Fatalf("Axioms returned unexpected error: %v", err)
	}
	if axioms != nil {
		t.Errorf("Axioms with neither key = %v, want nil", axioms)
	}
}

func TestAxioms_DeclaredTermWithUnrecognizedComparatorIsAHardError(t *testing.T) {
	concepts := axiomsTestConcepts(t)
	root := loadNode(t, "axioms: the shipping fee sort of resembles 1\n")

	_, err := Axioms(root, concepts)
	if err == nil {
		t.Fatal("Axioms with a declared term but an unrecognized comparator = nil error, want an error (a vocabulary mistake, not an unobservable premise)")
	}
}
