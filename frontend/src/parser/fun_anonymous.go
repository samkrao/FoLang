package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// anonymous-function-expression — section 8.
//
//	anonymous-function-expression = [ "forall", "(", type-parameter-list, ")", "." ],
//	                                parameter-list, return-type-clause, block
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
// An anonymous function places its body DIRECTLY after the signature, with no "="
// between them (docs/grammar/folang.ebnf, preamble). That is the one thing separating
// the literal from a named function-definition, which requires the "=", so accepting an
// optional "=" here erased the distinction the grammar draws.
//
// Note the two different terminators in the examples above, which is the
// expression-brace rule at work. Bound to a name with ":=", the anonymous function is an
// EXPRESSION and its statement still needs a ";". Used as the direct inline body of a
// function-kind declaration, the same syntax is a BODY and takes none — that case is
// selected by startsAnonymousFunction and handled by decl_functionobject.go.

// parseAnonymousFunctionExpression parses the anonymous-function-expression
// production.
//
// Implements: anonymous-function-expression
func (p *parser) parseAnonymousFunctionExpression() ast.Expr {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	if p.unit == unitEntry {
		p.reportf(p.cur(), "anonymous functions are not allowed in an application entry file")
	}

	// An anonymous function declares no name in the enclosing scope, so the whole
	// expression — type parameters, parameters, results and body — is its context.
	defer p.pushContext(symboltable.S_FunctionSymbol)()

	// The optional forall prefix makes the anonymous function polymorphic.
	var typeParams []symboltable.GenericTypeParam
	if p.atKeyword("forall") {
		p.advance()
		p.expect(scanlex.OPEN_PAREN, "to open the type-parameter list of an anonymous function")
		typeParams = p.parseTypeParameterList()
		p.expect(scanlex.CLOSE_PAREN, "to close the type-parameter list of an anonymous function")
		p.expect(scanlex.DOT, "after the type-parameter list of an anonymous function")
	}

	params := p.parseParameterList()
	results := p.parseReturnTypeClause()

	// The body follows the signature directly. A "=" here is the named-function
	// spelling, so it is reported rather than absorbed; the body is still parsed so
	// one stray token does not cascade.
	if p.atOp("=") {
		p.report(p.cur(), "an anonymous function places its body directly after the signature and takes no \"=\"; the \"=\" binding belongs to a named function declaration")
		p.advance()
	}

	body := p.parseScopeBlock("an anonymous function body")

	symb := p.functionSymbol("anonymous")
	symb.Anonymous = true
	symb.FunctionExpression = true
	symb.IsBody = true
	symb.IsGeneric = len(typeParams) > 0
	symb.Closure = true

	return ast.FunctionExpr{Span: p.spanFrom(spanStart), TypeParams: typeParams,
		Parameters: params,
		Body:       statementsOf(body),
		ReturnType: results,
		AsExpr:     true,
		Symb:       symb,
	}
}

// closure-declaration — section 8.
//
//	closure-declaration = annotations, identifier, "=", parameter-list,
//	                      { parameter-list }, "==>>", expression,
//	                      statement-end
//
// This is the abbreviated closure form of docs/language-ref.md, "Other ways to
// declare closures/function objects and types/curried functions":
//
//	closure = (factor int, val int) ==>> factor * val;
//	curry   = (factor int)(val int) ==>> factor * val;
//
// DECISION-FUN-002: the "=" makes this a NAMED closure declaration and "==>>"
// introduces its expression body. One parameter list declares an ordinary closure and
// two or more declare a curried one, so currying is a property of the parameter lists
// rather than of a separate production.
//
// "==>>" is what keeps this production apart from everything else an identifier can
// begin. `result = compute(x);` is an ordinary assignment because no "==>>" follows the
// parenthesised group, and the marker is distinct from "=>", which introduces lambdas
// and bare function-pattern clauses, and from "=>>", which delegates.

