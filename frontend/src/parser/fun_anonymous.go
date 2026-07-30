package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// anonymous-function-expression — section 8.
//
//	anonymous-function-expression = [ "forall", "(", type-parameter-list, ")", "." ],
//	                                parameter-list, return-type-clause,
//	                                [ "=" ], block
//
// An anonymous function is an expression, so it can be bound, passed, returned or
// called immediately (docs/language-ref.md, "Anonymous Functions"):
//
//	add := (a int, b int) -> (int) {
//	    this.return a + b;
//	};
//
//	res := (a int, b int) -> (int) {
//	    this.return a * b;
//	}(10, 20);
//
// The immediately-invoked form needs no special handling: the trailing "(10, 20)" is an
// ordinary call suffix that the postfix chain absorbs.
//
// Note the two different terminators in the examples above, which is the
// expression-brace rule of DECISION-SYN-006 at work. Bound to a name with ":=", the
// anonymous function is an EXPRESSION and its statement still needs a ";". Used as the
// direct inline body of a function-kind declaration, the same syntax is a BODY and
// takes none — that case is selected by startsAnonymousFunction and handled by
// decl_functionobject.go.

// parseAnonymousFunctionExpression parses the anonymous-function-expression
// production.
func (p *parser) parseAnonymousFunctionExpression() ast.Expr {
	// The optional forall prefix makes the anonymous function polymorphic.
	var typeParams []string
	if p.atKeyword("forall") {
		p.advance()
		p.expect(scanlex.OPEN_PAREN, "to open the type-parameter list of an anonymous function")
		for _, tp := range p.parseTypeParameterList() {
			typeParams = append(typeParams, tp.Name)
		}
		p.expect(scanlex.CLOSE_PAREN, "to close the type-parameter list of an anonymous function")
		p.expect(scanlex.DOT, "after the type-parameter list of an anonymous function")
	}

	params := p.parseParameterList()
	results := p.parseReturnTypeClause()

	// DECISION-FUN-001: the "=" before the block body is optional.
	p.acceptOp("=")

	body := p.parseBlock("an anonymous function body")

	symb := p.functionSymbol("anonymous")
	symb.Anonymous = true
	symb.FunctionExpression = true
	symb.IsBody = true
	symb.IsGeneric = len(typeParams) > 0
	symb.Closure = true

	return ast.FunctionExpr{
		Parameters: params,
		Body:       statementsOf(body),
		ReturnType: results,
		AsExpr:     true,
		Symb:       symb,
	}
}

// closure-declaration — section 8.
//
//	closure-declaration         = curried-closure-declaration
//	                            | arrow-closure-declaration
//	curried-closure-declaration = annotations, identifier, parameter-list,
//	                              parameter-list, { parameter-list },
//	                              [ return-type-clause ], "=", expression,
//	                              statement-end
//	arrow-closure-declaration   = annotations, identifier, parameter-list,
//	                              "=>", parameter-list, "=", expression,
//	                              statement-end
//
// These are the two abbreviated closure forms of docs/language-ref.md, "Other ways to
// declare closures/function objects and types/curried functions":
//
//	closure(factor int) => (x int) = x * factor;
//	curry(factor int)(val int) = factor * val;
//
// DECISION-FUN-002 requires TWO OR MORE parameter lists for the curried form, which is
// what keeps an ordinary assignment to a call result — `total = compute(x) = …` would
// be malformed anyway, but `result = compute(x);` must not be captured — outside this
// production.

// atClosureDeclaration reports whether the cursor begins a closure-declaration.
func (p *parser) atClosureDeclaration() bool {
	if !p.atIdentifier() {
		return false
	}
	return p.lookaheadOnly(func() bool {
		p.advance() // the name
		if !p.at(scanlex.OPEN_PAREN) {
			return false
		}
		p.skipBalanced(scanlex.OPEN_PAREN, scanlex.CLOSE_PAREN)

		// The arrow form: one parameter list, "=>", then a second list.
		if p.atOp("=>") {
			p.advance()
			return p.at(scanlex.OPEN_PAREN)
		}

		// The curried form: at least two parameter lists, then "=" and an
		// expression rather than a block.
		if !p.at(scanlex.OPEN_PAREN) {
			return false
		}
		for p.at(scanlex.OPEN_PAREN) {
			p.skipBalanced(scanlex.OPEN_PAREN, scanlex.CLOSE_PAREN)
		}
		if p.at(scanlex.ARROW) {
			p.advance()
			if !p.at(scanlex.OPEN_PAREN) {
				return false
			}
			p.skipBalanced(scanlex.OPEN_PAREN, scanlex.CLOSE_PAREN)
		}
		// An expression body distinguishes this from a curried function
		// declaration, which has a block body and is parsed as a
		// function-declaration instead.
		return p.atOp("=") && !p.lookaheadOnly(func() bool {
			p.advance()
			return p.at(scanlex.OPEN_CURLY)
		})
	})
}

