package parser

import (
	"fmt"
	"strings"

	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/helpers"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
	"golang.org/x/exp/slices"
)

// parse_type_decl_stmt parses FoLang type-family declarations.
//
// Feature examples:
//
//	Employee co.lang.struct = { Id co.lang.int; Name co.lang.string; }
//	Id      co.lang.type    = co.lang.int;
//	UserId  co.lang.newtype = co.lang.int;
//	Result  co.lang.data    = Ok(User) | Err(co.lang.string);
//
// This function keeps the current codebase style: read the declaration name,
// classify the kind token, then route either into:
// 1. a body-based declaration parser (`{ ... }`)
// 2. an alias/newtype/type-expression parser
func parse_type_decl_stmt(p *parser, stmtK StmtKind, kwdrwd string, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	pr := ParseResult{
		Node:   nil,
		Errors: []helpers.ErrorInterface{},
	}

	token := p.advance()
	var explicitType ast.Type
	var typeExpr ast.TypeExpr
	alias := false
	adtt := false
	newt := false
	subtype := false
	supertype := false
	opaquetype := false
	associatedtype := false
	dependenttype := false
	hkt := false
	hot := false
	hrt := false
	typetype := "NORMAL"
	funType := false
	newTypeName := ""
	adt := "NORMAL"
	startToken := token.Kind

	name, err_ := p.expectError(scanlex.IDENTIFIER, fmt.Sprintf("Following %s expected type name however instead recieved %s instead\n",
		scanlex.TokenKindString(startToken), scanlex.TokenKindString(p.currentTokenKind())))
	p.addErr(err_)

	if p.SymbolTable_.ExistsType(*p.Fs, name.Value, string(symboltable.S_TypeSymbol)) {
		err_ = p.errorExpection("Type "+helpers.RemoveAfterUnderscore(name.Value, "_")+" already decclared", helpers.AlreadyDeclared)
		p.addErr(err_)
	}
	structKind, _ := checkKindsandTypes(p, KIND_, 0, "co.lang.struct")
	cstructKind, _ := checkKindsandTypes(p, KIND_, 0, "co.lang.cstruct")
	unionKind, _ := checkKindsandTypes(p, KIND_, 0, "co.lang.union")
	moduleKind, _ := checkKindsandTypes(p, KIND_, 0, "co.lang.module")
	clzKind, _ := checkKindsandTypes(p, KIND_, 0, "co.lang.class")
	typeKind, _ := checkKindsandTypes(p, KIND_, 0, "co.lang.type")
	kindKind, _ := checkKindsandTypes(p, KIND_, 0, "co.lang.newtype")
	subtypeKind, _ := checkKindsandTypes(p, KIND_, 0, "co.lang.subtype")
	supertypeKind, _ := checkKindsandTypes(p, KIND_, 0, "co.lang.supertype")
	opaquetypeKind, _ := checkKindsandTypes(p, KIND_, 0, "co.lang.opaquetype")
	associatedtypeKind, _ := checkKindsandTypes(p, KIND_, 0, "co.lang.associatedType")
	dependenttypeKind, _ := checkKindsandTypes(p, KIND_, 0, "co.lang.dependenttype")
	HigherKindType, _ := checkKindsandTypes(p, KIND_, 0, "co.lang.hkt")
	HigherOrderType, _ := checkKindsandTypes(p, KIND_, 0, "co.lang.hot")
	HigherRankType, _ := checkKindsandTypes(p, KIND_, 0, "co.lang.hrt")
	valueKind, _ := checkKindsandTypes(p, KIND_, 0, "co.lang.data")

	// Keep the current context/symbol table so we can enter a type-local scope
	// for body-based declarations and then restore the outer parse environment.
	actCtxId := p.Context_
	actSymbId := p.SymbolTable_

	// Feature examples:
	//   Employee co.lang.struct = { ... }
	//   Packet   co.lang.cstruct = { ... }
	//   ApiMod   co.lang.module = { ... }
	//
	// Body-based declarations open a nested type scope because their members are
	// declared inside the new type/module/class container.
	if structKind || cstructKind || unionKind || clzKind || moduleKind {
		p.advance()
		p.Context_, p.SymbolTable_ = CreateNewContext(actCtxId.Id, string(symboltable.S_TypeSymbol))
		p.Context_.ContextType_ = string(symboltable.S_TypeSymbol)
		actCtxId.ChildCtxIds = append(actCtxId.ChildCtxIds, p.Context_.Id)
		p.Context_.ParentCtxSymbolTableId = actSymbId.Id
		p.Fs.AddContext(p.Context_)
		p.Fs.AddSymbolTable(p.SymbolTable_)

	} else if subtypeKind {
		subtype = true
		p.advance()
	} else if supertypeKind {
		supertype = true
		p.advance()
	} else if opaquetypeKind {
		opaquetype = true
		p.advance()
	} else if associatedtypeKind {
		associatedtype = true
		p.advance()
	} else if dependenttypeKind {
		dependenttype = true
		p.advance()
	} else if typeKind {
		// Feature example:
		//   Id co.lang.type = co.lang.int;
		alias = true
		p.advance()

	} else if kindKind {
		// Feature example:
		//   UserId co.lang.newtype = co.lang.int;
		newt = true
		p.advance()
	} else if valueKind {
		// Feature example:
		//   Result co.lang.data = Ok(User) | Err(co.lang.string);
		alias = true
		adtt = true
		p.advance()
	} else if HigherKindType {
		hkt = true
		p.advance()
	} else if HigherOrderType {
		hot = true
		p.advance()

	} else if HigherRankType {
		hrt = true
		p.advance()
	} else {
		err := p.errorExpection("expected one of the kind but found "+p.currentToken().Value, helpers.InvalidSyntax)
		p.addErr(err)
	}
	if p.currentTokenKind() == scanlex.SEMI_COLON || p.currentTokenKind() == scanlex.COMMA {
		var tk scanlex.Token = p.currentToken()
		err_ = p.errorObj(&tk, "Expected type Definition")
		p.addErr(err_)
	}
	_, err := p.expect(scanlex.ASSIGNMENT)
	p.addErr(err)
	flag := false
	body := make([]ast.Stmt, 0)

	// Feature examples:
	//   Employee co.lang.struct = { ... }   -> parse body
	//   Id       co.lang.type   = co.lang.int; -> parse alias expression
	if !alias && !newt {
		flag, body = parse_type_definition(p, name, ddaps)
	} else {
		flag, typeExpr, explicitType, typetype, funType, newTypeName = parse_type_alias_newtype(p, newt, alias, adtt, false, ddaps)
	}
	if !flag {
		err_ = p.errorExpection("Expected type definition or aliase ", helpers.InvalidSyntax)
		p.addErr(err_)
	}
	hsymb := symboltable.HokrtlSymbol{
		HKType: hkt,
		HRType: hrt,
		HOType: hot,
	}

	symb := symboltable.TypeSymbol{
		AsExpr:         false,
		Alias:          alias,
		AssociatedType: associatedtype,
		DependentType:  dependenttype,
		OpaqueType:     opaquetype,
		SubType:        subtype,
		SuperType:      supertype,
		NewType:        newt,
		HOKRTyple:      hsymb,
		UDT:            true,
		FunType:        funType,
		ADT:            adtt,
		UnionType:      unionKind,

		SymbolDetails: symboltable.SymbolDetails{
			SymbolType_: string(symboltable.S_TypeSymbol),
		},
	}

	typde := ast.TypeDeclarationStmt{
		Name:        name.Value,
		Type_:       explicitType,
		Body:        body,
		Typetype:    typetype,
		Kind:        "using",
		NewTypeName: newTypeName,
		ADT_:        adt,
		Symb:        &symb,
	}

	if explicitType != nil {
		if bdt, ok := explicitType.(ast.BuiltInDataType); ok {
			typde.SubType_ = bdt.Value
		} else {
			typde.SubType_ = "fun"
		}

	} else {
		typde.TypeExpr = typeExpr
	}

	// Mark cstruct declarations (C-like value type for zone boundary safety)
	if cstructKind {
		typde.SubType_ = "CSTRUCT"
	}

	pr.Node = typde
	//before returning restoring current cotnext id to the original context id and symbol table id backup while entering
	p.Context_ = actCtxId
	p.SymbolTable_ = actSymbId

	updateContext(p, pr.Node, false, false)

	return pr
}

