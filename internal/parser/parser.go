package parser

import (
	"fmt"
	"strconv"
	"strings"

	"php-src-go/internal/ast"
	"php-src-go/internal/lexer"
)

const (
	_ int = iota
	lowest
	wordLogical
	assignment
	ternary
	logicalOr
	logicalAnd
	bitwiseOr
	bitwiseXor
	bitwiseAnd
	equals
	lessGreater
	shift
	sum
	product
	prefix
)

var precedences = map[lexer.TokenType]int{
	lexer.AndWord: wordLogical, lexer.OrWord: wordLogical, lexer.XorWord: wordLogical,
	lexer.Assign:       assignment,
	lexer.AssignRef:    assignment,
	lexer.PlusAssign:   assignment,
	lexer.MinusAssign:  assignment,
	lexer.MulAssign:    assignment,
	lexer.DivAssign:    assignment,
	lexer.ConcatAssign: assignment,
	lexer.ModAssign:    assignment,
	lexer.Question:     ternary,
	lexer.OrOr:         logicalOr,
	lexer.AndAnd:       logicalAnd,
	lexer.Pipe:         bitwiseOr,
	lexer.Caret:        bitwiseXor,
	lexer.Ampersand:    bitwiseAnd,
	lexer.EQ:           equals, lexer.NotEQ: equals,
	lexer.StrictEQ: equals, lexer.StrictNotEQ: equals,
	lexer.LT: lessGreater, lexer.GT: lessGreater, lexer.LTE: lessGreater, lexer.GTE: lessGreater,
	lexer.InstanceOf: lessGreater,
	lexer.ShiftLeft:  shift, lexer.ShiftRight: shift,
	lexer.Plus: sum, lexer.Minus: sum, lexer.Dot: sum,
	lexer.Asterisk: product, lexer.Slash: product, lexer.Percent: product,
}

type Parser struct {
	l                *lexer.Lexer
	cur, peek        lexer.Token
	errors           []string
	currentNamespace string
	imports          map[string]string
}

func New(l *lexer.Lexer) *Parser {
	p := &Parser{l: l, imports: map[string]string{}}
	p.next()
	p.next()
	return p
}
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
	case lexer.Semicolon:
		return nil
	case lexer.Var, lexer.VarVar:
		if p.peek.Type == lexer.Assign || isCompoundAssignOp(p.peek.Type) {
			return p.parseAssign()
		}
		return p.parseExprOrPropertyAssign()
	case lexer.Increment, lexer.Decrement:
		return p.parseExprStmt()
	case lexer.Echo:
		return p.parseEcho()
	case lexer.Print:
		return p.parsePrint()
	case lexer.Exit:
		return p.parseExit()
	case lexer.Unset:
		return p.parseUnset()
	case lexer.If:
		return p.parseIf()
	case lexer.While:
		return p.parseWhile()
	case lexer.Do:
		return p.parseDoWhile()
	case lexer.For:
		return p.parseFor()
	case lexer.Foreach:
		return p.parseForeach()
	case lexer.Switch:
		return p.parseSwitch()
	case lexer.Break:
		return p.parseBreak()
	case lexer.Continue:
		return p.parseContinue()
	case lexer.Function:
		return p.parseFunctionDecl()
	case lexer.Namespace:
		return p.parseNamespaceDecl()
	case lexer.Use:
		return p.parseUseImport()
	case lexer.Return:
		return p.parseReturn()
	case lexer.Global:
		return p.parseGlobal()
	case lexer.Include:
		return p.parseInclude(false, false)
	case lexer.Require:
		return p.parseInclude(true, false)
	case lexer.IncludeOnce:
		return p.parseInclude(false, true)
	case lexer.RequireOnce:
		return p.parseInclude(true, true)
	case lexer.Class:
		return p.parseClassDecl(false, false)
	case lexer.Interface:
		return p.parseInterfaceDecl()
	case lexer.Final:
		return p.parseClassDeclWithLeadingModifier("final")
	case lexer.Abstract:
		return p.parseClassDeclWithLeadingModifier("abstract")
	case lexer.Ident:
		return p.parseExprOrPropertyAssign()
	case lexer.Static:
		if p.peek.Type == lexer.Var {
			return p.parseStaticLocal()
		}
		return p.parseExprOrPropertyAssign()
	case lexer.Self, lexer.Parent:
		return p.parseExprOrPropertyAssign()
	case lexer.Throw:
		return p.parseThrow()
	case lexer.Try:
		return p.parseTryCatch()
	case lexer.LBrace:
		return p.parseBlock()
	default:
		return p.parseExprStmt()
	}
}

func (p *Parser) parseNamespaceDecl() ast.Stmt {
	if !p.expectPeek(lexer.Ident) {
		return nil
	}
	p.currentNamespace = trimLeadingNamespace(p.cur.Literal)
	p.imports = map[string]string{}
	p.consumeSemi()
	return nil
}

func (p *Parser) parseUseImport() ast.Stmt {
	for {
		if !p.expectPeek(lexer.Ident) {
			return nil
		}
		name := trimLeadingNamespace(p.cur.Literal)
		alias := namespaceBaseName(name)
		if p.peek.Type == lexer.As {
			p.next()
			if !p.expectPeek(lexer.Ident) {
				return nil
			}
			alias = namespaceBaseName(p.cur.Literal)
		}
		p.imports[alias] = name
		if p.peek.Type != lexer.Comma {
			break
		}
		p.next()
	}
	p.consumeSemi()
	return nil
}

func (p *Parser) parseAssign() ast.Stmt {
	expr := p.parseExpression(lowest)
	p.consumeSemi()
	return &ast.ExprStmt{Expr: expr}
}

func (p *Parser) parseIncDecStmt() ast.Stmt {
	target := p.parseVariableTarget()
	p.next()
	op := p.cur.Literal
	p.consumeSemi()
	return &ast.IncDecStmt{Target: target, Op: op}
}

func (p *Parser) parsePrefixIncDecStmt() ast.Stmt {
	op := p.cur.Literal
	if p.peek.Type != lexer.Var && p.peek.Type != lexer.VarVar {
		p.errors = append(p.errors, fmt.Sprintf("expected variable, got %s", p.peek.Type))
		return nil
	}
	p.next()
	target := p.parseVariableTarget()
	p.consumeSemi()
	return &ast.IncDecStmt{Target: target, Op: op}
}

