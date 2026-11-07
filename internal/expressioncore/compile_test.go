package expressioncore

import (
	"testing"
	"time"
)

func mustDecimal(t *testing.T, s string) Decimal {
	t.Helper()
	d, err := NewDecimal(s)
	if err != nil {
		t.Fatalf("NewDecimal(%q) returned unexpected error: %v", s, err)
	}
	return d
}

func TestCompileComparison_DecimalComparators(t *testing.T) {
	cases := []struct {
		comparator  string
		term, value string
		value2      string // "" when not a two-value comparator
		want        bool
	}{
		{"is", "7.50", "7.5", "", true},
		{"is", "7.50", "9.99", "", false},
		{"equals", "7.50", "7.5", "", true},
		{"is not", "7.50", "9.99", "", true},
		{"is not", "7.50", "7.5", "", false},
		{"does not equal", "7.50", "9.99", "", true},
		{"is greater than", "9.99", "7.50", "", true},
		{"is greater than", "7.50", "9.99", "", false},
		{"is less than", "7.50", "9.99", "", true},
		{"is less than", "9.99", "7.50", "", false},
		{"is at least", "7.50", "7.50", "", true},
		{"is at least", "7.49", "7.50", "", false},
		{"is at most", "7.50", "7.50", "", true},
		{"is at most", "7.51", "7.50", "", false},
		{"is between", "100.00", "50.00", "150.00", true},
		{"is between", "50.00", "50.00", "150.00", true},  // inclusive lower bound
		{"is between", "150.00", "50.00", "150.00", true}, // inclusive upper bound
		{"is between", "49.99", "50.00", "150.00", false},
		{"is between", "150.01", "50.00", "150.00", false},
		{"is strictly between", "100.00", "50.00", "150.00", true},
		{"is strictly between", "50.00", "50.00", "150.00", false},  // exclusive lower bound
		{"is strictly between", "150.00", "50.00", "150.00", false}, // exclusive upper bound
	}

	for _, c := range cases {
		prg, err := CompileComparison(c.comparator, DecimalType)
		if err != nil {
			t.Fatalf("CompileComparison(%q) returned unexpected error: %v", c.comparator, err)
		}
		term := mustDecimal(t, c.term)
		value := mustDecimal(t, c.value)
		var value2 any = Decimal{}
		if c.value2 != "" {
			value2 = mustDecimal(t, c.value2)
		}
		got, err := EvalComparison(prg, term, value, value2)
		if err != nil {
			t.Fatalf("EvalComparison(%q, %s, %s, %s) returned unexpected error: %v", c.comparator, c.term, c.value, c.value2, err)
		}
		if got != c.want {
			t.Errorf("EvalComparison(%q, %s, %s, %s) = %v, want %v", c.comparator, c.term, c.value, c.value2, got, c.want)
		}
	}
}

func TestCompileComparison_StringEquality(t *testing.T) {
	prg, err := CompileComparison("is", StringType)
	if err != nil {
		t.Fatalf("CompileComparison(\"is\") returned unexpected error: %v", err)
	}
	got, err := EvalComparison(prg, "regular", "regular", "")
	if err != nil {
		t.Fatalf("EvalComparison returned unexpected error: %v", err)
	}
	if !got {
		t.Error(`EvalComparison("regular", "regular") = false, want true`)
	}

	got, err = EvalComparison(prg, "regular", "subscription", "")
	if err != nil {
		t.Fatalf("EvalComparison returned unexpected error: %v", err)
	}
	if got {
		t.Error(`EvalComparison("regular", "subscription") = true, want false`)
	}
}

func TestCompileComparison_UnrecognizedComparator(t *testing.T) {
	if _, err := CompileComparison("is one of", DecimalType); err == nil {
		t.Error(`CompileComparison("is one of") = nil error, want one (not in cjtr's supported vocabulary)`)
	}
}

// TestEvalComparison_ThousandsPerSecond grounds "Local performance
// within hardware bounds": "The expression core SHALL be able to
// perform at least thousands of evaluations per second on a machine
// with 6 GB of RAM." It compiles once (as a real caller - cjtr.Evaluate's
// own compare() - does, relying on CompileComparison's own cache
// rather than recompiling per call) and times 10,000 evaluations,
// asserting that batch completes within one second - a looser bound
// than "at least thousands per second" demands, so this fails loudly
// well before the budget is actually at risk, on any machine, not
// just a 6 GB one this test has no way to provision.
func TestEvalComparison_ThousandsPerSecond(t *testing.T) {
	prg, err := CompileComparison("is at least", DecimalType)
	if err != nil {
		t.Fatalf("CompileComparison returned unexpected error: %v", err)
	}
	term := mustDecimal(t, "7.50")
	value := mustDecimal(t, "0.00")

	const n = 10000
	start := time.Now()
	for i := 0; i < n; i++ {
		if _, err := EvalComparison(prg, term, value, Decimal{}); err != nil {
			t.Fatalf("EvalComparison returned unexpected error: %v", err)
		}
	}
	elapsed := time.Since(start)
	if elapsed > time.Second {
		t.Errorf("%d evaluations took %v, want well under 1s (thousands/sec budget)", n, elapsed)
	}
}
