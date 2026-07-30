package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// function-object-declaration — section 7.
//
//	function-object-declaration = annotations, declaration-name,
//	                              "co.lang.function", "=",
//	                              function-object-binding
//	function-object-binding     = anonymous-function-expression,
//	                              body-closure-guard
//	                            | non-anonymous-function-expression,
//	                              statement-end
//
// This declaration is the clearest illustration of DECISION-SYN-007, because the same kind
// token takes two bindings with DIFFERENT terminators
// (docs/language-ref.md, "Other ways to declare closures/function objects"):
//
//	someFArg co.lang.function = (a co.lang.int)->(co.lang.int) = {
//	    this.return a * 2;
//	}                                                  a BODY: ends at "}", no ";"
//
//	oObj co.lang.function = add;                       an EXPRESSION: ends at ";"
//
// The first is a direct anonymous function used as the declaration's inline body; the
// second binds an existing callable. Ordered choice alone cannot separate them, so the
// anonymous-function reading is selected by the affirmative guard
// startsAnonymousFunction and the expression reading is everything else.

// parseFunctionObjectDeclaration parses the function-object-declaration production.
func (p *parser) parseFunctionObjectDeclaration(declName name, annotations annotationSet) ast.Stmt {
	// A forward declaration of the function kind ends at ";" with no binding.
	if p.at(scanlex.SEMI_COLON) {
		p.advance()
		symb := p.functionSymbol(declName.Scanned)
		symb.FunctionObject = true
		symb.IsBody = false
		return ast.FunctionDeclarationStmt{
			Name:  declName.Scanned,
			Dapst: annotations.list(),
			Symb:  symb,
		}
	}

	p.expectOp("=", "before a function-object binding")

	if p.startsAnonymousFunction() {
		return p.parseFunctionObjectInlineBody(declName, annotations)
	}
	return p.parseFunctionObjectExpressionBinding(declName, annotations)
}

// parseFunctionObjectInlineBody parses the anonymous-function-expression alternative of
// function-object-binding.
//
// The anonymous function is the declaration's inline body, so it ends at its closing brace
// and body-closure-guard rejects a following ";".
func (p *parser) parseFunctionObjectInlineBody(declName name, annotations annotationSet) ast.Stmt {
	fn := p.parseAnonymousFunctionExpression()
	p.bodyClosureGuard("a function-object body")

	fnExpr, ok := fn.(ast.FunctionExpr)
	if !ok {
		p.failf(p.cur(), "expected an anonymous function as the body of %q", declName.Logical)
	}

	symb := p.functionSymbol(declName.Scanned)
	symb.FunctionObject = true
	symb.Closure = true
	symb.IsBody = true

	return ast.FunctionDeclarationStmt{
		Parameters: [][]ast.Parameter{fnExpr.Parameters},
		Name:       declName.Scanned,
		Body:       fnExpr.Body,
		ReturnType: fnExpr.ReturnType,
		Dapst:      annotations.list(),
		Symb:       symb,
	}
}

// parseFunctionObjectExpressionBinding parses the non-anonymous-function-expression
// alternative of function-object-binding.
//
// The expression is an ordinary binding, so the statement ends with ";".
func (p *parser) parseFunctionObjectExpressionBinding(declName name, annotations annotationSet) ast.Stmt {
	target := p.parseExpression()
	p.statementEnd("a function-object binding")

	symb := p.functionSymbol(declName.Scanned)
	symb.FunctionObject = true
	symb.IsBody = false

	return ast.FunctionDeclarationStmt{
		Name: declName.Scanned,
		Body: []ast.Stmt{
			ast.ExpressionStmt{Expression: target, Symb: p.stmtSymbol("function-object-binding")},
		},
		Dapst: annotations.list(),
		Symb:  symb,
	}
}

