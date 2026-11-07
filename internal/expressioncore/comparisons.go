package expressioncore

import (
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/operators"
	"github.com/google/cel-go/common/types"
)

// ComparisonOptions declares CEL's four ordering operators (<, <=, >,
// >=) for two Decimal operands, so CEL's type-checker accepts an
// expression like "term < value" where both are DecimalType.
//
// No runtime binding is declared here: CEL's standard library already
// binds each of these operators to a single, generic implementation
// that dispatches via [traits.Comparer] whenever an operand's type
// declares the Comparer trait - exactly what DecimalType already
// does - so adding a second, Decimal-specific binding would conflict
// with it ("singleton function incompatible with specialized
// overloads", confirmed empirically). Only the overload signature is
// missing; the dispatch mechanism it needs already exists.
func ComparisonOptions() []cel.EnvOption {
	decimalOverload := func(operator, overloadID string) cel.EnvOption {
		return cel.Function(operator,
			cel.Overload(overloadID, []*cel.Type{DecimalType, DecimalType}, types.BoolType),
		)
	}

	return []cel.EnvOption{
		decimalOverload(operators.Less, "decimal_less_decimal"),
		decimalOverload(operators.LessEquals, "decimal_less_equals_decimal"),
		decimalOverload(operators.Greater, "decimal_greater_decimal"),
		decimalOverload(operators.GreaterEquals, "decimal_greater_equals_decimal"),
	}
}
