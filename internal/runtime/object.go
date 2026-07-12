package runtime

import "php-src-go/internal/ast"

type Object struct {
	ClassName string
	Methods   map[string]*ast.FunctionDeclStmt
	Props     map[string]Value
}

func NewObject(className string, methods map[string]*ast.FunctionDeclStmt, props map[string]Value) *Object {
	if props == nil {
		props = map[string]Value{}
	}
	return &Object{ClassName: className, Methods: methods, Props: props}
}
