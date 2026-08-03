package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// Return-type clauses — the return-type-clause family of section 4.
//
//	return-type-clause = "->", "(", [ return-item-list ], ")"
//	return-item-list   = return-item, { ",", return-item }
//	return-item        = [ identifier ], type-expression
//
// A FoLang function may return several values, and each may be named
// (docs/language-ref.md, "Named Returns"):
//
//	fun1(k co.lang.int)->(co.lang.int, co.lang.char) = { … }
//	doManythings(a co.lang.int)->(r co.lang.int, e co.lang.exception) = { … }

// parseReturnTypeClause parses the return-type-clause production, consuming the
// leading "->".
func (p *parser) parseReturnTypeClause() []ast.Returns {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	p.expect(scanlex.ARROW, "to begin a return-type clause")
	return p.parseParenthesizedReturnList()
}

// parseParenthesizedReturnList parses the parenthesised part of a
// return-type-clause, after the "->" has already been consumed.
//
// It is shared with arrow-type-tail, where the same parenthesised list spells the
// results of a function type.
func (p *parser) parseParenthesizedReturnList() []ast.Returns {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	p.expect(scanlex.OPEN_PAREN, "to open a return-type clause")

	var results []ast.Returns
	if !p.at(scanlex.CLOSE_PAREN) {
		results = append(results, p.parseReturnItem())
		for p.accept(scanlex.COMMA) {
			results = append(results, p.parseReturnItem())
		}
	}
	p.expect(scanlex.CLOSE_PAREN, "to close a return-type clause")
	return results
}

// parseReturnItem parses the return-item production:
//
//	return-item = [ identifier ], type-expression
//
// The optional leading identifier names the result. Deciding whether a leading
// name is present needs care, because a bare type is itself a name: in
// `->(r co.lang.int)` the "r" is a result name, while in `->(Employee)` the
// "Employee" is the type. The two are told apart by what follows — a name is only
// a result name when another type follows it.
func (p *parser) parseReturnItem() ast.Returns {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	if p.atIdentifier() && p.namePrecedesType() {
		named := p.parseIdentifier("as a result name")
		t := p.parseTypeExpression()
		return ast.Returns{
			SymbolDeclStmt: p.declFor(named.Scanned, t.actType(), t.fullType()),
			IsNamed:        true,
			Type_:          t.fullType(),
			WhatType:       "result",
			Symb:           p.genericSymbol(named.Scanned, symboltable.S_VariableDetails, t.actType()),
		}
	}

	t := p.parseTypeExpression()
	return ast.Returns{
		SymbolDeclStmt: p.declFor("", t.actType(), t.fullType()),
		Type_:          t.fullType(),
		OnlyType:       true,
		WhatType:       "result",
		Symb:           p.genericSymbol("", symboltable.S_VariableDetails, t.actType()),
	}
}

// namePrecedesType reports whether the identifier at the cursor NAMES the item that
// follows it, rather than being the head of that item's own type.
//
// Both readings begin with an identifier, and the two positions that use this — a
// return-item and a function-type parameter — spell the name as optional:
//
//	->(r co.lang.int)        "r" names the result, co.lang.int is its type
//	->(Employee)             "Employee" IS the type
//
// A following "(" is the case that needs real lookahead, because it is ambiguous:
//
//	->(Matrix(r, c))         one result whose type is the generic Matrix(r, c)
//	->(f (A)->(B))           a result named "f" whose type is a function type
//
// A type-argument list belongs to the name before it, so the identifier is part of the
// type. A function type's parameter list is a separate item, so the identifier is a
// name. The "->" after the balanced group is what separates them: only a function type
// has one. Reading a type-argument list as a name was a silent misparse for a
// single-argument generic — `->(Vector(n))` became a result named "Vector" of type "n" —
// and an error for two or more.
func (p *parser) namePrecedesType() bool {
	next := p.peek(1)
	if next.Kind != scanlex.OPEN_PAREN {
		return p.startsTypeExpression(next)
	}
	return p.lookaheadOnly(func() bool {
		p.advance() // the identifier
		p.skipBalanced(scanlex.OPEN_PAREN, scanlex.CLOSE_PAREN)
		return p.at(scanlex.ARROW)
	})
}

// startsTypeExpression reports whether tok could begin a type-expression.
//
// It is used wherever the grammar makes a leading identifier optional and the
// decision turns on whether a type follows it — return items, parameters and
// typed variable declarators all share this shape.
func (p *parser) startsTypeExpression(tok scanlex.Token) bool {
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
func (p *parser) parseFunctionType() ast.Type {
	if traceEnabled {
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

	results := p.parseReturnTypeClause()

	return ast.FunctionType{
		Params:  [][]ast.Parameter{params},
		Results: results,
		Symb:    p.typeSymbol("co.lang.function"),
	}
}

// parseFunctionTypeParameter parses one entry of a function type's parameter list,
// accepting either a bare type or a named one.
//
//	function-type-parameter = type-expression
//	                        | identifier, type-expression
func (p *parser) parseFunctionTypeParameter() ast.Parameter {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	if p.atIdentifier() && p.namePrecedesType() {
		named := p.parseIdentifier("as a parameter name")
		t := p.parseTypeExpression()
		return ast.Parameter{
			SymbolDeclStmt: p.declFor(named.Scanned, t.actType(), t.fullType()),
			Name_:          named.Scanned,
			Type_:          t.fullType(),
			WhatType:       "param",
			Symb:           p.genericSymbol(named.Scanned, symboltable.S_VariableDetails, t.actType()),
		}
	}

	t := p.parseTypeExpression()
	return ast.Parameter{
		SymbolDeclStmt: p.declFor("", t.actType(), t.fullType()),
		Type_:          t.fullType(),
		OnlyType:       true,
		WhatType:       "param",
		Symb:           p.genericSymbol("", symboltable.S_VariableDetails, t.actType()),
	}
}
