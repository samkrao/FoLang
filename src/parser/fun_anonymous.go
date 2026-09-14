package parser

import (
	"github.com/samkrao/fo-lang/src/ast"
	symboltable "github.com/samkrao/fo-lang/src/context"
	"github.com/samkrao/fo-lang/src/scanlex"
)

// anonymous-function-expression — section 8.
//
//	anonymous-function-expression = parameter-list,
//	                                declaration-return-type-clause, block,
//	                                anonymous-function-binding-context-guard,
//	                                anonymous-function-generic-context-guard
//
// An anonymous function creates a function object only at a binding initializer.
// Once bound, that object can be passed, returned or invoked like any other value.
// The literal itself may also be invoked immediately when the invocation result is
// stored by the same binding (docs/language-ref.md, "Anonymous Functions"):
//
//	add := (a int, b int) -> (int) {
//	    this => a + b;
//	};
//
//	res := (a int, b int) -> (int) {
//	    this => a * b;
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
// expression-brace rule at work. In every permitted form the anonymous function is an
// EXPRESSION, and the enclosing binding statement still needs its ";".

// parseAnonymousFunctionExpression parses the anonymous-function-expression
// production.
//
// Implements: anonymous-function-expression
// Implements: anonymous-function-generic-context-guard
// Implements: anonymous-function-binding-context-guard
func (p *parser) parseAnonymousFunctionExpression() ast.Expr {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	if p.unit == unitEntry {
		p.reportf(p.cur(), "anonymous functions are not allowed in an application entry file")
	}

	symb := p.functionSymbol("anonymous")
	symb.Anonymous = true
	symb.FunctionExpression = true
	symb.IsBody = true
	symb.Closure = true

	// An anonymous function declares no name in the enclosing scope, so the whole
	// expression — type parameters, parameters, results and body — is its context.
	defer p.pushContext(symboltable.S_FunctionSymbol, symb)()

	params := p.parseParameterList(false)
	results := p.parseReturnTypeClause()

	// The body follows the signature directly. A "=" here is the named-function
	// spelling, so it is reported rather than absorbed; the body is still parsed so
	// one stray token does not cascade.
	if p.atOp("=") {
		p.report(p.cur(), "an anonymous function places its body directly after the signature and takes no \"=\"; the \"=\" binding belongs to a named function declaration")
		p.advance()
	}

	body := p.parseScopeBlock("an anonymous function body")

	// Generic names used here must come from the enclosing generic declaration;
	// an anonymous function cannot introduce its own forall binder.
	return ast.FunctionExpr{NodeName: "FunctionExpr", Span: p.spanFrom(spanStart),
		Parameters: params,
		Body:       statementsOf(body),
		ReturnType: results,
		AsExpr:     true,
		Symb:       symb,
	}
}

// parseAnonymousClassExpression parses the anonymous-class-expression production:
//
//	anonymous-class-expression = "co.class", "{", { class-member }, "}"
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
	if kindTok.Value != "co.class" {
		p.failf(kindTok, "expected \"co.class\" to begin an anonymous class expression, found %s", describeToken(kindTok))
	}
	p.advance()
	symb := p.classSymbol("anonymous")
	symb.Anonymous = true

	var members []ast.Stmt
	p.scoped(symboltable.S_ClassSymbol, func() {
		p.expect(scanlex.OPEN_CURLY, "to open an anonymous class expression")
		members = p.parseClassMembers()
		p.expect(scanlex.CLOSE_CURLY, "to close an anonymous class expression")
	}, symb)

	return ast.StatementExpr{NodeName: "StatementExpr", Span: p.spanFrom(spanStart), Statement: ast.ClassDeclarationStmt{NodeName: "ClassDeclarationStmt", Span: p.spanFrom(spanStart), Name: "anonymous",
		Body: members,
		Symb: symb,
	},
		Symb: p.exprSymbol("anonymous-class"),
	}
}
