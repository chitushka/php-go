package parser_test

import (
	"fmt"
	"testing"

	"github.com/chitushka/php-go/internal/ast"
	"github.com/chitushka/php-go/internal/lexer"
	"github.com/chitushka/php-go/internal/parser"
)

// parseProgram parses src (which must include its own <?php tag) and fails
// the test immediately if the parser reported any errors.
func parseProgram(t *testing.T, src string) *ast.Program {
	t.Helper()
	p := parser.New(lexer.New(src))
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parse errors for %q: %v", src, p.Errors())
	}
	return program
}

// parseExpr parses a single expression statement and returns its Expr node.
func parseExpr(t *testing.T, src string) ast.Expr {
	t.Helper()
	program := parseProgram(t, "<?php "+src+";")
	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement for %q, got %d", src, len(program.Statements))
	}
	stmt, ok := program.Statements[0].(*ast.ExprStmt)
	if !ok {
		t.Fatalf("expected *ast.ExprStmt for %q, got %T", src, program.Statements[0])
	}
	return stmt.Expr
}

// stringifyExpr renders an expression tree as a fully parenthesized string,
// so precedence/associativity can be checked by comparing readable strings
// instead of hand-building nested AST literals for every case.
func stringifyExpr(e ast.Expr) string {
	switch n := e.(type) {
	case *ast.IntLit:
		return fmt.Sprintf("%d", n.Value)
	case *ast.FloatLit:
		return fmt.Sprintf("%g", n.Value)
	case *ast.StringLit:
		return fmt.Sprintf("%q", n.Value)
	case *ast.BoolLit:
		return fmt.Sprintf("%t", n.Value)
	case *ast.NullLit:
		return "null"
	case *ast.Ident:
		return "$" + n.Name
	case *ast.BareConstExpr:
		return n.Name
	case *ast.PrefixExpr:
		return fmt.Sprintf("(%s%s)", n.Op, stringifyExpr(n.Right))
	case *ast.IncDecExpr:
		if n.Post {
			return fmt.Sprintf("(%s%s)", stringifyExpr(n.Target), n.Op)
		}
		return fmt.Sprintf("(%s%s)", n.Op, stringifyExpr(n.Target))
	case *ast.InfixExpr:
		return fmt.Sprintf("(%s %s %s)", stringifyExpr(n.Left), n.Op, stringifyExpr(n.Right))
	case *ast.TernaryExpr:
		return fmt.Sprintf("(%s ? %s : %s)", stringifyExpr(n.Cond), stringifyExpr(n.Then), stringifyExpr(n.Else))
	case *ast.AssignExpr:
		if n.ByRef {
			return fmt.Sprintf("(%s =& %s)", stringifyExpr(n.Target), stringifyExpr(n.Value))
		}
		if n.Op == "=" {
			return fmt.Sprintf("(%s = %s)", stringifyExpr(n.Target), stringifyExpr(n.Value))
		}
		return fmt.Sprintf("(%s %s= %s)", stringifyExpr(n.Target), n.Op, stringifyExpr(n.Value))
	case *ast.InstanceOfExpr:
		if n.ClassExpr != nil {
			return fmt.Sprintf("(%s instanceof %s)", stringifyExpr(n.Left), stringifyExpr(n.ClassExpr))
		}
		return fmt.Sprintf("(%s instanceof %s)", stringifyExpr(n.Left), n.ClassName)
	default:
		return fmt.Sprintf("<%T>", e)
	}
}