// type-constructor-primary — section 7.
//
//	type-constructor-primary = annotations, function-name, parameter-list,
//	                           { parameter-list }, return-type-clause,
//	                           type-constructor-binding
//	type-constructor-binding = function-definition
//	                         | function-delegation
//	                         | "=", type-expression, statement-end
//	                         | "=", non-block-expression, statement-end
//	                         | statement-end
//
// A type constructor is a function that returns a TYPE, so its binding may be a type
// expression. DECISION-TYP-002 puts the type-expression alternative BEFORE the expression
// alternative, which is what makes
//
//	Vector(n co.lang.int)->(co.lang.dependentType) = co.lang.int->([n]);
//
// parse as a type rather than as an unparseable expression: `co.lang.int->([n])` is a valid
// type but not a valid value expression.

// parseTypeConstructorPrimary parses the type-constructor-primary production.
func (p *parser) parseTypeConstructorPrimary(annotations annotationSet) ast.Stmt {
	ctorName := p.parseFunctionName("as a type constructor name")
	paramLists := p.parseParameterLists()
	results := p.parseReturnTypeClause()

	decl := ast.FunctionDeclarationStmt{
		Parameters: paramLists,
		Name:       ctorName.Scanned,
		ReturnType: results,
		Dapst:      annotations.list(),
		Symb:       p.functionSymbol(ctorName.Scanned),
	}
	p.applyFunctionFlags(&decl, annotations)

	return p.parseTypeConstructorBinding(ctorName, decl, annotations)
}

// parseTypeConstructorBinding parses the type-constructor-binding production.
func (p *parser) parseTypeConstructorBinding(ctorName name, decl ast.FunctionDeclarationStmt, annotations annotationSet) ast.Stmt {
	switch {
	// function-definition: a block body.
	case p.at(scanlex.OPEN_CURLY):
		return p.finishFunctionDefinition(decl)

	// function-delegation.
	case p.atOp("=>"), p.atOp("=>>"):
		return p.parseFunctionDelegation(decl)

	case p.atOp("="):
		p.advance()

		// function-definition with the optional "=" present.
		if p.startsDirectBody() && !p.looksLikeMapLiteral() {
			return p.finishFunctionDefinition(decl)
		}

		// DECISION-TYP-002: the type-expression reading is tried first.
		if bound, ok := p.tryTypeConstructorTypeBinding(ctorName, decl, annotations); ok {
			return bound
		}

		// Otherwise an ordinary expression binding.
		return p.parseFunctionAliasBinding(decl)

	// A forward declaration.
	default:
		p.statementEnd("a type constructor forward declaration")
		decl.Symb.IsBody = false
		return decl
	}
}

// tryTypeConstructorTypeBinding attempts the `"=", type-expression, statement-end`
// alternative of type-constructor-binding.
//
// The attempt is speculative and is accepted only when the type expression runs right up to
// the statement terminator, so a token sequence that merely starts like a type does not
// hijack an expression binding.
func (p *parser) tryTypeConstructorTypeBinding(ctorName name, decl ast.FunctionDeclarationStmt, annotations annotationSet) (ast.Stmt, bool) {
	var bound ast.Stmt

	matched := p.speculate(func() bool {
		t := p.parseTypeExpression()
		if !p.at(scanlex.SEMI_COLON) {
			return false
		}
		p.advance()

		symb := p.typeSymbol(ctorName.Scanned)
		symb.DependentType = true
		symb.IsGenericType = true

		bound = ast.TypeDeclarationStmt{
			Name:     ctorName.Scanned,
			Type_:    t.Node,
			Kind:     "co.lang.dependentType",
			SubType_: "TYPE_CONSTRUCTOR",
			Typetype: "UDT",
			SDapst:   annotations.list(),
			Symb:     symb,
		}
		return true
	})

	return bound, matched
}

// annotated-function-primary — section 7.
//
//	annotated-function-primary = one-or-more-annotations, function-declaration
//
// This is a function declaration promoted to a primary declaration by its annotations, so
// that a macro, a template, an operator, an extension or an indexer can be the single
// declaration of a package source file. The annotation decides which wrapper node the
// declaration becomes.

