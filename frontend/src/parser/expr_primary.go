package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// primary-expression — section 11 of docs/grammar/folang.ebnf.
//
//	primary-expression = literal
//	                   | special-binding
//	                   | "this"
//	                   | "self"
//	                   | qualified-name
//	                   | grouped-expression
//	                   | tuple-expression
//	                   | array-literal
//	                   | typed-collection-literal
//	                   | object-construction
//	                   | anonymous-class-expression
//	                   | block
//	                   | anonymous-function-expression
//	                   | lambda-expression
//	                   | let-expression
//	                   | comprehension-expression
//
// This is the Pratt engine's null denotation: the single place that decides what a
// token means when an operand is expected. Several alternatives begin with the
// same token, so the order below matters and each ambiguous case delegates to a
// guard in guards.go rather than to backtracking.

// parsePrimary parses one primary-expression.
//
// Implements: primary-expression
func (p *parser) parsePrimary() ast.Expr {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	defer p.enter()()

	switch {
	// Reserved spellings are refused before anything else, so a reserved
	// operator produces a precise diagnostic rather than "unexpected token".
	case p.atReservedOperator():
		return p.parseReservedOperatorError()

	// A pre-declared glyph in OPERAND position is still an error, but not because
	// the glyph is unsupported: `∪` and `∩` are active binary INFIX operators, so
	// neither has a prefix reading to take here.
	case scanlex.IsPredeclaredOperatorSpelling(p.lexeme()):
		p.reportPredeclaredOperatorGlyph()

	// Literals.
	case p.atAny(scanlex.NUMBER, scanlex.STRING, scanlex.CHAR, scanlex.BUILT_IN_CONSTANTS, scanlex.BOOL):
		return p.parseLiteral()

	// special-binding: "$" or "$1".
	case p.at(scanlex.BIND_VAR):
		return p.parseSpecialBinding()

	// A range operator in operand position is the open-lower-bound form, the
	// second alternative of range-expression: ".. 100".
	case p.atAnyOp("..", "<..", "..<", "<..<"):
		return p.parsePrefixRange()

	// "_" is contextual rather than a general primary expression. Pattern
	// parsing consumes its own wildcard production, and the call parser consumes
	// it only for each's first key/index argument. The ONE expression position it
	// holds is the refinement-candidate: inside a co.lang.refinementType
	// predicate it denotes the candidate value of the base type.
	case p.at(scanlex.DISCARD_WILD_VAR):
		if p.refinementCandidateGuard() {
			return p.parseRefinementCandidate()
		}
		p.fail(p.cur(), `"_" is a contextual wildcard allowed only in patterns, as the first key/index argument of each, or as the candidate value inside a co.lang.refinementType predicate`)
		return nil // unreachable: fail panics

	// "(" opens a grouped expression, a tuple, or an anonymous function whose
	// parameter list starts here.
	case p.at(scanlex.OPEN_PAREN):
		if p.startsAnonymousFunction() {
			return p.parseAnonymousFunctionExpression()
		}
		return p.parseGroupedOrTupleExpression()

	// "[" opens an array literal.
	case p.at(scanlex.OPEN_BRACKET):
		return p.parseArrayLiteral()

	// "{" in operand position is ALWAYS a block used as an expression. There is no
	// untyped map literal: a braced `{ … }` map body is an object-literal
	// representation, so it is a collection BODY reachable only behind a type
	// prefix through typed-collection-literal, and never a value in its own right
	// (docs/language-ref.md, "Canonical Object and Collection Construction";
	// docs/grammar/folang.ebnf, primary-expression).
	case p.at(scanlex.OPEN_CURLY):
		return p.parseBlockExpression()

	// "|" opens a lambda.
	case p.atOp("|"):
		return p.parseLambdaExpression()

	// "forall" introduces a polymorphic anonymous function, but only when the
	// COMPLETE `forall(…).(…)->(…){` form is present. `forall` is contextual, not
	// hard reserved, so without that form the spelling is an ordinary identifier
	// and falls through to the qualified-name case below (docs/grammar/folang.ebnf,
	// forall-context-guard).
	case p.atKeyword("forall") && p.startsAnonymousFunction():
		return p.parseAnonymousFunctionExpression()

	// "let" introduces a let-expression.
	case p.atKeyword("let"):
		return p.parseLetExpression()

	// "for" introduces a comprehension.
	case p.atKeyword("for"):
		return p.parseComprehensionExpression()

	// "this" and "self" are operands in their own right; their members are
	// reached through the ordinary postfix chain.
	case p.atKeyword("this"), p.atKeyword("self"):
		return p.parseSelfReference()

	// A built-in kind in expression position is the anonymous class expression
	// `co.lang.class { … }`.
	case p.at(scanlex.BUILT_IN_KIND):
		return p.parseAnonymousClassExpression()

	// typed-collection-literal: a built-in collection type followed directly by
	// the literal body that type takes. It is tested before the general built-in
	// path because `co.core.List[…]` and `co.core.Set(…)` share their token span
	// with an index and a call on the same name.
	case p.atTypedCollectionLiteral():
		return p.parseTypedCollectionLiteral()

	// A folded built-in statement expression such as `co.out` or `this.return`.
	case p.at(scanlex.BUIL_IN_STMT_EXPRS):
		return p.parseBuiltinStatementExpression()

	// A built-in type name used as a value, which pattern matching relies on:
	// `x.match(co.pattern.Type).case(co.lang.int => …)`.
	case p.at(scanlex.BUILT_IN_TYPE):
		// object-construction begins with the complete type-postfix-expression,
		// which includes built-in types. Test the following field initializer
		// shape before committing to the ordinary type-as-value interpretation.
		if p.looksLikeObjectConstruction() {
			return p.parseObjectConstruction()
		}
		return p.parseTypeAsExpression()

	// A name into which the scanner folded a trailing ".match", which only a following
	// ".case" or ".default" reveals.
	case p.atFoldedMatchSubject():
		return p.parseFoldedMatchChain()

	// A name: possibly an object construction, otherwise a plain reference.
	case p.atIdentifier():
		if p.looksLikeObjectConstruction() {
			return p.parseObjectConstruction()
		}
		return p.parseNameExpression()
	}

	p.failf(p.cur(), "expected an expression, found %s", describeToken(p.cur()))
	return nil // unreachable: failf panics
}

