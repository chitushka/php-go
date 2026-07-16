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

type CompoundAssignStmt struct {
	Target Expr
	Op     string
	Value  Expr
}

func (*CompoundAssignStmt) stmtNode() {}

type IncDecStmt struct {
	Target Expr
	Op     string
}

func (*IncDecStmt) stmtNode() {}

type EchoStmt struct{ Values []Expr }

func (*EchoStmt) stmtNode() {}

type UnsetStmt struct{ Targets []Expr }

func (*UnsetStmt) stmtNode() {}

type ExitStmt struct{ Value Expr }

func (*ExitStmt) stmtNode() {}

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

type DoWhileStmt struct {
	Body *BlockStmt
	Cond Expr
}

func (*DoWhileStmt) stmtNode() {}

type ForStmt struct {
	Init Stmt
	Cond Expr
	Post Stmt
	Body *BlockStmt
}

func (*ForStmt) stmtNode() {}

type ForeachStmt struct {
	Source     Expr
	KeyVar     string
	ValueVar   string
	ValueByRef bool
	Body       *BlockStmt
}

func (*ForeachStmt) stmtNode() {}

type SwitchStmt struct {
	Value Expr
	Cases []SwitchCase
}

func (*SwitchStmt) stmtNode() {}

type SwitchCase struct {
	Expr       Expr
	Statements []Stmt
	Default    bool
}

type BreakStmt struct {
	Level int
}

func (*BreakStmt) stmtNode() {}

type ContinueStmt struct {
	Level int
}

func (*ContinueStmt) stmtNode() {}

type FunctionDeclStmt struct {
	Name         string
	Params       []Param
	Body         *BlockStmt
	Visibility   string
	Static       bool
	Owner        string
	Final        bool
	Abstract     bool
	ReturnsByRef bool
}

func (*FunctionDeclStmt) stmtNode() {}

type Param struct {
	Name    string
	Default Expr
	ByRef   bool
}

type ClosureUse struct {
	Name  string
	ByRef bool
}

type ReturnStmt struct {
	Value Expr
}

func (*ReturnStmt) stmtNode() {}

type GlobalStmt struct {
	Names []string
}

func (*GlobalStmt) stmtNode() {}

type StaticLocalStmt struct {
	Decls []StaticLocalDecl
}

func (*StaticLocalStmt) stmtNode() {}

type StaticLocalDecl struct {
	Name    string
	Default Expr
}

type IncludeStmt struct {
	Path    Expr
	Require bool
	Once    bool
}

func (*IncludeStmt) stmtNode() {}

type ClassDeclStmt struct {
	Name       string
	Parent     string
	Implements []string
	Properties []*ClassProperty
	Constants  []*ClassConstant
	Methods    []*FunctionDeclStmt
	Final      bool
	Abstract   bool
}

func (*ClassDeclStmt) stmtNode() {}

type InterfaceDeclStmt struct {
	Name      string
	Parents   []string
	Constants []*ClassConstant
	Methods   []*FunctionDeclStmt
}

func (*InterfaceDeclStmt) stmtNode() {}

type ClassProperty struct {
	Name       string
	Visibility string
	Static     bool
	Final      bool
	Abstract   bool
	Default    Expr
	Owner      string
}

type ClassConstant struct {
	Name  string
	Value Expr
	Owner string
}

type PropertyAssignStmt struct {
	Object   Expr
	Property string
	Value    Expr
}

func (*PropertyAssignStmt) stmtNode() {}

type StaticPropertyAssignStmt struct {
	ClassName string
	Property  string
	Value     Expr
}

func (*StaticPropertyAssignStmt) stmtNode() {}

type IndexAssignStmt struct {
	Target *IndexExpr
	Value  Expr
}

func (*IndexAssignStmt) stmtNode() {}

type ThrowStmt struct {
	Value Expr
}

func (*ThrowStmt) stmtNode() {}

type TryCatchStmt struct {
	TryBlock *BlockStmt
	Catches  []CatchClause
}

type CatchClause struct {
	Class string
	Var   string
	Block *BlockStmt
}

func (*TryCatchStmt) stmtNode() {}

type ExprStmt struct{ Expr Expr }

func (*ExprStmt) stmtNode() {}

