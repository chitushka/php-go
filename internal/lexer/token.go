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
	Dot             TokenType = "."
	Bang            TokenType = "!"
	Asterisk        TokenType = "*"
	Slash           TokenType = "/"
	AndAnd          TokenType = "&&"
	OrOr            TokenType = "||"
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
	Colon     TokenType = ":"
	LParen    TokenType = "("
	RParen    TokenType = ")"
	LBracket  TokenType = "["
	RBracket  TokenType = "]"
	LBrace    TokenType = "{"
	RBrace    TokenType = "}"

	Echo       TokenType = "ECHO"
	If         TokenType = "IF"
	Else       TokenType = "ELSE"
	While      TokenType = "WHILE"
	Do         TokenType = "DO"
	For        TokenType = "FOR"
	Foreach    TokenType = "FOREACH"
	Switch     TokenType = "SWITCH"
	Case       TokenType = "CASE"
	Default    TokenType = "DEFAULT"
	Break      TokenType = "BREAK"
	Continue   TokenType = "CONTINUE"
	As         TokenType = "AS"
	Array      TokenType = "ARRAY"
	Function   TokenType = "FUNCTION"
	Return     TokenType = "RETURN"
	Include    TokenType = "INCLUDE"
	Require    TokenType = "REQUIRE"
	Class      TokenType = "CLASS"
	Extends    TokenType = "EXTENDS"
	Interface  TokenType = "INTERFACE"
	Implements TokenType = "IMPLEMENTS"
	Public     TokenType = "PUBLIC"
	Private    TokenType = "PRIVATE"
	Protected  TokenType = "PROTECTED"
	Static     TokenType = "STATIC"
	Self       TokenType = "SELF"
	Parent     TokenType = "PARENT"
	Const      TokenType = "CONST"
	Final      TokenType = "FINAL"
	Abstract   TokenType = "ABSTRACT"
	NewKeyword TokenType = "NEW"
	InstanceOf TokenType = "INSTANCEOF"
	Try        TokenType = "TRY"
	Catch      TokenType = "CATCH"
	Throw      TokenType = "THROW"
	True       TokenType = "TRUE"
	False      TokenType = "FALSE"
	Null       TokenType = "NULL"
)

var keywords = map[string]TokenType{
	"echo":       Echo,
	"if":         If,
	"else":       Else,
	"while":      While,
	"do":         Do,
	"for":        For,
	"foreach":    Foreach,
	"switch":     Switch,
	"case":       Case,
	"default":    Default,
	"break":      Break,
	"continue":   Continue,
	"as":         As,
	"array":      Array,
	"function":   Function,
	"return":     Return,
	"include":    Include,
	"require":    Require,
	"class":      Class,
	"extends":    Extends,
	"interface":  Interface,
	"implements": Implements,
	"public":     Public,
	"private":    Private,
	"protected":  Protected,
	"static":     Static,
	"self":       Self,
	"parent":     Parent,
	"const":      Const,
	"final":      Final,
	"abstract":   Abstract,
	"new":        NewKeyword,
	"instanceof": InstanceOf,
	"try":        Try,
	"catch":      Catch,
	"throw":      Throw,
	"true":       True,
	"false":      False,
	"null":       Null,
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return Ident
}
