package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// general-kind-declaration — section 6.
//
//	general-kind-declaration = annotations, filename-derived-name,
//	                           general-declarable-kind, [ kind-options ],
//	                           general-kind-binding
//	general-kind-binding     = "=", general-kind-block
//	                         | "=", type-expression, statement-end
//	                         | "=", non-block-expression, statement-end
//	                         | statement-end
//	general-kind-block       = "{", { general-kind-member }, body-close
//	general-kind-member      = field-declaration
//	                         | embedded-field-declaration
//	                         | signature-type-component
//	                         | function-declaration
//	                         | function-specification
//
// DECISION-KIND-001 is why this production exists. Many built-in kinds have no
// dedicated production of their own, and without a catch-all they were absorbed by
// variable-declaration, so a declaration like
//
//	_ co.lang.kind = block | macro;
//
// silently parsed as a variable. This production stops that, accepting a block body, a
// type-expression body, an expression body, or a forward form.
//
// Revision 26 narrowed the rule from "every remaining built-in kind" to "every
// ENABLED kind this list enumerates". A kind name that is reserved but has no
// specified syntax or semantics — co.lang.typeconstructor and co.lang.typefunction
// — is deliberately absent, so it is rejected as unsupported rather than silently
// given a general body it was never specified to have.
//
// Ordered choice matters. A specific declaration from section 6 or 7 is tried first, and in
// statement position variable-declaration is preferred, so an ordinary declarator is
// unaffected. The kinds that also name built-in DATA types — co.lang.value, co.lang.nothing
// and co.lang.just — are excluded for the same reason: in a declarator they read as types.

// generalDeclarableKinds is the general-declarable-kind set.
//
// It is the built-in kind table minus the kinds that have their own production,
// minus the three that double as data types, and minus the reserved future names.
var generalDeclarableKinds = map[string]struct{}{
	"co.lang.loader":    {},
	"co.lang.role":      {},
	"co.lang.record":    {},
	"co.lang.property":  {},
	"co.lang.indexer":   {},
	"co.lang.trait":     {},
	"co.lang.mixin":     {},
	"co.lang.extension": {},
	"co.lang.concept":   {},
	"co.lang.macro":     {},
	"co.lang.template":  {},
	"co.lang.lambda":    {},
	"co.lang.behavior":  {},
	"co.lang.method":    {},
	"co.lang.namespace": {},
	"co.lang.stex":      {},
	"co.lang.kind":      {},
	"co.lang.level":     {},
	"co.lang.order":     {},
	"co.lang.rank":      {},
	"co.lang.hokrlt":    {},
	"co.lang.alias":     {},
}

// isGeneralDeclarableKind reports whether a kind token is handled by
// general-kind-declaration.
func isGeneralDeclarableKind(kind string) bool {
	_, ok := generalDeclarableKinds[kind]
	return ok
}

// parseGeneralKindDeclaration parses the general-kind-declaration production.
//
// Implements: general-kind-declaration
func (p *parser) parseGeneralKindDeclaration(declName name, kindTok scanlex.Token, annotations annotationSet) ast.Stmt {
	spanStart := p.pos
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	options := p.parseOptionalKindOptions()

	symb := p.typeSymbol(declName.Scanned)
	symb.TypeType = kindTok.Value
	symb.IsGenericType = annotations.has("@co.dap.generic")

	decl := ast.TypeDeclarationStmt{Span: p.spanFrom(spanStart), Name: declName.Scanned,
		Kind:     kindTok.Value,
		SubType_: "KIND",
		Typetype: "UDT",
		SDapst:   annotations.list(),
		KDapst:   annotations.list(),
		Symb:     symb,
	}
	if kind := firstOptionString(options, "kind"); kind != "" {
		decl.DependentKind = kind
	}

	return p.parseGeneralKindBinding(decl)
}

// parseGeneralKindBinding parses the general-kind-binding production.
//
// The four alternatives are separated by their leading token: a "{" after "=" is a block
// body, and after "=" without a brace the type-expression reading is tried before the
// expression reading, matching the priority DECISION-TYP-002 sets for type positions.
//
// Implements: general-kind-binding
func (p *parser) parseGeneralKindBinding(decl ast.TypeDeclarationStmt) ast.Stmt {
	spanStart := p.pos
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	if !p.atOp("=") {
		// The forward form: statement-end alone.
		p.statementEnd("a kind declaration")
		return decl
	}
	p.advance() // "="

	// general-kind-block.
	if p.at(scanlex.OPEN_CURLY) && p.startsDirectBody() {
		decl.Body = p.parseBracedBody("a kind declaration body", p.parseGeneralKindMember)
		return decl
	}

	// The type-expression alternative, tried first.
	if bound, ok := p.tryGeneralKindTypeBinding(decl); ok {
		return bound
	}

	// The expression alternative.
	value := p.parseExpression()
	p.statementEnd("a kind declaration")
	decl.Body = []ast.Stmt{
		ast.ExpressionStmt{Span: p.spanFrom(spanStart), Expression: value, Symb: p.stmtSymbol("kind-binding")},
	}
	return decl
}

