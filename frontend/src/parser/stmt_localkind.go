package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// Nested kind-declaration recovery recognizes a declaration inside a block whose
// kind is a built-in kind token rather than a type.
//
// DECISION-SYN-008 and the reference's "Local and/or Nested types and
// functions" section forbid physically nested independent named types and
// containers. Only an ordinary named local function and anonymous expressions
// are exceptions. The parser still recognizes this prefix so statement.go can
// issue one precise diagnostic and consume the declaration for recovery.
//
// DECISION-KIND-001 still governs the interaction with variable-declaration:
// the predicate claims only a name followed by a built-in KIND, so an ordinary
// declarator such as `z co.lang.int = 1;` remains unaffected.
//
// co.lang.block is the one kind this guard must NOT claim. DECISION-DECL-003
// makes a named block a statement, so parseStatement dispatches the
// identifier-headed spelling before reaching here; what still arrives is the "_"
// head, which dispatchKindDeclaration rejects with the naming rule.

// atLocalKindDeclaration reports whether the cursor begins a declaration whose kind is a
// built-in kind token.
//
// The shape is `name [ generic-clause ] built-in-kind`, so the probe skips the name and any
// parenthesised clause and requires a BUILT_IN_KIND token to follow.
func (p *parser) atLocalKindDeclaration() bool {
	if !p.atIdentifier() && !p.at(scanlex.DISCARD_WILD_VAR) {
		return false
	}
	return p.lookaheadOnly(func() bool {
		p.advance() // the name
		hasGenerics := false
		if p.at(scanlex.OPEN_PAREN) {
			hasGenerics = p.looksLikeGenericParameterClause()
			p.skipBalanced(scanlex.OPEN_PAREN, scanlex.CLOSE_PAREN)
		}
		if !p.at(scanlex.BUILT_IN_KIND) {
			return false
		}
		// `x co.lang.value` and the other overlapping names are variable
		// declarations in an executable block. An explicit generic clause cannot
		// belong to a variable declarator, so that shape remains a forbidden nested
		// kind declaration and receives the dedicated diagnostic.
		return hasGenerics || !isTypeFirstKind(p.lexeme())
	})
}

// parseLocalKindDeclaration consumes a forbidden kind-introduced declaration in
// statement position after the caller has diagnosed its physical nesting.
//
// It reads the same prefix the primary-declaration dispatcher does — name,
// optional generic clause, kind token — and hands off to the shared dispatcher
// only to retain useful recovery and follow-on diagnostics.
func (p *parser) parseLocalKindDeclaration(annotations annotationSet) ast.Stmt {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	declName := p.parseLocalDeclarationName()
	generics := p.parseOptionalGenericParameterClause()
	kindTok := p.expect(scanlex.BUILT_IN_KIND, "to declare a local declaration's kind")

	return p.dispatchKindDeclaration(declName, generics, kindTok, annotations)
}

// parseLocalDeclarationName reads the name of an already-diagnosed nested
// declaration. Both spellings are accepted here on purpose: this is a recovery
// path, and the declaration has been rejected for its position rather than for
// how it names itself, so re-reporting the head would bury the real diagnostic.
func (p *parser) parseLocalDeclarationName() name {
	if p.at(scanlex.DISCARD_WILD_VAR) {
		return nameFrom(p.advance())
	}
	return p.parseIdentifier("as a local declaration name")
}

// Physical nesting inside a DECLARATION body — docs/language-ref.md, "Struct
// Declaration Relationships", "Class Declaration Relationships" and "Invalid
// Physical Nesting".
//
// atLocalKindDeclaration above answers the same question for an executable
// block. The containers below need it too, and for the same reason: the
// reference gives physical nesting a rule of its own and names the replacement
// for it, so a member that breaks that rule should be told which rule it broke.
//
// Without this the nested declaration fell through to the container's ordinary
// member grammar and was reported by whatever failed first, which named
// something the author had not written:
//
//	_ co.lang.struct = { Address co.lang.struct = { … } }
//	    -> "a struct field cannot have a default value"
//	_ co.lang.class  = { Address co.lang.struct = { … } }
//	    -> "expected \";\" after a field declaration, found \"}\""
//
// It applies to every container whose member grammar admits NO kind-introduced
// declaration at all, which is all of them but three:
//
//	struct   cstruct  union     enum       class
//	trait    mixin    interface typeclass  object
//	matcher  instance extension                     -> guarded
//
//	unit     signature module                       -> exempt
//
// The three exemptions are the only member grammars that name a built-in kind:
// `unit-member` admits data-declaration, type-declaration,
// function-object-declaration and delegate-declaration; `signature-member` and
// `module-member` admit signature-type-component and the associated-type forms.
// Guarding those bodies would reject the reference's own examples.
//
// Everything else is guarded, `instance-body` included — it is
// `{ function-declaration | variable-declaration }`, and a variable declarator's
// type is a type-expression, which atLocalKindDeclaration already separates from
// a kind token through isTypeFirstKind. `x co.lang.int = 1;` is therefore
// untouched while `Inner co.lang.struct = { … }` is not.
//
// What the guard claims is a nested DEFINITION, never a forward declaration that
// shares its shape. `Dept co.lang.struct;` is the extern form the reference
// defines as a legal class and unit member, and it stays legal here; only a
// binding makes the declaration a physically nested one.

