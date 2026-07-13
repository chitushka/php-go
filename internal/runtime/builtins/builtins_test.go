package builtins

import (
	"os"
	"path/filepath"
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

func TestFilesystemBuiltins(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "sample.txt")
	if err := os.WriteFile(file, []byte("php"), 0644); err != nil {
		t.Fatal(err)
	}
	value, ok, err := Call("file_exists", []Value{StringValue(file)})
	if err != nil || !ok || !value.Truthy() {
		t.Fatalf("file_exists got value=%+v ok=%v err=%v", value, ok, err)
	}
	value, ok, err = Call("is_file", []Value{StringValue(file)})
	if err != nil || !ok || !value.Truthy() {
		t.Fatalf("is_file got value=%+v ok=%v err=%v", value, ok, err)
	}
	value, ok, err = Call("is_dir", []Value{StringValue(dir)})
	if err != nil || !ok || !value.Truthy() {
		t.Fatalf("is_dir got value=%+v ok=%v err=%v", value, ok, err)
	}
	value, ok, err = Call("file_get_contents", []Value{StringValue(file)})
	if err != nil || !ok || value.String() != "php" {
		t.Fatalf("file_get_contents got value=%q ok=%v err=%v", value.String(), ok, err)
	}
	out := filepath.Join(dir, "out.txt")
	value, ok, err = Call("file_put_contents", []Value{StringValue(out), StringValue("php 5.3")})
	if err != nil || !ok || value.Kind != IntKind || value.I != 7 {
		t.Fatalf("file_put_contents got value=%+v ok=%v err=%v", value, ok, err)
	}
	data, err := os.ReadFile(out)
	if err != nil || string(data) != "php 5.3" {
		t.Fatalf("written file got data=%q err=%v", string(data), err)
	}
}

func TestRegexBuiltins(t *testing.T) {
	value, ok, err := Call("preg_match", []Value{StringValue("/php/"), StringValue("php-src")})
	if err != nil || !ok || value.Kind != IntKind || value.I != 1 {
		t.Fatalf("preg_match got value=%+v ok=%v err=%v", value, ok, err)
	}
	value, ok, err = Call("preg_replace", []Value{StringValue("/5\\.3/"), StringValue("go"), StringValue("php 5.3")})
	if err != nil || !ok || value.String() != "php go" {
		t.Fatalf("preg_replace got value=%q ok=%v err=%v", value.String(), ok, err)
	}
}

func TestSerializationBuiltins(t *testing.T) {
	arr := NewArray()
	arr.Set(StringKey("name"), StringValue("php"))
	arr.Set(StringKey("version"), IntValue(53))
	value, ok, err := Call("serialize", []Value{ArrayValue(arr)})
	if err != nil || !ok || value.String() != `a:2:{s:4:"name";s:3:"php";s:7:"version";i:53;}` {
		t.Fatalf("serialize got value=%q ok=%v err=%v", value.String(), ok, err)
	}
	value, ok, err = Call("unserialize", []Value{value})
	if err != nil || !ok || value.Kind != ArrayKind {
		t.Fatalf("unserialize got value=%+v ok=%v err=%v", value, ok, err)
	}
	name, ok := value.A.Get(StringKey("name"))
	if !ok || name.String() != "php" {
		t.Fatalf("unserialized name got %q ok=%v", name.String(), ok)
	}
}

func TestJSONBuiltins(t *testing.T) {
	arr := NewArray()
	arr.Set(StringKey("name"), StringValue("php"))
	arr.Set(StringKey("version"), IntValue(53))
	value, ok, err := Call("json_encode", []Value{ArrayValue(arr)})
	if err != nil || !ok || value.String() != `{"name":"php","version":53}` {
		t.Fatalf("json_encode got value=%q ok=%v err=%v", value.String(), ok, err)
	}
	value, ok, err = Call("json_decode", []Value{StringValue(`{"name":"php","version":53}`), BoolValue(true)})
	if err != nil || !ok || value.Kind != ArrayKind {
		t.Fatalf("json_decode got value=%+v ok=%v err=%v", value, ok, err)
	}
	name, ok := value.A.Get(StringKey("name"))
	if !ok || name.String() != "php" {
		t.Fatalf("decoded name got %q ok=%v", name.String(), ok)
	}
}

func TestHashBuiltins(t *testing.T) {
	value, ok, err := Call("md5", []Value{StringValue("hello")})
	if err != nil || !ok || value.String() != "5d41402abc4b2a76b9719d911017c592" {
		t.Fatalf("md5 got value=%q ok=%v err=%v", value.String(), ok, err)
	}
	value, ok, err = Call("sha1", []Value{StringValue("hello")})
	if err != nil || !ok || value.String() != "aaf4c61ddcc5e8a2dabede0f3b482cd9aea9434d" {
		t.Fatalf("sha1 got value=%q ok=%v err=%v", value.String(), ok, err)
	}
	value, ok, err = Call("hash", []Value{StringValue("sha256"), StringValue("hello")})
	if err != nil || !ok || value.String() != "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824" {
		t.Fatalf("hash got value=%q ok=%v err=%v", value.String(), ok, err)
	}
}
