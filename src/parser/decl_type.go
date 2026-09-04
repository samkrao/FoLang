package parser

import (
	"github.com/samkrao/fo-lang/src/ast"
	symboltable "github.com/samkrao/fo-lang/src/context"
	"github.com/samkrao/fo-lang/src/scanlex"
)

// type-declaration and its relatives — section 6.
//
//	type-declaration               = polymorphic-type-declaration
//	                               | simple-type-declaration
//	simple-type-declaration        = annotations, identifier,
//	                                 type-declaration-kind, [ kind-options ],
//	                                 [ "=", type-expression ], statement-end
//	type-declaration-kind = "co.lang.type" | "co.lang.newtype"
//	                      | "co.lang.opaquetype" | "co.lang.subtype"
//	                      | "co.lang.supertype" | "co.lang.dependentType"
//	                      | "co.lang.kind"
//
// The kinds differ in how the new name relates to the type it is built from
// (docs/language-ref.md, "Type Declarations"):
//
//	x co.lang.type = co.lang.int;         alias: interchangeable with int
//	x co.lang.newtype = co.lang.int;      distinct type with int's representation
//	x co.lang.opaquetype = co.lang.int;   alias inside, distinct outside
//	y co.lang.type = co.lang.int | co.lang.char;   tagged union
//	test co.lang.subtype = co.lang.int;   covariant
//	test co.lang.supertype = co.lang.int; contravariant
//	blockormacro co.lang.kind = block | macro;      a kind-level union
//
// The set is CLOSED to the kinds the reference gives a source form. `co.lang.typealias`,
// `co.lang.associatedtype`, `co.lang.refinementType`, `co.lang.typetype` and
// `co.lang.typekind` appear only as rows of the Builtin Kinds table with no declaration
// syntax anywhere in the reference, so they stay reserved: a table-listed co.* name with
// no implemented source form must not be treated as ordinary user syntax
// (docs/grammar/folang.ebnf, preamble). `co.lang.kind` earns its place the other way
// round — the macro section declares one.
//
// The binding is optional, because a type may be declared and defined later.
//
// A type alias never introduces declaration-head parameters. Generic declarations use
// @co.dap.generic and value-indexed type families are functions returning
// co.lang.dependentType.

// refinement-type-declaration — section 6.
//
//	refinement-type-declaration = annotations, identifier,
//	                              "co.lang.refinementType", "=",
//	                              refinement-type-expression, statement-end
//	refinement-type-expression  = "(", type-expression, ")", ".where",
//	                              "(", expression, ")"
//
// A refinement type is a base type narrowed by a predicate
// (docs/language-ref.md, "Refinement Types"):
//
//	positiveInt    co.lang.refinementType = (co.lang.int).where(_ > 0);
//	percentage     co.lang.refinementType = (co.lang.int).where(_ >= 0 && _ <= 100);
//	nonEmptyString co.lang.refinementType = (co.lang.string).where(_.length > 0);
//
// It has its OWN production rather than joining the alias family because its
// binding is not a type-expression: `(T).where(pred)` is a fixed shape whose
// parenthesised base and `.where` argument are two different grammatical things.
//
// Inside the predicate — and nowhere else — `_` denotes the candidate value of
// the base type, which is what refinement-candidate and its guard record. Every
// occurrence refers to the same candidate; it introduces no variable into the
// enclosing scope, cannot be rebound or assigned, and does not escape the
// declaration.

// parseRefinementTypeDeclaration parses the refinement-type-declaration
// production.
//
// Implements: refinement-type-declaration
func (p *parser) parseRefinementTypeDeclaration(declName name, kindTok scanlex.Token, annotations annotationSet) ast.Stmt {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	p.expectOp("=", "before a refinement type's base type")
	base, predicate := p.parseRefinementTypeExpression()
	p.statementEnd("a refinement type declaration")

	symb := p.typeSymbol(declName.Scanned)
	symb.ExplicitType = true
	symb.RefinementType = true

	return ast.RefinementTypeDeclarationStmt{NodeName: "RefinementTypeDeclarationStmt", Span: p.spanFrom(spanStart), Name: declName.Scanned,
		BaseType:  base.fullType(),
		BaseName:  base.actType(),
		Predicate: predicate,
		SDapst:    annotations.list(),
		KDapst:    annotations.list(),
		Symb:      symb,
	}
}