// parseAnnotatedFunctionPrimary parses the annotated-function-primary production.
func (p *parser) parseAnnotatedFunctionPrimary(annotations annotationSet) ast.Stmt {
	decl := p.parseFunctionDeclaration(annotations)

	fn, ok := decl.(ast.FunctionDeclarationStmt)
	if !ok {
		return decl
	}
	return p.wrapAnnotatedFunction(fn, annotations)
}

// wrapAnnotatedFunction wraps a function declaration in the node its annotation selects.
//
// FoLang declares macros, templates, operators, extensions, indexers and matchers as
// annotated functions rather than with dedicated keywords, so this is where the annotation
// becomes a distinct node type.
func (p *parser) wrapAnnotatedFunction(fn ast.FunctionDeclarationStmt, annotations annotationSet) ast.Stmt {
	switch {
	case annotations.has("@co.dap.macro"):
		fn.Symb.Type_ = "macro"
		return ast.MacroStmt{
			FunctionDeclarationStmt: fn,
			Type_:                   "macro",
			IsExportable:            annotations.has("@co.dap.export"),
		}

	case annotations.has("@co.dap.template"):
		fn.Symb.Type_ = "template"
		return ast.TemplateStmt{FunctionDeclarationStmt: fn, Type_: "template"}

	case annotations.has("@co.dap.operator"):
		fn.Symb.IsOperator = true
		p.registerDeclaredOperator(fn, annotations)
		return ast.OperatorStmt{FunctionDeclarationStmt: fn, Type_: "operator"}

	case annotations.has("@co.dap.extension"):
		return ast.ExtensionStmt{
			FunctionDeclarationStmt: fn,
			ForType:                 annotations.optionString("@co.dap.extension", "fortype"),
			What:                    annotations.optionString("@co.dap.extension", "what"),
		}

	case annotations.has("@co.dap.indexer"):
		return ast.IndexerStmt{FunctionDeclarationStmt: fn, Type_: "indexer"}

	case annotations.has("@co.dap.matcher"):
		return ast.MatcherStmt{FunctionDeclarationStmt: fn, Type_: "matcher"}

	case annotations.has("@co.dap.generic"):
		fn.Symb.IsGeneric = true
		return ast.GenerricFun{
			FunctionDeclarationStmt: fn,
			Type_:                   "generic",
			Generic:                 *p.genericDetails(fn.Name, nil),
		}

	case annotations.has("@co.dap.annotation"),
		annotations.has("@co.dap.directive"),
		annotations.has("@co.dap.pragma"),
		annotations.has("@co.dap.decorator"):
		return ast.DDapStmt{FunctionDeclarationStmt: fn, Type_: "ddap"}
	}

	return fn
}

// registerDeclaredOperator adds a user-defined operator to the Pratt engine's registry.
//
// DECISION-EXT-001 requires a new symbol to declare its fixity, numeric precedence and
// associativity before it can be used; an overload of a built-in symbol keeps built-in
// precedence and so is not registered. The declaration carries those in its annotation:
//
//	@co.dap.operator(symbol="<+>", fixity=infix, precedence=65, associativity=left)
func (p *parser) registerDeclaredOperator(fn ast.FunctionDeclarationStmt, annotations annotationSet) {
	symbol := annotations.optionString("@co.dap.operator", "symbol")
	if symbol == "" {
		// Without a declared symbol this is an overload of a built-in operator,
		// which retains built-in precedence and needs no registration.
		return
	}

	switch annotations.optionString("@co.dap.operator", "fixity") {
	case "prefix":
		p.ops.registerPrefix(symbol)
	case "postfix":
		p.ops.registerPostfix(symbol)
	default:
		precedence := 50
		if v, ok := annotations.option("@co.dap.operator", "precedence"); ok {
			if i, isInt := v.(int64); isInt {
				precedence = int(i)
			}
		}
		assoc := leftAssoc
		if annotations.optionString("@co.dap.operator", "associativity") == "right" {
			assoc = rightAssoc
		}
		p.ops.registerInfix(symbol, precedence, assoc)
	}
}
