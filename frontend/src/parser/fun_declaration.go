package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// function-declaration and function-binding — section 8.
//
//	function-declaration = annotations, [ receiver-clause ], function-name,
//	                       parameter-list, { parameter-list },
//	                       [ return-type-clause ], function-binding
//	function-binding     = function-definition
//	                     | function-delegation
//	                     | function-alias-binding
//	                     | statement-end
//	function-definition  = [ "=" ], block, body-closure-guard
//	function-delegation  = ( "=>" | "=>>" ), expression,
//	                       { "=>>", expression }, statement-end
//	function-alias-binding = "=", non-block-expression, statement-end
//
// The four bindings are the four ways a function can be given a body:
//
//	add(a T)->(T) = { … }                              definition
//	add(a T)->(T) { … }                                definition, "=" omitted
//	fetch(id S)->(E) =>> mod.get(this, id);            delegation chain
//	shorthand(a T)->(T) = someOtherFunction;           alias
//	forwardDeclared(a T)->(T);                         forward declaration
//
// DECISION-FUN-001 makes the "=" before a block body optional, so the first two forms
// are equivalent.
//
// The definition form ends at its closing brace and takes NO semicolon, while every
// other form ends with one (DECISION-SYN-006). That is the distinction
// body-closure-guard enforces, and getting it wrong is the most common way to
// mis-parse a declaration body as an expression.

// parseFunctionDeclaration parses the function-declaration production.
func (p *parser) parseFunctionDeclaration(annotations annotationSet) ast.Stmt {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	var receiver *ast.FunctionReceiver
	if p.atReceiverClause() {
		receiver = p.parseReceiverClause()
	}

	funcName := p.parseFunctionName("as a function name")
	return p.continueFunctionDeclarationWithReceiver(funcName, receiver, annotations)
}

// continueFunctionDeclaration parses a function-declaration whose name has already been
// consumed.
//
// The primary-declaration dispatcher has to read a declaration's name before it can tell
// which kind of declaration it is, so by the time a function is identified the name is gone.
// This entry point lets that caller continue from the parameter list.
func (p *parser) continueFunctionDeclaration(funcName name, annotations annotationSet) ast.Stmt {
	return p.continueFunctionDeclarationWithReceiver(funcName, nil, annotations)
}

// continueFunctionDeclarationWithReceiver parses the rest of a function-declaration after
// its optional receiver clause and its name.
func (p *parser) continueFunctionDeclarationWithReceiver(funcName name, receiver *ast.FunctionReceiver, annotations annotationSet) ast.Stmt {
	paramLists := p.parseParameterLists()

	var results []ast.Returns
	if p.at(scanlex.ARROW) {
		results = p.parseReturnTypeClause()
	}

	decl := ast.FunctionDeclarationStmt{
		Parameters:         paramLists,
		Name:               funcName.Scanned,
		ReturnType:         results,
		AssociatedReceiver: receiver,
		Dapst:              annotations.list(),
		Symb:               p.functionSymbol(funcName.Scanned),
	}
	p.applyFunctionFlags(&decl, annotations)

	return p.parseFunctionBinding(decl)
}

// parseFunctionBinding parses the function-binding production and attaches the result
// to decl.
func (p *parser) parseFunctionBinding(decl ast.FunctionDeclarationStmt) ast.Stmt {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	switch {
	// function-delegation: "=>" or "=>>" followed by a chain of expressions.
	case p.atOp("=>"), p.atOp("=>>"):
		return p.parseFunctionDelegation(decl)

	// function-definition with the optional "=" present. The brace must follow,
	// otherwise this is an alias binding.
	case p.atOp("=") && p.definitionFollowsAssign():
		p.advance() // "="
		return p.finishFunctionDefinition(decl)

	// function-alias-binding: "=" followed by an expression.
	case p.atOp("="):
		p.advance()
		return p.parseFunctionAliasBinding(decl)

	// function-definition with the "=" omitted (DECISION-FUN-001).
	case p.at(scanlex.OPEN_CURLY):
		return p.finishFunctionDefinition(decl)

	// A forward declaration: the binding is just the statement terminator
	// (docs/language-ref.md, "Functions forward declaration").
	default:
		p.statementEnd("a function forward declaration")
		decl.Symb.IsBody = false
		return decl
	}
}