// parseRefinementTypeExpression parses the refinement-type-expression production
// and returns the base type and its predicate.
//
// Implements: refinement-type-expression
func (p *parser) parseRefinementTypeExpression() (typeRef, ast.Expr) {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	p.expect(scanlex.OPEN_PAREN, "to open a refinement type's base type")
	base := p.parseTypeExpression()
	p.expect(scanlex.CLOSE_PAREN, "to close a refinement type's base type")

	if !p.at(scanlex.DOT) || !p.atMemberNameAt(1, "where") {
		p.failf(p.cur(), "a refinement type narrows its base type with \".where( … )\", as in \"(co.lang.int).where(_ > 0)\"")
	}
	p.advance() // "."
	p.advance() // "where"
	p.expect(scanlex.OPEN_PAREN, "to open a refinement predicate")

	// The candidate context covers the predicate and nothing else, which is what
	// keeps `_` a wildcard everywhere outside it.
	popCandidate := p.pushRefinementPredicateContext()
	predicate := p.parseExpression()
	popCandidate()

	p.expect(scanlex.CLOSE_PAREN, "to close a refinement predicate")
	return base, predicate
}

// pushRefinementPredicateContext opens the region in which `_` denotes the
// refinement candidate, and returns the function that closes it.
func (p *parser) pushRefinementPredicateContext() func() {
	p.refinementPredicateDepth++
	return func() { p.refinementPredicateDepth-- }
}

// refinementCandidateGuard reports whether a `_` at this occurrence denotes the
// candidate value of the co.lang.refinementType declaration being parsed.
//
// `_` stays contextual everywhere else: it is the wildcard in pattern and discard
// positions and the declaration-name placeholder in a filename-derived primary.
// Only a refinement predicate gives it the candidate meaning.
//
// Implements: refinement-candidate-guard
func (p *parser) refinementCandidateGuard() bool {
	return p.refinementPredicateDepth > 0
}

// Implements: predicate-type-declaration
// Implements: predicate-type-expression
// Implements: predicate-type-binder
//
// parsePredicateTypeDeclaration parses a type-valued predicate declaration:
//
//	Name co.lang.predicateType =
//	    co.lang.type.where(candidate => predicate);
//
// The named candidate is a dedicated immutable co.lang.typevalue binding. Its
// child context spans only the predicate body; this is not the general lambda
// syntax and does not create a callable value.
func (p *parser) parsePredicateTypeDeclaration(declName name, kindTok scanlex.Token, annotations annotationSet) ast.Stmt {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	p.expectOp("=", "before a predicate type expression")
	if !p.at(scanlex.BUILT_IN_KIND) || p.lexeme() != "co.lang.type" {
		p.failf(p.cur(), "a predicate type must use co.lang.type.where(name => expression)")
	}
	p.advance()
	if !p.at(scanlex.DOT) || !p.atMemberNameAt(1, "where") {
		p.failf(p.cur(), "a predicate type must use co.lang.type.where(name => expression)")
	}
	p.advance() // "."
	p.advance() // "where"
	p.expect(scanlex.OPEN_PAREN, "to open a predicate type expression")

	var binder name
	var predicate ast.Expr
	var contextID string
	symb := p.typeSymbol(declName.Scanned)
	p.scoped(symboltable.S_PredicateType, func() {
		contextID = p.ctx.Id
		binder = p.parseIdentifier("as the predicate type's type-value binder")
		binderSymbol := p.varSymbol(binder.Scanned, "co.lang.typevalue")
		binderSymbol.Mutable = false
		binderSymbol.ExplicitType = true
		binderSymbol.LocalBinding = true
		p.declareNamed(binder, binderSymbol)
		p.expect(scanlex.EQGT, "after a predicate type binder")
		predicate = p.parseExpression()
	}, symb)

	p.expect(scanlex.CLOSE_PAREN, "to close a predicate type expression")
	p.statementEnd("a predicate type declaration")

	symb.ExplicitType = true
	symb.PredicateType = true
	return ast.PredicateTypeDeclarationStmt{NodeName: "PredicateTypeDeclarationStmt",
		Span:            p.spanFrom(spanStart),
		Name:            declName.Scanned,
		Binder:          binder.Scanned,
		Expression:      predicate,
		BinderContextId: contextID,
		SDapst:          annotations.list(),
		KDapst:          annotations.list(),
		Symb:            symb,
	}
}

