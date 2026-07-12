package runtime_test

import (
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
