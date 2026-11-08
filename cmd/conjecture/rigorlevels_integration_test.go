package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/farlleyferreira/conjecture-ai/internal/cjtr"
	"github.com/farlleyferreira/conjecture-ai/internal/conformancecycle"
	"github.com/farlleyferreira/conjecture-ai/internal/evidenceledger"
	"github.com/farlleyferreira/conjecture-ai/internal/sandbox"
)

// rigorFixture declares one user-defined law whose two input concepts
// are both boundable at a small scale - "discounted subtotal"'s own
// range (0.00..0.10, places 2) gives 11 values, crossed with "order
// type"'s 2 one_of values - 22 cases well within this package's own
// disclosed caps, so [conformancecycle.CheckHypothesisWithSelectedRigor]
// should select complete proof rather than falling back to bounded
// proof or directed sampling.
const rigorFixture = `
conjecture: 1.0.0

phenomenon: rigor levels demo

concepts:
  discounted subtotal:
    variable: input.discounted_subtotal
    type: decimal
    range: {min: 0.00, max: 0.10}
    places: 2
  order type:
    variable: input.order_type
    one_of: [regular, subscription]
  shipping fee:
    variable: output.shipping_fee
    type: decimal

rules:
  shipping fee rule:
    id: shipping-rule
    user_defined_laws:
      shipping fee is never negative:
        id: fee-never-negative
        always: the shipping fee is at least 0.00
        examples:
          - {discounted subtotal: 0.05, order type: regular, shipping fee: 7.50}
`

func writeRigorFixture(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "phenomena", "demo-area")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	path := filepath.Join(dir, "rigor-demo.cjtr")
	if err := os.WriteFile(path, []byte(rigorFixture), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	return path
}

// TestRunCheck_LawWithBoundableDomainsUsesCompleteProof is this
// change's own end-to-end proof: a real .cjtr file on disk, loaded by
// the real loadFicha, checked through the real
// checkHypothesisConformance path ("conjecture check"'s own per-
// hypothesis function) against a fake "docker" that computes a real,
// varying output per case (fakeRigorPath/testdata/fakerigor) instead
// of fakeImplPath's one canned string. This only passes if
// CheckHypothesisWithSelectedRigor actually ran complete proof's full
// 11x2 cross-product for real - a single directed-sampling round
// would never exercise most of that range, and a fake implementation
// that ignored its input could not tell the difference either way.
func TestRunCheck_LawWithBoundableDomainsUsesCompleteProof(t *testing.T) {
	path := writeRigorFixture(t)
	f, err := loadFicha(path)
	if err != nil {
		t.Fatalf("loadFicha returned unexpected error: %v", err)
	}
	if len(f.rules) != 1 {
		t.Fatalf("len(f.rules) = %d, want 1", len(f.rules))
	}
	laws, err := cjtr.UserDefinedLaws(f.rules[0].Body, f.concepts)
	if err != nil {
		t.Fatalf("UserDefinedLaws returned unexpected error: %v", err)
	}
	if len(laws) != 1 {
		t.Fatalf("len(laws) = %d, want 1", len(laws))
	}

	db := openCheckTestStore(t)
	sb := &sandbox.Sandbox{DockerBinary: fakeRigorPath}

	var stdout bytes.Buffer
	failed := checkHypothesisConformance(
		context.Background(), db, sb, "unused", []string{"compute"},
		laws[0], f.rules[0].CompositeID, "test-spec-version", true, f.concepts, f.tables, &stdout,
	)
	if failed {
		t.Fatalf("checkHypothesisConformance = true (failed), want false - stdout:\n%s", stdout.String())
	}
	if !strings.Contains(stdout.String(), conformancecycle.CompleteProof.String()) {
		t.Errorf("stdout = %q, want it to report the complete proof rigor level", stdout.String())
	}

	hypothesisID := f.rules[0].CompositeID + "::fee-never-negative"
	history, err := evidenceledger.History(db, hypothesisID)
	if err != nil {
		t.Fatalf("History returned unexpected error: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("len(history) = %d, want 1", len(history))
	}
	if history[0].Attacked != conformancecycle.CompleteProof.String() {
		t.Errorf("recorded evidence.Attacked = %q, want %q", history[0].Attacked, conformancecycle.CompleteProof.String())
	}
	if history[0].Unchecked != "" {
		t.Errorf("recorded evidence.Unchecked = %q, want empty (complete proof leaves nothing weaker unchecked)", history[0].Unchecked)
	}
}
