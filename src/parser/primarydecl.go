package parser

import (
	"github.com/samkrao/fo-lang/src/ast"
	symboltable "github.com/samkrao/fo-lang/src/context"
	"github.com/samkrao/fo-lang/src/scanlex"
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
//	                    | extension-declaration
//
// This is the ordinary `<Name>.fol` root. A package source file holds exactly one
// primary declaration and its name comes from the filename, so every alternative
// here spells filename-derived-name — "_" — in the declaration head
// (docs/language-ref.md, "File-Backed Primary Declarations").
//
// The set is CLOSED and every alternative is selected by a built-in kind token. It is
// exactly the list the reference enumerates under "Package Source Files" as the top-level
// declaration kinds a `<Name>.fol` file may carry. Everything else that once appeared here
// belongs to a different source form or a different container:
//
//	unit-declaration            <Fragment>.unit.fol or <Name>.comp.unit.fol
//	package-alias-declaration   the reserved package.fol source form
//	library-declaration         a library.fol surface file
//	data-declaration            a member of an ordinary unit
//	type-declaration            a member of an ordinary unit, and
//	function declarations      are members of an ordinary unit
//	function-object-declaration a unit member
//	delegate-declaration        a unit member
//	named-block-declaration     a statement
//	annotated-function-primary  a function is a unit member; the reference states
//	                            that FoLang has no free-flowing package functions
//	forward-type-declaration    an extern member inside a class or unit body,
//	                            written @co.dap.declare(extern)
//
// A general-kind-declaration catch-all once admitted about twenty-eight further co.lang.*
// names that the reference lists only in its built-in kind table and gives no declaration
// form. Those stay reserved and are rejected by name.
//
// # How the dispatch works
//
// Every alternative has the same prefix — annotations then "_" — and is then identified
// by a BUILT_IN_KIND token. The scanner has already folded `co.lang.struct` and friends
// into single tokens, so once the prefix is consumed the dispatch is a switch on one
// lexeme. That is why the parse functions in the decl_* files take declName and
// annotations as parameters: the dispatcher had to read them to get far enough to choose.

// parsePrimaryDeclaration parses one primary-declaration, reporting a diagnostic if the
// cursor does not begin one.
//
// Implements: primary-declaration
func (p *parser) parsePrimaryDeclaration() ast.Stmt {
	if traceEnabled || DEBUG_TRACE {
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
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}
	if !p.atPrimaryDeclaration() {
		return nil, false
	}

	annotations := p.parseAnnotations()
	p.rejectEffectsPlacement(annotations, "a file-backed primary declaration")

	// A function-shaped declaration is no longer a primary. Annotating it does not
	// promote it either: FoLang has no free-flowing package functions, so a function
	// belongs in a unit file whatever decorates it (docs/language-ref.md, "Functions").
	// The shape is still recognised here so the diagnostic can name that home rather
	// than leaving a "_" rule to fire on a declaration whose problem is its container.
	if p.atFunctionDeclarationShape() {
		p.fail(p.cur(), "a function declaration is not a file-backed primary declaration; package functions belong in an ordinary <Fragment>.unit.fol unit file")
	}

	declName := p.parseFilenameDerivedName("a primary declaration")

	// "_" and the typeclass parameter clause are separate grammar components, and
	// typeclass-declaration is the only primary that has such a clause. Reading it
	// here — before the kind is known — is what lets the no-head-parameters rule be
	// reported precisely for every other kind.
	clauseTok := p.cur()
	var typeclassParams []symboltable.GenericTypeParam
	hasParameterClause := false
	if p.at(scanlex.OPEN_PAREN) && p.looksLikeGenericParameterClause() {
		typeclassParams = p.parseTypeclassParameterClause()
		hasParameterClause = true
	}

	// Every alternative is selected by a kind token, so a binding here declares
	// nothing. The annotated spelling was annotated-contract-declaration until
	// co.lang.typeclass superseded it, and it is worth naming its replacement rather
	// than reporting a bare missing kind.
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
		// Only a parameterized co.lang.type/co.lang.data declaration, a signature
		// type component, and a typeclass parameter clause take declaration-head
		// parameters. Everything else uses the annotation.
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
	"co.lang.unit": "in a <Fragment>.unit.fol or <Name>.comp.unit.fol source file",
	"co.lang.data": "in an ordinary <Fragment>.unit.fol unit file",
	// A refinement type is a type declaration, so it shares the type family's
	// home even though its own production is separate.
	"co.lang.refinementType": "in an ordinary <Fragment>.unit.fol unit file or an application entry file",
	"co.lang.predicateType":  "in an ordinary <Fragment>.unit.fol unit file or an application entry file",
	"co.lang.component":      "in src/component.fol or components/<kind>/component.fol",
	// DECISION-DECL-002 and DECISION-DECL-003. All three keep an ordinary
	// identifier in their head, so the home named here is also where the "_"
	// spelling stops being the right one.
	"co.lang.function": "in an ordinary <Fragment>.unit.fol unit file, written \"<name> co.lang.function = …\"",
	"co.lang.delegate": "in an ordinary <Fragment>.unit.fol unit file, written \"<name> co.lang.delegate = …\"",
	"co.lang.block":    "inside a function or method body, written \"<name> co.lang.block = { … }\"",
}

// dispatchKindDeclaration routes a declaration to the production its built-in kind selects.
//
// generics is meaningful for exactly two of the routed productions. The
// declaration-head parameter clause was removed everywhere else, so a caller that has
// no clause to offer passes nil and the remaining productions take none.
func (p *parser) dispatchKindDeclaration(declName name, generics []symboltable.GenericTypeParam, kindTok scanlex.Token, annotations annotationSet) ast.Stmt {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
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
	case "co.lang.trait":
		return p.parseTraitDeclaration(declName, annotations)
	case "co.lang.mixin":
		return p.parseMixinDeclaration(declName, annotations)
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
	case "co.lang.extension":
		return p.parseExtensionDeclaration(declName, annotations)
	case "co.lang.refinementType":
		// refinement-type-declaration is its own production rather than a member
		// of the alias family: its binding is `(T).where(pred)`, which no
		// type-expression can be.
		return p.parseRefinementTypeDeclaration(declName, kindTok, annotations)
	case "co.lang.predicateType":
		return p.parsePredicateTypeDeclaration(declName, kindTok, annotations)
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

// atPrimaryDeclaration reports whether the cursor begins a primary-declaration.
//
// The test is deliberately structural: a declaration starts with annotations or a name, and
// what makes it a declaration rather than a statement is that a built-in kind token, a
// contract binding, or a function shape follows.
//
// It stays broader than the grammar on purpose. An explicitly named head is
// admitted here so tryParsePrimaryDeclaration can report the precise
// "must be written _" rule of a file-backed primary, rather than silently declining
// the declaration and leaving a statement parser to fail on it further along.
func (p *parser) atPrimaryDeclaration() bool {
	if !p.atAnnotation() && !p.atIdentifier() && !p.at(scanlex.DISCARD_WILD_VAR) {
		return false
	}

	return p.lookaheadOnly(func() bool {
		annotated := p.atAnnotation()
		p.skipAnnotationApplications()

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
		// and "=" with no kind token. It is no longer a declaration, but the shape
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

// atFunctionDeclarationShape reports whether the cursor begins a function declaration.
//
// primary-declaration has no function alternative, so this is used to reject one with a
// diagnostic that names the unit file it belongs in.
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
