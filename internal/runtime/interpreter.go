package runtime

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"php-src-go/internal/ast"
	"php-src-go/internal/lexer"
	"php-src-go/internal/parser"
	"php-src-go/internal/runtime/builtins"
)

type Interpreter struct {
	Vars            map[string]Value
	globals         map[string]*variableSlot
	functions       map[string]*ast.FunctionDeclStmt
	classes         map[string]*ast.ClassDeclStmt
	interfaces      map[string]*ast.InterfaceDeclStmt
	constants       map[string]map[string]Value
	staticVars      map[string]map[string]*variableSlot
	statics         map[string]map[string]Value
	staticMeta      map[string]map[string]PropertyMeta
	frames          []frame
	responseHeaders []string
	baseDir         string
	dirStack        []string
	Out             bytes.Buffer
}

func NewInterpreter() *Interpreter {
	return NewInterpreterWithBaseDir(".")
}

func NewInterpreterWithBaseDir(baseDir string) *Interpreter {
	if baseDir == "" {
		baseDir = "."
	}
	return &Interpreter{
		Vars:       map[string]Value{},
		globals:    map[string]*variableSlot{},
		functions:  map[string]*ast.FunctionDeclStmt{},
		classes:    map[string]*ast.ClassDeclStmt{},
		interfaces: map[string]*ast.InterfaceDeclStmt{},
		constants:  map[string]map[string]Value{},
		staticVars: map[string]map[string]*variableSlot{},
		statics:    map[string]map[string]Value{},
		staticMeta: map[string]map[string]PropertyMeta{},
		baseDir:    baseDir,
		dirStack:   []string{baseDir},
	}
}

func (i *Interpreter) Run(p *ast.Program) error {
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
			return err
		}
	}
	return nil
}

type returnSignal struct {
	value Value
}

type variableSlot struct {
	value      Value
	globalName string
}

type breakSignal struct{}

func (*breakSignal) Error() string { return "break used outside loop or switch" }

type continueSignal struct{}

func (*continueSignal) Error() string { return "continue used outside loop" }

