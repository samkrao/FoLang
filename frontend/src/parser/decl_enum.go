package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// enum-declaration — section 6.
//
//	enum-declaration = annotations, declaration-name,
//	                   [ generic-parameter-clause ], "co.lang.enum", "=",
//	                   enum-body
//	enum-body        = "{", [ enum-variant,
//	                          { enum-separator, enum-variant },
//	                          [ enum-separator ] ], body-close
//	enum-separator   = ","
//	enum-variant     = annotations, identifier,
//	                   [ "(", [ type-list ], ")" ],
//	                   [ "=", constant-expression ]
//
// DECISION-COL-001 is the rule that shapes this body: the comma is a SOFT item boundary
// between variants and the closing brace is the HARD structural end of the enum. A
// trailing comma is permitted. This is why an enum body is not parsed with the shared
// ";"-terminated member loop — its members are comma-separated, not
// semicolon-terminated.
//
// A variant may carry payload types, which makes the enum a tagged union, and may carry
// an explicit constant value.

// parseEnumDeclaration parses the enum-declaration production.
//
// Implements: enum-declaration
func (p *parser) parseEnumDeclaration(declName name, generics []symboltable.GenericTypeParam, annotations annotationSet) ast.Stmt {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	p.expectOp("=", "before an enum body")
	p.expect(scanlex.OPEN_CURLY, "to open an enum body")

	var variants []ast.Stmt

	for !p.at(scanlex.CLOSE_CURLY) && !p.atEOF() {
		startPos := p.pos

		var variant ast.Stmt
		ok := p.recoverItem(startPos, []scanlex.TokenKind{scanlex.COMMA, scanlex.CLOSE_CURLY}, func() {
			variant = p.parseEnumVariant()
		})
		if ok && variant != nil {
			variants = append(variants, variant)
		}

		// The comma is a soft boundary: it allows another variant but does not
		// terminate the enum.
		if !p.accept(scanlex.COMMA) {
			break
		}
	}

	p.expect(scanlex.CLOSE_CURLY, "to close an enum body")
	p.bodyClosureGuard("an enum body")

	symb := p.enumSymbol(declName.Scanned)
	applyTypeVisibility(&symb.SymbolDetails, annotations)

	return ast.TypeDeclarationStmt{
		Name:       declName.Scanned,
		TypeParams: generics,
		Body:       variants,
		Kind:       "co.lang.enum",
		SubType_:   "ENUM",
		Typetype:   "UDT",
		SDapst:     annotations.list(),
		KDapst:     annotations.list(),
		Symb:       symb,
	}
}

// parseEnumVariant parses the enum-variant production.
//
// A variant is a name, optionally with a payload type list and optionally with an
// explicit constant value:
//
//	Red, Green, Blue                       plain variants
//	Some(T), None()                        variants with payloads
//	Low = 1, High = 100                    variants with explicit values
//
// Implements: enum-variant
func (p *parser) parseEnumVariant() ast.Stmt {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	annotations := p.parseAnnotations()
	p.rejectOperatorPlacement(annotations, "an enum variant")
	variantName := p.parseIdentifier("as an enum variant name")

	// The optional payload type list makes this variant a constructor.
	var payload []ast.Type
	hasPayload := false
	if p.at(scanlex.OPEN_PAREN) {
		p.advance()
		if !p.at(scanlex.CLOSE_PAREN) {
			payload = p.parseTypeList()
		}
		p.expect(scanlex.CLOSE_PAREN, "to close an enum variant payload")
		hasPayload = true
	}

	// The optional explicit value. It is a constant-expression, so it cannot
	// contain an assignment.
	var value ast.Expr
	if p.acceptOp("=") {
		value = p.parseConstantExpression()
	}

	symb := p.varSymbol(variantName.Scanned, "co.lang.enum")
	symb.HasInitValue = value != nil

	decl := ast.VarDeclarationStmt{
		BasicVarStmt: ast.BasicVarStmt{
			Identifier:    variantName.Scanned,
			AssignedValue: value,
			Type_:         p.enumVariantType(variantName, payload, hasPayload),
			VarType:       "co.lang.enum",
			SDapst:        annotations.list(),
		},
		Symb: symb,
	}
	return decl
}

