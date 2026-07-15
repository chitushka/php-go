package runtime

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	goruntime "runtime"
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
	interfaces                  map[string]*ast.InterfaceDeclStmt
	constants                   map[string]map[string]Value
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
	baseDir                     string
	dirStack                    []string
	fileStack                   []string
	objects                     []*Object
	destructed                  map[*Object]bool
	Out                         bytes.Buffer
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
		classes:                     map[string]*ast.ClassDeclStmt{"Exception": builtinExceptionClass()},
		interfaces:                  map[string]*ast.InterfaceDeclStmt{},
		constants:                   map[string]map[string]Value{},
		runtimeConstants:            map[string]Value{},
		runtimeInsensitiveConstants: map[string]Value{},
		staticVars:                  map[string]map[string]*variableSlot{},
		statics:                     map[string]map[string]Value{},
		staticMeta:                  map[string]map[string]PropertyMeta{},
		uploadedFiles:               map[string]bool{},
		includedOnce:                map[string]bool{},
		baseDir:                     baseDir,
		dirStack:                    []string{baseDir},
		fileStack:                   []string{},
		destructed:                  map[*Object]bool{},
	}
}

func builtinExceptionClass() *ast.ClassDeclStmt {
	return &ast.ClassDeclStmt{
		Name: "Exception",
		Properties: []*ast.ClassProperty{
			{Name: "message", Visibility: "private", Default: &ast.StringLit{Value: ""}, Owner: "Exception"},
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

func (i *Interpreter) Run(p *ast.Program) error {
	return i.runProgram(p, true)
}

func (i *Interpreter) runProgram(p *ast.Program, shutdown bool) error {
	for _, s := range p.Statements {
		if iface, ok := s.(*ast.InterfaceDeclStmt); ok {
			if err := i.defineInterface(iface); err != nil {
				return err
			}
		}
	}
	for _, s := range p.Statements {
		if fn, ok := s.(*ast.FunctionDeclStmt); ok {
			if err := i.defineFunction(fn); err != nil {
				return err
			}
			continue
		}
		if class, ok := s.(*ast.ClassDeclStmt); ok {
			if err := i.defineClass(class); err != nil {
				return err
			}
		}
	}
	for _, s := range p.Statements {
		if _, ok := s.(*ast.FunctionDeclStmt); ok {
			continue
		}
		if _, ok := s.(*ast.ClassDeclStmt); ok {
			continue
		}
		if _, ok := s.(*ast.InterfaceDeclStmt); ok {
			continue
		}
		if _, err := i.exec(s); err != nil {
			if _, ok := err.(*ExitError); ok && shutdown {
				if shutdownErr := i.runDestructors(); shutdownErr != nil {
					return shutdownErr
				}
			}
			return err
		}
	}
	if shutdown {
		return i.runDestructors()
	}
	return nil
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
		if message, ok := e.value.O.Props["message"]; ok {
			return "uncaught " + e.value.O.ClassName + ": " + message.String()
		}
		return "uncaught " + e.value.O.ClassName
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
			result, err = Binary(n.Op, current, value)
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
			return nil, fmt.Errorf("foreach source must be array")
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
			return nil, fmt.Errorf("include/require path must be string")
		}
		if err := i.include(path.S, n.Require, n.Once); err != nil {
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
		return NullValue(), nil
	case *ast.VariableVariable:
		name := i.variableVariableName(n.Name)
		if v, ok := i.getVar(name); ok {
			return v, nil
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
			z, err := Binary("^", v, IntValue(-1))
			return z, err
		}
	case *ast.CloneExpr:
		return i.evalClone(n.Expr)
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
		return Binary(n.Op, l, r)
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
		className, err := i.resolveClassName(n.ClassName)
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
	resolved, err := i.resolveClassName(className)
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
		return value, ok, err
	}
	if value, output, ok, err := builtins.CallWithOutput(c.Name, args); ok || err != nil {
		if output != "" {
			i.Out.WriteString(output)
		}
		if err != nil {
			return NullValue(), true, err
		}
		return fromBuiltinValue(value), true, nil
	}
	value, ok, err := builtins.Call(c.Name, args)
	if !ok || err != nil {
		return NullValue(), ok, err
	}
	return fromBuiltinValue(value), true, nil
}

func (i *Interpreter) callDefine(args []ast.Expr) (Value, bool, error) {
	if len(args) < 2 || len(args) > 3 {
		return NullValue(), true, fmt.Errorf("define expects 2 or 3 arguments, got %d", len(args))
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

func (i *Interpreter) callDefined(args []ast.Expr) (Value, bool, error) {
	if len(args) != 1 {
		return NullValue(), true, fmt.Errorf("defined expects 1 argument, got %d", len(args))
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
		return NullValue(), true, fmt.Errorf("constant expects 1 argument, got %d", len(args))
	}
	name, err := i.eval(args[0])
	if err != nil {
		return NullValue(), true, err
	}
	value, ok := i.lookupNamedConstant(name.String())
	if !ok {
		return NullValue(), true, fmt.Errorf("undefined constant %s", name.String())
	}
	return value, true, nil
}

func (i *Interpreter) callIsset(args []ast.Expr) (Value, bool, error) {
	if len(args) == 0 {
		return NullValue(), true, fmt.Errorf("isset expects at least 1 argument, got 0")
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
		return NullValue(), true, fmt.Errorf("empty expects 1 argument, got %d", len(args))
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
		return NullValue(), fmt.Errorf("cannot fetch property %s on non-object", property)
	}
	if err := i.checkPropertyAccess(obj.O, property); err != nil {
		if i.hasPublicMethod(obj.O, "__get") {
			return i.callMagicGet(obj.O, property)
		}
		return NullValue(), err
	}
	if value, ok := obj.O.Props[property]; ok {
		return value, nil
	}
	if i.shouldUseGet(obj.O) {
		return i.callMagicGet(obj.O, property)
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
	resolved, err := i.resolveClassName(className)
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
		return NullValue(), fmt.Errorf("preg_match expects 3 arguments, got %d", len(c.Args))
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
		return NullValue(), err
	}
	arr := NewArray()
	matches := re.FindStringSubmatch(subject.String())
	if matches == nil {
		if err := i.assignBuiltinResult(c.Args[2], ArrayValue(arr)); err != nil {
			return NullValue(), err
		}
		return IntValue(0), nil
	}
	for _, match := range matches {
		arr.Append(StringValue(match))
	}
	if err := i.assignBuiltinResult(c.Args[2], ArrayValue(arr)); err != nil {
		return NullValue(), err
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
				return nil, fmt.Errorf("function %s expects at least %d arguments, got %d", fn.Name, i.requiredParamCount(fn), len(args))
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
				return nil, fmt.Errorf("function %s expects at least %d arguments, got %d", fn.Name, i.requiredParamCount(fn), len(args))
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

func (i *Interpreter) callClosureResult(closure *Closure, args []ast.Expr) (functionCallResult, error) {
	if closure == nil || closure.Fn == nil {
		return functionCallResult{}, fmt.Errorf("invalid closure")
	}
	fn := closure.Fn
	return i.callFunctionWithBinderResult(fn, len(args), nil, "", closure.Captures, func(idx int, param ast.Param) (*variableSlot, error) {
		if idx >= len(args) {
			if param.Default == nil {
				return nil, fmt.Errorf("function %s expects at least %d arguments, got %d", fn.Name, i.requiredParamCount(fn), len(args))
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

func (i *Interpreter) newObject(n *ast.NewExpr) (Value, error) {
	if n.ClassName == "Exception" {
		return i.newException(n.Args)
	}
	className, err := i.resolveClassName(n.ClassName)
	if err != nil {
		return NullValue(), err
	}
	class, ok := i.classes[className]
	if !ok {
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
	i.trackObject(obj)
	if constructor, ok := i.constructorForClass(class); ok {
		if _, err := i.callFunction(constructor, n.Args, obj, obj.ClassName); err != nil {
			return NullValue(), err
		}
	} else if i.isExceptionObject(obj) {
		if err := i.initExceptionObject(obj, n.Args); err != nil {
			return NullValue(), err
		}
	} else if len(n.Args) != 0 {
		return NullValue(), fmt.Errorf("class %s has no constructor", n.ClassName)
	}
	return ObjectValue(obj), nil
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
	i.trackObject(clone)
	if method, ok := clone.Methods["__clone"]; ok {
		if _, err := i.callFunction(method, nil, clone, clone.ClassName); err != nil {
			return NullValue(), err
		}
	}
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
	return NewObject(obj.ClassName, obj.Methods, props, meta)
}

func (i *Interpreter) runDestructors() error {
	for {
		obj := i.nextUndestructedObject()
		if obj == nil {
			return nil
		}
		i.destructed[obj] = true
		destructor, ok := obj.Methods["__destruct"]
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
	baseName := namespaceBaseName(class.Name)
	if method, ok := declaredMethod(class, baseName); ok {
		return method, true
	}
	if class.Parent == "" {
		return nil, false
	}
	parent, ok := i.classes[class.Parent]
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

func (i *Interpreter) callMethodName(object ast.Expr, methodName string, args []ast.Expr) (Value, error) {
	obj, err := i.eval(object)
	if err != nil {
		return NullValue(), err
	}
	if obj.Kind != ObjectKind || obj.O == nil {
		return NullValue(), fmt.Errorf("cannot call method %s on non-object", methodName)
	}
	if methodName == "getMessage" && i.isExceptionObject(obj.O) {
		if len(args) != 0 {
			return NullValue(), fmt.Errorf("Exception::getMessage expects 0 arguments, got %d", len(args))
		}
		if message, ok := obj.O.Props["message"]; ok {
			return message, nil
		}
		return StringValue(""), nil
	}
	method, ok := obj.O.Methods[methodName]
	if !ok {
		if methodName != "__call" && i.hasPublicMethod(obj.O, "__call") {
			return i.callMagicCall(obj.O, methodName, args)
		}
		return NullValue(), fmt.Errorf("undefined method %s::%s", obj.O.ClassName, methodName)
	}
	if err := i.checkMethodAccess(obj.O.ClassName, method); err != nil {
		if methodName != "__call" && i.hasPublicMethod(obj.O, "__call") {
			return i.callMagicCall(obj.O, methodName, args)
		}
		return NullValue(), err
	}
	return i.callFunction(method, args, obj.O, obj.O.ClassName)
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
	className, err := i.resolveClassName(staticClassName)
	if err != nil {
		return NullValue(), err
	}
	class, ok := i.classes[className]
	if !ok {
		return NullValue(), fmt.Errorf("undefined class %s", className)
	}
	methods := i.methodsForClass(class)
	method, ok := methods[methodName]
	if !ok {
		if i.hasPublicStaticMethod(methods, "__callStatic") {
			return i.callMagicCallStatic(className, methods["__callStatic"], methodName, args)
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
			return i.callMagicCallStatic(className, methods["__callStatic"], methodName, args)
		}
		return NullValue(), err
	}
	return i.callFunction(method, args, nil, className)
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
		if left.A == nil {
			return NullValue(), nil
		}
		value, _ := left.A.Get(KeyFromValue(index))
		return value, nil
	case String:
		offset, ok := indexOffset(index)
		if !ok || offset < 0 || offset >= len(left.S) {
			return NullValue(), nil
		}
		return StringValue(left.S[offset : offset+1]), nil
	default:
		return NullValue(), fmt.Errorf("cannot use index on %s", left.String())
	}
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
			value, err = Binary(n.Op, current, value)
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
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return NullValue(), false
	}
	if value, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
		return IntValue(value), true
	}
	if value, err := strconv.ParseFloat(trimmed, 64); err == nil {
		return FloatValue(value), true
	}
	return NullValue(), false
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
			return nil, fmt.Errorf("cannot assign index on non-array")
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
			return nil, fmt.Errorf("cannot assign index on non-array")
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
			return nil, fmt.Errorf("cannot assign index on non-array")
		}
		return current.A, nil
	default:
		value, err := i.eval(target)
		if err != nil {
			return nil, err
		}
		if value.Kind != ArrayKind || value.A == nil {
			return nil, fmt.Errorf("cannot assign index on non-array")
		}
		return value.A, nil
	}
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
	if err := i.checkPropertyAccess(obj.O, property); err != nil {
		if i.hasPublicMethod(obj.O, "__set") {
			_, callErr := i.callMagicSet(obj.O, property, value)
			return callErr
		}
		return err
	}
	if _, ok := obj.O.Props[property]; !ok && i.shouldUseSet(obj.O) {
		_, callErr := i.callMagicSet(obj.O, property, value)
		return callErr
	}
	obj.O.Props[property] = value
	return nil
}

func (i *Interpreter) assignList(target *ast.ListExpr, value Value) error {
	if value.Kind != ArrayKind || value.A == nil {
		return fmt.Errorf("list assignment expects array")
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
	if _, ok, inaccessible := i.visibleObjectProperty(obj.O, property); inaccessible {
		if i.shouldUseUnset(obj.O) {
			_, callErr := i.callMagicUnset(obj.O, property)
			return callErr
		}
		return i.checkPropertyAccess(obj.O, property)
	} else if !ok && i.shouldUseUnset(obj.O) {
		_, callErr := i.callMagicUnset(obj.O, property)
		return callErr
	}
	delete(obj.O.Props, property)
	delete(obj.O.PropMeta, property)
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
		if i.hasPublicMethod(value.O, "__toString") {
			result, err := i.callFunctionValues(value.O.Methods["__toString"], nil, value.O, value.O.ClassName)
			if err != nil {
				return "", err
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
	if err := i.checkPropertyAccess(value.O, property); err != nil {
		if i.hasPublicMethod(value.O, "__get") {
			return i.callMagicGet(value.O, property)
		}
		return NullValue(), nil
	}
	if propValue, ok := value.O.Props[property]; ok {
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
				meta[name] = PropertyMeta{Owner: "stdClass", Visibility: "public"}
			}
		} else if value.Kind != Null {
			props["scalar"] = value
			meta["scalar"] = PropertyMeta{Owner: "stdClass", Visibility: "public"}
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
	method, ok := obj.Methods[name]
	if !ok {
		return false
	}
	return method.Visibility == "" || method.Visibility == "public"
}

func (i *Interpreter) hasPublicStaticMethod(methods map[string]*ast.FunctionDeclStmt, name string) bool {
	method, ok := methods[name]
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

func (i *Interpreter) visibleObjectProperty(obj *Object, property string) (Value, bool, bool) {
	if err := i.checkPropertyAccess(obj, property); err != nil {
		return NullValue(), false, true
	}
	value, ok := obj.Props[property]
	return value, ok, false
}

func (i *Interpreter) callMagicGet(obj *Object, property string) (Value, error) {
	return i.callFunctionValues(obj.Methods["__get"], []Value{StringValue(property)}, obj, obj.ClassName)
}

func (i *Interpreter) callMagicSet(obj *Object, property string, value Value) (Value, error) {
	return i.callFunctionValues(obj.Methods["__set"], []Value{StringValue(property), value}, obj, obj.ClassName)
}

func (i *Interpreter) callMagicIsset(obj *Object, property string) (Value, error) {
	return i.callFunctionValues(obj.Methods["__isset"], []Value{StringValue(property)}, obj, obj.ClassName)
}

func (i *Interpreter) callMagicUnset(obj *Object, property string) (Value, error) {
	return i.callFunctionValues(obj.Methods["__unset"], []Value{StringValue(property)}, obj, obj.ClassName)
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
	return i.callFunctionValues(obj.Methods["__call"], []Value{StringValue(methodName), ArrayValue(values)}, obj, obj.ClassName)
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

func (i *Interpreter) newException(args []ast.Expr) (Value, error) {
	obj := NewObject("Exception", i.methodsForClass(i.classes["Exception"]), nil, nil)
	if err := i.initExceptionObject(obj, args); err != nil {
		return NullValue(), err
	}
	return ObjectValue(obj), nil
}

func (i *Interpreter) initExceptionObject(obj *Object, args []ast.Expr) error {
	if len(args) > 1 {
		return fmt.Errorf("Exception expects at most 1 argument, got %d", len(args))
	}
	message := StringValue("")
	if len(args) == 1 {
		value, err := i.eval(args[0])
		if err != nil {
			return err
		}
		message = StringValue(value.String())
	}
	obj.Props["message"] = message
	obj.PropMeta["message"] = PropertyMeta{Owner: "Exception", Visibility: "private"}
	return nil
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
	if _, exists := i.classes[class.Name]; exists {
		return fmt.Errorf("cannot redeclare class %s", class.Name)
	}
	if class.Final && class.Abstract {
		return fmt.Errorf("class %s cannot be both final and abstract", class.Name)
	}
	for _, method := range class.Methods {
		method.Owner = class.Name
		if method.Visibility == "" {
			method.Visibility = "public"
		}
		if err := validateFunctionCompatibility(method, "method "+class.Name+"::"+method.Name); err != nil {
			return err
		}
		if method.Final && method.Abstract {
			return fmt.Errorf("method %s::%s cannot be both final and abstract", class.Name, method.Name)
		}
	}
	for _, prop := range class.Properties {
		prop.Owner = class.Name
		if prop.Visibility == "" {
			prop.Visibility = "public"
		}
	}
	if class.Parent != "" {
		parent, ok := i.classes[class.Parent]
		if !ok {
			return fmt.Errorf("undefined parent class %s", class.Parent)
		}
		if parent.Final {
			return fmt.Errorf("class %s cannot extend final class %s", class.Name, parent.Name)
		}
		parentMethods := i.methodsForClass(parent)
		for _, method := range class.Methods {
			if parentMethod, ok := parentMethods[method.Name]; ok {
				if err := validateMethodOverrideCompatibility(class.Name, method, parentMethod); err != nil {
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
	i.classes[class.Name] = class
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
		i.statics[class.Name][prop.Name] = value
		i.staticMeta[class.Name][prop.Name] = PropertyMeta{Owner: prop.Owner, Visibility: prop.Visibility, Static: true}
	}
	if !class.Abstract {
		for _, method := range i.methodsForClass(class) {
			if method.Abstract {
				return fmt.Errorf("class %s contains abstract methods and must be abstract", class.Name)
			}
		}
		if err := i.checkInterfaceImplementations(class); err != nil {
			return err
		}
	}
	return nil
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

func validateMethodOverrideCompatibility(className string, method, parentMethod *ast.FunctionDeclStmt) error {
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
	if err := validateMethodSignatureCompatibility(className, method, parentMethod); err != nil {
		return err
	}
	return nil
}

func validateMethodSignatureCompatibility(className string, method, prototype *ast.FunctionDeclStmt) error {
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
	}
	if method.ReturnsByRef != prototype.ReturnsByRef {
		return fmt.Errorf("method %s return-by-reference modifier must match %s", methodLabel, prototypeLabel)
	}
	return nil
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
		if parent, ok := i.classes[class.Parent]; ok {
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
	if _, exists := i.interfaces[iface.Name]; exists {
		return fmt.Errorf("cannot redeclare interface %s", iface.Name)
	}
	for _, parentName := range iface.Parents {
		if _, ok := i.interfaces[parentName]; !ok {
			return fmt.Errorf("undefined parent interface %s", parentName)
		}
	}
	for _, method := range iface.Methods {
		method.Owner = iface.Name
		method.Visibility = "public"
		method.Abstract = true
		if err := validateFunctionCompatibility(method, "method "+iface.Name+"::"+method.Name); err != nil {
			return err
		}
	}
	i.interfaces[iface.Name] = iface
	return nil
}

func (i *Interpreter) checkInterfaceImplementations(class *ast.ClassDeclStmt) error {
	methods := i.methodsForClass(class)
	for _, ifaceName := range class.Implements {
		required, err := i.interfaceMethods(ifaceName)
		if err != nil {
			return err
		}
		for name, ifaceMethod := range required {
			method, ok := methods[name]
			if !ok || method.Abstract {
				return fmt.Errorf("class %s must implement interface method %s::%s", class.Name, ifaceName, name)
			}
			if method.Visibility != "" && method.Visibility != "public" {
				return fmt.Errorf("class %s interface method %s must be public", class.Name, name)
			}
			if err := validateMethodSignatureCompatibility(class.Name, method, ifaceMethod); err != nil {
				return err
			}
		}
	}
	return nil
}

func (i *Interpreter) interfaceMethods(name string) (map[string]*ast.FunctionDeclStmt, error) {
	iface, ok := i.interfaces[name]
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
		methods[method.Name] = method
	}
	return methods, nil
}

func (i *Interpreter) methodsForClass(class *ast.ClassDeclStmt) map[string]*ast.FunctionDeclStmt {
	methods := map[string]*ast.FunctionDeclStmt{}
	if class.Parent != "" {
		if parent, ok := i.classes[class.Parent]; ok {
			for name, method := range i.methodsForClass(parent) {
				methods[name] = method
			}
		}
	}
	for _, method := range class.Methods {
		methods[method.Name] = method
	}
	return methods
}

func (i *Interpreter) instanceDefaults(class *ast.ClassDeclStmt) (map[string]Value, map[string]PropertyMeta, error) {
	props := map[string]Value{}
	meta := map[string]PropertyMeta{}
	if class.Parent != "" {
		parent, ok := i.classes[class.Parent]
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
		props[prop.Name] = value
		meta[prop.Name] = PropertyMeta{Owner: prop.Owner, Visibility: prop.Visibility}
	}
	return props, meta, nil
}

func (i *Interpreter) getStaticProperty(className, property string) Value {
	resolved, err := i.resolveClassName(className)
	if err == nil {
		className = resolved
	}
	if value, _, ok := i.findStaticProperty(className, property); ok {
		return value
	}
	return NullValue()
}

func (i *Interpreter) setStaticProperty(className, property string, value Value) {
	resolved, err := i.resolveClassName(className)
	if err == nil {
		className = resolved
	}
	owner := className
	if _, info, ok := i.findStaticProperty(className, property); ok {
		owner = info.Owner
	}
	if _, ok := i.statics[owner]; !ok {
		i.statics[owner] = map[string]Value{}
	}
	i.statics[owner][property] = value
}

func (i *Interpreter) checkMethodAccess(_ string, method *ast.FunctionDeclStmt) error {
	switch method.Visibility {
	case "", "public":
		return nil
	case "private":
		if i.currentClass() == method.Owner {
			return nil
		}
	case "protected":
		if i.classesRelated(i.currentClass(), method.Owner) {
			return nil
		}
	}
	return fmt.Errorf("cannot access %s method %s::%s", method.Visibility, method.Owner, method.Name)
}

func (i *Interpreter) checkPropertyAccess(obj *Object, property string) error {
	info, ok := obj.PropMeta[property]
	if !ok {
		return nil
	}
	switch info.Visibility {
	case "", "public":
		return nil
	case "private":
		if i.currentClass() == info.Owner {
			return nil
		}
	case "protected":
		if i.classesRelated(i.currentClass(), info.Owner) {
			return nil
		}
	}
	return fmt.Errorf("cannot access %s property %s::$%s", info.Visibility, info.Owner, property)
}

func (i *Interpreter) checkStaticPropertyAccess(className, property string) error {
	resolved, err := i.resolveClassName(className)
	if err != nil {
		return err
	}
	className = resolved
	_, info, ok := i.findStaticProperty(className, property)
	if !ok {
		return fmt.Errorf("undefined static property %s::$%s", className, property)
	}
	switch info.Visibility {
	case "", "public":
		return nil
	case "private":
		if i.currentClass() == info.Owner {
			return nil
		}
	case "protected":
		if i.classesRelated(i.currentClass(), info.Owner) {
			return nil
		}
	}
	return fmt.Errorf("cannot access %s static property %s::$%s", info.Visibility, info.Owner, property)
}

func (i *Interpreter) findStaticProperty(className, property string) (Value, PropertyMeta, bool) {
	if props, ok := i.statics[className]; ok {
		if value, exists := props[property]; exists {
			if meta, metaOK := i.staticMeta[className][property]; metaOK {
				return value, meta, true
			}
			return value, PropertyMeta{Owner: className, Visibility: "public", Static: true}, true
		}
	}
	if class, ok := i.classes[className]; ok && class.Parent != "" {
		return i.findStaticProperty(class.Parent, property)
	}
	return NullValue(), PropertyMeta{}, false
}

func (i *Interpreter) getClassConstant(className, name string) (Value, error) {
	resolved, err := i.resolveClassName(className)
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
	if class, ok := i.classes[className]; ok && class.Parent != "" {
		return i.findClassConstant(class.Parent, name)
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
		class, ok := i.classes[current]
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
		return name, nil
	}
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
	if className == "Exception" && value.O.ClassName == "Exception" {
		return true
	}
	if _, ok := i.classes[className]; ok {
		return i.isSameOrSubclass(value.O.ClassName, className)
	}
	if _, ok := i.interfaces[className]; ok {
		return i.classImplementsInterface(value.O.ClassName, className)
	}
	return false
}

func (i *Interpreter) isSameOrSubclass(className, ancestor string) bool {
	for className != "" {
		if className == ancestor {
			return true
		}
		class, ok := i.classes[className]
		if !ok {
			return false
		}
		className = class.Parent
	}
	return false
}

func (i *Interpreter) classImplementsInterface(className, interfaceName string) bool {
	for className != "" {
		class, ok := i.classes[className]
		if !ok {
			return false
		}
		for _, implemented := range class.Implements {
			if implemented == interfaceName || i.interfaceExtendsInterface(implemented, interfaceName) {
				return true
			}
		}
		className = class.Parent
	}
	return false
}

func (i *Interpreter) interfaceExtendsInterface(name, ancestor string) bool {
	if name == ancestor {
		return true
	}
	iface, ok := i.interfaces[name]
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

func (i *Interpreter) include(path string, require bool, once bool) error {
	resolved := i.resolveIncludePath(path)
	if once && i.includedOnce[resolved] {
		return nil
	}
	data, err := os.ReadFile(resolved)
	if err != nil {
		kind := "include"
		if require {
			kind = "require"
		}
		return fmt.Errorf("%s failed for %q: %w", kind, path, err)
	}

	p := parser.New(lexer.New(string(data)))
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		return fmt.Errorf("parse errors in %s: %v", resolved, p.Errors())
	}

	if once {
		i.includedOnce[resolved] = true
	}
	i.dirStack = append(i.dirStack, filepath.Dir(resolved))
	i.fileStack = append(i.fileStack, resolved)
	err = i.runProgram(program, false)
	i.fileStack = i.fileStack[:len(i.fileStack)-1]
	i.dirStack = i.dirStack[:len(i.dirStack)-1]
	return err
}

func (i *Interpreter) resolveIncludePath(path string) string {
	resolved := path
	if !filepath.IsAbs(resolved) {
		resolved = filepath.Join(i.currentDir(), path)
	}
	if abs, err := filepath.Abs(resolved); err == nil {
		resolved = abs
	}
	return filepath.Clean(resolved)
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