// tryGeneralKindTypeBinding attempts the `"=", type-expression, statement-end`
// alternative of general-kind-binding.
//
// This is what makes `blockormacro co.lang.kind = block | macro;` parse as a kind whose
// definition is a union of two kinds, rather than as a variable initialized with a bitwise
// OR.
func (p *parser) tryGeneralKindTypeBinding(decl ast.TypeDeclarationStmt) (ast.Stmt, bool) {
	var bound ast.Stmt

	matched := p.speculate(func() bool {
		t := p.parseTypeExpression()
		if !p.at(scanlex.SEMI_COLON) {
			return false
		}
		p.advance()

		// A type-bodied kind has no derivation-specific declaration statement;
		// keep the complete bound type in its Type_ slot.
		decl.Type_ = t.fullType()
		decl.NewTypeName = t.actType()
		if t.Form == formUnion {
			decl.ADT_ = t.actType()
			decl.Typetype = "ADT"
		}
		bound = decl
		return true
	})

	return bound, matched
}

// parseGeneralKindMember parses the general-kind-member production.
//
// The body of a general kind is the most permissive in the grammar: it admits fields,
// embedded fields, type components, full function declarations and bare function
// specifications, so the dispatch has to check for each in turn.
//
// Implements: general-kind-member
func (p *parser) parseGeneralKindMember() ast.Stmt {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	annotations := p.parseAnnotations()

	switch {
	case p.atSignatureTypeComponent():
		p.rejectOperatorPlacement(annotations, "a general-kind type component")
		return p.parseSignatureTypeComponent(annotations)

	case p.atMemberFunctionDeclaration():
		// A specification ends at ";" while a declaration has a body, and
		// parseFunctionDeclaration already treats a ";" binding as a forward
		// declaration, so one call covers both alternatives.
		p.rejectOperatorPlacement(annotations, "a general-kind declaration")
		return p.parseFunctionDeclaration(annotations)

	case p.atEmbeddedField():
		p.rejectOperatorPlacement(annotations, "a general-kind field")
		return p.parseEmbeddedFieldDeclaration(annotations)

	default:
		p.rejectOperatorPlacement(annotations, "a general-kind field")
		return p.parseFieldDeclaration(annotations)
	}
}

// library-declaration — section 7.
//
//	library-declaration        = annotations, filename-derived-name,
//	                             "co.lang.library", "=", library-body
//	library-body               = "{", { library-member }, body-close
//	library-member             = import-directive
//	                           | surface-struct-declaration
//	                           | surface-cstruct-declaration
//	                           | function-declaration
//	surface-struct-declaration = annotations, identifier, "co.lang.struct", "=",
//	                             struct-body
//	surface-cstruct-declaration = annotations, identifier, "co.lang.cstruct",
//	                             "=", cstruct-body
//
// A library surface file declares the library's public boundary. Its members are restricted
// to imports, the two struct kinds and functions, because a surface may only expose types
// whose signatures are closed over the boundary (docs/language-ref.md, "Allowed Surface
// Declarations").
//
// DECISION-FILE-003: the surface struct forms are two of the six stated
// exceptions to filename-derived naming. One surface file carries SEVERAL
// declarations, so one filename cannot name them and each keeps an explicit
// identifier in its head. The surface itself is file-backed and still uses "_".

// parseLibraryDeclaration parses the library-declaration production.
//
// Implements: library-declaration
func (p *parser) parseLibraryDeclaration(declName name, annotations annotationSet) ast.Stmt {
	spanStart := p.pos
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	options := p.parseOptionalKindOptions()

	p.expectOp("=", "before a library body")
	members := p.parseBracedBody("a library body", p.parseLibraryMember)

	libType := firstOptionString(options, "type")
	if libType == "" {
		libType = annotations.optionString("@co.dap.library", "type")
	}
	if libType == "" {
		libType = "application"
	}

	// The restricted-import rules are stated in terms of the surface's identity, so it is
	// recorded for the check that runs once the file has been read.
	p.libraryName = declName.Logical
	p.libraryType = libType

	// A source library has to be built before its consumers, so the driver is told.
	p.buildLibs = true

	return ast.Library{Span: p.spanFrom(spanStart), Body: members,
		Symb: p.librarySymbol(declName.Scanned, libType),
	}
}

// parseLibraryMember parses the library-member production.
//
// Implements: library-member
// Implements: surface-struct-declaration
// Implements: surface-cstruct-declaration
func (p *parser) parseLibraryMember() ast.Stmt {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	// Imports are members in their own right, not annotations decorating the
	// declaration that follows. Parsing them through the directive production is
	// essential: it validates the closed field set and records the dependency edge
	// used by restricted-import and cycle checks.
	if p.atFileDirective() {
		if p.lexeme() == "@co.ddap.import" {
			return p.parseImportDirective()
		}
		p.failf(p.cur(), "a library body admits import directives but not %q", p.lexeme())
	}

	annotations := p.parseAnnotations()

	// A surface member is named in the source, not by the filename, because one
	// surface file carries several of them (DECISION-FILE-003).
	declName := p.parseIdentifier("as a library member name")

	if p.at(scanlex.BUILT_IN_KIND) {
		kindTok := p.advance()
		p.rejectOperatorPlacement(annotations, "a library type member")
		switch kindTok.Value {
		case "co.lang.struct":
			return p.parseStructDeclaration(declName, annotations)
		case "co.lang.cstruct":
			return p.parseCStructDeclaration(declName, annotations)
		}
		p.failf(kindTok, "a library surface admits only imports, struct and cstruct declarations, and functions; %q is not allowed here", kindTok.Value)
	}

	// A function declaration: the name has already been consumed, so the declaration
	// continues from its parameter list.
	if annotations.has("@co.dap.operator") {
		p.reportf(declName.Tok, "an operator function cannot be exported from a library surface; declare it in a class or a struct companion unit")
	}
	return p.continueFunctionDeclaration(declName, annotations)
}
