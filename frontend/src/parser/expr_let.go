package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// let-expression — section 11.
//
//	let-expression = "let", "(", "{", let-binding,
//	                 { ",", let-binding }, "}", ")",
//	                 ".in", "(", "{", expression, "}", ")"
//	let-binding    = ( identifier | special-binding ), "=", expression
//
// The braces inside the parentheses are part of the spelling, not a block
// (docs/language-ref.md, "Let Bindings"):
//
//	y co.lang.int = let({x = 10}).in({x + 1});
//	y co.lang.int = let({$ = 10}).in({$ + 1});
//
// "$" is the self-referential binding, which is what makes a recursive let
// expressible without naming the value being defined.
//
// The reference also documents an equivalent postfix `.where` form:
//
//	x co.lang.int = (x + 1).where(x = 10);
//	x co.lang.int = ($ + 1).where($ = 10);
//
// That one needs no production of its own: `.where` is an ordinary member and call
// suffix, so the postfix chain already parses it. parseWhereSuffix below exists only
// to record it as a let expression rather than as an anonymous call, so both
// spellings reach the semantic phase as the same node.

// parseLetExpression parses the let-expression production.
func (p *parser) parseLetExpression() ast.Expr {
	p.expectKeyword("let", "to begin a let expression")

	p.expect(scanlex.OPEN_PAREN, "to open the bindings of a let expression")
	p.expect(scanlex.OPEN_CURLY, "to open the bindings of a let expression")

	bindings := []ast.Stmt{p.parseLetBinding()}
	for p.accept(scanlex.COMMA) {
		if p.at(scanlex.CLOSE_CURLY) {
			break
		}
		bindings = append(bindings, p.parseLetBinding())
	}

	p.expect(scanlex.CLOSE_CURLY, "to close the bindings of a let expression")
	p.expect(scanlex.CLOSE_PAREN, "to close the bindings of a let expression")

	// ".in" is spelled as a member access, so the "." and the name are consumed
	// separately.
	p.expect(scanlex.DOT, "before \"in\" in a let expression")
	p.expectMemberName("in", "to introduce the body of a let expression")

	p.expect(scanlex.OPEN_PAREN, "to open the body of a let expression")
	p.expect(scanlex.OPEN_CURLY, "to open the body of a let expression")
	body := p.parseExpression()
	p.expect(scanlex.CLOSE_CURLY, "to close the body of a let expression")
	p.expect(scanlex.CLOSE_PAREN, "to close the body of a let expression")

	return ast.LetExpr{
		Stmt_: &ast.BlockStmt{
			Body: bindings,
			Symb: p.blockSymbol("let-bindings", false),
		},
		Expr_: body,
		Type_: ast.IN,
		Symb:  p.letSymbol("let"),
	}
}

// parseLetBinding parses the let-binding production:
//
//	let-binding = ( identifier | special-binding ), "=", expression
func (p *parser) parseLetBinding() ast.Stmt {
	var boundName string

	if p.at(scanlex.BIND_VAR) {
		boundName = p.cur().Value
		p.advance()
	} else {
		boundName = p.parseIdentifier("as a let binding name").Scanned
	}

	p.expectOp("=", "in a let binding")
	value := p.parseExpression()

	return ast.VarDeclarationStmt{
		BasicVarStmt: ast.BasicVarStmt{
			Identifier:    boundName,
			AssignedValue: value,
			VarType:       "let",
		},
		Symb: p.letBoundVarSymbol(boundName),
	}
}

// letBoundVarSymbol mints the symbol for a let-bound name. The binding is local and
// its type is inferred from the bound expression.
func (p *parser) letBoundVarSymbol(name string) *symboltable.VarSymbol {
	s := p.varSymbol(name, "co.lang.infer")
	s.LocalBinding = true
	s.Inferred = true
	s.HasInitValue = true
	return s
}

// parseWhereSuffix records the postfix `.where(binding)` form as a let expression.
//
// It is called from the postfix chain once `.where` has been recognised, with subject
// being the expression to its left. The result is an ast.LetExpr in WHERE form, so
// that `(x + 1).where(x = 10)` and `let({x = 10}).in({x + 1})` produce the same
// shape.
func (p *parser) parseWhereSuffix(subject ast.Expr) ast.Expr {
	p.expect(scanlex.OPEN_PAREN, "to open a where clause")

	bindings := []ast.Stmt{p.parseLetBinding()}
	for p.accept(scanlex.COMMA) {
		bindings = append(bindings, p.parseLetBinding())
	}

	p.expect(scanlex.CLOSE_PAREN, "to close a where clause")

	return ast.LetExpr{
		Stmt_: &ast.BlockStmt{
			Body: bindings,
			Symb: p.blockSymbol("where-bindings", false),
		},
		Expr_: subject,
		Type_: ast.WHERE,
		Symb:  p.letSymbol("where"),
	}
}

// expectMemberName consumes a member name with the given logical spelling.
//
// It compares against the logical name, because a member that follows ")" or "}"
// reaches the parser as a bare identifier carrying the "_fo" backend suffix, while
// one that follows a name may have been folded into a reserved-method token
// instead. Both spell the same member.
func (p *parser) expectMemberName(want string, context string) scanlex.Token {
	if p.isMemberNameToken(p.cur()) && logicalName(p.lexeme()) == want {
		return p.advance()
	}
	p.failf(p.cur(), "expected %q %s, found %s", want, context, describeToken(p.cur()))
	return eofToken // unreachable: failf panics
}

// atMemberName reports whether the cursor holds a member name with the given
// logical spelling.
func (p *parser) atMemberName(want string) bool {
	return p.isMemberNameToken(p.cur()) && logicalName(p.lexeme()) == want
}
