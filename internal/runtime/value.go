package runtime

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"php-src-go/internal/ast"
)

type Kind int

const (
	Null Kind = iota
	Bool
	Int
	Float
	String
	ArrayKind
	ObjectKind
	ClosureKind
)

type Closure struct {
	Fn       *ast.FunctionDeclStmt
	Captures map[string]*variableSlot
}

type Value struct {
	Kind Kind
	B    bool
	I    int64
	F    float64
	S    string
	A    *Array
	O    *Object
	C    *Closure
}

func NullValue() Value            { return Value{Kind: Null} }
func BoolValue(v bool) Value      { return Value{Kind: Bool, B: v} }
func IntValue(v int64) Value      { return Value{Kind: Int, I: v} }
func FloatValue(v float64) Value  { return Value{Kind: Float, F: v} }
func StringValue(v string) Value  { return Value{Kind: String, S: v} }
func ArrayValue(v *Array) Value   { return Value{Kind: ArrayKind, A: v} }
func ObjectValue(v *Object) Value { return Value{Kind: ObjectKind, O: v} }
func ClosureValue(v *Closure) Value {
	return Value{Kind: ClosureKind, C: v}
}

func (v Value) Truthy() bool {
	switch v.Kind {
	case Null:
		return false
	case Bool:
		return v.B
	case Int:
		return v.I != 0
	case Float:
		return v.F != 0
	case String:
		return v.S != "" && v.S != "0"
	case ArrayKind:
		return v.A != nil && len(v.A.Entries) > 0
	case ObjectKind:
		return v.O != nil
	case ClosureKind:
		return v.C != nil
	}
	return false
}

func (v Value) String() string {
	switch v.Kind {
	case Null:
		return ""
	case Bool:
		if v.B {
			return "1"
		}
		return ""
	case Int:
		return strconv.FormatInt(v.I, 10)
	case Float:
		return strconv.FormatFloat(v.F, 'f', -1, 64)
	case String:
		return v.S
	case ArrayKind:
		return "Array"
	case ObjectKind:
		return "Object"
	case ClosureKind:
		return "Closure"
	}
	return ""
}

func (v Value) number() (float64, bool) {
	switch v.Kind {
	case Int:
		return float64(v.I), true
	case Float:
		return v.F, true
	case Bool:
		if v.B {
			return 1, true
		}
		return 0, true
	case Null:
		return 0, true
	case String:
		return parseNumericPrefix(v.S), true
	}
	return 0, false
}

func Binary(op string, a, b Value) (Value, error) {
	switch op {
	case "+", "-", "*", "/", "%":
		na, oka := a.number()
		nb, okb := b.number()
		if !oka || !okb {
			return NullValue(), fmt.Errorf("non numeric operand")
		}
		switch op {
		case "+":
			return numericResult(na + nb)
		case "-":
			return numericResult(na - nb)
		case "*":
			return numericResult(na * nb)
		case "/":
			if nb == 0 {
				return NullValue(), fmt.Errorf("division by zero")
			}
			return FloatValue(na / nb), nil
		case "%":
			if nb == 0 {
				return NullValue(), fmt.Errorf("division by zero")
			}
			return IntValue(int64(na) % int64(nb)), nil
		}
	case "&", "|", "^", "<<", ">>":
		na, oka := a.number()
		nb, okb := b.number()
		if !oka || !okb {
			return NullValue(), fmt.Errorf("non numeric operand")
		}
		left := int64(na)
		right := uint(int64(nb))
		switch op {
		case "&":
			return IntValue(left & int64(nb)), nil
		case "|":
			return IntValue(left | int64(nb)), nil
		case "^":
			return IntValue(left ^ int64(nb)), nil
		case "<<":
			return IntValue(left << right), nil
		case ">>":
			return IntValue(left >> right), nil
		}
	case "==":
		return BoolValue(looseEqual(a, b)), nil
	case "!=", "<>":
		return BoolValue(!looseEqual(a, b)), nil
	case "===":
		return BoolValue(strictEqual(a, b)), nil
	case "!==":
		return BoolValue(!strictEqual(a, b)), nil
	case "<", ">", "<=", ">=":
		cmp, ok := looseCompare(a, b)
		if !ok {
			return NullValue(), fmt.Errorf("non comparable operand")
		}
		switch op {
		case "<":
			return BoolValue(cmp < 0), nil
		case ">":
			return BoolValue(cmp > 0), nil
		case "<=":
			return BoolValue(cmp <= 0), nil
		case ">=":
			return BoolValue(cmp >= 0), nil
		}
	}
	return NullValue(), fmt.Errorf("unsupported operator %s", op)
}

