package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// type-declaration and its relatives — section 6.
//
//	type-declaration               = parameterized-type-declaration
//	                               | simple-type-declaration
//	parameterized-type-declaration = annotations, identifier,
//	                                 generic-parameter-clause, "co.lang.type",
//	                                 [ kind-options ], [ "=", type-expression ],
//	                                 statement-end
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
// The production is split in two. Only the parameterized form takes a declaration-head
// parameter clause, and only `co.lang.type` may be parameterized:
// `Option(T) co.lang.type = Some(T) | None();` declares a type constructor, while an
// alias, newtype, subtype or dependent type is always simple. The split is what lets the
// clause be rejected by kind rather than silently accepted and dropped.

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
// Implements: parameterized-type-declaration
// Implements: simple-type-declaration
func (p *parser) parseTypeDeclaration(declName name, generics []symboltable.GenericTypeParam, kindTok scanlex.Token, annotations annotationSet) ast.Stmt {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	// parameterized-type-declaration exists only for co.lang.type. Every other
	// kind of this family is simple-type-declaration and has no clause slot at
	// all, so a clause there is a grammar error rather than dead metadata.
	if len(generics) != 0 && kindTok.Value != "co.lang.type" {
		p.reportf(kindTok, "only a co.lang.type declaration may be parameterized; %q takes no declaration-head type parameters", kindTok.Value)
	}

	// A kind may carry options, as in co.lang.dependentType->(kind=length).
	options := p.parseOptionalKindOptions()

	var definition typeRef
	hasDefinition := false
	if p.acceptOp("=") {
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

	decl := ast.TypeDeclarationStmt{Span: p.spanFrom(spanStart), Name: declName.Scanned,
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
	// co.lang.dependentType->(kind=length) records which dependent kind applies.
	if kind, ok := options["kind"]; ok {
		if s, isString := kind.(string); isString {
			decl.DependentKind = s
		}
	}
	return decl
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
//	signature-type-component = annotations, identifier,
//	                           [ generic-parameter-clause ], "co.lang.type",
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
	generics := p.parseOptionalGenericParameterClause()

	kindTok := p.cur()
	if kindTok.Kind != scanlex.BUILT_IN_KIND || kindTok.Value != "co.lang.type" {
		p.failf(kindTok, "expected \"co.lang.type\" in a signature type component, found %s", describeToken(kindTok))
	}
	p.advance()

	return p.parseTypeDeclaration(declName, generics, kindTok, annotations)
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

// parsePackageAliasDeclaration parses the package-alias-declaration production.
//
// Implements: package-alias-declaration
func (p *parser) parsePackageAliasDeclaration(declName name, annotations annotationSet) ast.Stmt {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	// Alone among the declarations, this production does not begin with
	// `annotations`, so a decorated package alias is not admitted by the grammar.
	if !annotations.empty() {
		p.reportf(p.cur(), "a %q declaration takes no annotations", "co.lang.package")
	}

	p.expectOp("=", "before a package metadata body")
	segment := p.parsePackageAliasBody()
	p.statementEnd("a package declaration")

	return ast.PackageStmt{Span: p.spanFrom(spanStart), Symb: p.packageSymbol(segment)}
}

// parsePackageAliasBody parses the package-alias-body production and returns the
// logical leaf segment it supplies.
//
// The body is a fixed one-entry property list rather than an open map: the
// grammar spells the key `name` and the value as a string-literal, so anything
// else is a syntax error here instead of an unknown field the semantic phase
// would have to reject later.
//
// Implements: package-alias-body
func (p *parser) parsePackageAliasBody() string {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	p.expect(scanlex.OPEN_CURLY, "to open a package metadata body")

	key := p.cur()
	if logicalName(key.Value) != "name" {
		p.failf(key, "a package metadata body holds exactly one entry, %s; found %s", "`name: \"segment\"`", describeToken(key))
	}
	p.advance()
	p.expect(scanlex.COLON, "after the package metadata key \"name\"")

	valueTok := p.cur()
	segment, ok := literalText(valueTok)
	if !ok || valueTok.Kind != scanlex.STRING {
		p.failf(valueTok, "the package name must be a string literal naming the logical leaf segment, found %s", describeToken(valueTok))
	}
	p.advance()

	if !isFoLangIdentifier(segment) {
		p.reportf(valueTok, "package segment %q is not a valid FoLang identifier", segment)
	}

	p.expect(scanlex.CLOSE_CURLY, "to close a package metadata body")
	return segment
}
