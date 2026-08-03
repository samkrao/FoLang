package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// primary-declaration — section 1 of docs/grammar/folang.ebnf.
//
//	primary-declaration = struct-declaration | cstruct-declaration
//	                    | enum-declaration | union-declaration | data-declaration
//	                    | class-declaration | interface-declaration
//	                    | signature-declaration | module-declaration
//	                    | unit-declaration | type-declaration
//	                    | object-declaration | instance-declaration
//	                    | matcher-instance-declaration
//	                    | function-object-declaration | delegate-declaration
//	                    | named-block-declaration
//	                    | annotated-contract-declaration
//	                    | annotated-function-primary
//	                    | type-constructor-primary | forward-type-declaration
//	                    | general-kind-declaration | package-alias-declaration
//
// A package source file holds exactly one of these
// (docs/language-ref.md, "Package Source Files"): a user-defined type, a function group, a
// macro, an extension, a template, a typeclass, a type constructor or a unit must each be
// in its own file.
//
// # How the dispatch works
//
// Nearly every alternative has the same prefix — annotations, a name, an optional generic
// clause — and is then identified by a BUILT_IN_KIND token. The scanner has already folded
// `co.lang.struct` and friends into single tokens, so once the prefix is consumed the
// dispatch is a switch on one lexeme. That is why the parse functions in the decl_* files
// take declName, generics and annotations as parameters: the dispatcher had to read them to
// get far enough to choose.
//
// The three alternatives with NO kind token are handled by lookahead before the name is
// consumed:
//
//   - annotated-contract-declaration: annotations, name, [generics], "=" contract-body
//   - type-constructor-primary:       name, parameter-list…, return-type-clause
//   - annotated-function-primary:     annotations, function-declaration

// parsePrimaryDeclaration parses one primary-declaration, reporting a diagnostic if the
// cursor does not begin one.
//
// Implements: primary-declaration
func (p *parser) parsePrimaryDeclaration() ast.Stmt {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	decl, ok := p.tryParsePrimaryDeclaration()
	if !ok {
		p.failf(p.cur(), "expected a declaration, found %s", describeToken(p.cur()))
	}
	return decl
}

// tryParsePrimaryDeclaration parses one primary-declaration and reports whether the cursor
// began one.
//
// It returns false without consuming anything when the cursor does not start a declaration,
// which is what lets an entry file fall through to a statement.
func (p *parser) tryParsePrimaryDeclaration() (ast.Stmt, bool) {
	if !p.atPrimaryDeclaration() {
		return nil, false
	}

	annotations := p.parseAnnotations()

	// type-constructor-primary has no name-then-kind shape: it is a function whose
	// return type is a type, so it is identified before the name is consumed.
	if p.atTypeConstructorPrimary() {
		return p.parseTypeConstructorPrimary(annotations), true
	}

	// annotated-contract-declaration is checked before annotated-function-primary because
	// the two shapes overlap: `Functor(F) = { … }` is admissible as both, since a bare
	// identifier is a valid untyped parameter. What separates them is that a contract's
	// paren group is a generic-parameter clause — bare names only — while a function's is
	// a parameter list of `name type` pairs.
	if p.atAnnotatedContractDeclaration(annotations) {
		declName := p.parseDeclarationName("as a contract name")
		declName = p.resolveKindlessFilenameDerivedName(declName)
		generics := p.parseOptionalGenericParameterClause()
		return p.parseAnnotatedContractDeclaration(declName, generics, annotations), true
	}

	// annotated-function-primary: an annotation promotes a plain function declaration
	// to a primary declaration.
	if !annotations.empty() && p.atFunctionDeclarationShape() {
		return p.parseAnnotatedFunctionPrimary(annotations), true
	}

	declName := p.parseDeclarationName("as a declaration name")
	generics := p.parseOptionalGenericParameterClause()

	// annotated-contract-declaration: annotations and a name, then "=" with no kind
	// token at all.
	if p.atOp("=") {
		if annotations.empty() {
			p.failf(p.cur(), "declaration %q is missing its kind, such as \"co.lang.struct\" or \"co.lang.class\"", declName.Logical)
		}
		declName = p.resolveKindlessFilenameDerivedName(declName)
		return p.parseAnnotatedContractDeclaration(declName, generics, annotations), true
	}

	// Everything else is selected by its built-in kind token.
	if !p.at(scanlex.BUILT_IN_KIND) {
		// A function declaration whose name has already been consumed.
		if p.at(scanlex.OPEN_PAREN) {
			return p.continueFunctionDeclaration(declName, annotations), true
		}
		p.failf(p.cur(), "expected a built-in kind after the declaration name %q, found %s", declName.Logical, describeToken(p.cur()))
	}

	kindTok := p.advance()
	return p.dispatchKindDeclaration(declName, generics, kindTok, annotations), true
}

