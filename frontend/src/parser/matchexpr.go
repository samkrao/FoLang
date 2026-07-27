package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// check_pattern_match upgrades a previously parsed statement expression into a
// pattern-match chain when the parser sees `.match(...)`.
//
// Feature example:
//
//	value.match(co.pattern.Any).case(x: x > 0 => x).default(0)
func check_pattern_match(p *parser, res *ParseResult, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	stex := res.Node.(ast.StatementExpr)
	stmt_ := stex.Statement
	p.advance() //eat dot
	p.advance() //eat match
	pme := ast.PatternExprStmt{}
	pme.Stmt_ = stmt_
	if p.currentTokenKind() == scanlex.OPEN_PAREN {
		// Feature examples:
		//   .match(co.pattern.Type)
		//   .match(co.pattern.Shape)
		//   .match(CustomMatcher)
		p.advance()
		if p.currentToken().Value == "co.pattern.Type" {
			pme.Matcher = true
			pme.MatcherType = "Type"
		} else if p.currentToken().Value == "co.pattern.Shape" {
			pme.Matcher = true
			pme.MatcherType = "Shape"
		} else if p.currentToken().Value == "co.pattern.Value" {
			pme.Matcher = true
			pme.MatcherType = "Value"
		} else if p.currentToken().Value == "co.pattern.Instance" {
			pme.Matcher = true
			pme.MatcherType = "Instance"
		} else if p.currentToken().Value == "co.pattern.Object" {
			pme.Matcher = true
			pme.MatcherType = "Object"
		} else if p.currentToken().Value == "co.pattern.Any" {
			pme.Matcher = true
			pme.MatcherType = "Any"
		} else if p.currentTokenKind() == scanlex.IDENTIFIER {
			pme.Matcher = true
			pme.CustomMatcher = true
			pme.MatcherType = p.currentToken().Value
		}
		p.advance()
		p.expect(scanlex.CLOSE_PAREN)
	}
	if !pme.Matcher {
		pme.MatcherType = "Any"
	}
	return handle_match_expr(p, pme, false, ddaps)

}

// parse_case_arm parses one `.case(...)` or `.default(...)` arm.
//
// Supported forms:
//   - `(binding: guard => body)` — bound variable + guard expression
//   - `(pattern => body)`        — direct value/type pattern
//   - `(body)`                   — used for `.default(...)`
//
// Feature examples:
//
//	.case(v: v > 10 => { this.return v; })
//	.case(co.lang.int => { this.return 1; })
//	.default({ this.return 0; })
func parse_case_arm(p *parser, isDefault bool, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ast.CaseStmt {
	defer p.traceCurrent()()

	_, err_ := p.expect(scanlex.OPEN_PAREN)
	p.addErr(err_)

	arm := ast.CaseStmt{Default: isDefault}

	if isDefault {
		// Feature example:
		//   .default({ this.return 0; })
		//
		// Default arms omit the guard and carry only the fallback body.
		if p.currentTokenKind() == scanlex.OPEN_CURLY {
			body := parse_block_stmt(p, ddaps)
			arm.Stmt_ = body.Node.(ast.Stmt)
		} else {
			expr := parse_expr(p, assignment, ddaps)
			arm.Stmt_ = ast.ExpressionStmt{Expression: expr.Node.(ast.Expr)}
		}
		_, err_ = p.expect(scanlex.CLOSE_PAREN)
		p.addErr(err_)
		return arm
	}

	// Feature example:
	//   .case(v: v > 10 => { this.return v; })
	//
	// The `identifier:` prefix binds the matched value for guard/body use.
	if p.currentTokenKind() == scanlex.IDENTIFIER && p.nextTokenSafe(1).Kind == scanlex.COLON {
		bindingTok := p.advance() // consume binding name
		arm.Binding = bindingTok.Value
		p.advance() // consume ':'

		sd := symboltable.SymbolDetails{
			Name_:       arm.Binding,
			SymbolType_: string(symboltable.S_VarSymbol),
		}
		vd := symboltable.VariableDetails{
			SymbolDetails: sd,
			ExplicitType:  false,
			Inferred:      true,
			SubType_:      "co.lang.var",
			SubID:         "VAR",
			VarType:       GetVarType(VAR),
		}
		vs := symboltable.VarSymbol{
			VariableDetails: vd,
		}
		vstm := ast.BasicVarStmt{
			Identifier: arm.Binding,
		}
		// Register binding in current scope so guard/body can reference it
		varDecl := ast.VarDeclarationStmt{
			BasicVarStmt: vstm,
			Symb:         &vs,
		}
		updateContext(p, varDecl, false, true)
	}

	// Feature examples:
	//   .case(v: v > 10 => ...)
	//   .case(co.lang.int => ...)
	//
	// Everything before `=>` is treated as the case guard or pattern.
	guard := parse_expr(p, assignment, ddaps)
	arm.Expr_ = guard.Node.(ast.Expr)

	// Expect =>
	_, err_ = p.expect(scanlex.EQGT)
	p.addErr(err_)

	// Feature example:
	//   .case(v: v > 10 => { this.return v; })
	//   .case(0 => 1)
	//
	// The arm body can be either a block or a single expression.
	if p.currentTokenKind() == scanlex.OPEN_CURLY {
		body := parse_block_stmt(p, ddaps)
		arm.Stmt_ = body.Node.(ast.Stmt)
	} else {
		expr := parse_expr(p, assignment, ddaps)
		arm.Stmt_ = ast.ExpressionStmt{Expression: expr.Node.(ast.Expr)}
	}

	_, err_ = p.expect(scanlex.CLOSE_PAREN)
	p.addErr(err_)
	return arm
}

// handle_match_expr parses the `.case(...).case(...).default(...)` suffix chain
// attached to a previously created PatternExprStmt.
//
// Feature example:
//
//	value.match(co.pattern.Value)
//	    .case(1 => "one")
//	    .case(2 => "two")
//	    .default("many")
func handle_match_expr(p *parser, ce ast.Stmt, inner bool, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	pr := ParseResult{}
	pes := ce.(ast.PatternExprStmt)

	for p.currentTokenKind() == scanlex.DOT && p.pos < len(p.tokens)-1 {
		p.advance() // consume '.'
		meth := p.currentToken().Value
		p.advance() // consume method name

		if meth == "case" {
			// Feature example:
			//   .case(v: v > 10 => { this.return v; })
			if p.currentTokenKind() == scanlex.OPEN_PAREN {
				arm := parse_case_arm(p, false, ddaps)
				pes.CaseExprStmt = append(pes.CaseExprStmt, arm)
			} else {
				p.addErr(p.errorObj(nil, "Expected '(' after 'case'"))
			}
		} else if meth == "default" {
			// Feature example:
			//   .default({ this.return 0; })
			if p.currentTokenKind() == scanlex.OPEN_PAREN {
				defArm := parse_case_arm(p, true, ddaps)
				pes.DefaultExprStmt = &defArm
			}
			break
		} else {
			break
		}
	}

	pr.Node = pes
	return pr
}
