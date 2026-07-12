package runtime

import (
	"fmt"
	"math"
	"strconv"
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
)

type Value struct {
	Kind Kind
	B    bool
	I    int64
	F    float64
	S    string
	A    *Array
	O    *Object
}

func NullValue() Value            { return Value{Kind: Null} }
func BoolValue(v bool) Value      { return Value{Kind: Bool, B: v} }
func IntValue(v int64) Value      { return Value{Kind: Int, I: v} }
func FloatValue(v float64) Value  { return Value{Kind: Float, F: v} }
func StringValue(v string) Value  { return Value{Kind: String, S: v} }
func ArrayValue(v *Array) Value   { return Value{Kind: ArrayKind, A: v} }
func ObjectValue(v *Object) Value { return Value{Kind: ObjectKind, O: v} }

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
		n, err := strconv.ParseFloat(v.S, 64)
		return n, err == nil
	}
	return 0, false
}

func Binary(op string, a, b Value) (Value, error) {
	switch op {
	case "+", "-", "*", "/":
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
		}
	case "==":
		return BoolValue(a.String() == b.String()), nil
	case "!=":
		return BoolValue(a.String() != b.String()), nil
	case "<", ">", "<=", ">=":
		na, oka := a.number()
		nb, okb := b.number()
		if !oka || !okb {
			return NullValue(), fmt.Errorf("non comparable operand")
		}
		switch op {
		case "<":
			return BoolValue(na < nb), nil
		case ">":
			return BoolValue(na > nb), nil
		case "<=":
			return BoolValue(na <= nb), nil
		case ">=":
			return BoolValue(na >= nb), nil
		}
	}
	return NullValue(), fmt.Errorf("unsupported operator %s", op)
}

func numericResult(v float64) (Value, error) {
	if math.Trunc(v) == v {
		return IntValue(int64(v)), nil
	}
	return FloatValue(v), nil
}
