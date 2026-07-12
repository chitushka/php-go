package ast

type Program struct{ Statements []Stmt }

type Stmt interface{ stmtNode() }
type Expr interface{ exprNode() }

type BlockStmt struct{ Statements []Stmt }

func (*BlockStmt) stmtNode() {}

type AssignStmt struct {
	Name  string
	Value Expr
}

func (*AssignStmt) stmtNode() {}

type EchoStmt struct{ Values []Expr }

func (*EchoStmt) stmtNode() {}

type IfStmt struct {
	Cond        Expr
	Consequence *BlockStmt
	Alternative *BlockStmt
}

func (*IfStmt) stmtNode() {}

type WhileStmt struct {
	Cond Expr
	Body *BlockStmt
}

func (*WhileStmt) stmtNode() {}

type ForeachStmt struct {
	Source   Expr
	KeyVar   string
	ValueVar string
	Body     *BlockStmt
}

func (*ForeachStmt) stmtNode() {}

type FunctionDeclStmt struct {
	Name   string
	Params []string
	Body   *BlockStmt
}

func (*FunctionDeclStmt) stmtNode() {}

type ReturnStmt struct {
	Value Expr
}

func (*ReturnStmt) stmtNode() {}

type ExprStmt struct{ Expr Expr }

func (*ExprStmt) stmtNode() {}

type Ident struct{ Name string }

func (*Ident) exprNode() {}

type IntLit struct{ Value int64 }

func (*IntLit) exprNode() {}

type FloatLit struct{ Value float64 }

func (*FloatLit) exprNode() {}

type StringLit struct{ Value string }

func (*StringLit) exprNode() {}

type BoolLit struct{ Value bool }

func (*BoolLit) exprNode() {}

type NullLit struct{}

func (*NullLit) exprNode() {}

type PrefixExpr struct {
	Op    string
	Right Expr
}

func (*PrefixExpr) exprNode() {}

type InfixExpr struct {
	Left  Expr
	Op    string
	Right Expr
}

func (*InfixExpr) exprNode() {}

type ArrayElement struct {
	Key   Expr
	Value Expr
}

type ArrayLit struct {
	Elements []ArrayElement
}

func (*ArrayLit) exprNode() {}

type CallExpr struct {
	Name string
	Args []Expr
}

func (*CallExpr) exprNode() {}
