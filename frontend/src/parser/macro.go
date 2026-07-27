package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// parse_macro_declaration parses a function declaration annotated as a macro.
//
// Feature example:
//
//	@co.dap.macro
//	makeGetter(name co.lang.string)->(co.lang.string) = { ... }
//
// Macros reuse function parsing but are marked with macro-specific symbol and
// scope metadata before being returned as ast.MacroStmt.
func parse_macro_declaration(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	pr := ParseResult{}

	tr := parse_fn_declaration(p, MACRO, ddaps)
	nd := *(tr.Node.(*ast.FunctionDeclarationStmt))
	nd.Symb.RestrictedToOverload = true
	nd.Symb.SymbolDetails.SymbolType_ = string(symboltable.S_MacroSymbol)
	nd.Scope = "MIXED"
	mfn := ast.MacroStmt{}
	mfn.IsExportable = false
	mfn.FunctionDeclarationStmt = nd
	pr.Node = &mfn

	return pr
}
