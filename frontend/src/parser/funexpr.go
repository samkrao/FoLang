package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/helpers"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// parse_lambda_expr parses inline lambdas of the form |x, y| => expr.
// These are only valid as callback arguments to collection operations.
func parse_lambda_expr(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	res := ParseResult{Errors: nil, Node: nil}

	// Consume opening |
	_, err_ := p.expect(scanlex.PIPE)
	p.addErr(err_)

	// Create a child context/scope for the lambda parameters
	savedCtx := p.Context_
	savedSym := p.SymbolTable_
	lambdaCtx, lambdaSymTab := CreateNewContext(savedCtx.Id, string(symboltable.S_LambdaSymbol))
	p.Context_ = lambdaCtx
	p.SymbolTable_ = lambdaSymTab
	p.Context_.ContextType_ = string(symboltable.S_LambdaSymbol)
	p.Context_.ParentCtxSymbolTableId = savedSym.Id
	savedCtx.ChildCtxIds = append(savedCtx.ChildCtxIds, p.Context_.Id)
	p.Fs.AddContext(p.Context_)
	p.Fs.AddSymbolTable(p.SymbolTable_)

	// Parse comma-separated parameter names until closing |
	params := make([]ast.Parameter, 0)
	for p.hasTokens() && p.currentTokenKind() != scanlex.PIPE {
		nameTok, err_ := p.expect(scanlex.IDENTIFIER)
		p.addErr(err_)
		sd := symboltable.SymbolDetails{
			Name_: nameTok.Value,
		}
		vd := symboltable.VariableDetails{
			SymbolDetails: sd,
			ExplicitType:  false,
			Inferred:      true,
			SubType_:      "co.lang.var",
			SubID:         "VAR",
			VarType:       GetVarType(PARAM),
			IsParam:       true,
		}
		symb := symboltable.VarSymbol{
			VariableDetails: vd,
		}
		vstmt := ast.BasicVarStmt{
			VarType:    GetVarType(PARAM),
			Identifier: nameTok.Value,
		}
		varDecl := ast.VarDeclarationStmt{
			Symb:         &symb,
			BasicVarStmt: vstmt,
		}

		param := ast.Parameter{
			SymbolDeclStmt: varDecl,
			Name_:          nameTok.Value,
			Scope:          "LEXICAL",
			WhatType:       "VAR",
		}
		params = append(params, param)

		// Register param in lambda scope
		updateContext(p, varDecl, false, false)

		if p.currentTokenKind() == scanlex.COMMA {
			p.advance()
		}
	}

	// Consume closing |
	_, err_ = p.expect(scanlex.PIPE)
	p.addErr(err_)

	// Consume =>
	_, err_ = p.expect(scanlex.EQGT)
	p.addErr(err_)

	// Parse single-expression body
	body := parse_expr(p, assignment, ddaps)

	// Restore outer scope
	p.Context_ = savedCtx
	p.SymbolTable_ = savedSym

	lambda := ast.LambdaExpr{
		Parameters: params,
		Body:       body.Node.(ast.Expr),
	}
	res.Node = lambda
	return res
}

// parse_function_object_declaration_stmt parses a function object variable declaration:
//
//	someFArg co.lang.function = (a co.lang.int, b co.lang.int) -> (co.lang.int) = { body }
//	oObj co.lang.function = add;
func parse_function_object_declaration_stmt(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	pr := ParseResult{Errors: nil, Node: nil}

	// 1. Parse variable name
	nameTok, err_ := p.expect(scanlex.IDENTIFIER)
	p.addErr(err_)
	name := nameTok.Value

	// 2. Skip "co.lang.function" token
	p.advance()

	// 3. Consume "="
	_, err_ = p.expect(scanlex.ASSIGNMENT)
	p.addErr(err_)

	var assignedValue ast.Expr

	if p.currentTokenKind() == scanlex.OPEN_PAREN {
		// Inline function: (params) -> (returns) = { body }
		savedCtx := p.Context_
		savedSym := p.SymbolTable_
		p.Context_, p.SymbolTable_ = CreateNewContext(savedCtx.Id, string(symboltable.S_FunctionSymbol))
		p.Context_.ContextType_ = string(symboltable.S_FunctionSymbol)
		p.Context_.ParentCtxSymbolTableId = savedSym.Id
		savedCtx.ChildCtxIds = append(savedCtx.ChildCtxIds, p.Context_.Id)
		p.Fs.AddContext(p.Context_)
		p.Fs.AddSymbolTable(p.SymbolTable_)

		functionParams, returnType, functionBody, asExpr, _, errs := parse_fn_params_and_body(p, false, false, "LEXICAL", ddaps)
		p.addErr(errs)

		p.Context_ = savedCtx
		p.SymbolTable_ = savedSym

		fnExpr := ast.FunctionExpr{
			Parameters: functionParams[0],
			ReturnType: returnType,
			Body:       functionBody,
			AsExpr:     asExpr,
		}
		assignedValue = fnExpr
	} else {
		// Assignment from existing function: oObj co.lang.function = add;
		exprResult := parse_expr(p, assignment, ddaps)
		p.addErr(exprResult.Errors)
		assignedValue = exprResult.Node.(ast.Expr)
	}
	sd := symboltable.SymbolDetails{
		Name_: name,
	}
	vd := symboltable.VariableDetails{
		SymbolDetails: sd,
		ExplicitType:  true,
		HasInitValue:  true,
		Inferred:      true,
		SubType_:      "co.lang.function",
		SubID:         "FUNCTION_OBJECT",
		VarType:       GetVarType(PARAM),
		IsParam:       true,
	}
	symb := symboltable.VarSymbol{
		VariableDetails: vd,
	}
	vstmt := ast.BasicVarStmt{
		AssignedValue: assignedValue,
		VarType:       GetVarType(VAR),
		Identifier:    name,
	}
	varDecl := ast.VarDeclarationStmt{
		BasicVarStmt: vstmt,
		Symb:         &symb,
	}
	varDecl.SetDap(ddaps)

	pr.Node = varDecl
	updateContext(p, varDecl, false, false)
	return pr
}