func looseEqual(a, b Value) bool {
	if a.Kind == Bool || b.Kind == Bool {
		return a.Truthy() == b.Truthy()
	}
	if a.Kind == Null || b.Kind == Null {
		return nullLooseEqual(a, b)
	}
	if isNumberKind(a.Kind) && b.Kind == String {
		return numericValue(a) == parseNumericPrefix(b.S)
	}
	if a.Kind == String && isNumberKind(b.Kind) {
		return parseNumericPrefix(a.S) == numericValue(b)
	}
	if isNumberKind(a.Kind) && isNumberKind(b.Kind) {
		return numericValue(a) == numericValue(b)
	}
	if a.Kind == String && b.Kind == String {
		if an, aok := fullNumericString(a.S); aok {
			if bn, bok := fullNumericString(b.S); bok {
				return an == bn
			}
		}
		return a.S == b.S
	}
	if a.Kind == ArrayKind || b.Kind == ArrayKind {
		return a.Kind == ArrayKind && b.Kind == ArrayKind && a.A == b.A
	}
	if a.Kind == ObjectKind || b.Kind == ObjectKind {
		return a.Kind == ObjectKind && b.Kind == ObjectKind && a.O == b.O
	}
	return a.String() == b.String()
}

func nullLooseEqual(a, b Value) bool {
	if a.Kind == Null && b.Kind == Null {
		return true
	}
	other := b
	if b.Kind == Null {
		other = a
	}
	switch other.Kind {
	case Bool:
		return !other.B
	case Int, Float:
		return numericValue(other) == 0
	case String:
		return other.S == ""
	case ArrayKind:
		return other.A == nil || len(other.A.Entries) == 0
	default:
		return false
	}
}

func looseCompare(a, b Value) (int, bool) {
	if a.Kind == Bool || b.Kind == Bool {
		return compareFloat(boolNumber(a.Truthy()), boolNumber(b.Truthy())), true
	}
	if a.Kind == Null || b.Kind == Null {
		return compareFloat(nullCompareNumber(a), nullCompareNumber(b)), true
	}
	if isNumberKind(a.Kind) && b.Kind == String {
		return compareFloat(numericValue(a), parseNumericPrefix(b.S)), true
	}
	if a.Kind == String && isNumberKind(b.Kind) {
		return compareFloat(parseNumericPrefix(a.S), numericValue(b)), true
	}
	if isNumberKind(a.Kind) && isNumberKind(b.Kind) {
		return compareFloat(numericValue(a), numericValue(b)), true
	}
	if a.Kind == String && b.Kind == String {
		if an, aok := fullNumericString(a.S); aok {
			if bn, bok := fullNumericString(b.S); bok {
				return compareFloat(an, bn), true
			}
		}
		return strings.Compare(a.S, b.S), true
	}
	return strings.Compare(a.String(), b.String()), true
}

func isNumberKind(kind Kind) bool {
	return kind == Int || kind == Float
}

func numericValue(value Value) float64 {
	switch value.Kind {
	case Int:
		return float64(value.I)
	case Float:
		return value.F
	default:
		return 0
	}
}

func fullNumericString(text string) (float64, bool) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return 0, false
	}
	value, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return 0, false
	}
	return value, true
}

func boolNumber(value bool) float64 {
	if value {
		return 1
	}
	return 0
}

func nullCompareNumber(value Value) float64 {
	if value.Kind == Null {
		return 0
	}
	if value.Kind == String {
		return parseNumericPrefix(value.S)
	}
	if value.Kind == Bool {
		return boolNumber(value.B)
	}
	if isNumberKind(value.Kind) {
		return numericValue(value)
	}
	if value.Kind == ArrayKind && value.A != nil && len(value.A.Entries) > 0 {
		return 1
	}
	return 0
}

func compareFloat(a, b float64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

func strictEqual(a, b Value) bool {
	if a.Kind != b.Kind {
		return false
	}
	switch a.Kind {
	case Null:
		return true
	case Bool:
		return a.B == b.B
	case Int:
		return a.I == b.I
	case Float:
		return a.F == b.F
	case String:
		return a.S == b.S
	case ArrayKind:
		return a.A == b.A
	case ObjectKind:
		return a.O == b.O
	default:
		return false
	}
}

func numericResult(v float64) (Value, error) {
	if math.Trunc(v) == v {
		return IntValue(int64(v)), nil
	}
	return FloatValue(v), nil
}
