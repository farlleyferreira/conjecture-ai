package expressioncore

import (
	"fmt"
	"math/big"
	"strings"
)

// EncodeComparatorConstraint translates one cjtr comparator over a
// free variable named varName against bound1 (and, for the two-value
// comparators, bound2) into an SMT-LIB2 boolean term - expression-
// core's own "Provable subset translatable to a solver" requirement:
// "A defined subset of compiled statements SHALL be translatable into
// a form usable by an SMT solver, to support consistency checking
// without executing code."
//
// This only ever encodes a comparison against a constant bound, never
// against another free variable: the provable subset this project
// currently supports is a single free variable's own declared domain
// (one decimal concept, compared against literal/constant bounds
// across possibly several statements) - exactly the shape spec-
// judge's own scenario names ("two statements... impose contradictory
// conditions on the same input range"). A comparison against another
// variable concept would need a second free variable and is not
// something this function (or anything calling it yet) attempts.
func EncodeComparatorConstraint(varName, comparator string, bound1, bound2 Decimal) (string, error) {
	b1 := smtRealLiteral(bound1)
	switch comparator {
	case "is", "equals":
		return fmt.Sprintf("(= %s %s)", varName, b1), nil
	case "is not", "does not equal":
		return fmt.Sprintf("(not (= %s %s))", varName, b1), nil
	case "is greater than":
		return fmt.Sprintf("(> %s %s)", varName, b1), nil
	case "is less than":
		return fmt.Sprintf("(< %s %s)", varName, b1), nil
	case "is at least":
		return fmt.Sprintf("(>= %s %s)", varName, b1), nil
	case "is at most":
		return fmt.Sprintf("(<= %s %s)", varName, b1), nil
	case "is between":
		b2 := smtRealLiteral(bound2)
		return fmt.Sprintf("(and (>= %s %s) (<= %s %s))", varName, b1, varName, b2), nil
	case "is strictly between":
		b2 := smtRealLiteral(bound2)
		return fmt.Sprintf("(and (> %s %s) (< %s %s))", varName, b1, varName, b2), nil
	default:
		return "", fmt.Errorf("expressioncore: %q is not a recognized comparator", comparator)
	}
}

// BuildSatisfiabilityQuery assembles a complete SMT-LIB2 script
// declaring varName as a free Real-sorted constant, asserting every
// one of constraints (each as produced by [EncodeComparatorConstraint]
// or otherwise already valid SMT-LIB2), and asking the solver to
// check satisfiability of their conjunction. An "unsat" result means
// no value of varName satisfies every constraint at once - a
// contradiction between whatever statements produced them.
func BuildSatisfiabilityQuery(varName string, constraints []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "(declare-const %s Real)\n", varName)
	for _, c := range constraints {
		fmt.Fprintf(&b, "(assert %s)\n", c)
	}
	b.WriteString("(check-sat)\n")
	return b.String()
}

// smtRealLiteral renders d as an SMT-LIB2 Real-sorted literal, via
// its own exact numerator/denominator ([big.Rat.Num]/[big.Rat.Denom])
// rather than its decimal text: SMT-LIB2's own Real-literal grammar
// requires a decimal point (a bare "100" is an Int literal, which
// would not type-check against a Real-sorted variable) and disallows
// a leading "-" on a numeral (a negative value is only well-formed as
// "(- n)") - encoding through the exact rational pair sidesteps both
// rules directly rather than patching the decimal text to satisfy
// them, and stays exact (no float, consistent with every other exact-
// decimal path in this project) for a value this package's own
// [NewDecimal] cannot represent as a terminating decimal to begin
// with (SMT-LIB2's own "(/ n d)" division form has no such
// restriction).
func smtRealLiteral(d Decimal) string {
	num, den := d.rat.Num(), d.rat.Denom()
	numText := num.String()
	if num.Sign() < 0 {
		numText = fmt.Sprintf("(- %s)", new(big.Int).Neg(num).String())
	}
	if den.Cmp(big.NewInt(1)) == 0 {
		return numText
	}
	return fmt.Sprintf("(/ %s %s)", numText, den.String())
}