// rejectNestedKindDeclaration reports a named kind-introduced declaration written
// directly inside a declaration body that cannot hold one.
//
// container names the enclosing body for the diagnostic. The check is a pure
// lookahead: it consumes nothing, so a body that does not begin a nested
// declaration proceeds to its ordinary member grammar untouched.
//
// It runs AFTER the member's annotations have been parsed, which is what puts the
// cursor on the declaration name. Every declaration production begins
// `annotations, ...`, and a nested declaration is annotated more often than not —
// `@co.dap.local` is the very thing the diagnostic recommends, so a reader who
// half-remembers the rule writes it in the wrong place, and a guard testing at
// the raw cursor sees the "@" and declines.
//
// Skipping the annotations in lookahead instead would find the name, but at the
// cost of reporting this rule BEFORE the metadata is validated. An unregistered
// `@co.*` name and a malformed argument list are errors in their own right, and
// they are the FIRST thing wrong with the member; announcing the nesting rule
// over them would hide a problem the author has to fix either way. Parsing the
// annotations first means each diagnostic is raised by the rule that owns it, in
// source order.
func (p *parser) rejectNestedKindDeclaration(container string) {
	if !p.atNestedKindDefinition() {
		return
	}
	kind := p.nestedKindName()

	// A kind the built-in table lists but no production admits is not misplaced,
	// it is undeclarable, and saying where to move it would be advice for a
	// declaration the language does not have. DECISION-KIND-001's diagnostic owns
	// that case wherever the spelling appears, so it is repeated here verbatim
	// rather than shadowed by the nesting rule.
	if !hasDeclarationForm(kind) {
		p.failf(p.cur(), "%q is a built-in kind name with no declaration form and cannot be declared", kind)
	}

	p.failf(p.cur(), "a named %s declaration cannot be physically nested in %s; %s",
		kind, container, nestedKindHome(kind))
}

// fileBackedPrimaryKinds is the closed set of built-in kinds that head a
// primary-declaration — the `<Name>.fol` forms of section 1 of the grammar.
//
// These are the declarations whose home is a source file of their own, and the
// only ones for which `@co.dap.local` is the right advice.
var fileBackedPrimaryKinds = map[string]bool{
	"co.lang.struct":    true,
	"co.lang.cstruct":   true,
	"co.lang.enum":      true,
	"co.lang.union":     true,
	"co.lang.class":     true,
	"co.lang.trait":     true,
	"co.lang.mixin":     true,
	"co.lang.interface": true,
	"co.lang.signature": true,
	"co.lang.module":    true,
	"co.lang.typeclass": true,
	"co.lang.object":    true,
	"co.lang.instance":  true,
	"co.lang.matcher":   true,
	"co.lang.extension": true,
}

// hasDeclarationForm reports whether a built-in kind has a source declaration
// production at all.
//
// The three sets below are the complete inventory, and each is the same one the
// parser already dispatches on, so a kind cannot gain a production here without
// gaining one there:
//
//	fileBackedPrimaryKinds   the <Name>.fol primaries
//	typeDeclarationKinds     the non-UDT type family
//	nonPrimaryKindHomes      the forms that live somewhere other than a primary —
//	                         unit, data, refinementType, component, function,
//	                         delegate and block
//
// Everything else in the built-in kind table — co.lang.loader, co.lang.macro,
// co.lang.role and the rest — is a RESERVED name the reference lists without
// giving it a declaration form.
func hasDeclarationForm(kind string) bool {
	if fileBackedPrimaryKinds[kind] {
		return true
	}
	if _, isTypeDeclaration := typeDeclarationKinds[kind]; isTypeDeclaration {
		return true
	}
	_, hasHome := nonPrimaryKindHomes[kind]
	return hasHome
}