// typeDeclarationKinds maps each type-declaration-kind to the symbol flag it sets.
var typeDeclarationKinds = map[string]string{
	"co.lang.type":          "alias",
	"co.lang.newtype":       "newtype",
	"co.lang.opaquetype":    "opaque",
	"co.lang.subtype":       "subtype",
	"co.lang.supertype":     "supertype",
	"co.lang.dependentType": "dependent",
	"co.lang.kind":          "kind",
}

// parseTypeDeclaration parses the type-declaration production.
//
// kindTok is the built-in kind token the dispatcher already matched, which is what
// selects the relationship between the declared name and its definition — and,
// together with the presence of a parameter clause, which of the two
// alternatives this is.
//
// Implements: type-declaration
// Implements: polymorphic-type-declaration
// Implements: simple-type-declaration
// Implements: nonpolymorphic-type-declaration-kind
// Implements: type-declaration-value
func (p *parser) parseTypeDeclaration(declName name, generics []symboltable.GenericTypeParam, kindTok scanlex.Token, annotations annotationSet) ast.Stmt {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	if len(generics) != 0 {
		p.failf(kindTok, "%q declarations do not take declaration-head parameters; use @co.dap.generic for generic declarations or a function returning co.lang.dependentType for value-indexed types", kindTok.Value)
	}

	// A kind may carry options, as in co.lang.dependentType->(kind=length).
	options := p.parseOptionalKindOptions()

	var definition typeRef
	hasDefinition := false
	if p.acceptOp("=") {
		// A `co.lang.variants( … )` right-hand side is a closed variant DEFINITION
		// rather than a type expression, so it is recognised before the ordinary
		// type reading claims its shape.
		if p.atVariantDefinition() {
			return p.parseVariantTypeDeclaration(declName, generics, kindTok, annotations, spanStart)
		}
		definition = p.parseTypeExpression()
		hasDefinition = true
	}

	p.statementEnd("a type declaration")

	symb := p.typeSymbol(declName.Scanned)
	applyTypeDeclarationKind(symb, kindTok.Value)
	symb.ExplicitType = hasDefinition
	symb.IsGenericType = len(generics) > 0
	symb.UnionType = hasDefinition && definition.Form == formUnion
	symb.FunType = hasDefinition && definition.Form == formFunction
	symb.ForallType = hasDefinition && definition.Form == formForall

	if kindTok.Value == "co.lang.dependentType" {
		decl := ast.DependentTypeDeclarationStmt{NodeName: "DependentTypeDeclarationStmt", Span: p.spanFrom(spanStart), Name: declName.Scanned,
			SDapst: annotations.list(), KDapst: annotations.list(), Symb: symb}
		if hasDefinition {
			decl.Type = definition.fullType()
		}
		if kind, ok := options["kind"]; ok {
			decl.DependentKind, _ = kind.(string)
		}
		return decl
	}

	decl := ast.TypeDeclarationStmt{NodeName: "TypeDeclarationStmt", Span: p.spanFrom(spanStart), Name: declName.Scanned,
		TypeParams: generics,
		Kind:       kindTok.Value,
		SubType_:   typeDeclarationKinds[kindTok.Value],
		Typetype:   typeTypeOf(definition, hasDefinition),
		SDapst:     annotations.list(),
		KDapst:     annotations.list(),
		Symb:       symb,
	}
	if hasDefinition {
		decl.Type_ = definition.fullType()
		decl.NewTypeName = definition.actType()
		if definition.Form == formUnion {
			decl.ADT_ = definition.actType()
		}
	}
	return decl
}

// The variant-definition right-hand side of a co.lang.type declaration.
//
//	Option(T) co.lang.type = co.lang.variants(Some(T), None());
//
// This is the reference's closed variant-based type definition
// (docs/language-ref.md, "Generic Declarations and Type Constructors"). It looks
// like an ordinary type application and is deliberately NOT one:
//
//	"Each item inside co.lang.variants(...) is a declaration, not a lookup of an
//	 already-existing symbol."
//
// So `Some` and `None` are introduced here as variant-constructor symbols owned
// by the enclosing type declaration, while the payload entries inside each
// variant's parentheses ARE type expressions and resolve normally — including
// against the enclosing declaration's type parameters, which is what makes the
// `T` of `Some(T)` the declaration's parameter rather than an unknown type.
//
// Reading this shape as an ordinary type-postfix-expression would parse without
// complaint and then resolve `Some` and `None` as existing types, which is
// exactly the lookup the reference forbids. That is why it is recognised before
// the type reading rather than after it.
//
// The declaration produces the same ast.TypeConstructorStmt a co.lang.data
// declaration does: both declare one type constructor and its closed set of
// variant constructors, and a consumer should not have to tell the two spellings
// apart.
//
// `co.lang.variants(...)` is valid only in this position — the variant-definition
// right-hand side of a co.lang.type declaration — and nowhere else.

