package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
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
// Using "|...|" anywhere else is an error, but that restriction is about WHERE the
// lambda appears rather than about its shape, so it is enforced by the semantic
// phase. The parser accepts the form wherever primary-expression admits it.
//
// The delimiter is the same "|" that spells bitwise OR and type union. There is no
// ambiguity in practice, because a lambda only ever appears in operand position and
// the operators only in infix position, which is the ordinary Pratt split.

// parseLambdaExpression parses the lambda-expression production.
func (p *parser) parseLambdaExpression() ast.Expr {
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
	if p.at(scanlex.OPEN_CURLY) && !p.looksLikeMapLiteral() {
		block := p.parseBlock("a lambda body")
		body = ast.StatementExpr{Statement: block, Symb: p.exprSymbol("lambda-body")}
	} else {
		body = p.parseExpression()
	}

	return ast.LambdaExpr{
		Parameters: params,
		Body:       body,
		Symb:       p.lambdaSymbol("lambda"),
	}
}

// parseLambdaParameter parses the lambda-parameter production:
//
//	lambda-parameter = identifier, [ type-expression ]
//
// The type is optional because a lambda's parameter types are normally inferred
// from the collection it is applied to.
func (p *parser) parseLambdaParameter() ast.Parameter {
	id := p.parseIdentifier("as a lambda parameter")

	// A type follows only when the next token can begin one. A "|" or a "," here
	// means the parameter is untyped.
	if p.startsTypeExpression(p.cur()) {
		t := p.parseTypeExpression()
		return ast.Parameter{
			SymbolDeclStmt: p.declFor(id.Scanned, t.actType(), t.Node),
			Name_:          id.Scanned,
			Type_:          t.Node,
			WhatType:       "param",
			Symb:           p.genericSymbol(id.Scanned, symboltable.S_VariableDetails, t.actType()),
		}
	}

	return ast.Parameter{
		SymbolDeclStmt: p.declFor(id.Scanned, "co.lang.infer", nil),
		Name_:          id.Scanned,
		WhatType:       "param",
		Symb:           p.genericSymbol(id.Scanned, symboltable.S_VariableDetails, "co.lang.infer"),
	}
}
