package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fixtureCjtr adapts the real worked example (conjecture/phenomena/shipping/
// free-shipping.cjtr) end to end, with the same adaptation used in
// internal/cjtr's own tests: the "otherwise" branch uses a plain decimal
// threshold instead of a table-lookup concept, which is not supported yet.
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
