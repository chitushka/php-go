package parser

import (
	"fmt"
	"strconv"

	"php-src-go/internal/ast"
	"php-src-go/internal/lexer"
)

const (
	_ int = iota
	lowest
	equals
	lessGreater
	sum
	product
	prefix
)

var precedences = map[lexer.TokenType]int{
	lexer.EQ: equals, lexer.NotEQ: equals,
	lexer.LT: lessGreater, lexer.GT: lessGreater, lexer.LTE: lessGreater, lexer.GTE: lessGreater,
	lexer.Plus: sum, lexer.Minus: sum,
	lexer.Asterisk: product, lexer.Slash: product,
}

type Parser struct {
	l         *lexer.Lexer
	cur, peek lexer.Token
	errors    []string
}

func New(l *lexer.Lexer) *Parser   { p := &Parser{l: l}; p.next(); p.next(); return p }
func (p *Parser) Errors() []string { return p.errors }
func (p *Parser) next()            { p.cur = p.peek; p.peek = p.l.NextToken() }

func (p *Parser) ParseProgram() *ast.Program {
	program := &ast.Program{}
	for p.cur.Type != lexer.EOF {
		if p.cur.Type == "" {
			p.next()
			continue
		}
		stmt := p.parseStatement()
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		}
		p.next()
	}
	return program
}

func (p *Parser) parseStatement() ast.Stmt {
	switch p.cur.Type {
	case lexer.Var:
		if p.peek.Type == lexer.Assign {
			return p.parseAssign()
		}
		return p.parseExprOrPropertyAssign()
	case lexer.Echo:
		return p.parseEcho()
	case lexer.If:
		return p.parseIf()
	case lexer.While:
		return p.parseWhile()
	case lexer.Foreach:
		return p.parseForeach()
	case lexer.Function:
		return p.parseFunctionDecl()
	case lexer.Return:
		return p.parseReturn()
	case lexer.Include:
		return p.parseInclude(false)
	case lexer.Require:
		return p.parseInclude(true)
	case lexer.Class:
		return p.parseClassDecl()
	case lexer.LBrace:
		return p.parseBlock()
	default:
		return p.parseExprStmt()
	}
}

func (p *Parser) parseAssign() ast.Stmt {
	name := p.cur.Literal
	p.next()
	p.next()
	value := p.parseExpression(lowest)
	p.consumeSemi()
	return &ast.AssignStmt{Name: name, Value: value}
}

func (p *Parser) parseEcho() ast.Stmt {
	stmt := &ast.EchoStmt{}
	p.next()
	stmt.Values = append(stmt.Values, p.parseExpression(lowest))
	for p.peek.Type == lexer.Comma {
		p.next()
		p.next()
		stmt.Values = append(stmt.Values, p.parseExpression(lowest))
	}
	p.consumeSemi()
	return stmt
}

func (p *Parser) parseIf() ast.Stmt {
	if !p.expectPeek(lexer.LParen) {
		return nil
	}
	p.next()
	cond := p.parseExpression(lowest)
	if !p.expectPeek(lexer.RParen) {
		return nil
	}
	if !p.expectPeek(lexer.LBrace) {
		return nil
	}
	cons := p.parseBlock().(*ast.BlockStmt)
	var alt *ast.BlockStmt
	if p.peek.Type == lexer.Else {
		p.next()
		if !p.expectPeek(lexer.LBrace) {
			return nil
		}
		alt = p.parseBlock().(*ast.BlockStmt)
	}
	return &ast.IfStmt{Cond: cond, Consequence: cons, Alternative: alt}
}

func (p *Parser) parseWhile() ast.Stmt {
	if !p.expectPeek(lexer.LParen) {
		return nil
	}
	p.next()
	cond := p.parseExpression(lowest)
	if !p.expectPeek(lexer.RParen) {
		return nil
	}
	if !p.expectPeek(lexer.LBrace) {
		return nil
	}
	body := p.parseBlock().(*ast.BlockStmt)
	return &ast.WhileStmt{Cond: cond, Body: body}
}

func (p *Parser) parseForeach() ast.Stmt {
	if !p.expectPeek(lexer.LParen) {
		return nil
	}
	p.next()
	source := p.parseExpression(lowest)
	if !p.expectPeek(lexer.As) {
		return nil
	}
	if !p.expectPeek(lexer.Var) {
		return nil
	}

	valueVar := p.cur.Literal
	keyVar := ""
	if p.peek.Type == lexer.DoubleArrow {
		keyVar = valueVar
		p.next()
		if !p.expectPeek(lexer.Var) {
			return nil
		}
		valueVar = p.cur.Literal
	}

	if !p.expectPeek(lexer.RParen) {
		return nil
	}
	if !p.expectPeek(lexer.LBrace) {
		return nil
	}
	body := p.parseBlock().(*ast.BlockStmt)
	return &ast.ForeachStmt{Source: source, KeyVar: keyVar, ValueVar: valueVar, Body: body}
}