func TestExpressionPrecedenceAndAssociativity(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		// Arithmetic: * / % bind tighter than + - ., all left-associative.
		{"1 + 2 * 3", "(1 + (2 * 3))"},
		{"1 * 2 + 3", "((1 * 2) + 3)"},
		{"1 + 2 + 3", "((1 + 2) + 3)"},
		{"2 - 3 - 4", "((2 - 3) - 4)"},
		{"10 - 2 * 3", "(10 - (2 * 3))"},
		{"10 / 2 / 5", "((10 / 2) / 5)"},
		{"1 . 2 + 3", "((1 . 2) + 3)"},
		{"1 + 2 . 3", "((1 + 2) . 3)"},

		// Unary prefix binds tighter than any infix operator.
		{"-1 * 2", "((-1) * 2)"},
		{"-1 + 2", "((-1) + 2)"},
		{"!true && false", "((!true) && false)"},
		{"-$a + $b", "((-$a) + $b)"},

		// Comparisons bind tighter than && / ||.
		{"1 < 2 && 2 < 3", "((1 < 2) && (2 < 3))"},
		{"1 == 1 || 2 == 3", "((1 == 1) || (2 == 3))"},

		// && binds tighter than ||.
		{"true || false && false", "(true || (false && false))"},
		{"true && false || true", "((true && false) || true)"},

		// Bitwise: & tighter than ^ tighter than |.
		{"1 | 2 ^ 3", "(1 | (2 ^ 3))"},
		{"1 ^ 2 & 3", "(1 ^ (2 & 3))"},
		{"1 & 2 | 3", "((1 & 2) | 3)"},

		// Shift binds looser than +/-, tighter than comparisons.
		{"1 << 2 + 3", "(1 << (2 + 3))"},
		{"1 + 2 << 3", "((1 + 2) << 3)"},
		{"1 << 2 < 3", "((1 << 2) < 3)"},

		// Ternary.
		{"true ? 1 : 2", "(true ? 1 : 2)"},
		{"1 < 2 ? 3 : 4", "((1 < 2) ? 3 : 4)"},
		{"1 < 2 ? 3 : 4 < 5", "((1 < 2) ? 3 : (4 < 5))"},

		// Assignment is right-associative.
		{"$a = $b = 5", "($a = ($b = 5))"},
		{"$a = 1 + 2", "($a = (1 + 2))"},
		{"$a += 1 + 2", "($a += (1 + 2))"},

		// Classic PHP gotcha: `and`/`or`/`xor` bind weaker than `=`, but
		// `&&`/`||` bind tighter than `=`.
		{"$a = true and false", "(($a = true) and false)"},
		{"$a = true && false", "($a = (true && false))"},
		{"$a = true or false", "(($a = true) or false)"},
		{"true and false or true", "((true and false) or true)"},

		// instanceof binds like a comparison operator.
		{"$a instanceof Foo && $b", "(($a instanceof Foo) && $b)"},

		// Postfix/prefix increment.
		{"$a++ + 1", "(($a++) + 1)"},
		{"++$a + 1", "((++$a) + 1)"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			expr := parseExpr(t, tt.input)
			if got := stringifyExpr(expr); got != tt.want {
				t.Fatalf("stringifyExpr(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestIfElseAstShape(t *testing.T) {
	program := parseProgram(t, `<?php if ($a > 1) { echo "yes"; } else { echo "no"; }`)
	stmt, ok := program.Statements[0].(*ast.IfStmt)
	if !ok {
		t.Fatalf("expected *ast.IfStmt, got %T", program.Statements[0])
	}
	if stringifyExpr(stmt.Cond) != "($a > 1)" {
		t.Fatalf("unexpected condition: %s", stringifyExpr(stmt.Cond))
	}
	if len(stmt.Consequence.Statements) != 1 {
		t.Fatalf("expected 1 consequence statement, got %d", len(stmt.Consequence.Statements))
	}
	if stmt.Alternative == nil || len(stmt.Alternative.Statements) != 1 {
		t.Fatalf("expected alternative block with 1 statement, got %#v", stmt.Alternative)
	}
}

func TestWhileAstShape(t *testing.T) {
	program := parseProgram(t, `<?php while ($i < 10) { $i++; }`)
	stmt, ok := program.Statements[0].(*ast.WhileStmt)
	if !ok {
		t.Fatalf("expected *ast.WhileStmt, got %T", program.Statements[0])
	}
	if stringifyExpr(stmt.Cond) != "($i < 10)" {
		t.Fatalf("unexpected condition: %s", stringifyExpr(stmt.Cond))
	}
	if len(stmt.Body.Statements) != 1 {
		t.Fatalf("expected 1 body statement, got %d", len(stmt.Body.Statements))
	}
}

func TestForAstShape(t *testing.T) {
	program := parseProgram(t, `<?php for ($i = 0; $i < 10; $i++) { echo $i; }`)
	stmt, ok := program.Statements[0].(*ast.ForStmt)
	if !ok {
		t.Fatalf("expected *ast.ForStmt, got %T", program.Statements[0])
	}
	if stmt.Init == nil || stmt.Cond == nil || stmt.Post == nil {
		t.Fatalf("expected non-nil Init/Cond/Post, got %#v", stmt)
	}
	if stringifyExpr(stmt.Cond) != "($i < 10)" {
		t.Fatalf("unexpected condition: %s", stringifyExpr(stmt.Cond))
	}
}

func TestForeachAstShapeByValueAndByRef(t *testing.T) {
	program := parseProgram(t, `<?php
foreach ($items as $k => $v) { echo $k; }
foreach ($items as &$v) { $v = 1; }`)
	if len(program.Statements) != 2 {
		t.Fatalf("expected 2 statements, got %d", len(program.Statements))
	}
	byValue, ok := program.Statements[0].(*ast.ForeachStmt)
	if !ok {
		t.Fatalf("expected *ast.ForeachStmt, got %T", program.Statements[0])
	}
	if byValue.KeyVar != "k" || byValue.ValueVar != "v" || byValue.ValueByRef {
		t.Fatalf("unexpected by-value foreach shape: %#v", byValue)
	}
	byRef, ok := program.Statements[1].(*ast.ForeachStmt)
	if !ok {
		t.Fatalf("expected *ast.ForeachStmt, got %T", program.Statements[1])
	}
	if !byRef.ValueByRef || byRef.ValueVar != "v" {
		t.Fatalf("unexpected by-ref foreach shape: %#v", byRef)
	}
}

func TestFunctionDeclAstShape(t *testing.T) {
	program := parseProgram(t, `<?php function add(int $a, $b = 2, &$c) { return $a + $b; }`)
	fn, ok := program.Statements[0].(*ast.FunctionDeclStmt)
	if !ok {
		t.Fatalf("expected *ast.FunctionDeclStmt, got %T", program.Statements[0])
	}
	if fn.Name != "add" {
		t.Fatalf("unexpected function name: %s", fn.Name)
	}
	if len(fn.Params) != 3 {
		t.Fatalf("expected 3 params, got %d", len(fn.Params))
	}
	if fn.Params[0].Name != "a" || fn.Params[0].TypeHint != "int" {
		t.Fatalf("unexpected param[0]: %#v", fn.Params[0])
	}
	if fn.Params[1].Name != "b" || fn.Params[1].Default == nil || stringifyExpr(fn.Params[1].Default) != "2" {
		t.Fatalf("unexpected param[1]: %#v", fn.Params[1])
	}
	if fn.Params[2].Name != "c" || !fn.Params[2].ByRef {
		t.Fatalf("unexpected param[2]: %#v", fn.Params[2])
	}
	ret, ok := fn.Body.Statements[0].(*ast.ReturnStmt)
	if !ok {
		t.Fatalf("expected *ast.ReturnStmt in body, got %T", fn.Body.Statements[0])
	}
	if stringifyExpr(ret.Value) != "($a + $b)" {
		t.Fatalf("unexpected return value: %s", stringifyExpr(ret.Value))
	}
}

func TestClassDeclAstShape(t *testing.T) {
	program := parseProgram(t, `<?php
class Base {}
class Shape extends Base implements Countable, ArrayAccess {
    private $sides = 4;
    const NAME = "shape";
    public function area() { return 0; }
}`)
	if len(program.Statements) != 2 {
		t.Fatalf("expected 2 statements, got %d", len(program.Statements))
	}
	class, ok := program.Statements[1].(*ast.ClassDeclStmt)
	if !ok {
		t.Fatalf("expected *ast.ClassDeclStmt, got %T", program.Statements[1])
	}
	if class.Name != "Shape" || class.Parent != "Base" {
		t.Fatalf("unexpected class name/parent: %q/%q", class.Name, class.Parent)
	}
	if len(class.Implements) != 2 || class.Implements[0] != "Countable" || class.Implements[1] != "ArrayAccess" {
		t.Fatalf("unexpected implements list: %#v", class.Implements)
	}
	if len(class.Properties) != 1 || class.Properties[0].Name != "sides" || class.Properties[0].Visibility != "private" {
		t.Fatalf("unexpected properties: %#v", class.Properties)
	}
	if stringifyExpr(class.Properties[0].Default) != "4" {
		t.Fatalf("unexpected property default: %s", stringifyExpr(class.Properties[0].Default))
	}
	if len(class.Constants) != 1 || class.Constants[0].Name != "NAME" {
		t.Fatalf("unexpected constants: %#v", class.Constants)
	}
	if len(class.Methods) != 1 || class.Methods[0].Name != "area" {
		t.Fatalf("unexpected methods: %#v", class.Methods)
	}
}

func TestArrayLiteralAstShape(t *testing.T) {
	expr := parseExpr(t, `array(1, 2, "k" => 3)`)
	lit, ok := expr.(*ast.ArrayLit)
	if !ok {
		t.Fatalf("expected *ast.ArrayLit, got %T", expr)
	}
	if len(lit.Elements) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(lit.Elements))
	}
	if lit.Elements[0].Key != nil || stringifyExpr(lit.Elements[0].Value) != "1" {
		t.Fatalf("unexpected element[0]: %#v", lit.Elements[0])
	}
	if lit.Elements[2].Key == nil || stringifyExpr(lit.Elements[2].Key) != `"k"` || stringifyExpr(lit.Elements[2].Value) != "3" {
		t.Fatalf("unexpected element[2]: %#v", lit.Elements[2])
	}
}

func TestClosureAstShapeUseByValueAndByRef(t *testing.T) {
	// A bare `function` at statement-start parses as a named function
	// declaration, so the closure form needs to appear in expression
	// position (e.g. the right-hand side of an assignment) here.
	expr := parseExpr(t, `$fn = function ($x) use ($a, &$b) { return $x + $a + $b; }`)
	assign, ok := expr.(*ast.AssignExpr)
	if !ok {
		t.Fatalf("expected *ast.AssignExpr, got %T", expr)
	}
	closure, ok := assign.Value.(*ast.ClosureExpr)
	if !ok {
		t.Fatalf("expected *ast.ClosureExpr, got %T", assign.Value)
	}
	if len(closure.Params) != 1 || closure.Params[0].Name != "x" {
		t.Fatalf("unexpected params: %#v", closure.Params)
	}
	if len(closure.Uses) != 2 {
		t.Fatalf("expected 2 uses, got %d", len(closure.Uses))
	}
	if closure.Uses[0].Name != "a" || closure.Uses[0].ByRef {
		t.Fatalf("unexpected use[0]: %#v", closure.Uses[0])
	}
	if closure.Uses[1].Name != "b" || !closure.Uses[1].ByRef {
		t.Fatalf("unexpected use[1]: %#v", closure.Uses[1])
	}
}

func TestListAssignmentAstShape(t *testing.T) {
	expr := parseExpr(t, `list($a, $b) = $pair`)
	assign, ok := expr.(*ast.AssignExpr)
	if !ok {
		t.Fatalf("expected *ast.AssignExpr, got %T", expr)
	}
	list, ok := assign.Target.(*ast.ListExpr)
	if !ok {
		t.Fatalf("expected *ast.ListExpr target, got %T", assign.Target)
	}
	if len(list.Targets) != 2 || stringifyExpr(list.Targets[0]) != "$a" || stringifyExpr(list.Targets[1]) != "$b" {
		t.Fatalf("unexpected list targets: %#v", list.Targets)
	}
}

func TestTryCatchMultipleClausesAstShape(t *testing.T) {
	program := parseProgram(t, `<?php
try {
    doSomething();
} catch (InvalidArgumentException $e) {
    echo "bad arg";
} catch (Exception $e) {
    echo "generic";
}`)
	stmt, ok := program.Statements[0].(*ast.TryCatchStmt)
	if !ok {
		t.Fatalf("expected *ast.TryCatchStmt, got %T", program.Statements[0])
	}
	if len(stmt.Catches) != 2 {
		t.Fatalf("expected 2 catch clauses, got %d", len(stmt.Catches))
	}
	if stmt.Catches[0].Class != "InvalidArgumentException" || stmt.Catches[0].Var != "e" {
		t.Fatalf("unexpected catch[0]: %#v", stmt.Catches[0])
	}
	if stmt.Catches[1].Class != "Exception" {
		t.Fatalf("unexpected catch[1]: %#v", stmt.Catches[1])
	}
}

func TestSwitchAstShape(t *testing.T) {
	program := parseProgram(t, `<?php
switch ($x) {
case 1:
    echo "one";
    break;
default:
    echo "other";
}`)
	stmt, ok := program.Statements[0].(*ast.SwitchStmt)
	if !ok {
		t.Fatalf("expected *ast.SwitchStmt, got %T", program.Statements[0])
	}
	if len(stmt.Cases) != 2 {
		t.Fatalf("expected 2 cases, got %d", len(stmt.Cases))
	}
	if stmt.Cases[0].Default || stringifyExpr(stmt.Cases[0].Expr) != "1" {
		t.Fatalf("unexpected case[0]: %#v", stmt.Cases[0])
	}
	if !stmt.Cases[1].Default {
		t.Fatalf("expected case[1] to be default: %#v", stmt.Cases[1])
	}
}

func TestNewExprAstShape(t *testing.T) {
	expr := parseExpr(t, `new Foo(1, $x)`)
	n, ok := expr.(*ast.NewExpr)
	if !ok {
		t.Fatalf("expected *ast.NewExpr, got %T", expr)
	}
	if n.ClassName != "Foo" || len(n.Args) != 2 {
		t.Fatalf("unexpected new expr: %#v", n)
	}
}

func TestMethodAndIndexChainAstShape(t *testing.T) {
	expr := parseExpr(t, `$obj->items[0]->name`)
	prop, ok := expr.(*ast.PropertyFetchExpr)
	if !ok {
		t.Fatalf("expected outer *ast.PropertyFetchExpr, got %T", expr)
	}
	if prop.Property != "name" {
		t.Fatalf("unexpected outer property: %s", prop.Property)
	}
	index, ok := prop.Object.(*ast.IndexExpr)
	if !ok {
		t.Fatalf("expected *ast.IndexExpr, got %T", prop.Object)
	}
	if stringifyExpr(index.Index) != "0" {
		t.Fatalf("unexpected index: %s", stringifyExpr(index.Index))
	}
	inner, ok := index.Left.(*ast.PropertyFetchExpr)
	if !ok || inner.Property != "items" {
		t.Fatalf("unexpected inner property fetch: %#v", index.Left)
	}
}

func TestParseErrorMessagesAndPositions(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "missing expression after operator reports line and column",
			src:  "<?php $a = ;",
			want: "unexpected token ; at 1:7",
		},
		{
			name: "missing expression after echo",
			src:  "<?php echo 1 + ;",
			want: "unexpected token ; at 1:11",
		},
		{
			name: "position resets across newlines",
			src:  "<?php\n$a = );",
			want: "unexpected token ) at 2:6",
		},
		{
			name: "unclosed paren reports expected token",
			src:  "<?php if ($a > 1 {}",
			want: "expected ), got EOF",
		},
		{
			name: "missing function body brace",
			src:  "<?php class Foo { public function bar() }",
			want: "expected {, got }",
		},
		{
			name: "assigning to a non-writable expression",
			src:  "<?php 5 = $a;",
			want: "expression is not writable",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := parser.New(lexer.New(tt.src))
			p.ParseProgram()
			errs := p.Errors()
			found := false
			for _, e := range errs {
				if e == tt.want {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("expected error %q in %v (src %q)", tt.want, errs, tt.src)
			}
		})
	}
}
