package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fixtureCjtr adapts the real worked example (conjecture/phenomena/shipping/
// free-shipping.cjtr) end to end: its "otherwise" branch uses a plain
// decimal threshold instead of the real example's table-lookup concept, to
// keep this fixture self-contained (no CSV file alongside it). Table
// concepts are exercised by their own CLI fixture below
// (fixtureCjtrWithTable / TestRunVerify_TableConcept), and more thoroughly
// by internal/cjtr's own tests.
const fixtureCjtr = `
conjecture: 1.0.0

phenomenon: free shipping above the threshold
description: "Frete gratis acima de R$ 100 no subtotal com desconto, menos em assinatura."

concepts:
  discounted subtotal:
    variable: input.discounted_subtotal
    type: decimal
  order type:
    variable: input.order_type
    one_of: [regular, subscription]
  shipping fee:
    variable: output.shipping_fee
    type: decimal
  free shipping threshold:
    value: 100.00

rules:
  shipping is free above the threshold:
    id: free-above
    hypotheses:
      the shipping fee is never negative:
        id: never-negative
        always: the shipping fee is at least 0.00

      above the threshold shipping is free:
        given:
          - the discounted subtotal is greater than the free shipping threshold
          - the order type is not subscription
        then: the shipping fee is 0.00
        otherwise: the shipping fee is at least 5.00
        examples:
          - {discounted subtotal: 150.00, order type: regular,      shipping fee: 0.00}
          - {discounted subtotal: 50.00,  order type: regular,      shipping fee: 7.50}
          - {discounted subtotal: 150.00, order type: subscription, shipping fee: 7.50}
`

// writeFixture writes fixtureCjtr under <tmp>/phenomena/shipping/
// free-shipping.cjtr, so runVerify derives area "shipping" and phenomenon
// "free-shipping" from the path, matching the specification's
// conjecture/phenomena/<area>/<phenomenon>.cjtr layout.
func writeFixture(t *testing.T, content string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "phenomena", "shipping")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	path := filepath.Join(dir, "free-shipping.cjtr")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	return path
}