// enumVariantType builds the type a variant carries.
//
// A plain variant's type is the enum itself, which the semantic phase substitutes. A
// variant with a payload is a constructor, so its type is a function from the payload to
// the enum, which is what makes `Some(1)` a call.
func (p *parser) enumVariantType(variantName name, payload []ast.Type, hasPayload bool) ast.Type {
	if !hasPayload {
		return ast.SymbolTypeNode{
			Value:      variantName.Scanned,
			SymbolType: string(symboltable.S_EnumSymbol),
			Symb:       p.typeSymbol(variantName.Scanned),
		}
	}

	return ast.FunctionType{
		Params:  [][]ast.Parameter{parametersFromTypes(p, payload)},
		Results: nil, // the result is the enclosing enum, resolved semantically
		Symb:    p.typeSymbol(variantName.Scanned),
	}
}

// union-declaration — section 6.
//
//	union-declaration = annotations, declaration-name,
//	                    [ generic-parameter-clause ], "co.lang.union", "=",
//	                    union-body
//	union-body        = "{", { field-declaration }, body-close
//
// A union body holds named fields that share storage, so unlike a struct it admits no
// embedded types.

// parseUnionDeclaration parses the union-declaration production.
//
// Implements: union-declaration
func (p *parser) parseUnionDeclaration(declName name, generics []symboltable.GenericTypeParam, annotations annotationSet) ast.Stmt {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	p.expectOp("=", "before a union body")

	members := p.parseBracedBody("a union body", func() ast.Stmt {
		memberAnnotations := p.parseAnnotations()
		p.rejectOperatorPlacement(memberAnnotations, "a union field")
		return p.parseFieldDeclaration(memberAnnotations)
	})

	symb := p.unionSymbol(declName.Scanned)
	applyTypeVisibility(&symb.SymbolDetails, annotations)

	return ast.TypeDeclarationStmt{
		TypeParams: generics,
		Name:       declName.Scanned,
		Body:       members,
		Kind:       "co.lang.union",
		SubType_:   "UNION",
		Typetype:   "UDT",
		SDapst:     annotations.list(),
		KDapst:     annotations.list(),
		Symb:       symb,
	}
}

// data-declaration — section 6.
//
//	data-declaration = annotations, declaration-name,
//	                   [ generic-parameter-clause ], "co.lang.data", "=",
//	                   data-variant, { "|", data-variant }, statement-end
//	data-variant     = qualified-name, [ "(", [ type-list ], ")" ]
//
// A data declaration is an algebraic sum type written inline, so it ends with ";" rather
// than with a body brace:
//
//	uniontype co.lang.data = co.lang.int | co.lang.float;
//	Option(T) co.lang.data = Some(T) | None();

// parseDataDeclaration parses the data-declaration production.
//
// Implements: data-declaration
func (p *parser) parseDataDeclaration(declName name, generics []symboltable.GenericTypeParam, annotations annotationSet) ast.Stmt {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	p.expectOp("=", "before the variants of a data declaration")

	variants := []ast.VariantConstructor{p.parseDataVariant()}
	for p.acceptOp("|") {
		variants = append(variants, p.parseDataVariant())
	}

	p.statementEnd("a data declaration")

	// Retain the complete generic records. In particular, F(_) carries arity
	// metadata in TypeKind/Types that cannot be reconstructed from F's name.
	// TypeParams remains populated for compatibility with existing AST clients.
	typeParamNames := make([]string, 0, len(generics))
	for _, generic := range generics {
		typeParamNames = append(typeParamNames, generic.Name)
	}
	return ast.TypeConstructorStmt{
		Name:          declName.Scanned,
		TypeParams:    typeParamNames,
		GenericParams: generics,
		Variants:      variants,
		SDapst:        annotations.list(),
		Symb:          p.typeConstructorSymbol(declName.Scanned),
	}
}

// parseDataVariant parses the data-variant production.
//
// The name may be qualified, which is what lets a data declaration alias existing types
// as its variants, as in `co.lang.int | co.lang.float`.
//
// Implements: data-variant
func (p *parser) parseDataVariant() ast.VariantConstructor {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	var variantName string
	if p.at(scanlex.BUILT_IN_TYPE) {
		variantName = p.advance().Value
	} else {
		variantName = p.parseQualifiedName("as a data variant name").Scanned
	}

	var typeArgs []string
	var payloadTypes []ast.Type
	if p.at(scanlex.OPEN_PAREN) {
		p.advance()
		if !p.at(scanlex.CLOSE_PAREN) {
			payloadTypes = p.parseTypeList()
			for _, t := range payloadTypes {
				typeArgs = append(typeArgs, actTypeOf(t))
			}
		}
		p.expect(scanlex.CLOSE_PAREN, "to close a data variant payload")
	}

	return ast.VariantConstructor{
		Name:         variantName,
		TypeArgs:     typeArgs,
		PayloadTypes: payloadTypes,
		Symb:         p.variantConstructorSymbol(variantName),
	}
}
