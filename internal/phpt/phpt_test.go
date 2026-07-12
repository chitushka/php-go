package phpt_test

import (
	"testing"

	"php-src-go/internal/phpt"
)

func TestRunPassingPHPT(t *testing.T) {
	input := `--TEST--
basic arithmetic
--FILE--
<?php echo 1 + 2 * 3;
--EXPECT--
7
`
	result := phpt.Run("basic.phpt", input)
	if !result.Passed {
		t.Fatalf("expected pass, got err=%v output=%q expected=%q", result.Err, result.Output, result.Expected)
	}
}

func TestRunFailingPHPT(t *testing.T) {
	input := `--TEST--
failing arithmetic
--FILE--
<?php echo 1 + 1;
--EXPECT--
3
`
	result := phpt.Run("fail.phpt", input)
	if result.Passed {
		t.Fatal("expected failure")
	}
	if result.Output != "2" {
		t.Fatalf("got output %q", result.Output)
	}
}

func TestParseRequiresFileAndExpect(t *testing.T) {
	if _, err := phpt.Parse("--TEST--\nmissing\n"); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestRunExpectF(t *testing.T) {
	input := `--TEST--
expectf wildcard
--FILE--
<?php echo 'value 123';
--EXPECTF--
value %d
`
	result := phpt.Run("expectf.phpt", input)
	if !result.Passed {
		t.Fatalf("expected pass, got output=%q expected=%q err=%v", result.Output, result.Expected, result.Err)
	}
}

func TestRunSkipIf(t *testing.T) {
	input := `--TEST--
skip example
--SKIPIF--
<?php echo 'skip unsupported feature';
--FILE--
<?php echo 'no';
--EXPECT--
no
`
	result := phpt.Run("skip.phpt", input)
	if !result.Skipped {
		t.Fatalf("expected skip, got %+v", result)
	}
}

func TestRunExpectedRuntimeError(t *testing.T) {
	input := `--TEST--
expected runtime error
--FILE--
<?php class Sample { private $a = 1; } $s = new Sample(); echo $s->a;
--EXPECTF--
cannot access private property Sample::$a
`
	result := phpt.Run("error.phpt", input)
	if !result.Passed {
		t.Fatalf("expected pass, got output=%q expected=%q err=%v", result.Output, result.Expected, result.Err)
	}
}
