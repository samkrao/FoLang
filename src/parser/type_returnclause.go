package parser

import (
	"github.com/samkrao/fo-lang/src/ast"
	symboltable "github.com/samkrao/fo-lang/src/context"
	"github.com/samkrao/fo-lang/src/scanlex"
)

// Return-type clauses — the return-type-clause family of section 4.
//
//	return-type-clause = "->", "(", [ return-item-list ], ")"
//	return-item-list   = return-item, { ",", return-item }
//	return-item        = type-expression
//
// A FoLang function may return several values. Result entries are types only;
// callers bind returned values explicitly:
//
//	fun1(k co.lang.int)->(co.lang.int, co.lang.char) = { … }
//	doManythings(a co.lang.int)->(co.lang.int, co.lang.exception) = { … }

// parseReturnTypeClause parses the return-type-clause production, consuming the
// leading "->".
//
// Implements: declaration-return-type-clause
func (p *parser) parseReturnTypeClause() []ast.Returns {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	p.expect(scanlex.ARROW, "to begin a return-type clause")
	return p.parseParenthesizedReturnList()
}

// parseTypeExpressionReturnClause is used only while defining a function type.
// Unlike an ordinary function declaration's result clause, the surrounding
// co.lang.type RHS is a type-producing context and may contain full type
// expressions, including nested function-type shapes. It does not admit ordinary
// runtime value expressions.
func (p *parser) parseTypeExpressionReturnClause() []ast.Returns {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	p.expect(scanlex.ARROW, "to begin a function-type result clause")
	return p.parseTypeExpressionParenthesizedReturnList()
}

// parseParenthesizedReturnList parses the parenthesised part of a
// return-type-clause, after the "->" has already been consumed.
//
// It is shared with arrow-type-tail, where the same parenthesised list spells the
// results of a function type.
func (p *parser) parseParenthesizedReturnList() []ast.Returns {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	return p.parseParenthesizedReturnListWith(false)
}

func (p *parser) parseTypeExpressionParenthesizedReturnList() []ast.Returns {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	return p.parseParenthesizedReturnListWith(true)
}

func (p *parser) parseParenthesizedReturnListWith(fullTypeExpression bool) []ast.Returns {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	p.expect(scanlex.OPEN_PAREN, "to open a return-type clause")

	var results []ast.Returns
	if !p.at(scanlex.CLOSE_PAREN) {
		results = append(results, p.parseReturnItemWith(fullTypeExpression))
		for p.accept(scanlex.COMMA) {
			results = append(results, p.parseReturnItemWith(fullTypeExpression))
		}
	}
	p.expect(scanlex.CLOSE_PAREN, "to close a return-type clause")
	return results
}

// parseReturnItem parses the return-item production:
//
//	return-item = type-expression
//
// Implements: return-item
// Implements: declaration-return-item
func (p *parser) parseReturnItem() ast.Returns {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	return p.parseReturnItemWith(false)
}

func (p *parser) parseReturnItemWith(fullTypeExpression bool) ast.Returns {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	var t typeRef
	if fullTypeExpression {
		t = p.parseTypeExpression()
	} else {
		t = p.parseTypeUse("as a function result type")
	}
	return ast.Returns{NodeName: "Returns", Span: p.spanFrom(spanStart), SymbolDeclStmt: p.declFor("", t.actType(), t.fullType()),
		Type_:    t.fullType(),
		OnlyType: true,
		WhatType: "result",
		Symb:     p.genericSymbol("", symboltable.S_VariableDetails, t.actType()),
	}
}

// namePrecedesType distinguishes the optional name in a receiver-like `name Type`
// pair from a bare type. A following parenthesis applies the current identifier as
// a parameterized type, so it cannot introduce a separate name.
func (p *parser) namePrecedesType() bool {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	next := p.peek(1)
	// A following parenthesis applies the current name as a parameterized type.
	if next.Kind == scanlex.OPEN_PAREN {
		return false
	}
	return p.startsTypeUse(next)
}

// namePrecedesFullTypeExpression retains the broader decision only inside a
// co.lang.type RHS, where a named component may itself have a parenthesized
// function type. Ordinary declarations never call this probe.
func (p *parser) namePrecedesFullTypeExpression() bool {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	if !p.atIdentifier() {
		return false
	}
	next := p.peek(1)
	if next.Kind != scanlex.OPEN_PAREN {
		return p.startsTypeExpression(next)
	}
	after, ok := p.tokenAfterMatchingParen(1)
	return ok && after.Kind == scanlex.ARROW
}

