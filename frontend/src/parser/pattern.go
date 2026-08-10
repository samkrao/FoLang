package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// Patterns — section 9 of docs/grammar/folang.ebnf.
//
//	pattern = wildcard
//	        | literal-pattern
//	        | binding-pattern
//	        | constructor-pattern
//	        | record-pattern
//	        | tuple-pattern
//	        | qualified-name
//
// Patterns appear in three places: a function-pattern clause's parameter list, a
// match case, and a comprehension binding. A pattern both tests a value's shape and
// binds names out of it, so the parsed form keeps the structure rather than
// collapsing to an expression: bindingNamesOf walks it to collect the names a
// comprehension or a case introduces.

// patternForm names which pattern alternative was matched.
type patternForm int

const (
	// patternWildcard is "_": matches anything and binds nothing.
	patternWildcard patternForm = iota
	// patternLiteral matches a literal value: fib(0).
	patternLiteral
	// patternBinding is a bare name: it matches anything and binds it.
	patternBinding
	// patternConstructor destructures a constructor: Some(x), None().
	patternConstructor
	// patternRecord destructures named fields: Point{x, y}.
	patternRecord
	// patternTuple destructures a tuple: (name, age).
	patternTuple
	// patternQualified is a dotted name matched by identity: xx.CAT.
	patternQualified
)

// pattern is a parsed pattern.
//
// Expr carries the pattern rendered as an expression, which is what the AST's
// pattern-bearing nodes store, and Form and Elements carry the structure the
// parser still needs — chiefly for collecting bound names.
type pattern struct {
	Form     patternForm
	Name     string
	Elements []pattern
	Expr     ast.Expr
	Tok      scanlex.Token
}

// parsePattern parses one pattern.
//
// The alternatives are separated by their leading token and, for the name forms, by
// what follows the name: a "(" makes it a constructor pattern, a "{" a record
// pattern, a dot a qualified name, and nothing at all a binding pattern.
//
// Implements: pattern
func (p *parser) parsePattern() pattern {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	defer p.enter()()

	start := p.cur()

	switch {
	// wildcard: "_" is admitted here because pattern matching is one of the
	// contexts that spells it directly.
	case p.at(scanlex.DISCARD_WILD_VAR):
		p.advance()
		return pattern{
			Form: patternWildcard,
			Expr: ast.SymbolExpr{Span: p.spanFrom(spanStart), Value: start.Value,
				SymbolType_: "wildcard",
				Symb:        p.exprSymbol(start.Value),
			},
			Tok: start,
		}

	// tuple-pattern: "(" pattern "," pattern { "," pattern } ")"
	case p.at(scanlex.OPEN_PAREN):
		return p.parseTuplePattern()

	// literal-pattern.
	case p.atAny(scanlex.NUMBER, scanlex.STRING, scanlex.CHAR, scanlex.BUILT_IN_CONSTANTS, scanlex.BOOL):
		lit := p.parseLiteral()
		return pattern{Form: patternLiteral, Expr: lit, Tok: start}

	// A built-in type used as a pattern, which type matching relies on:
	// x.match(co.pattern.Type).case(co.lang.int => …)
	case p.at(scanlex.BUILT_IN_TYPE):
		t := p.parseTypeExpression()
		return pattern{
			Form: patternQualified,
			Name: t.actType(),
			Expr: ast.SDTExpr{Span: p.spanFrom(spanStart), Type_: t.fullType(), Symb: p.exprSymbol(t.actType())},
			Tok:  start,
		}

	// A name: constructor, record, qualified or binding pattern.
	case p.atIdentifier(), p.at(scanlex.BUIL_IN_STMT_EXPRS):
		return p.parseNamePattern()
	}

	// A signed numeric pattern, as in fib(-1). Unary operators are not a
	// general pattern form, so the sign must be followed directly by a number.
	if p.atOp("-") || p.atOp("+") {
		op := p.advance()
		if !p.at(scanlex.NUMBER) {
			p.failf(p.cur(), "expected a numeric literal after %q in a pattern, found %s", op.Value, describeToken(p.cur()))
		}
		expr := ast.PrefixExpr{Span: p.spanFrom(spanStart), Operator: op,
			Right: p.parseLiteral(),
			Symb:  p.exprSymbol(op.Value),
		}
		return pattern{Form: patternLiteral, Expr: expr, Tok: start}
	}

	p.failf(p.cur(), "expected a pattern, found %s", describeToken(p.cur()))
	return pattern{} // unreachable: failf panics
}

