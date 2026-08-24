package parser

import (
	"github.com/samkrao/fo-lang/src/ast"
	symboltable "github.com/samkrao/fo-lang/src/context"
	"github.com/samkrao/fo-lang/src/scanlex"
)

// Common declaration components — section 5.
//
//	field-declaration          = annotations, identifier, type-expression,
//	                             [ "=", expression ], statement-end
//	embedded-field-declaration = annotations, type-expression, statement-end
//	value-specification        = annotations, identifier, type-expression,
//	                             statement-end
//
// These are the members that declaration bodies are made of. A field has a name; an
// embedded field has only a type, which is how a struct composes another type into
// itself; a value specification is a field with no initializer, which is what a
// signature body holds.

// parseStructMember parses the struct-member production:
//
//	struct-member = pure-field-declaration | embedded-field-declaration
//
// The two are told apart by whether a name precedes the type. An embedded field is
// just a type, so `Base;` embeds Base while `id co.lang.int;` declares a field named
// id.
//
// Implements: struct-member
func (p *parser) parseStructMember() ast.Stmt {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	annotations := p.parseAnnotations()
	p.rejectNestedKindDeclaration("a struct body")
	p.rejectOperatorPlacement(annotations, "a struct field")

	if p.atEmbeddedField() {
		return p.parseEmbeddedFieldDeclaration(annotations)
	}
	return p.parsePureFieldDeclaration(annotations, "struct")
}

// parsePureFieldDeclaration parses the pure-field-declaration production:
//
//	pure-field-declaration = annotations, identifier, type-expression, statement-end
//
// This is field-declaration WITHOUT the initializer option. A struct is pure data —
// docs/language-ref.md, "Struct Rules": structs cannot have default values to
// fields/members — and a cstruct is a further-restricted C-compatible representation,
// so both bodies parse their fields here. An initializer is reported rather than
// silently accepted, and its expression is still consumed so one illegal default does
// not cascade into follow-on errors.
//
// Implements: pure-field-declaration
func (p *parser) parsePureFieldDeclaration(annotations annotationSet, owner string) ast.Stmt {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	fieldName := p.parseIdentifier("as a field name")
	t := p.parseTypeExpression()

	if p.atOp("=") {
		p.reportf(p.cur(), "a %s field cannot have a default value; a %s is pure data, so %q must be initialized at construction instead", owner, owner, fieldName.Logical)
		p.advance() // "="
		p.parseExpression()
	}

	p.statementEnd("a field declaration")

	decl := p.lowerDeclarator(fieldName, t, nil, annotations)
	markAsField(decl)
	return decl
}

// atEmbeddedField reports whether the cursor begins an
// embedded-field-declaration rather than a named field.
//
// An embedded field is a type followed directly by ";", so the check is that no second
// type follows the first token.
func (p *parser) atEmbeddedField() bool {
	// Probe the actual type-expression production rather than duplicating a
	// one-token approximation of it. Embedded fields may be generic, quantified,
	// union, function or grouped/derived types; the old scanner skipped at most
	// one postfix group and consequently misclassified valid forms such as
	// `(Element->(*))->(&);` as named fields.
	return p.lookaheadOnly(func() bool {
		p.parseTypeExpression()
		return p.at(scanlex.SEMI_COLON)
	})
}

// parseFieldDeclaration parses the field-declaration production.
//
// Implements: field-declaration
func (p *parser) parseFieldDeclaration(annotations annotationSet) ast.Stmt {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	fieldName := p.parseIdentifier("as a field name")
	t := p.parseTypeExpression()

	var value ast.Expr
	if p.acceptOp("=") {
		value = p.parseExpression()
	}

	p.statementEnd("a field declaration")

	decl := p.lowerDeclarator(fieldName, t, value, annotations)
	markAsField(decl)
	return decl
}

