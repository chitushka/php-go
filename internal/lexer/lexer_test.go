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

func TestCloneKeywordToken(t *testing.T) {
	l := New("<?php $b = clone $a;")

	for _, want := range []TokenType{Var, Assign, Clone, Var, Semicolon, EOF} {
		got := l.NextToken()
		if got.Type != want {
			t.Fatalf("got token %s (%q), want %s", got.Type, got.Literal, want)
		}
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

func TestNumberLiteralForms(t *testing.T) {
	tests := []struct {
		source string
		typ    TokenType
		lit    string
	}{
		{"<?php 0x1A;", Int, "0x1A"},
		{"<?php 0X1a;", Int, "0X1a"},
		{"<?php 017;", Int, "017"},
		{"<?php 1e3;", Float, "1e3"},
		{"<?php 1E3;", Float, "1E3"},
		{"<?php 1.5e-3;", Float, "1.5e-3"},
		{"<?php 1.5e+3;", Float, "1.5e+3"},
		{"<?php .5;", Float, ".5"},
		{"<?php 5.;", Float, "5."},
		{"<?php 10;", Int, "10"},
	}
	for _, tt := range tests {
		l := New(tt.source)
		got := l.NextToken()
		if got.Type != tt.typ || got.Literal != tt.lit {
			t.Fatalf("source %q: got token %s (%q), want %s (%q)", tt.source, got.Type, got.Literal, tt.typ, tt.lit)
		}
	}
}

func TestDotOperatorNotConfusedWithFloatLiteral(t *testing.T) {
	l := New("<?php 1 . 2;")
	tests := []struct {
		typ TokenType
		lit string
	}{
		{Int, "1"},
		{Dot, "."},
		{Int, "2"},
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

func TestHeredocAndNowdocTokens(t *testing.T) {
	l := New("<?php <<<TXT\nHello $name\nTXT;\n<<<'RAW'\nRaw $name\nRAW;")

	// PHP's heredoc/nowdoc content excludes the single newline immediately
	// before the closing label line.
	got := l.NextToken()
	if got.Type != String || got.Literal != "Hello $name" || !got.Interpolate {
		t.Fatalf("got heredoc token %s (%q) interpolate=%v", got.Type, got.Literal, got.Interpolate)
	}
	if semi := l.NextToken(); semi.Type != Semicolon {
		t.Fatalf("got token %s, want semicolon", semi.Type)
	}
	got = l.NextToken()
	if got.Type != String || got.Literal != "Raw $name" || got.Interpolate {
		t.Fatalf("got nowdoc token %s (%q) interpolate=%v", got.Type, got.Literal, got.Interpolate)
	}
}

func TestHeredocEscapeSequences(t *testing.T) {
	l := New("<?php <<<TXT\nA\\tB\\n\\\\\\$name\\x41\nTXT;\n<<<'RAW'\nA\\tB\nRAW;")

	got := l.NextToken()
	want := "A\tB\n\\\\$name" + "A"
	if got.Type != String || got.Literal != want || !got.Interpolate {
		t.Fatalf("got heredoc token %s (%q) interpolate=%v, want %q", got.Type, got.Literal, got.Interpolate, want)
	}
	if semi := l.NextToken(); semi.Type != Semicolon {
		t.Fatalf("got token %s, want semicolon", semi.Type)
	}
	got = l.NextToken()
	if got.Type != String || got.Literal != `A\tB` || got.Interpolate {
		t.Fatalf("got nowdoc token %s (%q) interpolate=%v, want raw escapes untouched", got.Type, got.Literal, got.Interpolate)
	}
}