// parseNamePattern parses the name-led pattern alternatives:
//
//	constructor-pattern = qualified-name, "(", [ pattern, { ",", pattern } ], ")"
//	record-pattern      = qualified-name, "{", [ record-pattern-field,
//	                        { ",", record-pattern-field } ], "}"
//	binding-pattern     = identifier
//	qualified-name
//
// A single undotted name with nothing after it is a binding pattern, so `f(n)` binds
// n. A dotted name is matched by identity instead, because `xx.CAT` names an existing
// value rather than introducing one.
//
// Implements: binding-pattern
func (p *parser) parseNamePattern() pattern {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	start := p.cur()
	qn := p.parseQualifiedName("as a pattern")

	switch {
	case p.at(scanlex.OPEN_PAREN):
		return p.parseConstructorPattern(qn, start)
	case p.at(scanlex.OPEN_CURLY):
		return p.parseRecordPattern(qn, start)
	}

	ref := ast.SymbolExpr{Span: p.spanFrom(spanStart), Value: qn.Scanned,
		SymbolType_: "pattern",
		Symb:        p.exprSymbol(qn.Scanned),
	}

	// A dotted name is an identity match; a plain one binds.
	form := patternBinding
	if containsDot(qn.Logical) {
		form = patternQualified
	}

	return pattern{Form: form, Name: qn.Scanned, Expr: ref, Tok: start}
}

// parseConstructorPattern parses the constructor-pattern production.
//
// This is what makes algebraic data types destructurable
// (docs/language-ref.md, "Function Pattern"):
//
//	f(Some(x)) => { x + 1 }
//	f(None())  => { 0 }
//
// Implements: constructor-pattern
func (p *parser) parseConstructorPattern(qn name, start scanlex.Token) pattern {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	p.expect(scanlex.OPEN_PAREN, "to open a constructor pattern")

	var elements []pattern
	if !p.at(scanlex.CLOSE_PAREN) {
		elements = append(elements, p.parsePattern())
		for p.accept(scanlex.COMMA) {
			elements = append(elements, p.parsePattern())
		}
	}

	p.expect(scanlex.CLOSE_PAREN, "to close a constructor pattern")

	return pattern{
		Form:     patternConstructor,
		Name:     qn.Scanned,
		Elements: elements,
		Expr: ast.CallExpr{Span: p.spanFrom(spanStart), Method: ast.SymbolExpr{Span: p.spanFrom(spanStart), Value: qn.Scanned,
			SymbolType_: "constructor",
			Symb:        p.exprSymbol(qn.Scanned),
		},
			Arguments:   patternExprs(elements),
			SymbolType_: "constructor-pattern",
			Symb:        p.exprSymbol(qn.Scanned),
		},
		Tok: start,
	}
}