func (p *Parser) parseFunctionDecl() ast.Stmt {
	if !p.expectPeek(lexer.Ident) {
		return nil
	}
	name := p.cur.Literal
	if !p.expectPeek(lexer.LParen) {
		return nil
	}
	params := p.parseFunctionParams()
	if !p.expectPeek(lexer.LBrace) {
		return nil
	}
	body := p.parseBlock().(*ast.BlockStmt)
	return &ast.FunctionDeclStmt{Name: name, Params: params, Body: body}
}

func (p *Parser) parseFunctionParams() []string {
	var params []string
	if p.peek.Type == lexer.RParen {
		p.next()
		return params
	}
	for {
		if !p.expectPeek(lexer.Var) {
			return params
		}
		params = append(params, p.cur.Literal)
		if p.peek.Type != lexer.Comma {
			break
		}
		p.next()
	}
	p.expectPeek(lexer.RParen)
	return params
}

func (p *Parser) parseReturn() ast.Stmt {
	if p.peek.Type == lexer.Semicolon {
		p.next()
		return &ast.ReturnStmt{}
	}
	p.next()
	value := p.parseExpression(lowest)
	p.consumeSemi()
	return &ast.ReturnStmt{Value: value}
}

func (p *Parser) parseInclude(require bool) ast.Stmt {
	if p.peek.Type == lexer.Semicolon {
		p.errors = append(p.errors, "include/require expects a path expression")
		return nil
	}
	p.next()
	path := p.parseExpression(lowest)
	p.consumeSemi()
	return &ast.IncludeStmt{Path: path, Require: require}
}

func (p *Parser) parseClassDecl() ast.Stmt {
	if !p.expectPeek(lexer.Ident) {
		return nil
	}
	name := p.cur.Literal
	if !p.expectPeek(lexer.LBrace) {
		return nil
	}
	class := &ast.ClassDeclStmt{Name: name}
	p.next()
	for p.cur.Type != lexer.RBrace && p.cur.Type != lexer.EOF {
		if p.cur.Type != lexer.Function {
			p.errors = append(p.errors, fmt.Sprintf("expected method declaration in class %s, got %s", name, p.cur.Type))
			return class
		}
		method, ok := p.parseFunctionDecl().(*ast.FunctionDeclStmt)
		if ok && method != nil {
			class.Methods = append(class.Methods, method)
		}
		p.next()
	}
	return class
}

func (p *Parser) parseExprOrPropertyAssign() ast.Stmt {
	expr := p.parseExpression(lowest)
	if p.peek.Type == lexer.Assign {
		prop, ok := expr.(*ast.PropertyFetchExpr)
		if !ok {
			p.errors = append(p.errors, "only variables and object properties can be assigned")
			return &ast.ExprStmt{Expr: expr}
		}
		p.next()
		p.next()
		value := p.parseExpression(lowest)
		p.consumeSemi()
		return &ast.PropertyAssignStmt{Object: prop.Object, Property: prop.Property, Value: value}
	}
	p.consumeSemi()
	return &ast.ExprStmt{Expr: expr}
}

func (p *Parser) parseBlock() ast.Stmt {
	block := &ast.BlockStmt{}
	p.next()
	for p.cur.Type != lexer.RBrace && p.cur.Type != lexer.EOF {
		stmt := p.parseStatement()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}
		p.next()
	}
	return block
}

func (p *Parser) parseExprStmt() ast.Stmt {
	expr := p.parseExpression(lowest)
	p.consumeSemi()
	return &ast.ExprStmt{Expr: expr}
}

