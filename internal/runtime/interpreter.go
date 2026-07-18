package runtime

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	goruntime "runtime"
	"sort"
	"strconv"
	"strings"

	"php-src-go/internal/ast"
	"php-src-go/internal/lexer"
	"php-src-go/internal/parser"
	"php-src-go/internal/runtime/builtins"
)

type Interpreter struct {
	Vars                        map[string]Value
	globals                     map[string]*variableSlot
	functions                   map[string]*ast.FunctionDeclStmt
	classes                     map[string]*ast.ClassDeclStmt
	classAliases                map[string]classAlias
	interfaces                  map[string]*ast.InterfaceDeclStmt
	constants                   map[string]map[string]Value
	declaredClassNames          map[string]string
	runtimeConstants            map[string]Value
	runtimeInsensitiveConstants map[string]Value
	staticVars                  map[string]map[string]*variableSlot
	statics                     map[string]map[string]Value
	staticMeta                  map[string]map[string]PropertyMeta
	frames                      []frame
	closureSeq                  int
	responseHeaders             []string
	uploadedFiles               map[string]bool
	includedOnce                map[string]bool
	autoloading                 map[string]bool
	includePath                 []string
	baseDir                     string
	dirStack                    []string
	fileStack                   []string
	objects                     []*Object
	destructed                  map[*Object]bool
	diagnostics                 []Diagnostic
	errorPolicy                 ErrorPolicy
	Out                         bytes.Buffer
}

type classAlias struct {
	Name   string
	Target string
}

func NewInterpreter() *Interpreter {
	return NewInterpreterWithBaseDir(".")
}

func NewInterpreterWithBaseDir(baseDir string) *Interpreter {
	if baseDir == "" {
		baseDir = "."
	}
	return &Interpreter{
		Vars:                        map[string]Value{},
		globals:                     map[string]*variableSlot{},
		functions:                   map[string]*ast.FunctionDeclStmt{},
		classes:                     map[string]*ast.ClassDeclStmt{classNameKey("Exception"): builtinExceptionClass()},
		classAliases:                map[string]classAlias{},
		interfaces:                  map[string]*ast.InterfaceDeclStmt{},
		constants:                   map[string]map[string]Value{},
		declaredClassNames:          map[string]string{classNameKey("Exception"): "Exception"},
		runtimeConstants:            map[string]Value{},
		runtimeInsensitiveConstants: map[string]Value{},
		staticVars:                  map[string]map[string]*variableSlot{},
		statics:                     map[string]map[string]Value{},
		staticMeta:                  map[string]map[string]PropertyMeta{},
		uploadedFiles:               map[string]bool{},
		includedOnce:                map[string]bool{},
		autoloading:                 map[string]bool{},
		includePath:                 []string{"."},
		baseDir:                     baseDir,
		dirStack:                    []string{baseDir},
		fileStack:                   []string{},
		destructed:                  map[*Object]bool{},
		errorPolicy:                 DefaultErrorPolicy(),
	}
}

func builtinExceptionClass() *ast.ClassDeclStmt {
	return &ast.ClassDeclStmt{
		Name: "Exception",
		Properties: []*ast.ClassProperty{
			{Name: "message", Visibility: "private", Default: &ast.StringLit{Value: ""}, Owner: "Exception"},
			{Name: "code", Visibility: "private", Default: &ast.IntLit{Value: 0}, Owner: "Exception"},
			{Name: "file", Visibility: "private", Default: &ast.StringLit{Value: ""}, Owner: "Exception"},
			{Name: "line", Visibility: "private", Default: &ast.IntLit{Value: 0}, Owner: "Exception"},
			{Name: "previous", Visibility: "private", Default: &ast.NullLit{}, Owner: "Exception"},
		},
	}
}

func (i *Interpreter) SetEntryFile(path string) {
	if path == "" {
		i.fileStack = nil
		return
	}
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	i.fileStack = []string{filepath.Clean(path)}
}

func (i *Interpreter) SetIncludePath(path string) {
	i.includePath = splitIncludePath(path)
}

func (i *Interpreter) Diagnostics() []Diagnostic {
	out := make([]Diagnostic, len(i.diagnostics))
	copy(out, i.diagnostics)
	return out
}

func (i *Interpreter) SetErrorPolicy(policy ErrorPolicy) {
	i.errorPolicy = policy
}

func (i *Interpreter) ErrorPolicy() ErrorPolicy {
	return i.errorPolicy
}

func (i *Interpreter) Run(p *ast.Program) error {
	_, _, err := i.runProgramResult(p, true)
	return err
}

func (i *Interpreter) runProgram(p *ast.Program, shutdown bool) error {
	_, _, err := i.runProgramResult(p, shutdown)
	return err
}

func (i *Interpreter) runProgramResult(p *ast.Program, shutdown bool) (Value, bool, error) {
	statements := flattenNamespaceBlocks(p.Statements)
	for _, s := range statements {
		if class, ok := s.(*ast.ClassDeclStmt); ok {
			i.declaredClassNames[classNameKey(class.Name)] = class.Name
		}
	}
	for _, s := range statements {
		if fn, ok := s.(*ast.FunctionDeclStmt); ok {
			if err := i.defineFunction(fn); err != nil {
				return NullValue(), false, err
			}
		}
	}
	for _, s := range statements {
		if iface, ok := s.(*ast.InterfaceDeclStmt); ok {
			if err := i.defineInterface(iface); err != nil {
				return NullValue(), false, err
			}
		}
	}
	for _, s := range statements {
		if class, ok := s.(*ast.ClassDeclStmt); ok {
			if err := i.defineClass(class); err != nil {
				return NullValue(), false, err
			}
		}
	}
	for _, s := range statements {
		if _, ok := s.(*ast.FunctionDeclStmt); ok {
			continue
		}
		if _, ok := s.(*ast.ClassDeclStmt); ok {
			continue
		}
		if _, ok := s.(*ast.InterfaceDeclStmt); ok {
			continue
		}
		if ret, err := i.exec(s); err != nil {
			if _, ok := err.(*ExitError); ok && shutdown {
				if shutdownErr := i.runDestructors(); shutdownErr != nil {
					return NullValue(), false, shutdownErr
				}
			}
			return NullValue(), false, err
		} else if ret != nil {
			if shutdown {
				if shutdownErr := i.runDestructors(); shutdownErr != nil {
					return NullValue(), false, shutdownErr
				}
			}
			return ret.value, true, nil
		}
	}
	if shutdown {
		if err := i.runDestructors(); err != nil {
			return NullValue(), false, err
		}
	}
	return NullValue(), false, nil
}

func flattenNamespaceBlocks(statements []ast.Stmt) []ast.Stmt {
	var out []ast.Stmt
	for _, s := range statements {
		if block, ok := s.(*ast.NamespaceBlockStmt); ok {
			out = append(out, block.Statements...)
			continue
		}
		out = append(out, s)
	}
	return out
}

type returnSignal struct {
	value Value
	slot  *variableSlot
}

type variableSlot struct {
	value      Value
	globalName string
}

type breakSignal struct {
	Level int
}

func (*breakSignal) Error() string { return "break used outside loop or switch" }

type continueSignal struct {
	Level int
}

func (*continueSignal) Error() string { return "continue used outside loop" }

type ExitError struct {
	Code int
}

func (*ExitError) Error() string { return "exit" }

type frame struct {
	vars        map[string]*variableSlot
	className   string
	calledClass string
	function    string
	functionKey string
	returnByRef bool
}

type functionCallResult struct {
	value Value
	slot  *variableSlot
}

type thrownError struct {
	value Value
}

func (e *thrownError) Error() string {
	if e.value.Kind == ObjectKind && e.value.O != nil {
		message := exceptionProperty(e.value.O, "message", NullValue())
		if message.Kind == Null {
			return "uncaught " + e.value.O.ClassName
		}
		return "uncaught " + e.value.O.ClassName + ": " + message.String()
	}
	return "uncaught non-object"
}

func (i *Interpreter) exec(s ast.Stmt) (*returnSignal, error) {
	switch n := s.(type) {
	case *ast.BlockStmt:
		for _, st := range n.Statements {
			ret, err := i.exec(st)
			if err != nil || ret != nil {
				return ret, err
			}
		}
	case *ast.NamespaceBlockStmt:
		for _, st := range flattenNamespaceBlocks(n.Statements) {
			if _, ok := st.(*ast.FunctionDeclStmt); ok {
				continue
			}
			if _, ok := st.(*ast.ClassDeclStmt); ok {
				continue
			}
			if _, ok := st.(*ast.InterfaceDeclStmt); ok {
				continue
			}
			ret, err := i.exec(st)
			if err != nil || ret != nil {
				return ret, err
			}
		}
	case *ast.AssignStmt:
		v, err := i.eval(n.Value)
		if err != nil {
			return nil, err
		}
		i.setVar(n.Name, v)
	case *ast.CompoundAssignStmt:
		current, err := i.eval(n.Target)
		if err != nil {
			return nil, err
		}
		value, err := i.eval(n.Value)
		if err != nil {
			return nil, err
		}
		var result Value
		if n.Op == "." {
			left, err := i.stringify(current)
			if err != nil {
				return nil, err
			}
			right, err := i.stringify(value)
			if err != nil {
				return nil, err
			}
			result = StringValue(left + right)
		} else {
			result, err = i.binary(n.Op, current, value)
			if err != nil {
				return nil, err
			}
		}
		if err := i.assignTarget(n.Target, result); err != nil {
			return nil, err
		}
	case *ast.IncDecStmt:
		if _, err := i.evalIncDec(n.Target, n.Op, false); err != nil {
			return nil, err
		}
	case *ast.EchoStmt:
		for _, e := range n.Values {
			v, err := i.eval(e)
			if err != nil {
				return nil, err
			}
			text, err := i.stringify(v)
			if err != nil {
				return nil, err
			}
			i.Out.WriteString(text)
		}
	case *ast.UnsetStmt:
		for _, target := range n.Targets {
			if err := i.unsetTarget(target); err != nil {
				return nil, err
			}
		}
	case *ast.ExitStmt:
		code := 0
		if n.Value != nil {
			value, err := i.eval(n.Value)
			if err != nil {
				return nil, err
			}
			if value.Kind == Int {
				code = int(value.I)
			} else {
				text, err := i.stringify(value)
				if err != nil {
					return nil, err
				}
				i.Out.WriteString(text)
			}
		}
		return nil, &ExitError{Code: code}
	case *ast.IfStmt:
		v, err := i.eval(n.Cond)
		if err != nil {
			return nil, err
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
				return nil, err
			}
			if !v.Truthy() {
				break
			}
			ret, err := i.exec(n.Body)
			if handled, propagated, ok := consumeBreak(err); ok {
				if handled {
					break
				}
				return nil, propagated
			}
			if handled, propagated, ok := consumeContinue(err); ok {
				if !handled {
					return nil, propagated
				}
				guard++
				if guard > 1000000 {
					return nil, fmt.Errorf("while guard exceeded")
				}
				continue
			}
			if err != nil || ret != nil {
				return ret, err
			}
			guard++
			if guard > 1000000 {
				return nil, fmt.Errorf("while guard exceeded")
			}
		}
	case *ast.DoWhileStmt:
		guard := 0
		for {
			ret, err := i.exec(n.Body)
			if handled, propagated, ok := consumeBreak(err); ok {
				if handled {
					break
				}
				return nil, propagated
			}
			if handled, propagated, ok := consumeContinue(err); ok {
				if !handled {
					return nil, propagated
				}
			} else {
				if err != nil || ret != nil {
					return ret, err
				}
			}
			guard++
			if guard > 1000000 {
				return nil, fmt.Errorf("do/while guard exceeded")
			}
			v, err := i.eval(n.Cond)
			if err != nil {
				return nil, err
			}
			if !v.Truthy() {
				break
			}
		}
	case *ast.ForStmt:
		if n.Init != nil {
			if ret, err := i.exec(n.Init); err != nil || ret != nil {
				return ret, err
			}
		}
		guard := 0
		for {
			if n.Cond != nil {
				v, err := i.eval(n.Cond)
				if err != nil {
					return nil, err
				}
				if !v.Truthy() {
					break
				}
			}
			ret, err := i.exec(n.Body)
			if handled, propagated, ok := consumeBreak(err); ok {
				if handled {
					break
				}
				return nil, propagated
			}
			if handled, propagated, ok := consumeContinue(err); ok {
				if !handled {
					return nil, propagated
				}
			} else {
				if err != nil || ret != nil {
					return ret, err
				}
			}
			if n.Post != nil {
				if ret, err := i.exec(n.Post); err != nil || ret != nil {
					return ret, err
				}
			}
			guard++
			if guard > 1000000 {
				return nil, fmt.Errorf("for guard exceeded")
			}
		}
	case *ast.ForeachStmt:
		src, err := i.eval(n.Source)
		if err != nil {
			return nil, err
		}
		if src.Kind != ArrayKind || src.A == nil {
			if err := i.emitDiagnostic(Warning, "foreach source must be array"); err != nil {
				return nil, err
			}
			return nil, nil
		}
		for _, entry := range src.A.Entries {
			if n.KeyVar != "" {
				i.setVar(n.KeyVar, entry.Key.Value())
			}
			if n.ValueByRef {
				i.bindSlot(n.ValueVar, src.A.EnsureSlot(entry.Key))
			} else {
				i.setVar(n.ValueVar, entry.RuntimeValue())
			}
			ret, err := i.exec(n.Body)
			if handled, propagated, ok := consumeBreak(err); ok {
				if handled {
					break
				}
				return nil, propagated
			}
			if handled, propagated, ok := consumeContinue(err); ok {
				if !handled {
					return nil, propagated
				}
				continue
			}
			if err != nil || ret != nil {
				return ret, err
			}
		}
	case *ast.SwitchStmt:
		ret, err := i.execSwitch(n)
		if ret != nil {
			return ret, nil
		}
		if err != nil {
			if handled, propagated, ok := consumeBreak(err); ok {
				if handled {
					return nil, nil
				}
				return nil, propagated
			}
			if handled, propagated, ok := consumeContinue(err); ok {
				if handled {
					return nil, nil
				}
				return nil, propagated
			}
			return nil, err
		}
	case *ast.FunctionDeclStmt:
		if err := i.defineFunction(n); err != nil {
			return nil, err
		}
	case *ast.ClassDeclStmt:
		if err := i.defineClass(n); err != nil {
			return nil, err
		}
	case *ast.InterfaceDeclStmt:
		if err := i.defineInterface(n); err != nil {
			return nil, err
		}
	case *ast.PropertyAssignStmt:
		value, err := i.eval(n.Value)
		if err != nil {
			return nil, err
		}
		if err := i.assignTarget(&ast.PropertyFetchExpr{Object: n.Object, Property: n.Property}, value); err != nil {
			return nil, err
		}
	case *ast.StaticPropertyAssignStmt:
		value, err := i.eval(n.Value)
		if err != nil {
			return nil, err
		}
		if err := i.assignTarget(&ast.StaticPropertyFetchExpr{ClassName: n.ClassName, Property: n.Property}, value); err != nil {
			return nil, err
		}
	case *ast.IndexAssignStmt:
		value, err := i.eval(n.Value)
		if err != nil {
			return nil, err
		}
		if err := i.assignTarget(n.Target, value); err != nil {
			return nil, err
		}
	case *ast.ThrowStmt:
		value, err := i.eval(n.Value)
		if err != nil {
			return nil, err
		}
		if value.Kind != ObjectKind || value.O == nil {
			return nil, fmt.Errorf("can only throw objects")
		}
		return nil, &thrownError{value: value}
	case *ast.TryCatchStmt:
		ret, err := i.exec(n.TryBlock)
		if err == nil || ret != nil {
			return ret, err
		}
		thrown, ok := err.(*thrownError)
		if !ok {
			return nil, err
		}
		for _, catch := range n.Catches {
			if !i.exceptionMatches(thrown.value, catch.Class) {
				continue
			}
			i.setVar(catch.Var, thrown.value)
			return i.exec(catch.Block)
		}
		return nil, err
	case *ast.ReturnStmt:
		if n.Value == nil {
			return &returnSignal{value: NullValue()}, nil
		}
		if i.currentReturnByRef() {
			slot, err := i.referenceSlot(n.Value)
			if err != nil {
				return nil, err
			}
			return &returnSignal{value: slot.value, slot: slot}, nil
		}
		v, err := i.eval(n.Value)
		if err != nil {
			return nil, err
		}
		return &returnSignal{value: v}, nil
	case *ast.GlobalStmt:
		for _, name := range n.Names {
			i.bindGlobal(name)
		}
	case *ast.StaticLocalStmt:
		if err := i.bindStaticLocals(n.Decls); err != nil {
			return nil, err
		}
	case *ast.IncludeStmt:
		path, err := i.eval(n.Path)
		if err != nil {
			return nil, err
		}
		if path.Kind != String {
			if err := i.emitDiagnostic(FatalError, "include/require path must be string"); err != nil {
				return nil, err
			}
			return nil, nil
		}
		if _, err := i.include(path.S, n.Require, n.Once); err != nil {
			return nil, err
		}
	case *ast.ExprStmt:
		_, err := i.eval(n.Expr)
		return nil, err
	case *ast.BreakStmt:
		if n.Level < 1 {
			return nil, fmt.Errorf("break level must be a positive integer")
		}
		return nil, &breakSignal{Level: n.Level}
	case *ast.ContinueStmt:
		if n.Level < 1 {
			return nil, fmt.Errorf("continue level must be a positive integer")
		}
		return nil, &continueSignal{Level: n.Level}
	default:
		return nil, fmt.Errorf("unsupported statement %T", s)
	}
	return nil, nil
}

func (i *Interpreter) eval(e ast.Expr) (Value, error) {
	switch n := e.(type) {
	case *ast.Ident:
		if v, ok := i.getVar(n.Name); ok {
			return v, nil
		}
		if err := i.noticef("undefined variable %s", n.Name); err != nil {
			return NullValue(), err
		}
		return NullValue(), nil
	case *ast.VariableVariable:
		nameValue, ok := i.getVar(n.Name)
		if !ok {
			if err := i.noticef("undefined variable %s", n.Name); err != nil {
				return NullValue(), err
			}
			return NullValue(), nil
		}
		name := nameValue.String()
		if v, ok := i.getVar(name); ok {
			return v, nil
		}
		if err := i.noticef("undefined variable %s", name); err != nil {
			return NullValue(), err
		}
		return NullValue(), nil
	case *ast.IntLit:
		return IntValue(n.Value), nil
	case *ast.FloatLit:
		return FloatValue(n.Value), nil
	case *ast.StringLit:
		if n.Backtick {
			text := n.Value
			if n.Interpolate {
				var err error
				text, err = i.interpolateString(text)
				if err != nil {
					return NullValue(), err
				}
			}
			return i.evalBacktick(text)
		}
		if n.Interpolate {
			text, err := i.interpolateString(n.Value)
			if err != nil {
				return NullValue(), err
			}
			return StringValue(text), nil
		}
		return StringValue(n.Value), nil
	case *ast.BoolLit:
		return BoolValue(n.Value), nil
	case *ast.NullLit:
		return NullValue(), nil
	case *ast.MagicConstExpr:
		return i.evalMagicConst(n), nil
	case *ast.BareConstExpr:
		return i.runtimeConstantValue(n.Name), nil
	case *ast.PrefixExpr:
		v, err := i.eval(n.Right)
		if err != nil {
			return NullValue(), err
		}
		switch n.Op {
		case "!":
			return BoolValue(!v.Truthy()), nil
		case "+":
			z, err := Binary("+", IntValue(0), v)
			return z, err
		case "-":
			z, err := Binary("-", IntValue(0), v)
			return z, err
		case "~":
			z, err := BitwiseNot(v)
			return z, err
		}
	case *ast.CloneExpr:
		return i.evalClone(n.Expr)
	case *ast.IncludeExpr:
		path, err := i.eval(n.Path)
		if err != nil {
			return NullValue(), err
		}
		if path.Kind != String {
			if err := i.emitDiagnostic(FatalError, "include/require path must be string"); err != nil {
				return NullValue(), err
			}
			return BoolValue(false), nil
		}
		return i.include(path.S, n.Require, n.Once)
	case *ast.IncDecExpr:
		return i.evalIncDec(n.Target, n.Op, n.Post)
	case *ast.InfixExpr:
		l, err := i.eval(n.Left)
		if err != nil {
			return NullValue(), err
		}
		switch n.Op {
		case "&&", "and":
			if !l.Truthy() {
				return BoolValue(false), nil
			}
			r, err := i.eval(n.Right)
			if err != nil {
				return NullValue(), err
			}
			return BoolValue(r.Truthy()), nil
		case "||", "or":
			if l.Truthy() {
				return BoolValue(true), nil
			}
			r, err := i.eval(n.Right)
			if err != nil {
				return NullValue(), err
			}
			return BoolValue(r.Truthy()), nil
		case "xor":
			r, err := i.eval(n.Right)
			if err != nil {
				return NullValue(), err
			}
			return BoolValue(l.Truthy() != r.Truthy()), nil
		}
		r, err := i.eval(n.Right)
		if err != nil {
			return NullValue(), err
		}
		if n.Op == "." {
			left, err := i.stringify(l)
			if err != nil {
				return NullValue(), err
			}
			right, err := i.stringify(r)
			if err != nil {
				return NullValue(), err
			}
			return StringValue(left + right), nil
		}
		return i.binary(n.Op, l, r)
	case *ast.TernaryExpr:
		cond, err := i.eval(n.Cond)
		if err != nil {
			return NullValue(), err
		}
		if cond.Truthy() {
			if n.Then == nil {
				return cond, nil
			}
			return i.eval(n.Then)
		}
		if n.Else == nil {
			return NullValue(), nil
		}
		return i.eval(n.Else)
	case *ast.AssignExpr:
		return i.evalAssignExpr(n)
	case *ast.ListExpr:
		return NullValue(), nil
	case *ast.CastExpr:
		value, err := i.eval(n.Expr)
		if err != nil {
			return NullValue(), err
		}
		return i.castValue(n.Type, value)
	case *ast.InstanceOfExpr:
		value, err := i.eval(n.Left)
		if err != nil {
			return NullValue(), err
		}
		className, err := i.instanceOfClassName(n)
		if err != nil {
			return NullValue(), err
		}
		return BoolValue(i.valueInstanceOf(value, className)), nil
	case *ast.ArrayLit:
		arr := NewArray()
		for _, elem := range n.Elements {
			value, err := i.eval(elem.Value)
			if err != nil {
				return NullValue(), err
			}
			if elem.Key == nil {
				arr.Append(value)
				continue
			}
			key, err := i.eval(elem.Key)
			if err != nil {
				return NullValue(), err
			}
			arr.Set(KeyFromValue(key), value)
		}
		return ArrayValue(arr), nil
	case *ast.IndexExpr:
		return i.evalIndex(n)
	case *ast.CallExpr:
		return i.call(n)
	case *ast.VariableCallExpr:
		return i.callVariable(n)
	case *ast.ClosureExpr:
		return i.evalClosure(n)
	case *ast.NewExpr:
		return i.newObject(n)
	case *ast.PropertyFetchExpr:
		return i.fetchProperty(n.Object, n.Property)
	case *ast.DynamicPropertyFetchExpr:
		property, err := i.dynamicMemberName(n.Property)
		if err != nil {
			return NullValue(), err
		}
		return i.fetchProperty(n.Object, property)
	case *ast.MethodCallExpr:
		return i.callMethod(n)
	case *ast.DynamicMethodCallExpr:
		return i.callDynamicMethod(n)
	case *ast.StaticPropertyFetchExpr:
		return i.fetchStaticProperty(n.ClassName, n.Property)
	case *ast.DynamicStaticPropertyFetchExpr:
		property, err := i.dynamicMemberName(n.Property)
		if err != nil {
			return NullValue(), err
		}
		return i.fetchStaticProperty(n.ClassName, property)
	case *ast.ClassConstFetchExpr:
		return i.getClassConstant(n.ClassName, n.Name)
	case *ast.StaticCallExpr:
		return i.callStatic(n)
	case *ast.DynamicStaticCallExpr:
		return i.callDynamicStatic(n)
	}
	return NullValue(), fmt.Errorf("unsupported expression %T", e)
}

func (i *Interpreter) instanceOfClassName(n *ast.InstanceOfExpr) (string, error) {
	if n.ClassExpr == nil {
		return i.resolveClassNameAutoload(n.ClassName)
	}
	value, err := i.eval(n.ClassExpr)
	if err != nil {
		return "", err
	}
	if value.Kind == ObjectKind && value.O != nil {
		return value.O.ClassName, nil
	}
	name := value.String()
	if name == "" {
		return "", fmt.Errorf("instanceof class name must not be empty")
	}
	return i.resolveClassNameAutoload(name)
}

func (i *Interpreter) evalBacktick(command string) (Value, error) {
	var cmd *exec.Cmd
	if goruntime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", command)
	} else {
		cmd = exec.Command("sh", "-c", command)
	}
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && len(exitErr.Stderr) == 0 {
			return StringValue(string(out)), nil
		}
		return NullValue(), err
	}
	return StringValue(string(out)), nil
}