// nestedKindHome names where a nested declaration should have been written.
//
// The two families have different homes and the reference is specific about
// both, so one message cannot serve them. A file-backed primary keeps its own
// `<Name>.fol` and reaches its target through an association annotation; a
// non-UDT type declaration is instead the deliberate unit exception, and
// `@co.dap.local` is not what it needs.
//
// That exception has TWO containers, not one. "Physical Nesting Rules" permits
// these declarations "directly inside an ordinary unit, and inside a companion
// unit where their own rules permit association with the owner", and "Struct
// Companion Units" lists "non-UDT type declarations associated with the owner"
// among what a companion may declare. Naming only the ordinary unit would send
// an author writing a type for one struct to the wrong file.
func nestedKindHome(kind string) string {
	if fileBackedPrimaryKinds[kind] {
		return "declare it in its own package source file and restrict it to this declaration with @co.dap.local"
	}
	if _, isTypeDeclaration := typeDeclarationKinds[kind]; isTypeDeclaration {
		return "a non-UDT type declaration belongs in an ordinary <Fragment>.unit.fol unit file, or in a <StructName>.comp.unit.fol companion unit where its own rules permit association with the owner"
	}
	// The remaining forms are not primaries at all, and each has a home of its
	// own that nonPrimaryKindHomes already words for the misplaced-primary
	// diagnostic. Reusing it keeps one answer per kind: a function object and a
	// delegate belong in a unit, a named block inside a function or method body,
	// a component in its structural surface. None of them is helped by being told
	// to take a source file of its own.
	if home, known := nonPrimaryKindHomes[kind]; known {
		return "it belongs " + home
	}
	return "declare it in its own package source file and restrict it to this declaration with @co.dap.local"
}

// atNestedKindDefinition reports whether the cursor begins a nested kind
// DEFINITION, as opposed to a forward declaration of the same shape.
//
// The two are told apart by the binding, which is the only thing that separates
// them:
//
//	Dept co.lang.struct;          forward/extern declaration — a legal member
//	Dept co.lang.struct = { … }   a definition — physically nested, forbidden
//
// The reference gives the first its own section and states that
// "@co.dap.declare is optional" for functions and types, so the annotation cannot
// be the discriminator and the ANNOTATED spelling is not the only one to admit.
// The absence of a body is what makes a forward declaration one, and the
// nesting rule is about definitions: a forward declaration introduces no nested
// body and no nested scope, which is exactly what "physical nesting" means.
func (p *parser) atNestedKindDefinition() bool {
	if !p.atIdentifier() && !p.at(scanlex.DISCARD_WILD_VAR) {
		return false
	}
	return p.lookaheadOnly(func() bool {
		p.advance() // the name
		hasGenerics := false
		if p.at(scanlex.OPEN_PAREN) {
			hasGenerics = p.looksLikeGenericParameterClause()
			p.skipBalanced(scanlex.OPEN_PAREN, scanlex.CLOSE_PAREN)
		}
		if !p.at(scanlex.BUILT_IN_KIND) {
			return false
		}
		if requiresGenericClauseToNest(p.lexeme()) && !hasGenerics {
			return false
		}
		if !hasGenerics && !isNestableDeclarationKind(p.lexeme()) {
			return false
		}
		p.advance() // the kind token
		// kind-options may precede the binding: `co.lang.module->( … ) = { … }`.
		if p.at(scanlex.ARROW) {
			p.advance()
			if p.at(scanlex.OPEN_PAREN) {
				p.skipBalanced(scanlex.OPEN_PAREN, scanlex.CLOSE_PAREN)
			}
		}
		return p.atOp("=")
	})
}