func (p *Parser) parseVariableTarget() ast.Expr {
	if p.cur.Type == lexer.VarVar {
		return &ast.VariableVariable{Name: p.cur.Literal}
	}
	return &ast.Ident{Name: p.cur.Literal}
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

func (p *Parser) parsePrint() ast.Stmt {
	stmt := &ast.EchoStmt{}
	p.next()
	stmt.Values = append(stmt.Values, p.parseExpression(lowest))
	p.consumeSemi()
	return stmt
}

func (p *Parser) parseExit() ast.Stmt {
	stmt := &ast.ExitStmt{}
	if p.peek.Type == lexer.Semicolon {
		p.next()
		return stmt
	}
	if p.peek.Type == lexer.LParen {
		p.next()
		if p.peek.Type == lexer.RParen {
			p.next()
			p.consumeSemi()
			return stmt
		}
		p.next()
		stmt.Value = p.parseExpression(lowest)
		if !p.expectPeek(lexer.RParen) {
			return stmt
		}
		p.consumeSemi()
		return stmt
	}
	p.next()
	stmt.Value = p.parseExpression(lowest)
	p.consumeSemi()
	return stmt
}

func (p *Parser) parseUnset() ast.Stmt {
	stmt := &ast.UnsetStmt{}
	if !p.expectPeek(lexer.LParen) {
		return stmt
	}
	for {
		p.next()
		stmt.Targets = append(stmt.Targets, p.parseExpression(lowest))
		if p.peek.Type != lexer.Comma {
			break
		}
		p.next()
	}
	p.expectPeek(lexer.RParen)
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
	if p.peek.Type == lexer.Colon {
		p.next()
		return p.parseAlternativeIf(cond)
	}
	if !p.expectPeek(lexer.LBrace) {
		return nil
	}
	cons := p.parseBlock().(*ast.BlockStmt)
	var alt *ast.BlockStmt
	if p.peek.Type == lexer.ElseIf {
		p.next()
		nested := p.parseIf()
		if nested != nil {
			alt = &ast.BlockStmt{Statements: []ast.Stmt{nested}}
		}
	} else if p.peek.Type == lexer.Else {
		p.next()
		if p.peek.Type == lexer.If {
			p.next()
			nested := p.parseIf()
			if nested != nil {
				alt = &ast.BlockStmt{Statements: []ast.Stmt{nested}}
			}
		} else {
			if !p.expectPeek(lexer.LBrace) {
				return nil
			}
			alt = p.parseBlock().(*ast.BlockStmt)
		}
	}
	return &ast.IfStmt{Cond: cond, Consequence: cons, Alternative: alt}
}

func (p *Parser) parseAlternativeIf(cond ast.Expr) ast.Stmt {
	stmt := &ast.IfStmt{Cond: cond, Consequence: p.parseAlternativeControlBlock(lexer.ElseIf, lexer.Else, lexer.EndIf)}
	switch p.cur.Type {
	case lexer.ElseIf:
		nested := p.parseAlternativeElseIf()
		if nested != nil {
			stmt.Alternative = &ast.BlockStmt{Statements: []ast.Stmt{nested}}
		}
		return stmt
	case lexer.Else:
		if !p.expectPeek(lexer.Colon) {
			return stmt
		}
		stmt.Alternative = p.parseAlternativeControlBlock(lexer.EndIf)
	}
	if p.cur.Type != lexer.EndIf {
		p.errors = append(p.errors, fmt.Sprintf("expected endif, got %s", p.cur.Type))
		return stmt
	}
	p.consumeSemi()
	return stmt
}

func (p *Parser) parseAlternativeElseIf() ast.Stmt {
	if !p.expectPeek(lexer.LParen) {
		return nil
	}
	p.next()
	cond := p.parseExpression(lowest)
	if !p.expectPeek(lexer.RParen) {
		return nil
	}
	if !p.expectPeek(lexer.Colon) {
		return nil
	}
	return p.parseAlternativeIf(cond)
}

func (p *Parser) parseAlternativeControlBlock(stops ...lexer.TokenType) *ast.BlockStmt {
	block := &ast.BlockStmt{}
	p.next()
	for p.cur.Type != lexer.EOF && !tokenIn(p.cur.Type, stops) {
		stmt := p.parseStatement()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}
		p.next()
	}
	return block
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
	if p.peek.Type == lexer.Colon {
		p.next()
		return p.parseAlternativeWhile(cond)
	}
	if !p.expectPeek(lexer.LBrace) {
		return nil
	}
	body := p.parseBlock().(*ast.BlockStmt)
	return &ast.WhileStmt{Cond: cond, Body: body}
}

func (p *Parser) parseAlternativeWhile(cond ast.Expr) ast.Stmt {
	body := p.parseAlternativeControlBlock(lexer.EndWhile)
	if p.cur.Type != lexer.EndWhile {
		p.errors = append(p.errors, fmt.Sprintf("expected endwhile, got %s", p.cur.Type))
		return &ast.WhileStmt{Cond: cond, Body: body}
	}
	p.consumeSemi()
	return &ast.WhileStmt{Cond: cond, Body: body}
}

func (p *Parser) parseDoWhile() ast.Stmt {
	if !p.expectPeek(lexer.LBrace) {
		return nil
	}
	body := p.parseBlock().(*ast.BlockStmt)
	if !p.expectPeek(lexer.While) {
		return nil
	}
	if !p.expectPeek(lexer.LParen) {
		return nil
	}
	p.next()
	cond := p.parseExpression(lowest)
	if !p.expectPeek(lexer.RParen) {
		return nil
	}
	p.consumeSemi()
	return &ast.DoWhileStmt{Body: body, Cond: cond}
}

func (p *Parser) parseFor() ast.Stmt {
	if !p.expectPeek(lexer.LParen) {
		return nil
	}
	p.next()
	var init ast.Stmt
	if p.cur.Type != lexer.Semicolon {
		init = p.parseForClauseStmt()
		if !p.expectPeek(lexer.Semicolon) {
			return nil
		}
	}

	var cond ast.Expr
	if p.peek.Type == lexer.Semicolon {
		p.next()
	} else {
		p.next()
		cond = p.parseExpression(lowest)
		if !p.expectPeek(lexer.Semicolon) {
			return nil
		}
	}

	var post ast.Stmt
	if p.peek.Type == lexer.RParen {
		p.next()
	} else {
		p.next()
		post = p.parseForClauseStmt()
		if !p.expectPeek(lexer.RParen) {
			return nil
		}
	}
	if p.peek.Type == lexer.Colon {
		p.next()
		return p.parseAlternativeFor(init, cond, post)
	}
	if !p.expectPeek(lexer.LBrace) {
		return nil
	}
	body := p.parseBlock().(*ast.BlockStmt)
	return &ast.ForStmt{Init: init, Cond: cond, Post: post, Body: body}
}

