package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// Type derivations — the type-derivation family of section 4.
//
//	type-derivation          = "(", derivation-specification, ")"
//	derivation-specification = pointer-specification
//	                         | array-specification
//	                         | reference-specification
//	                         | range-type-specification
//	                         | slice-type-specification
//	                         | thunk-type-specification
//	                         | address-type-specification
//	                         | derivation-attribute-list
//
// A derivation is what turns a base type into a pointer, array, reference,
// address, thunk, slice or range (docs/language-ref.md, "Variable Kinds"):
//
//	somePtr    co.lang.int->(*)     someDblPtr co.lang.int->(**)
//	someArray  co.lang.int->([5])   someJagged co.lang.int->([2][3])
//	someRef    co.lang.int->(&)     someLValue co.lang.int->(&&)
//	someHpRef  co.lang.int->(~)     someAddr   co.lang.int->(@)
//	someThunk  co.lang.int->(^)     someSlice  co.lang.int->([:])
//	someRange  co.lang.int->(..)
//
// DECISION-TYP-001 allows every one of these to carry a trailing attribute list,
// which is what admits the fat-pointer and address-manipulation forms
// `co.lang.int->(*, kind=region, meta={})` and `co.lang.word->(repr=intptr)`.

// startsDerivationSpecification reports whether the "(" at the cursor opens a
// type-derivation rather than a parenthesised type list.
//
// This is the lookahead that lets parseArrowTypeTail choose without backtracking.
// The decision is made on the token just inside the paren: every
// derivation-specification begins with a sigil, a "[", or an `attribute=` pair,
// and none of those can begin a type.
func (p *parser) startsDerivationSpecification() bool {
	if !p.at(scanlex.OPEN_PAREN) {
		return false
	}
	inner := p.peek(1)

	switch inner.Value {
	// "**" appears here as well as "*" because the scanner fuses a star pair into one
	// token, so ->(**) presents its whole pointer depth as a single lexeme.
	case "*", "**", "&", "&&", "~", "@", "^", "..", "[:]":
		return true
	}
	if inner.Kind == scanlex.OPEN_BRACKET || inner.Kind == scanlex.OB_COLON_CB {
		return true
	}
	// A bare derivation-attribute-list: `identifier "=" value`, as in
	// co.lang.word->(repr=intptr).
	if inner.IsOneOfMany(scanlex.IDENTIFIER, scanlex.COMPOSITE_IDENTIFER) {
		return p.peek(2).Value == "="
	}
	return false
}

// parseTypeDerivation parses one type-derivation and applies it to base.
//
// The returned typeRef keeps base's lowered node as its element type, because the
// declaration nodes this feeds expect the underlying type rather than a decorated
// one. Form records which derivation was applied.
func (p *parser) parseTypeDerivation(base typeRef) typeRef {
	open := p.expect(scanlex.OPEN_PAREN, "to open a type derivation")
	derived := p.parseDerivationSpecification(base, open)
	p.expect(scanlex.CLOSE_PAREN, "to close a type derivation")
	return derived
}

// parseDerivationSpecification dispatches on the sigil that opens the derivation.
func (p *parser) parseDerivationSpecification(base typeRef, open scanlex.Token) typeRef {
	switch {
	case p.atOp("*"), p.atOp("**"):
		return p.parsePointerSpecification(base, open)
	case p.atOp("&"), p.atOp("&&"):
		return p.parseReferenceSpecification(base, open)
	case p.atOp("~"):
		return p.parseHeapReferenceSpecification(base, open)
	case p.atOp("@"):
		return p.parseAddressSpecification(base, open)
	case p.atOp("^"):
		return p.parseThunkSpecification(base, open)
	case p.at(scanlex.OB_COLON_CB):
		return p.parseSliceSpecification(base, open)
	case p.atOp(".."):
		return p.parseRangeSpecification(base, open)
	case p.at(scanlex.OPEN_BRACKET):
		return p.parseArraySpecification(base, open)
	default:
		return p.parseBareAttributeDerivation(base, open)
	}
}