// atReservedOperator reports whether the cursor holds one of the spellings the
// scanner recognises but the parser must refuse (DECISION-OP-005).
func (p *parser) atReservedOperator() bool {
	_, reserved := reservedOperators[p.lexeme()]
	return reserved
}

// reportPredeclaredOperatorGlyph reports a pre-declared operator glyph written in
// operand position, and aborts.
//
// The two glyphs are ENABLED, not reserved: `∪` and `∩` are language-owned binary
// infix operators with fixed parse properties, and an expression using one parses
// (docs/language-ref.md, "Pre-Declared Operator Glyphs"). What they do not have is
// a prefix form — predeclared-glyph-expression places them between two
// multiplicative operands and nowhere else — so a glyph reaching parsePrimary has
// no left operand and cannot be read at all.
//
// A missing overload implementation is a different failure entirely and belongs
// to operator resolution, not here.
//
// Implements: predeclared-operator-glyph
func (p *parser) reportPredeclaredOperatorGlyph() {
	tok := p.cur()
	p.reportUnsupported(tok, "pre-declared operator "+tok.Value+" is a binary infix operator and has no prefix form; it needs a left operand")
	panic(bailout{})
}

// parseReservedOperatorError reports a reserved spelling in operand position and
// aborts.
//
// The grammar requires that the parser reject these rather than ignore them, so
// that a user-defined operator cannot silently claim a spelling before the
// language assigns it a meaning.
func (p *parser) parseReservedOperatorError() ast.Expr {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	p.reportReservedOperator()
	return nil // unreachable: reportReservedOperator panics
}

// reportReservedOperator reports a reserved operator spelling and aborts.
//
// The Disclaimer's operator rule is about USE, not about position: a spelling the
// language pre-defines but has not registered with a meaning raises an
// unsupported-feature error wherever it is written. Operand position reaches this
// through parsePrimary and infix position through the Pratt loop, so both produce
// the reserved diagnostic instead of one of them degrading into a
// missing-terminator error (docs/language-ref.md, Disclaimer and C.7).
//
// Implements: reserved-operator
func (p *parser) reportReservedOperator() {
	tok := p.cur()
	why := reservedOperators[tok.Value]
	p.reportUnsupported(tok, "the operator "+tok.Value+" is "+why+" and cannot be used or overloaded yet")
	panic(bailout{})
}

// parseSelfReference parses the "this" primary-expression and the
// self-expression production.
//
// "this" is a hard reserved word and always denotes the receiver. "self" is
// CONTEXTUAL: the lexer leaves the spelling available as an identifier and the
// parser reclassifies the occurrence only where self-context-guard holds
// (docs/grammar/folang.ebnf, self-expression).
//
// In operand position both are references whose members the postfix chain
// reaches, so the two share this parse; what the guard changes is the symbol type
// recorded on the node, which is what a later phase resolves the receiver from.
//
// Implements: self-expression
func (p *parser) parseSelfReference() ast.Expr {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	tok := p.advance()
	symbolType := "self-reference"
	if logicalName(tok.Value) == "self" && !p.selfContextGuard() {
		// Outside its contextual form `self` is an ordinary identifier spelling,
		// not the class/type receiver.
		symbolType = "identifier"
	}
	return ast.SymbolExpr{Span: p.spanFrom(spanStart), Value: tok.Value,
		SymbolType_: symbolType,
		Symb:        p.exprSymbol(tok.Value),
	}
}

