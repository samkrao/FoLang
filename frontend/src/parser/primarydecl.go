package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// primary-declaration — section 1 of docs/grammar/folang.ebnf.
//
//	primary-declaration = struct-declaration | cstruct-declaration
//	                    | enum-declaration | union-declaration
//	                    | class-declaration | interface-declaration
//	                    | signature-declaration | module-declaration
//	                    | typeclass-declaration
//	                    | object-declaration | instance-declaration
//	                    | matcher-instance-declaration
//	                    | annotated-function-primary
//	                    | forward-type-declaration
//
// This is the ordinary `<Name>.fol` root. A package source file holds exactly one
// primary declaration and its name comes from the filename, so every alternative
// here spells filename-derived-name — "_" — in the declaration head
// (docs/language-ref.md, "File-Backed Primary Declarations").
//
// Four declaration families were removed from this set in revision 23 because
// they belong to a different source form or a different container:
//
//	unit-declaration          <Fragment>.unit.fol or <Name>.comp.unit.fol
//	package-alias-declaration the reserved package.fol source form
//	data-declaration          a member of an ordinary unit
//	type-declaration          a member of an ordinary unit, and
//	type-level-function       likewise; both name themselves in their head
//
// typeclass-declaration was added in their place: co.lang.typeclass is now a
// dedicated production rather than a general kind, because a typeclass is the
// one primary that carries a parameter clause of its own.
//
// DECISION-DECL-001 closed the set in revision 27 to the top-level kinds
// docs/language-ref.md enumerates, removing five more alternatives:
//
//	function-object-declaration    a unit member (DECISION-DECL-002)
//	delegate-declaration           a unit member (DECISION-DECL-002)
//	named-block-declaration        a statement (DECISION-DECL-003)
//	annotated-contract-declaration a pre-co.lang.typeclass spelling, superseded
//	                               by typeclass-declaration
//	general-kind-declaration       a catch-all that admitted about twenty-eight
//	                               further co.lang.* kinds the reference lists
//	                               only in its built-in kind table and gives no
//	                               declaration form
//
// forward-type-declaration stays: it is the forward spelling of a struct or a
// class rather than an additional kind.
//
// # How the dispatch works
//
// Nearly every alternative has the same prefix — annotations then "_" — and is
// then identified by a BUILT_IN_KIND token. The scanner has already folded
// `co.lang.struct` and friends into single tokens, so once the prefix is
// consumed the dispatch is a switch on one lexeme. That is why the parse
// functions in the decl_* files take declName and annotations as parameters: the
// dispatcher had to read them to get far enough to choose.
//
// annotated-function-primary is the one alternative with NO kind token, so it is
// recognised by its function shape before the kind switch.

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

	// annotated-function-primary: an annotation promotes a plain function declaration
	// to a primary declaration. It is checked before the "_" head is required,
	// because it is the one primary that keeps an ordinary function name.
	if !annotations.empty() && p.atFunctionDeclarationShape() {
		return p.parseAnnotatedFunctionPrimary(annotations), true
	}

	declName := p.parseFilenameDerivedName("a primary declaration")

	// DECISION-TCLASS-001: "_" and the typeclass parameter clause are separate
	// grammar components, and typeclass-declaration is the only primary that has
	// such a clause. Reading it here — before the kind is known — is what lets
	// DECISION-GEN-001 be reported precisely for every other kind.
	clauseTok := p.cur()
	var typeclassParams []symboltable.GenericTypeParam
	hasParameterClause := false
	if p.at(scanlex.OPEN_PAREN) && p.looksLikeGenericParameterClause() {
		typeclassParams = p.parseTypeclassParameterClause()
		hasParameterClause = true
	}

	// DECISION-DECL-001: every remaining alternative but annotated-function-primary
	// is selected by a kind token, so a binding here declares nothing. The
	// annotated spelling was annotated-contract-declaration until revision 27
	// superseded it, and it is worth naming its replacement rather than reporting
	// a bare missing kind.
	if p.atOp("=") {
		if !annotations.empty() {
			p.failf(p.cur(), "a contract defined by its annotations alone is no longer a primary declaration; write the typeclass form \"_ (T) co.lang.typeclass = { … }\"")
		}
		p.failf(p.cur(), "this declaration is missing its kind, such as \"co.lang.struct\" or \"co.lang.class\"")
	}

	// Everything else is selected by its built-in kind token.
	if !p.at(scanlex.BUILT_IN_KIND) {
		p.failf(p.cur(), "expected a built-in kind after the declaration name \"_\", found %s", describeToken(p.cur()))
	}

	kindTok := p.advance()

	if kindTok.Value == "co.lang.typeclass" {
		if !hasParameterClause {
			p.failf(kindTok, "a typeclass declares its parameters in the head, as in \"_ (F(_)) co.lang.typeclass\"")
		}
		return p.parseTypeclassDeclaration(declName, typeclassParams, annotations), true
	}
	if hasParameterClause {
		// DECISION-GEN-001: only a parameterized co.lang.type/co.lang.data
		// declaration, a signature type component, and a typeclass parameter
		// clause take declaration-head parameters. Everything else uses the
		// annotation.
		p.failf(clauseTok,
			"%q does not take declaration-head type parameters; declare a generic struct, class, function or method with @co.dap.generic",
			kindTok.Value)
	}

	p.rejectNonPrimaryKind(kindTok)
	return p.dispatchKindDeclaration(declName, nil, kindTok, annotations), true
}

