package parser

import (
	"fmt"
	"strings"

	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/helpers"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

/*
	variable declarations, initialization, assignments and access
	expressions
	lambda expressions
	comprehensions
	let bindings
	where and let expressions
	function patterns
	inner functions (normal, cuurried, cloasures, overloaded)
	anonymous functions
	connditions, loops, matchers, each, containns, ternary operators
    function objects
	blocks
	named blocks
	labels
    defer functions
	lazy functions and calls
	function delegates

*/

func parse_fun_stmt(p *parser, funHasResults bool, innerFun bool, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	pr := ParseResult{}
	isFun := p.IsFunction
	p.IsFunction = false

	// all definitions start with Identifier

	if p.currentTokenKind() == scanlex.IDENTIFIER {
		_, whatKind := stmtKind(p, ddaps, -1)

		if p.currentToken().Kind == scanlex.IDENTIFIER && (whatKind != GRP_EXPR_CALL && whatKind != ILLEGAL_GRP) {
			pr := parse_type_kind_block(p, FUN, ddaps)
			n, ok := pr.Node.(ast.FunctionDeclarationStmt)
			if ok {
				n.Symb.IsInner = innerFun
			}
			return ParseResult{Node: n, Errors: pr.Errors}
		} else {
			pr = parse_stmt(p, CODE, ddaps)
			p.IsFunction = isFun
			return pr
		}

	} else if p.currentTokenKind() == scanlex.KEYWORD {
		if p.currentToken().Value == "this" {
			if p.nextTokenSafe(1).Kind == scanlex.ASSIGNMENT {

				p.IsFunction = isFun
				return pr
			} else {
				err_ := p.errorExpection("Expecting Assignment but found "+p.currentToken().Value, helpers.InvalidSyntax)
				p.addErr(err_)
			}
		} else if funHasResults {
			err_ := p.errorExpection("Expecting this but found "+p.currentToken().Value, helpers.InvalidSyntax)
			p.addErr(err_)
		}
	} else if p.currentTokenKind() == scanlex.BUIL_IN_STMT_EXPRS {
		if p.currentToken().Value == "this.return" {
			p.advance()
			p.IsFunction = isFun

			tr := parse_expression_stmt(p, ddaps)
			//fmt.Println(tr.Node)
			//	_, err_ := p.expect(scanlex.SEMI_COLON)
			//	p.addErr(err_)
			tr.Node = ast.ReturnStmt{
				StmtExpr_:    tr.Node.(ast.ExpressionStmt),
				MultiReturns: true,
				Symb: &symboltable.Symbol{
					SymbolDetails: symboltable.SymbolDetails{
						SymbolType_: string(symboltable.S_StatmentSymbol),
					},
				},
			}
			return tr

		} else if funHasResults {
			err_ := p.errorExpection("Found this", helpers.InvalidSyntax)
			p.addErr(err_)
		}
	} else {
		pr = parse_stmt(p, CODE, ddaps)
		p.IsFunction = isFun
		return pr
	}
	p.IsFunction = isFun
	pr = parse_stmt(p, CODE, ddaps)

	return pr
}

// parse_fn_fn_params parses one function parameter entry inside a parameter list.
//
// Feature examples:
//
//	fun add(a co.lang.int, b co.lang.int)->(co.lang.int) = { ... }
//	fun apply(~name co.lang.string, value? co.lang.int)->() = { ... }
//	fun collect(...rest co.lang.int)->() = { ... }
//	fun map(fn (co.lang.int)->(co.lang.int))->() = { ... }
//
// The returned OtherDetails map carries parameter modifiers such as named,
// default, optional, thunk, and variadic so the caller can shape ast.Parameter.
func parse_fn_fn_params(p *parser, type_ string, defaultArg bool, allowinit bool, isfunParam bool, noName bool, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	var pr ParseResult
	NamedArgs := false
	if p.currentTokenKind() == scanlex.TILD {
		NamedArgs = true
	}
	varArgs := false
	if p.currentTokenKind() == scanlex.DOT_DOT_DOT {
		varArgs = true
	}
	fnType := false
	if (p.currentTokenKind() == scanlex.IDENTIFIER || p.currentTokenKind() == scanlex.COMPOSITE_IDENTIFER) && p.nextTokenSafe(1).Value == "(" && checkFunctionType(p, 2) {
		tk := p.advance()
		pr = parse_fn_gen_declaration(p, FUNTYPE, true, true, true, tk.Value, ddaps)
		fnType = true
	} else if p.currentTokenKind() == scanlex.IDENTIFIER {

		pr = parse_decl_stmt(p, type_, defaultArg, allowinit, false, PARAM, "", false, ddaps)
	} else if p.currentTokenKind() == scanlex.OPEN_PAREN && checkFunctionType(p, 1) {
		err_ := p.errorExpection("Un Supported", helpers.UnSupported)
		p.addErr(err_)
	} else if p.currentTokenKind() == scanlex.IDENTIFIER || p.currentTokenKind() == scanlex.COMPOSITE_IDENTIFER || p.currentTokenKind() == scanlex.BUILT_IN_TYPE {
		pr = parse_decl_stmt(p, type_, defaultArg, allowinit, true, PARAM, "", false, ddaps)
	} else {
		err_ := p.errorExpection("Un Supported", helpers.UnSupported)
		p.addErr(err_)
	}
	init := false
	thunkArgs := false
	optionalArgs := false
	if !fnType {
		init = pr.Node.(ast.SymbolDeclStmt).IsInitialize()
		thunkArgs = pr.Node.(ast.SymbolDeclStmt).IsThunk()
		optionalArgs = pr.Node.(ast.SymbolDeclStmt).IsOptional()
	}
	pr.OtherDetails = map[string]any{
		"NamedArgs":    NamedArgs,
		"DefaultArgs":  init,
		"ThunkArgs":    thunkArgs,
		"OptionalArgs": optionalArgs,
		"VarArgs":      varArgs,
	}
	return pr
}

// parse_fn_fn_results parses one function return entry from the `->(...)`
// clause.
//
// Feature examples:
//
//	fun sum(a co.lang.int, b co.lang.int)->(co.lang.int) = { ... }
//	fun split(name co.lang.string)->(first co.lang.string, last co.lang.string) = { ... }
//	fun factory()->((co.lang.int)->(co.lang.string)) = { ... }
func parse_fn_fn_results(p *parser, type_ string, defaultArg bool, allowinit bool, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	var pr ParseResult
	if p.currentTokenKind() == scanlex.IDENTIFIER && p.nextTokenSafe(1).Value == "(" && checkFunctionType(p, 2) {
		tk := p.advance()

		pr = parse_fn_gen_declaration(p, FUNTYPE, true, true, true, tk.Value, ddaps)
		otherDetail := map[string]any{
			"FunParam": true,
		}
		pr.OtherDetails = otherDetail
	} else {
		pr = parse_decl_stmt(p, type_, defaultArg, allowinit, false, RESULT, "", false, ddaps)
	}
	return pr
}

// parse_fn_params_and_body parses the full callable signature and the optional
// implementation body.
//
// Feature examples:
//
//	fun add(a co.lang.int, b co.lang.int)->(co.lang.int) = { this = a + b; }
//	fun map(a co.lang.int)(b co.lang.int)->(co.lang.int) = { this = a + b; }
//	fun trace(msg co.lang.string)->() =>> logger.info(msg) =>> sink.write(msg);
//
// The parser supports curried parameter groups, named/default/optional params,
// function-typed params and results, normal block bodies, and delegate chains
// joined with `=>>`.
func parse_fn_params_and_body(p *parser, noBody bool, onlyParamTypes bool, scope string, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ([][]ast.Parameter, []ast.Returns, []ast.Stmt, bool, bool, []helpers.ErrorInterface) {
	defer p.traceCurrent()()

	var functionParams [][]ast.Parameter
	/*
	 * fun () name (params)(params)->(retus)={}
	 *
	 * function can return in the following way
	 * this.return a,b,.... can be anywhere
	 *     or
	 * this = a,b,..... as last statement otherwise assigning will not return will continue
	 *     or
	 *  a,b,.....  as last statement otherwise will not return will continue after this
	 */

	/*
	 * Default params is a co.lang.int = 10
	 */
	_, err_ := p.expect(scanlex.OPEN_PAREN)
	p.addErr(err_)
	defaultArg := false
	row := 0
	col := 0
	for {
		col = 0
		fnParams := make([]ast.Parameter, 0)
		for p.hasTokens() && p.currentTokenKind() != scanlex.CLOSE_PAREN {
			var tr ParseResult
			// Feature example:
			//   fun add(a co.lang.int, b co.lang.int)->(...)
			//
			// This is the common "named parameter with explicit type" path.
			if p.currentToken().Kind == scanlex.IDENTIFIER && (p.nextTokenSafe(1).Kind == scanlex.BUILT_IN_TYPE || p.nextTokenSafe(1).Kind == scanlex.IDENTIFIER) {
				tr = parse_fn_fn_params(p, "param", defaultArg, true, false, false, ddaps)

				// Feature example:
				//   fun run(fn (co.lang.int)->(co.lang.int))->(...)
				//
				// Function-typed parameters reuse function parsing in a no-body mode.
			} else if p.currentToken().Kind == scanlex.IDENTIFIER && (p.nextTokenSafe(1).Value == "(" && checkFunctionType(p, 2)) {
				tr = parse_fn_fn_params(p, "param", defaultArg, false, false, false, ddaps)
			} else if noBody && onlyParamTypes {
				// Feature example:
				//   someDelegate co.lang.delegate = (co.lang.int, co.lang.string) -> ()
				//
				// Delegate/function-type parsing can accept anonymous parameter types.
				tr = parse_fn_fn_params(p, "param", defaultArg, true, false, true, ddaps)
				//type_, errInt_ = parse_type(p, defalt_bp)

			}
			var param ast.Parameter

			syb := tr.Node.(ast.SymbolDeclStmt)
			othDet := tr.OtherDetails
			//need to retrieve default values
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
					if v, okk := val.(bool); okk {
						if v {
							whatType = string(symboltable.S_FunctionSymbol)
						}
					}
				}
			}

			param = ast.Parameter{
				SymbolDeclStmt: syb,
				Optional:       false,
				HasDefault:     defaultArg,
				Scope:          scope,
				VarArgs:        false,
				Default:        defValue,
				Name_:          nam,
				OnlyType:       onlyParamTypes,
				UseVarDecl:     onlyParamTypes,
				WhatType:       whatType,
				DefaultArgs:    othDet["DefaultArgs"].(bool),
				OptionalArgs:   othDet["OptionalArgs"].(bool),
				NamedArgs:      othDet["NamedArgs"].(bool),
				ThunkArgs:      othDet["ThunkArgs"].(bool),
				Variadic:       othDet["VarArgs"].(bool),
			}

			if !noBody && !onlyParamTypes {
				defaultArg = syb.IsInitialize()
				arg := ast.Argument{}
				syb.SetInner(true)
				param.Argument_ = &arg

			} else {

				param.MappedParam = "__co_intern_" + fmt.Sprint(row) + "_" + fmt.Sprint(col)

				symb := symboltable.Symbol{
					SymbolDet: &symboltable.VarSymbol{
						VariableDetails: symboltable.VariableDetails{
							SymbolDetails: symboltable.SymbolDetails{
								IsInternal_: true,
							}},
					},
				}
				param.Symb = &symb
			}
			fnParams = append(fnParams, param)

			//updateContext(p, param, symboltable.Parameter, false)

			if !p.currentToken().IsOneOfMany(scanlex.CLOSE_PAREN, scanlex.EOF) {
				_, err_ := p.expect(scanlex.COMMA)
				col = col + 1
				p.addErr(err_)

			}
		}
		functionParams = append(functionParams, fnParams)
		if p.nextTokenSafe(1).Kind != scanlex.OPEN_PAREN {
			break
		}

		_, err_ = p.expect(scanlex.CLOSE_PAREN)
		p.addErr(err_)
		row = row + 1
		_, err_ = p.expect(scanlex.OPEN_PAREN)
		if err_ != nil {
			p.addErr(err_)
		}
	}

	_, err_ = p.expect(scanlex.CLOSE_PAREN)
	p.addErr(err_)
	returnType := make([]ast.Returns, 0)
	var errs []helpers.ErrorInterface
	_, err_ = p.expect(scanlex.ARROW)
	if err_ != nil {
		p.addErr(err_)
	}
	_, err_ = p.expect(scanlex.OPEN_PAREN)
	p.addErr(err_)
	onlyType := false
	firstret := true
	hasResults := false
	for p.hasTokens() && p.currentTokenKind() != scanlex.CLOSE_PAREN {
		hasResults = true
		if p.currentTokenKind() == scanlex.IDENTIFIER {
			// Feature example:
			//   fun split(name co.lang.string)->(first co.lang.string, last co.lang.string)
			//
			// Named return values are parsed as declarations and cannot be mixed
			// with plain type-only results in the same return clause.
			if onlyType {
				err_ := p.errorObj(nil, "Cannot mix return results named and un named")
				p.addErr(err_)
			}
			if firstret {
				firstret = false
			}
			onlyType = false
			tr := parse_fn_fn_results(p, "result", false, false, ddaps)
			result := ast.Returns{
				SymbolDeclStmt: tr.Node.(ast.SymbolDeclStmt),
				OnlyType:       false,
				Symb: &symboltable.Symbol{
					SymbolDetails: symboltable.SymbolDetails{
						SymbolType_: string(symboltable.S_StatmentSymbol),
					},
				},
			}

			returnType = append(returnType, result)

			//updateContext(p, result, symboltable.Return, false)

			if !p.currentToken().IsOneOfMany(scanlex.CLOSE_PAREN, scanlex.EOF) {
				_, err_ = p.expect(scanlex.COMMA)
				p.addErr(err_)
				if p.currentTokenKind() != scanlex.IDENTIFIER {
					err_ = p.errorExpection("Cannot have mixed return types", helpers.InvalidSyntax)
					p.addErr(err_)
				}
			}
		} else {
			// Feature examples:
			//   fun id(a co.lang.int)->(co.lang.int)
			//   fun wrap()->((co.lang.int)->(co.lang.int))
			//
			// Unnamed results can be plain types or nested function types.
			if !firstret && !onlyType {
				err_ := p.errorObj(nil, "Cannot mix return results named and un named")
				p.addErr(err_)

			} else if firstret {
				firstret = false

			}
			onlyType = true
			var result ast.Returns
			if p.currentToken().Value == "(" && p.currentTokenKind() == scanlex.OPEN_PAREN {
				retType, _ := parse_fn_type(p, defalt_bp, ddaps)
				result = ast.Returns{
					OnlyType: true,
					Type_:    retType,
					WhatType: "FUN",
					Symb: &symboltable.Symbol{
						SymbolDetails: symboltable.SymbolDetails{
							SymbolType_: string(symboltable.S_StatmentSymbol),
						},
					},
				}
				returnType = append(returnType, result)

			} else {
				retType, retErrs := parse_type(p, defalt_bp, ddaps)
				p.addErr(retErrs)
				result = ast.Returns{
					OnlyType: true,
					Type_:    retType,
					WhatType: "VAR",
					Symb: &symboltable.Symbol{
						SymbolDetails: symboltable.SymbolDetails{
							SymbolType_: string(symboltable.S_StatmentSymbol),
						},
					},
				}

				returnType = append(returnType, result)
			}
			//updateContext(p, result, symboltable.Return, false)

			if !p.currentToken().IsOneOfMany(scanlex.CLOSE_PAREN, scanlex.EOF) {
				_, err_ = p.expect(scanlex.COMMA)
				p.addErr(err_)
				if p.currentTokenKind() == scanlex.IDENTIFIER {
					err_ = p.errorExpection("Cannot have mixed return types", helpers.InvalidSyntax)
					p.addErr(err_)
				}
			}

		}
		firstret = false
	}
	_, err_ = p.expect(scanlex.CLOSE_PAREN)
	p.addErr(err_)
	if !noBody {
		asExpr := false
		// Handle =>> (delegation/chaining) as alternative to = { body }
		//
		// Feature example:
		//   fun trace(msg co.lang.string)->() =>> logger.info(msg) =>> sink.write(msg);
		if p.hasTokens() && p.currentTokenKind() == scanlex.EQGTGT {
			p.advance() // consume =>>
			asExpr = true
			delegateChain := []ast.Stmt{}
			for {
				expr := parse_expr(p, defalt_bp, ddaps)
				if expr.Node != nil {
					delegateChain = append(delegateChain, ast.ExpressionStmt{Expression: expr.Node.(ast.Expr)})
				}
				if p.hasTokens() && p.currentTokenKind() == scanlex.EQGTGT {
					p.advance() // consume chained =>>
				} else {
					break
				}
			}
			if p.hasTokens() && p.currentTokenKind() == scanlex.SEMI_COLON {
				p.advance()
			}
			errs = []helpers.ErrorInterface{}
			return functionParams, returnType, delegateChain, asExpr, true, errs
		}
		// Feature example:
		//   fun add(a co.lang.int, b co.lang.int)->(co.lang.int) = {
		//       this = a + b;
		//   }
		//
		// Standard function bodies are parsed as blocks after `=`.
		_, err_ = p.expect(scanlex.ASSIGNMENT)
		p.addErr(err_)
		asExpr = true
		p.IsFunction = true
		tm := parse_block_stmt_generic(p, hasResults, true, ddaps)
		p.IsFunction = false
		functionBody := ast.ExpectStmt[ast.BlockStmt](tm.Node.(ast.Stmt)).Body
		errs = []helpers.ErrorInterface{}
		return functionParams, returnType, functionBody, asExpr, false, errs
	}
	return functionParams, returnType, nil, false, false, errs
}

