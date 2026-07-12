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
