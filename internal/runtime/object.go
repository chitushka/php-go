package runtime

import "php-src-go/internal/ast"

type Object struct {
	ID             int
	ClassName      string
	Methods        map[string]*ast.FunctionDeclStmt
	Props          map[string]Value
	PropMeta       map[string]PropertyMeta
	ExceptionTrace []ExceptionTraceFrame
}

// objectIDCounter backs the per-object handle number shown by var_dump as
// `object(ClassName)#N`. Unlike real PHP, handles are never reused after an
// object becomes unreachable (this interpreter relies on Go's GC, which
// doesn't expose collection order) — ids are simply monotonically
// increasing for the process lifetime.
var objectIDCounter int

func nextObjectID() int {
	objectIDCounter++
	return objectIDCounter
}

type ExceptionTraceFrame struct {
	Function string
	Class    string
	Type     string
}

type PropertyMeta struct {
	Owner      string
	Name       string
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
	return &Object{ID: nextObjectID(), ClassName: className, Methods: methods, Props: props, PropMeta: meta}
}
