package runtime

import "php-src-go/internal/ast"

type Object struct {
	ClassName string
	Methods   map[string]*ast.FunctionDeclStmt
	Props     map[string]Value
}

func NewObject(class *ast.ClassDeclStmt) *Object {
	methods := map[string]*ast.FunctionDeclStmt{}
	for _, method := range class.Methods {
		methods[method.Name] = method
	}
	return &Object{ClassName: class.Name, Methods: methods, Props: map[string]Value{}}
}

func methodsByName(class *ast.ClassDeclStmt) map[string]*ast.FunctionDeclStmt {
	methods := map[string]*ast.FunctionDeclStmt{}
	for _, method := range class.Methods {
		methods[method.Name] = method
	}
	return methods
}
