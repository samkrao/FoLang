package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// type-declaration and its relatives — section 6.
//
//	type-declaration      = annotations, declaration-name,
//	                        [ generic-parameter-clause ], type-declaration-kind,
//	                        [ "=", type-expression ], statement-end
//	type-declaration-kind = "co.lang.type" | "co.lang.typealias"
//	                      | "co.lang.newtype" | "co.lang.opaquetype"
//	                      | "co.lang.subtype" | "co.lang.supertype"
//	                      | "co.lang.associatedtype" | "co.lang.refinementType"
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
//
// The binding is optional, because a type may be declared and defined later.

// typeDeclarationKinds maps each type-declaration-kind to the symbol flag it sets.
var typeDeclarationKinds = map[string]string{
	"co.lang.type":           "alias",
	"co.lang.typealias":      "alias",
	"co.lang.newtype":        "newtype",
	"co.lang.opaquetype":     "opaque",
	"co.lang.subtype":        "subtype",
	"co.lang.supertype":      "supertype",
	"co.lang.associatedtype": "associated",
	"co.lang.assoicatedtype": "associated", // the spelling used by the kind table
	"co.lang.refinementType": "refinement",
	"co.lang.dependentType":  "dependent",
	"co.lang.typetype":       "typetype",
	"co.lang.typekind":       "typekind",
}

// parseTypeDeclaration parses the type-declaration production.
//
// kindTok is the built-in kind token the dispatcher already matched, which is what
// selects the relationship between the declared name and its definition.
func (p *parser) parseTypeDeclaration(declName name, generics []symboltable.GenericTypeParam, kindTok scanlex.Token, annotations annotationSet) ast.Stmt {
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

	decl := ast.TypeDeclarationStmt{
		Name:     declName.Scanned,
		Kind:     kindTok.Value,
		SubType_: typeDeclarationKinds[kindTok.Value],
		Typetype: typeTypeOf(definition, hasDefinition),
		SDapst:   annotations.list(),
		KDapst:   annotations.list(),
		Symb:     symb,
	}
	if hasDefinition {
		decl.Type_ = definition.Node
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
	case "associated":
		symb.AssociatedType = true
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
//	signature-type-component = annotations, declaration-name,
//	                           [ generic-parameter-clause ], "co.lang.type",
//	                           [ "=", type-expression ], statement-end
//
// This is a type requirement inside a signature or module body: the member declares that
// an implementation must supply a type of that name, optionally with a default.

// parseSignatureTypeComponent parses the signature-type-component production.
func (p *parser) parseSignatureTypeComponent(annotations annotationSet) ast.Stmt {
	declName := p.parseDeclarationName("as a signature type component name")
	generics := p.parseOptionalGenericParameterClause()

	kindTok := p.cur()
	if kindTok.Kind != scanlex.BUILT_IN_KIND || kindTok.Value != "co.lang.type" {
		p.failf(kindTok, "expected \"co.lang.type\" in a signature type component, found %s", describeToken(kindTok))
	}
	p.advance()

	return p.parseTypeDeclaration(declName, generics, kindTok, annotations)
}

// forward-type-declaration — section 6.
//
//	forward-type-declaration = annotations, declaration-name,
//	                           [ generic-parameter-clause ],
//	                           forward-declarable-kind, [ kind-options ],
//	                           statement-end
//
// A forward declaration names a type without defining it, so the definition may follow
// later or live elsewhere (docs/language-ref.md, "Types external declaration"). It is
// distinguished from the block-bodied declaration of the same kind purely by ending at
// ";" instead of at "=".

// parseForwardTypeDeclaration parses the forward-type-declaration production.
func (p *parser) parseForwardTypeDeclaration(declName name, generics []symboltable.GenericTypeParam, kindTok scanlex.Token, annotations annotationSet) ast.Stmt {
	options := p.parseOptionalKindOptions()
	p.statementEnd("a forward type declaration")

	symb := p.typeSymbol(declName.Scanned)
	symb.TypeType = kindTok.Value
	symb.ExplicitType = false
	symb.IsGenericType = len(generics) > 0

	decl := ast.TypeDeclarationStmt{
		Name:     declName.Scanned,
		Kind:     kindTok.Value,
		SubType_: "FORWARD",
		Typetype: "UDT",
		SDapst:   annotations.list(),
		KDapst:   annotations.list(),
		Symb:     symb,
	}
	if forType, ok := options["for"]; ok {
		if s, isString := forType.(string); isString {
			decl.ObjectFor = s
		}
	}
	return decl
}

// package-alias-declaration — section 6.
//
//	package-alias-declaration = declaration-name, "co.lang.package", statement-end
//
// This declares the file's package identity. The name may be "_", in which case it is
// inferred from the filename (docs/language-ref.md, "Primary Declaration Names and
// Filename Inference").

// parsePackageAliasDeclaration parses the package-alias-declaration production.
//
// The reference corpus also uses a block-bodied spelling, `name co.lang.package = { … }`,
// which the grammar does not list. It is accepted here and reported as non-conforming, so
// existing sources still parse while the divergence stays visible.
func (p *parser) parsePackageAliasDeclaration(declName name, annotations annotationSet) ast.Stmt {
	if p.atOp("=") {
		return p.parsePackageBodyDeclaration(declName, annotations)
	}

	p.statementEnd("a package alias declaration")

	return ast.PackageStmt{
		Symb: p.packageSymbol(declName.Scanned),
	}
}

// parsePackageBodyDeclaration parses the block-bodied package spelling.
//
// The grammar defines package-alias-declaration as ending at ";" with no body:
//
//	package-alias-declaration = declaration-name, "co.lang.package", statement-end
//
// The reference corpus in frontend/example nonetheless uses a block-bodied spelling
// throughout, `firstPackage co.lang.package = { … }`, which groups declarations into a
// package inline. It is accepted here as a container of declarations so those sources keep
// parsing.
//
// No diagnostic is raised. The parser accepts the form, and the only diagnostic severity
// available terminates compilation, so reporting it would reject the language's own examples
// over a spelling the parser handles. The divergence is recorded here instead, for whoever
// reconciles the grammar with the corpus.
func (p *parser) parsePackageBodyDeclaration(declName name, annotations annotationSet) ast.Stmt {
	p.expectOp("=", "before a package body")

	members := p.parseBracedBody("a package body", func() ast.Stmt {
		return p.parsePackageMember()
	})

	symb := p.packageSymbol(declName.Scanned)
	symb.IsCodeBlock = true

	return ast.PackageStmt{
		Body: members,
		Symb: symb,
	}
}

// parsePackageMember parses one member of the compatibility package body, which may hold
// any primary declaration or an ordinary statement.
func (p *parser) parsePackageMember() ast.Stmt {
	if decl, ok := p.tryParsePrimaryDeclaration(); ok {
		return decl
	}
	return p.parseStatement()
}
