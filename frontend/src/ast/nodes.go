package ast

import "reflect"

// Stmt_to_Type maps built-in statement kind strings to their reflect.Type.
var Stmt_to_Type map[string]reflect.Type = map[string]reflect.Type{

	"this.break":    reflect.TypeOf(BreakStmt{}),
	"this.continue": reflect.TypeOf(ContinueStmt{}),
	"this.return":   reflect.TypeOf(ReturnStmt{}),
}
