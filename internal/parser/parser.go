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
		return p.parseExprStmt()
	case lexer.Echo:
		return p.parseEcho()
	case lexer.If:
		return p.parseIf()
	case lexer.While:
		return p.parseWhile()
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
	for p.peek.Type != lexer.Semicolon && p.peek.Type != lexer.RParen && precedence < p.peekPrecedence() {
		p.next()
		left = p.parseInfix(left)
	}
	return left
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