type Ident struct{ Name string }

func (*Ident) exprNode() {}

type VariableVariable struct{ Name string }

func (*VariableVariable) exprNode() {}

type IntLit struct{ Value int64 }

func (*IntLit) exprNode() {}

type FloatLit struct{ Value float64 }

func (*FloatLit) exprNode() {}

type StringLit struct {
	Value       string
	Interpolate bool
	Backtick    bool
}

func (*StringLit) exprNode() {}

type BoolLit struct{ Value bool }

func (*BoolLit) exprNode() {}

type NullLit struct{}

func (*NullLit) exprNode() {}

type MagicConstExpr struct {
	Name string
	Line int
}

func (*MagicConstExpr) exprNode() {}

type BareConstExpr struct {
	Name string
}

func (*BareConstExpr) exprNode() {}

type PrefixExpr struct {
	Op    string
	Right Expr
}

func (*PrefixExpr) exprNode() {}

type CloneExpr struct {
	Expr Expr
}

func (*CloneExpr) exprNode() {}

type IncDecExpr struct {
	Target Expr
	Op     string
	Post   bool
}

func (*IncDecExpr) exprNode() {}

type InfixExpr struct {
	Left  Expr
	Op    string
	Right Expr
}

func (*InfixExpr) exprNode() {}

type TernaryExpr struct {
	Cond Expr
	Then Expr
	Else Expr
}

func (*TernaryExpr) exprNode() {}

type AssignExpr struct {
	Target Expr
	Op     string
	Value  Expr
	ByRef  bool
}

func (*AssignExpr) exprNode() {}

type ListExpr struct {
	Targets []Expr
}

func (*ListExpr) exprNode() {}

type CastExpr struct {
	Type string
	Expr Expr
}

func (*CastExpr) exprNode() {}

type ArrayElement struct {
	Key   Expr
	Value Expr
}

type ArrayLit struct {
	Elements []ArrayElement
}

func (*ArrayLit) exprNode() {}

type IndexExpr struct {
	Left  Expr
	Index Expr
}

func (*IndexExpr) exprNode() {}

type CallExpr struct {
	Name string
	Args []Expr
}

func (*CallExpr) exprNode() {}

type VariableCallExpr struct {
	Callee Expr
	Args   []Expr
}

func (*VariableCallExpr) exprNode() {}

type ClosureExpr struct {
	Params       []Param
	Uses         []ClosureUse
	Body         *BlockStmt
	ReturnsByRef bool
}

func (*ClosureExpr) exprNode() {}

type NewExpr struct {
	ClassName string
	ClassExpr Expr
	Args      []Expr
	Line      int
}

func (*NewExpr) exprNode() {}

type PropertyFetchExpr struct {
	Object   Expr
	Property string
}

func (*PropertyFetchExpr) exprNode() {}

type DynamicPropertyFetchExpr struct {
	Object   Expr
	Property Expr
}

func (*DynamicPropertyFetchExpr) exprNode() {}

type MethodCallExpr struct {
	Object Expr
	Method string
	Args   []Expr
}

func (*MethodCallExpr) exprNode() {}

type DynamicMethodCallExpr struct {
	Object Expr
	Method Expr
	Args   []Expr
}

func (*DynamicMethodCallExpr) exprNode() {}

type StaticPropertyFetchExpr struct {
	ClassName string
	Property  string
}

func (*StaticPropertyFetchExpr) exprNode() {}

type DynamicStaticPropertyFetchExpr struct {
	ClassName string
	Property  Expr
}

func (*DynamicStaticPropertyFetchExpr) exprNode() {}

type ClassConstFetchExpr struct {
	ClassName string
	Name      string
}

func (*ClassConstFetchExpr) exprNode() {}

type StaticCallExpr struct {
	ClassName string
	Method    string
	Args      []Expr
}

func (*StaticCallExpr) exprNode() {}

type DynamicStaticCallExpr struct {
	ClassName string
	Method    Expr
	Args      []Expr
}

func (*DynamicStaticCallExpr) exprNode() {}

type InstanceOfExpr struct {
	Left      Expr
	ClassName string
	ClassExpr Expr
}

func (*InstanceOfExpr) exprNode() {}
