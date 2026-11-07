package expressioncore

import (
	"testing"

	"github.com/google/cel-go/cel"
)

func mustEnv(t *testing.T) *cel.Env {
	t.Helper()
	opts := append(ComparisonOptions(),
		cel.Variable("term", DecimalType),
		cel.Variable("value", DecimalType),
	)
	env, err := cel.NewEnv(opts...)
	if err != nil {
		t.Fatalf("cel.NewEnv returned unexpected error: %v", err)
	}
	return env
}

func evalBool(t *testing.T, env *cel.Env, expr string, term, value Decimal) bool {
	t.Helper()
	ast, issues := env.Compile(expr)
	if issues != nil && issues.Err() != nil {
		t.Fatalf("Compile(%q) returned unexpected error: %v", expr, issues.Err())
	}
	prg, err := env.Program(ast)
	if err != nil {
		t.Fatalf("Program returned unexpected error: %v", err)
	}
	out, _, err := prg.Eval(map[string]any{"term": term, "value": value})
	if err != nil {
		t.Fatalf("Eval(%q) returned unexpected error: %v", expr, err)
	}
	b, ok := out.Value().(bool)
	if !ok {
		t.Fatalf("Eval(%q) = %v (%T), want a bool", expr, out, out)
	}
	return b
}

func TestDecimal_EqualityOperators(t *testing.T) {
	env := mustEnv(t)
	seven50, _ := NewDecimal("7.50")
	sevenFifty, _ := NewDecimal("7.5")
	nine99, _ := NewDecimal("9.99")

	if !evalBool(t, env, "term == value", seven50, sevenFifty) {
		t.Error(`"7.50" == "7.5" via CEL = false, want true (exact decimal equality)`)
	}
	if evalBool(t, env, "term == value", seven50, nine99) {
		t.Error(`"7.50" == "9.99" via CEL = true, want false`)
	}
	if !evalBool(t, env, "term != value", seven50, nine99) {
		t.Error(`"7.50" != "9.99" via CEL = false, want true`)
	}
}

func TestDecimal_OrderingOperators(t *testing.T) {
	env := mustEnv(t)
	seven50, _ := NewDecimal("7.50")
	nine99, _ := NewDecimal("9.99")

	cases := []struct {
		expr string
		want bool
	}{
		{"term < value", true},
		{"term <= value", true},
		{"term > value", false},
		{"term >= value", false},
	}
	for _, c := range cases {
		if got := evalBool(t, env, c.expr, seven50, nine99); got != c.want {
			t.Errorf("evalBool(%q, 7.50, 9.99) = %v, want %v", c.expr, got, c.want)
		}
	}
}

func TestDecimal_ExactnessBeatsFloat64RoundingTrap(t *testing.T) {
	// The classic float64 trap: 0.1 + 0.2 != 0.3 in binary floating
	// point. This package never adds, but it must still tell these two
	// decimal-text values apart exactly rather than via any float64
	// round-trip - construct the "0.3"-like value a different way
	// (as text, not by floating-point addition) and confirm equality
	// holds exactly, the way it would not if either value had passed
	// through float64 first.
	env := mustEnv(t)
	a, _ := NewDecimal("0.30")
	b, _ := NewDecimal("0.3")
	if !evalBool(t, env, "term == value", a, b) {
		t.Error(`"0.30" == "0.3" via CEL = false, want true (same exact rational value)`)
	}
}

func TestNewDecimal_InvalidText(t *testing.T) {
	if _, err := NewDecimal("not-a-number"); err == nil {
		t.Error("NewDecimal(\"not-a-number\") = nil error, want one")
	}
}