// parse_type_definition parses the body of a body-based type declaration.
//
// Feature examples:
//
//	Employee co.lang.struct = {
//	    Id   co.lang.int;
//	    Name co.lang.string;
//	}
//
//	Result co.lang.union = {
//	    Ok;
//	    Err;
//	}
func parse_type_definition(p *parser, name scanlex.Token, ddaps map[scanlex.DirectiveKind][]ast.Stmt) (bool, []ast.Stmt) {
	defer p.traceCurrent()()

	body := make([]ast.Stmt, 0)

	if p.currentTokenKind() != scanlex.OPEN_CURLY {
		return false, body
	}

	_, err := p.expect(scanlex.OPEN_CURLY)
	p.addErr(err)

	for p.hasTokens() && p.currentTokenKind() != scanlex.EOF && p.currentTokenKind() != scanlex.CLOSE_CURLY {
		var tr ParseResult
		if p.currentTokenKind() == scanlex.IDENTIFIER || p.currentTokenKind() == scanlex.COMPOSITE_IDENTIFER {
			// Feature examples inside a type body:
			//   Id co.lang.int;
			//   Name co.lang.string;
			//   EmbeddedType;
			//
			// A following type token means a field/member declaration. A trailing
			// semicolon directly after the identifier means an embedded/composable
			// reference.
			if p.nextTokenSafe(1).Kind == scanlex.IDENTIFIER || p.nextTokenSafe(1).Kind == scanlex.COMPOSITE_IDENTIFER || p.nextTokenSafe(1).Kind == scanlex.BUILT_IN_TYPE {
				tr = parse_decl_stmt(p, "field", false, true, false, VAR, "", false, ddaps)
			} else if p.nextTokenSafe(1).Kind == scanlex.SEMI_COLON {
				tr = parse_composable(p, ddaps)
			}
		}

		body = append(body, tr.Node.(ast.Stmt))
	}
	_, err = p.expect(scanlex.CLOSE_CURLY)
	p.addErr(err)
	return true, body
}

