package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// general-kind-declaration — section 6.
//
//	general-kind-declaration = annotations, declaration-name,
//	                           [ generic-parameter-clause ],
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
// DECISION-KIND-001 is why this production exists. Around 28 built-in kinds have no
// dedicated production of their own, and without a catch-all they were absorbed by
// variable-declaration, so a declaration like
//
//	blockormacro co.lang.kind = block | macro;
//
// silently parsed as a variable. This production stops that: every remaining built-in kind
// is parsed here, accepting a block body, a type-expression body, an expression body, or a
// forward form.
//
// Ordered choice matters. A specific declaration from section 6 or 7 is tried first, and in
// statement position variable-declaration is preferred, so an ordinary declarator is
// unaffected. The kinds that also name built-in DATA types — co.lang.value, co.lang.nothing
// and co.lang.just — are deliberately excluded, because in a declarator they read as types.

// generalDeclarableKinds is the general-declarable-kind set.
//
// It is the built-in kind table minus the kinds that have their own production and minus the
// three that double as data types, which DECISION-KIND-001 excludes.
var generalDeclarableKinds = map[string]struct{}{
	"co.lang.realm":         {},
	"co.lang.loader":        {},
	"co.lang.role":          {},
	"co.lang.record":        {},
	"co.lang.property":      {},
	"co.lang.indexer":       {},
	"co.lang.trait":         {},
	"co.lang.mixin":         {},
	"co.lang.extension":     {},
	"co.lang.typeclass":     {},
	"co.lang.concept":       {},
	"co.lang.macro":         {},
	"co.lang.template":      {},
	"co.lang.lambda":        {},
	"co.lang.behavior":      {},
	"co.lang.method":        {},
	"co.lang.operator":      {},
	"co.lang.namespace":     {},
	"co.lang.stex":          {},
	"co.lang.kind":          {},
	"co.lang.level":         {},
	"co.lang.order":         {},
	"co.lang.rank":          {},
	"co.lang.dependentType": {},
	"co.lang.hokrlt":        {},
	"co.lang.hokrtl":        {},
	"co.lang.typetype":      {},
	"co.lang.typekind":      {},
	"co.lang.alias":         {},
}

// isGeneralDeclarableKind reports whether a kind token is handled by
// general-kind-declaration.
func isGeneralDeclarableKind(kind string) bool {
	_, ok := generalDeclarableKinds[kind]
	return ok
}

// parseGeneralKindDeclaration parses the general-kind-declaration production.
func (p *parser) parseGeneralKindDeclaration(declName name, generics []symboltable.GenericTypeParam, kindTok scanlex.Token, annotations annotationSet) ast.Stmt {
	options := p.parseOptionalKindOptions()

	symb := p.typeSymbol(declName.Scanned)
	symb.TypeType = kindTok.Value
	symb.IsGenericType = len(generics) > 0

	decl := ast.TypeDeclarationStmt{
		Name:       declName.Scanned,
		TypeParams: generics,
		Kind:       kindTok.Value,
		SubType_:   "KIND",
		Typetype:   "UDT",
		SDapst:     annotations.list(),
		KDapst:     annotations.list(),
		Symb:       symb,
	}
	if kind := firstOptionString(options, "kind"); kind != "" {
		decl.DependentKind = kind
	}

	// An operator kind declares a new operator symbol, which the Pratt engine has to
	// know about before any use of it is parsed (DECISION-EXT-001).
	if kindTok.Value == "co.lang.operator" {
		p.registerKindDeclaredOperator(declName, options, annotations)
	}

	return p.parseGeneralKindBinding(decl)
}

// parseGeneralKindBinding parses the general-kind-binding production.
//
// The four alternatives are separated by their leading token: a "{" after "=" is a block
// body, and after "=" without a brace the type-expression reading is tried before the
// expression reading, matching the priority DECISION-TYP-002 sets for type positions.
func (p *parser) parseGeneralKindBinding(decl ast.TypeDeclarationStmt) ast.Stmt {
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
		ast.ExpressionStmt{Expression: value, Symb: p.stmtSymbol("kind-binding")},
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

		decl.Type_ = t.Node
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
func (p *parser) parseGeneralKindMember() ast.Stmt {
	annotations := p.parseAnnotations()

	switch {
	case p.atSignatureTypeComponent():
		return p.parseSignatureTypeComponent(annotations)

	case p.atMemberFunctionDeclaration():
		// A specification ends at ";" while a declaration has a body, and
		// parseFunctionDeclaration already treats a ";" binding as a forward
		// declaration, so one call covers both alternatives.
		return p.parseFunctionDeclaration(annotations)

	case p.atEmbeddedField():
		return p.parseEmbeddedFieldDeclaration(annotations)

	default:
		return p.parseFieldDeclaration(annotations)
	}
}

// registerKindDeclaredOperator registers an operator declared through the
// co.lang.operator kind.
//
// The fixity, precedence and associativity come from the kind options, which is the other
// spelling DECISION-EXT-001 admits alongside the @co.dap.operator annotation:
//
//	myOp co.lang.operator->(symbol="<+>", fixity=infix, precedence=65) = …
func (p *parser) registerKindDeclaredOperator(declName name, options map[string]any, annotations annotationSet) {
	merged := map[string]any{}
	for key, value := range options {
		merged[key] = value
	}
	for _, key := range []string{"symbol", "mode", "fixity", "precedence", "associativity", "arity"} {
		if _, exists := merged[key]; exists {
			continue
		}
		if value, ok := annotations.option("@co.dap.operator", key); ok {
			merged[key] = value
		}
	}
	p.registerOperatorDeclaration(merged, "a co.lang.operator declaration")
}

// library-declaration — section 7.
//
//	library-declaration = annotations, declaration-name, "co.lang.library", "=",
//	                      library-body
//	library-body        = "{", { library-member }, body-close
//	library-member      = import-directive
//	                    | struct-declaration
//	                    | cstruct-declaration
//	                    | function-declaration
//
// A library surface file declares the library's public boundary. Its members are restricted
// to imports, the two struct kinds and functions, because a surface may only expose types
// whose signatures are closed over the boundary (docs/language-ref.md, "Allowed Surface
// Declarations").

// parseLibraryDeclaration parses the library-declaration production.
func (p *parser) parseLibraryDeclaration(declName name, annotations annotationSet) ast.Stmt {
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

	return ast.Library{
		Body: members,
		Symb: p.librarySymbol(declName.Scanned, libType),
	}
}

// parseLibraryMember parses the library-member production.
func (p *parser) parseLibraryMember() ast.Stmt {
	annotations := p.parseAnnotations()

	// An import directive inside a surface body.
	if !annotations.empty() && p.startsNothingAfterAnnotations() {
		return p.annotationOnlyStatement(annotations)
	}

	declName := p.parseDeclarationName("as a library member name")
	generics := p.parseOptionalGenericParameterClause()

	if p.at(scanlex.BUILT_IN_KIND) {
		kindTok := p.advance()
		switch kindTok.Value {
		case "co.lang.struct":
			return p.parseStructDeclaration(declName, generics, annotations)
		case "co.lang.cstruct":
			return p.parseCStructDeclaration(declName, generics, annotations)
		}
		p.failf(kindTok, "a library surface admits only imports, struct and cstruct declarations, and functions; %q is not allowed here", kindTok.Value)
	}

	// A function declaration: the name has already been consumed, so the declaration
	// continues from its parameter list.
	return p.continueFunctionDeclaration(declName, annotations)
}