// definitionFollowsAssign reports whether the "=" at the cursor introduces a block
// body rather than an alias expression.
//
// The two are distinguished by what follows: a "{" that is a body rather than a map
// literal means a definition. An anonymous function that is itself a direct inline
// body also counts, which is what makes the function-object form work.
func (p *parser) definitionFollowsAssign() bool {
	return p.lookaheadOnly(func() bool {
		p.advance() // "="
		return p.startsDirectBody() && !p.looksLikeMapLiteral()
	})
}

// finishFunctionDefinition parses the block body of a function-definition.
//
// The body ends at its closing brace and takes no semicolon, which
// body-closure-guard asserts.
func (p *parser) finishFunctionDefinition(decl ast.FunctionDeclarationStmt) ast.Stmt {
	body := p.parseBlock("a function body")
	p.bodyClosureGuard("a function body")

	decl.Body = statementsOf(body)
	decl.Symb.IsBody = true
	return decl
}

// parseFunctionDelegation parses the function-delegation production:
//
//	function-delegation = ( "=>" | "=>>" ), expression,
//	                      { "=>>", expression }, statement-end
//
// A delegating function forwards its work rather than computing it
// (docs/language-ref.md, "Function Chaining"):
//
//	fetchEmployee(empId co.lang.string)->(Employee)=>>empMod.getEmployee(this, empId);
//
// In a "=>>" chain each stage's result is available to the next through the "$1",
// "$2", … result bindings.
func (p *parser) parseFunctionDelegation(decl ast.FunctionDeclarationStmt) ast.Stmt {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	firstOp := p.advance() // "=>" or "=>>"

	stages := []ast.Stmt{
		ast.ExpressionStmt{
			Expression: p.parseExpression(),
			Symb:       p.stmtSymbol("delegation"),
		},
	}

	for p.atOp("=>>") {
		p.advance()
		stages = append(stages, ast.ExpressionStmt{
			Expression: p.parseExpression(),
			Symb:       p.stmtSymbol("delegation"),
		})
	}

	p.statementEnd("a function delegation")

	decl.Body = stages
	decl.Symb.IsBody = true
	decl.Symb.FunctionChain = firstOp.Value == "=>>" || len(stages) > 1
	decl.Symb.Delegate = true
	return decl
}

// parseFunctionAliasBinding parses the function-alias-binding production:
//
//	function-alias-binding = "=", non-block-expression, statement-end
//
// This binds the declared name to an existing callable rather than to a body.
func (p *parser) parseFunctionAliasBinding(decl ast.FunctionDeclarationStmt) ast.Stmt {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	target := p.parseExpression()
	p.statementEnd("a function alias binding")

	decl.Body = []ast.Stmt{
		ast.ExpressionStmt{Expression: target, Symb: p.stmtSymbol("alias")},
	}
	decl.Symb.IsBody = false
	decl.Symb.FunctionObject = true
	return decl
}

// parseFunctionSpecification parses the function-specification production:
//
//	function-specification = annotations, [ receiver-clause ], function-name,
//	                         parameter-list, { parameter-list },
//	                         [ return-type-clause ], statement-end
//
// A specification is a signature with no body. It is what an interface, a signature
// and a contract body are made of.
func (p *parser) parseFunctionSpecification(annotations annotationSet) ast.Stmt {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	var receiver *ast.FunctionReceiver
	if p.atReceiverClause() {
		receiver = p.parseReceiverClause()
	}

	funcName := p.parseFunctionName("as a function name")
	paramLists := p.parseParameterLists()

	var results []ast.Returns
	if p.at(scanlex.ARROW) {
		results = p.parseReturnTypeClause()
	}

	p.statementEnd("a function specification")

	decl := ast.FunctionDeclarationStmt{
		Parameters:         paramLists,
		Name:               funcName.Scanned,
		ReturnType:         results,
		AssociatedReceiver: receiver,
		Dapst:              annotations.list(),
		Symb:               p.functionSymbol(funcName.Scanned),
	}
	decl.Symb.IsBody = false
	decl.Symb.OnlyParamTypes = true
	p.applyFunctionFlags(&decl, annotations)
	return decl
}

// local-function-declaration — section 8.
//
//	local-function-declaration = annotations, function-name, parameter-list,
//	                             { parameter-list }, return-type-clause,
//	                             function-definition
//
// DECISION-SYN-003 requires BOTH a return-type clause and a block body for a function
// declared inside a block. That is what keeps `foo();` an expression statement rather
// than a forward declaration, and it is what admits the reference's inner-function
// form (docs/language-ref.md, "Inner Function"):
//
//	someother()->()={
//	    co.out.println(p);
//	}