// dispatchKindDeclaration routes a declaration to the production its built-in kind selects.
//
// A forward declaration is recognised first, because it shares its kind token with the
// block-bodied form and differs only by ending at ";" rather than at "=".
func (p *parser) dispatchKindDeclaration(declName name, generics []symboltable.GenericTypeParam, kindTok scanlex.Token, annotations annotationSet) ast.Stmt {
	// A filename-derived name still carries its kind suffix at this point, and the
	// kind token is what decides whether that suffix is dropped or is an error.
	declName = p.resolveFilenameDerivedName(declName, kindTok)

	// forward-type-declaration: the kind is forward-declarable and no binding follows.
	if isForwardDeclarableKind(kindTok.Value) && p.atForwardDeclarationEnd() {
		return p.parseForwardTypeDeclaration(declName, generics, kindTok, annotations)
	}

	switch kindTok.Value {
	case "co.lang.struct":
		return p.parseStructDeclaration(declName, generics, annotations)
	case "co.lang.cstruct":
		return p.parseCStructDeclaration(declName, generics, annotations)
	case "co.lang.enum":
		return p.parseEnumDeclaration(declName, generics, annotations)
	case "co.lang.union":
		return p.parseUnionDeclaration(declName, generics, annotations)
	case "co.lang.data":
		return p.parseDataDeclaration(declName, generics, annotations)
	case "co.lang.class":
		return p.parseClassDeclaration(declName, generics, annotations)
	case "co.lang.interface":
		return p.parseInterfaceDeclaration(declName, generics, annotations)
	case "co.lang.signature":
		return p.parseSignatureDeclaration(declName, generics, annotations)
	case "co.lang.module":
		return p.parseModuleDeclaration(declName, generics, annotations)
	case "co.lang.unit":
		return p.parseUnitDeclaration(declName, generics, annotations)
	case "co.lang.object":
		return p.parseObjectDeclaration(declName, generics, annotations)
	case "co.lang.instance":
		return p.parseInstanceDeclaration(declName, generics, annotations)
	case "co.lang.matcher":
		return p.parseMatcherInstanceDeclaration(declName, generics, annotations)
	case "co.lang.function":
		return p.parseFunctionObjectDeclaration(declName, generics, annotations)
	case "co.lang.delegate":
		return p.parseDelegateDeclaration(declName, generics, annotations)
	case "co.lang.block":
		return p.parseNamedBlockDeclaration(declName, generics, annotations)
	case "co.lang.library":
		if len(generics) != 0 {
			p.failf(kindTok, "a co.lang.library surface cannot declare generic parameters")
		}
		return p.parseLibraryDeclaration(declName, annotations)
	case "co.lang.package":
		if len(generics) != 0 {
			p.failf(kindTok, "a co.lang.package alias cannot declare generic parameters")
		}
		return p.parsePackageAliasDeclaration(declName, annotations)
	}

	// type-declaration covers the alias family.
	if _, isTypeKind := typeDeclarationKinds[kindTok.Value]; isTypeKind {
		return p.parseTypeDeclaration(declName, generics, kindTok, annotations)
	}

	// DECISION-KIND-001: every remaining built-in kind is parsed by
	// general-kind-declaration rather than falling through to variable-declaration.
	if isGeneralDeclarableKind(kindTok.Value) {
		return p.parseGeneralKindDeclaration(declName, generics, kindTok, annotations)
	}

	p.failf(kindTok, "%q is not a declarable kind", kindTok.Value)
	return nil // unreachable: failf panics
}

