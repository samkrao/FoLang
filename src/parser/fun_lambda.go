package parser

import (
	"github.com/samkrao/fo-lang/src/ast"
	symboltable "github.com/samkrao/fo-lang/src/context"
	"github.com/samkrao/fo-lang/src/scanlex"
)

// lambda-expression — section 8.
//
//	lambda-expression = "|", [ lambda-parameter, { ",", lambda-parameter } ],
//	                    "|", "=>", ( expression | block )
//	lambda-parameter  = identifier, [ type-expression ]
//
// A lambda is only permitted as an inline callback argument to a collection
// operation — map, filter, reduce, forEach, sortBy, groupBy
// (docs/language-ref.md, "Lambda"):
//
//	nums.map(|x| => x*x)
//	words.filter(|s| => s.len() > 3)
//	pairs.reduce(|acc, e| => acc + e, 0)
//	list.sortBy(|a, b| => a.score - b.score)
//
// Using "|...|" anywhere else is an error. The call parser records the immediate
// target while reading its arguments, allowing this parser to reject standalone,
// nested and non-collection lambdas without guessing from the lambda's shape.
//
// The delimiter is the same "|" that spells bitwise OR and type union. There is no
// ambiguity in practice, because a lambda only ever appears in operand position and
// the operators only in infix position, which is the ordinary Pratt split.

// parseLambdaExpression parses a lambda found outside a direct call-argument
// branch. Such a lambda is always rejected: the release profile admits lambdas
// only as direct callbacks of the closed collection-operation set.
//
// Implements: lambda-expression
func (p *parser) parseLambdaExpression() ast.Expr {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	return p.parseLambdaExpressionWithPermission(false)
}

// parseLambdaExpressionWithPermission parses the lambda-expression production.
//
// allowed describes this exact argument position rather than an enclosing
// expression. Keeping the permission lexical prevents a legal outer
// `items.map(|x| => ...)` callback from leaking into `helper(|y| => ...)` inside
// its body. A nested collection callback remains legal because its own call
// supplies allowed=true.
func (p *parser) parseLambdaExpressionWithPermission(allowed bool) ast.Expr {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	if !allowed {
		p.reportf(p.cur(), "a lambda is only allowed as a direct callback argument to each, map, filter, reduce, forEach, sortBy, or groupBy")
	}

	symb := p.lambdaSymbol("lambda")
	// A lambda's parameters and body are one scope, so the context opens at the
	// parameter list rather than at a body brace the expression form does not have.
	defer p.pushContext(symboltable.S_LambdaSymbol, symb)()

	p.expectOp("|", "to open a lambda parameter list")

	// The parameter list is delimited by the same "|" that spells the type-union operator,
	// so union parsing is suppressed while the list is being read.
	p.lambdaParamDepth++
	var params []ast.Parameter
	if !p.atOp("|") {
		params = append(params, p.parseLambdaParameter())
		for p.accept(scanlex.COMMA) {
			params = append(params, p.parseLambdaParameter())
		}
	}
	p.lambdaParamDepth--

	p.expectOp("|", "to close a lambda parameter list")
	p.expectOp("=>", "before a lambda body")

	// The body is a single expression or a block. A block body is wrapped so the
	// node's single-expression Body field can carry it.
	var body ast.Expr
	if p.at(scanlex.OPEN_CURLY) {
		block := p.parseScopeBlock("a lambda body")
		body = ast.StatementExpr{NodeName: "StatementExpr", Span: p.spanFrom(spanStart), Statement: block, Symb: p.exprSymbol("lambda-body")}
	} else {
		body = p.parseExpression()
	}

	return ast.LambdaExpr{NodeName: "LambdaExpr", Span: p.spanFrom(spanStart), Parameters: params,
		Body: body,
		Symb: symb,
	}
}

// pushLambdaCallContext records whether the call whose arguments are about to
// be parsed is one of the collection operations that permits lambdas. Callers
// must invoke the returned cleanup after the argument list, including on nested
// calls, so the top of the stack always describes the immediate call.
func (p *parser) pushLambdaCallContext(allowed bool) func() {
	p.lambdaCallContexts = append(p.lambdaCallContexts, allowed)
	return func() {
		p.lambdaCallContexts = p.lambdaCallContexts[:len(p.lambdaCallContexts)-1]
	}
}

// parseDirectLambdaArgument parses the lambda branch of argument. Permission
// is raised only around this direct argument, not while parsing an arbitrary
// expression that merely contains a lambda.
func (p *parser) parseDirectLambdaArgument() ast.Expr {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	allowed := len(p.lambdaCallContexts) > 0 && p.lambdaCallContexts[len(p.lambdaCallContexts)-1]
	return p.parseLambdaExpressionWithPermission(allowed)
}

// parseLambdaParameter parses the lambda-parameter production:
//
//	lambda-parameter = identifier, [ type-expression ]
//
// The type is optional because a lambda's parameter types are normally inferred
// from the collection it is applied to.
//
// Implements: lambda-parameter
func (p *parser) parseLambdaParameter() ast.Parameter {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	id := p.parseIdentifier("as a lambda parameter")

	// A type follows only when the next token can begin one. A "|" or a "," here
	// means the parameter is untyped.
	if p.startsTypeExpression(p.cur()) {
		t := p.parseTypeExpression()
		fullType := t.fullType()
		declarator := p.declFor(id.Scanned, t.actType(), fullType)
		p.declareDeclarator(id, declarator)
		return ast.Parameter{NodeName: "Parameter", Span: p.spanFrom(spanStart), SymbolDeclStmt: declarator,
			Name_: id.Scanned,
			// Lambda parameters are ordinary parameter slots, not declaration
			// statements, so a derivation must travel with the type.
			Type_:    fullType,
			WhatType: "param",
			Symb:     p.genericSymbol(id.Scanned, symboltable.S_VariableDetails, t.actType()),
		}
	}

	declarator := p.declFor(id.Scanned, "co.lang.infer", nil)
	p.declareDeclarator(id, declarator)
	return ast.Parameter{NodeName: "Parameter", Span: p.spanFrom(spanStart), SymbolDeclStmt: declarator,
		Name_:    id.Scanned,
		WhatType: "param",
		Symb:     p.genericSymbol(id.Scanned, symboltable.S_VariableDetails, "co.lang.infer"),
	}
}
