package runtime

import (
	"bytes"
	"fmt"

	"php-src-go/internal/ast"
)

type Interpreter struct {
	Vars map[string]Value
	Out  bytes.Buffer
}

func NewInterpreter() *Interpreter { return &Interpreter{Vars: map[string]Value{}} }

func (i *Interpreter) Run(p *ast.Program) error {
	for _, s := range p.Statements {
		if err := i.exec(s); err != nil {
			return err
		}
	}
	return nil
}

func (i *Interpreter) exec(s ast.Stmt) error {
	switch n := s.(type) {
	case *ast.BlockStmt:
		for _, st := range n.Statements {
			if err := i.exec(st); err != nil {
				return err
			}
		}
	case *ast.AssignStmt:
		v, err := i.eval(n.Value)
		if err != nil {
			return err
		}
		i.Vars[n.Name] = v
	case *ast.EchoStmt:
		for _, e := range n.Values {
			v, err := i.eval(e)
			if err != nil {
				return err
			}
			i.Out.WriteString(v.String())
		}
	case *ast.IfStmt:
		v, err := i.eval(n.Cond)
		if err != nil {
			return err
		}
		if v.Truthy() {
			return i.exec(n.Consequence)
		} else if n.Alternative != nil {
			return i.exec(n.Alternative)
		}
	case *ast.WhileStmt:
		guard := 0
		for {
			v, err := i.eval(n.Cond)
			if err != nil {
				return err
			}
			if !v.Truthy() {
				break
			}
			if err := i.exec(n.Body); err != nil {
				return err
			}
			guard++
			if guard > 1000000 {
				return fmt.Errorf("while guard exceeded")
			}
		}
	case *ast.ExprStmt:
		_, err := i.eval(n.Expr)
		return err
	default:
		return fmt.Errorf("unsupported statement %T", s)
	}
	return nil
}

func (i *Interpreter) eval(e ast.Expr) (Value, error) {
	switch n := e.(type) {
	case *ast.Ident:
		if v, ok := i.Vars[n.Name]; ok {
			return v, nil
		}
		return NullValue(), nil
	case *ast.IntLit:
		return IntValue(n.Value), nil
	case *ast.FloatLit:
		return FloatValue(n.Value), nil
	case *ast.StringLit:
		return StringValue(n.Value), nil
	case *ast.BoolLit:
		return BoolValue(n.Value), nil
	case *ast.NullLit:
		return NullValue(), nil
	case *ast.PrefixExpr:
		v, err := i.eval(n.Right)
		if err != nil {
			return NullValue(), err
		}
		switch n.Op {
		case "!":
			return BoolValue(!v.Truthy()), nil
		case "-":
			z, err := Binary("-", IntValue(0), v)
			return z, err
		}
	case *ast.InfixExpr:
		l, err := i.eval(n.Left)
		if err != nil {
			return NullValue(), err
		}
		r, err := i.eval(n.Right)
		if err != nil {
			return NullValue(), err
		}
		return Binary(n.Op, l, r)
	}
	return NullValue(), fmt.Errorf("unsupported expression %T", e)
}