// atClosureDeclaration reports whether the cursor begins a closure-declaration.
func (p *parser) atClosureDeclaration() bool {
	if !p.atIdentifier() {
		return false
	}
	return p.lookaheadOnly(func() bool {
		p.advance() // the name
		if !p.acceptOp("=") {
			return false
		}
		if !p.at(scanlex.OPEN_PAREN) {
			return false
		}
		// One or more parameter lists; two or more make the closure curried.
		for p.at(scanlex.OPEN_PAREN) {
			p.skipBalanced(scanlex.OPEN_PAREN, scanlex.CLOSE_PAREN)
		}
		// The marker is what selects this production. Without it the same prefix
		// is an ordinary assignment to a call result.
		return p.at(scanlex.EQEQGTGT)
	})
}

// parseClosureDeclaration parses the closure-declaration production.
//
// The parameter lists follow the "=", and "==>>" introduces the expression body:
//
//	closure = (factor int, val int) ==>> factor * val;   one list, ordinary closure
//	curry   = (factor int)(val int) ==>> factor * val;   two lists, curried closure
//
// Implements: closure-declaration
func (p *parser) parseClosureDeclaration(annotations annotationSet) ast.Stmt {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	closureName := p.parseIdentifier("as a closure name")
	p.expectOp("=", "before the parameter lists of a closure declaration")

	// The closure's name is declared where the statement is written; its parameters
	// and its body are its own scope. A closure body is an expression rather than a
	// braced block, but the scope it needs is the same one a block body would open.
	symb := p.functionSymbol(closureName.Scanned)

	var lists [][]ast.Parameter
	var body ast.Expr
	var bodySymb *symboltable.StatmentSymbol
	p.scoped(symboltable.S_FunctionSymbol, func() {
		// DECISION-FUN-002: one list is an ordinary closure, two or more are curried.
		lists = p.parseParameterLists()

		p.expectOp("==>>", "before the body of a closure declaration")
		body = p.parseExpression()
		bodySymb = p.stmtSymbol("closure-body")
	})
	p.statementEnd("a closure declaration")

	symb.Closure = true
	symb.Curried = len(lists) > 1
	symb.IsBody = true

	decl := ast.FunctionDeclarationStmt{Span: p.spanFrom(spanStart), Parameters: lists,
		Name: closureName.Scanned,
		Body: []ast.Stmt{
			ast.ExpressionStmt{Span: p.spanFrom(spanStart), Expression: body, Symb: bodySymb},
		},
		Dapst: annotations.list(),
		Symb:  symb,
	}
	p.applyFunctionFlags(&decl, annotations)
	decl.Symb.Closure = true
	p.declareFunction(closureName.Tok, &decl)
	return decl
}

// parseAnonymousClassExpression parses the anonymous-class-expression production:
//
//	anonymous-class-expression = "co.lang.class", "{", { class-member }, "}"
//
// This is a class written inline as a value. Its closing brace ends an EXPRESSION, so
// the enclosing statement still needs its terminator (DECISION-SYN-006).
//
// Implements: anonymous-class-expression
func (p *parser) parseAnonymousClassExpression() ast.Expr {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	kindTok := p.cur()
	if kindTok.Value != "co.lang.class" {
		p.failf(kindTok, "expected \"co.lang.class\" to begin an anonymous class expression, found %s", describeToken(kindTok))
	}
	p.advance()

	var members []ast.Stmt
	p.scoped(symboltable.S_ClassSymbol, func() {
		p.expect(scanlex.OPEN_CURLY, "to open an anonymous class expression")
		members = p.parseClassMembers()
		p.expect(scanlex.CLOSE_CURLY, "to close an anonymous class expression")
	})

	symb := p.classSymbol("anonymous")
	symb.Anonymous = true

	return ast.StatementExpr{Span: p.spanFrom(spanStart), Statement: ast.ClassDeclarationStmt{Span: p.spanFrom(spanStart), Name: "anonymous",
		Body: members,
		Symb: symb,
	},
		Symb: p.exprSymbol("anonymous-class"),
	}
}