// parse_type_alias_newtype parses the right-hand side of alias/newtype/data
// declarations.
//
// Feature examples:
//
//	Id     co.lang.type    = co.lang.int;
//	UserId co.lang.newtype = co.lang.int;
//	Mapper co.lang.type    = (co.lang.int)->(co.lang.string);
//	Result co.lang.data    = Ok(User) | Err(co.lang.string);
func parse_type_alias_newtype(p *parser, newt bool, alias bool, adt bool, kind bool, ddaps map[scanlex.DirectiveKind][]ast.Stmt) (bool, ast.TypeExpr, ast.Type, string, bool, string) {
	defer p.traceCurrent()()

	var typeExpr ast.TypeExpr
	typetype := "NORMAL"
	var explicitType ast.Type
	funType := false
	newTypeName := "NORMAL"
	var errs []helpers.ErrorInterface
	if p.currentTokenKind() == scanlex.BUILT_IN_TYPE || p.currentToken().Kind == scanlex.IDENTIFIER || p.currentTokenKind() == scanlex.COMPOSITE_IDENTIFER {
		// Feature examples:
		//   Id     co.lang.type = co.lang.int;
		//   Result co.lang.data = Ok(User) | Err(co.lang.string);
		if alias {
			if adt {
				p.IsADT = true
				pr := parse_expr(p, defalt_bp, ddaps)
				expr := pr.Node
				typeExpr = expr.(ast.TypeExpr)
				if _, ok := expr.(ast.SDTExpr); ok {
					typetype = "UDT"
				} else {
					typetype = "ADT"
				}
			} else {
				explicitType, errs = parse_type(p, defalt_bp, ddaps)
				p.addErr(errs)
			}
		} else if newt {
			// Feature example:
			//   UserId co.lang.newtype = co.lang.int;
			//
			// Newtypes wrap a concrete underlying type and are not parsed through
			// the ADT/function-type branch.
			symbTab := *p.SymbolTable_
			v := p.currentToken().Value
			if !symbTab.ExistsType(*p.Fs, v, string(symboltable.S_TypeSymbol)) {

				explicitType, errs = parse_type(p, defalt_bp, ddaps)
				if errs != nil {

					err_ := p.errorObj(nil, "Invalid Type")
					p.addErr(err_)
				}

				if kind {
					if !slices.Contains(Fo_Types, explicitType.(ast.BuiltInDataType).Value) {
						err_ := p.errorExpection("Invalid type type", helpers.InvalidSyntax)
						p.addErr(err_)
					}
				}

			} else {
				explicitType, errs = parse_type(p, defalt_bp, ddaps)
				if errs != nil {

					err_ := p.errorObj(nil, "Invalid Type")
					p.addErr(err_)
				}

				if kind {
					if !slices.Contains(Fo_Types, explicitType.(ast.BuiltInDataType).Value) {
						err_ := p.errorExpection("Invalid type type", helpers.InvalidSyntax)
						p.addErr(err_)
					}
				}
			}
		}
		p.IsADT = false
	} else if p.currentTokenKind() == scanlex.OPEN_PAREN {
		// Feature example:
		//   Mapper co.lang.type = (co.lang.int)->(co.lang.string);
		//
		// Parenthesized type syntax is used for function types and some ADT
		// expressions.
		if alias {
			//function params and returns and create function type
			isFunType := checkFunctionType(p, 1)
			if isFunType {
				// type f = fun (int, int )->(int)
				var err []helpers.ErrorInterface
				explicitType, err = parse_fn_type(p, defalt_bp, ddaps)
				p.addErr(err)
				typetype = "FUN"
				funType = true
			} else if adt {
				p.IsADT = true
				pr := parse_expr(p, defalt_bp, ddaps)
				expr := pr.Node
				typeExpr = expr.(ast.TypeExpr)
				if _, ok := expr.(ast.SDTExpr); ok {
					typetype = "UDT"
				} else {
					typetype = "ADT"
				}
				p.IsADT = false
			} else {
				explicitType, errs = parse_type(p, defalt_bp, ddaps)
				p.addErr(errs)

			}
		} else {
			err_ := p.errorExpection("Subtyping for function types or union types unsupported ", helpers.UnSupported)
			p.addErr(err_)
		}
	} else {
		err_ := p.errorExpection("Expected Type but found "+p.currentToken().Value, helpers.InvalidSyntax)
		p.addErr(err_)
	}
	_, err_ := p.expectAny(scanlex.SEMI_COLON, scanlex.COMMA)
	p.addErr(err_)
	return true, typeExpr, explicitType, typetype, funType, newTypeName
}