func (i *Interpreter) evalClosure(c *ast.ClosureExpr) (Value, error) {
	i.closureSeq++
	fn := &ast.FunctionDeclStmt{
		Name:         fmt.Sprintf("{closure#%d}", i.closureSeq),
		Params:       c.Params,
		Body:         c.Body,
		ReturnsByRef: c.ReturnsByRef,
	}
	if err := validateFunctionCompatibility(fn, "closure"); err != nil {
		return NullValue(), err
	}
	captures := map[string]*variableSlot{}
	for _, use := range c.Uses {
		if use.ByRef {
			captures[use.Name] = i.ensureSlot(use.Name)
			continue
		}
		value, ok := i.getVar(use.Name)
		if !ok {
			value = NullValue()
		}
		captures[use.Name] = &variableSlot{value: value}
	}
	return ClosureValue(&Closure{Fn: fn, Captures: captures}), nil
}

func (i *Interpreter) evalMagicConst(c *ast.MagicConstExpr) Value {
	switch strings.ToLower(c.Name) {
	case "__file__":
		return StringValue(i.currentFile())
	case "__dir__":
		file := i.currentFile()
		if file == "" {
			return StringValue("")
		}
		return StringValue(filepath.Dir(file))
	case "__line__":
		return IntValue(int64(c.Line))
	case "__function__":
		return StringValue(i.currentFunction())
	case "__class__":
		return StringValue(i.currentClass())
	case "__method__":
		return StringValue(i.currentMethod())
	default:
		return NullValue()
	}
}

func (i *Interpreter) runtimeConstantValue(name string) Value {
	value, ok := i.lookupRuntimeConstant(name)
	if !ok {
		return StringValue(name)
	}
	return value
}

func (i *Interpreter) lookupRuntimeConstant(name string) (Value, bool) {
	switch strings.ToLower(name) {
	case "true":
		return BoolValue(true), true
	case "false":
		return BoolValue(false), true
	case "null":
		return NullValue(), true
	}
	if value, ok := i.runtimeConstants[name]; ok {
		return value, true
	}
	if value, ok := i.runtimeInsensitiveConstants[strings.ToLower(name)]; ok {
		return value, true
	}
	if base := namespaceBaseName(name); base != name {
		return i.lookupRuntimeConstant(base)
	}
	return NullValue(), false
}

func (i *Interpreter) runtimeConstantNameTaken(name string, insensitive bool) bool {
	if _, ok := i.lookupRuntimeConstant(name); ok {
		return true
	}
	if !insensitive {
		return false
	}
	needle := strings.ToLower(name)
	for existing := range i.runtimeConstants {
		if strings.ToLower(existing) == needle {
			return true
		}
	}
	_, ok := i.runtimeInsensitiveConstants[needle]
	return ok
}

func (i *Interpreter) lookupNamedConstant(name string) (Value, bool) {
	if value, ok := i.lookupRuntimeConstant(name); ok {
		return value, true
	}
	className, constName, ok := strings.Cut(name, "::")
	if !ok || className == "" || constName == "" {
		return NullValue(), false
	}
	resolved, err := i.resolveClassNameAutoload(className)
	if err != nil {
		return NullValue(), false
	}
	return i.findClassConstant(resolved, constName)
}

func (i *Interpreter) call(c *ast.CallExpr) (Value, error) {
	if value, ok, err := i.callBuiltin(c); ok || err != nil {
		return value, err
	}
	fn, ok := i.functions[functionNameKey(c.Name)]
	if !ok {
		if base := namespaceBaseName(c.Name); base != c.Name {
			return i.call(&ast.CallExpr{Name: base, Args: c.Args})
		}
		return NullValue(), fmt.Errorf("undefined function %s", c.Name)
	}
	return i.callFunction(fn, c.Args, nil, "")
}

func (i *Interpreter) callVariable(c *ast.VariableCallExpr) (Value, error) {
	callee, err := i.eval(c.Callee)
	if err != nil {
		return NullValue(), err
	}
	if callee.Kind == ClosureKind {
		return i.callClosure(callee.C, c.Args)
	}
	if callee.Kind == ObjectKind && callee.O != nil {
		return i.callInvokableObject(callee.O, c.Args)
	}
	return i.call(&ast.CallExpr{Name: callee.String(), Args: c.Args})
}

func (i *Interpreter) dynamicMemberName(expr ast.Expr) (string, error) {
	value, err := i.eval(expr)
	if err != nil {
		return "", err
	}
	return value.String(), nil
}

func (i *Interpreter) callBuiltin(c *ast.CallExpr) (Value, bool, error) {
	switch strings.ToLower(c.Name) {
	case "isset":
		return i.callIsset(c.Args)
	case "empty":
		return i.callEmpty(c.Args)
	case "define":
		return i.callDefine(c.Args)
	case "defined":
		return i.callDefined(c.Args)
	case "constant":
		return i.callConstant(c.Args)
	case "serialize":
		return i.callSerialize(c.Args)
	case "unserialize":
		return i.callUnserialize(c.Args)
	case "class_exists":
		return i.callClassExists(c.Args)
	case "interface_exists":
		return i.callInterfaceExists(c.Args)
	case "get_class":
		return i.callGetClass(c.Args)
	case "get_called_class":
		return i.callGetCalledClass(c.Args)
	case "get_object_vars":
		return i.callGetObjectVars(c.Args)
	case "get_class_methods":
		return i.callGetClassMethods(c.Args)
	case "get_class_vars":
		return i.callGetClassVars(c.Args)
	case "get_declared_classes":
		return i.callGetDeclaredClasses(c.Args)
	case "get_declared_interfaces":
		return i.callGetDeclaredInterfaces(c.Args)
	case "class_alias":
		return i.callClassAlias(c.Args)
	case "is_callable":
		return i.callIsCallable(c.Args)
	case "call_user_func":
		return i.callUserFunc(c.Args)
	case "call_user_func_array":
		return i.callUserFuncArray(c.Args)
	case "method_exists":
		return i.callMethodExists(c.Args)
	case "property_exists":
		return i.callPropertyExists(c.Args)
	case "get_parent_class":
		return i.callGetParentClass(c.Args)
	case "is_subclass_of":
		return i.callIsSubclassOf(c.Args)
	case "is_a":
		return i.callIsA(c.Args)
	case "class_parents":
		return i.callClassParents(c.Args)
	case "class_implements":
		return i.callClassImplements(c.Args)
	}

	if strings.EqualFold(c.Name, "preg_match") && len(c.Args) >= 3 {
		value, err := i.callPregMatchWithCaptures(c)
		return value, true, err
	}

	args := make([]builtins.Value, 0, len(c.Args))
	runtimeArgs := make([]Value, 0, len(c.Args))
	for _, arg := range c.Args {
		value, err := i.eval(arg)
		if err != nil {
			return NullValue(), true, err
		}
		runtimeArgs = append(runtimeArgs, value)
		args = append(args, toBuiltinValue(value))
	}
	if value, ok, err := i.callSAPIBuiltin(c.Name, runtimeArgs); ok || err != nil {
		if err != nil {
			return i.handleBuiltinWarning(err, value, ok)
		}
		return value, ok, nil
	}
	if value, output, ok, err := builtins.CallWithOutput(c.Name, args); ok || err != nil {
		if output != "" {
			i.Out.WriteString(output)
		}
		if err != nil {
			return i.handleBuiltinWarning(err, fromBuiltinValue(value), true)
		}
		return fromBuiltinValue(value), true, nil
	}
	value, ok, err := builtins.Call(c.Name, args)
	if err != nil {
		return i.handleBuiltinWarning(err, fromBuiltinValue(value), ok)
	}
	if !ok {
		return NullValue(), false, nil
	}
	return fromBuiltinValue(value), true, nil
}

func (i *Interpreter) handleBuiltinWarning(err error, value Value, ok bool) (Value, bool, error) {
	if err == nil {
		return value, ok, nil
	}
	if diagErr := i.emitDiagnostic(Warning, err.Error()); diagErr != nil {
		return NullValue(), ok, diagErr
	}
	return value, ok, nil
}

func (i *Interpreter) builtinWarningf(value Value, ok bool, format string, args ...any) (Value, bool, error) {
	return i.handleBuiltinWarning(fmt.Errorf(format, args...), value, ok)
}

func (i *Interpreter) recoverableErrorf(format string, args ...any) error {
	return i.emitDiagnostic(RecoverableError, fmt.Sprintf(format, args...))
}

func (i *Interpreter) noticef(format string, args ...any) error {
	return i.emitDiagnostic(Notice, fmt.Sprintf(format, args...))
}

func isCallableWarning(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	if msg == "invalid callable" || msg == "object is not callable" {
		return true
	}
	return strings.HasPrefix(msg, "object of class ") && strings.HasSuffix(msg, " is not callable")
}

func isSerializedFormatWarning(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return msg == "unexpected end of serialized value" ||
		msg == "serialized string length out of range" ||
		msg == "serialized class name length out of range" ||
		strings.HasPrefix(msg, "unsupported serialized type ") ||
		strings.HasPrefix(msg, "delimiter ") ||
		strings.HasPrefix(msg, "expected ") ||
		strings.HasPrefix(msg, "strconv.")
}

func (i *Interpreter) callDefine(args []ast.Expr) (Value, bool, error) {
	if len(args) < 2 || len(args) > 3 {
		return i.builtinWarningf(NullValue(), true, "define expects 2 or 3 arguments, got %d", len(args))
	}
	nameValue, err := i.eval(args[0])
	if err != nil {
		return NullValue(), true, err
	}
	value, err := i.eval(args[1])
	if err != nil {
		return NullValue(), true, err
	}
	insensitive := false
	if len(args) == 3 {
		flag, err := i.eval(args[2])
		if err != nil {
			return NullValue(), true, err
		}
		insensitive = flag.Truthy()
	}
	name := nameValue.String()
	if name == "" || strings.Contains(name, "::") {
		return BoolValue(false), true, nil
	}
	if i.runtimeConstantNameTaken(name, insensitive) {
		return BoolValue(false), true, nil
	}
	if insensitive {
		key := strings.ToLower(name)
		i.runtimeInsensitiveConstants[key] = value
		return BoolValue(true), true, nil
	}
	i.runtimeConstants[name] = value
	return BoolValue(true), true, nil
}

func (i *Interpreter) callClassExists(args []ast.Expr) (Value, bool, error) {
	if len(args) < 1 || len(args) > 2 {
		return i.builtinWarningf(NullValue(), true, "class_exists expects 1 or 2 arguments, got %d", len(args))
	}
	nameValue, err := i.eval(args[0])
	if err != nil {
		return NullValue(), true, err
	}
	name := nameValue.String()
	autoload := true
	if len(args) == 2 {
		autoloadValue, err := i.eval(args[1])
		if err != nil {
			return NullValue(), true, err
		}
		autoload = autoloadValue.Truthy()
	}
	if autoload {
		if err := i.autoloadClass(name); err != nil {
			return NullValue(), true, err
		}
	}
	_, ok := i.classByName(name)
	return BoolValue(ok), true, nil
}

func (i *Interpreter) callInterfaceExists(args []ast.Expr) (Value, bool, error) {
	if len(args) < 1 || len(args) > 2 {
		return i.builtinWarningf(NullValue(), true, "interface_exists expects 1 or 2 arguments, got %d", len(args))
	}
	nameValue, err := i.eval(args[0])
	if err != nil {
		return NullValue(), true, err
	}
	name := nameValue.String()
	autoload := true
	if len(args) == 2 {
		autoloadValue, err := i.eval(args[1])
		if err != nil {
			return NullValue(), true, err
		}
		autoload = autoloadValue.Truthy()
	}
	if autoload {
		if err := i.autoloadClass(name); err != nil {
			return NullValue(), true, err
		}
	}
	_, ok := i.interfaceByName(name)
	return BoolValue(ok), true, nil
}

func (i *Interpreter) callGetClass(args []ast.Expr) (Value, bool, error) {
	if len(args) > 1 {
		return i.builtinWarningf(NullValue(), true, "get_class expects 0 or 1 arguments, got %d", len(args))
	}
	if len(args) == 0 {
		if thisObj := i.currentThis(); thisObj != nil {
			return StringValue(thisObj.ClassName), true, nil
		}
		return i.builtinWarningf(NullValue(), true, "get_class expects object")
	}
	value, err := i.eval(args[0])
	if err != nil {
		return NullValue(), true, err
	}
	if value.Kind != ObjectKind || value.O == nil {
		return i.builtinWarningf(NullValue(), true, "get_class expects object")
	}
	return StringValue(value.O.ClassName), true, nil
}

func (i *Interpreter) callGetCalledClass(args []ast.Expr) (Value, bool, error) {
	if len(args) != 0 {
		return i.builtinWarningf(NullValue(), true, "get_called_class expects 0 arguments, got %d", len(args))
	}
	if called := i.currentCalledClass(); called != "" {
		return StringValue(called), true, nil
	}
	return i.builtinWarningf(NullValue(), true, "get_called_class used outside class context")
}

func (i *Interpreter) callGetObjectVars(args []ast.Expr) (Value, bool, error) {
	if len(args) != 1 {
		return i.builtinWarningf(NullValue(), true, "get_object_vars expects 1 argument, got %d", len(args))
	}
	value, err := i.eval(args[0])
	if err != nil {
		return NullValue(), true, err
	}
	if value.Kind != ObjectKind || value.O == nil {
		return i.builtinWarningf(NullValue(), true, "get_object_vars expects object")
	}
	result := NewArray()
	names := make([]string, 0, len(value.O.Props))
	for name := range value.O.Props {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		info, hasMeta := value.O.PropMeta[name]
		if hasMeta && !i.propertyAccessible(info) {
			continue
		}
		result.Set(StringKey(propertyNameForKey(name, info)), value.O.Props[name])
	}
	return ArrayValue(result), true, nil
}

func (i *Interpreter) callGetClassMethods(args []ast.Expr) (Value, bool, error) {
	if len(args) != 1 {
		return i.builtinWarningf(NullValue(), true, "get_class_methods expects 1 argument, got %d", len(args))
	}
	target, err := i.eval(args[0])
	if err != nil {
		return NullValue(), true, err
	}
	if target.Kind == String {
		if err := i.autoloadClass(target.String()); err != nil {
			return NullValue(), true, err
		}
	}
	methods, ok := i.methodsForTarget(target)
	if !ok {
		return BoolValue(false), true, nil
	}
	seen := map[string]bool{}
	names := make([]string, 0, len(methods))
	for _, method := range methods {
		key := methodNameKey(method.Name)
		if i.methodAccessible(method) && !seen[key] {
			names = append(names, method.Name)
			seen[key] = true
		}
	}
	sort.Strings(names)
	result := NewArray()
	for _, name := range names {
		result.Append(StringValue(name))
	}
	return ArrayValue(result), true, nil
}

