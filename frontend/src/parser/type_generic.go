package parser

import (
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// Generic parameter clauses and kind options — section 5 of
// docs/grammar/folang.ebnf.

// parseGenericParameterClause parses the generic-parameter-clause production:
//
//	generic-parameter-clause = "(", generic-parameter,
//	                           { ",", generic-parameter }, ")"
//
// It is the type-parameter list a declaration carries before its kind, as in
// `LinkedList(T) co.lang.struct = { … }`.
//
// Implements: generic-parameter-clause
func (p *parser) parseGenericParameterClause() []symboltable.GenericTypeParam {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	p.expect(scanlex.OPEN_PAREN, "to open a generic-parameter clause")

	params := []symboltable.GenericTypeParam{p.parseGenericParameter()}
	for p.accept(scanlex.COMMA) {
		params = append(params, p.parseGenericParameter())
	}

	p.expect(scanlex.CLOSE_PAREN, "to close a generic-parameter clause")
	return params
}

// parseGenericParameter parses the generic-parameter production:
//
//	generic-parameter = identifier, [ generic-arity-clause ]
//
// The optional arity clause is what makes a parameter higher-kinded
// (DECISION-GEN-001): `Transformer(F(_), G(_))` declares F and G as
// one-argument type constructors rather than as types.
//
// A constraint written with ":" — `T: Orderable` — is also accepted here, because
// the reference uses that spelling for bounded parameters even though the EBNF
// carries bounds in an annotation instead.
//
// Implements: generic-parameter
func (p *parser) parseGenericParameter() symboltable.GenericTypeParam {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	id := p.parseIdentifier("as a generic parameter")
	param := symboltable.GenericTypeParam{Name: id.Scanned}

	if p.at(scanlex.OPEN_PAREN) {
		arity := p.parseGenericArityClause()
		// The arity is recorded as the parameter's kind so the semantic phase can
		// check applications against it: one slot per expected argument.
		param.TypeKind = "type"
		param.Types = joinNames(arity)
	}

	if p.accept(scanlex.COLON) {
		param.Constraint = p.parseTypeExpression().actType()
	}

	return param
}

// parseGenericArityClause parses the generic-arity-clause production:
//
//	generic-arity-clause = "(", generic-arity-slot,
//	                       { ",", generic-arity-slot }, ")"
//	generic-arity-slot   = "_" | identifier
//
// A slot is either the anonymous "_" placeholder or a named one; both only declare
// that an argument is expected at that position.
//
// Implements: generic-arity-clause
func (p *parser) parseGenericArityClause() []string {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	p.expect(scanlex.OPEN_PAREN, "to open a generic-arity clause")

	slots := []string{p.parseGenericAritySlot()}
	for p.accept(scanlex.COMMA) {
		slots = append(slots, p.parseGenericAritySlot())
	}

	p.expect(scanlex.CLOSE_PAREN, "to close a generic-arity clause")
	return slots
}

// parseGenericAritySlot parses one generic-arity-slot.
//
// Implements: generic-arity-slot
func (p *parser) parseGenericAritySlot() string {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	if p.at(scanlex.DISCARD_WILD_VAR) {
		return p.advance().Value
	}
	return p.parseIdentifier("as a generic-arity slot").Scanned
}

// parseOptionalGenericParameterClause parses a generic-parameter-clause when one is
// present.
//
// It must not consume the parameter list of a function declaration, which has the
// same leading "(". The two are distinguished by content: a generic-parameter
// clause holds only bare identifiers, optional arity clauses and ":" constraints,
// whereas a function parameter list holds `name type` pairs. The check is done as
// pure lookahead so nothing is consumed when the answer is no.
func (p *parser) parseOptionalGenericParameterClause() []symboltable.GenericTypeParam {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	if !p.at(scanlex.OPEN_PAREN) || !p.looksLikeGenericParameterClause() {
		return nil
	}
	return p.parseGenericParameterClause()
}

// looksLikeGenericParameterClause reports whether the "(" at the cursor opens a
// generic-parameter clause rather than a function parameter list.
func (p *parser) looksLikeGenericParameterClause() bool {
	return p.lookaheadOnly(func() bool {
		p.advance() // "("
		if p.at(scanlex.CLOSE_PAREN) {
			return false // "()" is an empty parameter list, never a generic clause.
		}
		for {
			if !p.atIdentifier() {
				return false
			}
			p.advance()

			// A nested "(" here is an arity clause, which only a generic
			// parameter has.
			if p.at(scanlex.OPEN_PAREN) {
				p.skipBalanced(scanlex.OPEN_PAREN, scanlex.CLOSE_PAREN)
			} else if p.accept(scanlex.COLON) {
				// A constraint; skip the bounding type.
				if !p.atIdentifier() && !p.at(scanlex.BUILT_IN_TYPE) {
					return false
				}
				p.advance()
			} else if !p.atAny(scanlex.COMMA, scanlex.CLOSE_PAREN) {
				// Anything else after the name — a type, a "=" default — makes
				// this a function parameter list.
				return false
			}

			if p.accept(scanlex.COMMA) {
				continue
			}
			return p.at(scanlex.CLOSE_PAREN)
		}
	})
}

// parseKindOptions parses the kind-options production:
//
//	kind-options = "->", "(", [ annotation-argument-list ], ")"
//
// These are the options a declaration attaches to its kind, as in
// `co.lang.instance->(for=Functor, type=List)` or
// `co.lang.dependentType->(kind=length)`.
//
// Implements: kind-options
func (p *parser) parseKindOptions() map[string]any {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	p.expect(scanlex.ARROW, "to begin a kind-options clause")
	p.expect(scanlex.OPEN_PAREN, "to open a kind-options clause")

	options := map[string]any{}
	if !p.at(scanlex.CLOSE_PAREN) {
		for _, arg := range p.parseAnnotationArgumentList() {
			options[arg.Key] = arg.Value
		}
	}

	p.expect(scanlex.CLOSE_PAREN, "to close a kind-options clause")
	return options
}

// parseOptionalKindOptions parses a kind-options clause when one follows.
func (p *parser) parseOptionalKindOptions() map[string]any {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	if !p.at(scanlex.ARROW) {
		return nil
	}
	return p.parseKindOptions()
}

// joinNames renders a slice of names as a comma-separated string, which is the
// shape GenericTypeParam.Types uses.
func joinNames(names []string) string {
	out := ""
	for i, n := range names {
		if i > 0 {
			out += ","
		}
		out += n
	}
	return out
}