// parse_delegate_stmt parses a delegate declaration.
//
// Feature example:
//
//	@co.dap.delegate
//	someDelegate co.lang.delegate = (a co.lang.int, b co.lang.int) -> (co.lang.int);
//
// The current token on entry is the delegate name. Unlike a function
// declaration, delegates do not have a body; they store a function type.
func parse_delegate_stmt(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	pr := ParseResult{}
	var delgName string
	var typ_ ast.Stmt
	flag := isDelegateDecl(p, -1, ddaps)
	if flag {
		tk, err_ := p.expect(scanlex.IDENTIFIER)
		p.addErr(err_)
		delgName = tk.Value

		// Feature example:
		//   someDelegate co.lang.delegate = ...
		//
		// After the delegate name we must consume the delegate kind marker before
		// reading the function-type payload.
		if ok, _ := checkKindsandTypes(p, KIND_, 0, "co.lang.delegate"); ok {
			p.advance()
		} else {
			err_ := p.errorExpection("co.lang.delegate expected but found "+p.currentToken().Value, helpers.InvalidSyntax)
			p.addErr(err_)
		}

		_, err_ = p.expect(scanlex.ASSIGNMENT)
		p.addErr(err_)
		typ, errs := parse_fn_type(p, defalt_bp, ddaps)
		typ_ = ast.TypeStmt{
			Type_: typ,
		}
		p.addErr(errs)
		_, err_ = p.expectAny(scanlex.SEMI_COLON, scanlex.COMMA)
		p.addErr(err_)

	}
	symb := symboltable.DelegateSymbol{
		Name: delgName,
		SymbolDetails: symboltable.SymbolDetails{
			SymbolType_: string(symboltable.S_DelegateSymbol),
		},
	}
	nd := ast.DelegateStmt{
		Type_: typ_,
		Symb:  &symb,
	}
	pr.Node = nd
	return pr
}