type frame struct {
	vars        map[string]*variableSlot
	className   string
	calledClass string
	functionKey string
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
			if _, ok := err.(*breakSignal); ok {
				break
			}
			if _, ok := err.(*continueSignal); ok {
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
			if _, ok := err.(*breakSignal); ok {
				break
			}
			if _, ok := err.(*continueSignal); !ok {
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
			if _, ok := err.(*breakSignal); ok {
				break
			}
			if _, ok := err.(*continueSignal); !ok {
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
			i.setVar(n.ValueVar, entry.Value)
			ret, err := i.exec(n.Body)
			if _, ok := err.(*breakSignal); ok {
				break
			}
			if _, ok := err.(*continueSignal); ok {
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
			if _, ok := err.(*breakSignal); ok {
				return nil, nil
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
		obj, err := i.eval(n.Object)
		if err != nil {
			return nil, err
		}
		if obj.Kind != ObjectKind || obj.O == nil {
			return nil, fmt.Errorf("cannot assign property %s on non-object", n.Property)
		}
		value, err := i.eval(n.Value)
		if err != nil {
			return nil, err
		}
		if err := i.checkPropertyAccess(obj.O, n.Property); err != nil {
			if i.hasPublicMethod(obj.O, "__set") {
				_, callErr := i.callMagicSet(obj.O, n.Property, value)
				return nil, callErr
			}
			return nil, err
		}
		if _, ok := obj.O.Props[n.Property]; !ok && i.shouldUseSet(obj.O) {
			_, callErr := i.callMagicSet(obj.O, n.Property, value)
			return nil, callErr
		}
		obj.O.Props[n.Property] = value
	case *ast.StaticPropertyAssignStmt:
		value, err := i.eval(n.Value)
		if err != nil {
			return nil, err
		}
		if err := i.checkStaticPropertyAccess(n.ClassName, n.Property); err != nil {
			return nil, err
		}
		i.setStaticProperty(n.ClassName, n.Property, value)
	case *ast.IndexAssignStmt:
		value, err := i.eval(n.Value)
		if err != nil {
			return nil, err
		}
		if err := i.assignIndex(n.Target, value); err != nil {
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
		if !i.exceptionMatches(thrown.value, n.CatchClass) {
			return nil, err
		}
		i.setVar(n.CatchVar, thrown.value)
		return i.exec(n.CatchBlock)
	case *ast.ReturnStmt:
		if n.Value == nil {
			return &returnSignal{value: NullValue()}, nil
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
		if err := i.include(path.S, n.Require); err != nil {
			return nil, err
		}
	case *ast.ExprStmt:
		_, err := i.eval(n.Expr)
		return nil, err
	case *ast.BreakStmt:
		return nil, &breakSignal{}
	case *ast.ContinueStmt:
		return nil, &continueSignal{}
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
		switch n.Op {
		case "&&":
			if !l.Truthy() {
				return BoolValue(false), nil
			}
			r, err := i.eval(n.Right)
			if err != nil {
				return NullValue(), err
			}
			return BoolValue(r.Truthy()), nil
		case "||":
			if l.Truthy() {
				return BoolValue(true), nil
			}
			r, err := i.eval(n.Right)
			if err != nil {
				return NullValue(), err
			}
			return BoolValue(r.Truthy()), nil
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
	case *ast.NewExpr:
		return i.newObject(n)
	case *ast.PropertyFetchExpr:
		obj, err := i.eval(n.Object)
		if err != nil {
			return NullValue(), err
		}
		if obj.Kind != ObjectKind || obj.O == nil {
			return NullValue(), fmt.Errorf("cannot fetch property %s on non-object", n.Property)
		}
		if err := i.checkPropertyAccess(obj.O, n.Property); err != nil {
			if i.hasPublicMethod(obj.O, "__get") {
				return i.callMagicGet(obj.O, n.Property)
			}
			return NullValue(), err
		}
		if value, ok := obj.O.Props[n.Property]; ok {
			return value, nil
		}
		if i.shouldUseGet(obj.O) {
			return i.callMagicGet(obj.O, n.Property)
		}
		return NullValue(), nil
	case *ast.MethodCallExpr:
		return i.callMethod(n)
	case *ast.StaticPropertyFetchExpr:
		if err := i.checkStaticPropertyAccess(n.ClassName, n.Property); err != nil {
			return NullValue(), err
		}
		return i.getStaticProperty(n.ClassName, n.Property), nil
	case *ast.ClassConstFetchExpr:
		return i.getClassConstant(n.ClassName, n.Name)
	case *ast.StaticCallExpr:
		return i.callStatic(n)
	}
	return NullValue(), fmt.Errorf("unsupported expression %T", e)
}

func (i *Interpreter) call(c *ast.CallExpr) (Value, error) {
	if value, ok, err := i.callBuiltin(c); ok || err != nil {
		return value, err
	}
	fn, ok := i.functions[c.Name]
	if !ok {
		return NullValue(), fmt.Errorf("undefined function %s", c.Name)
	}
	return i.callFunction(fn, c.Args, nil, "")
}

func (i *Interpreter) callBuiltin(c *ast.CallExpr) (Value, bool, error) {
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
		out.Set(toBuiltinArrayKey(entry.Key), toBuiltinValue(entry.Value))
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
	return i.callFunctionWithBinder(fn, len(args), thisObj, calledClass, func(idx int, param ast.Param) (*variableSlot, error) {
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
			ident, ok := args[idx].(*ast.Ident)
			if !ok {
				return nil, fmt.Errorf("argument %d for %s must be a variable passed by reference", idx+1, fn.Name)
			}
			return i.ensureSlot(ident.Name), nil
		}
		value, err := i.eval(args[idx])
		if err != nil {
			return nil, err
		}
		return &variableSlot{value: value}, nil
	})
}

func (i *Interpreter) callFunctionValues(fn *ast.FunctionDeclStmt, args []Value, thisObj *Object, calledClass string) (Value, error) {
	return i.callFunctionWithBinder(fn, len(args), thisObj, calledClass, func(idx int, param ast.Param) (*variableSlot, error) {
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
}

func (i *Interpreter) callFunctionWithBinder(fn *ast.FunctionDeclStmt, provided int, thisObj *Object, calledClass string, bind func(int, ast.Param) (*variableSlot, error)) (Value, error) {
	if fn.Abstract {
		return NullValue(), fmt.Errorf("cannot call abstract method %s::%s", fn.Owner, fn.Name)
	}
	if provided > len(fn.Params) {
		return NullValue(), fmt.Errorf("function %s expects at most %d arguments, got %d", fn.Name, len(fn.Params), provided)
	}
	if calledClass == "" {
		calledClass = fn.Owner
	}
	callFrame := frame{vars: map[string]*variableSlot{}, className: fn.Owner, calledClass: calledClass, functionKey: functionKey(fn)}
	if thisObj != nil {
		callFrame.vars["this"] = &variableSlot{value: ObjectValue(thisObj)}
	}
	for idx, param := range fn.Params {
		slot, err := bind(idx, param)
		if err != nil {
			return NullValue(), err
		}
		callFrame.vars[param.Name] = slot
	}
	i.frames = append(i.frames, callFrame)
	ret, err := i.exec(fn.Body)
	i.frames = i.frames[:len(i.frames)-1]
	if err != nil {
		return NullValue(), err
	}
	if ret == nil {
		return NullValue(), nil
	}
	return ret.value, nil
}

func (i *Interpreter) requiredParamCount(fn *ast.FunctionDeclStmt) int {
	required := 0
	for _, param := range fn.Params {
		if param.Default == nil {
			required++
		}
	}
	return required
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
	if constructor, ok := obj.Methods["__construct"]; ok {
		if _, err := i.callFunction(constructor, n.Args, obj, obj.ClassName); err != nil {
			return NullValue(), err
		}
	} else if len(n.Args) != 0 {
		return NullValue(), fmt.Errorf("class %s has no constructor", n.ClassName)
	}
	return ObjectValue(obj), nil
}

func (i *Interpreter) callMethod(n *ast.MethodCallExpr) (Value, error) {
	obj, err := i.eval(n.Object)
	if err != nil {
		return NullValue(), err
	}
	if obj.Kind != ObjectKind || obj.O == nil {
		return NullValue(), fmt.Errorf("cannot call method %s on non-object", n.Method)
	}
	if obj.O.ClassName == "Exception" && n.Method == "getMessage" {
		if len(n.Args) != 0 {
			return NullValue(), fmt.Errorf("Exception::getMessage expects 0 arguments, got %d", len(n.Args))
		}
		if message, ok := obj.O.Props["message"]; ok {
			return message, nil
		}
		return StringValue(""), nil
	}
	method, ok := obj.O.Methods[n.Method]
	if !ok {
		if n.Method != "__call" && i.hasPublicMethod(obj.O, "__call") {
			return i.callMagicCall(obj.O, n.Method, n.Args)
		}
		return NullValue(), fmt.Errorf("undefined method %s::%s", obj.O.ClassName, n.Method)
	}
	if err := i.checkMethodAccess(obj.O.ClassName, method); err != nil {
		if n.Method != "__call" && i.hasPublicMethod(obj.O, "__call") {
			return i.callMagicCall(obj.O, n.Method, n.Args)
		}
		return NullValue(), err
	}
	return i.callFunction(method, n.Args, obj.O, obj.O.ClassName)
}

func (i *Interpreter) callStatic(n *ast.StaticCallExpr) (Value, error) {
	className, err := i.resolveClassName(n.ClassName)
	if err != nil {
		return NullValue(), err
	}
	class, ok := i.classes[className]
	if !ok {
		return NullValue(), fmt.Errorf("undefined class %s", className)
	}
	methods := i.methodsForClass(class)
	method, ok := methods[n.Method]
	if !ok {
		return NullValue(), fmt.Errorf("undefined static method %s::%s", className, n.Method)
	}
	if !method.Static {
		thisObj := i.currentThis()
		if thisObj == nil {
			return NullValue(), fmt.Errorf("method %s::%s is not static", className, n.Method)
		}
		if err := i.checkMethodAccess(className, method); err != nil {
			return NullValue(), err
		}
		return i.callFunction(method, n.Args, thisObj, thisObj.ClassName)
	}
	if err := i.checkMethodAccess(className, method); err != nil {
		return NullValue(), err
	}
	return i.callFunction(method, n.Args, nil, className)
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

func (i *Interpreter) evalIndex(n *ast.IndexExpr) (Value, error) {
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

func (i *Interpreter) assignIndex(target *ast.IndexExpr, value Value) error {
	index, err := i.eval(target.Index)
	if err != nil {
		return err
	}
	key := KeyFromValue(index)
	if ident, ok := target.Left.(*ast.Ident); ok {
		current, exists := i.getVar(ident.Name)
		if !exists || current.Kind == Null {
			arr := NewArray()
			arr.Set(key, value)
			i.setVar(ident.Name, ArrayValue(arr))
			return nil
		}
		if current.Kind != ArrayKind || current.A == nil {
			return fmt.Errorf("cannot assign index on non-array")
		}
		current.A.Set(key, value)
		return nil
	}
	left, err := i.eval(target.Left)
	if err != nil {
		return err
	}
	if left.Kind != ArrayKind || left.A == nil {
		return fmt.Errorf("cannot assign index on non-array")
	}
	left.A.Set(key, value)
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

func (i *Interpreter) hasPublicMethod(obj *Object, name string) bool {
	method, ok := obj.Methods[name]
	if !ok {
		return false
	}
	return method.Visibility == "" || method.Visibility == "public"
}

func (i *Interpreter) shouldUseGet(obj *Object) bool {
	return i.hasPublicMethod(obj, "__get") && i.currentClass() != obj.ClassName
}

func (i *Interpreter) shouldUseSet(obj *Object) bool {
	return i.hasPublicMethod(obj, "__set") && i.currentClass() != obj.ClassName
}

func (i *Interpreter) callMagicGet(obj *Object, property string) (Value, error) {
	return i.callFunctionValues(obj.Methods["__get"], []Value{StringValue(property)}, obj, obj.ClassName)
}

func (i *Interpreter) callMagicSet(obj *Object, property string, value Value) (Value, error) {
	return i.callFunctionValues(obj.Methods["__set"], []Value{StringValue(property), value}, obj, obj.ClassName)
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

func (i *Interpreter) newException(args []ast.Expr) (Value, error) {
	if len(args) > 1 {
		return NullValue(), fmt.Errorf("Exception expects at most 1 argument, got %d", len(args))
	}
	message := StringValue("")
	if len(args) == 1 {
		value, err := i.eval(args[0])
		if err != nil {
			return NullValue(), err
		}
		message = StringValue(value.String())
	}
	obj := NewObject("Exception", map[string]*ast.FunctionDeclStmt{}, map[string]Value{"message": message}, map[string]PropertyMeta{"message": {Owner: "Exception", Visibility: "private"}})
	return ObjectValue(obj), nil
}

func (i *Interpreter) exceptionMatches(value Value, catchClass string) bool {
	return i.valueInstanceOf(value, catchClass)
}

func (i *Interpreter) defineFunction(fn *ast.FunctionDeclStmt) error {
	if _, exists := i.functions[fn.Name]; exists {
		return fmt.Errorf("cannot redeclare function %s", fn.Name)
	}
	i.functions[fn.Name] = fn
	return nil
}

func (i *Interpreter) defineClass(class *ast.ClassDeclStmt) error {
	if _, exists := i.classes[class.Name]; exists {
		return fmt.Errorf("cannot redeclare class %s", class.Name)
	}
	if class.Final && class.Abstract {
		return fmt.Errorf("class %s cannot be both final and abstract", class.Name)
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
			if parentMethod, ok := parentMethods[method.Name]; ok && parentMethod.Final {
				return fmt.Errorf("cannot override final method %s::%s", parentMethod.Owner, parentMethod.Name)
			}
		}
	}
	i.classes[class.Name] = class
	for _, method := range class.Methods {
		method.Owner = class.Name
		if method.Visibility == "" {
			method.Visibility = "public"
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
			if len(method.Params) != len(ifaceMethod.Params) {
				return fmt.Errorf("class %s method %s must have %d parameters for interface %s", class.Name, name, len(ifaceMethod.Params), ifaceName)
			}
			if method.Visibility != "" && method.Visibility != "public" {
				return fmt.Errorf("class %s interface method %s must be public", class.Name, name)
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

func (i *Interpreter) include(path string, require bool) error {
	resolved := path
	if !filepath.IsAbs(resolved) {
		resolved = filepath.Join(i.currentDir(), path)
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

	i.dirStack = append(i.dirStack, filepath.Dir(resolved))
	err = i.Run(program)
	i.dirStack = i.dirStack[:len(i.dirStack)-1]
	return err
}

func (i *Interpreter) currentDir() string {
	if len(i.dirStack) == 0 {
		return i.baseDir
	}
	return i.dirStack[len(i.dirStack)-1]
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

func (i *Interpreter) SetGlobal(name string, value Value) {
	i.setVar(name, value)
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