func (i *Interpreter) callGetClassVars(args []ast.Expr) (Value, bool, error) {
	if len(args) != 1 {
		return i.builtinWarningf(NullValue(), true, "get_class_vars expects 1 argument, got %d", len(args))
	}
	target, err := i.eval(args[0])
	if err != nil {
		return NullValue(), true, err
	}
	if target.Kind != String {
		return BoolValue(false), true, nil
	}
	if err := i.autoloadClass(target.String()); err != nil {
		return NullValue(), true, err
	}
	class, ok := i.classByName(target.String())
	if !ok {
		return BoolValue(false), true, nil
	}
	props, meta, err := i.instanceDefaults(class)
	if err != nil {
		return NullValue(), true, err
	}
	for _, current := range i.classLineage(class) {
		for name, value := range i.statics[current.Name] {
			props[name] = value
			if info, ok := i.staticMeta[current.Name][name]; ok {
				meta[name] = info
			} else {
				meta[name] = PropertyMeta{Owner: current.Name, Name: name, Visibility: "public", Static: true}
			}
		}
	}
	names := make([]string, 0, len(props))
	for name, info := range meta {
		if i.propertyAccessible(info) {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	result := NewArray()
	for _, name := range names {
		result.Set(StringKey(propertyNameForKey(name, meta[name])), props[name])
	}
	return ArrayValue(result), true, nil
}

func (i *Interpreter) callGetDeclaredClasses(args []ast.Expr) (Value, bool, error) {
	if len(args) != 0 {
		return i.builtinWarningf(NullValue(), true, "get_declared_classes expects 0 arguments, got %d", len(args))
	}
	names := make([]string, 0, len(i.classes)+len(i.classAliases))
	for _, class := range i.classes {
		names = append(names, class.Name)
	}
	for _, alias := range i.classAliases {
		names = append(names, alias.Name)
	}
	sort.Strings(names)
	result := NewArray()
	for _, name := range names {
		result.Append(StringValue(name))
	}
	return ArrayValue(result), true, nil
}

func (i *Interpreter) callGetDeclaredInterfaces(args []ast.Expr) (Value, bool, error) {
	if len(args) != 0 {
		return i.builtinWarningf(NullValue(), true, "get_declared_interfaces expects 0 arguments, got %d", len(args))
	}
	names := make([]string, 0, len(i.interfaces))
	for _, iface := range i.interfaces {
		names = append(names, iface.Name)
	}
	sort.Strings(names)
	result := NewArray()
	for _, name := range names {
		result.Append(StringValue(name))
	}
	return ArrayValue(result), true, nil
}

func (i *Interpreter) callClassAlias(args []ast.Expr) (Value, bool, error) {
	if len(args) < 2 || len(args) > 3 {
		return i.builtinWarningf(NullValue(), true, "class_alias expects 2 or 3 arguments, got %d", len(args))
	}
	originalValue, err := i.eval(args[0])
	if err != nil {
		return NullValue(), true, err
	}
	aliasValue, err := i.eval(args[1])
	if err != nil {
		return NullValue(), true, err
	}
	autoload := true
	if len(args) == 3 {
		autoloadValue, err := i.eval(args[2])
		if err != nil {
			return NullValue(), true, err
		}
		autoload = autoloadValue.Truthy()
	}
	original := strings.TrimPrefix(originalValue.String(), "\\")
	alias := strings.TrimPrefix(aliasValue.String(), "\\")
	if original == "" || alias == "" {
		return BoolValue(false), true, nil
	}
	if autoload {
		if err := i.autoloadClass(original); err != nil {
			return NullValue(), true, err
		}
	}
	class, ok := i.classByName(original)
	if !ok {
		return BoolValue(false), true, nil
	}
	if _, exists := i.classLikeName(alias); exists {
		return BoolValue(false), true, nil
	}
	i.classAliases[classNameKey(alias)] = classAlias{Name: alias, Target: class.Name}
	return BoolValue(true), true, nil
}

func (i *Interpreter) callIsCallable(args []ast.Expr) (Value, bool, error) {
	if len(args) < 1 || len(args) > 3 {
		return i.builtinWarningf(NullValue(), true, "is_callable expects 1 to 3 arguments, got %d", len(args))
	}
	value, err := i.eval(args[0])
	if err != nil {
		return NullValue(), true, err
	}
	syntaxOnly := false
	if len(args) >= 2 {
		flag, err := i.eval(args[1])
		if err != nil {
			return NullValue(), true, err
		}
		syntaxOnly = flag.Truthy()
	}
	callableName := i.callableName(value)
	if len(args) == 3 {
		if err := i.assignBuiltinResult(args[2], StringValue(callableName)); err != nil {
			return NullValue(), true, err
		}
	}
	if syntaxOnly {
		return BoolValue(i.isCallableSyntax(value)), true, nil
	}
	return BoolValue(i.isCallableValue(value)), true, nil
}

func (i *Interpreter) isCallableSyntax(value Value) bool {
	switch value.Kind {
	case ClosureKind:
		return value.C != nil
	case ObjectKind:
		return value.O != nil
	case String:
		return value.String() != ""
	case ArrayKind:
		if value.A == nil {
			return false
		}
		target, okTarget := value.A.Get(IntKey(0))
		method, okMethod := value.A.Get(IntKey(1))
		if !okTarget || !okMethod || method.String() == "" {
			return false
		}
		return target.Kind == ObjectKind || target.Kind == String
	default:
		return false
	}
}

func (i *Interpreter) isCallableValue(value Value) bool {
	switch value.Kind {
	case ClosureKind:
		return value.C != nil
	case ObjectKind:
		if value.O == nil {
			return false
		}
		method, ok := i.objectMethod(value.O, "__invoke")
		return ok && i.methodAccessible(method)
	case String:
		name := value.String()
		if className, methodName, ok := strings.Cut(name, "::"); ok {
			return i.isCallableStaticMethod(className, methodName)
		}
		_, ok := i.functions[functionNameKey(name)]
		return ok
	case ArrayKind:
		if value.A == nil {
			return false
		}
		target, okTarget := value.A.Get(IntKey(0))
		method, okMethod := value.A.Get(IntKey(1))
		if !okTarget || !okMethod {
			return false
		}
		methodName := method.String()
		if methodName == "" || strings.EqualFold(methodName, "__clone") {
			return false
		}
		switch target.Kind {
		case ObjectKind:
			if target.O == nil {
				return false
			}
			if method, ok := i.objectMethod(target.O, methodName); ok && i.methodAccessible(method) {
				return true
			}
			return i.hasPublicMethod(target.O, "__call")
		case String:
			return i.isCallableStaticMethod(target.String(), methodName)
		default:
			return false
		}
	default:
		return false
	}
}

func (i *Interpreter) isCallableStaticMethod(className, methodName string) bool {
	if className == "" || methodName == "" || strings.EqualFold(methodName, "__clone") {
		return false
	}
	resolved, err := i.resolveClassNameAutoload(className)
	if err != nil {
		return false
	}
	class, ok := i.classByName(resolved)
	if !ok {
		return false
	}
	methods := i.methodsForClass(class)
	if method, ok := i.staticMethod(methods, resolved, methodName); ok {
		return method.Static && i.methodAccessible(method)
	}
	return i.hasPublicStaticMethod(methods, "__callStatic")
}

func (i *Interpreter) callableName(value Value) string {
	switch value.Kind {
	case ClosureKind:
		return "Closure::__invoke"
	case ObjectKind:
		if value.O == nil {
			return ""
		}
		return value.O.ClassName + "::__invoke"
	case String:
		return value.String()
	case ArrayKind:
		if value.A == nil {
			return ""
		}
		target, okTarget := value.A.Get(IntKey(0))
		method, okMethod := value.A.Get(IntKey(1))
		if !okTarget || !okMethod {
			return ""
		}
		switch target.Kind {
		case ObjectKind:
			if target.O == nil {
				return "::" + method.String()
			}
			return target.O.ClassName + "::" + method.String()
		case String:
			return target.String() + "::" + method.String()
		default:
			return method.String()
		}
	default:
		return ""
	}
}

func (i *Interpreter) callUserFunc(args []ast.Expr) (Value, bool, error) {
	if len(args) < 1 {
		return i.handleBuiltinWarning(fmt.Errorf("call_user_func expects at least 1 argument, got %d", len(args)), NullValue(), true)
	}
	callable, err := i.eval(args[0])
	if err != nil {
		return NullValue(), true, err
	}
	value, err := i.callCallableValue(callable, args[1:])
	if isCallableWarning(err) {
		return i.handleBuiltinWarning(err, value, true)
	}
	return value, true, err
}

func (i *Interpreter) callUserFuncArray(args []ast.Expr) (Value, bool, error) {
	if len(args) != 2 {
		return i.handleBuiltinWarning(fmt.Errorf("call_user_func_array expects 2 arguments, got %d", len(args)), NullValue(), true)
	}
	callable, err := i.eval(args[0])
	if err != nil {
		return NullValue(), true, err
	}
	argArray, err := i.eval(args[1])
	if err != nil {
		return NullValue(), true, err
	}
	if argArray.Kind != ArrayKind || argArray.A == nil {
		return i.handleBuiltinWarning(fmt.Errorf("call_user_func_array expects argument 2 to be array"), NullValue(), true)
	}
	values := make([]Value, 0, len(argArray.A.Entries))
	for _, entry := range argArray.A.Entries {
		values = append(values, entry.RuntimeValue())
	}
	value, err := i.callCallableValueValues(callable, values)
	if isCallableWarning(err) {
		return i.handleBuiltinWarning(err, value, true)
	}
	return value, true, err
}

func (i *Interpreter) callCallableValue(callable Value, args []ast.Expr) (Value, error) {
	switch callable.Kind {
	case ClosureKind:
		return i.callClosure(callable.C, args)
	case ObjectKind:
		return i.callInvokableObject(callable.O, args)
	case String:
		name := callable.String()
		if className, methodName, ok := strings.Cut(name, "::"); ok {
			return i.callStaticName(className, methodName, args)
		}
		return i.call(&ast.CallExpr{Name: name, Args: args})
	case ArrayKind:
		if callable.A == nil {
			return NullValue(), fmt.Errorf("invalid callable")
		}
		target, okTarget := callable.A.Get(IntKey(0))
		method, okMethod := callable.A.Get(IntKey(1))
		if !okTarget || !okMethod || method.String() == "" {
			return NullValue(), fmt.Errorf("invalid callable")
		}
		methodName := method.String()
		switch target.Kind {
		case ObjectKind:
			return i.callObjectMethodValue(target.O, methodName, args)
		case String:
			return i.callStaticName(target.String(), methodName, args)
		default:
			return NullValue(), fmt.Errorf("invalid callable")
		}
	default:
		return NullValue(), fmt.Errorf("invalid callable")
	}
}

func (i *Interpreter) callCallableValueValues(callable Value, args []Value) (Value, error) {
	switch callable.Kind {
	case ClosureKind:
		return i.callClosureValues(callable.C, args)
	case ObjectKind:
		return i.callInvokableObjectValues(callable.O, args)
	case String:
		name := callable.String()
		if className, methodName, ok := strings.Cut(name, "::"); ok {
			return i.callStaticNameValues(className, methodName, args)
		}
		return i.callNamedFunctionValues(name, args)
	case ArrayKind:
		if callable.A == nil {
			return NullValue(), fmt.Errorf("invalid callable")
		}
		target, okTarget := callable.A.Get(IntKey(0))
		method, okMethod := callable.A.Get(IntKey(1))
		if !okTarget || !okMethod || method.String() == "" {
			return NullValue(), fmt.Errorf("invalid callable")
		}
		methodName := method.String()
		switch target.Kind {
		case ObjectKind:
			return i.callObjectMethodValueValues(target.O, methodName, args)
		case String:
			return i.callStaticNameValues(target.String(), methodName, args)
		default:
			return NullValue(), fmt.Errorf("invalid callable")
		}
	default:
		return NullValue(), fmt.Errorf("invalid callable")
	}
}

func (i *Interpreter) callNamedFunctionValues(name string, args []Value) (Value, error) {
	fn, ok := i.functions[functionNameKey(name)]
	if ok {
		return i.callFunctionValues(fn, args, nil, "")
	}
	if base := namespaceBaseName(name); base != name {
		return i.callNamedFunctionValues(base, args)
	}
	runtimeArgs := append([]Value(nil), args...)
	if value, ok, err := i.callSAPIBuiltin(name, runtimeArgs); ok || err != nil {
		if err != nil {
			value, _, err = i.handleBuiltinWarning(err, value, ok)
		}
		return value, err
	}
	builtinArgs := make([]builtins.Value, 0, len(args))
	for _, arg := range args {
		builtinArgs = append(builtinArgs, toBuiltinValue(arg))
	}
	if value, output, ok, err := builtins.CallWithOutput(name, builtinArgs); ok || err != nil {
		if output != "" {
			i.Out.WriteString(output)
		}
		if err != nil {
			runtimeValue, _, warnErr := i.handleBuiltinWarning(err, fromBuiltinValue(value), ok)
			return runtimeValue, warnErr
		}
		return fromBuiltinValue(value), nil
	}
	if value, ok, err := builtins.Call(name, builtinArgs); ok || err != nil {
		if err != nil {
			runtimeValue, _, warnErr := i.handleBuiltinWarning(err, fromBuiltinValue(value), ok)
			return runtimeValue, warnErr
		}
		return fromBuiltinValue(value), nil
	}
	return NullValue(), fmt.Errorf("undefined function %s", name)
}

func (i *Interpreter) callMethodExists(args []ast.Expr) (Value, bool, error) {
	if len(args) != 2 {
		return i.builtinWarningf(NullValue(), true, "method_exists expects 2 arguments, got %d", len(args))
	}
	target, err := i.eval(args[0])
	if err != nil {
		return NullValue(), true, err
	}
	methodValue, err := i.eval(args[1])
	if err != nil {
		return NullValue(), true, err
	}
	if target.Kind == String {
		if err := i.autoloadClass(target.String()); err != nil {
			return NullValue(), true, err
		}
	}
	methods, ok := i.methodsForTarget(target)
	if !ok {
		return BoolValue(false), true, nil
	}
	return BoolValue(methodMapHasName(methods, methodValue.String())), true, nil
}

func (i *Interpreter) callPropertyExists(args []ast.Expr) (Value, bool, error) {
	if len(args) != 2 {
		return i.builtinWarningf(NullValue(), true, "property_exists expects 2 arguments, got %d", len(args))
	}
	target, err := i.eval(args[0])
	if err != nil {
		return NullValue(), true, err
	}
	propertyValue, err := i.eval(args[1])
	if err != nil {
		return NullValue(), true, err
	}
	property := propertyValue.String()
	if target.Kind == ObjectKind && target.O != nil {
		if _, _, _, ok, inaccessible := i.objectProperty(target.O, property); ok || inaccessible {
			return BoolValue(true), true, nil
		}
		class, ok := i.classByName(target.O.ClassName)
		if !ok {
			return BoolValue(false), true, nil
		}
		return BoolValue(i.classDeclaresProperty(class, property)), true, nil
	}
	if target.Kind != String {
		return BoolValue(false), true, nil
	}
	if err := i.autoloadClass(target.String()); err != nil {
		return NullValue(), true, err
	}
	class, ok := i.classByName(target.String())
	if !ok {
		return BoolValue(false), true, nil
	}
	return BoolValue(i.classDeclaresProperty(class, property)), true, nil
}

func (i *Interpreter) callGetParentClass(args []ast.Expr) (Value, bool, error) {
	if len(args) > 1 {
		return i.builtinWarningf(NullValue(), true, "get_parent_class expects 0 or 1 arguments, got %d", len(args))
	}
	var className string
	if len(args) == 0 {
		className = i.currentClass()
		if className == "" {
			return BoolValue(false), true, nil
		}
	} else {
		value, err := i.eval(args[0])
		if err != nil {
			return NullValue(), true, err
		}
		switch value.Kind {
		case ObjectKind:
			if value.O == nil {
				return BoolValue(false), true, nil
			}
			className = value.O.ClassName
		case String:
			className = value.String()
			if err := i.autoloadClass(className); err != nil {
				return NullValue(), true, err
			}
		default:
			return BoolValue(false), true, nil
		}
	}
	class, ok := i.classByName(className)
	if !ok || class.Parent == "" {
		return BoolValue(false), true, nil
	}
	return StringValue(class.Parent), true, nil
}

func (i *Interpreter) callIsSubclassOf(args []ast.Expr) (Value, bool, error) {
	if len(args) < 2 || len(args) > 3 {
		return i.builtinWarningf(NullValue(), true, "is_subclass_of expects 2 or 3 arguments, got %d", len(args))
	}
	target, err := i.eval(args[0])
	if err != nil {
		return NullValue(), true, err
	}
	classValue, err := i.eval(args[1])
	if err != nil {
		return NullValue(), true, err
	}
	allowString := true
	if len(args) == 3 {
		allowValue, err := i.eval(args[2])
		if err != nil {
			return NullValue(), true, err
		}
		allowString = allowValue.Truthy()
	}
	className, ok, err := i.classNameFromValue(target, allowString)
	if err != nil {
		return NullValue(), true, err
	}
	if !ok {
		return BoolValue(false), true, nil
	}
	matches, err := i.classLikeMatches(className, classValue.String(), false)
	if err != nil {
		return NullValue(), true, err
	}
	return BoolValue(matches), true, nil
}

func (i *Interpreter) callIsA(args []ast.Expr) (Value, bool, error) {
	if len(args) < 2 || len(args) > 3 {
		return i.builtinWarningf(NullValue(), true, "is_a expects 2 or 3 arguments, got %d", len(args))
	}
	target, err := i.eval(args[0])
	if err != nil {
		return NullValue(), true, err
	}
	classValue, err := i.eval(args[1])
	if err != nil {
		return NullValue(), true, err
	}
	allowString := false
	if len(args) == 3 {
		allowValue, err := i.eval(args[2])
		if err != nil {
			return NullValue(), true, err
		}
		allowString = allowValue.Truthy()
	}
	className, ok, err := i.classNameFromValue(target, allowString)
	if err != nil {
		return NullValue(), true, err
	}
	if !ok {
		return BoolValue(false), true, nil
	}
	matches, err := i.classLikeMatches(className, classValue.String(), true)
	if err != nil {
		return NullValue(), true, err
	}
	return BoolValue(matches), true, nil
}

func (i *Interpreter) callClassParents(args []ast.Expr) (Value, bool, error) {
	if len(args) < 1 || len(args) > 2 {
		return i.builtinWarningf(NullValue(), true, "class_parents expects 1 or 2 arguments, got %d", len(args))
	}
	target, err := i.eval(args[0])
	if err != nil {
		return NullValue(), true, err
	}
	autoload := true
	if len(args) == 2 {
		autoloadValue, err := i.eval(args[1])
		if err != nil {
			return NullValue(), true, err
		}
		autoload = autoloadValue.Truthy()
	}
	className, ok, err := i.classNameFromIntrospectionValue(target, autoload)
	if err != nil {
		return NullValue(), true, err
	}
	if !ok {
		return BoolValue(false), true, nil
	}
	class, ok := i.classByName(className)
	if !ok {
		return BoolValue(false), true, nil
	}
	parents := NewArray()
	for class.Parent != "" {
		parent, ok := i.classByName(class.Parent)
		if !ok {
			break
		}
		parents.Set(StringKey(parent.Name), StringValue(parent.Name))
		class = parent
	}
	return ArrayValue(parents), true, nil
}

func (i *Interpreter) callClassImplements(args []ast.Expr) (Value, bool, error) {
	if len(args) < 1 || len(args) > 2 {
		return i.builtinWarningf(NullValue(), true, "class_implements expects 1 or 2 arguments, got %d", len(args))
	}
	target, err := i.eval(args[0])
	if err != nil {
		return NullValue(), true, err
	}
	autoload := true
	if len(args) == 2 {
		autoloadValue, err := i.eval(args[1])
		if err != nil {
			return NullValue(), true, err
		}
		autoload = autoloadValue.Truthy()
	}
	name, ok, err := i.classNameFromIntrospectionValue(target, autoload)
	if err != nil {
		return NullValue(), true, err
	}
	if !ok {
		return BoolValue(false), true, nil
	}
	interfaces := NewArray()
	seen := map[string]bool{}
	addInterface := func(interfaceName string) {}
	addInterface = func(interfaceName string) {
		iface, ok := i.interfaceByName(interfaceName)
		if !ok {
			return
		}
		key := classNameKey(iface.Name)
		if seen[key] {
			return
		}
		seen[key] = true
		interfaces.Set(StringKey(iface.Name), StringValue(iface.Name))
		for _, parentName := range iface.Parents {
			addInterface(parentName)
		}
	}
	if iface, ok := i.interfaceByName(name); ok {
		for _, parentName := range iface.Parents {
			addInterface(parentName)
		}
		return ArrayValue(interfaces), true, nil
	}
	class, ok := i.classByName(name)
	if !ok {
		return BoolValue(false), true, nil
	}
	for class != nil {
		for _, interfaceName := range class.Implements {
			addInterface(interfaceName)
		}
		if class.Parent == "" {
			break
		}
		parent, ok := i.classByName(class.Parent)
		if !ok {
			break
		}
		class = parent
	}
	return ArrayValue(interfaces), true, nil
}

func (i *Interpreter) classNameFromIntrospectionValue(value Value, autoload bool) (string, bool, error) {
	if value.Kind == ObjectKind && value.O != nil {
		return value.O.ClassName, true, nil
	}
	if value.Kind != String {
		return "", false, nil
	}
	name := value.String()
	if autoload {
		if err := i.autoloadClass(name); err != nil {
			return "", false, err
		}
	}
	if canonical, ok := i.classLikeName(name); ok {
		return canonical, true, nil
	}
	return name, true, nil
}

func (i *Interpreter) classNameFromValue(value Value, allowString bool) (string, bool, error) {
	if value.Kind == ObjectKind && value.O != nil {
		return value.O.ClassName, true, nil
	}
	if value.Kind != String || !allowString {
		return "", false, nil
	}
	name := value.String()
	if err := i.autoloadClass(name); err != nil {
		return "", false, err
	}
	if canonical, ok := i.classLikeName(name); ok {
		return canonical, true, nil
	}
	return name, true, nil
}

func (i *Interpreter) classLikeMatches(className, targetName string, allowSame bool) (bool, error) {
	if err := i.autoloadClass(targetName); err != nil {
		return false, err
	}
	if allowSame && strings.EqualFold(className, targetName) {
		return true, nil
	}
	if _, ok := i.classByName(targetName); ok {
		return i.isStrictSubclassOf(className, targetName), nil
	}
	if _, ok := i.interfaceByName(targetName); ok {
		if iface, classIsInterface := i.interfaceByName(className); classIsInterface {
			if !allowSame && strings.EqualFold(iface.Name, targetName) {
				return false, nil
			}
			return i.interfaceExtendsInterface(iface.Name, targetName), nil
		}
		return i.classImplementsInterface(className, targetName), nil
	}
	return false, nil
}

func (i *Interpreter) isStrictSubclassOf(className, ancestor string) bool {
	if strings.EqualFold(className, ancestor) {
		return false
	}
	return i.isSameOrSubclass(className, ancestor)
}

func (i *Interpreter) methodsForTarget(target Value) (map[string]*ast.FunctionDeclStmt, bool) {
	if target.Kind == ObjectKind && target.O != nil {
		return target.O.Methods, true
	}
	if target.Kind != String {
		return nil, false
	}
	if class, ok := i.classByName(target.String()); ok {
		return i.methodsForClass(class), true
	}
	if iface, ok := i.interfaceByName(target.String()); ok {
		methods, err := i.interfaceMethods(iface.Name)
		return methods, err == nil
	}
	return nil, false
}

func (i *Interpreter) classLineage(class *ast.ClassDeclStmt) []*ast.ClassDeclStmt {
	if class == nil {
		return nil
	}
	lineage := []*ast.ClassDeclStmt{}
	if class.Parent != "" {
		if parent, ok := i.classByName(class.Parent); ok {
			lineage = append(lineage, i.classLineage(parent)...)
		}
	}
	return append(lineage, class)
}

func (i *Interpreter) classDeclaresProperty(class *ast.ClassDeclStmt, property string) bool {
	for class != nil {
		for _, prop := range class.Properties {
			if prop.Name == property {
				return true
			}
		}
		if class.Parent == "" {
			return false
		}
		parent, ok := i.classByName(class.Parent)
		if !ok {
			return false
		}
		class = parent
	}
	return false
}

func (i *Interpreter) callDefined(args []ast.Expr) (Value, bool, error) {
	if len(args) != 1 {
		return i.builtinWarningf(NullValue(), true, "defined expects 1 argument, got %d", len(args))
	}
	name, err := i.eval(args[0])
	if err != nil {
		return NullValue(), true, err
	}
	_, ok := i.lookupNamedConstant(name.String())
	return BoolValue(ok), true, nil
}

func (i *Interpreter) callConstant(args []ast.Expr) (Value, bool, error) {
	if len(args) != 1 {
		return i.builtinWarningf(NullValue(), true, "constant expects 1 argument, got %d", len(args))
	}
	name, err := i.eval(args[0])
	if err != nil {
		return NullValue(), true, err
	}
	value, ok := i.lookupNamedConstant(name.String())
	if !ok {
		return i.builtinWarningf(NullValue(), true, "undefined constant %s", name.String())
	}
	return value, true, nil
}

func (i *Interpreter) callSerialize(args []ast.Expr) (Value, bool, error) {
	if len(args) != 1 {
		return i.builtinWarningf(NullValue(), true, "serialize expects 1 arguments, got %d", len(args))
	}
	value, err := i.eval(args[0])
	if err != nil {
		return NullValue(), true, err
	}
	serialized, err := i.serializeValue(value)
	if err != nil {
		return NullValue(), true, err
	}
	return StringValue(serialized), true, nil
}

func (i *Interpreter) callUnserialize(args []ast.Expr) (Value, bool, error) {
	if len(args) != 1 {
		return i.builtinWarningf(NullValue(), true, "unserialize expects 1 arguments, got %d", len(args))
	}
	raw, err := i.eval(args[0])
	if err != nil {
		return NullValue(), true, err
	}
	parser := runtimeSerializedParser{input: raw.String(), interp: i}
	value, err := parser.parse()
	if err != nil {
		if !isSerializedFormatWarning(err) {
			return NullValue(), true, err
		}
		return i.handleBuiltinWarning(err, BoolValue(false), true)
	}
	return value, true, nil
}

func (i *Interpreter) serializeValue(value Value) (string, error) {
	switch value.Kind {
	case Null:
		return "N;", nil
	case Bool:
		if value.B {
			return "b:1;", nil
		}
		return "b:0;", nil
	case Int:
		return "i:" + strconv.FormatInt(value.I, 10) + ";", nil
	case Float:
		return "d:" + strconv.FormatFloat(value.F, 'f', -1, 64) + ";", nil
	case String:
		return `s:` + strconv.Itoa(len(value.S)) + `:"` + value.S + `";`, nil
	case ArrayKind:
		return i.serializeArray(value.A)
	case ObjectKind:
		return i.serializeObject(value.O)
	default:
		return "N;", nil
	}
}

func (i *Interpreter) serializeArray(arr *Array) (string, error) {
	if arr == nil {
		return "a:0:{}", nil
	}
	var b strings.Builder
	b.WriteString("a:")
	b.WriteString(strconv.Itoa(len(arr.Entries)))
	b.WriteString(":{")
	for _, entry := range arr.Entries {
		if entry.Key.IsInt {
			b.WriteString("i:")
			b.WriteString(strconv.FormatInt(entry.Key.Int, 10))
			b.WriteString(";")
		} else {
			key, err := i.serializeValue(StringValue(entry.Key.Str))
			if err != nil {
				return "", err
			}
			b.WriteString(key)
		}
		value, err := i.serializeValue(entry.RuntimeValue())
		if err != nil {
			return "", err
		}
		b.WriteString(value)
	}
	b.WriteString("}")
	return b.String(), nil
}

func (i *Interpreter) serializeObject(obj *Object) (string, error) {
	if obj == nil {
		return "N;", nil
	}
	props, err := i.serializableObjectProperties(obj)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString("O:")
	b.WriteString(strconv.Itoa(len(obj.ClassName)))
	b.WriteString(`:"`)
	b.WriteString(obj.ClassName)
	b.WriteString(`":`)
	b.WriteString(strconv.Itoa(len(props)))
	b.WriteString(":{")
	for _, prop := range props {
		key, err := i.serializeValue(StringValue(prop.name))
		if err != nil {
			return "", err
		}
		value, err := i.serializeValue(prop.value)
		if err != nil {
			return "", err
		}
		b.WriteString(key)
		b.WriteString(value)
	}
	b.WriteString("}")
	return b.String(), nil
}

type serializedObjectProperty struct {
	name  string
	value Value
}

func (i *Interpreter) serializableObjectProperties(obj *Object) ([]serializedObjectProperty, error) {
	if i.hasPublicMethod(obj, "__sleep") {
		method := obj.Methods[methodNameKey("__sleep")]
		names, err := i.callFunctionValues(method, nil, obj, obj.ClassName)
		if err != nil {
			return nil, err
		}
		if names.Kind != ArrayKind || names.A == nil {
			return nil, fmt.Errorf("__sleep must return array")
		}
		props := make([]serializedObjectProperty, 0, len(names.A.Entries))
		for _, entry := range names.A.Entries {
			name := entry.RuntimeValue().String()
			_, value, info, ok, _ := i.objectPropertyInScope(obj, name, method.Owner)
			if !ok {
				value = NullValue()
			}
			props = append(props, serializedObjectProperty{name: serializedObjectPropertyName(name, info), value: value})
		}
		return props, nil
	}
	props := make([]serializedObjectProperty, 0, len(obj.Props))
	for name, value := range obj.Props {
		props = append(props, serializedObjectProperty{name: serializedObjectPropertyName(name, obj.PropMeta[name]), value: value})
	}
	return props, nil
}

func serializedObjectPropertyName(key string, info PropertyMeta) string {
	name := propertyNameForKey(key, info)
	switch info.Visibility {
	case "private":
		if info.Owner != "" {
			return "\x00" + info.Owner + "\x00" + name
		}
	case "protected":
		return "\x00*\x00" + name
	}
	return name
}

type runtimeSerializedParser struct {
	input  string
	pos    int
	interp *Interpreter
}

func (p *runtimeSerializedParser) parse() (Value, error) {
	if p.pos >= len(p.input) {
		return NullValue(), fmt.Errorf("unexpected end of serialized value")
	}
	switch p.input[p.pos] {
	case 'N':
		p.pos += 2
		return NullValue(), nil
	case 'b':
		n, err := p.readNumberAfterPrefix()
		if err != nil {
			return NullValue(), err
		}
		return BoolValue(n != 0), nil
	case 'i':
		n, err := p.readNumberAfterPrefix()
		if err != nil {
			return NullValue(), err
		}
		return IntValue(n), nil
	case 'd':
		raw, err := p.readRawAfterPrefix()
		if err != nil {
			return NullValue(), err
		}
		f, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return NullValue(), err
		}
		return FloatValue(f), nil
	case 's':
		return p.parseString()
	case 'a':
		return p.parseArray()
	case 'O':
		return p.parseObject()
	default:
		return NullValue(), fmt.Errorf("unsupported serialized type %q", p.input[p.pos])
	}
}

func (p *runtimeSerializedParser) parseString() (Value, error) {
	p.pos += 2
	lengthText, err := p.readUntil(':')
	if err != nil {
		return NullValue(), err
	}
	length, err := strconv.Atoi(lengthText)
	if err != nil {
		return NullValue(), err
	}
	if err := p.expectByte('"'); err != nil {
		return NullValue(), err
	}
	if p.pos+length > len(p.input) {
		return NullValue(), fmt.Errorf("serialized string length out of range")
	}
	s := p.input[p.pos : p.pos+length]
	p.pos += length
	if err := p.expectByte('"'); err != nil {
		return NullValue(), err
	}
	if err := p.expectByte(';'); err != nil {
		return NullValue(), err
	}
	return StringValue(s), nil
}

func (p *runtimeSerializedParser) parseArray() (Value, error) {
	p.pos += 2
	countText, err := p.readUntil(':')
	if err != nil {
		return NullValue(), err
	}
	count, err := strconv.Atoi(countText)
	if err != nil {
		return NullValue(), err
	}
	if err := p.expectByte('{'); err != nil {
		return NullValue(), err
	}
	arr := NewArray()
	for idx := 0; idx < count; idx++ {
		key, err := p.parse()
		if err != nil {
			return NullValue(), err
		}
		value, err := p.parse()
		if err != nil {
			return NullValue(), err
		}
		arr.Set(KeyFromValue(key), value)
	}
	if err := p.expectByte('}'); err != nil {
		return NullValue(), err
	}
	return ArrayValue(arr), nil
}

func (p *runtimeSerializedParser) parseObject() (Value, error) {
	p.pos += 2
	lengthText, err := p.readUntil(':')
	if err != nil {
		return NullValue(), err
	}
	length, err := strconv.Atoi(lengthText)
	if err != nil {
		return NullValue(), err
	}
	if err := p.expectByte('"'); err != nil {
		return NullValue(), err
	}
	if p.pos+length > len(p.input) {
		return NullValue(), fmt.Errorf("serialized class name length out of range")
	}
	className := p.input[p.pos : p.pos+length]
	p.pos += length
	if err := p.expectByte('"'); err != nil {
		return NullValue(), err
	}
	if err := p.expectByte(':'); err != nil {
		return NullValue(), err
	}
	countText, err := p.readUntil(':')
	if err != nil {
		return NullValue(), err
	}
	count, err := strconv.Atoi(countText)
	if err != nil {
		return NullValue(), err
	}
	if err := p.expectByte('{'); err != nil {
		return NullValue(), err
	}
	if err := p.interp.autoloadClass(className); err != nil {
		return NullValue(), err
	}
	if class, ok := p.interp.classByName(className); ok {
		className = class.Name
	}
	props := map[string]Value{}
	meta := map[string]PropertyMeta{}
	for idx := 0; idx < count; idx++ {
		key, err := p.parse()
		if err != nil {
			return NullValue(), err
		}
		value, err := p.parse()
		if err != nil {
			return NullValue(), err
		}
		name, info := unserializedObjectProperty(className, key.String())
		props[name] = value
		meta[name] = info
	}
	if err := p.expectByte('}'); err != nil {
		return NullValue(), err
	}
	methods := map[string]*ast.FunctionDeclStmt{}
	if class, ok := p.interp.classByName(className); ok {
		methods = p.interp.methodsForClass(class)
	}
	obj := NewObject(className, methods, props, meta)
	if p.interp.hasPublicMethod(obj, "__wakeup") {
		if _, err := p.interp.callFunctionValues(obj.Methods[methodNameKey("__wakeup")], nil, obj, obj.ClassName); err != nil {
			return NullValue(), err
		}
	}
	p.interp.trackObject(obj)
	return ObjectValue(obj), nil
}

func unserializedObjectProperty(className, raw string) (string, PropertyMeta) {
	if strings.HasPrefix(raw, "\x00") {
		rest := strings.TrimPrefix(raw, "\x00")
		owner, name, ok := strings.Cut(rest, "\x00")
		if ok {
			if owner == "*" {
				return name, PropertyMeta{Owner: className, Name: name, Visibility: "protected"}
			}
			return propertyStorageKey(owner, "private", name), PropertyMeta{Owner: owner, Name: name, Visibility: "private"}
		}
	}
	return raw, PropertyMeta{Owner: className, Name: raw, Visibility: "public"}
}

func (p *runtimeSerializedParser) readNumberAfterPrefix() (int64, error) {
	raw, err := p.readRawAfterPrefix()
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(raw, 10, 64)
}

func (p *runtimeSerializedParser) readRawAfterPrefix() (string, error) {
	p.pos += 2
	return p.readUntil(';')
}

func (p *runtimeSerializedParser) readUntil(delim byte) (string, error) {
	start := p.pos
	for p.pos < len(p.input) && p.input[p.pos] != delim {
		p.pos++
	}
	if p.pos >= len(p.input) {
		return "", fmt.Errorf("delimiter %q not found", delim)
	}
	out := p.input[start:p.pos]
	p.pos++
	return out, nil
}

func (p *runtimeSerializedParser) expectByte(want byte) error {
	if p.pos >= len(p.input) || p.input[p.pos] != want {
		return fmt.Errorf("expected %q", want)
	}
	p.pos++
	return nil
}

func (i *Interpreter) callIsset(args []ast.Expr) (Value, bool, error) {
	if len(args) == 0 {
		return i.builtinWarningf(NullValue(), true, "isset expects at least 1 argument, got 0")
	}
	for _, arg := range args {
		value, exists, err := i.lookupConstructValue(arg)
		if err != nil {
			return NullValue(), true, err
		}
		if !exists || value.Kind == Null {
			return BoolValue(false), true, nil
		}
	}
	return BoolValue(true), true, nil
}

func (i *Interpreter) callEmpty(args []ast.Expr) (Value, bool, error) {
	if len(args) != 1 {
		return i.builtinWarningf(NullValue(), true, "empty expects 1 argument, got %d", len(args))
	}
	if value, handled, err := i.lookupEmptyValue(args[0]); handled {
		if err != nil {
			return NullValue(), true, err
		}
		return BoolValue(!value.Truthy()), true, nil
	}
	value, exists, err := i.lookupConstructValue(args[0])
	if err != nil {
		return NullValue(), true, err
	}
	if !exists {
		return BoolValue(true), true, nil
	}
	return BoolValue(!value.Truthy()), true, nil
}

func (i *Interpreter) lookupEmptyValue(expr ast.Expr) (Value, bool, error) {
	switch n := expr.(type) {
	case *ast.PropertyFetchExpr:
		return i.lookupEmptyObjectProperty(n.Object, n.Property)
	case *ast.DynamicPropertyFetchExpr:
		property, err := i.dynamicMemberName(n.Property)
		if err != nil {
			return NullValue(), true, err
		}
		return i.lookupEmptyObjectProperty(n.Object, property)
	default:
		return NullValue(), false, nil
	}
}

func (i *Interpreter) lookupEmptyObjectProperty(object ast.Expr, property string) (Value, bool, error) {
	obj, err := i.eval(object)
	if err != nil || obj.Kind != ObjectKind || obj.O == nil {
		return NullValue(), true, err
	}
	if value, ok, inaccessible := i.visibleObjectProperty(obj.O, property); ok {
		return value, true, nil
	} else if inaccessible && !i.shouldUseIsset(obj.O) {
		return NullValue(), true, nil
	}
	if !i.shouldUseIsset(obj.O) {
		return NullValue(), true, nil
	}
	isset, err := i.callMagicIsset(obj.O, property)
	if err != nil || !isset.Truthy() {
		return NullValue(), true, err
	}
	if i.hasPublicMethod(obj.O, "__get") {
		value, err := i.callMagicGet(obj.O, property)
		return value, true, err
	}
	return BoolValue(true), true, nil
}

func (i *Interpreter) lookupConstructValue(expr ast.Expr) (Value, bool, error) {
	switch n := expr.(type) {
	case *ast.Ident:
		value, ok := i.getVar(n.Name)
		return value, ok, nil
	case *ast.VariableVariable:
		value, ok := i.getVar(i.variableVariableName(n.Name))
		return value, ok, nil
	case *ast.IndexExpr:
		return i.lookupIndexValue(n)
	case *ast.PropertyFetchExpr:
		return i.lookupPropertyValue(n)
	case *ast.DynamicPropertyFetchExpr:
		return i.lookupDynamicPropertyValue(n)
	case *ast.StaticPropertyFetchExpr:
		return i.lookupStaticPropertyValue(n)
	case *ast.DynamicStaticPropertyFetchExpr:
		return i.lookupDynamicStaticPropertyValue(n)
	default:
		value, err := i.eval(expr)
		return value, true, err
	}
}

func (i *Interpreter) lookupIndexValue(expr *ast.IndexExpr) (Value, bool, error) {
	if expr.Index == nil {
		return NullValue(), false, nil
	}
	left, exists, err := i.lookupConstructValue(expr.Left)
	if err != nil || !exists {
		return NullValue(), false, err
	}
	index, err := i.eval(expr.Index)
	if err != nil {
		return NullValue(), false, err
	}
	switch left.Kind {
	case ArrayKind:
		if left.A == nil {
			return NullValue(), false, nil
		}
		value, ok := left.A.Get(KeyFromValue(index))
		return value, ok, nil
	case String:
		offset, ok := indexOffset(index)
		if !ok || offset < 0 || offset >= len(left.S) {
			return NullValue(), false, nil
		}
		return StringValue(left.S[offset : offset+1]), true, nil
	default:
		return NullValue(), false, nil
	}
}

func (i *Interpreter) lookupPropertyValue(expr *ast.PropertyFetchExpr) (Value, bool, error) {
	return i.lookupObjectProperty(expr.Object, expr.Property)
}

func (i *Interpreter) lookupDynamicPropertyValue(expr *ast.DynamicPropertyFetchExpr) (Value, bool, error) {
	property, err := i.dynamicMemberName(expr.Property)
	if err != nil {
		return NullValue(), false, err
	}
	return i.lookupObjectProperty(expr.Object, property)
}

func (i *Interpreter) lookupObjectProperty(object ast.Expr, property string) (Value, bool, error) {
	obj, err := i.eval(object)
	if err != nil || obj.Kind != ObjectKind || obj.O == nil {
		return NullValue(), false, err
	}
	if value, ok, inaccessible := i.visibleObjectProperty(obj.O, property); ok {
		return value, true, nil
	} else if inaccessible && !i.shouldUseIsset(obj.O) {
		return NullValue(), false, nil
	}
	if !i.shouldUseIsset(obj.O) {
		return NullValue(), false, nil
	}
	value, err := i.callMagicIsset(obj.O, property)
	if err != nil || !value.Truthy() {
		return NullValue(), false, err
	}
	return BoolValue(true), true, nil
}

func (i *Interpreter) fetchProperty(object ast.Expr, property string) (Value, error) {
	obj, err := i.eval(object)
	if err != nil {
		return NullValue(), err
	}
	if obj.Kind != ObjectKind || obj.O == nil {
		if err := i.noticef("cannot fetch property %s on non-object", property); err != nil {
			return NullValue(), err
		}
		return NullValue(), nil
	}
	_, value, info, ok, inaccessible := i.objectProperty(obj.O, property)
	if inaccessible {
		if i.hasPublicMethod(obj.O, "__get") {
			return i.callMagicGet(obj.O, property)
		}
		return NullValue(), fmt.Errorf("cannot access %s property %s::$%s", info.Visibility, info.Owner, property)
	}
	if ok {
		return value, nil
	}
	if i.shouldUseGet(obj.O) {
		return i.callMagicGet(obj.O, property)
	}
	if err := i.noticef("undefined property %s::$%s", obj.O.ClassName, property); err != nil {
		return NullValue(), err
	}
	return NullValue(), nil
}

func (i *Interpreter) lookupStaticPropertyValue(expr *ast.StaticPropertyFetchExpr) (Value, bool, error) {
	return i.lookupStaticProperty(expr.ClassName, expr.Property)
}

func (i *Interpreter) lookupDynamicStaticPropertyValue(expr *ast.DynamicStaticPropertyFetchExpr) (Value, bool, error) {
	property, err := i.dynamicMemberName(expr.Property)
	if err != nil {
		return NullValue(), false, err
	}
	return i.lookupStaticProperty(expr.ClassName, property)
}

func (i *Interpreter) lookupStaticProperty(className, property string) (Value, bool, error) {
	resolved, err := i.resolveClassNameAutoload(className)
	if err != nil {
		return NullValue(), false, nil
	}
	value, info, ok := i.findStaticProperty(resolved, property)
	if !ok {
		return NullValue(), false, nil
	}
	switch info.Visibility {
	case "", "public":
		return value, true, nil
	case "private":
		if i.currentClass() == info.Owner {
			return value, true, nil
		}
	case "protected":
		if i.classesRelated(i.currentClass(), info.Owner) {
			return value, true, nil
		}
	}
	return NullValue(), false, nil
}

func (i *Interpreter) fetchStaticProperty(className, property string) (Value, error) {
	if err := i.checkStaticPropertyAccess(className, property); err != nil {
		return NullValue(), err
	}
	return i.getStaticProperty(className, property), nil
}

func (i *Interpreter) callPregMatchWithCaptures(c *ast.CallExpr) (Value, error) {
	if len(c.Args) < 3 || len(c.Args) > 3 {
		value, _, err := i.builtinWarningf(NullValue(), true, "preg_match expects 3 arguments, got %d", len(c.Args))
		return value, err
	}
	pattern, err := i.eval(c.Args[0])
	if err != nil {
		return NullValue(), err
	}
	subject, err := i.eval(c.Args[1])
	if err != nil {
		return NullValue(), err
	}
	re, err := regexp.Compile(runtimeRegexPattern(pattern.String()))
	if err != nil {
		value, _, warnErr := i.handleBuiltinWarning(err, BoolValue(false), true)
		return value, warnErr
	}
	arr := NewArray()
	matches := re.FindStringSubmatch(subject.String())
	if matches == nil {
		if err := i.assignBuiltinResult(c.Args[2], ArrayValue(arr)); err != nil {
			value, _, warnErr := i.handleBuiltinWarning(err, NullValue(), true)
			return value, warnErr
		}
		return IntValue(0), nil
	}
	for _, match := range matches {
		arr.Append(StringValue(match))
	}
	if err := i.assignBuiltinResult(c.Args[2], ArrayValue(arr)); err != nil {
		value, _, warnErr := i.handleBuiltinWarning(err, NullValue(), true)
		return value, warnErr
	}
	return IntValue(1), nil
}

func (i *Interpreter) assignBuiltinResult(target ast.Expr, value Value) error {
	switch n := target.(type) {
	case *ast.Ident:
		i.setVar(n.Name, value)
		return nil
	case *ast.VariableVariable:
		i.setVar(i.variableVariableName(n.Name), value)
		return nil
	case *ast.IndexExpr:
		return i.assignIndex(n, value)
	default:
		return fmt.Errorf("preg_match matches argument must be writable")
	}
}

func runtimeRegexPattern(pattern string) string {
	if len(pattern) < 2 {
		return pattern
	}
	delimiter := pattern[0]
	if strings.ContainsRune(`/#~!%`, rune(delimiter)) {
		for idx := len(pattern) - 1; idx > 0; idx-- {
			if pattern[idx] == delimiter {
				return pattern[1:idx]
			}
		}
	}
	return pattern
}

func (i *Interpreter) callSAPIBuiltin(name string, args []Value) (Value, bool, error) {
	switch strings.ToLower(name) {
	case "header":
		if len(args) < 1 || len(args) > 2 {
			return NullValue(), true, fmt.Errorf("header expects between 1 and 2 arguments, got %d", len(args))
		}
		replace := true
		if len(args) == 2 {
			replace = args[1].Truthy()
		}
		i.addResponseHeader(args[0].String(), replace)
		return NullValue(), true, nil
	case "headers_list":
		if len(args) != 0 {
			return NullValue(), true, fmt.Errorf("headers_list expects 0 arguments, got %d", len(args))
		}
		arr := NewArray()
		for _, header := range i.responseHeaders {
			arr.Append(StringValue(header))
		}
		return ArrayValue(arr), true, nil
	case "is_uploaded_file":
		if len(args) != 1 {
			return NullValue(), true, fmt.Errorf("is_uploaded_file expects 1 arguments, got %d", len(args))
		}
		path := args[0].String()
		if !i.uploadedFiles[path] {
			return BoolValue(false), true, nil
		}
		info, err := os.Stat(path)
		return BoolValue(err == nil && !info.IsDir()), true, nil
	case "move_uploaded_file":
		if len(args) != 2 {
			return NullValue(), true, fmt.Errorf("move_uploaded_file expects 2 arguments, got %d", len(args))
		}
		src := args[0].String()
		dst := args[1].String()
		if !i.uploadedFiles[src] {
			return BoolValue(false), true, nil
		}
		if err := os.Rename(src, dst); err != nil {
			data, readErr := os.ReadFile(src)
			if readErr != nil {
				return BoolValue(false), true, nil
			}
			if writeErr := os.WriteFile(dst, data, 0644); writeErr != nil {
				return BoolValue(false), true, nil
			}
			if removeErr := os.Remove(src); removeErr != nil {
				return BoolValue(false), true, nil
			}
		}
		delete(i.uploadedFiles, src)
		return BoolValue(true), true, nil
	default:
		return NullValue(), false, nil
	}
}

func (i *Interpreter) addResponseHeader(header string, replace bool) {
	header = strings.TrimSpace(header)
	if header == "" {
		return
	}
	if replace {
		name := responseHeaderName(header)
		if name != "" {
			for idx, existing := range i.responseHeaders {
				if strings.EqualFold(responseHeaderName(existing), name) {
					i.responseHeaders[idx] = header
					return
				}
			}
		}
	}
	i.responseHeaders = append(i.responseHeaders, header)
}

func responseHeaderName(header string) string {
	name, _, ok := strings.Cut(header, ":")
	if !ok {
		return ""
	}
	return strings.TrimSpace(name)
}

func toBuiltinValue(value Value) builtins.Value {
	switch value.Kind {
	case Null:
		return builtins.NullValue()
	case Bool:
		return builtins.BoolValue(value.B)
	case Int:
		return builtins.IntValue(value.I)
	case Float:
		return builtins.FloatValue(value.F)
	case String:
		return builtins.StringValue(value.S)
	case ArrayKind:
		return builtins.ArrayValue(toBuiltinArray(value.A))
	case ObjectKind:
		return builtins.ObjectValue()
	case ClosureKind:
		return builtins.ObjectValue()
	default:
		return builtins.NullValue()
	}
}

func toBuiltinArray(arr *Array) *builtins.Array {
	out := builtins.NewArray()
	if arr == nil {
		return out
	}
	for _, entry := range arr.Entries {
		out.Set(toBuiltinArrayKey(entry.Key), toBuiltinValue(entry.RuntimeValue()))
	}
	return out
}

func toBuiltinArrayKey(key ArrayKey) builtins.ArrayKey {
	if key.IsInt {
		return builtins.IntKey(key.Int)
	}
	return builtins.StringKey(key.Str)
}

func fromBuiltinValue(value builtins.Value) Value {
	switch value.Kind {
	case builtins.NullKind:
		return NullValue()
	case builtins.BoolKind:
		return BoolValue(value.B)
	case builtins.IntKind:
		return IntValue(value.I)
	case builtins.FloatKind:
		return FloatValue(value.F)
	case builtins.StringKind:
		return StringValue(value.S)
	case builtins.ArrayKind:
		return ArrayValue(fromBuiltinArray(value.A))
	default:
		return NullValue()
	}
}

func fromBuiltinArray(arr *builtins.Array) *Array {
	out := NewArray()
	if arr == nil {
		return out
	}
	for _, entry := range arr.Entries {
		out.Set(fromBuiltinArrayKey(entry.Key), fromBuiltinValue(entry.Value))
	}
	return out
}

func fromBuiltinArrayKey(key builtins.ArrayKey) ArrayKey {
	if key.IsInt {
		return IntKey(key.Int)
	}
	return StringKey(key.Str)
}

func (i *Interpreter) callFunction(fn *ast.FunctionDeclStmt, args []ast.Expr, thisObj *Object, calledClass string) (Value, error) {
	result, err := i.callFunctionResult(fn, args, thisObj, calledClass)
	return result.value, err
}

func (i *Interpreter) callFunctionResult(fn *ast.FunctionDeclStmt, args []ast.Expr, thisObj *Object, calledClass string) (functionCallResult, error) {
	return i.callFunctionWithBinderResult(fn, len(args), thisObj, calledClass, nil, func(idx int, param ast.Param) (*variableSlot, error) {
		if idx >= len(args) {
			if param.Default == nil {
				return i.missingArgumentSlot(fn, len(args))
			}
			value, err := i.eval(param.Default)
			if err != nil {
				return nil, err
			}
			return &variableSlot{value: value}, nil
		}
		if param.ByRef {
			slot, err := i.referenceSlot(args[idx])
			if err != nil {
				return nil, fmt.Errorf("argument %d for %s must be a variable passed by reference", idx+1, fn.Name)
			}
			return slot, nil
		}
		value, err := i.eval(args[idx])
		if err != nil {
			return nil, err
		}
		return &variableSlot{value: value}, nil
	})
}

func (i *Interpreter) callFunctionValues(fn *ast.FunctionDeclStmt, args []Value, thisObj *Object, calledClass string) (Value, error) {
	result, err := i.callFunctionWithBinderResult(fn, len(args), thisObj, calledClass, nil, func(idx int, param ast.Param) (*variableSlot, error) {
		if idx >= len(args) {
			if param.Default == nil {
				return i.missingArgumentSlot(fn, len(args))
			}
			value, err := i.eval(param.Default)
			if err != nil {
				return nil, err
			}
			return &variableSlot{value: value}, nil
		}
		if param.ByRef {
			return nil, fmt.Errorf("argument %d for %s must be a variable passed by reference", idx+1, fn.Name)
		}
		return &variableSlot{value: args[idx]}, nil
	})
	return result.value, err
}

func (i *Interpreter) callClosure(closure *Closure, args []ast.Expr) (Value, error) {
	if closure == nil || closure.Fn == nil {
		return NullValue(), fmt.Errorf("invalid closure")
	}
	result, err := i.callClosureResult(closure, args)
	return result.value, err
}

func (i *Interpreter) callClosureValues(closure *Closure, args []Value) (Value, error) {
	if closure == nil || closure.Fn == nil {
		return NullValue(), fmt.Errorf("invalid closure")
	}
	fn := closure.Fn
	result, err := i.callFunctionWithBinderResult(fn, len(args), nil, "", closure.Captures, func(idx int, param ast.Param) (*variableSlot, error) {
		if idx >= len(args) {
			if param.Default == nil {
				return i.missingArgumentSlot(fn, len(args))
			}
			value, err := i.eval(param.Default)
			if err != nil {
				return nil, err
			}
			return &variableSlot{value: value}, nil
		}
		if param.ByRef {
			return nil, fmt.Errorf("argument %d for %s must be a variable passed by reference", idx+1, fn.Name)
		}
		return &variableSlot{value: args[idx]}, nil
	})
	return result.value, err
}

func (i *Interpreter) callClosureResult(closure *Closure, args []ast.Expr) (functionCallResult, error) {
	if closure == nil || closure.Fn == nil {
		return functionCallResult{}, fmt.Errorf("invalid closure")
	}
	fn := closure.Fn
	return i.callFunctionWithBinderResult(fn, len(args), nil, "", closure.Captures, func(idx int, param ast.Param) (*variableSlot, error) {
		if idx >= len(args) {
			if param.Default == nil {
				return i.missingArgumentSlot(fn, len(args))
			}
			value, err := i.eval(param.Default)
			if err != nil {
				return nil, err
			}
			return &variableSlot{value: value}, nil
		}
		if param.ByRef {
			slot, err := i.referenceSlot(args[idx])
			if err != nil {
				return nil, fmt.Errorf("argument %d for %s must be a variable passed by reference", idx+1, fn.Name)
			}
			return slot, nil
		}
		value, err := i.eval(args[idx])
		if err != nil {
			return nil, err
		}
		return &variableSlot{value: value}, nil
	})
}

func (i *Interpreter) missingArgumentSlot(fn *ast.FunctionDeclStmt, provided int) (*variableSlot, error) {
	if err := i.emitDiagnostic(Warning, fmt.Sprintf("function %s expects at least %d arguments, got %d", fn.Name, i.requiredParamCount(fn), provided)); err != nil {
		return nil, err
	}
	return &variableSlot{value: NullValue()}, nil
}

func (i *Interpreter) callFunctionWithBinder(fn *ast.FunctionDeclStmt, provided int, thisObj *Object, calledClass string, bind func(int, ast.Param) (*variableSlot, error)) (Value, error) {
	result, err := i.callFunctionWithBinderResult(fn, provided, thisObj, calledClass, nil, bind)
	return result.value, err
}

func (i *Interpreter) callFunctionWithBinderResult(fn *ast.FunctionDeclStmt, provided int, thisObj *Object, calledClass string, captures map[string]*variableSlot, bind func(int, ast.Param) (*variableSlot, error)) (functionCallResult, error) {
	if fn.Abstract {
		return functionCallResult{}, fmt.Errorf("cannot call abstract method %s::%s", fn.Owner, fn.Name)
	}
	if provided > len(fn.Params) {
		return functionCallResult{}, fmt.Errorf("function %s expects at most %d arguments, got %d", fn.Name, len(fn.Params), provided)
	}
	if calledClass == "" {
		calledClass = fn.Owner
	}
	callFrame := frame{vars: map[string]*variableSlot{}, className: fn.Owner, calledClass: calledClass, function: fn.Name, functionKey: functionKey(fn), returnByRef: fn.ReturnsByRef}
	for name, slot := range captures {
		callFrame.vars[name] = slot
	}
	if thisObj != nil {
		callFrame.vars["this"] = &variableSlot{value: ObjectValue(thisObj)}
	}
	for idx, param := range fn.Params {
		slot, err := bind(idx, param)
		if err != nil {
			return functionCallResult{}, err
		}
		if err := i.checkParamTypeHint(fn, param, slot.value, idx); err != nil {
			return functionCallResult{}, err
		}
		callFrame.vars[param.Name] = slot
	}
	i.frames = append(i.frames, callFrame)
	ret, err := i.exec(fn.Body)
	i.frames = i.frames[:len(i.frames)-1]
	if err != nil {
		return functionCallResult{}, err
	}
	if ret == nil {
		return functionCallResult{value: NullValue()}, nil
	}
	return functionCallResult{value: ret.value, slot: ret.slot}, nil
}

func (i *Interpreter) requiredParamCount(fn *ast.FunctionDeclStmt) int {
	return requiredParamCount(fn)
}

func functionKey(fn *ast.FunctionDeclStmt) string {
	if fn.Owner != "" {
		return fn.Owner + "::" + fn.Name
	}
	return fn.Name
}

func (i *Interpreter) checkParamTypeHint(fn *ast.FunctionDeclStmt, param ast.Param, value Value, idx int) error {
	if param.TypeHint == "" {
		return nil
	}
	if value.Kind == Null && paramDefaultAllowsNull(param) {
		return nil
	}
	if strings.EqualFold(param.TypeHint, "array") {
		if value.Kind == ArrayKind {
			return nil
		}
		return i.recoverableErrorf("argument %d for %s must be of type array", idx+1, functionKey(fn))
	}
	typeName, err := i.resolveParamTypeHint(fn, param.TypeHint)
	if err != nil {
		return err
	}
	if value.Kind == ObjectKind && value.O != nil {
		matches, err := i.classLikeMatches(value.O.ClassName, typeName, true)
		if err != nil {
			return err
		}
		if matches {
			return nil
		}
	}
	return i.recoverableErrorf("argument %d for %s must be of type %s", idx+1, functionKey(fn), typeName)
}

func paramDefaultAllowsNull(param ast.Param) bool {
	_, ok := param.Default.(*ast.NullLit)
	return ok
}

func (i *Interpreter) resolveParamTypeHint(fn *ast.FunctionDeclStmt, hint string) (string, error) {
	switch hint {
	case "self":
		if fn.Owner == "" {
			return "", fmt.Errorf("self type hint used outside class context")
		}
		return fn.Owner, nil
	case "parent":
		if fn.Owner == "" {
			return "", fmt.Errorf("parent type hint used outside class context")
		}
		class, ok := i.classByName(fn.Owner)
		if !ok || class.Parent == "" {
			return "", fmt.Errorf("class %s has no parent for parent type hint", fn.Owner)
		}
		return class.Parent, nil
	default:
		return hint, nil
	}
}

func (i *Interpreter) newObject(n *ast.NewExpr) (Value, error) {
	className, err := i.newClassName(n)
	if err != nil {
		return NullValue(), err
	}
	if className == "Exception" {
		return i.newException(n)
	}
	class, ok := i.classByName(className)
	if !ok {
		if iface, interfaceOK := i.interfaceByName(className); interfaceOK {
			return NullValue(), fmt.Errorf("cannot instantiate interface %s", iface.Name)
		}
		return NullValue(), fmt.Errorf("undefined class %s", className)
	}
	if class.Abstract {
		return NullValue(), fmt.Errorf("cannot instantiate abstract class %s", class.Name)
	}
	props, meta, err := i.instanceDefaults(class)
	if err != nil {
		return NullValue(), err
	}
	obj := NewObject(class.Name, i.methodsForClass(class), props, meta)
	if constructor, ok := i.constructorForClass(class); ok {
		if err := i.checkMethodAccess(class.Name, constructor); err != nil {
			return NullValue(), err
		}
		if _, err := i.callFunction(constructor, n.Args, obj, obj.ClassName); err != nil {
			return NullValue(), err
		}
	} else if i.isExceptionObject(obj) {
		if err := i.initExceptionObject(obj, n.Args, i.currentFile(), n.Line); err != nil {
			return NullValue(), err
		}
	} else if len(n.Args) != 0 {
		return NullValue(), fmt.Errorf("class %s has no constructor", className)
	}
	i.trackObject(obj)
	return ObjectValue(obj), nil
}

func (i *Interpreter) newClassName(n *ast.NewExpr) (string, error) {
	if n.ClassExpr != nil {
		value, err := i.eval(n.ClassExpr)
		if err != nil {
			return "", err
		}
		name := value.String()
		if name == "" {
			return "", fmt.Errorf("class name must not be empty")
		}
		return i.resolveClassNameAutoload(name)
	}
	return i.resolveClassNameAutoload(n.ClassName)
}

func (i *Interpreter) trackObject(obj *Object) {
	if obj == nil {
		return
	}
	i.objects = append(i.objects, obj)
}

func (i *Interpreter) evalClone(expr ast.Expr) (Value, error) {
	value, err := i.eval(expr)
	if err != nil {
		return NullValue(), err
	}
	if value.Kind != ObjectKind || value.O == nil {
		return NullValue(), fmt.Errorf("clone expects object")
	}
	clone := i.cloneObject(value.O)
	if method, ok := i.objectMethod(clone, "__clone"); ok {
		if err := i.checkMethodAccess(clone.ClassName, method); err != nil {
			return NullValue(), err
		}
		if _, err := i.callFunction(method, nil, clone, clone.ClassName); err != nil {
			return NullValue(), err
		}
	}
	i.trackObject(clone)
	return ObjectValue(clone), nil
}

func (i *Interpreter) cloneObject(obj *Object) *Object {
	props := map[string]Value{}
	for name, value := range obj.Props {
		props[name] = value
	}
	meta := map[string]PropertyMeta{}
	for name, info := range obj.PropMeta {
		meta[name] = info
	}
	clone := NewObject(obj.ClassName, obj.Methods, props, meta)
	clone.ExceptionTrace = append([]ExceptionTraceFrame(nil), obj.ExceptionTrace...)
	return clone
}

func (i *Interpreter) runDestructors() error {
	for {
		obj := i.nextUndestructedObject()
		if obj == nil {
			return nil
		}
		i.destructed[obj] = true
		destructor, ok := i.objectMethod(obj, "__destruct")
		if !ok {
			continue
		}
		if _, err := i.callFunction(destructor, nil, obj, obj.ClassName); err != nil {
			return err
		}
	}
}

func (i *Interpreter) nextUndestructedObject() *Object {
	for idx := len(i.objects) - 1; idx >= 0; idx-- {
		obj := i.objects[idx]
		if obj == nil || i.destructed[obj] {
			continue
		}
		return obj
	}
	return nil
}

func (i *Interpreter) constructorForClass(class *ast.ClassDeclStmt) (*ast.FunctionDeclStmt, bool) {
	if class == nil {
		return nil, false
	}
	if method, ok := declaredMethod(class, "__construct"); ok {
		return method, true
	}
	for _, method := range class.Methods {
		if isOldStyleConstructor(class.Name, method.Name) {
			return method, true
		}
	}
	if class.Parent == "" {
		return nil, false
	}
	parent, ok := i.classByName(class.Parent)
	if !ok {
		return nil, false
	}
	return i.constructorForClass(parent)
}

func declaredMethod(class *ast.ClassDeclStmt, name string) (*ast.FunctionDeclStmt, bool) {
	for _, method := range class.Methods {
		if strings.EqualFold(method.Name, name) {
			return method, true
		}
	}
	return nil, false
}

func classDeclaresMethod(class *ast.ClassDeclStmt, name string) bool {
	if class == nil {
		return false
	}
	_, ok := declaredMethod(class, name)
	return ok
}

func isOldStyleConstructor(className, methodName string) bool {
	if strings.Contains(className, "\\") {
		return false
	}
	return strings.EqualFold(methodName, className)
}

func (i *Interpreter) callMethod(n *ast.MethodCallExpr) (Value, error) {
	return i.callMethodName(n.Object, n.Method, n.Args)
}

func (i *Interpreter) callDynamicMethod(n *ast.DynamicMethodCallExpr) (Value, error) {
	method, err := i.dynamicMemberName(n.Method)
	if err != nil {
		return NullValue(), err
	}
	return i.callMethodName(n.Object, method, n.Args)
}

func (i *Interpreter) callInvokableObject(obj *Object, args []ast.Expr) (Value, error) {
	if obj == nil {
		return NullValue(), fmt.Errorf("object is not callable")
	}
	method, ok := i.objectMethod(obj, "__invoke")
	if !ok {
		return NullValue(), fmt.Errorf("object of class %s is not callable", obj.ClassName)
	}
	if err := i.checkMethodAccess(obj.ClassName, method); err != nil {
		return NullValue(), err
	}
	return i.callFunction(method, args, obj, obj.ClassName)
}

func (i *Interpreter) callInvokableObjectValues(obj *Object, args []Value) (Value, error) {
	if obj == nil {
		return NullValue(), fmt.Errorf("object is not callable")
	}
	method, ok := i.objectMethod(obj, "__invoke")
	if !ok {
		return NullValue(), fmt.Errorf("object of class %s is not callable", obj.ClassName)
	}
	if err := i.checkMethodAccess(obj.ClassName, method); err != nil {
		return NullValue(), err
	}
	return i.callFunctionValues(method, args, obj, obj.ClassName)
}

func (i *Interpreter) callMethodName(object ast.Expr, methodName string, args []ast.Expr) (Value, error) {
	obj, err := i.eval(object)
	if err != nil {
		return NullValue(), err
	}
	if obj.Kind != ObjectKind || obj.O == nil {
		return NullValue(), fmt.Errorf("cannot call method %s on non-object", methodName)
	}
	return i.callObjectMethodValue(obj.O, methodName, args)
}

func (i *Interpreter) callObjectMethodValue(obj *Object, methodName string, args []ast.Expr) (Value, error) {
	if obj == nil {
		return NullValue(), fmt.Errorf("cannot call method %s on non-object", methodName)
	}
	if i.isExceptionObject(obj) {
		if value, ok, err := i.callExceptionMethod(obj, methodName, args); ok || err != nil {
			return value, err
		}
	}
	if strings.EqualFold(methodName, "__clone") {
		return NullValue(), fmt.Errorf("cannot call __clone directly; use clone")
	}
	method, ok := i.objectMethod(obj, methodName)
	if !ok {
		if methodName != "__call" && i.hasPublicMethod(obj, "__call") {
			return i.callMagicCall(obj, methodName, args)
		}
		return NullValue(), fmt.Errorf("undefined method %s::%s", obj.ClassName, methodName)
	}
	if err := i.checkMethodAccess(obj.ClassName, method); err != nil {
		if methodName != "__call" && i.hasPublicMethod(obj, "__call") {
			return i.callMagicCall(obj, methodName, args)
		}
		return NullValue(), err
	}
	return i.callFunction(method, args, obj, obj.ClassName)
}

func (i *Interpreter) callObjectMethodValueValues(obj *Object, methodName string, args []Value) (Value, error) {
	if obj == nil {
		return NullValue(), fmt.Errorf("cannot call method %s on non-object", methodName)
	}
	if strings.EqualFold(methodName, "__clone") {
		return NullValue(), fmt.Errorf("cannot call __clone directly; use clone")
	}
	method, ok := i.objectMethod(obj, methodName)
	if !ok {
		if methodName != "__call" && i.hasPublicMethod(obj, "__call") {
			return i.callMagicCallValues(obj, methodName, args)
		}
		return NullValue(), fmt.Errorf("undefined method %s::%s", obj.ClassName, methodName)
	}
	if err := i.checkMethodAccess(obj.ClassName, method); err != nil {
		if methodName != "__call" && i.hasPublicMethod(obj, "__call") {
			return i.callMagicCallValues(obj, methodName, args)
		}
		return NullValue(), err
	}
	return i.callFunctionValues(method, args, obj, obj.ClassName)
}

func (i *Interpreter) callStatic(n *ast.StaticCallExpr) (Value, error) {
	return i.callStaticName(n.ClassName, n.Method, n.Args)
}

func (i *Interpreter) callDynamicStatic(n *ast.DynamicStaticCallExpr) (Value, error) {
	method, err := i.dynamicMemberName(n.Method)
	if err != nil {
		return NullValue(), err
	}
	return i.callStaticName(n.ClassName, method, n.Args)
}

func (i *Interpreter) callStaticName(staticClassName, methodName string, args []ast.Expr) (Value, error) {
	className, err := i.resolveClassNameAutoload(staticClassName)
	if err != nil {
		return NullValue(), err
	}
	class, ok := i.classByName(className)
	if !ok {
		return NullValue(), fmt.Errorf("undefined class %s", className)
	}
	if strings.EqualFold(methodName, "__clone") {
		return NullValue(), fmt.Errorf("cannot call __clone directly; use clone")
	}
	methods := i.methodsForClass(class)
	method, ok := i.staticMethod(methods, className, methodName)
	if !ok {
		if i.hasPublicStaticMethod(methods, "__callStatic") {
			return i.callMagicCallStatic(i.staticCallCalledClass(staticClassName, className), methods[methodNameKey("__callStatic")], methodName, args)
		}
		return NullValue(), fmt.Errorf("undefined static method %s::%s", className, methodName)
	}
	if !method.Static {
		thisObj := i.currentThis()
		if thisObj == nil {
			return NullValue(), fmt.Errorf("method %s::%s is not static", className, methodName)
		}
		if err := i.checkMethodAccess(className, method); err != nil {
			return NullValue(), err
		}
		return i.callFunction(method, args, thisObj, thisObj.ClassName)
	}
	if err := i.checkMethodAccess(className, method); err != nil {
		if methodName != "__callStatic" && i.hasPublicStaticMethod(methods, "__callStatic") {
			return i.callMagicCallStatic(i.staticCallCalledClass(staticClassName, className), methods[methodNameKey("__callStatic")], methodName, args)
		}
		return NullValue(), err
	}
	return i.callFunction(method, args, nil, i.staticCallCalledClass(staticClassName, className))
}

func (i *Interpreter) callStaticNameValues(staticClassName, methodName string, args []Value) (Value, error) {
	className, err := i.resolveClassNameAutoload(staticClassName)
	if err != nil {
		return NullValue(), err
	}
	class, ok := i.classByName(className)
	if !ok {
		return NullValue(), fmt.Errorf("undefined class %s", className)
	}
	if strings.EqualFold(methodName, "__clone") {
		return NullValue(), fmt.Errorf("cannot call __clone directly; use clone")
	}
	methods := i.methodsForClass(class)
	method, ok := i.staticMethod(methods, className, methodName)
	if !ok {
		if i.hasPublicStaticMethod(methods, "__callStatic") {
			return i.callMagicCallStaticValues(i.staticCallCalledClass(staticClassName, className), methods[methodNameKey("__callStatic")], methodName, args)
		}
		return NullValue(), fmt.Errorf("undefined static method %s::%s", className, methodName)
	}
	if !method.Static {
		thisObj := i.currentThis()
		if thisObj == nil {
			return NullValue(), fmt.Errorf("method %s::%s is not static", className, methodName)
		}
		if err := i.checkMethodAccess(className, method); err != nil {
			return NullValue(), err
		}
		return i.callFunctionValues(method, args, thisObj, thisObj.ClassName)
	}
	if err := i.checkMethodAccess(className, method); err != nil {
		if methodName != "__callStatic" && i.hasPublicStaticMethod(methods, "__callStatic") {
			return i.callMagicCallStaticValues(i.staticCallCalledClass(staticClassName, className), methods[methodNameKey("__callStatic")], methodName, args)
		}
		return NullValue(), err
	}
	return i.callFunctionValues(method, args, nil, i.staticCallCalledClass(staticClassName, className))
}

func (i *Interpreter) staticCallCalledClass(staticClassName, resolvedClassName string) string {
	switch staticClassName {
	case "self", "parent", "static":
		if called := i.currentCalledClass(); called != "" {
			return called
		}
	}
	return resolvedClassName
}

func (i *Interpreter) execSwitch(n *ast.SwitchStmt) (*returnSignal, error) {
	switchValue, err := i.eval(n.Value)
	if err != nil {
		return nil, err
	}
	start := -1
	defaultIndex := -1
	for idx, c := range n.Cases {
		if c.Default {
			if defaultIndex == -1 {
				defaultIndex = idx
			}
			continue
		}
		value, err := i.eval(c.Expr)
		if err != nil {
			return nil, err
		}
		matched, err := Binary("==", switchValue, value)
		if err != nil {
			return nil, err
		}
		if matched.Truthy() {
			start = idx
			break
		}
	}
	if start == -1 {
		start = defaultIndex
	}
	if start == -1 {
		return nil, nil
	}
	for idx := start; idx < len(n.Cases); idx++ {
		for _, stmt := range n.Cases[idx].Statements {
			ret, err := i.exec(stmt)
			if err != nil || ret != nil {
				return ret, err
			}
		}
	}
	return nil, nil
}

func consumeBreak(err error) (bool, error, bool) {
	sig, ok := err.(*breakSignal)
	if !ok {
		return false, err, false
	}
	if sig.Level <= 1 {
		return true, nil, true
	}
	return false, &breakSignal{Level: sig.Level - 1}, true
}

func consumeContinue(err error) (bool, error, bool) {
	sig, ok := err.(*continueSignal)
	if !ok {
		return false, err, false
	}
	if sig.Level <= 1 {
		return true, nil, true
	}
	return false, &continueSignal{Level: sig.Level - 1}, true
}

func (i *Interpreter) evalIndex(n *ast.IndexExpr) (Value, error) {
	if n.Index == nil {
		return NullValue(), fmt.Errorf("cannot use [] for reading")
	}
	left, err := i.eval(n.Left)
	if err != nil {
		return NullValue(), err
	}
	index, err := i.eval(n.Index)
	if err != nil {
		return NullValue(), err
	}
	switch left.Kind {
	case ArrayKind:
		key := KeyFromValue(index)
		if left.A == nil {
			if err := i.noticef("%s", undefinedArrayIndexMessage(key)); err != nil {
				return NullValue(), err
			}
			return NullValue(), nil
		}
		value, ok := left.A.Get(key)
		if !ok {
			if err := i.noticef("%s", undefinedArrayIndexMessage(key)); err != nil {
				return NullValue(), err
			}
			return NullValue(), nil
		}
		return value, nil
	case String:
		offset, ok := indexOffset(index)
		if !ok || offset < 0 || offset >= len(left.S) {
			if err := i.noticef("undefined offset %s", index.String()); err != nil {
				return NullValue(), err
			}
			return NullValue(), nil
		}
		return StringValue(left.S[offset : offset+1]), nil
	default:
		if err := i.noticef("cannot use index on %s", left.String()); err != nil {
			return NullValue(), err
		}
		return NullValue(), nil
	}
}

func undefinedArrayIndexMessage(key ArrayKey) string {
	if key.IsInt {
		return fmt.Sprintf("undefined offset %d", key.Int)
	}
	return fmt.Sprintf("undefined index %s", key.Str)
}

func (i *Interpreter) evalAssignExpr(n *ast.AssignExpr) (Value, error) {
	if n.ByRef {
		slot, err := i.referenceSlot(n.Value)
		if err != nil {
			return NullValue(), err
		}
		if err := i.bindReferenceTarget(n.Target, slot); err != nil {
			return NullValue(), err
		}
		return slot.value, nil
	}
	value, err := i.eval(n.Value)
	if err != nil {
		return NullValue(), err
	}
	if n.Op != "=" {
		current, err := i.eval(n.Target)
		if err != nil {
			return NullValue(), err
		}
		if n.Op == "." {
			left, err := i.stringify(current)
			if err != nil {
				return NullValue(), err
			}
			right, err := i.stringify(value)
			if err != nil {
				return NullValue(), err
			}
			value = StringValue(left + right)
		} else {
			value, err = i.binary(n.Op, current, value)
			if err != nil {
				return NullValue(), err
			}
		}
	}
	if err := i.assignTarget(n.Target, value); err != nil {
		return NullValue(), err
	}
	return value, nil
}

func (i *Interpreter) referenceSlot(expr ast.Expr) (*variableSlot, error) {
	switch n := expr.(type) {
	case *ast.Ident:
		return i.ensureSlot(n.Name), nil
	case *ast.VariableVariable:
		return i.ensureSlot(i.variableVariableName(n.Name)), nil
	case *ast.IndexExpr:
		return i.arrayIndexSlot(n)
	case *ast.CallExpr:
		return i.callFunctionReference(n)
	case *ast.VariableCallExpr:
		return i.callVariableFunctionReference(n)
	default:
		return nil, fmt.Errorf("reference assignment value must be a variable")
	}
}

func (i *Interpreter) callFunctionReference(c *ast.CallExpr) (*variableSlot, error) {
	fn, ok := i.functions[functionNameKey(c.Name)]
	if !ok {
		if base := namespaceBaseName(c.Name); base != c.Name {
			return i.callFunctionReference(&ast.CallExpr{Name: base, Args: c.Args})
		}
		return nil, fmt.Errorf("undefined function %s", c.Name)
	}
	if !fn.ReturnsByRef {
		return nil, fmt.Errorf("function %s does not return by reference", c.Name)
	}
	result, err := i.callFunctionResult(fn, c.Args, nil, "")
	if err != nil {
		return nil, err
	}
	if result.slot == nil {
		return nil, fmt.Errorf("function %s did not return a reference", c.Name)
	}
	return result.slot, nil
}

func (i *Interpreter) callVariableFunctionReference(c *ast.VariableCallExpr) (*variableSlot, error) {
	callee, err := i.eval(c.Callee)
	if err != nil {
		return nil, err
	}
	if callee.Kind == ClosureKind {
		if callee.C == nil || callee.C.Fn == nil {
			return nil, fmt.Errorf("invalid closure")
		}
		if !callee.C.Fn.ReturnsByRef {
			return nil, fmt.Errorf("closure does not return by reference")
		}
		result, err := i.callClosureResult(callee.C, c.Args)
		if err != nil {
			return nil, err
		}
		if result.slot == nil {
			return nil, fmt.Errorf("closure did not return a reference")
		}
		return result.slot, nil
	}
	return i.callFunctionReference(&ast.CallExpr{Name: callee.String(), Args: c.Args})
}

func (i *Interpreter) bindReferenceTarget(target ast.Expr, slot *variableSlot) error {
	switch n := target.(type) {
	case *ast.Ident:
		i.bindSlot(n.Name, slot)
		return nil
	case *ast.VariableVariable:
		i.bindSlot(i.variableVariableName(n.Name), slot)
		return nil
	case *ast.IndexExpr:
		return i.bindIndexReference(n, slot)
	default:
		return fmt.Errorf("reference assignment target must be a variable")
	}
}

func (i *Interpreter) arrayIndexSlot(target *ast.IndexExpr) (*variableSlot, error) {
	if target.Index == nil {
		return nil, fmt.Errorf("cannot use [] as reference assignment value")
	}
	arr, err := i.ensureWritableArray(target.Left)
	if err != nil {
		return nil, err
	}
	index, err := i.eval(target.Index)
	if err != nil {
		return nil, err
	}
	return arr.EnsureSlot(KeyFromValue(index)), nil
}

func (i *Interpreter) bindIndexReference(target *ast.IndexExpr, slot *variableSlot) error {
	arr, err := i.ensureWritableArray(target.Left)
	if err != nil {
		return err
	}
	if target.Index == nil {
		arr.AppendSlot(slot)
		return nil
	}
	index, err := i.eval(target.Index)
	if err != nil {
		return err
	}
	arr.SetSlot(KeyFromValue(index), slot)
	return nil
}

func (i *Interpreter) evalIncDec(target ast.Expr, op string, post bool) (Value, error) {
	current, err := i.eval(target)
	if err != nil {
		return NullValue(), err
	}
	next := incrementValue(current)
	if op == "--" {
		next = decrementValue(current)
	}
	if err := i.assignTarget(target, next); err != nil {
		return NullValue(), err
	}
	if post {
		return current, nil
	}
	return next, nil
}

func incrementValue(value Value) Value {
	switch value.Kind {
	case Int:
		return IntValue(value.I + 1)
	case Float:
		return FloatValue(value.F + 1)
	case Null:
		return IntValue(1)
	case String:
		if value.S == "" {
			return StringValue("1")
		}
		if numeric, ok := parseNumericStringValue(value.S); ok {
			return incrementValue(numeric)
		}
		return StringValue(incrementString(value.S))
	default:
		return value
	}
}

func decrementValue(value Value) Value {
	switch value.Kind {
	case Int:
		return IntValue(value.I - 1)
	case Float:
		return FloatValue(value.F - 1)
	case String:
		if value.S == "" {
			return IntValue(-1)
		}
		if numeric, ok := parseNumericStringValue(value.S); ok {
			return decrementValue(numeric)
		}
		return value
	default:
		return value
	}
}

func parseNumericStringValue(text string) (Value, bool) {
	start := 0
	for start < len(text) && isPHPWhitespace(text[start]) {
		start++
	}
	if start == len(text) {
		return NullValue(), false
	}

	body := text[start:]
	if len(body) > 2 && body[0] == '0' && (body[1] == 'x' || body[1] == 'X') {
		if len(body) == 2 {
			return NullValue(), false
		}
		for idx := 2; idx < len(body); idx++ {
			if !isHexDigit(body[idx]) {
				return NullValue(), false
			}
		}
		if value, err := strconv.ParseInt(body[2:], 16, 64); err == nil {
			return IntValue(value), true
		}
		if value, err := strconv.ParseUint(body[2:], 16, 64); err == nil {
			return FloatValue(float64(value)), true
		}
		return NullValue(), false
	}

	end, isFloat := numericStringEnd(body)
	if end != len(body) {
		return NullValue(), false
	}
	if !isFloat {
		if value, err := strconv.ParseInt(body, 10, 64); err == nil {
			return IntValue(value), true
		}
	}
	if value, err := strconv.ParseFloat(body, 64); err == nil {
		if isFloat {
			return FloatValue(value), true
		}
		return FloatValue(value), true
	}
	return NullValue(), false
}

func isPHPWhitespace(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' || ch == '\v' || ch == '\f'
}

func isHexDigit(ch byte) bool {
	return (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')
}

func numericStringEnd(text string) (int, bool) {
	if text == "" {
		return 0, false
	}
	end := 0
	if text[end] == '+' || text[end] == '-' {
		end++
		if end == len(text) {
			return end, false
		}
	}
	digits := 0
	for end < len(text) && text[end] >= '0' && text[end] <= '9' {
		end++
		digits++
	}
	isFloat := false
	if end < len(text) && text[end] == '.' {
		isFloat = true
		end++
		for end < len(text) && text[end] >= '0' && text[end] <= '9' {
			end++
			digits++
		}
	}
	if digits == 0 {
		return end, false
	}
	if end < len(text) && (text[end] == 'e' || text[end] == 'E') {
		expEnd := end + 1
		if expEnd < len(text) && (text[expEnd] == '+' || text[expEnd] == '-') {
			expEnd++
		}
		expDigits := 0
		for expEnd < len(text) && text[expEnd] >= '0' && text[expEnd] <= '9' {
			expEnd++
			expDigits++
		}
		if expDigits > 0 {
			end = expEnd
			isFloat = true
		}
	}
	return end, isFloat
}

func incrementString(text string) string {
	data := []byte(text)
	carry := false
	last := byte(0)

	for pos := len(data) - 1; pos >= 0; pos-- {
		ch := data[pos]
		switch {
		case ch >= 'a' && ch <= 'z':
			last = 'a'
			if ch == 'z' {
				data[pos] = 'a'
				carry = true
			} else {
				data[pos]++
				carry = false
			}
		case ch >= 'A' && ch <= 'Z':
			last = 'A'
			if ch == 'Z' {
				data[pos] = 'A'
				carry = true
			} else {
				data[pos]++
				carry = false
			}
		case ch >= '0' && ch <= '9':
			last = '1'
			if ch == '9' {
				data[pos] = '0'
				carry = true
			} else {
				data[pos]++
				carry = false
			}
		default:
			carry = false
		}
		if !carry {
			break
		}
	}
	if carry {
		data = append([]byte{last}, data...)
	}
	return string(data)
}

func (i *Interpreter) assignIndex(target *ast.IndexExpr, value Value) error {
	var key ArrayKey
	appendValue := target.Index == nil
	if !appendValue {
		index, err := i.eval(target.Index)
		if err != nil {
			return err
		}
		key = KeyFromValue(index)
	}
	arr, err := i.ensureWritableArray(target.Left)
	if err != nil {
		return err
	}
	if arr == nil {
		return nil
	}
	if appendValue {
		arr.Append(value)
	} else {
		arr.Set(key, value)
	}
	return nil
}

func (i *Interpreter) ensureWritableArray(target ast.Expr) (*Array, error) {
	switch n := target.(type) {
	case *ast.Ident:
		current, exists := i.getVar(n.Name)
		if !exists || current.Kind == Null {
			arr := NewArray()
			i.setVar(n.Name, ArrayValue(arr))
			return arr, nil
		}
		if current.Kind != ArrayKind || current.A == nil {
			return i.warnCannotAssignIndexOnNonArray()
		}
		return current.A, nil
	case *ast.VariableVariable:
		name := i.variableVariableName(n.Name)
		current, exists := i.getVar(name)
		if !exists || current.Kind == Null {
			arr := NewArray()
			i.setVar(name, ArrayValue(arr))
			return arr, nil
		}
		if current.Kind != ArrayKind || current.A == nil {
			return i.warnCannotAssignIndexOnNonArray()
		}
		return current.A, nil
	case *ast.IndexExpr:
		if n.Index == nil {
			return nil, fmt.Errorf("cannot use [] as nested assignment target")
		}
		parent, err := i.ensureWritableArray(n.Left)
		if err != nil {
			return nil, err
		}
		if parent == nil {
			return nil, nil
		}
		index, err := i.eval(n.Index)
		if err != nil {
			return nil, err
		}
		key := KeyFromValue(index)
		current, exists := parent.Get(key)
		if !exists || current.Kind == Null {
			arr := NewArray()
			parent.Set(key, ArrayValue(arr))
			return arr, nil
		}
		if current.Kind != ArrayKind || current.A == nil {
			return i.warnCannotAssignIndexOnNonArray()
		}
		return current.A, nil
	default:
		value, err := i.eval(target)
		if err != nil {
			return nil, err
		}
		if value.Kind != ArrayKind || value.A == nil {
			return i.warnCannotAssignIndexOnNonArray()
		}
		return value.A, nil
	}
}

func (i *Interpreter) warnCannotAssignIndexOnNonArray() (*Array, error) {
	if err := i.emitDiagnostic(Warning, "cannot assign index on non-array"); err != nil {
		return nil, err
	}
	return nil, nil
}

func (i *Interpreter) assignTarget(target ast.Expr, value Value) error {
	switch n := target.(type) {
	case *ast.Ident:
		i.setVar(n.Name, value)
		return nil
	case *ast.VariableVariable:
		i.setVar(i.variableVariableName(n.Name), value)
		return nil
	case *ast.PropertyFetchExpr:
		return i.assignProperty(n.Object, n.Property, value)
	case *ast.DynamicPropertyFetchExpr:
		property, err := i.dynamicMemberName(n.Property)
		if err != nil {
			return err
		}
		return i.assignProperty(n.Object, property, value)
	case *ast.StaticPropertyFetchExpr:
		return i.assignStaticProperty(n.ClassName, n.Property, value)
	case *ast.DynamicStaticPropertyFetchExpr:
		property, err := i.dynamicMemberName(n.Property)
		if err != nil {
			return err
		}
		return i.assignStaticProperty(n.ClassName, property, value)
	case *ast.IndexExpr:
		return i.assignIndex(n, value)
	case *ast.ListExpr:
		return i.assignList(n, value)
	default:
		return fmt.Errorf("expression is not writable")
	}
}

func (i *Interpreter) assignStaticProperty(className, property string, value Value) error {
	if err := i.checkStaticPropertyAccess(className, property); err != nil {
		return err
	}
	i.setStaticProperty(className, property, value)
	return nil
}

func (i *Interpreter) assignProperty(object ast.Expr, property string, value Value) error {
	obj, err := i.eval(object)
	if err != nil {
		return err
	}
	if obj.Kind != ObjectKind || obj.O == nil {
		return fmt.Errorf("cannot assign property %s on non-object", property)
	}
	key, _, info, ok, inaccessible := i.objectProperty(obj.O, property)
	if inaccessible {
		if i.hasPublicMethod(obj.O, "__set") {
			_, callErr := i.callMagicSet(obj.O, property, value)
			return callErr
		}
		return fmt.Errorf("cannot access %s property %s::$%s", info.Visibility, info.Owner, property)
	}
	if !ok && i.shouldUseSet(obj.O) {
		_, callErr := i.callMagicSet(obj.O, property, value)
		return callErr
	}
	if key == "" {
		key = property
	}
	obj.O.Props[key] = value
	return nil
}

func (i *Interpreter) assignList(target *ast.ListExpr, value Value) error {
	if value.Kind != ArrayKind || value.A == nil {
		if err := i.emitDiagnostic(Warning, "list assignment expects array"); err != nil {
			return err
		}
		for _, elem := range target.Targets {
			if elem == nil {
				continue
			}
			if err := i.assignTarget(elem, NullValue()); err != nil {
				return err
			}
		}
		return nil
	}
	for idx, elem := range target.Targets {
		if elem == nil {
			continue
		}
		elemValue, ok := value.A.Get(IntKey(int64(idx)))
		if !ok {
			elemValue = NullValue()
		}
		if err := i.assignTarget(elem, elemValue); err != nil {
			return err
		}
	}
	return nil
}

func (i *Interpreter) unsetTarget(target ast.Expr) error {
	switch n := target.(type) {
	case *ast.Ident:
		i.deleteVar(n.Name)
		return nil
	case *ast.VariableVariable:
		i.deleteVar(i.variableVariableName(n.Name))
		return nil
	case *ast.IndexExpr:
		if n.Index == nil {
			return fmt.Errorf("cannot use [] for unset")
		}
		index, err := i.eval(n.Index)
		if err != nil {
			return err
		}
		key := KeyFromValue(index)
		if ident, ok := n.Left.(*ast.Ident); ok {
			current, exists := i.getVar(ident.Name)
			if !exists || current.Kind != ArrayKind || current.A == nil {
				return nil
			}
			current.A.Delete(key)
			return nil
		}
		left, err := i.eval(n.Left)
		if err != nil {
			return err
		}
		if left.Kind == ArrayKind && left.A != nil {
			left.A.Delete(key)
		}
		return nil
	case *ast.PropertyFetchExpr:
		return i.unsetProperty(n.Object, n.Property)
	case *ast.DynamicPropertyFetchExpr:
		property, err := i.dynamicMemberName(n.Property)
		if err != nil {
			return err
		}
		return i.unsetProperty(n.Object, property)
	case *ast.StaticPropertyFetchExpr:
		return i.unsetStaticProperty(n.ClassName, n.Property)
	case *ast.DynamicStaticPropertyFetchExpr:
		property, err := i.dynamicMemberName(n.Property)
		if err != nil {
			return err
		}
		return i.unsetStaticProperty(n.ClassName, property)
	default:
		return nil
	}
}

func (i *Interpreter) unsetProperty(object ast.Expr, property string) error {
	obj, err := i.eval(object)
	if err != nil {
		return err
	}
	if obj.Kind != ObjectKind || obj.O == nil {
		return nil
	}
	key, _, info, ok, inaccessible := i.objectProperty(obj.O, property)
	if inaccessible {
		if i.shouldUseUnset(obj.O) {
			_, callErr := i.callMagicUnset(obj.O, property)
			return callErr
		}
		return fmt.Errorf("cannot access %s property %s::$%s", info.Visibility, info.Owner, property)
	} else if !ok && i.shouldUseUnset(obj.O) {
		_, callErr := i.callMagicUnset(obj.O, property)
		return callErr
	}
	if key == "" {
		key = property
	}
	delete(obj.O.Props, key)
	delete(obj.O.PropMeta, key)
	return nil
}

func (i *Interpreter) unsetStaticProperty(className, property string) error {
	if err := i.checkStaticPropertyAccess(className, property); err != nil {
		return err
	}
	resolved, err := i.resolveClassNameAutoload(className)
	if err != nil {
		return err
	}
	owner := resolved
	if _, info, ok := i.findStaticProperty(resolved, property); ok {
		owner = info.Owner
	}
	if _, _, slotOwner, key, ok := i.findStaticPropertySlot(resolved, property); ok {
		owner = slotOwner
		property = key
	}
	if props, ok := i.statics[owner]; ok {
		delete(props, property)
	}
	if meta, ok := i.staticMeta[owner]; ok {
		delete(meta, property)
	}
	return nil
}

func indexOffset(value Value) (int, bool) {
	switch value.Kind {
	case Int:
		return int(value.I), true
	case Bool:
		if value.B {
			return 1, true
		}
		return 0, true
	case Float:
		return int(value.F), true
	case String:
		var n int
		_, err := fmt.Sscanf(value.S, "%d", &n)
		return n, err == nil
	default:
		return 0, false
	}
}

func (i *Interpreter) stringify(value Value) (string, error) {
	if value.Kind == ObjectKind && value.O != nil {
		if i.isExceptionObject(value.O) {
			return i.exceptionString(value.O), nil
		}
		if i.hasPublicMethod(value.O, "__toString") {
			result, err := i.callFunctionValues(value.O.Methods[methodNameKey("__toString")], nil, value.O, value.O.ClassName)
			if err != nil {
				return "", err
			}
			if result.Kind != String {
				return "", fmt.Errorf("__toString must return string")
			}
			return result.String(), nil
		}
	}
	return value.String(), nil
}

func (i *Interpreter) interpolateString(text string) (string, error) {
	var out strings.Builder
	for idx := 0; idx < len(text); {
		if text[idx] == '\\' && idx+1 < len(text) && text[idx+1] == '$' {
			out.WriteByte('$')
			idx += 2
			continue
		}
		if text[idx] == '{' && idx+1 < len(text) && text[idx+1] == '$' {
			end := interpolationBraceEnd(text, idx+1)
			if end < 0 {
				out.WriteByte(text[idx])
				idx++
				continue
			}
			value, err := i.interpolationExprValue(text[idx+1 : end])
			if err != nil {
				return "", err
			}
			rendered, err := i.stringify(value)
			if err != nil {
				return "", err
			}
			out.WriteString(rendered)
			idx = end + 1
			continue
		}
		if text[idx] == '$' && idx+1 < len(text) && text[idx+1] == '{' {
			end := interpolationBraceEnd(text, idx+1)
			if end < 0 {
				out.WriteByte(text[idx])
				idx++
				continue
			}
			value, err := i.dollarBraceInterpolationValue(text[idx+2 : end])
			if err != nil {
				return "", err
			}
			rendered, err := i.stringify(value)
			if err != nil {
				return "", err
			}
			out.WriteString(rendered)
			idx = end + 1
			continue
		}
		if text[idx] != '$' || idx+1 >= len(text) || !isVarNameStart(text[idx+1]) {
			out.WriteByte(text[idx])
			idx++
			continue
		}
		value, end, err := i.interpolationVariableValue(text, idx+1)
		if err != nil {
			return "", err
		}
		rendered, err := i.stringify(value)
		if err != nil {
			return "", err
		}
		out.WriteString(rendered)
		idx = end
	}
	return out.String(), nil
}

func (i *Interpreter) dollarBraceInterpolationValue(expr string) (Value, error) {
	if expr == "" {
		return NullValue(), nil
	}
	if expr[0] == '$' {
		nameValue, err := i.interpolationExprValue(expr)
		if err != nil {
			return NullValue(), err
		}
		value, _ := i.getVar(nameValue.String())
		return value, nil
	}
	value, end, err := i.interpolationVariableValue("$"+expr, 1)
	if err != nil {
		return NullValue(), err
	}
	if end != len(expr)+1 {
		value, _ = i.getVar(expr)
	}
	return value, nil
}

func (i *Interpreter) interpolationExprValue(expr string) (Value, error) {
	if strings.HasPrefix(expr, "$$") {
		name, end := interpolationName(expr, 2)
		if name == "" || end != len(expr) {
			return NullValue(), nil
		}
		nameValue, _ := i.getVar(name)
		value, _ := i.getVar(nameValue.String())
		return value, nil
	}
	if strings.HasPrefix(expr, "$") {
		value, _, err := i.interpolationVariableValue(expr, 1)
		return value, err
	}
	value, _, err := i.interpolationVariableValue("$"+expr, 1)
	return value, err
}

func (i *Interpreter) interpolationVariableValue(text string, start int) (Value, int, error) {
	name, idx := interpolationName(text, start)
	if name == "" {
		return NullValue(), start, nil
	}
	value, _ := i.getVar(name)
	for idx < len(text) {
		switch {
		case text[idx] == '[':
			key, next, ok, err := i.interpolationArrayKey(text, idx+1)
			if err != nil || !ok {
				return value, idx, err
			}
			value = interpolationIndexValue(value, key)
			idx = next
		case strings.HasPrefix(text[idx:], "->"):
			prop, next := interpolationName(text, idx+2)
			if prop == "" {
				return value, idx, nil
			}
			var err error
			value, err = i.interpolationPropertyValue(value, prop)
			if err != nil {
				return NullValue(), idx, err
			}
			idx = next
		default:
			return value, idx, nil
		}
	}
	return value, idx, nil
}

func (i *Interpreter) interpolationArrayKey(text string, start int) (ArrayKey, int, bool, error) {
	if start >= len(text) {
		return StringKey(""), start, false, nil
	}
	if text[start] == '$' {
		value, end, err := i.interpolationVariableValue(text, start+1)
		if err != nil {
			return StringKey(""), end, false, err
		}
		if end >= len(text) || text[end] != ']' {
			return StringKey(""), start, false, nil
		}
		return KeyFromValue(value), end + 1, true, nil
	}
	if text[start] == '\'' || text[start] == '"' {
		key, end, ok := interpolationQuotedKey(text, start)
		if !ok || end >= len(text) || text[end] != ']' {
			return StringKey(""), start, false, nil
		}
		return StringKey(key), end + 1, true, nil
	}
	end := start
	for end < len(text) && text[end] != ']' {
		end++
	}
	if end >= len(text) {
		return StringKey(""), start, false, nil
	}
	raw := strings.TrimSpace(text[start:end])
	if raw == "" {
		return StringKey(""), start, false, nil
	}
	if n, err := strconv.ParseInt(raw, 10, 64); err == nil {
		return IntKey(n), end + 1, true, nil
	}
	return StringKey(raw), end + 1, true, nil
}

func (i *Interpreter) interpolationPropertyValue(value Value, property string) (Value, error) {
	if value.Kind != ObjectKind || value.O == nil {
		return NullValue(), nil
	}
	_, propValue, _, ok, inaccessible := i.objectProperty(value.O, property)
	if inaccessible {
		if i.hasPublicMethod(value.O, "__get") {
			return i.callMagicGet(value.O, property)
		}
		return NullValue(), nil
	}
	if ok {
		return propValue, nil
	}
	if i.shouldUseGet(value.O) {
		return i.callMagicGet(value.O, property)
	}
	return NullValue(), nil
}

func interpolationIndexValue(value Value, key ArrayKey) Value {
	switch value.Kind {
	case ArrayKind:
		if value.A == nil {
			return NullValue()
		}
		item, _ := value.A.Get(key)
		return item
	case String:
		offset, ok := indexOffset(key.Value())
		if !ok || offset < 0 || offset >= len(value.S) {
			return NullValue()
		}
		return StringValue(value.S[offset : offset+1])
	default:
		return NullValue()
	}
}

func interpolationName(text string, start int) (string, int) {
	if start >= len(text) || !isVarNameStart(text[start]) {
		return "", start
	}
	end := start + 1
	for end < len(text) && isVarNameChar(text[end]) {
		end++
	}
	return text[start:end], end
}

func interpolationQuotedKey(text string, start int) (string, int, bool) {
	quote := text[start]
	var out strings.Builder
	idx := start + 1
	for idx < len(text) {
		if text[idx] == quote {
			return out.String(), idx + 1, true
		}
		if text[idx] == '\\' && idx+1 < len(text) {
			idx++
		}
		out.WriteByte(text[idx])
		idx++
	}
	return "", start, false
}

func interpolationBraceEnd(text string, start int) int {
	quote := byte(0)
	for idx := start + 1; idx < len(text); idx++ {
		if quote != 0 {
			if text[idx] == '\\' && idx+1 < len(text) {
				idx++
				continue
			}
			if text[idx] == quote {
				quote = 0
			}
			continue
		}
		switch text[idx] {
		case '\'', '"':
			quote = text[idx]
		case '}':
			return idx
		}
	}
	return -1
}

func isVarNameStart(ch byte) bool {
	return ('a' <= ch && ch <= 'z') || ('A' <= ch && ch <= 'Z') || ch == '_'
}

func isVarNameChar(ch byte) bool {
	return isVarNameStart(ch) || ('0' <= ch && ch <= '9')
}

func (i *Interpreter) castValue(typ string, value Value) (Value, error) {
	switch strings.ToLower(typ) {
	case "int", "integer":
		return IntValue(castInt(value)), nil
	case "bool", "boolean":
		return BoolValue(value.Truthy()), nil
	case "float", "double", "real":
		return FloatValue(castFloat(value)), nil
	case "string":
		text, err := i.stringify(value)
		if err != nil {
			return NullValue(), err
		}
		return StringValue(text), nil
	case "array":
		if value.Kind == ArrayKind {
			return value, nil
		}
		arr := NewArray()
		if value.Kind != Null {
			arr.Append(value)
		}
		return ArrayValue(arr), nil
	case "object":
		if value.Kind == ObjectKind {
			return value, nil
		}
		props := map[string]Value{}
		meta := map[string]PropertyMeta{}
		if value.Kind == ArrayKind && value.A != nil {
			for _, entry := range value.A.Entries {
				name := arrayKeyPropertyName(entry.Key)
				props[name] = entry.RuntimeValue()
				meta[name] = PropertyMeta{Owner: "stdClass", Name: name, Visibility: "public"}
			}
		} else if value.Kind != Null {
			props["scalar"] = value
			meta["scalar"] = PropertyMeta{Owner: "stdClass", Name: "scalar", Visibility: "public"}
		}
		return ObjectValue(NewObject("stdClass", map[string]*ast.FunctionDeclStmt{}, props, meta)), nil
	case "unset":
		return NullValue(), nil
	default:
		return value, nil
	}
}

func arrayKeyPropertyName(key ArrayKey) string {
	if key.IsInt {
		return strconv.FormatInt(key.Int, 10)
	}
	return key.Str
}

func castInt(value Value) int64 {
	switch value.Kind {
	case Null:
		return 0
	case Bool:
		if value.B {
			return 1
		}
		return 0
	case Int:
		return value.I
	case Float:
		return int64(value.F)
	case String:
		return int64(parseNumericPrefix(value.S))
	default:
		return 1
	}
}

func castFloat(value Value) float64 {
	switch value.Kind {
	case Null:
		return 0
	case Bool:
		if value.B {
			return 1
		}
		return 0
	case Int:
		return float64(value.I)
	case Float:
		return value.F
	case String:
		return parseNumericPrefix(value.S)
	default:
		return 1
	}
}

func parseNumericPrefix(text string) float64 {
	text = strings.TrimLeft(text, " \t\r\n")
	if text == "" {
		return 0
	}
	end := 0
	if text[end] == '+' || text[end] == '-' {
		end++
	}
	digits := 0
	for end < len(text) && text[end] >= '0' && text[end] <= '9' {
		end++
		digits++
	}
	if end < len(text) && text[end] == '.' {
		end++
		for end < len(text) && text[end] >= '0' && text[end] <= '9' {
			end++
			digits++
		}
	}
	if digits == 0 {
		return 0
	}
	if end < len(text) && (text[end] == 'e' || text[end] == 'E') {
		expEnd := end + 1
		if expEnd < len(text) && (text[expEnd] == '+' || text[expEnd] == '-') {
			expEnd++
		}
		expDigits := 0
		for expEnd < len(text) && text[expEnd] >= '0' && text[expEnd] <= '9' {
			expEnd++
			expDigits++
		}
		if expDigits > 0 {
			end = expEnd
		}
	}
	parsed, err := strconv.ParseFloat(text[:end], 64)
	if err != nil {
		return 0
	}
	return parsed
}

func (i *Interpreter) hasPublicMethod(obj *Object, name string) bool {
	method, ok := obj.Methods[methodNameKey(name)]
	if !ok {
		return false
	}
	return method.Visibility == "" || method.Visibility == "public"
}

func (i *Interpreter) hasPublicStaticMethod(methods map[string]*ast.FunctionDeclStmt, name string) bool {
	method, ok := methods[methodNameKey(name)]
	if !ok {
		return false
	}
	return method.Static && (method.Visibility == "" || method.Visibility == "public")
}

func (i *Interpreter) shouldUseGet(obj *Object) bool {
	return i.hasPublicMethod(obj, "__get") && i.currentClass() != obj.ClassName
}

func (i *Interpreter) shouldUseSet(obj *Object) bool {
	return i.hasPublicMethod(obj, "__set") && i.currentClass() != obj.ClassName
}

func (i *Interpreter) shouldUseIsset(obj *Object) bool {
	return i.hasPublicMethod(obj, "__isset") && i.currentClass() != obj.ClassName
}

func (i *Interpreter) shouldUseUnset(obj *Object) bool {
	return i.hasPublicMethod(obj, "__unset") && i.currentClass() != obj.ClassName
}

func (i *Interpreter) objectProperty(obj *Object, property string) (string, Value, PropertyMeta, bool, bool) {
	return i.objectPropertyInScope(obj, property, i.currentClass())
}

func (i *Interpreter) objectPropertyInScope(obj *Object, property string, scope string) (string, Value, PropertyMeta, bool, bool) {
	if obj == nil {
		return "", NullValue(), PropertyMeta{}, false, false
	}
	if scope != "" {
		key := propertyStorageKey(scope, "private", property)
		if value, ok := obj.Props[key]; ok {
			info := obj.PropMeta[key]
			if info.Name == "" {
				info.Name = property
			}
			return key, value, info, true, !i.propertyAccessibleInScope(info, scope)
		}
	}
	if value, ok := obj.Props[property]; ok {
		info := obj.PropMeta[property]
		if info.Name == "" {
			info.Name = property
		}
		if !i.propertyAccessibleInScope(info, scope) {
			return property, value, info, false, true
		}
		return property, value, info, true, false
	}
	lineage := i.objectClassLineage(obj)
	for idx := len(lineage) - 1; idx >= 0; idx-- {
		class := lineage[idx]
		key := propertyStorageKey(class.Name, "private", property)
		value, ok := obj.Props[key]
		if !ok {
			continue
		}
		info := obj.PropMeta[key]
		if info.Name == "" {
			info.Name = property
		}
		if !i.propertyAccessibleInScope(info, scope) {
			return key, value, info, false, true
		}
		return key, value, info, true, false
	}
	return "", NullValue(), PropertyMeta{}, false, false
}

func (i *Interpreter) visibleObjectProperty(obj *Object, property string) (Value, bool, bool) {
	_, value, _, ok, inaccessible := i.objectProperty(obj, property)
	return value, ok, inaccessible
}

func (i *Interpreter) objectClassLineage(obj *Object) []*ast.ClassDeclStmt {
	if obj == nil {
		return nil
	}
	class, ok := i.classByName(obj.ClassName)
	if !ok {
		return nil
	}
	return i.classLineage(class)
}

func (i *Interpreter) objectMethod(obj *Object, methodName string) (*ast.FunctionDeclStmt, bool) {
	if obj == nil {
		return nil, false
	}
	if current := i.currentClass(); current != "" {
		if method, ok := obj.Methods[privateMemberKey(current, methodName)]; ok {
			return method, true
		}
	}
	if method, ok := obj.Methods[methodNameKey(methodName)]; ok {
		return method, true
	}
	lineage := i.objectClassLineage(obj)
	for idx := len(lineage) - 1; idx >= 0; idx-- {
		if method, ok := obj.Methods[privateMemberKey(lineage[idx].Name, methodName)]; ok {
			return method, true
		}
	}
	return nil, false
}

func (i *Interpreter) staticMethod(methods map[string]*ast.FunctionDeclStmt, className, methodName string) (*ast.FunctionDeclStmt, bool) {
	if current := i.currentClass(); current != "" {
		if method, ok := methods[privateMemberKey(current, methodName)]; ok {
			return method, true
		}
	}
	if method, ok := methods[methodNameKey(methodName)]; ok {
		return method, true
	}
	if class, ok := i.classByName(className); ok {
		lineage := i.classLineage(class)
		for idx := len(lineage) - 1; idx >= 0; idx-- {
			if method, ok := methods[privateMemberKey(lineage[idx].Name, methodName)]; ok {
				return method, true
			}
		}
	}
	return nil, false
}

func (i *Interpreter) callMagicGet(obj *Object, property string) (Value, error) {
	return i.callFunctionValues(obj.Methods[methodNameKey("__get")], []Value{StringValue(property)}, obj, obj.ClassName)
}

func (i *Interpreter) callMagicSet(obj *Object, property string, value Value) (Value, error) {
	return i.callFunctionValues(obj.Methods[methodNameKey("__set")], []Value{StringValue(property), value}, obj, obj.ClassName)
}

func (i *Interpreter) callMagicIsset(obj *Object, property string) (Value, error) {
	return i.callFunctionValues(obj.Methods[methodNameKey("__isset")], []Value{StringValue(property)}, obj, obj.ClassName)
}

func (i *Interpreter) callMagicUnset(obj *Object, property string) (Value, error) {
	return i.callFunctionValues(obj.Methods[methodNameKey("__unset")], []Value{StringValue(property)}, obj, obj.ClassName)
}

func (i *Interpreter) callMagicCall(obj *Object, methodName string, args []ast.Expr) (Value, error) {
	values := NewArray()
	for _, arg := range args {
		value, err := i.eval(arg)
		if err != nil {
			return NullValue(), err
		}
		values.Append(value)
	}
	return i.callFunctionValues(obj.Methods[methodNameKey("__call")], []Value{StringValue(methodName), ArrayValue(values)}, obj, obj.ClassName)
}

func (i *Interpreter) callMagicCallValues(obj *Object, methodName string, args []Value) (Value, error) {
	values := NewArray()
	for _, arg := range args {
		values.Append(arg)
	}
	return i.callFunctionValues(obj.Methods[methodNameKey("__call")], []Value{StringValue(methodName), ArrayValue(values)}, obj, obj.ClassName)
}

func (i *Interpreter) callMagicCallStatic(className string, method *ast.FunctionDeclStmt, methodName string, args []ast.Expr) (Value, error) {
	values := NewArray()
	for _, arg := range args {
		value, err := i.eval(arg)
		if err != nil {
			return NullValue(), err
		}
		values.Append(value)
	}
	return i.callFunctionValues(method, []Value{StringValue(methodName), ArrayValue(values)}, nil, className)
}

func (i *Interpreter) callMagicCallStaticValues(className string, method *ast.FunctionDeclStmt, methodName string, args []Value) (Value, error) {
	values := NewArray()
	for _, arg := range args {
		values.Append(arg)
	}
	return i.callFunctionValues(method, []Value{StringValue(methodName), ArrayValue(values)}, nil, className)
}

func (i *Interpreter) newException(n *ast.NewExpr) (Value, error) {
	exceptionClass, _ := i.classByName("Exception")
	obj := NewObject("Exception", i.methodsForClass(exceptionClass), nil, nil)
	if err := i.initExceptionObject(obj, n.Args, i.currentFile(), n.Line); err != nil {
		return NullValue(), err
	}
	return ObjectValue(obj), nil
}

func (i *Interpreter) initExceptionObject(obj *Object, args []ast.Expr, file string, line int) error {
	if len(args) > 3 {
		return fmt.Errorf("Exception expects at most 3 arguments, got %d", len(args))
	}
	message := StringValue("")
	code := IntValue(0)
	previous := NullValue()
	if len(args) >= 1 {
		value, err := i.eval(args[0])
		if err != nil {
			return err
		}
		message = StringValue(value.String())
	}
	if len(args) >= 2 {
		value, err := i.eval(args[1])
		if err != nil {
			return err
		}
		code = IntValue(valueToInt(value))
	}
	if len(args) >= 3 {
		value, err := i.eval(args[2])
		if err != nil {
			return err
		}
		if value.Kind != Null {
			if value.Kind != ObjectKind || value.O == nil || !i.isExceptionObject(value.O) {
				return fmt.Errorf("Exception previous must be an Exception or null")
			}
			previous = value
		}
	}
	setExceptionProperty(obj, "message", message)
	setExceptionProperty(obj, "code", code)
	setExceptionProperty(obj, "file", StringValue(file))
	setExceptionProperty(obj, "line", IntValue(int64(line)))
	setExceptionProperty(obj, "previous", previous)
	obj.ExceptionTrace = i.captureExceptionTrace()
	return nil
}

func (i *Interpreter) callExceptionMethod(obj *Object, methodName string, args []ast.Expr) (Value, bool, error) {
	if len(args) != 0 {
		switch strings.ToLower(methodName) {
		case "getmessage", "getcode", "getfile", "getline", "getprevious", "gettrace", "gettraceasstring", "__tostring":
			return NullValue(), true, fmt.Errorf("Exception::%s expects 0 arguments, got %d", methodName, len(args))
		}
	}
	switch strings.ToLower(methodName) {
	case "getmessage":
		return exceptionProperty(obj, "message", StringValue("")), true, nil
	case "getcode":
		return exceptionProperty(obj, "code", IntValue(0)), true, nil
	case "getfile":
		return exceptionProperty(obj, "file", StringValue("")), true, nil
	case "getline":
		return exceptionProperty(obj, "line", IntValue(0)), true, nil
	case "getprevious":
		return exceptionProperty(obj, "previous", NullValue()), true, nil
	case "gettrace":
		return ArrayValue(exceptionTraceArray(obj.ExceptionTrace)), true, nil
	case "gettraceasstring":
		return StringValue(exceptionTraceString(obj.ExceptionTrace)), true, nil
	case "__tostring":
		return StringValue(i.exceptionString(obj)), true, nil
	default:
		return NullValue(), false, nil
	}
}

func (i *Interpreter) captureExceptionTrace() []ExceptionTraceFrame {
	trace := make([]ExceptionTraceFrame, 0, len(i.frames))
	for idx := len(i.frames) - 1; idx >= 0; idx-- {
		frame := i.frames[idx]
		if frame.function == "" {
			continue
		}
		traceFrame := ExceptionTraceFrame{Function: frame.function}
		if frame.className != "" {
			traceFrame.Class = frame.className
			traceFrame.Type = "::"
		}
		trace = append(trace, traceFrame)
	}
	return trace
}

func exceptionTraceArray(trace []ExceptionTraceFrame) *Array {
	result := NewArray()
	for _, frame := range trace {
		entry := NewArray()
		entry.Set(StringKey("function"), StringValue(frame.Function))
		if frame.Class != "" {
			entry.Set(StringKey("class"), StringValue(frame.Class))
			entry.Set(StringKey("type"), StringValue(frame.Type))
		}
		result.Append(ArrayValue(entry))
	}
	return result
}

func exceptionTraceString(trace []ExceptionTraceFrame) string {
	lines := make([]string, 0, len(trace))
	for idx, frame := range trace {
		name := frame.Function
		if frame.Class != "" {
			name = frame.Class + frame.Type + frame.Function
		}
		lines = append(lines, fmt.Sprintf("#%d %s()", idx, name))
	}
	return strings.Join(lines, "\n")
}

func exceptionProperty(obj *Object, name string, fallback Value) Value {
	key := propertyStorageKey("Exception", "private", name)
	if value, ok := obj.Props[key]; ok {
		return value
	}
	if value, ok := obj.Props[name]; ok {
		return value
	}
	return fallback
}

func setExceptionProperty(obj *Object, name string, value Value) {
	key := propertyStorageKey("Exception", "private", name)
	obj.Props[key] = value
	obj.PropMeta[key] = PropertyMeta{Owner: "Exception", Name: name, Visibility: "private"}
}

func valueToInt(value Value) int64 {
	if number, ok := value.number(); ok {
		return int64(number)
	}
	return 0
}

func (i *Interpreter) exceptionString(obj *Object) string {
	message := exceptionProperty(obj, "message", StringValue("")).String()
	file := exceptionProperty(obj, "file", StringValue("")).String()
	line := exceptionProperty(obj, "line", IntValue(0)).String()
	if file == "" {
		return obj.ClassName + ": " + message
	}
	return obj.ClassName + ": " + message + " in " + file + ":" + line
}

func (i *Interpreter) isExceptionObject(obj *Object) bool {
	if obj == nil {
		return false
	}
	return i.isSameOrSubclass(obj.ClassName, "Exception")
}

func (i *Interpreter) exceptionMatches(value Value, catchClass string) bool {
	return i.valueInstanceOf(value, catchClass)
}

func (i *Interpreter) defineFunction(fn *ast.FunctionDeclStmt) error {
	key := functionNameKey(fn.Name)
	if _, exists := i.functions[key]; exists {
		return fmt.Errorf("cannot redeclare function %s", fn.Name)
	}
	if err := validateFunctionCompatibility(fn, "function "+fn.Name); err != nil {
		return err
	}
	i.functions[key] = fn
	return nil
}

func functionNameKey(name string) string {
	return strings.ToLower(name)
}

func methodNameKey(name string) string {
	return strings.ToLower(name)
}

func methodStorageKey(method *ast.FunctionDeclStmt) string {
	if method != nil && method.Visibility == "private" && method.Owner != "" {
		return privateMemberKey(method.Owner, method.Name)
	}
	return methodNameKey(method.Name)
}

func methodMapHasName(methods map[string]*ast.FunctionDeclStmt, name string) bool {
	key := methodNameKey(name)
	for _, method := range methods {
		if methodNameKey(method.Name) == key {
			return true
		}
	}
	return false
}

func privateMemberKey(owner, name string) string {
	return "\x00" + strings.ToLower(owner) + "\x00" + strings.ToLower(name)
}

func propertyStorageKey(owner, visibility, name string) string {
	if visibility == "private" && owner != "" {
		return privateMemberKey(owner, name)
	}
	return name
}

func propertyNameForKey(key string, info PropertyMeta) string {
	if info.Name != "" {
		return info.Name
	}
	parts := strings.Split(key, "\x00")
	if len(parts) == 3 && parts[0] == "" {
		return parts[2]
	}
	return key
}

func classNameKey(name string) string {
	return strings.ToLower(strings.TrimPrefix(name, "\\"))
}

func (i *Interpreter) classByName(name string) (*ast.ClassDeclStmt, bool) {
	key := classNameKey(name)
	if class, ok := i.classes[key]; ok {
		return class, true
	}
	if alias, ok := i.classAliases[key]; ok {
		class, ok := i.classes[classNameKey(alias.Target)]
		return class, ok
	}
	return nil, false
}

func (i *Interpreter) knownClassName(name string) (string, bool) {
	if class, ok := i.classByName(name); ok {
		return class.Name, true
	}
	name, ok := i.declaredClassNames[classNameKey(name)]
	return name, ok
}

func (i *Interpreter) interfaceByName(name string) (*ast.InterfaceDeclStmt, bool) {
	iface, ok := i.interfaces[classNameKey(name)]
	return iface, ok
}

func (i *Interpreter) classLikeName(name string) (string, bool) {
	if class, ok := i.classByName(name); ok {
		return class.Name, true
	}
	if iface, ok := i.interfaceByName(name); ok {
		return iface.Name, true
	}
	return "", false
}

func (i *Interpreter) autoloadClass(name string) error {
	if _, ok := i.classLikeName(name); ok {
		return nil
	}
	fn, ok := i.functions[functionNameKey("__autoload")]
	if !ok {
		return nil
	}
	key := classNameKey(name)
	if i.autoloading[key] {
		return nil
	}
	i.autoloading[key] = true
	defer delete(i.autoloading, key)
	_, err := i.callFunctionValues(fn, []Value{StringValue(name)}, nil, "")
	return err
}

func namespaceBaseName(name string) string {
	name = strings.TrimPrefix(name, "\\")
	if idx := strings.LastIndex(name, "\\"); idx >= 0 {
		return name[idx+1:]
	}
	return name
}

func validateFunctionCompatibility(fn *ast.FunctionDeclStmt, label string) error {
	seenOptional := ""
	for _, param := range fn.Params {
		if param.Default != nil {
			if err := validateDefaultExpression(param.Default); err != nil {
				return fmt.Errorf("%s parameter $%s default must be a constant expression", label, param.Name)
			}
			if seenOptional == "" {
				seenOptional = param.Name
			}
			continue
		}
		if seenOptional != "" {
			return fmt.Errorf("%s required parameter $%s cannot follow optional parameter $%s", label, param.Name, seenOptional)
		}
	}
	return nil
}

func validateDefaultExpression(expr ast.Expr) error {
	switch n := expr.(type) {
	case *ast.IntLit, *ast.FloatLit, *ast.StringLit, *ast.BoolLit, *ast.NullLit, *ast.BareConstExpr, *ast.MagicConstExpr, *ast.ClassConstFetchExpr:
		return nil
	case *ast.PrefixExpr:
		if n.Op != "+" && n.Op != "-" {
			return fmt.Errorf("invalid default expression")
		}
		switch n.Right.(type) {
		case *ast.IntLit, *ast.FloatLit:
			return nil
		default:
			return fmt.Errorf("invalid default expression")
		}
	case *ast.ArrayLit:
		for _, elem := range n.Elements {
			if elem.Key != nil {
				if err := validateDefaultExpression(elem.Key); err != nil {
					return err
				}
			}
			if err := validateDefaultExpression(elem.Value); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("invalid default expression")
	}
}

func (i *Interpreter) defineClass(class *ast.ClassDeclStmt) error {
	if _, exists := i.classByName(class.Name); exists {
		return fmt.Errorf("cannot redeclare class %s", class.Name)
	}
	if _, exists := i.interfaceByName(class.Name); exists {
		return fmt.Errorf("cannot redeclare class %s", class.Name)
	}
	if class.Final && class.Abstract {
		return fmt.Errorf("class %s cannot be both final and abstract", class.Name)
	}
	hasModernConstructor := classDeclaresMethod(class, "__construct")
	seenMethods := map[string]string{}
	for _, method := range class.Methods {
		method.Owner = class.Name
		methodKey := methodNameKey(method.Name)
		if previous, exists := seenMethods[methodKey]; exists {
			return fmt.Errorf("cannot redeclare method %s::%s", class.Name, previous)
		}
		seenMethods[methodKey] = method.Name
		if method.Visibility == "" {
			method.Visibility = "public"
		}
		if err := validateMagicMethodDeclaration(class.Name, method); err != nil {
			return err
		}
		if !hasModernConstructor && isOldStyleConstructor(class.Name, method.Name) && method.Static {
			return fmt.Errorf("old-style constructor %s::%s must not be static", class.Name, method.Name)
		}
		if err := validateFunctionCompatibility(method, "method "+class.Name+"::"+method.Name); err != nil {
			return err
		}
		if method.Final && method.Abstract {
			return fmt.Errorf("method %s::%s cannot be both final and abstract", class.Name, method.Name)
		}
		if method.Abstract && method.Visibility == "private" {
			return fmt.Errorf("abstract method %s::%s cannot be private", class.Name, method.Name)
		}
	}
	seenProperties := map[string]bool{}
	for _, prop := range class.Properties {
		prop.Owner = class.Name
		if seenProperties[prop.Name] {
			return fmt.Errorf("cannot redeclare property %s::$%s", class.Name, prop.Name)
		}
		seenProperties[prop.Name] = true
		if prop.Visibility == "" {
			prop.Visibility = "public"
		}
		if prop.Abstract {
			return fmt.Errorf("property %s::$%s cannot be abstract", class.Name, prop.Name)
		}
		if prop.Final {
			return fmt.Errorf("property %s::$%s cannot be final", class.Name, prop.Name)
		}
		if prop.Default != nil {
			if err := validateDefaultExpression(prop.Default); err != nil {
				return fmt.Errorf("property %s::$%s default must be a constant expression", class.Name, prop.Name)
			}
		}
	}
	if err := validateClassConstants(class.Name, class.Constants, "class constant"); err != nil {
		return err
	}
	if class.Parent != "" {
		if err := i.autoloadClass(class.Parent); err != nil {
			return err
		}
		parent, ok := i.classByName(class.Parent)
		if !ok {
			if iface, interfaceOK := i.interfaceByName(class.Parent); interfaceOK {
				return fmt.Errorf("class %s cannot extend interface %s", class.Name, iface.Name)
			}
			return fmt.Errorf("undefined parent class %s", class.Parent)
		}
		class.Parent = parent.Name
		if parent.Final {
			return fmt.Errorf("class %s cannot extend final class %s", class.Name, parent.Name)
		}
		parentMethods := i.methodsForClass(parent)
		for _, method := range class.Methods {
			if parentMethod, ok := parentMethods[methodNameKey(method.Name)]; ok {
				if err := i.validateMethodOverrideCompatibility(class.Name, method, parentMethod); err != nil {
					return err
				}
			}
		}
		parentProperties := i.inheritedPropertiesForClass(parent)
		for _, prop := range class.Properties {
			if parentProp, ok := parentProperties[prop.Name]; ok {
				if err := validatePropertyOverrideCompatibility(class.Name, prop, parentProp); err != nil {
					return err
				}
			}
		}
	}
	seenInterfaces := map[string]string{}
	for idx, ifaceName := range class.Implements {
		if err := i.autoloadClass(ifaceName); err != nil {
			return err
		}
		iface, ok := i.interfaceByName(ifaceName)
		if !ok {
			if implementedClass, classOK := i.knownClassName(ifaceName); classOK {
				return fmt.Errorf("class %s cannot implement class %s", class.Name, implementedClass)
			}
			return fmt.Errorf("undefined interface %s", ifaceName)
		}
		if previous, exists := seenInterfaces[classNameKey(iface.Name)]; exists {
			return fmt.Errorf("class %s cannot implement interface %s more than once", class.Name, previous)
		}
		if class.Parent != "" && i.classImplementsInterface(class.Parent, iface.Name) {
			return fmt.Errorf("class %s cannot implement already implemented interface %s", class.Name, iface.Name)
		}
		seenInterfaces[classNameKey(iface.Name)] = iface.Name
		class.Implements[idx] = iface.Name
	}
	i.classes[classNameKey(class.Name)] = class
	if _, ok := i.constants[class.Name]; !ok {
		i.constants[class.Name] = map[string]Value{}
	}
	for _, constant := range class.Constants {
		constant.Owner = class.Name
		value, err := i.eval(constant.Value)
		if err != nil {
			return err
		}
		i.constants[class.Name][constant.Name] = value
	}
	if _, ok := i.statics[class.Name]; !ok {
		i.statics[class.Name] = map[string]Value{}
	}
	if _, ok := i.staticMeta[class.Name]; !ok {
		i.staticMeta[class.Name] = map[string]PropertyMeta{}
	}
	for _, prop := range class.Properties {
		if !prop.Static {
			continue
		}
		value := NullValue()
		if prop.Default != nil {
			var err error
			value, err = i.eval(prop.Default)
			if err != nil {
				return err
			}
		}
		key := propertyStorageKey(prop.Owner, prop.Visibility, prop.Name)
		i.statics[class.Name][key] = value
		i.staticMeta[class.Name][key] = PropertyMeta{Owner: prop.Owner, Name: prop.Name, Visibility: prop.Visibility, Static: true}
	}
	if class.Abstract {
		if err := i.checkInterfaceImplementations(class, false); err != nil {
			return err
		}
	} else {
		for _, method := range i.methodsForClass(class) {
			if method.Abstract {
				return fmt.Errorf("class %s contains abstract methods and must be abstract", class.Name)
			}
		}
		if err := i.checkInterfaceImplementations(class, true); err != nil {
			return err
		}
	}
	return nil
}

func validateMagicMethodDeclaration(className string, method *ast.FunctionDeclStmt) error {
	name := magicMethodName(method.Name)
	switch strings.ToLower(method.Name) {
	case "__get", "__set", "__isset", "__unset":
		if method.Visibility != "public" {
			return fmt.Errorf("magic method %s::%s must be public", className, name)
		}
		if method.Static {
			return fmt.Errorf("magic method %s::%s must not be static", className, name)
		}
	case "__sleep", "__wakeup", "__tostring":
		if method.Visibility != "public" {
			return fmt.Errorf("magic method %s::%s must be public", className, name)
		}
		if method.Static {
			return fmt.Errorf("magic method %s::%s must not be static", className, name)
		}
	case "__clone":
		if method.Static {
			return fmt.Errorf("magic method %s::__clone must not be static", className)
		}
	case "__construct":
		if method.Static {
			return fmt.Errorf("magic method %s::__construct must not be static", className)
		}
	case "__destruct":
		if method.Static {
			return fmt.Errorf("magic method %s::__destruct must not be static", className)
		}
	case "__invoke":
		if method.Visibility != "public" {
			return fmt.Errorf("magic method %s::__invoke must be public", className)
		}
		if method.Static {
			return fmt.Errorf("magic method %s::__invoke must not be static", className)
		}
	case "__call":
		if method.Visibility != "public" {
			return fmt.Errorf("magic method %s::__call must be public", className)
		}
		if method.Static {
			return fmt.Errorf("magic method %s::__call must not be static", className)
		}
	case "__callstatic":
		if method.Visibility != "public" {
			return fmt.Errorf("magic method %s::__callStatic must be public", className)
		}
		if !method.Static {
			return fmt.Errorf("magic method %s::__callStatic must be static", className)
		}
	}
	if expected, ok := magicMethodParamCount(name); ok && len(method.Params) != expected {
		return fmt.Errorf("magic method %s::%s must take exactly %d arguments", className, name, expected)
	}
	return nil
}

func magicMethodParamCount(name string) (int, bool) {
	switch name {
	case "__get", "__isset", "__unset":
		return 1, true
	case "__set", "__call", "__callStatic":
		return 2, true
	case "__sleep", "__wakeup", "__toString":
		return 0, true
	case "__clone":
		return 0, true
	case "__destruct":
		return 0, true
	default:
		return 0, false
	}
}

func magicMethodName(name string) string {
	switch strings.ToLower(name) {
	case "__get":
		return "__get"
	case "__set":
		return "__set"
	case "__isset":
		return "__isset"
	case "__unset":
		return "__unset"
	case "__call":
		return "__call"
	case "__callstatic":
		return "__callStatic"
	case "__sleep":
		return "__sleep"
	case "__wakeup":
		return "__wakeup"
	case "__tostring":
		return "__toString"
	case "__clone":
		return "__clone"
	case "__construct":
		return "__construct"
	case "__destruct":
		return "__destruct"
	case "__invoke":
		return "__invoke"
	default:
		return name
	}
}

func validatePropertyOverrideCompatibility(className string, prop, parentProp *ast.ClassProperty) error {
	if parentProp.Visibility == "private" {
		return nil
	}
	if prop.Static != parentProp.Static {
		return fmt.Errorf("cannot change static modifier of inherited property %s::$%s in %s", parentProp.Owner, parentProp.Name, className)
	}
	if visibilityRank(prop.Visibility) < visibilityRank(parentProp.Visibility) {
		return fmt.Errorf("cannot reduce visibility of inherited property %s::$%s in %s", parentProp.Owner, parentProp.Name, className)
	}
	return nil
}

func (i *Interpreter) validateMethodOverrideCompatibility(className string, method, parentMethod *ast.FunctionDeclStmt) error {
	if parentMethod.Visibility == "private" {
		return nil
	}
	if parentMethod.Final {
		return fmt.Errorf("cannot override final method %s::%s", parentMethod.Owner, parentMethod.Name)
	}
	if method.Static != parentMethod.Static {
		return fmt.Errorf("cannot change static modifier of inherited method %s::%s in %s", parentMethod.Owner, parentMethod.Name, className)
	}
	if visibilityRank(method.Visibility) < visibilityRank(parentMethod.Visibility) {
		return fmt.Errorf("cannot reduce visibility of inherited method %s::%s in %s", parentMethod.Owner, parentMethod.Name, className)
	}
	if isModernConstructor(method) && isModernConstructor(parentMethod) {
		return nil
	}
	if err := i.validateMethodSignatureCompatibility(className, method, parentMethod); err != nil {
		return err
	}
	return nil
}

func isModernConstructor(method *ast.FunctionDeclStmt) bool {
	return method != nil && strings.EqualFold(method.Name, "__construct")
}

func (i *Interpreter) validateMethodSignatureCompatibility(className string, method, prototype *ast.FunctionDeclStmt) error {
	methodLabel := className + "::" + method.Name
	prototypeLabel := prototype.Owner + "::" + prototype.Name
	if len(method.Params) < len(prototype.Params) {
		return fmt.Errorf("method %s must accept at least %d parameters to match %s", methodLabel, len(prototype.Params), prototypeLabel)
	}
	methodRequired := requiredParamCount(method)
	prototypeRequired := requiredParamCount(prototype)
	if methodRequired > prototypeRequired {
		return fmt.Errorf("method %s cannot require %d parameters to match %s requiring %d", methodLabel, methodRequired, prototypeLabel, prototypeRequired)
	}
	for idx, prototypeParam := range prototype.Params {
		if method.Params[idx].ByRef != prototypeParam.ByRef {
			return fmt.Errorf("method %s parameter %d reference modifier must match %s", methodLabel, idx+1, prototypeLabel)
		}
		methodHint := i.normalizedParamTypeHint(method, method.Params[idx])
		prototypeHint := i.normalizedParamTypeHint(prototype, prototypeParam)
		if !strings.EqualFold(methodHint, prototypeHint) {
			return fmt.Errorf("method %s parameter %d type hint must match %s", methodLabel, idx+1, prototypeLabel)
		}
	}
	if method.ReturnsByRef != prototype.ReturnsByRef {
		return fmt.Errorf("method %s return-by-reference modifier must match %s", methodLabel, prototypeLabel)
	}
	return nil
}

func (i *Interpreter) normalizedParamTypeHint(fn *ast.FunctionDeclStmt, param ast.Param) string {
	hint := param.TypeHint
	switch hint {
	case "", "array":
		return hint
	case "self":
		return fn.Owner
	case "parent":
		if class, ok := i.classByName(fn.Owner); ok {
			return class.Parent
		}
		return ""
	default:
		return hint
	}
}

func (i *Interpreter) validateInterfaceMethodCompatibility(className string, method, prototype *ast.FunctionDeclStmt) error {
	if method.Static != prototype.Static {
		return fmt.Errorf("method %s::%s static modifier must match %s::%s", className, method.Name, prototype.Owner, prototype.Name)
	}
	return i.validateMethodSignatureCompatibility(className, method, prototype)
}

func requiredParamCount(fn *ast.FunctionDeclStmt) int {
	required := 0
	for _, param := range fn.Params {
		if param.Default == nil {
			required++
		}
	}
	return required
}

func visibilityRank(visibility string) int {
	switch visibility {
	case "public", "":
		return 3
	case "protected":
		return 2
	case "private":
		return 1
	default:
		return 0
	}
}

func (i *Interpreter) inheritedPropertiesForClass(class *ast.ClassDeclStmt) map[string]*ast.ClassProperty {
	properties := map[string]*ast.ClassProperty{}
	if class.Parent != "" {
		if parent, ok := i.classByName(class.Parent); ok {
			for name, prop := range i.inheritedPropertiesForClass(parent) {
				properties[name] = prop
			}
		}
	}
	for _, prop := range class.Properties {
		if prop.Visibility == "private" {
			continue
		}
		properties[prop.Name] = prop
	}
	return properties
}

func (i *Interpreter) defineInterface(iface *ast.InterfaceDeclStmt) error {
	if _, exists := i.interfaceByName(iface.Name); exists {
		return fmt.Errorf("cannot redeclare interface %s", iface.Name)
	}
	if _, exists := i.classByName(iface.Name); exists {
		return fmt.Errorf("cannot redeclare interface %s", iface.Name)
	}
	if err := validateClassConstants(iface.Name, iface.Constants, "interface constant"); err != nil {
		return err
	}
	parentMethods := map[string]*ast.FunctionDeclStmt{}
	seenParents := map[string]string{}
	for idx, parentName := range iface.Parents {
		if err := i.autoloadClass(parentName); err != nil {
			return err
		}
		parent, ok := i.interfaceByName(parentName)
		if !ok {
			if parentClass, classOK := i.knownClassName(parentName); classOK {
				return fmt.Errorf("interface %s cannot extend class %s", iface.Name, parentClass)
			}
			return fmt.Errorf("undefined parent interface %s", parentName)
		}
		if previous, exists := seenParents[classNameKey(parent.Name)]; exists {
			return fmt.Errorf("interface %s cannot extend interface %s more than once", iface.Name, previous)
		}
		seenParents[classNameKey(parent.Name)] = parent.Name
		iface.Parents[idx] = parent.Name
		methods, err := i.interfaceMethods(parent.Name)
		if err != nil {
			return err
		}
		for name, method := range methods {
			if existing, ok := parentMethods[name]; ok {
				merged, err := i.compatibleInterfacePrototype(iface.Name, existing, method)
				if err != nil {
					return err
				}
				parentMethods[name] = merged
				continue
			}
			parentMethods[name] = method
		}
	}
	seenMethods := map[string]string{}
	for _, method := range iface.Methods {
		method.Owner = iface.Name
		methodKey := methodNameKey(method.Name)
		if previous, exists := seenMethods[methodKey]; exists {
			return fmt.Errorf("cannot redeclare method %s::%s", iface.Name, previous)
		}
		seenMethods[methodKey] = method.Name
		if method.Visibility != "" && method.Visibility != "public" {
			return fmt.Errorf("interface method %s::%s must be public", iface.Name, method.Name)
		}
		if method.Final {
			return fmt.Errorf("interface method %s::%s cannot be final", iface.Name, method.Name)
		}
		method.Visibility = "public"
		method.Abstract = true
		if err := validateFunctionCompatibility(method, "method "+iface.Name+"::"+method.Name); err != nil {
			return err
		}
		if parentMethod, ok := parentMethods[methodNameKey(method.Name)]; ok {
			if err := i.validateInterfaceMethodCompatibility(iface.Name, method, parentMethod); err != nil {
				return err
			}
		}
	}
	i.interfaces[classNameKey(iface.Name)] = iface
	if _, ok := i.constants[iface.Name]; !ok {
		i.constants[iface.Name] = map[string]Value{}
	}
	for _, constant := range iface.Constants {
		constant.Owner = iface.Name
		value, err := i.eval(constant.Value)
		if err != nil {
			return err
		}
		i.constants[iface.Name][constant.Name] = value
	}
	return nil
}

func validateClassConstants(owner string, constants []*ast.ClassConstant, label string) error {
	seenConstants := map[string]bool{}
	for _, constant := range constants {
		constant.Owner = owner
		if seenConstants[constant.Name] {
			return fmt.Errorf("cannot redeclare %s %s::%s", label, owner, constant.Name)
		}
		seenConstants[constant.Name] = true
		if err := validateDefaultExpression(constant.Value); err != nil {
			return fmt.Errorf("%s %s::%s value must be a constant expression", label, owner, constant.Name)
		}
	}
	return nil
}

func (i *Interpreter) compatibleInterfacePrototype(interfaceName string, existing, candidate *ast.FunctionDeclStmt) (*ast.FunctionDeclStmt, error) {
	candidateErr := i.validateInterfaceMethodCompatibility(interfaceName, candidate, existing)
	if candidateErr == nil {
		if i.validateInterfaceMethodCompatibility(interfaceName, existing, candidate) == nil {
			return existing, nil
		}
		return candidate, nil
	}
	if i.validateInterfaceMethodCompatibility(interfaceName, existing, candidate) == nil {
		return existing, nil
	}
	return nil, candidateErr
}

func (i *Interpreter) checkInterfaceImplementations(class *ast.ClassDeclStmt, requireAll bool) error {
	methods := i.methodsForClass(class)
	for _, ifaceName := range i.implementedInterfacesForClass(class) {
		required, err := i.interfaceMethods(ifaceName)
		if err != nil {
			return err
		}
		for name, ifaceMethod := range required {
			method, ok := methods[name]
			if !ok {
				if requireAll {
					return fmt.Errorf("class %s must implement interface method %s::%s", class.Name, ifaceName, ifaceMethod.Name)
				}
				continue
			}
			if requireAll && method.Abstract {
				return fmt.Errorf("class %s must implement interface method %s::%s", class.Name, ifaceName, ifaceMethod.Name)
			}
			if method.Visibility != "" && method.Visibility != "public" {
				return fmt.Errorf("class %s interface method %s must be public", class.Name, name)
			}
			if err := i.validateInterfaceMethodCompatibility(class.Name, method, ifaceMethod); err != nil {
				return err
			}
		}
	}
	return nil
}

func (i *Interpreter) implementedInterfacesForClass(class *ast.ClassDeclStmt) []string {
	seen := map[string]bool{}
	var names []string
	var collect func(*ast.ClassDeclStmt)
	collect = func(current *ast.ClassDeclStmt) {
		if current == nil {
			return
		}
		if current.Parent != "" {
			if parent, ok := i.classByName(current.Parent); ok {
				collect(parent)
			}
		}
		for _, ifaceName := range current.Implements {
			key := classNameKey(ifaceName)
			if seen[key] {
				continue
			}
			seen[key] = true
			names = append(names, ifaceName)
		}
	}
	collect(class)
	return names
}

func (i *Interpreter) interfaceMethods(name string) (map[string]*ast.FunctionDeclStmt, error) {
	iface, ok := i.interfaceByName(name)
	if !ok {
		return nil, fmt.Errorf("undefined interface %s", name)
	}
	methods := map[string]*ast.FunctionDeclStmt{}
	for _, parentName := range iface.Parents {
		parentMethods, err := i.interfaceMethods(parentName)
		if err != nil {
			return nil, err
		}
		for methodName, method := range parentMethods {
			methods[methodName] = method
		}
	}
	for _, method := range iface.Methods {
		methods[methodNameKey(method.Name)] = method
	}
	return methods, nil
}

func (i *Interpreter) methodsForClass(class *ast.ClassDeclStmt) map[string]*ast.FunctionDeclStmt {
	methods := map[string]*ast.FunctionDeclStmt{}
	if class.Parent != "" {
		if parent, ok := i.classByName(class.Parent); ok {
			for name, method := range i.methodsForClass(parent) {
				methods[name] = method
			}
		}
	}
	for _, method := range class.Methods {
		methods[methodStorageKey(method)] = method
	}
	return methods
}

func (i *Interpreter) instanceDefaults(class *ast.ClassDeclStmt) (map[string]Value, map[string]PropertyMeta, error) {
	props := map[string]Value{}
	meta := map[string]PropertyMeta{}
	if class.Parent != "" {
		parent, ok := i.classByName(class.Parent)
		if !ok {
			return nil, nil, fmt.Errorf("undefined parent class %s", class.Parent)
		}
		parentProps, parentMeta, err := i.instanceDefaults(parent)
		if err != nil {
			return nil, nil, err
		}
		for name, value := range parentProps {
			props[name] = value
		}
		for name, info := range parentMeta {
			meta[name] = info
		}
	}
	for _, prop := range class.Properties {
		if prop.Static {
			continue
		}
		value := NullValue()
		if prop.Default != nil {
			var err error
			value, err = i.eval(prop.Default)
			if err != nil {
				return nil, nil, err
			}
		}
		key := propertyStorageKey(prop.Owner, prop.Visibility, prop.Name)
		props[key] = value
		meta[key] = PropertyMeta{Owner: prop.Owner, Name: prop.Name, Visibility: prop.Visibility}
	}
	return props, meta, nil
}

func (i *Interpreter) getStaticProperty(className, property string) Value {
	resolved, err := i.resolveClassNameAutoload(className)
	if err == nil {
		className = resolved
	}
	if value, _, ok := i.findStaticProperty(className, property); ok {
		return value
	}
	return NullValue()
}

func (i *Interpreter) setStaticProperty(className, property string, value Value) {
	resolved, err := i.resolveClassNameAutoload(className)
	if err == nil {
		className = resolved
	}
	owner := className
	key := property
	if _, _, slotOwner, slotKey, ok := i.findStaticPropertySlot(className, property); ok {
		owner = slotOwner
		key = slotKey
	}
	if _, ok := i.statics[owner]; !ok {
		i.statics[owner] = map[string]Value{}
	}
	i.statics[owner][key] = value
}

func (i *Interpreter) checkMethodAccess(_ string, method *ast.FunctionDeclStmt) error {
	if i.methodAccessible(method) {
		return nil
	}
	return fmt.Errorf("cannot access %s method %s::%s", method.Visibility, method.Owner, method.Name)
}

func (i *Interpreter) checkPropertyAccess(obj *Object, property string) error {
	_, _, info, ok, inaccessible := i.objectProperty(obj, property)
	if !ok && !inaccessible {
		return nil
	}
	if !inaccessible {
		return nil
	}
	return fmt.Errorf("cannot access %s property %s::$%s", info.Visibility, info.Owner, property)
}

func (i *Interpreter) checkStaticPropertyAccess(className, property string) error {
	resolved, err := i.resolveClassNameAutoload(className)
	if err != nil {
		return err
	}
	className = resolved
	_, info, ok := i.findStaticProperty(className, property)
	if !ok {
		return fmt.Errorf("undefined static property %s::$%s", className, property)
	}
	if i.propertyAccessible(info) {
		return nil
	}
	return fmt.Errorf("cannot access %s static property %s::$%s", info.Visibility, info.Owner, property)
}

func (i *Interpreter) methodAccessible(method *ast.FunctionDeclStmt) bool {
	switch method.Visibility {
	case "", "public":
		return true
	case "private":
		return strings.EqualFold(i.currentClass(), method.Owner)
	case "protected":
		return i.classesRelated(i.currentClass(), method.Owner)
	}
	return false
}

func (i *Interpreter) propertyAccessible(info PropertyMeta) bool {
	return i.propertyAccessibleInScope(info, i.currentClass())
}

func (i *Interpreter) propertyAccessibleInScope(info PropertyMeta, scope string) bool {
	switch info.Visibility {
	case "", "public":
		return true
	case "private":
		return strings.EqualFold(scope, info.Owner)
	case "protected":
		return i.classesRelated(scope, info.Owner)
	}
	return false
}

func (i *Interpreter) findStaticProperty(className, property string) (Value, PropertyMeta, bool) {
	value, meta, _, _, ok := i.findStaticPropertySlot(className, property)
	return value, meta, ok
}

func (i *Interpreter) findStaticPropertySlot(className, property string) (Value, PropertyMeta, string, string, bool) {
	if current := i.currentClass(); current != "" && i.isSameOrSubclass(className, current) {
		privateKey := propertyStorageKey(current, "private", property)
		if props, ok := i.statics[current]; ok {
			if value, exists := props[privateKey]; exists {
				if meta, metaOK := i.staticMeta[current][privateKey]; metaOK {
					return value, meta, current, privateKey, true
				}
				return value, PropertyMeta{Owner: current, Name: property, Visibility: "private", Static: true}, current, privateKey, true
			}
		}
	}
	if props, ok := i.statics[className]; ok {
		privateKey := propertyStorageKey(className, "private", property)
		for _, key := range []string{property, privateKey} {
			if value, exists := props[key]; exists {
				if meta, metaOK := i.staticMeta[className][key]; metaOK {
					return value, meta, className, key, true
				}
				return value, PropertyMeta{Owner: className, Name: property, Visibility: "public", Static: true}, className, key, true
			}
		}
	}
	if class, ok := i.classByName(className); ok && class.Parent != "" {
		return i.findStaticPropertySlot(class.Parent, property)
	}
	return NullValue(), PropertyMeta{}, "", "", false
}

func (i *Interpreter) getClassConstant(className, name string) (Value, error) {
	resolved, err := i.resolveClassNameAutoload(className)
	if err != nil {
		return NullValue(), err
	}
	value, ok := i.findClassConstant(resolved, name)
	if !ok {
		return NullValue(), fmt.Errorf("undefined class constant %s::%s", resolved, name)
	}
	return value, nil
}

func (i *Interpreter) findClassConstant(className, name string) (Value, bool) {
	if constants, ok := i.constants[className]; ok {
		if value, exists := constants[name]; exists {
			return value, true
		}
	}
	if class, ok := i.classByName(className); ok {
		if class.Parent != "" {
			if value, ok := i.findClassConstant(class.Parent, name); ok {
				return value, true
			}
		}
		for _, ifaceName := range class.Implements {
			if value, ok := i.findInterfaceConstant(ifaceName, name); ok {
				return value, true
			}
		}
	}
	if _, ok := i.interfaceByName(className); ok {
		return i.findInterfaceConstant(className, name)
	}
	return NullValue(), false
}

func (i *Interpreter) findInterfaceConstant(interfaceName, name string) (Value, bool) {
	if constants, ok := i.constants[interfaceName]; ok {
		if value, exists := constants[name]; exists {
			return value, true
		}
	}
	if iface, ok := i.interfaceByName(interfaceName); ok {
		for _, parentName := range iface.Parents {
			if value, ok := i.findInterfaceConstant(parentName, name); ok {
				return value, true
			}
		}
	}
	return NullValue(), false
}

func (i *Interpreter) currentClass() string {
	if len(i.frames) == 0 {
		return ""
	}
	return i.frames[len(i.frames)-1].className
}

func (i *Interpreter) currentFunction() string {
	if len(i.frames) == 0 {
		return ""
	}
	return i.frames[len(i.frames)-1].function
}

func (i *Interpreter) currentReturnByRef() bool {
	return len(i.frames) > 0 && i.frames[len(i.frames)-1].returnByRef
}

func (i *Interpreter) currentMethod() string {
	if len(i.frames) == 0 {
		return ""
	}
	frame := i.frames[len(i.frames)-1]
	if frame.className == "" || frame.function == "" {
		return ""
	}
	return frame.className + "::" + frame.function
}

func (i *Interpreter) currentCalledClass() string {
	if len(i.frames) == 0 {
		return ""
	}
	return i.frames[len(i.frames)-1].calledClass
}

func (i *Interpreter) currentThis() *Object {
	if len(i.frames) == 0 {
		return nil
	}
	slot, ok := i.frames[len(i.frames)-1].vars["this"]
	if !ok || slot.value.Kind != ObjectKind {
		return nil
	}
	return slot.value.O
}

func (i *Interpreter) resolveClassName(name string) (string, error) {
	switch name {
	case "self":
		if current := i.currentClass(); current != "" {
			return current, nil
		}
		return "", fmt.Errorf("self used outside class context")
	case "parent":
		current := i.currentClass()
		if current == "" {
			return "", fmt.Errorf("parent used outside class context")
		}
		class, ok := i.classByName(current)
		if !ok || class.Parent == "" {
			return "", fmt.Errorf("class %s has no parent", current)
		}
		return class.Parent, nil
	case "static":
		if called := i.currentCalledClass(); called != "" {
			return called, nil
		}
		if current := i.currentClass(); current != "" {
			return current, nil
		}
		return "", fmt.Errorf("static used outside class context")
	default:
		if canonical, ok := i.classLikeName(name); ok {
			return canonical, nil
		}
		return name, nil
	}
}

func (i *Interpreter) resolveClassNameAutoload(name string) (string, error) {
	resolved, err := i.resolveClassName(name)
	if err != nil {
		return "", err
	}
	if _, ok := i.classLikeName(resolved); ok {
		return resolved, nil
	}
	if err := i.autoloadClass(resolved); err != nil {
		return "", err
	}
	if canonical, ok := i.classLikeName(resolved); ok {
		return canonical, nil
	}
	return resolved, nil
}

func (i *Interpreter) classesRelated(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	return i.isSameOrSubclass(a, b) || i.isSameOrSubclass(b, a)
}

func (i *Interpreter) valueInstanceOf(value Value, className string) bool {
	if value.Kind != ObjectKind || value.O == nil {
		return false
	}
	if strings.EqualFold(className, "Exception") && strings.EqualFold(value.O.ClassName, "Exception") {
		return true
	}
	if _, ok := i.classByName(className); ok {
		return i.isSameOrSubclass(value.O.ClassName, className)
	}
	if _, ok := i.interfaceByName(className); ok {
		return i.classImplementsInterface(value.O.ClassName, className)
	}
	return false
}

func (i *Interpreter) isSameOrSubclass(className, ancestor string) bool {
	if class, ok := i.classByName(className); ok {
		className = class.Name
	}
	if class, ok := i.classByName(ancestor); ok {
		ancestor = class.Name
	}
	for className != "" {
		if strings.EqualFold(className, ancestor) {
			return true
		}
		class, ok := i.classByName(className)
		if !ok {
			return false
		}
		className = class.Parent
	}
	return false
}

func (i *Interpreter) classImplementsInterface(className, interfaceName string) bool {
	for className != "" {
		class, ok := i.classByName(className)
		if !ok {
			return false
		}
		for _, implemented := range class.Implements {
			if strings.EqualFold(implemented, interfaceName) || i.interfaceExtendsInterface(implemented, interfaceName) {
				return true
			}
		}
		className = class.Parent
	}
	return false
}

func (i *Interpreter) interfaceExtendsInterface(name, ancestor string) bool {
	if strings.EqualFold(name, ancestor) {
		return true
	}
	iface, ok := i.interfaceByName(name)
	if !ok {
		return false
	}
	for _, parent := range iface.Parents {
		if i.interfaceExtendsInterface(parent, ancestor) {
			return true
		}
	}
	return false
}

func (i *Interpreter) include(path string, require bool, once bool) (Value, error) {
	resolved := i.resolveIncludePath(path)
	if once && i.includedOnce[resolved] {
		return BoolValue(true), nil
	}
	data, err := os.ReadFile(resolved)
	if err != nil {
		kind := "include"
		if require {
			kind = "require"
		}
		message := fmt.Sprintf("%s failed for %q: %v", kind, path, err)
		if !require {
			if err := i.emitDiagnostic(Warning, message); err != nil {
				return NullValue(), err
			}
			return BoolValue(false), nil
		}
		if err := i.emitDiagnostic(FatalError, message); err != nil {
			return NullValue(), err
		}
		return BoolValue(false), nil
	}

	p := parser.New(lexer.New(string(data)))
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		if err := i.emitDiagnostic(ParseError, fmt.Sprintf("parse errors in %s: %v", resolved, p.Errors())); err != nil {
			return NullValue(), err
		}
		return BoolValue(false), nil
	}

	if once {
		i.includedOnce[resolved] = true
	}
	i.dirStack = append(i.dirStack, filepath.Dir(resolved))
	i.fileStack = append(i.fileStack, resolved)
	value, returned, err := i.runProgramResult(program, false)
	i.fileStack = i.fileStack[:len(i.fileStack)-1]
	i.dirStack = i.dirStack[:len(i.dirStack)-1]
	if err != nil {
		return NullValue(), err
	}
	if !returned {
		return IntValue(1), nil
	}
	return value, nil
}

func (i *Interpreter) emitDiagnostic(severity ErrorSeverity, message string) error {
	if i.errorPolicy.Reports(severity) {
		i.diagnostics = append(i.diagnostics, Diagnostic{Severity: severity, Message: message})
	}
	if i.errorPolicy.Halts(severity) {
		return newRuntimeError(severity, message)
	}
	return nil
}

func (i *Interpreter) binary(op string, a, b Value) (Value, error) {
	value, diagnostic, err := BinaryWithDiagnostic(op, a, b)
	if err != nil {
		return NullValue(), err
	}
	if diagnostic != nil {
		if err := i.emitDiagnostic(diagnostic.Severity, diagnostic.Message); err != nil {
			return NullValue(), err
		}
	}
	return value, nil
}

func (i *Interpreter) resolveIncludePath(path string) string {
	if filepath.IsAbs(path) {
		return cleanIncludePath(path)
	}
	if hasExplicitRelativePrefix(path) {
		return cleanIncludePath(filepath.Join(i.currentDir(), path))
	}
	for _, entry := range i.includePath {
		if entry == "" {
			continue
		}
		dir := entry
		if !filepath.IsAbs(dir) {
			dir = filepath.Join(i.baseDir, dir)
		}
		candidate := cleanIncludePath(filepath.Join(dir, path))
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return cleanIncludePath(filepath.Join(i.currentDir(), path))
}

func cleanIncludePath(path string) string {
	resolved := path
	if abs, err := filepath.Abs(resolved); err == nil {
		resolved = abs
	}
	return filepath.Clean(resolved)
}

func splitIncludePath(path string) []string {
	if path == "" {
		return nil
	}
	return filepath.SplitList(path)
}

func hasExplicitRelativePrefix(path string) bool {
	return path == "." || path == ".." ||
		strings.HasPrefix(path, "."+string(filepath.Separator)) ||
		strings.HasPrefix(path, ".."+string(filepath.Separator)) ||
		strings.HasPrefix(path, "./") ||
		strings.HasPrefix(path, "../") ||
		strings.HasPrefix(path, `.\\`) ||
		strings.HasPrefix(path, `..\\`)
}

func (i *Interpreter) currentDir() string {
	if len(i.dirStack) == 0 {
		return i.baseDir
	}
	return i.dirStack[len(i.dirStack)-1]
}

func (i *Interpreter) currentFile() string {
	if len(i.fileStack) == 0 {
		return ""
	}
	return i.fileStack[len(i.fileStack)-1]
}

func (i *Interpreter) currentFrame() map[string]*variableSlot {
	if len(i.frames) == 0 {
		return nil
	}
	return i.frames[len(i.frames)-1].vars
}

func (i *Interpreter) getVar(name string) (Value, bool) {
	if frame := i.currentFrame(); frame != nil {
		slot, ok := frame[name]
		if !ok {
			return NullValue(), false
		}
		return slot.value, true
	}
	slot, ok := i.globals[name]
	if !ok {
		return NullValue(), false
	}
	return slot.value, true
}

func (i *Interpreter) setVar(name string, v Value) {
	if frame := i.currentFrame(); frame != nil {
		slot, ok := frame[name]
		if !ok {
			slot = &variableSlot{}
			frame[name] = slot
		}
		i.setSlot(slot, v)
		return
	}
	slot := i.globalSlot(name)
	i.setSlot(slot, v)
}

func (i *Interpreter) deleteVar(name string) {
	if frame := i.currentFrame(); frame != nil {
		delete(frame, name)
		return
	}
	delete(i.globals, name)
	delete(i.Vars, name)
}

func (i *Interpreter) SetGlobal(name string, value Value) {
	i.setVar(name, value)
}

func (i *Interpreter) SetUploadedFiles(paths []string) {
	if i.uploadedFiles == nil {
		i.uploadedFiles = map[string]bool{}
	}
	for _, path := range paths {
		if path != "" {
			i.uploadedFiles[path] = true
		}
	}
}

func (i *Interpreter) ensureSlot(name string) *variableSlot {
	if frame := i.currentFrame(); frame != nil {
		if slot, ok := frame[name]; ok {
			return slot
		}
		slot := &variableSlot{}
		frame[name] = slot
		return slot
	}
	return i.globalSlot(name)
}

func (i *Interpreter) bindSlot(name string, slot *variableSlot) {
	if frame := i.currentFrame(); frame != nil {
		frame[name] = slot
		return
	}
	i.globals[name] = slot
	i.Vars[name] = slot.value
}

func (i *Interpreter) globalSlot(name string) *variableSlot {
	if slot, ok := i.globals[name]; ok {
		return slot
	}
	value := NullValue()
	if existing, ok := i.Vars[name]; ok {
		value = existing
	}
	slot := &variableSlot{value: value, globalName: name}
	i.globals[name] = slot
	i.Vars[name] = value
	return slot
}

func (i *Interpreter) setSlot(slot *variableSlot, value Value) {
	slot.value = value
	if slot.globalName != "" {
		i.Vars[slot.globalName] = value
	}
}

func (i *Interpreter) bindGlobal(name string) {
	if frame := i.currentFrame(); frame != nil {
		frame[name] = i.globalSlot(name)
	}
}

func (i *Interpreter) variableVariableName(nameVar string) string {
	value, ok := i.getVar(nameVar)
	if !ok {
		return ""
	}
	return value.String()
}

func (i *Interpreter) bindStaticLocals(decls []ast.StaticLocalDecl) error {
	if len(i.frames) == 0 {
		for _, decl := range decls {
			value := NullValue()
			if decl.Default != nil {
				var err error
				value, err = i.eval(decl.Default)
				if err != nil {
					return err
				}
			}
			i.setVar(decl.Name, value)
		}
		return nil
	}
	key := i.frames[len(i.frames)-1].functionKey
	if key == "" {
		key = "<anonymous>"
	}
	if _, ok := i.staticVars[key]; !ok {
		i.staticVars[key] = map[string]*variableSlot{}
	}
	for _, decl := range decls {
		slot, ok := i.staticVars[key][decl.Name]
		if !ok {
			value := NullValue()
			if decl.Default != nil {
				var err error
				value, err = i.eval(decl.Default)
				if err != nil {
					return err
				}
			}
			slot = &variableSlot{value: value}
			i.staticVars[key][decl.Name] = slot
		}
		i.frames[len(i.frames)-1].vars[decl.Name] = slot
	}
	return nil
}
