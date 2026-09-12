package ast

import "reflect"

// Stmt_to_Type maps built-in statement kind strings to their reflect.Type.
var Stmt_to_Type map[string]reflect.Type = map[string]reflect.Type{

	"this ->|": reflect.TypeOf(BreakStmt{NodeName: "BreakStmt"}),
	"this ->":  reflect.TypeOf(ContinueStmt{NodeName: "ContinueStmt"}),
	"this =>":  reflect.TypeOf(ReturnStmt{NodeName: "ReturnStmt"}),
}
