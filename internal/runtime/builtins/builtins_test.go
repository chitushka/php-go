package builtins

import "testing"

func TestStringBuiltins(t *testing.T) {
	value, ok, err := Call("substr", []Value{StringValue("abcdef"), IntValue(1), IntValue(3)})
	if err != nil || !ok || value.String() != "bcd" {
		t.Fatalf("substr got value=%q ok=%v err=%v", value.String(), ok, err)
	}
	value, ok, err = Call("strpos", []Value{StringValue("abcdef"), StringValue("de")})
	if err != nil || !ok || value.Kind != IntKind || value.I != 3 {
		t.Fatalf("strpos got value=%+v ok=%v err=%v", value, ok, err)
	}
}

func TestArrayBuiltins(t *testing.T) {
	arr := NewArray()
	arr.Append(StringValue("a"))
	arr.Append(StringValue("b"))
	value, ok, err := Call("implode", []Value{StringValue("-"), ArrayValue(arr)})
	if err != nil || !ok || value.String() != "a-b" {
		t.Fatalf("implode got value=%q ok=%v err=%v", value.String(), ok, err)
	}
	value, ok, err = Call("count", []Value{ArrayValue(arr)})
	if err != nil || !ok || value.Kind != IntKind || value.I != 2 {
		t.Fatalf("count got value=%+v ok=%v err=%v", value, ok, err)
	}
}

func TestVariableAndTypeBuiltins(t *testing.T) {
	value, ok, err := Call("isset", []Value{StringValue("x"), NullValue()})
	if err != nil || !ok || value.Truthy() {
		t.Fatalf("isset got value=%+v ok=%v err=%v", value, ok, err)
	}
	value, ok, err = Call("is_string", []Value{StringValue("x")})
	if err != nil || !ok || !value.Truthy() {
		t.Fatalf("is_string got value=%+v ok=%v err=%v", value, ok, err)
	}
}