// atLocalFunctionDeclaration reports whether the cursor begins a
// local-function-declaration.
//
// Both the return-type clause and the body are required, so the probe checks for the
// full `name ( … ) -> ( … ) [=] {` shape.
func (p *parser) atLocalFunctionDeclaration() bool {
	if !p.atIdentifier() && !p.atLifecycleName() {
		return false
	}
	return p.lookaheadOnly(func() bool {
		p.advance() // the name
		if !p.at(scanlex.OPEN_PAREN) {
			return false
		}
		for p.at(scanlex.OPEN_PAREN) {
			p.skipBalanced(scanlex.OPEN_PAREN, scanlex.CLOSE_PAREN)
		}
		// DECISION-SYN-003: the return-type clause is mandatory.
		if !p.at(scanlex.ARROW) {
			return false
		}
		p.advance()
		if !p.at(scanlex.OPEN_PAREN) {
			return false
		}
		p.skipBalanced(scanlex.OPEN_PAREN, scanlex.CLOSE_PAREN)
		p.acceptOp("=")
		return p.at(scanlex.OPEN_CURLY)
	})
}

// parseLocalFunctionDeclaration parses the local-function-declaration production.
func (p *parser) parseLocalFunctionDeclaration(annotations annotationSet) ast.Stmt {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	funcName := p.parseFunctionName("as a local function name")
	paramLists := p.parseParameterLists()
	results := p.parseReturnTypeClause()

	decl := ast.FunctionDeclarationStmt{
		Parameters: paramLists,
		Name:       funcName.Scanned,
		ReturnType: results,
		Scope:      "local",
		Dapst:      annotations.list(),
		Symb:       p.functionSymbol(funcName.Scanned),
	}
	decl.Symb.InnerFunction = true
	decl.Symb.IsInner = true
	p.applyFunctionFlags(&decl, annotations)

	p.acceptOp("=") // DECISION-FUN-001: optional before a block body.
	return p.finishFunctionDefinition(decl)
}