// parseRecordPattern parses the record-pattern and record-pattern-field
// productions:
//
//	record-pattern       = qualified-name, "{", [ record-pattern-field,
//	                         { ",", record-pattern-field } ], "}"
//	record-pattern-field = identifier, [ ":", pattern ]
//
// A field with no ":" binds the field's own name, so `Point{x, y}` binds x and y.
// This is the shape matcher of docs/language-ref.md, "Pattern Matching":
//
//	x.match(co.pattern.Shape).case(Point{x, y} => …).default(…);
//
// Implements: record-pattern
func (p *parser) parseRecordPattern(qn name, start scanlex.Token) pattern {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	p.expect(scanlex.OPEN_CURLY, "to open a record pattern")

	var elements []pattern
	var fields []ast.Expr
	for !p.at(scanlex.CLOSE_CURLY) && !p.atEOF() {
		fieldStart := p.cur()
		field := p.parseIdentifier("as a record pattern field name")

		var value pattern
		if p.accept(scanlex.COLON) {
			value = p.parsePattern()
		} else {
			// A bare field name binds the field to that name.
			value = pattern{
				Form: patternBinding,
				Name: field.Scanned,
				Expr: ast.SymbolExpr{Span: p.spanFrom(spanStart), Value: field.Scanned,
					SymbolType_: "pattern",
					Symb:        p.exprSymbol(field.Scanned),
				},
				Tok: fieldStart,
			}
		}
		elements = append(elements, value)
		// Retain the field label independently of the nested pattern. Without
		// this pair, Employee{id: value, name: _} degrades to two positional
		// patterns and cannot be matched by field name later.
		fields = append(fields, ast.AssignmentExpr{Span: p.spanFrom(spanStart), Assigne: ast.SymbolExpr{Span: p.spanFrom(spanStart), Value: field.Scanned,
			SymbolType_: "field-name",
			Symb:        p.exprSymbol(field.Scanned),
		},
			AssignedValue: value.Expr,
			Symb:          p.exprSymbol(field.Scanned),
		})

		if !p.accept(scanlex.COMMA) {
			break
		}
		if p.at(scanlex.CLOSE_CURLY) {
			p.fail(p.cur(), "a comma in a record pattern must be followed by another field; trailing commas are not allowed")
		}
	}

	p.expect(scanlex.CLOSE_CURLY, "to close a record pattern")

	return pattern{
		Form:     patternRecord,
		Name:     qn.Scanned,
		Elements: elements,
		Expr: ast.NewExpr{Span: p.spanFrom(spanStart), Instantiation: ast.CallExpr{Span: p.spanFrom(spanStart), Method: ast.SymbolExpr{Span: p.spanFrom(spanStart), Value: qn.Scanned,
			SymbolType_: "record",
			Symb:        p.exprSymbol(qn.Scanned),
		},
			Arguments:   fields,
			SymbolType_: "record-pattern",
			Symb:        p.exprSymbol(qn.Scanned),
		},
			Symb: p.exprSymbol(qn.Scanned),
		},
		Tok: start,
	}
}

// parseTuplePattern parses the tuple-pattern production:
//
//	tuple-pattern = "(", pattern, ",", pattern, { ",", pattern }, ")"
//
// At least two elements are required. The grammar has no grouped-pattern
// alternative, so a parenthesised single pattern is rejected.
//
// Implements: tuple-pattern
func (p *parser) parseTuplePattern() pattern {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	start := p.expect(scanlex.OPEN_PAREN, "to open a tuple pattern")

	elements := []pattern{p.parsePattern()}
	if !p.accept(scanlex.COMMA) {
		p.failf(p.cur(), "a tuple pattern requires at least two comma-separated patterns")
	}
	elements = append(elements, p.parsePattern())
	for p.accept(scanlex.COMMA) {
		elements = append(elements, p.parsePattern())
	}

	p.expect(scanlex.CLOSE_PAREN, "to close a tuple pattern")

	return pattern{
		Form:     patternTuple,
		Elements: elements,
		Expr: ast.GroupingExpr{Span: p.spanFrom(spanStart), Expr_: p.foldComma(patternExprs(elements)),
			Symb: p.exprSymbol("tuple-pattern"),
		},
		Tok: start,
	}
}

// parsePatternParameterList parses the pattern-parameter-list production:
//
//	pattern-parameter-list = "(", [ pattern, { ",", pattern } ], ")"
//
// Implements: pattern-parameter-list
func (p *parser) parsePatternParameterList() []pattern {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	p.expect(scanlex.OPEN_PAREN, "to open a pattern parameter list")

	var patterns []pattern
	if !p.at(scanlex.CLOSE_PAREN) {
		patterns = append(patterns, p.parsePattern())
		for p.accept(scanlex.COMMA) {
			patterns = append(patterns, p.parsePattern())
		}
	}

	p.expect(scanlex.CLOSE_PAREN, "to close a pattern parameter list")
	return patterns
}

// patternExprs projects a pattern slice onto its expression forms.
func patternExprs(patterns []pattern) []ast.Expr {
	exprs := make([]ast.Expr, 0, len(patterns))
	for _, pat := range patterns {
		exprs = append(exprs, pat.Expr)
	}
	return exprs
}

// containsDot reports whether a name is dotted, which distinguishes a qualified
// reference from a plain binding name.
func containsDot(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == '.' {
			return true
		}
	}
	return false
}
