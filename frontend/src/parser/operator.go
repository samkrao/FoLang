package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// parse_operator_function parses a function annotated with @co.dap.operator.
//
//	@co.dap.operator(symbol='+', mode=overload)
//	add(a Employee, b Employee)->(Employee) = {}
//
// Entry: current token is the function name IDENTIFIER (or receiver '(').
func parse_operator_function(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	tr := parse_fn_declaration(p, FUNCTION, ddaps)
	if tr.Node == nil {
		return ParseResult{}
	}
	fn := *(tr.Node.(*ast.FunctionDeclarationStmt))
	fn.Symb.IsOperator = true

	ost := ast.OperatorStmt{}
	fn.Symb.SymbolDetails.SymbolType_ = string(symboltable.S_OperatorDetails)
	ost.FunctionDeclarationStmt = fn
	return ParseResult{Node: &ost, Errors: tr.Errors}
}
