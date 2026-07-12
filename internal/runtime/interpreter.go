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
	Vars       map[string]Value
	functions  map[string]*ast.FunctionDeclStmt
	classes    map[string]*ast.ClassDeclStmt
	interfaces map[string]*ast.InterfaceDeclStmt
	statics    map[string]map[string]Value
	staticMeta map[string]map[string]PropertyMeta
	frames     []frame
	baseDir    string
	dirStack   []string
	Out        bytes.Buffer
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
		functions:  map[string]*ast.FunctionDeclStmt{},
		classes:    map[string]*ast.ClassDeclStmt{},
		interfaces: map[string]*ast.InterfaceDeclStmt{},
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

type frame struct {
	vars        map[string]Value
	className   string
	calledClass string
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
		if err := i.checkPropertyAccess(obj.O, n.Property); err != nil {
			return nil, err
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
		if err := i.checkStaticPropertyAccess(n.ClassName, n.Property); err != nil {
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
		if err := i.checkPropertyAccess(obj.O, n.Property); err != nil {
			return NullValue(), err
		}
		if value, ok := obj.O.Props[n.Property]; ok {
			return value, nil
		}
		return NullValue(), nil
	case *ast.MethodCallExpr:
		return i.callMethod(n)
	case *ast.StaticPropertyFetchExpr:
		if err := i.checkStaticPropertyAccess(n.ClassName, n.Property); err != nil {
			return NullValue(), err
		}
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
	return i.callFunction(fn, c.Args, nil, "")
}

func (i *Interpreter) callFunction(fn *ast.FunctionDeclStmt, args []ast.Expr, thisObj *Object, calledClass string) (Value, error) {
	if fn.Abstract {
		return NullValue(), fmt.Errorf("cannot call abstract method %s::%s", fn.Owner, fn.Name)
	}
	if len(args) != len(fn.Params) {
		return NullValue(), fmt.Errorf("function %s expects %d arguments, got %d", fn.Name, len(fn.Params), len(args))
	}
	if calledClass == "" {
		calledClass = fn.Owner
	}
	callFrame := frame{vars: map[string]Value{}, className: fn.Owner, calledClass: calledClass}
	if thisObj != nil {
		callFrame.vars["this"] = ObjectValue(thisObj)
	}
	for idx, param := range fn.Params {
		arg, err := i.eval(args[idx])
		if err != nil {
			return NullValue(), err
		}
		callFrame.vars[param] = arg
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
		return NullValue(), fmt.Errorf("undefined method %s::%s", obj.O.ClassName, n.Method)
	}
	if err := i.checkMethodAccess(obj.O.ClassName, method); err != nil {
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
	value, ok := i.frames[len(i.frames)-1].vars["this"]
	if !ok || value.Kind != ObjectKind {
		return nil
	}
	return value.O
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
	return i.frames[len(i.frames)-1].vars
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
