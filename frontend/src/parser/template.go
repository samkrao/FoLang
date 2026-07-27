package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

func parse_template_declaration(p *parser, name string, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()
	pr := ParseResult{}

	tr := parse_fn_declaration(p, TEMPLATE, ddaps)
	nd := *(tr.Node.(*ast.FunctionDeclarationStmt))
	nd.Scope = "DYNAMIC" //variables avaialbility check should be at call site
	nd.Symb.RestrictedToOverload = true
	nd.Symb.IsExportable = false
	nd.Symb.SymbolDetails.SymbolType_ = string(symboltable.S_TemplateDetails)
	if name == "@co.dap.inline" {
		nd.Symb.Inline = true
	}
	tfn := ast.TemplateStmt{}
	tfn.FunctionDeclarationStmt = nd
	pr.Node = &tfn

	return pr
}