// forwardDeclarableKinds is the forward-declarable-kind set of section 6.
var forwardDeclarableKinds = map[string]struct{}{
	"co.lang.struct":    {},
	"co.lang.cstruct":   {},
	"co.lang.class":     {},
	"co.lang.interface": {},
	"co.lang.signature": {},
	"co.lang.module":    {},
	"co.lang.enum":      {},
	"co.lang.union":     {},
	"co.lang.data":      {},
	"co.lang.object":    {},
	"co.lang.instance":  {},
	"co.lang.function":  {},
}

// isForwardDeclarableKind reports whether a kind may be forward declared.
func isForwardDeclarableKind(kind string) bool {
	_, ok := forwardDeclarableKinds[kind]
	return ok
}

// atForwardDeclarationEnd reports whether a declaration ends without a binding, which makes
// it a forward declaration.
//
// The kind may carry options before the terminator, so a "->" is skipped before the ";" is
// looked for.
func (p *parser) atForwardDeclarationEnd() bool {
	if p.at(scanlex.SEMI_COLON) {
		return true
	}
	if !p.at(scanlex.ARROW) {
		return false
	}
	return p.lookaheadOnly(func() bool {
		p.advance() // "->"
		if !p.at(scanlex.OPEN_PAREN) {
			return false
		}
		p.skipBalanced(scanlex.OPEN_PAREN, scanlex.CLOSE_PAREN)
		return p.at(scanlex.SEMI_COLON)
	})
}

// atPrimaryDeclaration reports whether the cursor begins a primary-declaration.
//
// The test is deliberately structural: a declaration starts with annotations or a name, and
// what makes it a declaration rather than a statement is that a built-in kind token, a
// contract binding, or a function shape follows.
func (p *parser) atPrimaryDeclaration() bool {
	if !p.atAnnotation() && !p.atIdentifier() && !p.at(scanlex.DISCARD_WILD_VAR) {
		return false
	}

	return p.lookaheadOnly(func() bool {
		annotated := p.atAnnotation()
		for p.atAnnotation() {
			p.advance()
			if p.at(scanlex.OPEN_PAREN) {
				p.skipBalanced(scanlex.OPEN_PAREN, scanlex.CLOSE_PAREN)
			}
		}

		if !p.atIdentifier() && !p.at(scanlex.DISCARD_WILD_VAR) && !p.atLifecycleName() {
			return false
		}
		if p.atLifecycleName() {
			p.advance()
		}
		p.advance() // the name

		// A generic clause or a parameter list may follow.
		sawParens := false
		for p.at(scanlex.OPEN_PAREN) {
			p.skipBalanced(scanlex.OPEN_PAREN, scanlex.CLOSE_PAREN)
			sawParens = true
		}

		switch {
		// The common case: a built-in kind token identifies the declaration.
		case p.at(scanlex.BUILT_IN_KIND):
			return true

		// annotated-contract-declaration: annotations, a name and "=" with a braced
		// body, and no kind token.
		case annotated && p.atOp("="):
			return true

		// A function-shaped declaration: a parameter list and then a return-type
		// clause or a body.
		case sawParens && p.at(scanlex.ARROW):
			return true
		case sawParens && annotated && (p.atOp("=") || p.at(scanlex.OPEN_CURLY)):
			return true
		}
		return false
	})
}

