package parser

import (
	"strings"

	"github.com/samkrao/fo-lang/frontend/src/ast"
	"github.com/samkrao/fo-lang/frontend/src/helpers"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// parse_atdap parses one leading built-in directive/decorator/pragma/annotation
// token and dispatches to the shared directive parser when the token matches
// the requested directive kind.
//
// Feature examples:
//
//	@co.dap.generic(typename=T)
//	@co.ddap.import(package="hr.employee", as="emp")
//
// The token under the cursor has already been recognized by the scanner as a
// built-in FoLang directive-like token.
func parse_atdap(p *parser, kind scanlex.DirectiveKind, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	tk := p.currentToken()
	dap := strings.TrimSuffix(tk.Value, "_fo")

	actualKind, ok := scanlex.Built_in_directive_kind(dap)
	if !ok {
		err_ := p.errorExpection("Invalid directive/decorator/pragma/annotation "+dap, helpers.UnSupported)
		p.addErr(err_)
		// Still consume the offending directive's tokens (generically, as an
		// unknown kind) so the parser advances past it instead of looping on
		// the same token forever.
		res := parse_ddaps(p, scanlex.Invalid, ddaps)
		res.Empty = true
		return res
	}

	if actualKind != kind {
		err_ := p.errorExpection("Mismatched directive/decorator/pragma/annotation "+dap+": found "+scanlex.KindToString[actualKind]+", expected "+scanlex.KindToString[kind], helpers.UnSupported)
		p.addErr(err_)
		// Parse it under its real kind so its tokens are consumed correctly,
		// but exclude it from the caller's result set since it doesn't match
		// the requested kind. This lets collection continue over the rest of
		// the directives instead of aborting on the first mismatch.
		res := parse_ddaps(p, actualKind, ddaps)
		res.Empty = true
		return res
	}

	return parse_ddaps(p, kind, ddaps)
}

// parse_ddaps parses the parameter list for a FoLang directive-like construct
// and stores the result as an ast.DirectiveStmt.
//
// Feature examples:
//
//	@co.dap.generic(typename=T)
//	@co.ddap.import(package="hr.employee", as="emp", realm="app")
//
// Parameter values are normalized into `map[string]any` entries so later
// parser stages can inspect annotations without reparsing tokens.
func parse_ddaps(p *parser, kind scanlex.DirectiveKind, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	pr := ParseResult{}
	tk := p.advance()

	nod := ast.DirectiveStmt{}
	nod.Name = strings.TrimSuffix(tk.Value, "_fo")
	dapType := kind
	params := map[string]any{}
	if p.currentTokenKind() == scanlex.OPEN_PAREN {
		tk = p.advance()
		for p.currentTokenKind() != scanlex.CLOSE_PAREN {
			tk = p.advance()
			key := strings.TrimSuffix(tk.Value, "_fo")
			_, err := p.expect(scanlex.ASSIGNMENT)
			if err != nil {
				p.addErr(err)
			}
			tk = p.advance()
			val := strings.TrimSuffix(tk.Value, "_fo")
			vals := strings.Split(val, ",")
			anyVals := make([]any, len(vals))
			for i, v := range vals {
				tempV := strings.Trim(v, "\"")
				if strings.TrimSpace(tempV) == "" {
					anyVals[i] = "_"
				} else {
					anyVals[i] = tempV
				}
			}
			params[key] = anyVals
			if p.currentTokenKind() != scanlex.CLOSE_PAREN {
				tk, err = p.expect(scanlex.COMMA)

				p.addErr(err)
			}
		}
		_, err := p.expect(scanlex.CLOSE_PAREN)
		p.addErr(err)
	}

	nod.DirectiveKind_ = scanlex.KindToString[dapType]
	//DDAPS, _ := scanlex.Built_in_directives(dap)

	nod.Parameters = params
	nod.DirectiveType = scanlex.KindToPhase[dapType]
	nod.DirectiveScope_ = scanlex.KindToScope[dapType]
	pr.Node = &nod

	return pr
}
