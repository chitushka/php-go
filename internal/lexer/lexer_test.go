package lexer

import "testing"

func TestShortEchoOpenTagTokenizesAsEcho(t *testing.T) {
	l := New("<?= $name ?>")

	tests := []TokenType{Echo, Var, Semicolon, EOF}
	for _, want := range tests {
		if got := l.NextToken(); got.Type != want {
			t.Fatalf("got token %s (%q), want %s", got.Type, got.Literal, want)
		}
	}
}

func TestInlineHTMLTokenizesAsEchoString(t *testing.T) {
	l := New("Hello <?php echo $name; ?>!")

	tests := []struct {
		typ TokenType
		lit string
	}{
		{Echo, "echo"},
		{String, "Hello "},
		{Semicolon, ";"},
		{Echo, "echo"},
		{Var, "name"},
		{Semicolon, ";"},
		{Semicolon, ";"},
		{Echo, "echo"},
		{String, "!"},
		{Semicolon, ";"},
		{EOF, ""},
	}
	for _, want := range tests {
		got := l.NextToken()
		if got.Type != want.typ || got.Literal != want.lit {
			t.Fatalf("got token %s (%q), want %s (%q)", got.Type, got.Literal, want.typ, want.lit)
		}
	}
}

func TestVariableVariableToken(t *testing.T) {
	l := New("<?php $$name = 'value';")

	got := l.NextToken()
	if got.Type != VarVar || got.Literal != "name" {
		t.Fatalf("got token %s (%q), want VARVAR (name)", got.Type, got.Literal)
	}
}

func TestStringEscapes(t *testing.T) {
	tests := []struct {
		source      string
		want        string
		interpolate bool
	}{
		{`<?php "a\n\t\x41\101\\\"\$x"` + ";", "a\n\tAA\\\"\\$x", true},
		{`<?php 'it\'s \n \\ ok'` + ";", `it's \n \ ok`, false},
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
		if got.Interpolate != tt.interpolate {
			t.Fatalf("got interpolate %v, want %v", got.Interpolate, tt.interpolate)
		}
	}
}