// rejectNonPrimaryKind reports a built-in kind that is declarable in FoLang but
// not as a file-backed primary declaration.
//
// Each of these has a different home, and naming that home is the whole value of
// the check: a bare "not a primary declaration" would leave the author to guess
// which of the reserved source forms or containers the declaration belongs in
// (docs/language-ref.md, "Package Source Files").
func (p *parser) rejectNonPrimaryKind(kindTok scanlex.Token) {
	if home, misplaced := nonPrimaryKindHomes[kindTok.Value]; misplaced {
		p.failf(kindTok, "a %q declaration is not a file-backed primary declaration; it belongs %s", kindTok.Value, home)
	}
	if _, isTypeKind := typeDeclarationKinds[kindTok.Value]; isTypeKind {
		p.failf(kindTok, "a %q declaration is not a file-backed primary declaration; it belongs in an ordinary <Fragment>.unit.fol unit file", kindTok.Value)
	}
}

// nonPrimaryKindHomes names the source form or container that owns each kind
// revisions 23 and 27 removed from primary-declaration.
var nonPrimaryKindHomes = map[string]string{
	"co.lang.unit":    "in a <Fragment>.unit.fol or <Name>.comp.unit.fol source file",
	"co.lang.package": "in the reserved package.fol source form",
	"co.lang.data":    "in an ordinary <Fragment>.unit.fol unit file",
	"co.lang.library": "in a library surface file",
	// DECISION-DECL-002 and DECISION-DECL-003. All three keep an ordinary
	// identifier in their head, so the home named here is also where the "_"
	// spelling stops being the right one.
	"co.lang.function": "in an ordinary <Fragment>.unit.fol unit file, written \"<name> co.lang.function = …\"",
	"co.lang.delegate": "in an ordinary <Fragment>.unit.fol unit file, written \"<name> co.lang.delegate = …\"",
	"co.lang.block":    "inside a function or method body, written \"<name> co.lang.block = { … }\"",
}