func (p *Parser) parseAlternativeFor(init ast.Stmt, cond ast.Expr, post ast.Stmt) ast.Stmt {
	body := p.parseAlternativeControlBlock(lexer.EndFor)
	if p.cur.Type != lexer.EndFor {
		p.errors = append(p.errors, fmt.Sprintf("expected endfor, got %s", p.cur.Type))
		return &ast.ForStmt{Init: init, Cond: cond, Post: post, Body: body}
	}
	p.consumeSemi()
	return &ast.ForStmt{Init: init, Cond: cond, Post: post, Body: body}
}

func (p *Parser) parseForClauseStmt() ast.Stmt {
	if p.cur.Type == lexer.Var && (p.peek.Type == lexer.Assign || isCompoundAssignOp(p.peek.Type)) {
		return &ast.ExprStmt{Expr: p.parseExpression(lowest)}
	}
	if p.cur.Type == lexer.Var && (p.peek.Type == lexer.Increment || p.peek.Type == lexer.Decrement) {
		target := &ast.Ident{Name: p.cur.Literal}
		p.next()
		return &ast.IncDecStmt{Target: target, Op: p.cur.Literal}
	}
	if (p.cur.Type == lexer.Increment || p.cur.Type == lexer.Decrement) && p.peek.Type == lexer.Var {
		op := p.cur.Literal
		p.next()
		return &ast.IncDecStmt{Target: &ast.Ident{Name: p.cur.Literal}, Op: op}
	}
	expr := p.parseExpression(lowest)
	if p.peek.Type == lexer.Assign || isCompoundAssignOp(p.peek.Type) {
		p.next()
		op := p.cur.Type
		p.next()
		value := p.parseExpression(lowest)
		if isCompoundAssignOp(op) {
			return &ast.CompoundAssignStmt{Target: expr, Op: compoundBinaryOp(op), Value: value}
		}
		switch target := expr.(type) {
		case *ast.PropertyFetchExpr:
			return &ast.PropertyAssignStmt{Object: target.Object, Property: target.Property, Value: value}
		case *ast.DynamicPropertyFetchExpr:
			return &ast.ExprStmt{Expr: &ast.AssignExpr{Target: target, Op: "=", Value: value}}
		case *ast.StaticPropertyFetchExpr:
			return &ast.StaticPropertyAssignStmt{ClassName: target.ClassName, Property: target.Property, Value: value}
		case *ast.DynamicStaticPropertyFetchExpr:
			return &ast.ExprStmt{Expr: &ast.AssignExpr{Target: target, Op: "=", Value: value}}
		case *ast.IndexExpr:
			return &ast.IndexAssignStmt{Target: target, Value: value}
		default:
			p.errors = append(p.errors, "only variables, object properties and indexes can be assigned")
		}
	}
	return &ast.ExprStmt{Expr: expr}
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

	valueVar, valueByRef, ok := p.parseForeachVariable()
	if !ok {
		return nil
	}
	keyVar := ""
	if p.peek.Type == lexer.DoubleArrow {
		if valueByRef {
			p.errors = append(p.errors, "foreach key cannot be assigned by reference")
		}
		keyVar = valueVar
		p.next()
		valueVar, valueByRef, ok = p.parseForeachVariable()
		if !ok {
			return nil
		}
	}

	if !p.expectPeek(lexer.RParen) {
		return nil
	}
	if p.peek.Type == lexer.Colon {
		p.next()
		return p.parseAlternativeForeach(source, keyVar, valueVar, valueByRef)
	}
	if !p.expectPeek(lexer.LBrace) {
		return nil
	}
	body := p.parseBlock().(*ast.BlockStmt)
	return &ast.ForeachStmt{Source: source, KeyVar: keyVar, ValueVar: valueVar, ValueByRef: valueByRef, Body: body}
}

func (p *Parser) parseForeachVariable() (string, bool, bool) {
	byRef := false
	if p.peek.Type == lexer.Ampersand {
		p.next()
		byRef = true
	}
	if !p.expectPeek(lexer.Var) {
		return "", byRef, false
	}
	return p.cur.Literal, byRef, true
}

func (p *Parser) parseAlternativeForeach(source ast.Expr, keyVar string, valueVar string, valueByRef bool) ast.Stmt {
	body := p.parseAlternativeControlBlock(lexer.EndForeach)
	if p.cur.Type != lexer.EndForeach {
		p.errors = append(p.errors, fmt.Sprintf("expected endforeach, got %s", p.cur.Type))
		return &ast.ForeachStmt{Source: source, KeyVar: keyVar, ValueVar: valueVar, ValueByRef: valueByRef, Body: body}
	}
	p.consumeSemi()
	return &ast.ForeachStmt{Source: source, KeyVar: keyVar, ValueVar: valueVar, ValueByRef: valueByRef, Body: body}
}

func (p *Parser) parseSwitch() ast.Stmt {
	if !p.expectPeek(lexer.LParen) {
		return nil
	}
	p.next()
	value := p.parseExpression(lowest)
	if !p.expectPeek(lexer.RParen) {
		return nil
	}
	if p.peek.Type == lexer.Colon {
		p.next()
		return p.parseAlternativeSwitch(value)
	}
	if !p.expectPeek(lexer.LBrace) {
		return nil
	}
	stmt := &ast.SwitchStmt{Value: value}
	p.next()
	for p.cur.Type != lexer.RBrace && p.cur.Type != lexer.EOF {
		caseStmt, ok := p.parseSwitchCase()
		if !ok {
			return stmt
		}
		stmt.Cases = append(stmt.Cases, caseStmt)
	}
	return stmt
}

func (p *Parser) parseAlternativeSwitch(value ast.Expr) ast.Stmt {
	stmt := &ast.SwitchStmt{Value: value}
	p.next()
	for p.cur.Type != lexer.EndSwitch && p.cur.Type != lexer.EOF {
		caseStmt, ok := p.parseSwitchCase()
		if !ok {
			return stmt
		}
		stmt.Cases = append(stmt.Cases, caseStmt)
	}
	if p.cur.Type != lexer.EndSwitch {
		p.errors = append(p.errors, fmt.Sprintf("expected endswitch, got %s", p.cur.Type))
		return stmt
	}
	p.consumeSemi()
	return stmt
}

