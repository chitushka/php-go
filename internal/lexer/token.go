package lexer

type TokenType string

type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Column  int
}

const (
	Illegal TokenType = "ILLEGAL"
	EOF     TokenType = "EOF"

	Ident  TokenType = "IDENT"
	Var    TokenType = "VAR"
	Int    TokenType = "INT"
	Float  TokenType = "FLOAT"
	String TokenType = "STRING"

	Assign   TokenType = "="
	Plus     TokenType = "+"
	Minus    TokenType = "-"
	Bang     TokenType = "!"
	Asterisk TokenType = "*"
	Slash    TokenType = "/"
	LT       TokenType = "<"
	GT       TokenType = ">"
	EQ       TokenType = "=="
	NotEQ    TokenType = "!="
	LTE      TokenType = "<="
	GTE      TokenType = ">="

	Comma     TokenType = ","
	Semicolon TokenType = ";"
	LParen    TokenType = "("
	RParen    TokenType = ")"
	LBrace    TokenType = "{"
	RBrace    TokenType = "}"

	Echo  TokenType = "ECHO"
	If    TokenType = "IF"
	Else  TokenType = "ELSE"
	While TokenType = "WHILE"
	True  TokenType = "TRUE"
	False TokenType = "FALSE"
	Null  TokenType = "NULL"
)

var keywords = map[string]TokenType{
	"echo":  Echo,
	"if":    If,
	"else":  Else,
	"while": While,
	"true":  True,
	"false": False,
	"null":  Null,
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return Ident
}
