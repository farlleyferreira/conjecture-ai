package cjtr

import (
	"testing"
	"time"
)

// TestEvaluate_MeetsExpressionCorePerformanceBound checks this
// package's direct evaluator against expression-core's own "Local
// performance within hardware bounds" requirement (openspec/specs/
// expression-core/spec.md): "The expression core SHALL be able to
// perform at least thousands of evaluations per second on a machine
// with 6 GB of RAM."
//
// Evaluate's own doc comment already says plainly that it stands in
// for the real expression-core capability (no CEL/FEEL compilation,
// no provable subset translatable to a solver) rather than being it;
// this test does not change that. It only holds the one requirement
// that is a plain, measurable number - the same evaluator already
// used throughout this package's tests, run enough times to get a
// stable measurement, checked against a deliberately conservative
// reading of "thousands" (1,000/s, its lowest literal floor) so the
// test does not become flaky on a slower machine over a bound the
// specification itself never pins to an exact number.
func TestEvaluate_MeetsExpressionCorePerformanceBound(t *testing.T) {
	concepts := loadEvaluateConcepts(t)
	stmt, err := ParseComparison("the discounted subtotal is greater than the free shipping threshold", concepts)
	if err != nil {
		t.Fatalf("ParseComparison returned unexpected error: %v", err)
	}
	bindings := Bindings{"input.discounted_subtotal": "150.00"}

	const iterations = 5000
	start := time.Now()
	for i := 0; i < iterations; i++ {
		if _, err := Evaluate(stmt, concepts, nil, bindings); err != nil {
			t.Fatalf("Evaluate returned unexpected error: %v", err)
		}
	}
	elapsed := time.Since(start)

	const minimumPerSecond = 1000
	perSecond := float64(iterations) / elapsed.Seconds()
	if perSecond < minimumPerSecond {
		t.Errorf(
			"Evaluate managed %.0f evaluations/second (%d in %v), want at least %d",
			perSecond, iterations, elapsed, minimumPerSecond,
		)
	}
}