func (p *Parser) parseExpression(precedence int) ast.Expr {
	var left ast.Expr
	switch p.cur.Type {
	case lexer.Var:
		left = &ast.Ident{Name: p.cur.Literal}
	case lexer.Ident:
		if p.peek.Type == lexer.LParen {
			left = p.parseCall()
		} else {
			p.errors = append(p.errors, fmt.Sprintf("unexpected bare identifier %s at %d:%d", p.cur.Literal, p.cur.Line, p.cur.Column))
			return &ast.NullLit{}
		}
	case lexer.Int:
		v, _ := strconv.ParseInt(p.cur.Literal, 10, 64)
		left = &ast.IntLit{Value: v}
	case lexer.Float:
		v, _ := strconv.ParseFloat(p.cur.Literal, 64)
		left = &ast.FloatLit{Value: v}
	case lexer.String:
		left = &ast.StringLit{Value: p.cur.Literal}
	case lexer.True:
		left = &ast.BoolLit{Value: true}
	case lexer.False:
		left = &ast.BoolLit{Value: false}
	case lexer.Null:
		left = &ast.NullLit{}
	case lexer.Array:
		left = p.parseArray()
	case lexer.NewKeyword:
		left = p.parseNew()
	case lexer.Bang, lexer.Minus:
		op := p.cur.Literal
		p.next()
		left = &ast.PrefixExpr{Op: op, Right: p.parseExpression(prefix)}
	case lexer.LParen:
		p.next()
		left = p.parseExpression(lowest)
		if !p.expectPeek(lexer.RParen) {
			return left
		}
	default:
		p.errors = append(p.errors, fmt.Sprintf("unexpected token %s at %d:%d", p.cur.Type, p.cur.Line, p.cur.Column))
		return &ast.NullLit{}
	}
	for p.peek.Type == lexer.ObjectOp {
		p.next()
		left = p.parseObjectAccess(left)
	}
	for p.peek.Type != lexer.Semicolon && p.peek.Type != lexer.RParen && precedence < p.peekPrecedence() {
		p.next()
		left = p.parseInfix(left)
		for p.peek.Type == lexer.ObjectOp {
			p.next()
			left = p.parseObjectAccess(left)
		}
	}
	return left
}

func (p *Parser) parseNew() ast.Expr {
	expr := &ast.NewExpr{}
	if !p.expectPeek(lexer.Ident) {
		return expr
	}
	expr.ClassName = p.cur.Literal
	if p.peek.Type != lexer.LParen {
		return expr
	}
	p.next()
	expr.Args = p.parseCallArgs()
	return expr
}

func (p *Parser) parseObjectAccess(object ast.Expr) ast.Expr {
	if !p.expectPeek(lexer.Ident) {
		return object
	}
	name := p.cur.Literal
	if p.peek.Type == lexer.LParen {
		p.next()
		return &ast.MethodCallExpr{Object: object, Method: name, Args: p.parseCallArgs()}
	}
	return &ast.PropertyFetchExpr{Object: object, Property: name}
}

func (p *Parser) parseCall() ast.Expr {
	call := &ast.CallExpr{Name: p.cur.Literal}
	if !p.expectPeek(lexer.LParen) {
		return call
	}
	call.Args = p.parseCallArgs()
	return call
}

func (p *Parser) parseCallArgs() []ast.Expr {
	var args []ast.Expr
	if p.peek.Type == lexer.RParen {
		p.next()
		return args
	}
	for {
		p.next()
		args = append(args, p.parseExpression(lowest))
		if p.peek.Type != lexer.Comma {
			break
		}
		p.next()
	}
	p.expectPeek(lexer.RParen)
	return args
}

func (p *Parser) parseArray() ast.Expr {
	lit := &ast.ArrayLit{}
	if !p.expectPeek(lexer.LParen) {
		return lit
	}
	if p.peek.Type == lexer.RParen {
		p.next()
		return lit
	}

	for {
		p.next()
		first := p.parseExpression(lowest)
		elem := ast.ArrayElement{Value: first}
		if p.peek.Type == lexer.DoubleArrow {
			elem.Key = first
			p.next()
			p.next()
			elem.Value = p.parseExpression(lowest)
		}
		lit.Elements = append(lit.Elements, elem)

		if p.peek.Type != lexer.Comma {
			break
		}
		p.next()
		if p.peek.Type == lexer.RParen {
			p.next()
			return lit
		}
	}

	if !p.expectPeek(lexer.RParen) {
		return lit
	}
	return lit
}

func (p *Parser) parseInfix(left ast.Expr) ast.Expr {
	expr := &ast.InfixExpr{Left: left, Op: p.cur.Literal}
	prec := p.curPrecedence()
	p.next()
	expr.Right = p.parseExpression(prec)
	return expr
}
func (p *Parser) consumeSemi() {
	if p.peek.Type == lexer.Semicolon {
		p.next()
	}
}
func (p *Parser) expectPeek(t lexer.TokenType) bool {
	if p.peek.Type == t {
		p.next()
		return true
	}
	p.errors = append(p.errors, fmt.Sprintf("expected %s, got %s", t, p.peek.Type))
	return false
}
func (p *Parser) peekPrecedence() int {
	if p, ok := precedences[p.peek.Type]; ok {
		return p
	}
	return lowest
}
func (p *Parser) curPrecedence() int {
	if p, ok := precedences[p.cur.Type]; ok {
		return p
	}
	return lowest
}
