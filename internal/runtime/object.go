package runtime

import "php-src-go/internal/ast"

type Object struct {
	ClassName      string
	Methods        map[string]*ast.FunctionDeclStmt
	Props          map[string]Value
	PropMeta       map[string]PropertyMeta
	ExceptionTrace []string
}

type PropertyMeta struct {
	Owner      string
	Visibility string
	Static     bool
}

func NewObject(className string, methods map[string]*ast.FunctionDeclStmt, props map[string]Value, meta map[string]PropertyMeta) *Object {
	if props == nil {
		props = map[string]Value{}
	}
	if meta == nil {
		meta = map[string]PropertyMeta{}
	}
	return &Object{ClassName: className, Methods: methods, Props: props, PropMeta: meta}
}
