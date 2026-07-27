package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

func parse_fo_kindtype_decl(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt, kind StmtKind) ParseResult {
	defer p.traceCurrent()()

	pr := ParseResult{}
	switch kind {
	case VARFIELDMEMBERPROP:
		return parse_fo_var_decl_stmt(p, ddaps)
	default:
		return pr
	}
}