func (p *Parser) parseSwitchCase() (ast.SwitchCase, bool) {
	c := ast.SwitchCase{}
	switch p.cur.Type {
	case lexer.Case:
		p.next()
		c.Expr = p.parseExpression(lowest)
		if !p.expectPeek(lexer.Colon) {
			return c, false
		}
	case lexer.Default:
		c.Default = true
		if !p.expectPeek(lexer.Colon) {
			return c, false
		}
	default:
		p.errors = append(p.errors, fmt.Sprintf("expected switch case/default, got %s", p.cur.Type))
		return c, false
	}
	p.next()
	for p.cur.Type != lexer.Case && p.cur.Type != lexer.Default && p.cur.Type != lexer.RBrace && p.cur.Type != lexer.EndSwitch && p.cur.Type != lexer.EOF {
		stmt := p.parseStatement()
		if stmt != nil {
			c.Statements = append(c.Statements, stmt)
		}
		p.next()
	}
	return c, true
}

func (p *Parser) parseFunctionDecl() ast.Stmt {
	return p.parseFunctionDeclWith("", false)
}

func (p *Parser) parseFunctionDeclWith(visibility string, static bool) ast.Stmt {
	return p.parseFunctionDeclFull(visibility, static, false, false, true, "", "")
}

func (p *Parser) parseFunctionDeclFull(visibility string, static bool, final bool, abstract bool, qualifyName bool, owner string, abstractKind string) ast.Stmt {
	returnsByRef := false
	if p.peek.Type == lexer.Ampersand {
		p.next()
		returnsByRef = true
	}
	if !p.expectPeek(lexer.Ident) {
		return nil
	}
	name := p.cur.Literal
	if qualifyName {
		name = p.resolveDeclarationName(name)
	}
	if !p.expectPeek(lexer.LParen) {
		return nil
	}
	params := p.parseFunctionParams()
	if p.peek.Type == lexer.Semicolon {
		p.next()
		if !abstract && owner != "" {
			p.errors = append(p.errors, fmt.Sprintf("Non-abstract method %s::%s() must contain body", owner, name))
		}
		return &ast.FunctionDeclStmt{Name: name, Params: params, Visibility: visibility, Static: static, Final: final, Abstract: abstract, ReturnsByRef: returnsByRef}
	}
	if abstract && p.peek.Type == lexer.LBrace {
		p.next()
		p.parseBlock()
		if owner != "" {
			if abstractKind == "" {
				abstractKind = "Abstract"
			}
			p.errors = append(p.errors, fmt.Sprintf("%s function %s::%s() cannot contain body", abstractKind, owner, name))
		}
		return &ast.FunctionDeclStmt{Name: name, Params: params, Visibility: visibility, Static: static, Final: final, Abstract: abstract, ReturnsByRef: returnsByRef}
	}
	if !p.expectPeek(lexer.LBrace) {
		return nil
	}
	body := p.parseBlock().(*ast.BlockStmt)
	return &ast.FunctionDeclStmt{Name: name, Params: params, Body: body, Visibility: visibility, Static: static, Final: final, Abstract: abstract, ReturnsByRef: returnsByRef}
}

func (p *Parser) parseFunctionParams() []ast.Param {
	var params []ast.Param
	if p.peek.Type == lexer.RParen {
		p.next()
		return params
	}
	for {
		param := ast.Param{}
		if p.peek.Type == lexer.Ampersand {
			p.next()
			param.ByRef = true
		}
		if !p.expectPeek(lexer.Var) {
			return params
		}
		param.Name = p.cur.Literal
		if p.peek.Type == lexer.Assign {
			p.next()
			p.next()
			param.Default = p.parseExpression(lowest)
		}
		params = append(params, param)
		if p.peek.Type != lexer.Comma {
			break
		}
		p.next()
	}
	p.expectPeek(lexer.RParen)
	return params
}

func (p *Parser) parseClosure() ast.Expr {
	returnsByRef := false
	if p.peek.Type == lexer.Ampersand {
		p.next()
		returnsByRef = true
	}
	if !p.expectPeek(lexer.LParen) {
		return &ast.NullLit{}
	}
	params := p.parseFunctionParams()
	uses := p.parseClosureUses()
	if !p.expectPeek(lexer.LBrace) {
		return &ast.NullLit{}
	}
	body := p.parseBlock().(*ast.BlockStmt)
	return &ast.ClosureExpr{Params: params, Uses: uses, Body: body, ReturnsByRef: returnsByRef}
}