// parse_dependent_type_decl parses a dependent type declaration:
//
//	x co.lang.dependentType->(kind=length) = co.lang.int->([5]);
//
// Entry: current token is the name IDENTIFIER; next token is co.lang.dependentType.
func parse_dependent_type_decl(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	pr := ParseResult{
		Node:   nil,
		Errors: []helpers.ErrorInterface{},
	}

	name, err_ := p.expect(scanlex.IDENTIFIER)
	p.addErr(err_)

	// consume co.lang.dependentType
	if ok, _ := checkKindsandTypes(p, KIND_, 0, "co.lang.dependentType"); ok {
		p.advance()
	} else {
		err_ = p.errorExpection("co.lang.dependentType expected but found "+p.currentToken().Value, helpers.InvalidSyntax)
		p.addErr(err_)
	}

	// parse optional ->(kind=...) metadata
	depKind := ""
	if p.currentTokenKind() == scanlex.ARROW {
		p.advance() // eat ->
		if p.currentTokenKind() == scanlex.OPEN_PAREN {
			p.advance() // eat (
			for p.hasTokens() && p.currentTokenKind() != scanlex.CLOSE_PAREN && p.currentTokenKind() != scanlex.EOF {
				tk := p.advance()
				key := strings.TrimSuffix(tk.Value, "_fo")
				if p.currentTokenKind() == scanlex.ASSIGNMENT {
					p.advance() // eat =
					valTk := p.advance()
					val := strings.TrimSuffix(valTk.Value, "_fo")
					if key == "kind" {
						depKind = val
					}
				}
				if p.currentTokenKind() == scanlex.COMMA {
					p.advance()
				}
			}
			_, err_ = p.expect(scanlex.CLOSE_PAREN)
			p.addErr(err_)
		}
	}

	actCtxId := p.Context_
	actSymbId := p.SymbolTable_
	p.Context_, p.SymbolTable_ = CreateNewContext(actCtxId.Id, string(symboltable.S_TypeSymbol))
	p.Context_.ContextType_ = string(symboltable.S_TypeSymbol)
	actCtxId.ChildCtxIds = append(actCtxId.ChildCtxIds, p.Context_.Id)
	p.Context_.ParentCtxSymbolTableId = actSymbId.Id
	p.Fs.AddContext(p.Context_)
	p.Fs.AddSymbolTable(p.SymbolTable_)

	_, err_ = p.expect(scanlex.ASSIGNMENT)
	p.addErr(err_)

	// parse RHS base type (e.g. co.lang.int)
	baseType, errs := parse_type(p, defalt_bp, ddaps)
	p.addErr(errs)

	// parse optional ->([constraint]) after the base type
	var constraintExpr ast.Expr
	if p.currentTokenKind() == scanlex.ARROW {
		p.advance() // eat ->
		if p.currentTokenKind() == scanlex.OPEN_PAREN {
			p.advance() // eat (
			cpr := parse_expr(p, defalt_bp, ddaps)
			if cpr.Node != nil {
				constraintExpr = cpr.Node.(ast.Expr)
			}
			_, err_ = p.expect(scanlex.CLOSE_PAREN)
			p.addErr(err_)
		}
	}

	// consume trailing semicolon or comma
	_, err_ = p.expectAny(scanlex.SEMI_COLON, scanlex.COMMA)
	p.addErr(err_)

	// build the type: wrap as DependentType if we have a base
	var depType ast.Type
	if baseType != nil {
		if ndBase, ok := baseType.(ast.NonDependentType); ok {
			depType = ast.DependentType{
				Base: ndBase,
				Expr: constraintExpr,
				Symb: &symboltable.TypeSymbol{
					SymbolDetails: symboltable.SymbolDetails{
						SymbolType_: string(symboltable.S_TypeSymbol),
					},
				},
			}
		} else {
			depType = baseType
		}
	}

	symb := symboltable.TypeSymbol{
		Alias: true,
		UDT:   true,
		SymbolDetails: symboltable.SymbolDetails{
			SymbolType_: string(symboltable.S_TypeSymbol),
		},
	}

	typde := ast.TypeDeclarationStmt{
		Name:          name.Value,
		Type_:         depType,
		DependentKind: depKind,
		SubType_:      "DEPENDENT",
		Symb:          &symb,
	}
	pr.Node = typde

	p.Context_ = actCtxId
	p.SymbolTable_ = actSymbId

	updateContext(p, pr.Node, false, false)
	return pr
}
