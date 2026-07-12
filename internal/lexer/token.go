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

	Assign          TokenType = "="
	Plus            TokenType = "+"
	Minus           TokenType = "-"
	Bang            TokenType = "!"
	Asterisk        TokenType = "*"
	Slash           TokenType = "/"
	ObjectOp        TokenType = "->"
	LT              TokenType = "<"
	GT              TokenType = ">"
	EQ              TokenType = "=="
	NotEQ           TokenType = "!="
	LTE             TokenType = "<="
	GTE             TokenType = ">="
	DoubleArrow     TokenType = "=>"
	ScopeResolution TokenType = "::"

	Comma     TokenType = ","
	Semicolon TokenType = ";"
	LParen    TokenType = "("
	RParen    TokenType = ")"
	LBrace    TokenType = "{"
	RBrace    TokenType = "}"

	Echo       TokenType = "ECHO"
	If         TokenType = "IF"
	Else       TokenType = "ELSE"
	While      TokenType = "WHILE"
	Foreach    TokenType = "FOREACH"
	As         TokenType = "AS"
	Array      TokenType = "ARRAY"
	Function   TokenType = "FUNCTION"
	Return     TokenType = "RETURN"
	Include    TokenType = "INCLUDE"
	Require    TokenType = "REQUIRE"
	Class      TokenType = "CLASS"
	Extends    TokenType = "EXTENDS"
	Public     TokenType = "PUBLIC"
	Private    TokenType = "PRIVATE"
	Protected  TokenType = "PROTECTED"
	Static     TokenType = "STATIC"
	NewKeyword TokenType = "NEW"
	Try        TokenType = "TRY"
	Catch      TokenType = "CATCH"
	Throw      TokenType = "THROW"
	True       TokenType = "TRUE"
	False      TokenType = "FALSE"
	Null       TokenType = "NULL"
)

var keywords = map[string]TokenType{
	"echo":      Echo,
	"if":        If,
	"else":      Else,
	"while":     While,
	"foreach":   Foreach,
	"as":        As,
	"array":     Array,
	"function":  Function,
	"return":    Return,
	"include":   Include,
	"require":   Require,
	"class":     Class,
	"extends":   Extends,
	"public":    Public,
	"private":   Private,
	"protected": Protected,
	"static":    Static,
	"new":       NewKeyword,
	"try":       Try,
	"catch":     Catch,
	"throw":     Throw,
	"true":      True,
	"false":     False,
	"null":      Null,
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return Ident
}
