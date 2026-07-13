package lexer

import "strings"

type Lexer struct {
	input        string
	position     int
	readPosition int
	ch           byte
	line         int
	column       int
}

func New(input string) *Lexer {
	input = strings.TrimPrefix(input, "<?php")
	l := &Lexer{input: input, line: 1}
	l.readChar()
	return l
}

func (l *Lexer) NextToken() Token {
	l.skipWhitespace()
	line, col := l.line, l.column
	var tok Token

	switch l.ch {
	case '=':
		if l.peekChar() == '>' {
			ch := l.ch
			l.readChar()
			tok = l.token(DoubleArrow, string(ch)+string(l.ch), line, col)
		} else if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = l.token(EQ, string(ch)+string(l.ch), line, col)
		} else {
			tok = l.token(Assign, string(l.ch), line, col)
		}
	case '!':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = l.token(NotEQ, string(ch)+string(l.ch), line, col)
		} else {
			tok = l.token(Bang, string(l.ch), line, col)
		}
	case '<':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = l.token(LTE, string(ch)+string(l.ch), line, col)
		} else {
			tok = l.token(LT, string(l.ch), line, col)
		}
	case '>':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = l.token(GTE, string(ch)+string(l.ch), line, col)
		} else {
			tok = l.token(GT, string(l.ch), line, col)
		}
	case '+':
		tok = l.token(Plus, string(l.ch), line, col)
	case '.':
		tok = l.token(Dot, string(l.ch), line, col)
	case '&':
		if l.peekChar() == '&' {
			ch := l.ch
			l.readChar()
			tok = l.token(AndAnd, string(ch)+string(l.ch), line, col)
		} else {
			tok = l.token(Illegal, string(l.ch), line, col)
		}
	case '|':
		if l.peekChar() == '|' {
			ch := l.ch
			l.readChar()
			tok = l.token(OrOr, string(ch)+string(l.ch), line, col)
		} else {
			tok = l.token(Illegal, string(l.ch), line, col)
		}
	case '-':
		if l.peekChar() == '>' {
			ch := l.ch
			l.readChar()
			tok = l.token(ObjectOp, string(ch)+string(l.ch), line, col)
		} else {
			tok = l.token(Minus, string(l.ch), line, col)
		}
	case '*':
		tok = l.token(Asterisk, string(l.ch), line, col)
	case '/':
		tok = l.token(Slash, string(l.ch), line, col)
	case ',':
		tok = l.token(Comma, string(l.ch), line, col)
	case ':':
		if l.peekChar() == ':' {
			ch := l.ch
			l.readChar()
			tok = l.token(ScopeResolution, string(ch)+string(l.ch), line, col)
		} else {
			tok = l.token(Illegal, string(l.ch), line, col)
		}
	case ';':
		tok = l.token(Semicolon, string(l.ch), line, col)
	case '(':
		tok = l.token(LParen, string(l.ch), line, col)
	case ')':
		tok = l.token(RParen, string(l.ch), line, col)
	case '[':
		tok = l.token(LBracket, string(l.ch), line, col)
	case ']':
		tok = l.token(RBracket, string(l.ch), line, col)
	case '{':
		tok = l.token(LBrace, string(l.ch), line, col)
	case '}':
		tok = l.token(RBrace, string(l.ch), line, col)
	case '$':
		l.readChar()
		lit := l.readIdentifier()
		return Token{Type: Var, Literal: lit, Line: line, Column: col}
	case '\'', '"':
		quote := l.ch
		lit := l.readString(quote)
		return Token{Type: String, Literal: lit, Line: line, Column: col}
	case 0:
		tok = l.token(EOF, "", line, col)
	default:
		if isLetter(l.ch) {
			lit := l.readIdentifier()
			return Token{Type: LookupIdent(lit), Literal: lit, Line: line, Column: col}
		}
		if isDigit(l.ch) {
			lit, typ := l.readNumber()
			return Token{Type: typ, Literal: lit, Line: line, Column: col}
		}
		tok = l.token(Illegal, string(l.ch), line, col)
	}
	l.readChar()
	return tok
}

func (l *Lexer) token(t TokenType, lit string, line, col int) Token {
	return Token{Type: t, Literal: lit, Line: line, Column: col}
}

func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPosition]
	}
	l.position = l.readPosition
	l.readPosition++
	if l.ch == '\n' {
		l.line++
		l.column = 0
	} else {
		l.column++
	}
}

func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\r' || l.ch == '\n' {
		l.readChar()
	}
}

func (l *Lexer) readIdentifier() string {
	pos := l.position
	for isLetter(l.ch) || isDigit(l.ch) {
		l.readChar()
	}
	return l.input[pos:l.position]
}

func (l *Lexer) readNumber() (string, TokenType) {
	pos := l.position
	typ := Int
	for isDigit(l.ch) {
		l.readChar()
	}
	if l.ch == '.' && isDigit(l.peekChar()) {
		typ = Float
		l.readChar()
		for isDigit(l.ch) {
			l.readChar()
		}
	}
	return l.input[pos:l.position], typ
}

func (l *Lexer) readString(quote byte) string {
	l.readChar()
	pos := l.position
	for l.ch != quote && l.ch != 0 {
		l.readChar()
	}
	lit := l.input[pos:l.position]
	l.readChar()
	return lit
}

func isLetter(ch byte) bool { return ('a' <= ch && ch <= 'z') || ('A' <= ch && ch <= 'Z') || ch == '_' }
func isDigit(ch byte) bool  { return '0' <= ch && ch <= '9' }
