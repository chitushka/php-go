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
	input = normalizeSource(input)
	l := &Lexer{input: input, line: 1}
	l.readChar()
	return l
}

func normalizeSource(input string) string {
	var out strings.Builder
	inPHP := false
	for input != "" {
		if !inPHP {
			openPHP := strings.Index(input, "<?php")
			openEcho := strings.Index(input, "<?=")
			if openPHP < 0 && openEcho < 0 {
				writeInlineHTML(&out, input)
				break
			}
			useEcho := openEcho >= 0 && (openPHP < 0 || openEcho < openPHP)
			open := openPHP
			tagLen := len("<?php")
			if useEcho {
				open = openEcho
				tagLen = len("<?=")
			}
			writeInlineHTML(&out, input[:open])
			input = input[open+tagLen:]
			if useEcho {
				out.WriteString(" echo ")
			}
			inPHP = true
			continue
		}

		close := strings.Index(input, "?>")
		if close < 0 {
			out.WriteString(input)
			break
		}
		out.WriteString(input[:close])
		out.WriteString(";")
		input = input[close+len("?>"):]
		inPHP = false
	}
	return out.String()
}

func writeInlineHTML(out *strings.Builder, text string) {
	if text == "" {
		return
	}
	out.WriteString(" echo '")
	for idx := 0; idx < len(text); idx++ {
		switch text[idx] {
		case '\\', '\'':
			out.WriteByte('\\')
		}
		out.WriteByte(text[idx])
	}
	out.WriteString("';")
}

