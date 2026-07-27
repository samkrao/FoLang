package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/helpers"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// parse_type_constructor_stmt parses a @co.dap.hokrt annotated type constructor.
//
// Syntax:
//
//	@co.dap.hokrt
//	Option(T) co.lang.type = Some(T) | None();
//
// The ddaps map must already contain the @co.dap.hokrt DirectiveStmt.
func parse_type_constructor_stmt(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	pr := ParseResult{
		Node:   nil,
		Errors: []helpers.ErrorInterface{},
	}

	// --- 1. Parse the type constructor name (e.g. "Option") ---
	name, err := p.expectError(scanlex.IDENTIFIER,
		"expected type constructor name after @co.dap.hokrt")
	p.addErr(err)

	// --- 2. Parse optional type parameters: (T, U, ...) ---
	typeParams := []string{}
	if p.currentTokenKind() == scanlex.OPEN_PAREN {
		p.advance() // consume '('
		for p.currentTokenKind() != scanlex.CLOSE_PAREN && p.currentTokenKind() != scanlex.EOF {
			param, e := p.expectError(scanlex.IDENTIFIER, "expected type parameter name")
			p.addErr(e)
			typeParams = append(typeParams, param.Value)
			if p.currentTokenKind() == scanlex.COMMA {
				p.advance()
			}
		}
		_, e := p.expect(scanlex.CLOSE_PAREN)
		p.addErr(e)
	}

	// --- 3. Expect the kind annotation: co.lang.type ---
	if ok, _ := checkKindsandTypes(p, KIND_, 0, "co.lang.type"); ok {
		p.advance()
	} else {
		e := p.errorExpection(
			"expected 'co.lang.type' after type constructor parameters, got "+p.currentToken().Value,
			helpers.InvalidSyntax)
		p.addErr(e)
	}

	// --- 4. Expect '=' ---
	_, err = p.expect(scanlex.ASSIGNMENT)
	p.addErr(err)

	// --- 5. Parse variant list: Ctor1(T) | Ctor2() | ... ---
	variants := []ast.VariantConstructor{}
	for {
		variantName, e := p.expectError(scanlex.IDENTIFIER, "expected variant constructor name")
		p.addErr(e)

		variantArgs := []string{}
		if p.currentTokenKind() == scanlex.OPEN_PAREN {
			p.advance() // consume '('
			for p.currentTokenKind() != scanlex.CLOSE_PAREN && p.currentTokenKind() != scanlex.EOF {
				arg, ae := p.expectError(scanlex.IDENTIFIER, "expected type argument in variant")
				p.addErr(ae)
				variantArgs = append(variantArgs, arg.Value)
				if p.currentTokenKind() == scanlex.COMMA {
					p.advance()
				}
			}
			_, ae := p.expect(scanlex.CLOSE_PAREN)
			p.addErr(ae)
		}

		vc := ast.VariantConstructor{
			Name:     variantName.Value,
			TypeArgs: variantArgs,
		}
		variants = append(variants, vc)

		// continue on '|', otherwise stop
		if p.currentTokenKind() == scanlex.PIPE {
			p.advance()
		} else {
			break
		}
	}

	// --- 6. Expect statement terminator ---
	_, err = p.expectAny(scanlex.SEMI_COLON, scanlex.COMMA)
	p.addErr(err)

	// --- 7. Build context (same pattern as parse_type_decl_stmt) ---
	actCtxId := p.Context_
	actSymbId := p.SymbolTable_

	p.Context_, p.SymbolTable_ = CreateNewContext(actCtxId.Id, string(symboltable.S_TypeConstructor))
	p.Context_.ContextType_ = string(symboltable.S_TypeConstructor)
	actCtxId.ChildCtxIds = append(actCtxId.ChildCtxIds, p.Context_.Id)
	p.Context_.ParentCtxSymbolTableId = actSymbId.Id
	p.Fs.AddContext(p.Context_)
	p.Fs.AddSymbolTable(p.SymbolTable_)

	tcStmt := ast.TypeConstructorStmt{
		Name:       name.Value,
		TypeParams: typeParams,
		Variants:   variants,
		Symb: &symboltable.TypeConstructor{
			SymbolDetails: symboltable.SymbolDetails{
				SymbolType_: string(symboltable.S_TypeConstructor),
			},
		},
	}

	p.Context_ = actCtxId
	p.SymbolTable_ = actSymbId

	updateContext(p, tcStmt, false, false)

	pr.Node = tcStmt
	return pr
}
