package cjtr

import (
	"strings"
	"testing"

	"github.com/farlleyferreira/conjecture-ai/internal/yamlstrict"
)

// realExample is a trimmed version of the worked .cjtr example from the
// conjecture specification (conjecture/phenomena/shipping/free-shipping.cjtr),
// kept faithful to its actual field names and shape.
const realExample = `
conjecture: 1.0.0

phenomenon: free shipping above the threshold
description: "Frete gratis acima de R$ 100 no subtotal com desconto, menos em assinatura."

rules:
  shipping is free above the threshold:
    id: free-above
    when: the shipping fee is calculated
    unchanged: [discounted subtotal, taxes]

    hypotheses:
      the shipping fee is never negative:
        id: never-negative
        always: the shipping fee is at least 0.00
`

func loadNode(t *testing.T, src string) *yamlstrict.Node {
	t.Helper()
	n, err := yamlstrict.Load(strings.NewReader(src))
	if err != nil {
		t.Fatalf("yamlstrict.Load returned unexpected error: %v", err)
	}
	return n
}

// realExampleConcepts declares just enough concepts for realExample's own
// "unchanged" clause ("discounted subtotal", "taxes") to validate.
var realExampleConcepts = map[string]Concept{
	"discounted subtotal": {Name: "discounted subtotal"},
	"taxes":               {Name: "taxes"},
}

func TestRules_RealExample(t *testing.T) {
	root := loadNode(t, realExample)

	rules, err := Rules(root, "shipping", "free-shipping", realExampleConcepts, nil)
	if err != nil {
		t.Fatalf("Rules returned unexpected error: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("Rules returned %d rules, want 1", len(rules))
	}

	got := rules[0]
	want := Rule{
		Name:        "shipping is free above the threshold",
		ID:          "free-above",
		CompositeID: "shipping::free-shipping::free-above",
		Unchanged:   []string{"discounted subtotal", "taxes"},
		When:        "the shipping fee is calculated",
	}
	if got.Name != want.Name || got.ID != want.ID || got.CompositeID != want.CompositeID || got.When != want.When {
		t.Errorf("Rules()[0] = %+v, want %+v", got, want)
	}
	if len(got.Unchanged) != 2 || got.Unchanged[0] != "discounted subtotal" || got.Unchanged[1] != "taxes" {
		t.Errorf("Rules()[0].Unchanged = %v, want %v", got.Unchanged, want.Unchanged)
	}
	if got.Body == nil || got.Body.Kind != yamlstrict.MappingKind {
		t.Errorf("Rules()[0].Body = %v, want the rule's mapping node", got.Body)
	}
	if _, ok := got.Body.Get("hypotheses"); !ok {
		t.Error("Rules()[0].Body should expose the rule's own fields (e.g. \"hypotheses\"), but Get(\"hypotheses\") found nothing")
	}
}

func TestRules_WhenMatchesObservation(t *testing.T) {
	root := loadNode(t, realExample)

	observation := &Observation{Operation: "the shipping fee is calculated"}
	rules, err := Rules(root, "shipping", "free-shipping", realExampleConcepts, observation)
	if err != nil {
		t.Fatalf("Rules returned unexpected error: %v", err)
	}
	if rules[0].When != observation.Operation {
		t.Errorf("rules[0].When = %q, want %q", rules[0].When, observation.Operation)
	}
}

func TestRules_WhenMismatchesObservation(t *testing.T) {
	root := loadNode(t, realExample)

	observation := &Observation{Operation: "a different operation entirely"}
	_, err := Rules(root, "shipping", "free-shipping", realExampleConcepts, observation)
	if err == nil {
		t.Fatal("Rules with \"when\" not matching the declared observation operation = nil error, want an error")
	}
}

func TestRules_UnchangedUndeclaredTerm(t *testing.T) {
	src := `
rules:
  shipping is free above the threshold:
    id: free-above
    unchanged: [nonexistent term]
`
	root := loadNode(t, src)

	_, err := Rules(root, "shipping", "free-shipping", map[string]Concept{}, nil)
	if err == nil {
		t.Fatal("Rules with \"unchanged\" naming an undeclared concept = nil error, want an error")
	}
}

func TestRules_MissingRulesKey(t *testing.T) {
	root := loadNode(t, "conjecture: 1.0.0\nphenomenon: x\n")

	_, err := Rules(root, "shipping", "free-shipping", nil, nil)
	if err == nil {
		t.Fatal("Rules with no \"rules\" key = nil error, want an error")
	}
}

func TestRules_RuleMissingID(t *testing.T) {
	src := `
rules:
  shipping is free above the threshold:
    when: the shipping fee is calculated
`
	root := loadNode(t, src)

	_, err := Rules(root, "shipping", "free-shipping", nil, nil)
	if err == nil {
		t.Fatal("Rules with a rule missing \"id\" = nil error, want an error")
	}
}

func TestRules_InvalidID(t *testing.T) {
	src := `
rules:
  shipping is free above the threshold:
    id: Not A Valid Slug
`
	root := loadNode(t, src)

	_, err := Rules(root, "shipping", "free-shipping", nil, nil)
	if err == nil {
		t.Fatal("Rules with an invalid id segment = nil error, want an error")
	}
}

func TestRules_MultipleRules(t *testing.T) {
	src := `
rules:
  shipping is free above the threshold:
    id: free-above
  shipping fee is never negative:
    id: never-negative
`
	root := loadNode(t, src)

	rules, err := Rules(root, "shipping", "free-shipping", nil, nil)
	if err != nil {
		t.Fatalf("Rules returned unexpected error: %v", err)
	}
	if len(rules) != 2 {
		t.Fatalf("Rules returned %d rules, want 2", len(rules))
	}
	if rules[0].CompositeID != "shipping::free-shipping::free-above" {
		t.Errorf("rules[0].CompositeID = %q", rules[0].CompositeID)
	}
	if rules[1].CompositeID != "shipping::free-shipping::never-negative" {
		t.Errorf("rules[1].CompositeID = %q", rules[1].CompositeID)
	}
}