// parse_anon_fn_expr parses an anonymous function expression:
//
//	(a co.lang.int, b co.lang.int) -> (co.lang.int) = { this.return a + b; }
//
// Called from parse_grouping_expr_st when checkFunctionType detects function syntax.
func parse_anon_fn_expr(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	res := ParseResult{Errors: nil, Node: nil}

	name := "__anon_fn_" + helpers.GenUnique(4)

	savedCtx := p.Context_
	savedSym := p.SymbolTable_
	p.Context_, p.SymbolTable_ = CreateNewContext(savedCtx.Id, string(symboltable.S_FunctionPattern))
	p.Context_.ContextType_ = string(symboltable.S_FunctionSymbol)
	p.Context_.ParentCtxSymbolTableId = savedSym.Id
	savedCtx.ChildCtxIds = append(savedCtx.ChildCtxIds, p.Context_.Id)
	p.Fs.AddContext(p.Context_)
	p.Fs.AddSymbolTable(p.SymbolTable_)

	functionParams, returnType, functionBody, asExpr, _, errs := parse_fn_params_and_body(p, false, false, "LEXICAL", ddaps)
	p.addErr(errs)

	p.Context_ = savedCtx
	p.SymbolTable_ = savedSym
	fnsymb := symboltable.FunctionSymbol{
		SymbolDetails: symboltable.SymbolDetails{
			Name_:       name,
			SymbolType_: string(symboltable.S_FunctionSymbol),
			IsInternal_: true,
			Type_:       "",
		},
	}
	fnExpr := ast.FunctionExpr{
		Parameters: functionParams[0],
		ReturnType: returnType,
		Body:       functionBody,
		AsExpr:     asExpr,
		Symb:       &fnsymb,
	}
	res.Node = fnExpr
	return res
}

// parse_anon_type_expr parses an anonymous type expression in value context:
//
//	co.lang.class{ name co.lang.string }
//	co.lang.class{}
//
// The result can be chained with .new() for instantiation.
func parse_anon_type_expr(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	res := ParseResult{Errors: nil, Node: nil}

	kindTok := p.currentToken()
	p.advance() // consume BUILT_IN_KIND

	if kindTok.Value == "co.lang.class" {
		name := "__anon_class_" + helpers.GenUnique(4)

		savedCtx := p.Context_
		savedSym := p.SymbolTable_
		p.Context_, p.SymbolTable_ = CreateNewContext(savedCtx.Id, string(symboltable.S_TypeSymbol))
		p.Context_.ContextType_ = string(symboltable.S_ClassSymbol)
		savedCtx.ChildCtxIds = append(savedCtx.ChildCtxIds, p.Context_.Id)
		p.Context_.ParentCtxSymbolTableId = savedSym.Id
		p.Fs.AddContext(p.Context_)
		p.Fs.AddSymbolTable(p.SymbolTable_)

		body := parseClassBody(p, ddaps)

		p.Context_ = savedCtx
		p.SymbolTable_ = savedSym

		cls := ast.ClassDeclarationStmt{
			Name: name,
			Body: body,
		}

		res.Node = ast.StatementExpr{
			Statement: cls,
		}
	} else {
		p.addErr(p.errorObj(nil, "Anonymous type expressions are only supported for co.lang.class"))
	}

	return res
}