func TestRunVerify_ConsistentExamples(t *testing.T) {
	path := writeFixture(t, fixtureCjtr)

	var stdout, stderr bytes.Buffer
	code := runVerify([]string{path}, &stdout, &stderr)

	if code != 0 {
		t.Errorf("runVerify exit code = %d, want 0; stdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "shipping::free-shipping::free-above") {
		t.Errorf("stdout should mention the rule's composite ID, got:\n%s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "3/3 examples consistent") {
		t.Errorf("stdout should report 3/3 consistent examples, got:\n%s", stdout.String())
	}
}

func TestRunVerify_DetectsViolation(t *testing.T) {
	broken := strings.Replace(fixtureCjtr,
		"- {discounted subtotal: 150.00, order type: regular,      shipping fee: 0.00}",
		"- {discounted subtotal: 150.00, order type: regular,      shipping fee: 7.50}",
		1,
	)
	path := writeFixture(t, broken)

	var stdout, stderr bytes.Buffer
	code := runVerify([]string{path}, &stdout, &stderr)

	if code != 1 {
		t.Errorf("runVerify exit code = %d, want 1 (a row was made inconsistent); stdout:\n%s", code, stdout.String())
	}
	if !strings.Contains(stdout.String(), "FAIL") {
		t.Errorf("stdout should report a FAIL, got:\n%s", stdout.String())
	}
}

// fixtureCjtrWithTable uses the real example's own table concept
// ("table shipping fee"), exactly as declared in the specification
// (table: + by: region/weight), instead of adapting it away.
const fixtureCjtrWithTable = `
conjecture: 1.0.0

phenomenon: free shipping above the threshold
description: "Frete gratis acima de R$ 100 no subtotal com desconto, menos em assinatura."

concepts:
  discounted subtotal:
    variable: input.discounted_subtotal
    type: decimal
  order type:
    variable: input.order_type
    one_of: [regular, subscription]
  shipping fee:
    variable: output.shipping_fee
    type: decimal
  free shipping threshold:
    value: 100.00
  table shipping fee:
    label: frete da tabela
    table: bases/shipping-table.csv
    by:
      region: {label: regiao, variable: input.region, type: text}
      weight: {label: peso do pacote, variable: input.weight, type: decimal, unit: kg}

rules:
  shipping is free above the threshold:
    id: free-above
    hypotheses:
      the shipping fee is never negative:
        id: never-negative
        always: the shipping fee is at least 0.00

      above the threshold shipping is free:
        given:
          - the discounted subtotal is greater than the free shipping threshold
          - the order type is not subscription
        then: the shipping fee is 0.00
        otherwise: the shipping fee equals the table shipping fee
        examples:
          - {discounted subtotal: 50.00, order type: regular, region: SP, weight: 1.00, shipping fee: 9.90}
`

// writeFixtureWithTable writes content under <tmp>/phenomena/shipping/
// free-shipping.cjtr (like writeFixture) and csv under
// <tmp>/bases/shipping-table.csv, matching the path convention runVerify
// uses to resolve a table concept's "table" field (relative to the
// conjecture tree's root, the parent of "phenomena/").
func writeFixtureWithTable(t *testing.T, content, csv string) string {
	t.Helper()
	root := t.TempDir()

	phenomenaDir := filepath.Join(root, "phenomena", "shipping")
	if err := os.MkdirAll(phenomenaDir, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	path := filepath.Join(phenomenaDir, "free-shipping.cjtr")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	basesDir := filepath.Join(root, "bases")
	if err := os.MkdirAll(basesDir, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(basesDir, "shipping-table.csv"), []byte(csv), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	return path
}

func TestRunVerify_TableConcept(t *testing.T) {
	path := writeFixtureWithTable(t, fixtureCjtrWithTable, "region,weight,fee\nSP,1.00,9.90\nRJ,1.00,12.50\n")

	var stdout, stderr bytes.Buffer
	code := runVerify([]string{path}, &stdout, &stderr)

	if code != 0 {
		t.Errorf("runVerify exit code = %d, want 0; stdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "1/1 examples consistent") {
		t.Errorf("stdout should report 1/1 consistent example, got:\n%s", stdout.String())
	}
}

func TestRunVerify_TableConcept_NoMatchingRow(t *testing.T) {
	// The example's region/weight (SP, 1.00) has no row in this CSV, so
	// the table lookup itself fails - this should surface as an ERROR for
	// that example, not a silent pass or a misleading FAIL.
	path := writeFixtureWithTable(t, fixtureCjtrWithTable, "region,weight,fee\nRJ,1.00,12.50\n")

	var stdout, stderr bytes.Buffer
	code := runVerify([]string{path}, &stdout, &stderr)

	if code != 1 {
		t.Errorf("runVerify exit code = %d, want 1 (the table has no matching row)", code)
	}
	if !strings.Contains(stdout.String(), "ERROR") {
		t.Errorf("stdout should report an ERROR for the unresolvable table lookup, got:\n%s", stdout.String())
	}
}

// fixtureCjtrWithLaw adds the real worked example's own
// "user_defined_laws" entry ("subscription orders never get free
// shipping") to fixtureCjtr, with its own example row, to check that
// runVerify checks a law's examples the same way it checks a
// hypothesis's.
const fixtureCjtrWithLaw = `
conjecture: 1.0.0

phenomenon: free shipping above the threshold
description: "Frete gratis acima de R$ 100 no subtotal com desconto, menos em assinatura."

concepts:
  discounted subtotal:
    variable: input.discounted_subtotal
    type: decimal
  order type:
    variable: input.order_type
    one_of: [regular, subscription]
  shipping fee:
    variable: output.shipping_fee
    type: decimal
  free shipping threshold:
    value: 100.00

rules:
  shipping is free above the threshold:
    id: free-above
    user_defined_laws:
      subscription orders never get free shipping:
        given: the order type is subscription
        then: the shipping fee is 7.50
        examples:
          - {order type: subscription, shipping fee: 7.50}
`

func TestRunVerify_UserDefinedLaw(t *testing.T) {
	path := writeFixture(t, fixtureCjtrWithLaw)

	var stdout, stderr bytes.Buffer
	code := runVerify([]string{path}, &stdout, &stderr)

	if code != 0 {
		t.Errorf("runVerify exit code = %d, want 0; stdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "subscription orders never get free shipping") {
		t.Errorf("stdout should mention the law's name, got:\n%s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "1/1 examples consistent") {
		t.Errorf("stdout should report 1/1 consistent example, got:\n%s", stdout.String())
	}
}

// fixtureCjtrWithPhenomenonLevelLaw declares "user_defined_laws" at the
// file's top level, as a sibling of "rules" - the specification's other
// documented placement for it ("aninhada no nível onde ela vale...no topo
// do arquivo, para o fenômeno inteiro"), distinct from
// fixtureCjtrWithLaw's rule-scoped placement above.
const fixtureCjtrWithPhenomenonLevelLaw = `
conjecture: 1.0.0

phenomenon: free shipping above the threshold
description: "Frete gratis acima de R$ 100 no subtotal com desconto, menos em assinatura."

concepts:
  discounted subtotal:
    variable: input.discounted_subtotal
    type: decimal
  order type:
    variable: input.order_type
    one_of: [regular, subscription]
  shipping fee:
    variable: output.shipping_fee
    type: decimal
  free shipping threshold:
    value: 100.00

user_defined_laws:
  subscription orders never get free shipping:
    given: the order type is subscription
    then: the shipping fee is 7.50
    examples:
      - {order type: subscription, shipping fee: 7.50}

rules:
  shipping is free above the threshold:
    id: free-above
    hypotheses:
      the shipping fee is never negative:
        id: never-negative
        always: the shipping fee is at least 0.00
`

func TestRunVerify_PhenomenonLevelLaw(t *testing.T) {
	path := writeFixture(t, fixtureCjtrWithPhenomenonLevelLaw)

	var stdout, stderr bytes.Buffer
	code := runVerify([]string{path}, &stdout, &stderr)

	if code != 0 {
		t.Errorf("runVerify exit code = %d, want 0; stdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "shipping::free-shipping: subscription orders never get free shipping") {
		t.Errorf("stdout should report the phenomenon-level law under the area::phenomenon composite ID, got:\n%s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "1/1 examples consistent") {
		t.Errorf("stdout should report 1/1 consistent example (the rule's own hypothesis declares no examples), got:\n%s", stdout.String())
	}
}

func TestRunVerify_PhenomenonLevelLawDetectsViolation(t *testing.T) {
	broken := strings.Replace(fixtureCjtrWithPhenomenonLevelLaw,
		"- {order type: subscription, shipping fee: 7.50}",
		"- {order type: subscription, shipping fee: 0.00}",
		1,
	)
	path := writeFixture(t, broken)

	var stdout, stderr bytes.Buffer
	code := runVerify([]string{path}, &stdout, &stderr)

	if code != 1 {
		t.Errorf("runVerify exit code = %d, want 1 (the phenomenon-level law's example contradicts its own \"then\")", code)
	}
	if !strings.Contains(stdout.String(), "FAIL") {
		t.Errorf("stdout should report a FAIL, got:\n%s", stdout.String())
	}
}

func TestRunVerify_UserDefinedLawDetectsViolation(t *testing.T) {
	broken := strings.Replace(fixtureCjtrWithLaw,
		"- {order type: subscription, shipping fee: 7.50}",
		"- {order type: subscription, shipping fee: 0.00}",
		1,
	)
	path := writeFixture(t, broken)

	var stdout, stderr bytes.Buffer
	code := runVerify([]string{path}, &stdout, &stderr)

	if code != 1 {
		t.Errorf("runVerify exit code = %d, want 1 (the law's example contradicts its own \"then\")", code)
	}
	if !strings.Contains(stdout.String(), "FAIL") {
		t.Errorf("stdout should report a FAIL, got:\n%s", stdout.String())
	}
}

// fixtureCjtrWithObservation adds the real worked example's own
// file-level "observation", "scope", and "simplifications" fields to
// fixtureCjtr, with its rule's "when" repeating the declared operation
// exactly, as the real example does. It also adds a file-level "axioms"
// entry (observable) and a rule-level "assumes" entry (unobservable),
// to exercise both at once.
const fixtureCjtrWithObservation = `
conjecture: 1.0.0

phenomenon: free shipping above the threshold
description: "Frete gratis acima de R$ 100 no subtotal com desconto, menos em assinatura."

simplifications: [domestic shipping only, one delivery per order]

scope:
  inside: [cart, coupon, shipping calculation]
  later: [store pickup]
  outside: [international shipping]

observation:
  operation: the shipping fee is calculated
  interface: HTTP POST /orders/shipping

axioms: the shipping fee is at least 0.00

concepts:
  discounted subtotal:
    variable: input.discounted_subtotal
    type: decimal
  order type:
    variable: input.order_type
    one_of: [regular, subscription]
  shipping fee:
    variable: output.shipping_fee
    type: decimal
  free shipping threshold:
    value: 100.00

rules:
  shipping is free above the threshold:
    id: free-above
    when: the shipping fee is calculated
    assumes: the balance never goes negative
    hypotheses:
      the shipping fee is never negative:
        id: never-negative
        always: the shipping fee is at least 0.00
`

func TestRunVerify_WhenMatchesObservation(t *testing.T) {
	path := writeFixture(t, fixtureCjtrWithObservation)

	var stdout, stderr bytes.Buffer
	code := runVerify([]string{path}, &stdout, &stderr)

	if code != 0 {
		t.Errorf("runVerify exit code = %d, want 0; stdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
}

func TestRunVerify_WhenMismatchesObservation(t *testing.T) {
	broken := strings.Replace(fixtureCjtrWithObservation,
		"when: the shipping fee is calculated",
		"when: a completely different operation",
		1,
	)
	path := writeFixture(t, broken)

	var stdout, stderr bytes.Buffer
	code := runVerify([]string{path}, &stdout, &stderr)

	if code != 1 {
		t.Errorf("runVerify exit code = %d, want 1 (\"when\" does not match the declared observation operation)", code)
	}
	if !strings.Contains(stderr.String(), "does not match") {
		t.Errorf("stderr should explain the \"when\"/observation mismatch, got %q", stderr.String())
	}
}

func TestRunVerify_MalformedScope(t *testing.T) {
	broken := strings.Replace(fixtureCjtrWithObservation,
		"inside: [cart, coupon, shipping calculation]",
		"inside: cart",
		1,
	)
	path := writeFixture(t, broken)

	var stdout, stderr bytes.Buffer
	code := runVerify([]string{path}, &stdout, &stderr)

	if code != 1 {
		t.Errorf("runVerify exit code = %d, want 1 (\"scope.inside\" is a scalar, not a list)", code)
	}
}

func TestRunVerify_MalformedSimplifications(t *testing.T) {
	broken := strings.Replace(fixtureCjtrWithObservation,
		"simplifications: [domestic shipping only, one delivery per order]",
		"simplifications: domestic shipping only",
		1,
	)
	path := writeFixture(t, broken)

	var stdout, stderr bytes.Buffer
	code := runVerify([]string{path}, &stdout, &stderr)

	if code != 1 {
		t.Errorf("runVerify exit code = %d, want 1 (\"simplifications\" is a scalar, not a list)", code)
	}
}

func TestRunVerify_AxiomsWithVocabularyMistakeFails(t *testing.T) {
	// A declared term ("shipping fee") followed by a made-up comparator is
	// a vocabulary mistake, not an unobservable premise - it must still
	// fail, not silently fall back to declared text.
	broken := strings.Replace(fixtureCjtrWithObservation,
		"axioms: the shipping fee is at least 0.00",
		"axioms: the shipping fee sort of resembles 1",
		1,
	)
	path := writeFixture(t, broken)

	var stdout, stderr bytes.Buffer
	code := runVerify([]string{path}, &stdout, &stderr)

	if code != 1 {
		t.Errorf("runVerify exit code = %d, want 1 (a declared term with an unrecognized comparator is a hard error)", code)
	}
}

func TestRunVerify_MissingFile(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runVerify([]string{filepath.Join(t.TempDir(), "nope.cjtr")}, &stdout, &stderr)

	if code != 1 {
		t.Errorf("runVerify with a missing file: exit code = %d, want 1", code)
	}
	if stderr.String() == "" {
		t.Error("runVerify with a missing file should write an error to stderr")
	}
}

func TestRunVerify_NoArguments(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runVerify(nil, &stdout, &stderr)

	if code != 1 {
		t.Errorf("runVerify with no arguments: exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Errorf("stderr should contain a usage message, got %q", stderr.String())
	}
}
