package lexer

type TokenType string

type Token struct {
	Type        TokenType
	Literal     string
	Line        int
	Column      int
	Interpolate bool
	Backtick    bool
}

const (
	Illegal TokenType = "ILLEGAL"
	EOF     TokenType = "EOF"

	Ident  TokenType = "IDENT"
	Var    TokenType = "VAR"
	VarVar TokenType = "VARVAR"
	Int    TokenType = "INT"
	Float  TokenType = "FLOAT"
	String TokenType = "STRING"

	Assign          TokenType = "="
	AssignRef       TokenType = "=&"
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

	Echo        TokenType = "ECHO"
	Print       TokenType = "PRINT"
	Exit        TokenType = "EXIT"
	If          TokenType = "IF"
	ElseIf      TokenType = "ELSEIF"
	Else        TokenType = "ELSE"
	EndIf       TokenType = "ENDIF"
	While       TokenType = "WHILE"
	EndWhile    TokenType = "ENDWHILE"
	Do          TokenType = "DO"
	For         TokenType = "FOR"
	EndFor      TokenType = "ENDFOR"
	Foreach     TokenType = "FOREACH"
	EndForeach  TokenType = "ENDFOREACH"
	Switch      TokenType = "SWITCH"
	EndSwitch   TokenType = "ENDSWITCH"
	Case        TokenType = "CASE"
	Default     TokenType = "DEFAULT"
	Break       TokenType = "BREAK"
	Continue    TokenType = "CONTINUE"
	As          TokenType = "AS"
	Array       TokenType = "ARRAY"
	List        TokenType = "LIST"
	Function    TokenType = "FUNCTION"
	Return      TokenType = "RETURN"
	Global      TokenType = "GLOBAL"
	Unset       TokenType = "UNSET"
	Include     TokenType = "INCLUDE"
	Require     TokenType = "REQUIRE"
	IncludeOnce TokenType = "INCLUDE_ONCE"
	RequireOnce TokenType = "REQUIRE_ONCE"
	Class       TokenType = "CLASS"
	Extends     TokenType = "EXTENDS"
	Interface   TokenType = "INTERFACE"
	Implements  TokenType = "IMPLEMENTS"
	Public      TokenType = "PUBLIC"
	Private     TokenType = "PRIVATE"
	Protected   TokenType = "PROTECTED"
	Static      TokenType = "STATIC"
	Self        TokenType = "SELF"
	Parent      TokenType = "PARENT"
	Const       TokenType = "CONST"
	Final       TokenType = "FINAL"
	Abstract    TokenType = "ABSTRACT"
	NewKeyword  TokenType = "NEW"
	InstanceOf  TokenType = "INSTANCEOF"
	AndWord     TokenType = "AND"
	OrWord      TokenType = "OR"
	XorWord     TokenType = "XOR"
	Try         TokenType = "TRY"
	Catch       TokenType = "CATCH"
	Throw       TokenType = "THROW"
	True        TokenType = "TRUE"
	False       TokenType = "FALSE"
	Null        TokenType = "NULL"
)

var keywords = map[string]TokenType{
	"echo":         Echo,
	"print":        Print,
	"exit":         Exit,
	"die":          Exit,
	"if":           If,
	"elseif":       ElseIf,
	"else":         Else,
	"endif":        EndIf,
	"while":        While,
	"endwhile":     EndWhile,
	"do":           Do,
	"for":          For,
	"endfor":       EndFor,
	"foreach":      Foreach,
	"endforeach":   EndForeach,
	"switch":       Switch,
	"endswitch":    EndSwitch,
	"case":         Case,
	"default":      Default,
	"break":        Break,
	"continue":     Continue,
	"as":           As,
	"array":        Array,
	"list":         List,
	"function":     Function,
	"return":       Return,
	"global":       Global,
	"unset":        Unset,
	"include":      Include,
	"require":      Require,
	"include_once": IncludeOnce,
	"require_once": RequireOnce,
	"class":        Class,
	"extends":      Extends,
	"interface":    Interface,
	"implements":   Implements,
	"public":       Public,
	"private":      Private,
	"protected":    Protected,
	"static":       Static,
	"self":         Self,
	"parent":       Parent,
	"const":        Const,
	"final":        Final,
	"abstract":     Abstract,
	"new":          NewKeyword,
	"instanceof":   InstanceOf,
	"and":          AndWord,
	"or":           OrWord,
	"xor":          XorWord,
	"try":          Try,
	"catch":        Catch,
	"throw":        Throw,
	"true":         True,
	"false":        False,
	"null":         Null,
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return Ident
}