// parsePointerSpecification parses the pointer-specification production:
//
//	pointer-specification = pointer-stars, [ ",", derivation-attribute-list ]
//	pointer-stars         = "*", { "*" }
//
// The star count is the pointer depth: ->(*) has degree one, ->(**) degree two,
// and ->(*****) degree five. The grammar's `"*", { "*" }` admits every positive
// degree; one and two are examples rather than special parser cases or a limit.
//
// The scanner emits "**" as one fused token (see tokenstream.go), so a run of
// stars can arrive as a mixture of "*" and "**" and both are counted.
func (p *parser) parsePointerSpecification(base typeRef, open scanlex.Token) typeRef {
	stars := 0
	for p.atOp("*") || p.atOp("**") {
		stars += len(p.advance().Value)
	}

	out := base
	out.Form = formPointer
	out.PointerCount = stars
	out.Attrs = p.parseOptionalAttributeTail()
	out.Tok = open
	return out
}

// parseReferenceSpecification parses the reference-specification production for
// "&" and "&&":
//
//	reference-specification = ( "&" | "&&" | "~" ),
//	                          [ ",", derivation-attribute-list ]
//
// ->(&) is a reference and ->(&&) an lvalue reference (docs/language-ref.md,
// "Reference Declaration"). The "~" alternative is a heap-allocated reference and
// is handled separately so the declaration lowering can tell them apart.
func (p *parser) parseReferenceSpecification(base typeRef, open scanlex.Token) typeRef {
	tok := p.advance()

	out := base
	out.Form = formReference
	out.RefCount = len(tok.Value) // "&" -> 1, "&&" -> 2
	out.Attrs = p.parseOptionalAttributeTail()
	out.Tok = open
	return out
}

// parseHeapReferenceSpecification parses the "~" alternative of
// reference-specification, a heap-allocated reference.
func (p *parser) parseHeapReferenceSpecification(base typeRef, open scanlex.Token) typeRef {
	p.advance() // "~"

	out := base
	out.Form = formHeapReference
	out.RefCount = 1
	out.Attrs = p.parseOptionalAttributeTail()
	out.Tok = open
	return out
}

// parseAddressSpecification parses the address-type-specification production:
//
//	address-type-specification = "@", [ ",", derivation-attribute-list ]
func (p *parser) parseAddressSpecification(base typeRef, open scanlex.Token) typeRef {
	p.advance() // "@"

	out := base
	out.Form = formAddress
	out.Attrs = p.parseOptionalAttributeTail()
	out.Tok = open
	return out
}

// parseThunkSpecification parses the thunk-type-specification production:
//
//	thunk-type-specification = "^", [ ",", derivation-attribute-list ]
//
// A thunk is the lazily evaluated form of its base type.
func (p *parser) parseThunkSpecification(base typeRef, open scanlex.Token) typeRef {
	p.advance() // "^"

	out := base
	out.Form = formThunk
	out.Attrs = p.parseOptionalAttributeTail()
	out.Tok = open
	return out
}

// parseSliceSpecification parses the slice-type-specification production:
//
//	slice-type-specification = "[:]", [ ",", derivation-attribute-list ]
//
// The scanner emits "[:]" as the single token OB_COLON_CB, so it cannot be
// confused with an empty index or an array dimension.
func (p *parser) parseSliceSpecification(base typeRef, open scanlex.Token) typeRef {
	p.advance() // "[:]"

	out := base
	out.Node = ast.ListType{Underlying: base.Node, Symb: p.typeSymbol("co.lang.slice")}
	out.Form = formSlice
	out.Attrs = p.parseOptionalAttributeTail()
	out.Tok = open
	return out
}

// parseRangeSpecification parses the range-type-specification production:
//
//	range-type-specification = "..", [ ",", derivation-attribute-list ]
//
// This is the typed range declaration `someRange co.lang.int->(..)`, as distinct
// from a range expression such as `1..10`.
func (p *parser) parseRangeSpecification(base typeRef, open scanlex.Token) typeRef {
	p.advance() // ".."

	out := base
	out.Form = formRange
	out.Attrs = p.parseOptionalAttributeTail()
	out.Tok = open
	return out
}