// atTypeConstructorPrimary reports whether the cursor begins a type-constructor-primary.
//
// A type constructor is a function whose return-type clause names a type-producing kind, so
// the probe looks for the `name ( … ) -> ( kind )` shape with a type kind inside the return
// clause.
func (p *parser) atTypeConstructorPrimary() bool {
	if !p.atIdentifier() {
		return false
	}
	return p.lookaheadOnly(func() bool {
		p.advance() // the name
		if !p.at(scanlex.OPEN_PAREN) {
			return false
		}
		for p.at(scanlex.OPEN_PAREN) {
			p.skipBalanced(scanlex.OPEN_PAREN, scanlex.CLOSE_PAREN)
		}
		if !p.at(scanlex.ARROW) {
			return false
		}
		p.advance()
		if !p.at(scanlex.OPEN_PAREN) {
			return false
		}
		p.advance()
		// Recognize the otherwise invalid named-result spelling as constructor-shaped
		// too, so parseTypeConstructorPrimary can issue the precise rule that a type
		// constructor has one unnamed type result instead of accepting it as a loose
		// ordinary function.
		if p.atIdentifier() && p.startsTypeExpression(p.peek(1)) {
			p.advance()
		}
		// The return type must be one of the type-producing kinds for this to be a
		// type constructor rather than an ordinary function.
		_, typeProducing := typeConstructorResultKinds[p.lexeme()]
		return typeProducing
	})
}

// atAnnotatedContractDeclaration reports whether the cursor begins an
// annotated-contract-declaration.
//
// The production is `one-or-more-annotations, declaration-name,
// [ generic-parameter-clause ], "=", contract-body`, and the annotation is what supplies the
// kind, since there is no kind token. The paren group, when present, must be a
// generic-parameter clause: that is what distinguishes
//
//	@co.dap.Functor
//	Functor(F) = { map(value F, f co.lang.int) -> (F); }
//
// from an annotated function declaration such as
//
//	@co.dap.inline
//	add(a co.lang.int, b co.lang.int) = { … }
//
// where the paren group is a parameter list of `name type` pairs.
func (p *parser) atAnnotatedContractDeclaration(annotations annotationSet) bool {
	if annotations.empty() {
		return false
	}
	if !p.atIdentifier() && !p.at(scanlex.DISCARD_WILD_VAR) {
		return false
	}
	return p.lookaheadOnly(func() bool {
		p.advance() // the name

		if p.at(scanlex.OPEN_PAREN) {
			if !p.looksLikeGenericParameterClause() {
				return false
			}
			p.skipBalanced(scanlex.OPEN_PAREN, scanlex.CLOSE_PAREN)
		}
		if !p.atOp("=") {
			return false
		}
		p.advance()
		return p.at(scanlex.OPEN_CURLY)
	})
}

// atFunctionDeclarationShape reports whether the cursor begins a function declaration, used
// to recognise annotated-function-primary.
func (p *parser) atFunctionDeclarationShape() bool {
	if p.atReceiverClause() {
		return true
	}
	if !p.atIdentifier() && !p.atLifecycleName() {
		return false
	}
	return p.lookaheadOnly(func() bool {
		if p.atLifecycleName() {
			p.advance()
		}
		p.advance() // the name
		if !p.at(scanlex.OPEN_PAREN) {
			return false
		}
		for p.at(scanlex.OPEN_PAREN) {
			p.skipBalanced(scanlex.OPEN_PAREN, scanlex.CLOSE_PAREN)
		}
		// A kind token here means this is a kind declaration, not a function.
		if p.at(scanlex.BUILT_IN_KIND) {
			return false
		}
		return p.at(scanlex.ARROW) || p.atOp("=") || p.atOp("=>") || p.atOp("=>>") ||
			p.at(scanlex.OPEN_CURLY) || p.at(scanlex.SEMI_COLON)
	})
}