// variantDefinitionName is the reserved spelling that opens a variant definition.
const variantDefinitionName = "co.lang.variants"

// atVariantDefinition reports whether the cursor begins a variant-definition
// right-hand side.
//
// The spelling arrives SPLIT. `co.lang` is a registered built-in namespace and
// `variants` is not one of its members, so the scanner's dotted fold stops at the
// namespace and emits `co.lang`, a DOT and the member — the same shape any
// `co.<namespace>.<member>(` call has. Matching that shape is what recognises the
// form; matching the whole spelling as one lexeme never fires.
func (p *parser) atVariantDefinition() bool {
	if !p.at(scanlex.BUIL_IN_STMT_EXPRS) || p.lexeme() != "co.lang" {
		return false
	}
	return p.peek(1).Kind == scanlex.DOT &&
		logicalName(p.peek(2).Value) == "variants" &&
		p.peek(3).Kind == scanlex.OPEN_PAREN
}

// consumeVariantDefinitionHead consumes the split `co.lang` DOT `variants` head.
func (p *parser) consumeVariantDefinitionHead() {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}
	p.advance() // co.lang
	p.advance() // "."
	p.advance() // variants
}

// parseVariantTypeDeclaration parses a co.lang.type declaration whose right-hand
// side is `co.lang.variants( … )`.
func (p *parser) parseVariantTypeDeclaration(
	declName name,
	generics []symboltable.GenericTypeParam,
	kindTok scanlex.Token,
	annotations annotationSet,
	spanStart int,
) ast.Stmt {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	if kindTok.Value != "co.lang.type" {
		p.failf(kindTok, "%s is the variant-definition right-hand side of a co.lang.type declaration; %q takes an ordinary type expression", variantDefinitionName, kindTok.Value)
	}

	p.consumeVariantDefinitionHead()
	p.expect(scanlex.OPEN_PAREN, "to open a variant definition")

	var variants []ast.VariantConstructor
	declared := map[string]bool{}
	for !p.at(scanlex.CLOSE_PAREN) && !p.atEOF() {
		variant := p.parseVariantConstructorDeclaration()
		// Two variants of one name would give the enclosing type two constructors
		// that cannot be told apart at a use site.
		if declared[variant.Name] {
			p.reportf(p.cur(), "variant %q is declared more than once in %s", logicalName(variant.Name), logicalName(declName.Scanned))
		}
		declared[variant.Name] = true
		variants = append(variants, variant)

		if !p.accept(scanlex.COMMA) {
			break
		}
		if p.at(scanlex.CLOSE_PAREN) {
			p.failf(p.cur(), "a variant definition does not allow a trailing comma after its last constructor")
		}
	}

	p.expect(scanlex.CLOSE_PAREN, "to close a variant definition")
	p.statementEnd("a variant type declaration")

	if len(variants) == 0 {
		p.failf(kindTok, "%s defines a closed set of variants and requires at least one", variantDefinitionName)
	}

	typeParamNames := make([]string, 0, len(generics))
	for _, generic := range generics {
		typeParamNames = append(typeParamNames, generic.Name)
	}
	symb := p.typeConstructorSymbol(declName.Scanned)
	p.declareNamed(declName, symb)

	return ast.TypeConstructorStmt{NodeName: "TypeConstructorStmt", Span: p.spanFrom(spanStart), Name: declName.Scanned,
		TypeParams:    typeParamNames,
		GenericParams: generics,
		Variants:      variants,
		SDapst:        annotations.list(),
		Symb:          symb,
	}
}

