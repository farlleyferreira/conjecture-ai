package expressioncore

import (
	"fmt"
	"sync"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
)

// comparatorExpressions maps each comparator phrase internal/cjtr's
// own grammar supports (statement.go's supportedComparators) to the
// CEL expression text it compiles to, over the variable names "term",
// "value", and - for the two-value comparators - "value2". Boundary
// inclusivity matches cjtr.Evaluate's own compare() exactly: "is
// between" includes both ends, "is strictly between" excludes both -
// this package does not re-derive that rule, only mirrors it.
var comparatorExpressions = map[string]string{
	"is":                  "term == value",
	"equals":              "term == value",
	"is not":              "term != value",
	"does not equal":      "term != value",
	"is greater than":     "term > value",
	"is less than":        "term < value",
	"is at least":         "term >= value",
	"is at most":          "term <= value",
	"is between":          "term >= value && term <= value2",
	"is strictly between": "term > value && term < value2",
}

// CompileComparison compiles comparator - one of the phrases
// comparatorExpressions declares - into a CEL program comparing
// "term" against "value" (and, for "is between"/"is strictly
// between", "value2"), over operands of the given operandType.
//
// operandType is a caller decision, not something this function
// infers: internal/cjtr's own compare() always parses an ordering
// comparator's operands as exact decimals regardless of the term
// concept's declared type (see evaluate.go's own numericCompare,
// called unconditionally for every comparator but equality) and only
// an equality comparator ever compares plain text, so a caller
// compiling a decimal concept's statement passes [DecimalType], one
// comparing anything else's equality passes CEL's own native
// [types.StringType] - this function does not look at a concept at
// all, so it cannot make that choice itself.
func CompileComparison(comparator string, operandType *cel.Type) (cel.Program, error) {
	expr, ok := comparatorExpressions[comparator]
	if !ok {
		return nil, fmt.Errorf("expressioncore: %q is not a recognized comparator", comparator)
	}

	// There are only a handful of (comparator, operandType) pairs ever
	// requested - every value [comparatorExpressions] names, times two
	// operand types - so caching the compiled result keeps a caller
	// like cjtr.Evaluate (compare()), which may call this once per
	// comparison across thousands of sampled cases, from re-parsing
	// and re-type-checking the same CEL expression every time: "Local
	// performance within hardware bounds... thousands of evaluations
	// per second."
	cacheKey := comparator + "|" + operandType.TypeName()
	if cached, ok := programCache.Load(cacheKey); ok {
		return cached.(cel.Program), nil
	}

	opts := []cel.EnvOption{
		cel.Variable("term", operandType),
		cel.Variable("value", operandType),
	}
	if comparator == "is between" || comparator == "is strictly between" {
		opts = append(opts, cel.Variable("value2", operandType))
	}
	if operandType == DecimalType {
		opts = append(opts, ComparisonOptions()...)
	}

	env, err := cel.NewEnv(opts...)
	if err != nil {
		return nil, fmt.Errorf("expressioncore: building environment for %q: %w", comparator, err)
	}

	ast, issues := env.Compile(expr)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("expressioncore: compiling %q: %w", comparator, issues.Err())
	}
	prg, err := env.Program(ast)
	if err != nil {
		return nil, fmt.Errorf("expressioncore: planning program for %q: %w", comparator, err)
	}
	programCache.Store(cacheKey, prg)
	return prg, nil
}

// programCache memoizes CompileComparison's own result by "comparator|
// operandType" - see CompileComparison's own doc comment for why.
// cel.Program's own Eval is safe for concurrent use against the same
// compiled Program (it carries no per-call mutable state), so sharing
// one cached Program across callers is safe.
var programCache sync.Map

// EvalComparison evaluates prg (as returned by [CompileComparison])
// against term, value, and value2 (ignored unless prg declared it -
// CEL simply does not look up an unused activation entry), returning
// the boolean result.
func EvalComparison(prg cel.Program, term, value, value2 any) (bool, error) {
	out, _, err := prg.Eval(map[string]any{"term": term, "value": value, "value2": value2})
	if err != nil {
		return false, fmt.Errorf("expressioncore: evaluating: %w", err)
	}
	b, ok := out.Value().(bool)
	if !ok {
		return false, fmt.Errorf("expressioncore: evaluation produced %T, want bool", out.Value())
	}
	return b, nil
}

// StringType is CEL's own native string type, re-exported here so a
// caller compiling a plain-text equality comparison need not import
// github.com/google/cel-go/common/types itself just for this one
// value.
var StringType = types.StringType