func (p *Parser) parseClosureUses() []ast.ClosureUse {
	var uses []ast.ClosureUse
	if p.peek.Type != lexer.Use {
		return uses
	}
	p.next()
	if !p.expectPeek(lexer.LParen) {
		return uses
	}
	if p.peek.Type == lexer.RParen {
		p.next()
		return uses
	}
	for {
		use := ast.ClosureUse{}
		if p.peek.Type == lexer.Ampersand {
			p.next()
			use.ByRef = true
		}
		if !p.expectPeek(lexer.Var) {
			return uses
		}
		use.Name = p.cur.Literal
		uses = append(uses, use)
		if p.peek.Type != lexer.Comma {
			break
		}
		p.next()
	}
	p.expectPeek(lexer.RParen)
	return uses
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

func (p *Parser) parseGlobal() ast.Stmt {
	stmt := &ast.GlobalStmt{}
	for {
		if !p.expectPeek(lexer.Var) {
			return stmt
		}
		stmt.Names = append(stmt.Names, p.cur.Literal)
		if p.peek.Type != lexer.Comma {
			break
		}
		p.next()
	}
	p.consumeSemi()
	return stmt
}

func (p *Parser) parseStaticLocal() ast.Stmt {
	stmt := &ast.StaticLocalStmt{}
	for {
		if !p.expectPeek(lexer.Var) {
			return stmt
		}
		decl := ast.StaticLocalDecl{Name: p.cur.Literal}
		if p.peek.Type == lexer.Assign {
			p.next()
			p.next()
			decl.Default = p.parseExpression(lowest)
		}
		stmt.Decls = append(stmt.Decls, decl)
		if p.peek.Type != lexer.Comma {
			break
		}
		p.next()
	}
	p.consumeSemi()
	return stmt
}

func (p *Parser) parseBreak() ast.Stmt {
	level := p.parseLoopControlLevel("break")
	return &ast.BreakStmt{Level: level}
}

func (p *Parser) parseContinue() ast.Stmt {
	level := p.parseLoopControlLevel("continue")
	return &ast.ContinueStmt{Level: level}
}

func (p *Parser) parseLoopControlLevel(name string) int {
	level := 1
	if p.peek.Type == lexer.Semicolon {
		p.consumeSemi()
		return level
	}
	if !p.expectPeek(lexer.Int) {
		p.consumeSemi()
		return level
	}
	parsed, err := strconv.Atoi(p.cur.Literal)
	if err != nil || parsed < 1 {
		p.errors = append(p.errors, fmt.Sprintf("%s level must be a positive integer", name))
	} else {
		level = parsed
	}
	if p.peek.Type != lexer.Semicolon {
		p.errors = append(p.errors, fmt.Sprintf("%s level must be followed by semicolon", name))
	}
	p.consumeSemi()
	return level
}

func (p *Parser) parseInclude(require bool, once bool) ast.Stmt {
	if p.peek.Type == lexer.Semicolon {
		p.errors = append(p.errors, "include/require expects a path expression")
		return nil
	}
	p.next()
	path := p.parseExpression(lowest)
	p.consumeSemi()
	return &ast.IncludeStmt{Path: path, Require: require, Once: once}
}

func (p *Parser) parseClassDeclWithLeadingModifier(mod string) ast.Stmt {
	if p.peek.Type != lexer.Class {
		p.errors = append(p.errors, fmt.Sprintf("%s must be followed by class", mod))
		return nil
	}
	p.next()
	return p.parseClassDecl(mod == "final", mod == "abstract")
}

func (p *Parser) parseClassDecl(final bool, abstract bool) ast.Stmt {
	name, ok := p.parseDeclarationClassName()
	if !ok {
		return nil
	}
	parent := ""
	if p.peek.Type == lexer.Extends {
		p.next()
		parsedParent, ok := p.parseClassReferenceName()
		if !ok {
			return nil
		}
		parent = parsedParent
	}
	implements := p.parseImplementsList()
	if !p.expectPeek(lexer.LBrace) {
		return nil
	}
	class := &ast.ClassDeclStmt{Name: name, Parent: parent, Implements: implements, Final: final, Abstract: abstract}
	p.next()
	for p.cur.Type != lexer.RBrace && p.cur.Type != lexer.EOF {
		visibility, static, finalMember, abstractMember := p.parseMemberModifiers()
		switch p.cur.Type {
		case lexer.Function:
			method, ok := p.parseFunctionDeclFull(visibility, static, finalMember, abstractMember, false, name, "Abstract").(*ast.FunctionDeclStmt)
			if ok && method != nil {
				class.Methods = append(class.Methods, method)
			}
		case lexer.Var:
			class.Properties = append(class.Properties, p.parseClassProperty(visibility, static, finalMember, abstractMember))
		case lexer.Const:
			class.Constants = append(class.Constants, p.parseClassConstants()...)
		default:
			p.errors = append(p.errors, fmt.Sprintf("expected class member in %s, got %s", name, p.cur.Type))
			return class
		}
		p.next()
	}
	return class
}

func (p *Parser) parseImplementsList() []string {
	var names []string
	if p.peek.Type != lexer.Implements {
		return names
	}
	p.next()
	for {
		name, ok := p.parseClassReferenceName()
		if !ok {
			return names
		}
		names = append(names, name)
		if p.peek.Type != lexer.Comma {
			break
		}
		p.next()
	}
	return names
}

func (p *Parser) parseInterfaceDecl() ast.Stmt {
	name, ok := p.parseDeclarationClassName()
	if !ok {
		return nil
	}
	var parents []string
	if p.peek.Type == lexer.Extends {
		p.next()
		for {
			parent, ok := p.parseClassReferenceName()
			if !ok {
				return nil
			}
			parents = append(parents, parent)
			if p.peek.Type != lexer.Comma {
				break
			}
			p.next()
		}
	}
	if !p.expectPeek(lexer.LBrace) {
		return nil
	}
	iface := &ast.InterfaceDeclStmt{Name: name, Parents: parents}
	p.next()
	for p.cur.Type != lexer.RBrace && p.cur.Type != lexer.EOF {
		visibility, static, finalMember, _ := p.parseMemberModifiers()
		switch p.cur.Type {
		case lexer.Const:
			iface.Constants = append(iface.Constants, p.parseClassConstants()...)
		case lexer.Function:
			method, ok := p.parseFunctionDeclFull(visibility, static, finalMember, true, false, name, "Interface").(*ast.FunctionDeclStmt)
			if ok && method != nil {
				iface.Methods = append(iface.Methods, method)
			}
		default:
			p.errors = append(p.errors, fmt.Sprintf("expected interface member in %s, got %s", name, p.cur.Type))
			return iface
		}
		p.next()
	}
	return iface
}

func (p *Parser) parseDeclarationClassName() (string, bool) {
	if isReservedClassNameToken(p.peek.Type) {
		p.next()
		p.errors = append(p.errors, reservedClassNameError(p.cur.Literal))
		return "", false
	}
	if !p.expectPeek(lexer.Ident) {
		return "", false
	}
	return p.resolveDeclarationName(p.cur.Literal), true
}

func (p *Parser) parseClassReferenceName() (string, bool) {
	if isReservedClassNameToken(p.peek.Type) {
		p.next()
		p.errors = append(p.errors, reservedClassNameError(p.cur.Literal))
		return "", false
	}
	if !p.expectPeek(lexer.Ident) {
		return "", false
	}
	return p.resolveClassName(p.cur.Literal), true
}

func isReservedClassNameToken(token lexer.TokenType) bool {
	return token == lexer.Self || token == lexer.Parent || token == lexer.Static
}

func reservedClassNameError(name string) string {
	return fmt.Sprintf("cannot use '%s' as class name as it is reserved", strings.ToLower(name))
}

func (p *Parser) parseMemberModifiers() (string, bool, bool, bool) {
	visibility := "public"
	static := false
	final := false
	abstract := false
	for {
		switch p.cur.Type {
		case lexer.Public:
			visibility = "public"
		case lexer.Private:
			visibility = "private"
		case lexer.Protected:
			visibility = "protected"
		case lexer.Static:
			static = true
		case lexer.Final:
			final = true
		case lexer.Abstract:
			abstract = true
		default:
			return visibility, static, final, abstract
		}
		p.next()
	}
}

func (p *Parser) parseClassProperty(visibility string, static bool, final bool, abstract bool) *ast.ClassProperty {
	prop := &ast.ClassProperty{Name: p.cur.Literal, Visibility: visibility, Static: static, Final: final, Abstract: abstract}
	if p.peek.Type == lexer.Assign {
		p.next()
		p.next()
		prop.Default = p.parseExpression(lowest)
	}
	p.consumeSemi()
	return prop
}

func (p *Parser) parseClassConstants() []*ast.ClassConstant {
	var constants []*ast.ClassConstant
	for {
		if !p.expectPeek(lexer.Ident) {
			return constants
		}
		name := p.cur.Literal
		if !p.expectPeek(lexer.Assign) {
			return constants
		}
		p.next()
		value := p.parseExpression(lowest)
		constants = append(constants, &ast.ClassConstant{Name: name, Value: value})
		if p.peek.Type != lexer.Comma {
			break
		}
		p.next()
	}
	p.consumeSemi()
	return constants
}

func (p *Parser) parseThrow() ast.Stmt {
	if p.peek.Type == lexer.Semicolon {
		p.errors = append(p.errors, "throw expects an expression")
		return nil
	}
	p.next()
	value := p.parseExpression(lowest)
	p.consumeSemi()
	return &ast.ThrowStmt{Value: value}
}

func (p *Parser) parseTryCatch() ast.Stmt {
	if !p.expectPeek(lexer.LBrace) {
		return nil
	}
	tryBlock := p.parseBlock().(*ast.BlockStmt)
	var catches []ast.CatchClause
	for p.peek.Type == lexer.Catch {
		p.next()
		clause, ok := p.parseCatchClause()
		if !ok {
			return nil
		}
		catches = append(catches, clause)
	}
	if len(catches) == 0 {
		p.errors = append(p.errors, "try expects at least one catch")
		return nil
	}
	return &ast.TryCatchStmt{TryBlock: tryBlock, Catches: catches}
}

func (p *Parser) parseCatchClause() (ast.CatchClause, bool) {
	if !p.expectPeek(lexer.LParen) {
		return ast.CatchClause{}, false
	}
	if !p.expectPeek(lexer.Ident) {
		return ast.CatchClause{}, false
	}
	catchClass := p.resolveClassName(p.cur.Literal)
	if !p.expectPeek(lexer.Var) {
		return ast.CatchClause{}, false
	}
	catchVar := p.cur.Literal
	if !p.expectPeek(lexer.RParen) {
		return ast.CatchClause{}, false
	}
	if !p.expectPeek(lexer.LBrace) {
		return ast.CatchClause{}, false
	}
	catchBlock := p.parseBlock().(*ast.BlockStmt)
	return ast.CatchClause{Class: catchClass, Var: catchVar, Block: catchBlock}, true
}

func (p *Parser) parseExprOrPropertyAssign() ast.Stmt {
	expr := p.parseExpression(lowest)
	if p.peek.Type == lexer.Assign || isCompoundAssignOp(p.peek.Type) {
		p.next()
		op := p.cur.Type
		p.next()
		value := p.parseExpression(lowest)
		p.consumeSemi()
		if isCompoundAssignOp(op) {
			return &ast.CompoundAssignStmt{Target: expr, Op: compoundBinaryOp(op), Value: value}
		}
		switch target := expr.(type) {
		case *ast.PropertyFetchExpr:
			return &ast.PropertyAssignStmt{Object: target.Object, Property: target.Property, Value: value}
		case *ast.DynamicPropertyFetchExpr:
			return &ast.ExprStmt{Expr: &ast.AssignExpr{Target: target, Op: "=", Value: value}}
		case *ast.StaticPropertyFetchExpr:
			return &ast.StaticPropertyAssignStmt{ClassName: target.ClassName, Property: target.Property, Value: value}
		case *ast.DynamicStaticPropertyFetchExpr:
			return &ast.ExprStmt{Expr: &ast.AssignExpr{Target: target, Op: "=", Value: value}}
		case *ast.IndexExpr:
			return &ast.IndexAssignStmt{Target: target, Value: value}
		default:
			p.errors = append(p.errors, "only variables, object properties and indexes can be assigned")
			return &ast.ExprStmt{Expr: expr}
		}
	}
	if p.peek.Type == lexer.Increment || p.peek.Type == lexer.Decrement {
		p.next()
		op := p.cur.Literal
		p.consumeSemi()
		return &ast.IncDecStmt{Target: expr, Op: op}
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
	case lexer.VarVar:
		left = &ast.VariableVariable{Name: p.cur.Literal}
	case lexer.Ident:
		if strings.EqualFold(p.cur.Literal, "__NAMESPACE__") {
			left = &ast.StringLit{Value: p.currentNamespace}
		} else if isMagicConstant(p.cur.Literal) {
			left = &ast.MagicConstExpr{Name: p.cur.Literal, Line: p.cur.Line}
		} else if p.peek.Type == lexer.ScopeResolution {
			left = p.parseStaticAccess()
		} else if p.peek.Type == lexer.LParen {
			left = p.parseCall()
		} else {
			left = &ast.BareConstExpr{Name: p.resolveConstantName(p.cur.Literal)}
		}
	case lexer.Self, lexer.Parent, lexer.Static:
		if p.peek.Type == lexer.ScopeResolution {
			left = p.parseStaticAccess()
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
		left = &ast.StringLit{Value: p.cur.Literal, Interpolate: p.cur.Interpolate, Backtick: p.cur.Backtick}
	case lexer.True:
		left = &ast.BoolLit{Value: true}
	case lexer.False:
		left = &ast.BoolLit{Value: false}
	case lexer.Null:
		left = &ast.NullLit{}
	case lexer.Array:
		left = p.parseArray()
	case lexer.List:
		left = p.parseList()
	case lexer.Function:
		left = p.parseClosure()
	case lexer.NewKeyword:
		left = p.parseNew()
	case lexer.Clone:
		p.next()
		left = &ast.CloneExpr{Expr: p.parseExpression(prefix)}
	case lexer.Bang, lexer.Minus, lexer.Plus, lexer.Tilde:
		op := p.cur.Literal
		p.next()
		left = &ast.PrefixExpr{Op: op, Right: p.parseExpression(prefix)}
	case lexer.Increment, lexer.Decrement:
		op := p.cur.Literal
		p.next()
		target := p.parseExpression(prefix)
		if !isWritableExpr(target) {
			p.errors = append(p.errors, "expression is not writable")
			return target
		}
		left = &ast.IncDecExpr{Target: target, Op: op}
	case lexer.LParen:
		if castType, ok := castTypeFromToken(p.peek); ok {
			p.next()
			if p.peek.Type == lexer.RParen {
				p.next()
				p.next()
				left = &ast.CastExpr{Type: castType, Expr: p.parseExpression(prefix)}
				break
			}
		}
		p.next()
		left = p.parseExpression(lowest)
		if !p.expectPeek(lexer.RParen) {
			return left
		}
	default:
		p.errors = append(p.errors, fmt.Sprintf("unexpected token %s at %d:%d", p.cur.Type, p.cur.Line, p.cur.Column))
		return &ast.NullLit{}
	}
	left = p.parsePostfix(left)
	for p.peek.Type != lexer.Semicolon && p.peek.Type != lexer.RParen && p.peek.Type != lexer.RBracket && p.peek.Type != lexer.Colon && precedence < p.peekPrecedence() {
		p.next()
		left = p.parseInfix(left)
		left = p.parsePostfix(left)
	}
	return left
}

func (p *Parser) parsePostfix(left ast.Expr) ast.Expr {
	for {
		switch p.peek.Type {
		case lexer.ObjectOp:
			p.next()
			left = p.parseObjectAccess(left)
		case lexer.LBracket:
			p.next()
			left = p.parseIndexAccess(left)
		case lexer.LParen:
			p.next()
			left = &ast.VariableCallExpr{Callee: left, Args: p.parseCallArgs()}
		case lexer.Increment, lexer.Decrement:
			p.next()
			if !isWritableExpr(left) {
				p.errors = append(p.errors, "expression is not writable")
				return left
			}
			left = &ast.IncDecExpr{Target: left, Op: p.cur.Literal, Post: true}
		default:
			return left
		}
	}
}

func (p *Parser) parseStaticAccess() ast.Expr {
	className := p.cur.Literal
	switch p.cur.Type {
	case lexer.Self:
		className = "self"
	case lexer.Parent:
		className = "parent"
	case lexer.Static:
		className = "static"
	default:
		className = p.resolveClassName(className)
	}
	if !p.expectPeek(lexer.ScopeResolution) {
		return &ast.NullLit{}
	}
	switch p.peek.Type {
	case lexer.Var:
		p.next()
		name := &ast.Ident{Name: p.cur.Literal}
		if p.peek.Type == lexer.LParen {
			p.next()
			return &ast.DynamicStaticCallExpr{ClassName: className, Method: name, Args: p.parseCallArgs()}
		}
		return &ast.StaticPropertyFetchExpr{ClassName: className, Property: p.cur.Literal}
	case lexer.VarVar:
		p.next()
		name := p.dynamicMemberName()
		if p.peek.Type == lexer.LParen {
			p.next()
			return &ast.DynamicStaticCallExpr{ClassName: className, Method: name, Args: p.parseCallArgs()}
		}
		return &ast.DynamicStaticPropertyFetchExpr{ClassName: className, Property: name}
	case lexer.Ident:
		p.next()
		name := p.cur.Literal
		if p.peek.Type != lexer.LParen {
			return &ast.ClassConstFetchExpr{ClassName: className, Name: name}
		}
		p.next()
		return &ast.StaticCallExpr{ClassName: className, Method: name, Args: p.parseCallArgs()}
	default:
		p.errors = append(p.errors, fmt.Sprintf("expected static member after %s::", className))
		return &ast.NullLit{}
	}
}

func (p *Parser) parseNew() ast.Expr {
	expr := &ast.NewExpr{Line: p.cur.Line}
	switch p.peek.Type {
	case lexer.Ident, lexer.Self, lexer.Parent, lexer.Static:
		p.next()
		expr.ClassName = p.resolveClassName(p.cur.Literal)
		switch p.cur.Type {
		case lexer.Self:
			expr.ClassName = "self"
		case lexer.Parent:
			expr.ClassName = "parent"
		case lexer.Static:
			expr.ClassName = "static"
		}
	case lexer.Var:
		p.next()
		expr.ClassExpr = &ast.Ident{Name: p.cur.Literal}
	case lexer.VarVar:
		p.next()
		expr.ClassExpr = &ast.VariableVariable{Name: p.cur.Literal}
	default:
		p.errors = append(p.errors, fmt.Sprintf("expected class name after new, got %s", p.peek.Type))
		return expr
	}
	if p.peek.Type != lexer.LParen {
		return expr
	}
	p.next()
	expr.Args = p.parseCallArgs()
	return expr
}

func (p *Parser) parseObjectAccess(object ast.Expr) ast.Expr {
	switch p.peek.Type {
	case lexer.Ident:
		p.next()
		name := p.cur.Literal
		if p.peek.Type == lexer.LParen {
			p.next()
			return &ast.MethodCallExpr{Object: object, Method: name, Args: p.parseCallArgs()}
		}
		return &ast.PropertyFetchExpr{Object: object, Property: name}
	case lexer.Var, lexer.VarVar:
		p.next()
		name := p.dynamicMemberName()
		if p.peek.Type == lexer.LParen {
			p.next()
			return &ast.DynamicMethodCallExpr{Object: object, Method: name, Args: p.parseCallArgs()}
		}
		return &ast.DynamicPropertyFetchExpr{Object: object, Property: name}
	default:
		p.errors = append(p.errors, fmt.Sprintf("expected object member after ->, got %s", p.peek.Type))
		return object
	}
}

func (p *Parser) dynamicMemberName() ast.Expr {
	if p.cur.Type == lexer.VarVar {
		return &ast.VariableVariable{Name: p.cur.Literal}
	}
	return &ast.Ident{Name: p.cur.Literal}
}

func (p *Parser) parseIndexAccess(left ast.Expr) ast.Expr {
	if p.peek.Type == lexer.RBracket {
		p.next()
		return &ast.IndexExpr{Left: left}
	}
	p.next()
	index := p.parseExpression(lowest)
	if !p.expectPeek(lexer.RBracket) {
		return left
	}
	return &ast.IndexExpr{Left: left, Index: index}
}

func (p *Parser) parseCall() ast.Expr {
	call := &ast.CallExpr{Name: p.resolveFunctionName(p.cur.Literal)}
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

func (p *Parser) parseList() ast.Expr {
	lit := &ast.ListExpr{}
	if !p.expectPeek(lexer.LParen) {
		return lit
	}
	for {
		if p.peek.Type == lexer.RParen {
			p.next()
			return lit
		}
		if p.peek.Type == lexer.Comma {
			lit.Targets = append(lit.Targets, nil)
			p.next()
			continue
		}
		p.next()
		target := p.parseExpression(lowest)
		if target != nil && !isWritableExpr(target) {
			p.errors = append(p.errors, "list element is not writable")
		}
		lit.Targets = append(lit.Targets, target)
		if p.peek.Type != lexer.Comma {
			break
		}
		p.next()
	}
	if !p.expectPeek(lexer.RParen) {
		return lit
	}
	return lit
}

func (p *Parser) parseInfix(left ast.Expr) ast.Expr {
	if p.cur.Type == lexer.Question {
		expr := &ast.TernaryExpr{Cond: left}
		if p.peek.Type == lexer.Colon {
			expr.Then = left
		} else {
			p.next()
			expr.Then = p.parseExpression(lowest)
		}
		if !p.expectPeek(lexer.Colon) {
			return expr
		}
		p.next()
		expr.Else = p.parseExpression(ternary - 1)
		return expr
	}
	if p.cur.Type == lexer.Assign || p.cur.Type == lexer.AssignRef || isCompoundAssignOp(p.cur.Type) {
		op := p.cur.Type
		p.next()
		value := p.parseExpression(assignment - 1)
		if !isWritableExpr(left) {
			p.errors = append(p.errors, "expression is not writable")
			return left
		}
		if op == lexer.AssignRef && !isReferenceWritableExpr(left) {
			p.errors = append(p.errors, "reference assignment target must be a variable")
			return left
		}
		if isCompoundAssignOp(op) {
			return &ast.AssignExpr{Target: left, Op: compoundBinaryOp(op), Value: value}
		}
		if op == lexer.AssignRef {
			if !isReferenceWritableExpr(value) {
				p.errors = append(p.errors, "reference assignment value must be a variable")
				return left
			}
			return &ast.AssignExpr{Target: left, Op: "=", Value: value, ByRef: true}
		}
		return &ast.AssignExpr{Target: left, Op: "=", Value: value}
	}
	if p.cur.Type == lexer.InstanceOf {
		expr := &ast.InstanceOfExpr{Left: left}
		switch p.peek.Type {
		case lexer.Ident, lexer.Self, lexer.Parent, lexer.Static:
			p.next()
			expr.ClassName = p.resolveClassName(p.cur.Literal)
			switch p.cur.Type {
			case lexer.Self:
				expr.ClassName = "self"
			case lexer.Parent:
				expr.ClassName = "parent"
			case lexer.Static:
				expr.ClassName = "static"
			}
		case lexer.Var:
			p.next()
			expr.ClassExpr = &ast.Ident{Name: p.cur.Literal}
		case lexer.VarVar:
			p.next()
			expr.ClassExpr = &ast.VariableVariable{Name: p.cur.Literal}
		default:
			p.errors = append(p.errors, fmt.Sprintf("expected class name after instanceof, got %s", p.peek.Type))
		}
		return expr
	}
	expr := &ast.InfixExpr{Left: left, Op: p.cur.Literal}
	prec := p.curPrecedence()
	p.next()
	expr.Right = p.parseExpression(prec)
	return expr
}

func isWritableExpr(expr ast.Expr) bool {
	switch expr.(type) {
	case *ast.Ident, *ast.VariableVariable, *ast.PropertyFetchExpr, *ast.DynamicPropertyFetchExpr, *ast.StaticPropertyFetchExpr, *ast.DynamicStaticPropertyFetchExpr, *ast.IndexExpr, *ast.ListExpr:
		return true
	default:
		return false
	}
}

func isReferenceWritableExpr(expr ast.Expr) bool {
	switch expr.(type) {
	case *ast.Ident, *ast.VariableVariable, *ast.IndexExpr, *ast.CallExpr, *ast.VariableCallExpr:
		return true
	default:
		return false
	}
}

func isMagicConstant(name string) bool {
	switch strings.ToLower(name) {
	case "__file__", "__dir__", "__line__", "__function__", "__class__", "__method__":
		return true
	default:
		return false
	}
}

func tokenIn(t lexer.TokenType, values []lexer.TokenType) bool {
	for _, value := range values {
		if t == value {
			return true
		}
	}
	return false
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

func isCompoundAssignOp(t lexer.TokenType) bool {
	switch t {
	case lexer.PlusAssign, lexer.MinusAssign, lexer.MulAssign, lexer.DivAssign, lexer.ConcatAssign, lexer.ModAssign:
		return true
	default:
		return false
	}
}

func compoundBinaryOp(t lexer.TokenType) string {
	switch t {
	case lexer.PlusAssign:
		return "+"
	case lexer.MinusAssign:
		return "-"
	case lexer.MulAssign:
		return "*"
	case lexer.DivAssign:
		return "/"
	case lexer.ConcatAssign:
		return "."
	case lexer.ModAssign:
		return "%"
	default:
		return ""
	}
}

func castTypeFromToken(tok lexer.Token) (string, bool) {
	switch tok.Type {
	case lexer.Array:
		return "array", true
	case lexer.Unset:
		return "unset", true
	case lexer.Ident:
		return tok.Literal, isCastType(tok.Literal)
	default:
		return "", false
	}
}

func (p *Parser) resolveDeclarationName(name string) string {
	name = trimLeadingNamespace(name)
	if p.currentNamespace == "" || strings.Contains(name, "\\") {
		return name
	}
	return p.currentNamespace + "\\" + name
}

func (p *Parser) resolveClassName(name string) string {
	if name == "self" || name == "parent" || name == "static" {
		return name
	}
	if strings.HasPrefix(name, "\\") {
		return trimLeadingNamespace(name)
	}
	first, rest, hasRest := strings.Cut(name, "\\")
	if imported, ok := p.imports[first]; ok {
		if hasRest {
			return imported + "\\" + rest
		}
		return imported
	}
	if p.currentNamespace == "" || strings.Contains(name, "\\") {
		return name
	}
	return p.currentNamespace + "\\" + name
}

func (p *Parser) resolveFunctionName(name string) string {
	if strings.HasPrefix(name, "\\") {
		return trimLeadingNamespace(name)
	}
	if p.currentNamespace == "" || strings.Contains(name, "\\") {
		return name
	}
	return p.currentNamespace + "\\" + name
}

func (p *Parser) resolveConstantName(name string) string {
	if strings.HasPrefix(name, "\\") {
		return trimLeadingNamespace(name)
	}
	if p.currentNamespace == "" || strings.Contains(name, "\\") {
		return name
	}
	return p.currentNamespace + "\\" + name
}

func trimLeadingNamespace(name string) string {
	return strings.TrimPrefix(name, "\\")
}

func namespaceBaseName(name string) string {
	name = trimLeadingNamespace(name)
	if idx := strings.LastIndex(name, "\\"); idx >= 0 {
		return name[idx+1:]
	}
	return name
}

func isCastType(name string) bool {
	switch name {
	case "int", "integer", "bool", "boolean", "float", "double", "real", "string", "array", "object", "unset":
		return true
	default:
		return false
	}
}
