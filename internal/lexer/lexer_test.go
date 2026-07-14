package lexer

import "testing"

func TestShortEchoOpenTagTokenizesAsEcho(t *testing.T) {
	l := New("<?= $name ?>")

	tests := []TokenType{Echo, Var, EOF}
	for _, want := range tests {
		if got := l.NextToken(); got.Type != want {
			t.Fatalf("got token %s (%q), want %s", got.Type, got.Literal, want)
		}
	}
}