// parseEmbeddedFieldDeclaration parses the embedded-field-declaration production.
//
// An embedded field composes another type into the enclosing one. It has no name of its
// own, so the type's name is used as the declarator's, which is how the composed
// members are later addressed.
//
// Implements: embedded-field-declaration
func (p *parser) parseEmbeddedFieldDeclaration(annotations annotationSet) ast.Stmt {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	t := p.parseTypeExpression()
	p.statementEnd("an embedded field declaration")

	symb := p.varSymbol(t.actType(), t.actType())
	symb.ExplicitType = true
	p.declareAs(p.cur(), t.actType(), symb)

	return ast.VarDeclarationStmt{Span: p.spanFrom(spanStart), BasicVarStmt: ast.BasicVarStmt{
		Identifier: t.actType(),
		// An embedded field has no specialised declaration node on which to
		// record a derivation, so its type slot must carry the complete type.
		Type_:   t.fullType(),
		VarType: t.actType(),
		SDapst:  annotations.list(),
	},
		Symb: symb,
	}
}

// parseValueSpecification parses the value-specification production:
//
//	value-specification = annotations, identifier, type-expression, statement-end
//
// A value specification declares that a value of some type must exist, with no
// initializer. It is what a signature body uses to require a value of an
// implementation.
//
// Implements: value-specification
func (p *parser) parseValueSpecification(annotations annotationSet) ast.Stmt {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	valueName := p.parseIdentifier("as a value specification name")
	t := p.parseTypeExpression()
	p.statementEnd("a value specification")

	decl := p.lowerDeclarator(valueName, t, nil, annotations)
	markAsField(decl)
	return decl
}

// markAsField records that a declarator is a member of a type rather than a local
// variable. The distinction matters for storage and visibility.
func markAsField(decl ast.Stmt) {
	switch d := decl.(type) {
	case ast.VarDeclarationStmt:
		d.Symb.IsParam = false
	case ast.ArrayVariableDeclStmt:
		d.Symb.IsParam = false
	case ast.PointerVariableDeclStmt:
		d.Symb.IsParam = false
	}
}

// parseMemberList reads a "}"-terminated run of members using parseMember for each.
//
// Member-level error recovery lives here, so a malformed member costs that member and
// the rest of the body still parses and still reports.
func (p *parser) parseMemberList(context string, parseMember func() ast.Stmt) []ast.Stmt {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	var members []ast.Stmt

	for !p.at(scanlex.CLOSE_CURLY) && !p.atEOF() {
		// Declaration-member grammars have no empty-member production. Executable
		// blocks still accept empty statements through parseStatement, but silently
		// discarding a semicolon here would make malformed class/struct/unit/etc.
		// bodies look conforming.
		if p.accept(scanlex.SEMI_COLON) {
			p.reportf(p.toks[p.pos-1], "a bare semicolon is not a declaration member; remove it from %s", context)
			continue
		}

		startPos := p.pos
		var member ast.Stmt
		ok := p.recoverItem(startPos, syncMember, func() {
			member = parseMember()
		})
		if ok && member != nil {
			members = append(members, member)
		}
	}
	return members
}

// parseBracedBody reads a "{" … "}" body of members and applies body-close.
//
//	body-close = "}", body-closure-guard
//
// The guard rejects an immediately following ";", because a declaration body ends
// structurally at its brace and takes no terminator (DECISION-SYN-006).
//
// The body's brace opens a new context, and kind is the scope kind that context
// is recorded under. That kind also selects the context's resolution policy, so a
// container body whose members may refer to one another in any order must not be
// given a declaration-ordered one. Members are not segmented the way block
// statements are: a container body is one visibility region, not a sequence of
// frontiers. See scope.go.
func (p *parser) parseBracedBody(kind symboltable.SymbolsToString, context string, parseMember func() ast.Stmt) []ast.Stmt {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	defer p.pushContext(kind)()

	p.expect(scanlex.OPEN_CURLY, "to open "+context)
	members := p.parseMemberList(context, parseMember)
	p.expect(scanlex.CLOSE_CURLY, "to close "+context)
	p.bodyClosureGuard(context)
	return members
}
