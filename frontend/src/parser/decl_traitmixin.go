package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
)

// trait-declaration and mixin-declaration — section 7.
//
//	trait-declaration = annotations, filename-derived-name,
//	                    "co.lang.trait", "=", trait-body
//	trait-body        = "{", { trait-member }, body-close
//	trait-member      = function-declaration, trait-member-guard
//
//	mixin-declaration = annotations, filename-derived-name,
//	                    "co.lang.mixin", "=", mixin-body
//	mixin-body        = "{", { mixin-member }, body-close
//	mixin-member      = field-declaration | function-declaration
//
// The two are composition forms rather than instantiable types, and the reference
// separates them by exactly one property — state:
//
//	trait   interface-like, may carry DEFAULT implementations, carries NO
//	        instance state, and admits no virtual method
//	        (docs/language-ref.md, "Traits")
//	mixin   the abstract-class-like form, which MAY carry state, abstract
//	        methods, implemented methods and virtual methods
//	        (docs/language-ref.md, "Mixins")
//
// That single difference is why they are two productions rather than one with a
// flag: `mixin-member` admits field-declaration and `trait-member` does not.
//
// Neither is instantiable and neither owns lifecycle machinery, so unlike
// class-declaration there is no lifecycle capability to push and no `self`
// receiver context: `self` is defined for the methods of a co.lang.class and for
// a target-bound extension's `@co.dap.class` methods, and a trait or mixin
// method is neither until a class composes it.
//
// Both follow interface-declaration's representation choice rather than
// class-declaration's. ast.TypeDeclarationStmt stores its symbol as an
// ITypeSymbol, which among the symbol kinds only TypeSymbol satisfies, so the
// declaration kind is recorded on the type symbol instead of through a dedicated
// TraitSymbol or MixinSymbol.

// parseTraitDeclaration parses the trait-declaration production.
//
// Implements: trait-declaration
// Implements: trait-body
func (p *parser) parseTraitDeclaration(declName name, annotations annotationSet) ast.Stmt {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	p.expectOp("=", "before a trait body")

	// A trait declares a TypeSymbol, but its BODY is an interface-shaped scope: the
	// members are functions that may call one another in any order, which is the
	// resolution policy S_InterfaceSymbol selects and S_TypeSymbol does not.
	members := p.parseBracedBody(symboltable.S_InterfaceSymbol, "a trait body", p.parseTraitMember)

	symb := p.typeSymbol(declName.Scanned)
	symb.TypeType = "co.lang.trait"
	applyTypeVisibility(&symb.SymbolDetails, annotations)

	return ast.TypeDeclarationStmt{Span: p.spanFrom(spanStart), Name: declName.Scanned,
		Body:     members,
		Kind:     "co.lang.trait",
		SubType_: "TRAIT",
		Typetype: "UDT",
		SDapst:   annotations.list(),
		KDapst:   annotations.list(),
		Symb:     symb,
	}
}

// parseTraitMember parses the trait-member production and applies
// trait-member-guard.
//
// The guard has two halves the parser can decide from the declaration shape
// alone. A member that is not a function is state, which a trait cannot carry;
// and `@co.dap.virtual` is refused outright. Both are reported where they are
// written, naming the trait rule rather than letting a field die in the function
// grammar as "expected \"(\" to open a parameter list".
//
// An abstract or bodyless function needs no special case: function-binding
// already admits a bare statement-end, so `someFunction()->();` is the same
// production as a defaulted one.
//
// Implements: trait-member
// Implements: trait-member-guard
func (p *parser) parseTraitMember() ast.Stmt {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	annotations := p.parseAnnotations()
	p.rejectNestedKindDeclaration("a trait body")
	p.rejectOperatorPlacement(annotations, "a trait")

	if !p.atMemberFunctionDeclaration() {
		// logicalName strips the backend lowering suffix the scanner adds, so the
		// diagnostic quotes the name the author wrote.
		p.failf(p.cur(), "a trait carries no instance state, so %q cannot be declared as a field; a trait declares functions, which may be abstract or carry a default implementation", logicalName(p.lexeme()))
	}
	if annotations.has("@co.dap.virtual") {
		p.failf(p.cur(), "a virtual method is not permitted in a trait; declare it in a co.lang.mixin, which is the composition form that admits virtual methods")
	}

	return p.parseDecoratedFunctionDeclaration(annotations)
}

// parseMixinDeclaration parses the mixin-declaration production.
//
// Implements: mixin-declaration
// Implements: mixin-body
func (p *parser) parseMixinDeclaration(declName name, annotations annotationSet) ast.Stmt {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	p.expectOp("=", "before a mixin body")

	// As for a trait, the scope kind names the body's shape rather than the symbol
	// the declaration mints. A mixin body carries state as well as functions, so it
	// is a class-shaped complete container.
	members := p.parseBracedBody(symboltable.S_ClassSymbol, "a mixin body", p.parseMixinMember)

	symb := p.typeSymbol(declName.Scanned)
	symb.TypeType = "co.lang.mixin"
	applyTypeVisibility(&symb.SymbolDetails, annotations)

	return ast.TypeDeclarationStmt{Span: p.spanFrom(spanStart), Name: declName.Scanned,
		Body:     members,
		Kind:     "co.lang.mixin",
		SubType_: "MIXIN",
		Typetype: "UDT",
		SDapst:   annotations.list(),
		KDapst:   annotations.list(),
		Symb:     symb,
	}
}

// parseMixinMember parses the mixin-member production.
//
// A mixin admits both alternatives, so the two are separated the way a class
// body separates them: a name followed by "(" begins a function, anything else
// is a field. There is no lifecycle alternative — `@@new` and `@@init` are
// class-only, and the lifecycle name is refused by the function-name rule.
//
// Implements: mixin-member
func (p *parser) parseMixinMember() ast.Stmt {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	annotations := p.parseAnnotations()
	p.rejectNestedKindDeclaration("a mixin body")

	if p.atMemberFunctionDeclaration() {
		p.rejectOperatorPlacement(annotations, "a mixin")
		return p.parseDecoratedFunctionDeclaration(annotations)
	}
	p.rejectOperatorPlacement(annotations, "a mixin field")
	return p.parseFieldDeclaration(annotations)
}