// parseVariantConstructorDeclaration parses one entry of a variant definition.
//
// The head is DECLARED rather than looked up, so it is read as a plain
// identifier: a qualified name would name something that already exists, which a
// variant declaration cannot do. The payload inside the parentheses is an
// ordinary type list and resolves normally.
func (p *parser) parseVariantConstructorDeclaration() ast.VariantConstructor {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	variantName := p.parseIdentifier("as a variant constructor name")

	var typeArgs []string
	var payloadTypes []ast.Type
	p.expect(scanlex.OPEN_PAREN, "to open a variant constructor payload")
	if !p.at(scanlex.CLOSE_PAREN) {
		payloadTypes = append(payloadTypes, p.parseTypeExpression().fullType())
		for p.accept(scanlex.COMMA) {
			if p.at(scanlex.CLOSE_PAREN) {
				p.failf(p.cur(), "a variant constructor payload does not allow a trailing comma")
			}
			payloadTypes = append(payloadTypes, p.parseTypeExpression().fullType())
		}
		for _, t := range payloadTypes {
			typeArgs = append(typeArgs, actTypeOf(t))
		}
	}
	p.expect(scanlex.CLOSE_PAREN, "to close a variant constructor payload")

	symb := p.variantConstructorSymbol(variantName.Scanned)
	p.declareQuietly(variantName.Scanned, symb)

	return ast.VariantConstructor{
		Name:         variantName.Scanned,
		TypeArgs:     typeArgs,
		PayloadTypes: payloadTypes,
		Symb:         symb,
	}
}

// applyTypeDeclarationKind sets the symbol flag for a type-declaration-kind.
//
// `co.lang.kind` has no flag of its own: what it declares is recorded by TypeType
// together with the union shape its definition already sets.
func applyTypeDeclarationKind(symb *symboltable.TypeSymbol, kind string) {
	symb.TypeType = kind

	switch typeDeclarationKinds[kind] {
	case "alias":
		symb.Alias = true
	case "newtype":
		symb.NewType = true
	case "opaque":
		symb.OpaqueType = true
	case "subtype":
		symb.SubType = true
	case "supertype":
		symb.SuperType = true
	case "dependent":
		symb.DependentType = true
	}
}

// typeTypeOf classifies a type declaration's definition as a built-in, user-defined or
// algebraic type, which is the coarse category the AST records.
func typeTypeOf(definition typeRef, hasDefinition bool) string {
	if !hasDefinition {
		return "UDT"
	}
	switch definition.Form {
	case formUnion:
		return "ADT"
	case formFunction:
		return "FUN"
	case formForall:
		return "FORALL"
	}
	if _, isBuiltin := definition.Node.(ast.BuiltInDataType); isBuiltin {
		return "BDT"
	}
	return "UDT"
}

// signature-type-component — section 7.
//
//	signature-type-component = annotations, identifier, "co.lang.type",
//	                           [ "=", type-expression ], statement-end
//
// This is a type requirement inside a signature or module body: the member declares that
// an implementation must supply a type of that name, optionally with a default.
//
// It is one of the three places DECISION-GEN-001 still admits a
// declaration-head parameter clause, because an abstract generic type
// constructor is exactly what a signature has to be able to require
// (docs/language-ref.md, "Abstract Generic Type Constructors"). Its name is a
// member name inside a body, so it is an identifier and never "_".

// parseSignatureTypeComponent parses the signature-type-component production.
//
// Implements: signature-type-component
func (p *parser) parseSignatureTypeComponent(annotations annotationSet) ast.Stmt {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	declName := p.parseIdentifier("as a signature type component name")
	kindTok := p.cur()
	if kindTok.Kind != scanlex.BUILT_IN_KIND || kindTok.Value != "co.lang.type" {
		p.failf(kindTok, "expected \"co.lang.type\" in a signature type component, found %s", describeToken(kindTok))
	}
	p.advance()

	return p.parseTypeDeclaration(declName, nil, kindTok, annotations)
}

// associated-type-requirement and associated-type-binding — section 7.
//
//	associated-type-requirement = annotations, identifier,
//	                              "co.lang.associatedType", statement-end
//	associated-type-binding     = annotations, identifier,
//	                              "co.lang.associatedType", "=", type-expression,
//	                              statement-end
//
// The two are one spelling in two containers, told apart by the binding
// (docs/language-ref.md, "Associated Type Components"):
//
//	// Repository.fol — a SIGNATURE states the requirement
//	_ co.lang.signature = {
//	    Entity co.lang.associatedType;
//	    find(id co.lang.int)->(Entity);
//	}
//
//	// EmployeeRepositoryImpl.fol — a matching MODULE supplies the binding
//	_ co.lang.module->(signature=Repository, matches=Repository) = {
//	    Entity co.lang.associatedType = hr.employee.Employee;
//	}
//
// A requirement does not define a representation: it says every matching module
// must supply a compatible type of that name. The generic clause makes it an
// abstract type CONSTRUCTOR of the stated arity — `Stack(T) co.lang.associatedType;`
// requires one type argument without saying what constructs it.
//
// The container is what makes the binding legal, not the spelling: inside a
// matching module the name must correspond to a requirement the matched signature
// declared, so a module cannot use this form to introduce unrelated local types.
// That correspondence needs the signature, so it is a semantic check; the parser
// owns the shape and which container admits which form.

