package runtime_test

import (
	"os"
	"path/filepath"
	"testing"

	"php-src-go/internal/lexer"
	"php-src-go/internal/parser"
	"php-src-go/internal/runtime"
)

func run(t *testing.T, src string) string {
	t.Helper()
	p := parser.New(lexer.New(src))
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse errors: %v", p.Errors())
	}
	i := runtime.NewInterpreter()
	if err := i.Run(program); err != nil {
		t.Fatalf("runtime error: %v", err)
	}
	return i.Out.String()
}

func TestEchoArithmetic(t *testing.T) {
	if got := run(t, "<?php echo 1 + 2 * 3;"); got != "7" {
		t.Fatalf("got %q", got)
	}
}
func TestVariablesAndIf(t *testing.T) {
	if got := run(t, "<?php $a = 4; if ($a > 2) { echo 'yes'; } else { echo 'no'; }"); got != "yes" {
		t.Fatalf("got %q", got)
	}
}
func TestWhile(t *testing.T) {
	if got := run(t, "<?php $i = 0; while ($i < 3) { echo $i; $i = $i + 1; }"); got != "012" {
		t.Fatalf("got %q", got)
	}
}

func TestArrayForeachValues(t *testing.T) {
	got := run(t, "<?php $items = array('a', 'b', 'c'); foreach ($items as $v) { echo $v; }")
	if got != "abc" {
		t.Fatalf("got %q", got)
	}
}

func TestArrayForeachKeysAndValues(t *testing.T) {
	got := run(t, "<?php $items = array('x' => 7, 3 => 'z', 'last'); foreach ($items as $k => $v) { echo $k; echo ':'; echo $v; echo ';'; }")
	if got != "x:7;3:z;4:last;" {
		t.Fatalf("got %q", got)
	}
}

func TestFunctionReturn(t *testing.T) {
	got := run(t, "<?php function add($a, $b) { return $a + $b; } echo add(2, 5);")
	if got != "7" {
		t.Fatalf("got %q", got)
	}
}

func TestFunctionLocalScope(t *testing.T) {
	got := run(t, "<?php $x = 1; function setx() { $x = 9; return $x; } echo setx(); echo $x;")
	if got != "91" {
		t.Fatalf("got %q", got)
	}
}

func TestFunctionImplicitNullReturn(t *testing.T) {
	got := run(t, "<?php function noop() { $x = 1; } echo 'a'; echo noop(); echo 'b';")
	if got != "ab" {
		t.Fatalf("got %q", got)
	}
}

func TestFunctionWithForeach(t *testing.T) {
	src := "<?php function sum_values($items) { $sum = 0; foreach ($items as $v) { $sum = $sum + $v; } return $sum; } echo sum_values(array(1, 2, 3));"
	got := run(t, src)
	if got != "6" {
		t.Fatalf("got %q", got)
	}
}

func TestFunctionCanBeCalledBeforeDeclaration(t *testing.T) {
	got := run(t, "<?php echo add(2, 3); function add($a, $b) { return $a + $b; }")
	if got != "5" {
		t.Fatalf("got %q", got)
	}
}

func runFile(t *testing.T, baseDir, src string) string {
	t.Helper()
	p := parser.New(lexer.New(src))
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse errors: %v", p.Errors())
	}
	i := runtime.NewInterpreterWithBaseDir(baseDir)
	if err := i.Run(program); err != nil {
		t.Fatalf("runtime error: %v", err)
	}
	return i.Out.String()
}

func TestIncludeSharesCurrentScope(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "vars.php"), []byte("<?php $x = $x + 2; echo $x;"), 0644); err != nil {
		t.Fatal(err)
	}
	got := runFile(t, dir, "<?php $x = 3; include 'vars.php'; echo $x;")
	if got != "55" {
		t.Fatalf("got %q", got)
	}
}

func TestRequireDefinesFunctions(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "math.php"), []byte("<?php function inc($v) { return $v + 1; }"), 0644); err != nil {
		t.Fatal(err)
	}
	got := runFile(t, dir, "<?php require 'math.php'; echo inc(4);")
	if got != "5" {
		t.Fatalf("got %q", got)
	}
}

func TestNestedIncludeUsesIncludedFileDirectory(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "lib")
	if err := os.Mkdir(sub, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "value.php"), []byte("<?php $value = 8;"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "loader.php"), []byte("<?php include 'value.php';"), 0644); err != nil {
		t.Fatal(err)
	}
	got := runFile(t, dir, "<?php include 'lib/loader.php'; echo $value;")
	if got != "8" {
		t.Fatalf("got %q", got)
	}
}