func (l *Lexer) NextToken() Token {
	l.skipIgnored()
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
			if l.peekChar() == '=' {
				l.readChar()
				tok = l.token(StrictEQ, string(ch)+"==", line, col)
			} else {
				tok = l.token(EQ, string(ch)+string(l.ch), line, col)
			}
		} else {
			tok = l.token(Assign, string(l.ch), line, col)
		}
	case '!':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			if l.peekChar() == '=' {
				l.readChar()
				tok = l.token(StrictNotEQ, string(ch)+"==", line, col)
			} else {
				tok = l.token(NotEQ, string(ch)+string(l.ch), line, col)
			}
		} else {
			tok = l.token(Bang, string(l.ch), line, col)
		}
	case '<':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = l.token(LTE, string(ch)+string(l.ch), line, col)
		} else if l.peekChar() == '<' {
			ch := l.ch
			l.readChar()
			tok = l.token(ShiftLeft, string(ch)+string(l.ch), line, col)
		} else if l.peekChar() == '>' {
			ch := l.ch
			l.readChar()
			tok = l.token(NotEQ, string(ch)+string(l.ch), line, col)
		} else {
			tok = l.token(LT, string(l.ch), line, col)
		}
	case '>':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = l.token(GTE, string(ch)+string(l.ch), line, col)
		} else if l.peekChar() == '>' {
			ch := l.ch
			l.readChar()
			tok = l.token(ShiftRight, string(ch)+string(l.ch), line, col)
		} else {
			tok = l.token(GT, string(l.ch), line, col)
		}
	case '+':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = l.token(PlusAssign, string(ch)+string(l.ch), line, col)
		} else if l.peekChar() == '+' {
			ch := l.ch
			l.readChar()
			tok = l.token(Increment, string(ch)+string(l.ch), line, col)
		} else {
			tok = l.token(Plus, string(l.ch), line, col)
		}
	case '.':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = l.token(ConcatAssign, string(ch)+string(l.ch), line, col)
		} else {
			tok = l.token(Dot, string(l.ch), line, col)
		}
	case '&':
		if l.peekChar() == '&' {
			ch := l.ch
			l.readChar()
			tok = l.token(AndAnd, string(ch)+string(l.ch), line, col)
		} else {
			tok = l.token(Ampersand, string(l.ch), line, col)
		}
	case '|':
		if l.peekChar() == '|' {
			ch := l.ch
			l.readChar()
			tok = l.token(OrOr, string(ch)+string(l.ch), line, col)
		} else {
			tok = l.token(Pipe, string(l.ch), line, col)
		}
	case '^':
		tok = l.token(Caret, string(l.ch), line, col)
	case '~':
		tok = l.token(Tilde, string(l.ch), line, col)
	case '-':
		if l.peekChar() == '>' {
			ch := l.ch
			l.readChar()
			tok = l.token(ObjectOp, string(ch)+string(l.ch), line, col)
		} else if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = l.token(MinusAssign, string(ch)+string(l.ch), line, col)
		} else if l.peekChar() == '-' {
			ch := l.ch
			l.readChar()
			tok = l.token(Decrement, string(ch)+string(l.ch), line, col)
		} else {
			tok = l.token(Minus, string(l.ch), line, col)
		}
	case '*':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = l.token(MulAssign, string(ch)+string(l.ch), line, col)
		} else {
			tok = l.token(Asterisk, string(l.ch), line, col)
		}
	case '/':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = l.token(DivAssign, string(ch)+string(l.ch), line, col)
		} else {
			tok = l.token(Slash, string(l.ch), line, col)
		}
	case '%':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = l.token(ModAssign, string(ch)+string(l.ch), line, col)
		} else {
			tok = l.token(Percent, string(l.ch), line, col)
		}
	case '?':
		if l.peekChar() == '>' {
			l.readChar()
			tok = l.token(EOF, "", line, col)
		} else {
			tok = l.token(Question, string(l.ch), line, col)
		}
	case ',':
		tok = l.token(Comma, string(l.ch), line, col)
	case ':':
		if l.peekChar() == ':' {
			ch := l.ch
			l.readChar()
			tok = l.token(ScopeResolution, string(ch)+string(l.ch), line, col)
		} else {
			tok = l.token(Colon, string(l.ch), line, col)
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

func (l *Lexer) skipIgnored() {
	for {
		for l.ch == ' ' || l.ch == '\t' || l.ch == '\r' || l.ch == '\n' {
			l.readChar()
		}
		if l.ch == '#' {
			l.skipLineComment()
			continue
		}
		if l.ch == '/' && l.peekChar() == '/' {
			l.skipLineComment()
			continue
		}
		if l.ch == '/' && l.peekChar() == '*' {
			l.skipBlockComment()
			continue
		}
		return
	}
}

func (l *Lexer) skipLineComment() {
	for l.ch != '\n' && l.ch != 0 {
		l.readChar()
	}
}

func (l *Lexer) skipBlockComment() {
	l.readChar()
	l.readChar()
	for l.ch != 0 {
		if l.ch == '*' && l.peekChar() == '/' {
			l.readChar()
			l.readChar()
			return
		}
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
	var b strings.Builder
	for l.ch != quote && l.ch != 0 {
		if l.ch != '\\' {
			b.WriteByte(l.ch)
			l.readChar()
			continue
		}
		l.readChar()
		if l.ch == 0 {
			b.WriteByte('\\')
			break
		}
		if quote == '\'' {
			switch l.ch {
			case '\\', '\'':
				b.WriteByte(l.ch)
			default:
				b.WriteByte('\\')
				b.WriteByte(l.ch)
			}
			l.readChar()
			continue
		}
		switch l.ch {
		case 'n':
			b.WriteByte('\n')
			l.readChar()
		case 'r':
			b.WriteByte('\r')
			l.readChar()
		case 't':
			b.WriteByte('\t')
			l.readChar()
		case 'v':
			b.WriteByte('\v')
			l.readChar()
		case 'e':
			b.WriteByte(0x1b)
			l.readChar()
		case 'f':
			b.WriteByte('\f')
			l.readChar()
		case '\\', '$', '"':
			b.WriteByte(l.ch)
			l.readChar()
		case 'x', 'X':
			l.readChar()
			value, digits := 0, 0
			for digits < 2 && isHexDigit(l.ch) {
				value = value*16 + hexValue(l.ch)
				digits++
				l.readChar()
			}
			if digits == 0 {
				b.WriteString("\\x")
			} else {
				b.WriteByte(byte(value))
			}
		default:
			if isOctalDigit(l.ch) {
				value, digits := 0, 0
				for digits < 3 && isOctalDigit(l.ch) {
					value = value*8 + int(l.ch-'0')
					digits++
					l.readChar()
				}
				b.WriteByte(byte(value))
				continue
			}
			b.WriteByte('\\')
			b.WriteByte(l.ch)
			l.readChar()
		}
	}
	if l.ch == quote {
		l.readChar()
	}
	return b.String()
}

func isLetter(ch byte) bool { return ('a' <= ch && ch <= 'z') || ('A' <= ch && ch <= 'Z') || ch == '_' }
func isDigit(ch byte) bool  { return '0' <= ch && ch <= '9' }

func isOctalDigit(ch byte) bool { return '0' <= ch && ch <= '7' }

func isHexDigit(ch byte) bool {
	return isDigit(ch) || ('a' <= ch && ch <= 'f') || ('A' <= ch && ch <= 'F')
}

func hexValue(ch byte) int {
	switch {
	case '0' <= ch && ch <= '9':
		return int(ch - '0')
	case 'a' <= ch && ch <= 'f':
		return int(ch-'a') + 10
	default:
		return int(ch-'A') + 10
	}
}