// applyFunctionFlags records on the symbol what a declaration's shape and annotations
// say about the function.
//
// The shape-derived flags are set from the parse itself: several parameter lists mean
// curried, and a parameter or result that is a function type means the function takes
// or returns functions. The rest come from annotations, which is how FoLang spells
// inline, lazy, native, visibility and the execution models.
func (p *parser) applyFunctionFlags(decl *ast.FunctionDeclarationStmt, annotations annotationSet) {
	symb := decl.Symb

	symb.Curried = len(decl.Parameters) > 1
	symb.Variadic = hasVariadicParameter(decl.Parameters)
	symb.NamedParams = hasNamedParameter(decl.Parameters)
	symb.OptionalArgs = hasOptionalParameter(decl.Parameters)
	symb.DefaultParams = hasDefaultParameter(decl.Parameters)
	symb.FWPF = takesFunctionParameter(decl.Parameters)
	symb.FWRF = returnsFunction(decl.ReturnType)
	symb.IsMethod = decl.AssociatedReceiver != nil
	symb.Associated = decl.AssociatedReceiver != nil

	// A curried function may not be variadic, and vice versa
	// (docs/language-ref.md, "Variadic Functions").
	if symb.Curried && symb.Variadic {
		p.reportf(p.cur(), "function %q cannot be both curried and variadic", logicalName(decl.Name))
	}

	for _, d := range annotations.all {
		switch d.Name {
		case "@co.dap.inline":
			symb.Inline = true
		case "@co.dap.lazy":
			symb.Lazy = true
		case "@co.dap.eager":
			symb.Eager = true
		case "@co.dap.native":
			symb.Native = true
		case "@co.dap.abstract":
			symb.Abstract = true
		case "@co.dap.virtual":
			symb.Virtual = true
		case "@co.dap.override":
			symb.Overrridden = true
		case "@co.dap.sealed":
			symb.Issealed = true
		case "@co.dap.public":
			symb.IsPublic = true
		case "@co.dap.private":
			symb.IsPrivate = true
		case "@co.dap.protected":
			symb.IsProtected = true
		case "@co.dap.package":
			symb.IsPackageScope = true
		case "@co.dap.export":
			symb.IsExportable = true
		case "@co.dap.static":
			symb.StaticMethod = true
		case "@co.dap.class", "@co.dap.method.class":
			symb.ClassMethod = true
		case "@co.dap.instance":
			symb.InstanceMethod = true
		case "@co.dap.object":
			symb.ObjectMethod = true
		case "@co.dap.dynamicscope":
			symb.DynamicScope = true
		case "@co.dap.lexicalscope":
			symb.LexicalStaticScope = true
		case "@co.dap.mixedscope":
			symb.MixedScope = true
		case "@co.dap.local", "@co.dap.nested", "@co.dap.inner":
			symb.InnerFunction = true
			symb.IsInner = true
		case "@co.dap.callback":
			symb.Callback = true
		case "@co.dap.async":
			symb.Async = true
		case "@co.dap.defer":
			symb.Defer = true
		case "@co.dap.generator":
			symb.Generator = true
		case "@co.dap.iterator":
			symb.Iterator = true
		case "@co.dap.coroutine":
			symb.Coroutine = true
		case "@co.dap.goroutine":
			symb.Goroutine = true
		case "@co.dap.concurrent":
			symb.Concurrent = true
		case "@co.dap.parallel":
			symb.Parallel = true
		case "@co.dap.thread":
			symb.Thread = true
		case "@co.dap.task":
			symb.Task = true
		case "@co.dap.fiber":
			symb.Fiber = true
		case "@co.dap.process":
			symb.Process = true
		case "@co.dap.spawn":
			symb.Spawn = true
		case "@co.dap.fork":
			symb.Fork = true
		case "@co.dap.exec":
			symb.Exec = true
		case "@co.dap.actor":
			symb.Actor = true
		case "@co.dap.event":
			symb.Event = true
		case "@co.dap.channel":
			symb.Channel = true
		case "@co.dap.promise":
			symb.Promise = true
		case "@co.dap.future":
			symb.Future = true
		case "@co.dap.operator":
			symb.IsOperator = true
		}
		symb.WhatisIt = append(symb.WhatisIt, d.Name)
	}
	decl.WhatisIt = symb.WhatisIt

	// A special function cannot be overloaded, used as a callback, or take part in
	// the execution models (docs/language-ref.md, "Some Restrictions on Special
	// Functions"). The flag records that so the semantic phase can enforce it.
	symb.RestrictedToOverload = symb.Curried || symb.Variadic || symb.NamedParams ||
		symb.OptionalArgs || symb.DefaultParams || symb.FWPF || symb.FWRF ||
		symb.DynamicScope || symb.MixedScope
}

// hasVariadicParameter reports whether any parameter list declares a variadic
// parameter.
func hasVariadicParameter(lists [][]ast.Parameter) bool {
	for _, list := range lists {
		for _, param := range list {
			if param.VarArgs {
				return true
			}
		}
	}
	return false
}

// hasNamedParameter reports whether any parameter is marked with "~".
func hasNamedParameter(lists [][]ast.Parameter) bool {
	for _, list := range lists {
		for _, param := range list {
			if param.NamedArgs {
				return true
			}
		}
	}
	return false
}

// hasOptionalParameter reports whether any parameter is marked with "?".
func hasOptionalParameter(lists [][]ast.Parameter) bool {
	for _, list := range lists {
		for _, param := range list {
			if param.Optional {
				return true
			}
		}
	}
	return false
}

// hasDefaultParameter reports whether any parameter carries a default value.
func hasDefaultParameter(lists [][]ast.Parameter) bool {
	for _, list := range lists {
		for _, param := range list {
			if param.HasDefault {
				return true
			}
		}
	}
	return false
}

// takesFunctionParameter reports whether any parameter's type is a function type,
// which makes this a higher-order function
// (docs/language-ref.md, "Functions Taking and Returning Functions").
func takesFunctionParameter(lists [][]ast.Parameter) bool {
	for _, list := range lists {
		for _, param := range list {
			if _, isFn := param.Type_.(ast.FunctionType); isFn {
				return true
			}
		}
	}
	return false
}

// returnsFunction reports whether any result's type is a function type.
func returnsFunction(results []ast.Returns) bool {
	for _, r := range results {
		if _, isFn := r.Type_.(ast.FunctionType); isFn {
			return true
		}
	}
	return false
}

// statementsOf unwraps a block statement into its body, so a declaration can store
// the statements directly.
func statementsOf(block ast.Stmt) []ast.Stmt {
	if b, ok := block.(*ast.BlockStmt); ok {
		return b.Body
	}
	if block == nil {
		return nil
	}
	return []ast.Stmt{block}
}
