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
	Percent         TokenType = "%"
	Ampersand       TokenType = "&"
	Pipe            TokenType = "|"
	Caret           TokenType = "^"
	Tilde           TokenType = "~"
	ShiftLeft       TokenType = "<<"
	ShiftRight      TokenType = ">>"
	AndAnd          TokenType = "&&"
	OrOr            TokenType = "||"
	Question        TokenType = "?"
	ObjectOp        TokenType = "->"
	LT              TokenType = "<"
	GT              TokenType = ">"
	EQ              TokenType = "=="
	NotEQ           TokenType = "!="
	StrictEQ        TokenType = "==="
	StrictNotEQ     TokenType = "!=="
	LTE             TokenType = "<="
	GTE             TokenType = ">="
	PlusAssign      TokenType = "+="
	MinusAssign     TokenType = "-="
	MulAssign       TokenType = "*="
	DivAssign       TokenType = "/="
	ConcatAssign    TokenType = ".="
	ModAssign       TokenType = "%="
	Increment       TokenType = "++"
	Decrement       TokenType = "--"
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
	Print      TokenType = "PRINT"
	Exit       TokenType = "EXIT"
	If         TokenType = "IF"
	ElseIf     TokenType = "ELSEIF"
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
	Global     TokenType = "GLOBAL"
	Unset      TokenType = "UNSET"
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
	AndWord    TokenType = "AND"
	OrWord     TokenType = "OR"
	XorWord    TokenType = "XOR"
	Try        TokenType = "TRY"
	Catch      TokenType = "CATCH"
	Throw      TokenType = "THROW"
	True       TokenType = "TRUE"
	False      TokenType = "FALSE"
	Null       TokenType = "NULL"
)

var keywords = map[string]TokenType{
	"echo":       Echo,
	"print":      Print,
	"exit":       Exit,
	"die":        Exit,
	"if":         If,
	"elseif":     ElseIf,
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
	"global":     Global,
	"unset":      Unset,
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
	"and":        AndWord,
	"or":         OrWord,
	"xor":        XorWord,
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
