package runtime

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"php-src-go/internal/ast"
	"php-src-go/internal/lexer"
	"php-src-go/internal/parser"
)

type Interpreter struct {
	Vars      map[string]Value
	functions map[string]*ast.FunctionDeclStmt
	classes   map[string]*ast.ClassDeclStmt
	statics   map[string]map[string]Value
	frames    []map[string]Value
	baseDir   string
	dirStack  []string
	Out       bytes.Buffer
}

func NewInterpreter() *Interpreter {
	return NewInterpreterWithBaseDir(".")
}

func NewInterpreterWithBaseDir(baseDir string) *Interpreter {
	if baseDir == "" {
		baseDir = "."
	}
	return &Interpreter{
		Vars:      map[string]Value{},
		functions: map[string]*ast.FunctionDeclStmt{},
		classes:   map[string]*ast.ClassDeclStmt{},
		statics:   map[string]map[string]Value{},
		baseDir:   baseDir,
		dirStack:  []string{baseDir},
	}
}

func (i *Interpreter) Run(p *ast.Program) error {
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
		if _, err := i.exec(s); err != nil {
			return err
		}
	}
	return nil
}

type returnSignal struct {
	value Value
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
			i.Out.WriteString(v.String())
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
			if err != nil || ret != nil {
				return ret, err
			}
			guard++
			if guard > 1000000 {
				return nil, fmt.Errorf("while guard exceeded")
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
			if err != nil || ret != nil {
				return ret, err
			}
		}
	case *ast.FunctionDeclStmt:
		if err := i.defineFunction(n); err != nil {
			return nil, err
		}
	case *ast.ClassDeclStmt:
		if err := i.defineClass(n); err != nil {
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
		obj.O.Props[n.Property] = value
	case *ast.StaticPropertyAssignStmt:
		value, err := i.eval(n.Value)
		if err != nil {
			return nil, err
		}
		i.setStaticProperty(n.ClassName, n.Property, value)
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
		r, err := i.eval(n.Right)
		if err != nil {
			return NullValue(), err
		}
		return Binary(n.Op, l, r)
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
		if value, ok := obj.O.Props[n.Property]; ok {
			return value, nil
		}
		return NullValue(), nil
	case *ast.MethodCallExpr:
		return i.callMethod(n)
	case *ast.StaticPropertyFetchExpr:
		return i.getStaticProperty(n.ClassName, n.Property), nil
	case *ast.StaticCallExpr:
		return i.callStatic(n)
	}
	return NullValue(), fmt.Errorf("unsupported expression %T", e)
}

func (i *Interpreter) call(c *ast.CallExpr) (Value, error) {
	fn, ok := i.functions[c.Name]
	if !ok {
		return NullValue(), fmt.Errorf("undefined function %s", c.Name)
	}
	if len(c.Args) != len(fn.Params) {
		return NullValue(), fmt.Errorf("function %s expects %d arguments, got %d", c.Name, len(fn.Params), len(c.Args))
	}
	return i.callFunction(fn, c.Args, nil)
}

func (i *Interpreter) callFunction(fn *ast.FunctionDeclStmt, args []ast.Expr, thisObj *Object) (Value, error) {
	if len(args) != len(fn.Params) {
		return NullValue(), fmt.Errorf("function %s expects %d arguments, got %d", fn.Name, len(fn.Params), len(args))
	}
	frame := map[string]Value{}
	if thisObj != nil {
		frame["this"] = ObjectValue(thisObj)
	}
	for idx, param := range fn.Params {
		arg, err := i.eval(args[idx])
		if err != nil {
			return NullValue(), err
		}
		frame[param] = arg
	}
	i.frames = append(i.frames, frame)
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

func (i *Interpreter) newObject(n *ast.NewExpr) (Value, error) {
	if n.ClassName == "Exception" {
		return i.newException(n.Args)
	}
	class, ok := i.classes[n.ClassName]
	if !ok {
		return NullValue(), fmt.Errorf("undefined class %s", n.ClassName)
	}
	props, err := i.instanceDefaults(class)
	if err != nil {
		return NullValue(), err
	}
	obj := NewObject(class.Name, i.methodsForClass(class), props)
	if constructor, ok := obj.Methods["__construct"]; ok {
		if _, err := i.callFunction(constructor, n.Args, obj); err != nil {
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
		return NullValue(), fmt.Errorf("undefined method %s::%s", obj.O.ClassName, n.Method)
	}
	return i.callFunction(method, n.Args, obj.O)
}

func (i *Interpreter) callStatic(n *ast.StaticCallExpr) (Value, error) {
	class, ok := i.classes[n.ClassName]
	if !ok {
		return NullValue(), fmt.Errorf("undefined class %s", n.ClassName)
	}
	methods := i.methodsForClass(class)
	method, ok := methods[n.Method]
	if !ok {
		return NullValue(), fmt.Errorf("undefined static method %s::%s", n.ClassName, n.Method)
	}
	if !method.Static {
		return NullValue(), fmt.Errorf("method %s::%s is not static", n.ClassName, n.Method)
	}
	return i.callFunction(method, n.Args, nil)
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
	obj := &Object{ClassName: "Exception", Methods: map[string]*ast.FunctionDeclStmt{}, Props: map[string]Value{"message": message}}
	return ObjectValue(obj), nil
}

func (i *Interpreter) exceptionMatches(value Value, catchClass string) bool {
	if value.Kind != ObjectKind || value.O == nil {
		return false
	}
	return catchClass == value.O.ClassName || catchClass == "Exception"
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
	i.classes[class.Name] = class
	if _, ok := i.statics[class.Name]; !ok {
		i.statics[class.Name] = map[string]Value{}
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
	}
	return nil
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

func (i *Interpreter) instanceDefaults(class *ast.ClassDeclStmt) (map[string]Value, error) {
	props := map[string]Value{}
	if class.Parent != "" {
		parent, ok := i.classes[class.Parent]
		if !ok {
			return nil, fmt.Errorf("undefined parent class %s", class.Parent)
		}
		parentProps, err := i.instanceDefaults(parent)
		if err != nil {
			return nil, err
		}
		for name, value := range parentProps {
			props[name] = value
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
				return nil, err
			}
		}
		props[prop.Name] = value
	}
	return props, nil
}

func (i *Interpreter) getStaticProperty(className, property string) Value {
	if props, ok := i.statics[className]; ok {
		if value, exists := props[property]; exists {
			return value
		}
	}
	if class, ok := i.classes[className]; ok && class.Parent != "" {
		return i.getStaticProperty(class.Parent, property)
	}
	return NullValue()
}

func (i *Interpreter) setStaticProperty(className, property string, value Value) {
	if _, ok := i.statics[className]; !ok {
		i.statics[className] = map[string]Value{}
	}
	i.statics[className][property] = value
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

func (i *Interpreter) currentFrame() map[string]Value {
	if len(i.frames) == 0 {
		return nil
	}
	return i.frames[len(i.frames)-1]
}

func (i *Interpreter) getVar(name string) (Value, bool) {
	if frame := i.currentFrame(); frame != nil {
		v, ok := frame[name]
		return v, ok
	}
	v, ok := i.Vars[name]
	return v, ok
}

func (i *Interpreter) setVar(name string, v Value) {
	if frame := i.currentFrame(); frame != nil {
		frame[name] = v
		return
	}
	i.Vars[name] = v
}