// parseArraySpecification parses the array-specification production:
//
//	array-specification    = array-dimension-group, { array-dimension-group },
//	                         [ ",", derivation-attribute-list ]
//	array-dimension-group  = "[", array-dimension-content, "]"
//
// One group is a plain array, several groups are a jagged array
// (docs/language-ref.md, "Array Declaration"):
//
//	co.lang.int->([5])       single dimension
//	co.lang.int->([2,3])     multi-dimensional
//	co.lang.int->([2][3])    jagged
//	co.lang.int->([...])     variable length
//	co.lang.int->([0])       zero length
//	co.lang.int->([.])       zero dimension
func (p *parser) parseArraySpecification(base typeRef, open scanlex.Token) typeRef {
	out := base
	out.Form = formArray
	out.Tok = open

	groups := 0
	for p.at(scanlex.OPEN_BRACKET) {
		p.advance() // "["
		dims, variable, zeroDim := p.parseArrayDimensionContent()
		p.expect(scanlex.CLOSE_BRACKET, "to close an array dimension group")

		if groups == 0 {
			out.Dims = dims
			out.VariableLength = variable
			out.ZeroDim = zeroDim
		}
		groups++
	}
	out.DimGroups = groups
	out.Attrs = p.parseOptionalAttributeTail()
	return out
}

// parseArrayDimensionContent parses the array-dimension-content production:
//
//	array-dimension-content = [ array-dimension ], { ",", [ array-dimension ] }
//	array-dimension         = "..." | "." | expression
//
// DECISION-TYP-003 allows a dimension to be elided in any position, so ->([,])
// and ->([]) are both well formed. An elided dimension is recorded as a nil entry
// so its position is preserved: ->([,]) is two elided dimensions, not zero.
//
// The two sigil dimensions are returned as flags rather than as expressions:
// "..." is the variable-length array and "." the zero-dimension array.
func (p *parser) parseArrayDimensionContent() (dims []ast.Expr, variable bool, zeroDim bool) {
	// An immediately closing bracket is a single elided dimension: ->([]).
	if p.at(scanlex.CLOSE_BRACKET) {
		return []ast.Expr{nil}, false, false
	}

	for {
		switch {
		case p.at(scanlex.DOT_DOT_DOT):
			p.advance()
			variable = true
			dims = append(dims, nil)
		case p.at(scanlex.DOT):
			p.advance()
			zeroDim = true
			dims = append(dims, nil)
		case p.atAny(scanlex.COMMA, scanlex.CLOSE_BRACKET):
			// An elided dimension between or after commas.
			dims = append(dims, nil)
		default:
			dims = append(dims, p.parseExpression())
		}

		if !p.accept(scanlex.COMMA) {
			return dims, variable, zeroDim
		}
	}
}

// parseBareAttributeDerivation parses the derivation-attribute-list alternative of
// derivation-specification, a derivation that consists only of attributes.
//
// This is the repr/sign/region word form of docs/language-ref.md, "Pointers for
// address manipulation":
//
//	y co.lang.word->(repr=intptr);
//	z co.lang.word->(sign=unsigned, repr=uintptr);
func (p *parser) parseBareAttributeDerivation(base typeRef, open scanlex.Token) typeRef {
	if p.at(scanlex.CLOSE_PAREN) {
		p.fail(open, "an empty type derivation \"->()\" has no meaning; write a derivation such as \"->(*)\" or an attribute such as \"->(repr=intptr)\"")
	}

	out := base
	out.Form = formWord
	out.Attrs = p.parseDerivationAttributeList()
	out.Tok = open
	return out
}

// parseOptionalAttributeTail parses the optional `"," derivation-attribute-list`
// that DECISION-TYP-001 permits on every derivation form.
func (p *parser) parseOptionalAttributeTail() map[string]any {
	if !p.accept(scanlex.COMMA) {
		return nil
	}
	return p.parseDerivationAttributeList()
}

// parseDerivationAttributeList parses the derivation-attribute-list production:
//
//	derivation-attribute-list = derivation-attribute,
//	                            { ",", derivation-attribute }
//	derivation-attribute      = annotation-key, "=", annotation-value
func (p *parser) parseDerivationAttributeList() map[string]any {
	attrs := map[string]any{}
	for {
		key := p.parseAnnotationKey("as a type attribute name")
		p.expectOp("=", "after a type attribute name")
		attrs[key] = p.parseAnnotationValue()

		if !p.accept(scanlex.COMMA) {
			return attrs
		}
	}
}