// selfContextGuard reports whether `self` at this occurrence has its
// language-defined class/type-receiver meaning.
//
// The reference gives it exactly two contexts, and both are about the ENCLOSING
// declaration rather than about the expression `self` sits in:
//
//   - any method declared by a co.lang.class, the lifecycle methods @@new and
//     @@init included; and
//   - an @co.dap.class method declared inside a target-bound co.lang.extension,
//     where it denotes the class/type context of the extension's `fortype`
//     target.
//
// The second context is why an extension's target is mandatory: without a
// `fortype` there would be no class context for `self` to denote, so the
// occurrence would have nothing to resolve against.
//
// Outside those two, `self` has no special class-method meaning and remains an
// ordinary identifier spelling.
//
// Implements: self-context-guard
func (p *parser) selfContextGuard() bool {
	return p.selfReceiverDepth > 0
}

// parseRefinementCandidate parses the refinement-candidate production: the `_`
// that denotes the value under test inside a refinement predicate.
//
// Every occurrence in one predicate refers to the SAME candidate, and the
// candidate has the declaration's base type. It binds nothing in the enclosing
// scope, so the node is a reference of its own symbol type rather than an
// ordinary name the surrounding scope could resolve.
//
// Implements: refinement-candidate
func (p *parser) parseRefinementCandidate() ast.Expr {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	tok := p.advance()
	return ast.SymbolExpr{Span: p.spanFrom(spanStart), Value: tok.Value,
		SymbolType_: "refinement-candidate",
		Symb:        p.exprSymbol(tok.Value),
	}
}

// pushSelfReceiverContext opens a region in which `self` has its class/type
// receiver meaning, and returns the function that closes it.
func (p *parser) pushSelfReceiverContext() func() {
	p.selfReceiverDepth++
	return func() { p.selfReceiverDepth-- }
}

// parseNameExpression parses the qualified-name alternative of primary-expression.
func (p *parser) parseNameExpression() ast.Expr {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	qn := p.parseExpressionQualifiedName("as an expression")
	return ast.SymbolExpr{Span: p.spanFrom(spanStart), Value: qn.Scanned,
		SymbolType_: "reference",
		Symb:        p.exprSymbol(qn.Scanned),
	}
}

// parseTypeAsExpression parses a built-in type name used where a value is
// expected.
//
// This is not a general coercion: it exists because pattern matching matches
// against types, so `co.lang.int` has to be admissible as a case pattern and as a
// match argument. The type is wrapped in ast.SDTExpr, which is the AST's
// type-as-expression node.
func (p *parser) parseTypeAsExpression() ast.Expr {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	// A built-in type followed immediately by "(" is ambiguous in the token
	// stream: in type position it begins a type-argument list, while in expression
	// position it is a call target such as
	// `co.lang.tag(co.lang.string, "Hello")`.  This function is reached only from
	// expression parsing, so leave the parenthesis for parsePostfix and consume
	// only the named type atom.  Other type-as-value forms retain the complete type
	// expression used by match patterns.
	var t typeRef
	if p.at(scanlex.BUILT_IN_TYPE) && p.peek(1).Kind == scanlex.OPEN_PAREN {
		t = p.parseNamedTypeAtom()
	} else {
		t = p.parseTypeExpression()
	}
	return ast.SDTExpr{Span: p.spanFrom(spanStart), Type_: t.fullType(), Symb: p.exprSymbol(t.actType())}
}

// parseBuiltinStatementExpression parses a folded built-in statement expression.
//
// The scanner folds a built-in path down to its namespace and leaves the method as
// a separate member token: `co.out.println(x)` arrives as BUIL_IN_STMT_EXPRS
// ("co.out"), DOT, BUILT_IN_METHOD("println"). The namespace is the operand here
// and the postfix chain picks up the member and the call, so no special handling
// of the call itself is needed.
//
// `this.return`, `this.break` and `this.continue` fold whole, and are recognised
// as statements by stmt_return.go rather than here.
func (p *parser) parseBuiltinStatementExpression() ast.Expr {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	if p.atVariantDefinition() {
		p.failf(p.cur(), "%s is valid only as the variant-definition right-hand side of a co.lang.type declaration", variantDefinitionName)
	}
	tok := p.advance()
	return ast.SymbolExpr{Span: p.spanFrom(spanStart), Value: tok.Value,
		SymbolType_: "builtin",
		Symb:        p.exprSymbol(tok.Value),
	}
}

// parseBlockExpression parses the block alternative of primary-expression.
//
// A block is a value: DECISION-BLK-001 makes its final unterminated expression the
// block's value, so `{ n = n + 100; "GT" }` evaluates to "GT". The block is
// wrapped in ast.StatementExpr, which is the AST's statement-as-expression node.
func (p *parser) parseBlockExpression() ast.Expr {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	block := p.parseBlock("a block expression")
	return ast.StatementExpr{Span: p.spanFrom(spanStart), Statement: block, Symb: p.exprSymbol("block")}
}
