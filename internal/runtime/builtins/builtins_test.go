package builtins

import (
	"strings"
	"testing"
)

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

func TestMoreStringBuiltins(t *testing.T) {
	value, ok, err := Call("trim", []Value{StringValue(" php ")})
	if err != nil || !ok || value.String() != "php" {
		t.Fatalf("trim got value=%q ok=%v err=%v", value.String(), ok, err)
	}
	value, ok, err = Call("str_replace", []Value{StringValue("5.3"), StringValue("go"), StringValue("php 5.3")})
	if err != nil || !ok || value.String() != "php go" {
		t.Fatalf("str_replace got value=%q ok=%v err=%v", value.String(), ok, err)
	}
}

func TestMathAndDateBuiltins(t *testing.T) {
	value, ok, err := Call("round", []Value{FloatValue(1.6)})
	if err != nil || !ok || value.Kind != IntKind || value.I != 2 {
		t.Fatalf("round got value=%+v ok=%v err=%v", value, ok, err)
	}
	value, ok, err = Call("max", []Value{IntValue(1), IntValue(7), IntValue(3)})
	if err != nil || !ok || value.Kind != IntKind || value.I != 7 {
		t.Fatalf("max got value=%+v ok=%v err=%v", value, ok, err)
	}
	value, ok, err = Call("date", []Value{StringValue("Y-m-d"), IntValue(0)})
	if err != nil || !ok || value.String() != "1970-01-01" {
		t.Fatalf("date got value=%q ok=%v err=%v", value.String(), ok, err)
	}
}

func TestOutputBuiltins(t *testing.T) {
	arr := NewArray()
	arr.Set(StringKey("name"), StringValue("php"))
	value, output, ok, err := CallWithOutput("print_r", []Value{ArrayValue(arr), BoolValue(true)})
	if err != nil || !ok || output != "" || !strings.Contains(value.String(), "[name] => php") {
		t.Fatalf("print_r return got value=%q output=%q ok=%v err=%v", value.String(), output, ok, err)
	}
	_, output, ok, err = CallWithOutput("var_dump", []Value{StringValue("php")})
	if err != nil || !ok || output != "string(3) \"php\"\n" {
		t.Fatalf("var_dump got output=%q ok=%v err=%v", output, ok, err)
	}
}