// parse_closure_or_curry_stmt parses closure and curried function declarations:
//
//	closure(factor int) => (x int) = x * factor;
//	curry(factor int)(val int) = factory * val;
func parse_closure_or_curry_stmt(p *parser, isClosure bool, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	pr := ParseResult{Errors: nil, Node: nil}

	// 1. Parse function name
	nameTok, err_ := p.expect(scanlex.IDENTIFIER)
	p.addErr(err_)
	name := nameTok.Value

	// 2. Set up function scope
	savedCtx := p.Context_
	savedSym := p.SymbolTable_
	p.Context_, p.SymbolTable_ = CreateNewContext(savedCtx.Id, string(symboltable.S_FunctionSymbol))
	p.Context_.ContextType_ = string(symboltable.S_FunctionSymbol)
	savedCtx.ChildCtxIds = append(savedCtx.ChildCtxIds, p.Context_.Id)
	p.Context_.ParentCtxSymbolTableId = savedSym.Id
	p.Fs.AddContext(p.Context_)
	p.Fs.AddSymbolTable(p.SymbolTable_)

	// 3. Parse parameter groups
	allParams := [][]ast.Parameter{}

	firstGroup := parseSimpleParamGroup(p, ddaps)
	allParams = append(allParams, firstGroup)

	if isClosure {
		// Closure: name(captures) => (params) = expr
		_, err_ = p.expect(scanlex.EQGT)
		p.addErr(err_)

		secondGroup := parseSimpleParamGroup(p, ddaps)
		allParams = append(allParams, secondGroup)
	} else {
		// Curry: name(params1)(params2)... = expr
		for p.currentTokenKind() == scanlex.OPEN_PAREN {
			group := parseSimpleParamGroup(p, ddaps)
			allParams = append(allParams, group)
		}
	}

	// 4. Consume = and parse expression body
	_, err_ = p.expect(scanlex.ASSIGNMENT)
	p.addErr(err_)

	bodyExpr := parse_expr(p, assignment, ddaps)
	var body []ast.Stmt
	if bodyExpr.Node != nil {
		body = []ast.Stmt{ast.ExpressionStmt{
			Expression: bodyExpr.Node.(ast.Expr),
		}}
	}

	// Consume optional semicolon
	if p.currentTokenKind() == scanlex.SEMI_COLON {
		p.advance()
	}

	// 5. Restore context
	p.Context_ = savedCtx
	p.SymbolTable_ = savedSym

	symb := symboltable.FunctionSymbol{
		Curried:         len(allParams) > 1 && !isClosure,
		Closure:         isClosure,
		InnerFunction:   false,
		ParentIsFunc:    false,
		Variadic:        false,
		VariantTypeArgs: false,
		OptionalArgs:    false,
		DefaultParams:   false,
		NamedParams:     false,
		FunTypeParRet:   false,
		IsInner:         false,
	}

	fn := ast.FunctionDeclarationStmt{
		Name:       name,
		Parameters: allParams,
		Body:       body,
		AsExpr:     true,
		Scope:      "LEXICAL",
		Symb:       &symb,
	}
	fn.SetDap(ddaps)
	pr.Node = fn

	updateContext(p, fn, false, false)
	return pr
}