// parseClosureDeclaration parses the closure-declaration production, dispatching to
// the arrow or the curried form.
func (p *parser) parseClosureDeclaration(annotations annotationSet) ast.Stmt {
	closureName := p.parseIdentifier("as a closure name")
	first := p.parseParameterList()

	if p.atOp("=>") {
		return p.parseArrowClosureDeclaration(closureName, first, annotations)
	}
	return p.parseCurriedClosureDeclaration(closureName, first, annotations)
}

// parseArrowClosureDeclaration parses the arrow-closure-declaration production.
//
// The first parameter list is the captured environment and the second the closure's
// own parameters:
//
//	closure(factor int) => (x int) = x * factor;
func (p *parser) parseArrowClosureDeclaration(closureName name, captured []ast.Parameter, annotations annotationSet) ast.Stmt {
	p.expectOp("=>", "between the capture list and the parameters of an arrow closure")
	params := p.parseParameterList()

	var results []ast.Returns
	if p.at(scanlex.ARROW) {
		results = p.parseReturnTypeClause()
	}

	p.expectOp("=", "before the body of an arrow closure")
	body := p.parseExpression()
	p.statementEnd("an arrow closure declaration")

	symb := p.functionSymbol(closureName.Scanned)
	symb.Closure = true
	symb.IsBody = true

	decl := ast.FunctionDeclarationStmt{
		Parameters: [][]ast.Parameter{captured, params},
		Name:       closureName.Scanned,
		ReturnType: results,
		Body: []ast.Stmt{
			ast.ExpressionStmt{Expression: body, Symb: p.stmtSymbol("closure-body")},
		},
		Dapst: annotations.list(),
		Symb:  symb,
	}
	p.applyFunctionFlags(&decl, annotations)
	decl.Symb.Closure = true
	return decl
}

// parseCurriedClosureDeclaration parses the curried-closure-declaration production.
//
// Two or more parameter lists are required by DECISION-FUN-002; the first has already
// been read by the caller:
//
//	curry(factor int)(val int) = factor * val;
func (p *parser) parseCurriedClosureDeclaration(closureName name, first []ast.Parameter, annotations annotationSet) ast.Stmt {
	lists := [][]ast.Parameter{first}
	for p.at(scanlex.OPEN_PAREN) {
		lists = append(lists, p.parseParameterList())
	}

	if len(lists) < 2 {
		p.failf(closureName.Tok, "a curried closure declaration needs two or more parameter lists; %q has one", closureName.Logical)
	}

	var results []ast.Returns
	if p.at(scanlex.ARROW) {
		results = p.parseReturnTypeClause()
	}

	p.expectOp("=", "before the body of a curried closure")
	body := p.parseExpression()
	p.statementEnd("a curried closure declaration")

	symb := p.functionSymbol(closureName.Scanned)
	symb.Curried = true
	symb.Closure = true
	symb.IsBody = true

	decl := ast.FunctionDeclarationStmt{
		Parameters: lists,
		Name:       closureName.Scanned,
		ReturnType: results,
		Body: []ast.Stmt{
			ast.ExpressionStmt{Expression: body, Symb: p.stmtSymbol("closure-body")},
		},
		Dapst: annotations.list(),
		Symb:  symb,
	}
	p.applyFunctionFlags(&decl, annotations)
	decl.Symb.Closure = true
	return decl
}

// parseAnonymousClassExpression parses the anonymous-class-expression production:
//
//	anonymous-class-expression = "co.lang.class", "{", { class-member }, "}"
//
// This is a class written inline as a value. Its closing brace ends an EXPRESSION, so
// the enclosing statement still needs its terminator (DECISION-SYN-006).
func (p *parser) parseAnonymousClassExpression() ast.Expr {
	kindTok := p.cur()
	if kindTok.Value != "co.lang.class" {
		p.failf(kindTok, "expected \"co.lang.class\" to begin an anonymous class expression, found %s", describeToken(kindTok))
	}
	p.advance()

	p.expect(scanlex.OPEN_CURLY, "to open an anonymous class expression")
	members := p.parseClassMembers()
	p.expect(scanlex.CLOSE_CURLY, "to close an anonymous class expression")

	symb := p.classSymbol("anonymous")
	symb.Anonymous = true

	return ast.StatementExpr{
		Statement: ast.ClassDeclarationStmt{
			Name: "anonymous",
			Body: members,
			Symb: symb,
		},
		Symb: p.exprSymbol("anonymous-class"),
	}
}