// parse_fn_declaration parses a regular function declaration entry point and
// forwards to the shared function generator.
//
// Feature examples:
//
//	fun add(a co.lang.int, b co.lang.int)->(co.lang.int) = { ... }
//	generic map(T)(items []T)->([]T) = { ... }
func parse_fn_declaration(p *parser, stmtK StmtKind, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()
	k := ""

	return parse_fn_gen_declaration(p, stmtK, false, false, false, k, ddaps)

}

// parse_fn_gen_declaration parses the shared callable grammar used by function
// declarations, function types, templates, macros, and delegates.
//
// Feature examples:
//
//	fun add(a co.lang.int)->(co.lang.int) = { ... }
//	(receiver Employee) fullName()->(co.lang.string) = { ... }
//	(co.lang.int, co.lang.int)->(co.lang.int)
//
// The same routine drives both declarations with bodies and "type only" forms.
func parse_fn_gen_declaration(p *parser, stmtK StmtKind, noBody bool, paramOnlyTypes bool, asFuncParam bool, funName string, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	pr := ParseResult{
		Node:   nil,
		Errors: []helpers.ErrorInterface{},
	}
	v := p.IsFunction
	p.IsFunction = true

	//Normal functions for function parameters or types the Statement kind must be FUNTYPE

	scope := "LEXICAL"
	switch stmtK {
	case TEMPLATE:
		scope = "DYNAMIC"
	case MACRO:
		scope = "MIXED"
	case FUN, FUNCTION, GENERIC:
		scope = "LEXICAL"
	default:
		scope = "STATIC"
	}
	//fun

	functionName := "OnlyType"
	var actCtxId *symboltable.Context
	var actSymbId *symboltable.SymbolTable
	dddaps := map[scanlex.DirectiveKind][]ast.Stmt{
		scanlex.ANNOTATION: ddaps[scanlex.ANNOTATION],
		scanlex.DECORATOR:  ddaps[scanlex.DECORATOR],
	}
	asso_reci := ast.FunctionReceiver{}
	hasReceiver := false
	if p.currentTokenKind() == scanlex.OPEN_PAREN {
		// Feature example:
		//   (emp Employee) fullName()->(co.lang.string) = { ... }
		//
		// Receiver parsing reuses variable declaration logic so methods and
		// associated functions share the same declaration shape as parameters.

		p.advance()
		if p.currentTokenKind() == scanlex.IDENTIFIER || p.currentTokenKind() == scanlex.COMPOSITE_IDENTIFER {
			//add special directives related to variable if any to dddaps

			aspr := parse_decl_stmt(p, "param", false, false, false, PARAM, "", false, dddaps)
			asso_reci.SymbolStmt = aspr.Node.(ast.SymbolDeclStmt)
			hasReceiver = true
		} else {
			_, err := p.expectAny(scanlex.IDENTIFIER, scanlex.COMPOSITE_IDENTIFER)
			p.addErr(err)
		}

		_, err_ := p.expect(scanlex.CLOSE_PAREN)
		p.addErr(err_)
	}

	if !noBody {

		functionName_p, err_ := p.expect(scanlex.IDENTIFIER)
		p.addErr(err_)
		functionName = functionName_p.Value
	} else {
		functionName = funName
	}
	// checking the function name is restricted one or not
	/*
				          NamedArgs
						  Optional Params
						  Default Params
						  Function type params or returns
						  Thunks
				          VarArgs / Variadic
		            might be declared with same name
	*/
	if !p.Context_.IsSymbolNameReusable(*p.Fs, functionName) {
		err_ := p.errorExpection("This function name cannot be used for many reasons please go through documents", helpers.InvalidSyntax)
		p.addErr(err_)

	}
	if !strings.HasPrefix(functionName, "__co_internal_fun_") {
		// keeping back up of current context id and symbol table id to restore backup
		actCtxId = p.Context_
		actSymbId = p.SymbolTable_
		//Creating new context for type specfic
		p.Context_, p.SymbolTable_ = CreateNewContext(actCtxId.Id, string(symboltable.S_FunctionSymbol))
		p.Context_.ContextType_ = string(symboltable.S_FunctionSymbol)
		actCtxId.ChildCtxIds = append(actCtxId.ChildCtxIds, p.Context_.Id)
		p.Context_.ParentCtxSymbolTableId = actSymbId.Id
		p.Fs.AddContext(p.Context_)
		p.Fs.AddSymbolTable(p.SymbolTable_)

	}

	functionParams, returnType, functionBody, asExpr, isDelegate, errs := parse_fn_params_and_body(p, noBody, paramOnlyTypes, scope, ddaps)
	p.addErr(errs)
	isCurried := false
	if len(functionParams) > 1 {
		isCurried = true
	}
	hasClosure := false
	for _, rtype := range returnType {
		if rtype.WhatType == "FUN" {
			hasClosure = true
			break
		}
	}

	if hasClosure && len(returnType) > 1 {
		err_ := p.errorExpection("Closure/function return type functions++++ currently don't support more than one return", helpers.InvalidSyntax)
		p.addErr(err_)
	}

	//check parameters contains any restrict things
	/*
		    like
			   Thunk
			   function type
			   varargs
			   optional
			   default
			   Named

	*/
	RestrictedToOverload := false
	if hasClosure {
		RestrictedToOverload = true
	} else if p.ParentIsFunc {
		RestrictedToOverload = true
	}

	for _, params := range functionParams {
		for _, param := range params {
			if param.NamedArgs || param.DefaultArgs || param.OptionalArgs || param.ThunkArgs {
				RestrictedToOverload = true
				break
			} else if param.WhatType == "FUN" {
				RestrictedToOverload = true
				break
			}
		}
	}

	symb := symboltable.FunctionSymbol{
		Curried:              isCurried,
		Closure:              hasClosure,
		InnerFunction:        false,
		IsBody:               !noBody,
		OnlyParamTypes:       paramOnlyTypes,
		AsFunctionParam:      asFuncParam,
		ParentIsFunc:         p.ParentIsFunc,
		Variadic:             false,
		VariantTypeArgs:      false,
		OptionalArgs:         false,
		DefaultParams:        false,
		NamedParams:          false,
		FunTypeParRet:        false,
		RestrictedToOverload: RestrictedToOverload,
		IsInner:              p.ParentIsFunc,
		Associated:           hasReceiver,
		Delegate:             isDelegate,
		SymbolDetails: symboltable.SymbolDetails{
			SymbolType_: string(symboltable.S_FunctionSymbol),
		},
	}
	fn := ast.FunctionDeclarationStmt{
		Parameters:         functionParams,
		ReturnType:         returnType,
		Body:               functionBody,
		Name:               functionName,
		Scope:              scope,
		AssociatedReceiver: &asso_reci,
		AsExpr:             asExpr,
		Symb:               &symb,
	}
	fn.SetDap(ddaps)
	_, typ := fn.GetActType()
	t := strings.Split(typ, "::")
	t1 := strings.TrimPrefix(t[0], "[")
	fn.ParamNames = t1
	t1 = strings.TrimSuffix(t[1], "]")
	fn.ResultNames = t1

	lastParam := fn.ResultNames + "_" + fn.ParamNames
	lastParam = strings.ReplaceAll(lastParam, "co.lang.", "")
	lastParam = strings.ReplaceAll(lastParam, ".", "_")
	lastParam, _ = helpers.GenID("fun", lastParam)

	lastParamName := "co_internal_" + helpers.GenUnique(4)
	fn.IntOverLdParamName = lastParamName
	fn.IntOverLdParam = lastParam

	pr.Node = fn
	if !strings.HasPrefix(functionName, "__co_internal_fun_") {
		//before returning restoring current cotnext id to the original context id and symbol table id backup while entering
		p.Context_ = actCtxId
		p.SymbolTable_ = actSymbId
	}
	p.IsFunction = v
	updateContext(p, pr.Node, false, false)
	return pr
}
