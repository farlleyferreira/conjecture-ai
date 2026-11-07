// Package expressioncore implements pieces of the expression-core
// capability (openspec/specs/expression-core/spec.md): a deterministic,
// side-effect-free expression engine every .cjtr statement is meant to
// compile to, evaluated via CEL (github.com/google/cel-go) rather than
// interpreting natural language at verification time.
//
// This package's first piece is exact decimal arithmetic on top of
// CEL: CEL's own built-in numeric types (int64, uint64, and double, a
// 64-bit IEEE binary float) have no arbitrary-precision decimal type,
// so a statement comparing monetary or other exact-decimal values
// would otherwise lose precision through double - exactly what
// "Exact decimal arithmetic" forbids. Rather than falling back to
// FEEL for every decimal comparison (no mature FEEL implementation
// exists for Go - building one would mean writing a expression-
// language parser/evaluator from scratch, a far larger and less
// grounded undertaking than extending CEL, which the specification
// itself anticipates as a real possibility: "The system SHALL fall
// back to FEEL for a statement only when CEL cannot represent it with
// exact decimal semantics... within the provable subset" - a
// condition this package exists so that decimal comparisons never
// meet), this package registers its own exact-decimal CEL type
// (Decimal, backed by math/big.Rat, the same representation
// internal/cjtr's own hand-written evaluator already uses and for the
// same reason) so decimal statements compile to CEL like anything
// else, with FEEL staying a true fallback for whatever this still
// cannot represent.
package expressioncore

import (
	"fmt"
	"math/big"
	"reflect"

	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"
)

// DecimalType is this package's registered CEL type for Decimal
// values. It declares the Comparer trait (traits.ComparerType) so
// CEL's ordering operators (<, <=, >, >=) dispatch to Decimal.Compare;
// equality (==, !=) needs no trait declaration - it is part of every
// [ref.Val] via Equal.
var DecimalType = types.NewObjectType("expressioncore.Decimal", traits.ComparerType)

// Decimal is one exact-decimal CEL value, backed by *big.Rat -
// [big.Rat.SetString] parses decimal text exactly (e.g. "100.00"
// becomes precisely 100, never a binary-float approximation), the
// same reason internal/cjtr's own parseExactDecimal uses it. This
// package does not import internal/cjtr (and nothing there imports
// this package yet): the two independently satisfy the same exact-
// decimal requirement, consistent with cjtr.Evaluate's own doc
// comment that it is "not yet a compiler to CEL... nothing here
// should be mistaken for the real expression-core capability" - a
// later change, not this one, decides whether/how cjtr comes to use
// this package instead of its own hand-written evaluator.
type Decimal struct {
	rat *big.Rat
}

// NewDecimal parses s as an exact decimal number.
func NewDecimal(s string) (Decimal, error) {
	r, ok := new(big.Rat).SetString(s)
	if !ok {
		return Decimal{}, fmt.Errorf("expressioncore: %q is not a valid exact decimal number", s)
	}
	return Decimal{rat: r}, nil
}

// ConvertToNative implements [ref.Val]. Only conversion to Decimal's
// own Go type is supported - nothing in this package's own callers
// needs a Decimal exposed as any other native Go numeric type (a
// float64 conversion would reintroduce exactly the precision loss
// this type exists to avoid, so it is deliberately not offered).
func (d Decimal) ConvertToNative(typeDesc reflect.Type) (any, error) {
	if typeDesc == reflect.TypeOf(Decimal{}) {
		return d, nil
	}
	return nil, fmt.Errorf("expressioncore: unsupported native conversion from Decimal to %v", typeDesc)
}

// ConvertToType implements [ref.Val]. A Decimal only converts to
// itself - no other CEL type in this package's current scope needs a
// Decimal value.
func (d Decimal) ConvertToType(typeVal ref.Type) ref.Val {
	if typeVal == DecimalType {
		return d
	}
	return types.NewErr("expressioncore: no conversion from Decimal to %v", typeVal)
}

// Equal implements [ref.Val] - "Exact decimal arithmetic": compared
// via [big.Rat.Cmp], never through a lossy float64 conversion.
func (d Decimal) Equal(other ref.Val) ref.Val {
	o, ok := other.(Decimal)
	if !ok {
		return types.MaybeNoSuchOverloadErr(other)
	}
	return types.Bool(d.rat.Cmp(o.rat) == 0)
}

// Type implements [ref.Val].
func (d Decimal) Type() ref.Type {
	return DecimalType
}

// Value implements [ref.Val], returning the underlying *big.Rat.
func (d Decimal) Value() any {
	return d.rat
}

// Compare implements [traits.Comparer] - the same exact comparison
// Equal uses, generalized to CEL's ordering operators (<, <=, >, >=).
func (d Decimal) Compare(other ref.Val) ref.Val {
	o, ok := other.(Decimal)
	if !ok {
		return types.MaybeNoSuchOverloadErr(other)
	}
	return types.Int(d.rat.Cmp(o.rat))
}