// atAssociatedTypeDeclaration reports whether the cursor begins an
// associated-type requirement or binding.
func (p *parser) atAssociatedTypeDeclaration() bool {
	if !p.atIdentifier() {
		return false
	}
	return p.lookaheadOnly(func() bool {
		p.advance() // the name
		return p.atBuiltinKind("co.lang.associatedType")
	})
}

// parseAssociatedTypeDeclaration parses the associated-type-requirement and
// associated-type-binding productions.
//
// requiresBinding says which container this is being read in: a signature states
// requirements and must not bind, while a module supplies bindings and must.
// Reporting the mismatch here is what keeps `Entity co.lang.associatedType;` in a
// module from silently becoming an unsatisfiable requirement.
//
// Implements: associated-type-requirement
// Implements: associated-type-binding
func (p *parser) parseAssociatedTypeDeclaration(annotations annotationSet, requiresBinding bool) ast.Stmt {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	declName := p.parseIdentifier("as an associated type name")
	kindTok := p.expectDeclarationKind("to declare an associated type")

	var definition typeRef
	bound := false
	if p.acceptOp("=") {
		definition = p.parseTypeExpression()
		bound = true
	}
	p.statementEnd("an associated type declaration")

	switch {
	case requiresBinding && !bound:
		p.failf(kindTok, "a module binds every associated type it declares, as in \"%s co.lang.associatedType = <type>;\"; the abstract form belongs to the signature", declName.Logical)
	case !requiresBinding && bound:
		p.failf(kindTok, "a signature declares an associated-type requirement without a binding, as in \"%s co.lang.associatedType;\"; each matching module supplies its own", declName.Logical)
	}

	symb := p.typeSymbol(declName.Scanned)
	symb.AssociatedType = true
	symb.ExplicitType = bound
	symb.IsGenericType = false

	decl := ast.TypeDeclarationStmt{NodeName: "TypeDeclarationStmt", Span: p.spanFrom(spanStart), Name: declName.Scanned,
		TypeParams: nil,
		Kind:       kindTok.Value,
		SubType_:   "associated",
		Typetype:   typeTypeOf(definition, bound),
		SDapst:     annotations.list(),
		KDapst:     annotations.list(),
		Symb:       symb,
	}
	if bound {
		decl.Type_ = definition.fullType()
		decl.NewTypeName = definition.actType()
	}
	return decl
}

// An external TYPE declaration — `@co.dap.declare(extern) Dept co.lang.struct;` — is not
// a declaration form of its own. It is written inside a class or unit body and matches
// pure-field-declaration, whose type-expression is the kind name
// (docs/language-ref.md, "Types external declaration"). There is no file-level forward
// spelling: primary-declaration admits only the block-bodied kinds.

// package-alias-declaration — section 6.
//
//	package-alias-declaration = filename-derived-name, "co.lang.package", "=",
//	                            package-alias-body, statement-end
//	package-alias-body        = "{", "name", ":", string-literal, "}"
//
// This is the package-ALIASING declaration. It renames the package segment a
// folder contributes, so a folder physically named /appl/hr/empl can be addressed
// as hr.emp. The declaration lives in the reserved `package.fol` source form
// inside the folder being renamed (docs/language-ref.md, "Package Source Files"):
//
//	// package.fol
//	_ co.lang.package = { name: "emp" };
//
// DECISION-PKG-001 settles two things that the older bare `emp co.lang.package;`
// form conflated. The logical leaf segment is data — a string in the body —
// rather than an identifier in the head, and `_` does NOT derive the literal name
// `package` from the reserved filename. The imports of that folder's members then
// use the alias:
//
//	@co.ddap.import(package="hr.emp.Employee", as="emp")
//
// The body's braces are MAP-shaped, not a declaration body, so DECISION-SYN-006
// requires the ";" and body-closure-guard does not apply.