// isNestableDeclarationKind reports whether a BUILT_IN_KIND token in a
// `name KIND = …` member introduces a DECLARATION rather than naming a field's
// type.
//
// This is where the container probe and the executable-block probe part company,
// and they have to: `name KIND = value` is ambiguous, but not the same ambiguity
// in both places.
//
// isTypeFirstKind, which atLocalKindDeclaration uses, folds two sets together —
// the built-in data types and the dedicated type-declaration kinds — and reads
// both as "a type, so this is a variable declarator". That is right for a block,
// where `T co.lang.type = a;` really is a type-level binding of the kind the
// lifecycle example writes.
//
// In a container member it is wrong for half the set. A field can be typed
// `co.lang.int`, so a built-in data type still means a field. Nothing can be
// typed `co.lang.type` or `co.lang.newtype`: those are declaration kinds, so
// `Alias co.lang.type = co.lang.int;` in a class body is a nested non-UDT type
// DEFINITION, which the reference names as the deliberate UNIT exception and
// prohibits "physically inside classes, structs, modules, functions, or
// executable blocks" (docs/language-ref.md, "Physical Nesting Rules"). Folding
// them together let a class silently accept one as a field carrying a default.
//
// The containers the reference does grant this context — unit, signature and
// module, through unit-member and signature-type-component — never reach this
// probe; they are exempt from the guard entirely.
func isNestableDeclarationKind(kind string) bool {
	return !isBuiltinTypeName(kind)
}

// isBuiltinTypeName reports whether a kind spelling is also listed as a usable
// built-in TYPE, in which case a member written `name KIND` may be a field.
//
// Several spellings are in both tables. `co.lang.data` is the clearest: it is a
// usable carrier type AND the head of data-declaration, and `co.lang.typeclass`
// and `co.lang.dependentType` overlap the same way. For those the kind token
// alone settles nothing, and only a declaration-head generic clause does — see
// requiresGenericClauseToNest.
func isBuiltinTypeName(kind string) bool {
	for _, builtin := range scanlex.Builtin_types {
		if builtin == kind {
			return true
		}
	}
	return false
}

// requiresGenericClauseToNest reports whether a kind needs a declaration-head
// generic clause before a container member can be read as a nested declaration.
//
// The kinds that need one are exactly the spellings that are ALSO usable types,
// because for those the member grammar of the enclosing body already has a
// production that matches:
//
//	class-member = field-declaration | function-declaration | lifecycle-…
//	field-declaration = annotations, identifier, type-expression,
//	                    [ "=", expression ], statement-end
//
// `payload co.lang.data = someValue;` in a class body is therefore an initialized
// FIELD, and the only production that matches it. A unit body reads the same
// tokens as a data-declaration, but that is not this body making a different
// choice about one ambiguity — the two member grammars are disjoint. `unit-member`
// has data-declaration and no field-declaration; `class-member` has
// field-declaration and no data-declaration. Each body has exactly one matching
// production and neither is resolving a conflict.
//
// A declaration-head generic clause is what cannot be a field: no field declarator
// takes one. `Shape(T) co.lang.data = Some(T) | None();` is unmistakable and stays
// caught, which is the same discriminator atLocalKindDeclaration applies in a
// block.
//
// Kinds that are NOT usable types need no such evidence. Nothing can be typed
// `co.lang.type` or `co.lang.struct`, so those spellings are unambiguous on their
// own.
func requiresGenericClauseToNest(kind string) bool {
	return isBuiltinTypeName(kind)
}

// nestedKindName returns the built-in kind spelling the nested declaration uses,
// so the diagnostic can name what was written rather than describing it.
//
// The caller has already consumed the member's annotations, so the cursor is on
// the declaration name and this reads the kind directly.
func (p *parser) nestedKindName() string {
	kind := ""
	p.lookaheadOnly(func() bool {
		p.advance() // the name
		if p.at(scanlex.OPEN_PAREN) {
			p.skipBalanced(scanlex.OPEN_PAREN, scanlex.CLOSE_PAREN)
		}
		if p.at(scanlex.BUILT_IN_KIND) {
			kind = p.lexeme()
		}
		return false
	})
	if kind == "" {
		return "kind"
	}
	return kind
}

// skipAnnotationApplications advances the cursor past a run of metadata
// applications, `"@" qualified-name [ "(" … ")" ]` each.
//
// It is lookahead machinery rather than a parse: no node is built and no
// diagnostic is raised, so it must only ever be called inside lookaheadOnly or
// speculate.
func (p *parser) skipAnnotationApplications() {
	for p.atAnnotation() {
		p.advance()
		if p.at(scanlex.OPEN_PAREN) {
			p.skipBalanced(scanlex.OPEN_PAREN, scanlex.CLOSE_PAREN)
		}
	}
}