// dispatchKindDeclaration routes a declaration to the production its built-in kind selects.
//
// A forward declaration is recognised first, because it shares its kind token with the
// block-bodied form and differs only by ending at ";" rather than at "=".
//
// generics is meaningful for exactly two of the routed productions.
// DECISION-GEN-001 removed the declaration-head parameter clause everywhere
// else, so a caller that has no clause to offer passes nil and the remaining
// productions take none.
func (p *parser) dispatchKindDeclaration(declName name, generics []symboltable.GenericTypeParam, kindTok scanlex.Token, annotations annotationSet) ast.Stmt {
	// forward-type-declaration: the kind is forward-declarable and no binding follows.
	if isForwardDeclarableKind(kindTok.Value) && p.atForwardDeclarationEnd() {
		return p.parseForwardTypeDeclaration(declName, kindTok, annotations)
	}

	switch kindTok.Value {
	case "co.lang.struct":
		return p.parseStructDeclaration(declName, annotations)
	case "co.lang.cstruct":
		return p.parseCStructDeclaration(declName, annotations)
	case "co.lang.enum":
		return p.parseEnumDeclaration(declName, annotations)
	case "co.lang.union":
		return p.parseUnionDeclaration(declName, annotations)
	case "co.lang.data":
		return p.parseDataDeclaration(declName, generics, annotations)
	case "co.lang.class":
		return p.parseClassDeclaration(declName, annotations)
	case "co.lang.interface":
		return p.parseInterfaceDeclaration(declName, annotations)
	case "co.lang.signature":
		return p.parseSignatureDeclaration(declName, annotations)
	case "co.lang.module":
		return p.parseModuleDeclaration(declName, annotations)
	case "co.lang.unit":
		return p.parseUnitDeclaration(declName, annotations)
	case "co.lang.object":
		return p.parseObjectDeclaration(declName, annotations)
	case "co.lang.instance":
		return p.parseInstanceDeclaration(declName, annotations)
	case "co.lang.matcher":
		return p.parseMatcherInstanceDeclaration(declName, annotations)
	case "co.lang.function":
		return p.parseFunctionObjectDeclaration(declName, annotations)
	case "co.lang.delegate":
		return p.parseDelegateDeclaration(declName, annotations)
	case "co.lang.block":
		// DECISION-DECL-003: a named block is a statement, and the statement
		// dispatcher claims the identifier-headed spelling before the
		// nested-declaration guard runs. What reaches here is the "_" head, from
		// the primary path or from that guard's recovery.
		p.failf(kindTok, "a named block is a statement inside a function or method body, written \"<name> co.lang.block = { … }\"")
	case "co.lang.library":
		return p.parseLibraryDeclaration(declName, annotations)
	case "co.lang.package":
		return p.parsePackageAliasDeclaration(declName, annotations)
	case "co.lang.typeclass":
		// typeclass-declaration is reachable only from primary-declaration,
		// which reads its parameter clause before the kind token.
		p.failf(kindTok, "a typeclass is a file-backed primary declaration written \"_ (T) co.lang.typeclass\" in its own <Name>.fol file")
	}

	// type-declaration covers the alias family.
	if _, isTypeKind := typeDeclarationKinds[kindTok.Value]; isTypeKind {
		return p.parseTypeDeclaration(declName, generics, kindTok, annotations)
	}

	// DECISION-KIND-001, as revised in register revision 30: a co.lang.* name the
	// built-in kind table lists but no production admits as a declaration is
	// rejected as an unsupported kind. It is not read as an identifier and not
	// absorbed by a variable declaration, which is the whole point of rejecting it
	// here rather than falling through.
	//
	// Rejection is now the ONLY behaviour. Until revision 27 a
	// general-kind-declaration catch-all gave about twenty-eight of these names a
	// body the reference never specified; DECISION-DECL-001 removed it, so this
	// site serves both those names and the reserved future ones —
	// co.lang.typeconstructor and co.lang.typefunction — that never had a
	// production. The register states the diagnostic continues to cite this ID.
	p.failf(kindTok, "%q is a built-in kind name with no declaration form and cannot be declared", kindTok.Value)
	return nil // unreachable: failf panics
}

// forwardDeclarableKinds is the forward-declarable-kind set of section 6.
//
// OQ-001 (docs/grammar/OPEN-QUESTIONS.md) is open over two of these entries.
// After revision 27 neither co.lang.function nor co.lang.data is a primary
// declaration, and forward-type-declaration spells filename-derived-name, so
// neither is reachable through the production that names them. Both are kept
// here, and both are treated ALIKE: rejectNonPrimaryKind turns the "_" spelling
// away before the forward reading is tried, while a unit member with an
// identifier head still reaches parseForwardTypeDeclaration. Whichever reading
// the question is settled on has to be applied to the pair, so do not remove one
// without the other.
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
//
// It stays broader than the grammar on purpose. An explicitly named head is
// admitted here so tryParsePrimaryDeclaration can report the precise
// "must be written _" rule of DECISION-FILE-001, rather than silently declining
// the declaration and leaving a statement parser to fail on it further along.
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

		// The withdrawn annotated-contract-declaration shape: annotations, a name
		// and "=" with no kind token. DECISION-DECL-001 removed it, but the shape
		// is still admitted here so tryParsePrimaryDeclaration can name the
		// typeclass form that replaced it instead of leaving a statement parser to
		// fail on a braced body further along.
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

// atTypeLevelFunctionDeclaration reports whether the cursor begins a type-level-function-declaration.
//
// A type constructor is a function whose return-type clause names a type-producing kind, so
// the probe looks for the `name ( … ) -> ( kind )` shape with a type kind inside the return
// clause.
func (p *parser) atTypeLevelFunctionDeclaration() bool {
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
		// too, so parseTypeLevelFunctionDeclaration can issue the precise rule that a type
		// constructor has one unnamed type result instead of accepting it as a loose
		// ordinary function.
		if p.atIdentifier() && p.startsTypeExpression(p.peek(1)) {
			p.advance()
		}
		// The return type must be one of the type-producing kinds for this to be a
		// type constructor rather than an ordinary function.
		_, typeProducing := typeLevelResultKinds[p.lexeme()]
		return typeProducing
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
