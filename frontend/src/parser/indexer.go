package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// parse_indexer_declaration parses a function annotated with @co.dap.indexer,
// or detected via the `[]` / `[]=` name pattern.
//
//	@co.dap.indexer(symbol="[]")
//	(g MyList) get(index co.lang.int)->(co.lang.int) = {
//	    this.return g.eles[index]
//	}
//
// Entry: current token is the function name IDENTIFIER (or receiver '('.
func parse_indexer_declaration(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	tr := parse_fn_declaration(p, FUNCTION, ddaps)
	if tr.Node == nil {
		return ParseResult{}
	}
	fn := *(tr.Node.(*ast.FunctionDeclarationStmt))
	fn.Symb.RestrictedToOverload = true
	fn.Symb.SymbolDetails.SymbolType_ = string(symboltable.S_Indexer)

	ist := ast.IndexerStmt{}
	ist.FunctionDeclarationStmt = fn
	return ParseResult{Node: &ist, Errors: tr.Errors}
}