// parseSimpleParamGroup parses a single parenthesized parameter group: (name type, name type, ...)
func parseSimpleParamGroup(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) []ast.Parameter {
	defer p.traceCurrent()()

	_, err_ := p.expect(scanlex.OPEN_PAREN)
	if err_ != nil {
		return nil
	}

	params := make([]ast.Parameter, 0)
	dddaps := map[scanlex.DirectiveKind][]ast.Stmt{
		scanlex.ANNOTATION: ddaps[scanlex.ANNOTATION],
		scanlex.DECORATOR:  ddaps[scanlex.DECORATOR],
	}

	for p.hasTokens() && p.currentTokenKind() != scanlex.CLOSE_PAREN {
		tr := parse_fn_fn_params(p, "param", false, true, false, false, dddaps)

		syb := tr.Node.(ast.SymbolDeclStmt)
		var defValue ast.Expr
		var nam string
		switch s := syb.(type) {
		case ast.VarDeclarationStmt:
			defValue = s.AssignedValue
			nam = s.Identifier
		case ast.PointerVariableDeclStmt:
			defValue = s.AssignedValue
			nam = s.Identifier
		case ast.ArrayVariableDeclStmt:
			defValue = s.AssignedValue
			nam = s.Identifier
		case ast.AddressVariableDeclStmt:
			defValue = s.AssignedValue
			nam = s.Identifier
		case ast.RefVariableDeclStmt:
			defValue = s.AssignedValue
			nam = s.Identifier
		}

		whatType := string(symboltable.S_VarSymbol)
		if len(tr.OtherDetails) > 0 {
			if val, ok := tr.OtherDetails["FunParam"]; ok {
				if v, okk := val.(bool); okk && v {
					whatType = string(symboltable.S_FunctionSymbol)
				}
			}
		}

		param := ast.Parameter{
			SymbolDeclStmt: syb,
			HasDefault:     false,
			Scope:          "LEXICAL",
			Default:        defValue,
			Name_:          nam,
			WhatType:       whatType,
			DefaultArgs:    tr.OtherDetails["DefaultArgs"].(bool),
			OptionalArgs:   tr.OtherDetails["OptionalArgs"].(bool),
			NamedArgs:      tr.OtherDetails["NamedArgs"].(bool),
			ThunkArgs:      tr.OtherDetails["ThunkArgs"].(bool),
			Variadic:       tr.OtherDetails["VarArgs"].(bool),
		}
		params = append(params, param)

		if !p.currentToken().IsOneOfMany(scanlex.CLOSE_PAREN, scanlex.EOF) {
			p.expect(scanlex.COMMA)
		}
	}

	p.expect(scanlex.CLOSE_PAREN)
	return params
}

func parse_fn_expr(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	res := ParseResult{
		Errors: nil,
		Node:   nil,
	}
	_, err_ := p.expect(scanlex.BUILT_IN_KIND)
	p.addErr(err_)
	functionParams, returnType, functionBody, asExpr, _, errs := parse_fn_params_and_body(p, false, false, "LEXICAL", ddaps)
	p.addErr(errs)
	res.Node = ast.FunctionExpr{
		Parameters: functionParams[0],
		ReturnType: returnType,
		Body:       functionBody,
		AsExpr:     asExpr,
	}
	return res
}

// parse_special_method_stmt parses @@new and @@init special method declarations:
//
//	@@new()->(co.lang.uninit)={ self.return co.const.none }
//	@@new(a co.lang.typevalue, b co.lang.typevalue)->(co.lang.uninit)={ ... }
//	@@init() = {}
//	@@init(id T, name R) = { this.parent.init(); this.id = id; this.name = name; }
//
// These are class-level constructor/initializer methods prefixed with @@.
func parse_special_method_stmt(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	pr := ParseResult{Errors: nil, Node: nil}

	// 1. Consume @@
	_, err_ := p.expect(scanlex.DOUBLE_AT)
	p.addErr(err_)

	// 2. Read method name (new or init)
	nameTok, err_ := p.expect(scanlex.IDENTIFIER)
	p.addErr(err_)
	name := "@@" + nameTok.Value

	// 3. Set up function scope
	savedCtx := p.Context_
	savedSym := p.SymbolTable_
	p.Context_, p.SymbolTable_ = CreateNewContext(savedCtx.Id, string(symboltable.S_FunctionSymbol))
	p.Context_.ContextType_ = string(symboltable.S_FunctionSymbol)
	savedCtx.ChildCtxIds = append(savedCtx.ChildCtxIds, p.Context_.Id)
	p.Context_.ParentCtxSymbolTableId = savedSym.Id
	p.Fs.AddContext(p.Context_)
	p.Fs.AddSymbolTable(p.SymbolTable_)

	// 4. Parse (params) -> (returns) = { body } or (params) = { body }
	functionParams, returnType, functionBody, asExpr, _, errs := parse_fn_params_and_body(p, false, false, "LEXICAL", ddaps)
	p.addErr(errs)

	// 5. Restore context
	p.Context_ = savedCtx
	p.SymbolTable_ = savedSym
	symb := symboltable.FunctionSymbol{
		InnerFunction:        false,
		ParentIsFunc:         false,
		Variadic:             false,
		VariantTypeArgs:      false,
		OptionalArgs:         false,
		DefaultParams:        false,
		NamedParams:          false,
		FunTypeParRet:        false,
		RestrictedToOverload: true,
		IsInner:              false,
		SymbolDetails: symboltable.SymbolDetails{
			SymbolType_: string(symboltable.S_FunctionSymbol),
		},
	}
	fn := ast.FunctionDeclarationStmt{
		Name:       name,
		Parameters: functionParams,
		Body:       functionBody,
		ReturnType: returnType,
		AsExpr:     asExpr,
		Scope:      "LEXICAL",
		Symb:       &symb,
	}
	fn.SetDap(ddaps)
	pr.Node = fn

	updateContext(p, fn, false, false)
	return pr
}