// startsTypeExpression reports whether tok could begin a type-expression.
//
// It is used wherever the grammar makes a leading identifier optional and the
// decision turns on whether a type follows it, such as function-type parameters.
func (p *parser) startsTypeExpression(tok scanlex.Token) bool {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	switch tok.Kind {
	case scanlex.BUILT_IN_TYPE,
		scanlex.IDENTIFIER,
		scanlex.COMPOSITE_IDENTIFER,
		scanlex.OPEN_PAREN:
		return true
	case scanlex.BUILT_IN_KIND:
		// A kind names a type wherever a type is expected: `T co.lang.type` and
		// `target co.lang.function` are ordinary parameters, and
		// `->(co.lang.dependentType)` is what a type constructor returns. The kind
		// tokens reach the type parser through qualified-name, which already accepts
		// them; only this predicate gated them out.
		return true
	case scanlex.BUIL_IN_STMT_EXPRS:
		// A co.* path that is not in the built-in type table arrives folded down to
		// its namespace, so `co.lang.map` presents as BUIL_IN_STMT_EXPRS("co.lang")
		// followed by the member. Every co.* path is always available, so such a path
		// is admissible as a type name.
		return true
	case scanlex.KEYWORD, scanlex.RESERVEDWORD:
		// Only "forall" begins a type; the other reserved words do not.
		return tok.Value == "forall"
	}
	return false
}

// parseFunctionType parses the function-type production:
//
//	function-type = "(", [ function-type-parameter,
//	                       { ",", function-type-parameter } ], ")",
//	                return-type-clause
//
// This is the standalone signature form used by a delegate declaration
// (docs/language-ref.md, "Function Delegates"):
//
//	@co.dap.delegate someDelegate co.lang.delegate =
//	    (a co.lang.int, b co.lang.int)->(co.lang.int, co.lang.int);
//
// The parameter list is spelled as a type-list by the grammar, but the reference
// examples name their parameters, so a name followed by a type is accepted and the
// name is kept.
//
// Implements: function-type
func (p *parser) parseFunctionType() ast.Type {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	p.expect(scanlex.OPEN_PAREN, "to open a function type")

	var params []ast.Parameter
	if !p.at(scanlex.CLOSE_PAREN) {
		params = append(params, p.parseFunctionTypeParameter())
		for p.accept(scanlex.COMMA) {
			if p.at(scanlex.CLOSE_PAREN) {
				p.fail(p.cur(), "a comma in a delegate parameter group must be followed by another type; trailing commas are not allowed")
			}
			params = append(params, p.parseFunctionTypeParameter())
		}
	}
	p.expect(scanlex.CLOSE_PAREN, "to close a function type")

	results := p.parseTypeExpressionReturnClause()

	return ast.FunctionType{NodeName: "FunctionType", Span: p.spanFrom(spanStart), Params: [][]ast.Parameter{params},
		Results: results,
		Symb:    p.typeSymbol("co.lang.function"),
	}
}

// parseFunctionTypeParameter parses one entry of a function type's parameter list,
// accepting either a bare type or a named one.
//
//	function-type-parameter = type-expression
//	                        | identifier, type-expression
//
// Implements: function-type-parameter
func (p *parser) parseFunctionTypeParameter() ast.Parameter {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	if p.atIdentifier() && p.namePrecedesFullTypeExpression() {
		named := p.parseIdentifier("as a parameter name")
		t := p.parseTypeExpression()
		return ast.Parameter{NodeName: "Parameter", Span: p.spanFrom(spanStart), SymbolDeclStmt: p.declFor(named.Scanned, t.actType(), t.fullType()),
			Name_:    named.Scanned,
			Type_:    t.fullType(),
			WhatType: "param",
			Symb:     p.genericSymbol(named.Scanned, symboltable.S_VariableDetails, t.actType()),
		}
	}

	t := p.parseTypeExpression()
	return ast.Parameter{NodeName: "Parameter", Span: p.spanFrom(spanStart), SymbolDeclStmt: p.declFor("", t.actType(), t.fullType()),
		Type_:    t.fullType(),
		OnlyType: true,
		WhatType: "param",
		Symb:     p.genericSymbol("", symboltable.S_VariableDetails, t.actType()),
	}
}
