package semantic_analysis

import (
	"reflect"

	"github.com/samkrao/fo-lang/src/ast"
)

var setType = reflect.TypeOf((*ast.SET)(nil)).Elem()

// Walk visits every AST node. It is reflection based only to centralize the
// traversal while the AST remains a large value-oriented sum type. It never
// walks symbol records and skips FunctionDeclarationStmt.Parent, the sole AST
// back-edge.
func Walk(root ast.SET, enter func(ast.SET) bool) {
	var walk func(reflect.Value)
	walk = func(value reflect.Value) {
		if !value.IsValid() {
			return
		}
		for value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer {
			if value.IsNil() {
				return
			}
			value = value.Elem()
		}
		if value.CanInterface() && value.Type().Implements(setType) {
			if !enter(value.Interface().(ast.SET)) {
				return
			}
		}
		switch value.Kind() {
		case reflect.Struct:
			if value.Type().PkgPath() != "github.com/samkrao/fo-lang/src/ast" {
				return
			}
			for i := 0; i < value.NumField(); i++ {
				field := value.Type().Field(i)
				if field.Name == "Parent" || field.PkgPath != "" {
					continue
				}
				walk(value.Field(i))
			}
		case reflect.Slice, reflect.Array:
			for i := 0; i < value.Len(); i++ {
				walk(value.Index(i))
			}
		case reflect.Map:
			iter := value.MapRange()
			for iter.Next() {
				walk(iter.Value())
			}
		}
	}
	walk(reflect.ValueOf(root))
}
