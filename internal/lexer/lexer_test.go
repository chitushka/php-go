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

func TestStringEscapes(t *testing.T) {
	tests := []struct {
		source string
		want   string
	}{
		{`<?php "a\n\t\x41\101\\\""` + ";", "a\n\tAA\\\""},
		{`<?php 'it\'s \n \\ ok'` + ";", `it's \n \ ok`},
	}

	for _, tt := range tests {
		l := New(tt.source)
		got := l.NextToken()
		if got.Type != String {
			t.Fatalf("got token %s, want STRING", got.Type)
		}
		if got.Literal != tt.want {
			t.Fatalf("got literal %q, want %q", got.Literal, tt.want)
		}
	}
}
