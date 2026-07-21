package builtins

import (
	"fmt"
	"math"
	"strconv"
)

type Kind int

const (
	NullKind Kind = iota
	BoolKind
	IntKind
	FloatKind
	StringKind
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
}

func NullValue() Value           { return Value{Kind: NullKind} }
func BoolValue(v bool) Value     { return Value{Kind: BoolKind, B: v} }
func IntValue(v int64) Value     { return Value{Kind: IntKind, I: v} }
func FloatValue(v float64) Value { return Value{Kind: FloatKind, F: v} }
func StringValue(v string) Value { return Value{Kind: StringKind, S: v} }
func ArrayValue(v *Array) Value  { return Value{Kind: ArrayKind, A: v} }
func ObjectValue() Value         { return Value{Kind: ObjectKind} }

func (v Value) Truthy() bool {
	switch v.Kind {
	case NullKind:
		return false
	case BoolKind:
		return v.B
	case IntKind:
		return v.I != 0
	case FloatKind:
		return v.F != 0
	case StringKind:
		return v.S != "" && v.S != "0"
	case ArrayKind:
		return v.A != nil && len(v.A.Entries) > 0
	case ObjectKind:
		return true
	}
	return false
}

func (v Value) String() string {
	switch v.Kind {
	case NullKind:
		return ""
	case BoolKind:
		if v.B {
			return "1"
		}
		return ""
	case IntKind:
		return strconv.FormatInt(v.I, 10)
	case FloatKind:
		return FormatPHPFloat(v.F)
	case StringKind:
		return v.S
	case ArrayKind:
		return "Array"
	case ObjectKind:
		return "Object"
	}
	return ""
}

func (v Value) Number() (float64, bool) {
	switch v.Kind {
	case IntKind:
		return float64(v.I), true
	case FloatKind:
		return v.F, true
	case BoolKind:
		if v.B {
			return 1, true
		}
		return 0, true
	case NullKind:
		return 0, true
	case StringKind:
		n, err := strconv.ParseFloat(v.S, 64)
		return n, err == nil
	}
	return 0, false
}

func IntValueOf(value Value) int {
	number, ok := value.Number()
	if !ok {
		return 0
	}
	if math.IsNaN(number) || math.IsInf(number, 0) {
		return 0
	}
	return int(number)
}

func checkArgCount(name string, got int, min int, max int) error {
	if got < min || got > max {
		if min == max {
			return fmt.Errorf("%s expects %d arguments, got %d", name, min, got)
		}
		return fmt.Errorf("%s expects between %d and %d arguments, got %d", name, min, max, got)
	}
	return nil
}
