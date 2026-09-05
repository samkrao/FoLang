package parser

import (
	"github.com/samkrao/fo-lang/src/ast"
	symboltable "github.com/samkrao/fo-lang/src/context"
	"github.com/samkrao/fo-lang/src/helpers"
	"github.com/samkrao/fo-lang/src/scanlex"
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
//	function-definition  = "=", block, body-closure-guard
//	function-delegation  = "=>>", expression,
//	                       { "=>>", expression }, statement-end
//	function-alias-binding = "=", non-block-expression, statement-end
//
// The four bindings are the four ways a function can be given a body:
//
//	add(a T)->(T) = { … }                              definition
//	fetch(id S)->(E) =>> mod.get(this, id);            delegation chain
//	shorthand(a T)->(T) = someOtherFunction;           alias
//	forwardDeclared(a T)->(T);                         forward declaration
//
// A NAMED function's block body requires the "=" (docs/grammar/folang.ebnf, preamble).
// Only an anonymous function literal juxtaposes its signature and its body, which is
// what keeps the two forms apart on sight. Every named body in the reference is written
// with the "=".
//
// Delegation is spelled "=>>" alone. The single "=>" belongs to the function-pattern
// clauses, where it separates a pattern head from its result; accepting it here made one
// arrow mean two unrelated things.
//
// The definition form ends at its closing brace and takes NO semicolon, while every
// other form ends with one. That is the distinction body-closure-guard enforces, and
// getting it wrong is the most common way to mis-parse a declaration body as an
// expression.

// parseFunctionDeclaration parses the function-declaration production.
//
// Implements: function-declaration
func (p *parser) parseFunctionDeclaration(annotations annotationSet) ast.Stmt {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	var receiver *ast.FunctionReceiver
	if p.atReceiverClause() {
		receiver = p.parseReceiverClause()
	}

	funcName := p.parseFunctionName("as a function name")
	return p.continueFunctionDeclarationWithReceiver(funcName, receiver, annotations)
}

// parseReceiverFunctionDeclaration parses a function after its enclosing
// companion-unit dispatcher has selected receiver-clause from context. It does
// no lookahead: direct unit members admit no competing parenthesized construct.
func (p *parser) parseReceiverFunctionDeclaration(annotations annotationSet) ast.Stmt {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	receiver := p.parseReceiverClause()
	funcName := p.parseFunctionName("as the name of an associated function")
	return p.continueFunctionDeclarationWithReceiver(funcName, receiver, annotations)
}

// continueFunctionDeclaration parses a function-declaration whose name has already been
// consumed.
//
// The primary-declaration dispatcher has to read a declaration's name before it can tell
// which kind of declaration it is, so by the time a function is identified the name is gone.
// This entry point lets that caller continue from the parameter list.
func (p *parser) continueFunctionDeclaration(funcName name, annotations annotationSet) ast.Stmt {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}
	return p.continueFunctionDeclarationWithReceiver(funcName, nil, annotations)
}

// continueFunctionDeclarationWithReceiver parses the rest of a function-declaration after
// its optional receiver clause and its name.
func (p *parser) continueFunctionDeclarationWithReceiver(funcName name, receiver *ast.FunctionReceiver, annotations annotationSet) ast.Stmt {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}
	spanStart := p.pos

	// The NAME belongs to the enclosing scope, so its symbol is minted before the
	// function's own context opens. Everything after it — parameters, results and
	// the body — belongs to that context, which is why it spans the parameter list
	// rather than starting at the body's brace (docs/language-ref.md, B.1).
	symb := p.functionSymbol(funcName.Scanned)
	defer p.pushContext(symboltable.S_FunctionSymbol, symb)()
	p.declareGenericAnnotationTypes(annotations)
	p.declareReceiver(receiver)

	paramLists := p.parseParameterLists(annotations.has("@co.dap.template"))

	var results []ast.Returns
	if p.at(scanlex.ARROW) {
		results = p.parseReturnTypeClause()
	}
	p.validateGenericAliasSignatureNames(annotations, paramLists, results)

	decl := ast.FunctionDeclarationStmt{NodeName: "FunctionDeclarationStmt", Span: p.spanFrom(spanStart), Parameters: paramLists,
		Name:               funcName.Scanned,
		ReturnType:         results,
		AssociatedReceiver: receiver,
		Dapst:              annotations.list(),
		Symb:               symb,
	}
	p.applyFunctionFlags(&decl, annotations)
	p.declareFunction(funcName.Tok, &decl)

	return p.parseFunctionBinding(decl)
}

// parseFunctionBinding parses the function-binding production and attaches the result
// to decl.
//
// Implements: function-binding
func (p *parser) parseFunctionBinding(decl ast.FunctionDeclarationStmt) ast.Stmt {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	// A "=>" here is the pattern-clause arrow, which no function-binding spells.
	if p.atOp("=>") {
		p.fail(p.cur(), "a delegating function forwards with \"=>>\"; \"=>\" introduces the result of a function-pattern clause, not a function binding")
	}

	switch {
	// function-delegation: "=>>" followed by a chain of expressions.
	case p.atOp("=>>"):
		return p.parseFunctionDelegation(decl)

	// function-definition. The brace must follow the "=", otherwise this is an
	// alias binding.
	case p.atOp("=") && p.definitionFollowsAssign():
		p.advance() // "="
		return p.finishFunctionDefinition(decl)

	// function-alias-binding: "=" followed by an expression.
	case p.atOp("="):
		p.advance()
		return p.parseFunctionAliasBinding(decl)

	// A named function's body requires the "=". The body is still parsed after the
	// report so that one missing token yields one diagnostic rather than cascading
	// through every statement inside the block.
	case p.at(scanlex.OPEN_CURLY):
		p.report(p.cur(), "a named function's block body is bound with \"=\"; write \"= {\" here — only an anonymous function literal places its body directly after the signature")
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
// The two are distinguished by what follows: a "{" always opens a body, because a
// braced group in operand position has no map-literal reading to compete with. An
// anonymous function that is itself a direct inline body also counts, which is what
// makes the function-object form work.
func (p *parser) definitionFollowsAssign() bool {
	// A bare brace cannot begin an expression, so `= {` selects a definition and
	// every other `=` selects an alias expression. Adding a bare braced value in
	// the future would invalidate this one-token decision.
	return p.peek(1).Kind == scanlex.OPEN_CURLY
}

// finishFunctionDefinition parses the block body of a function-definition.
//
// The body ends at its closing brace and takes no semicolon, which
// body-closure-guard asserts.
//
// Implements: function-definition
func (p *parser) finishFunctionDefinition(decl ast.FunctionDeclarationStmt) ast.Stmt {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}
	// The function's context is already open — see continueFunctionDeclarationWithReceiver
	// — so the body's brace joins it instead of nesting a block context inside it.
	body := p.parseScopeBlock("a function body")
	p.bodyClosureGuard("a function body")

	decl.Body = statementsOf(body)
	decl.Symb.IsBody = true
	return decl
}

// parseFunctionDelegation parses the function-delegation production:
//
//	function-delegation = "=>>", expression,
//	                      { "=>>", expression }, statement-end
//
// A delegating function forwards its work rather than computing it
// (docs/language-ref.md, "Function Chaining"):
//
//	fetchEmployee(empId co.lang.string)->(Employee)=>>empMod.getEmployee(this, empId);
//
// In a "=>>" chain each stage's result is available to the next through the "$1",
// "$2", … result bindings.
//
// Implements: function-delegation
func (p *parser) parseFunctionDelegation(decl ast.FunctionDeclarationStmt) ast.Stmt {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	p.advance() // "=>>"

	stages := []ast.Stmt{
		ast.ExpressionStmt{NodeName: "ExpressionStmt", Span: p.spanFrom(spanStart), Expression: p.parseExpression(),
			SymbolId: p.statementID("delegation"),
		},
	}

	for p.atOp("=>>") {
		p.advance()
		stages = append(stages, ast.ExpressionStmt{NodeName: "ExpressionStmt", Span: p.spanFrom(spanStart), Expression: p.parseExpression(),
			SymbolId: p.statementID("delegation"),
		})
	}

	p.statementEnd("a function delegation")

	decl.Body = stages
	decl.Symb.IsBody = true
	decl.Symb.FunctionChain = true
	decl.Symb.Delegate = true
	return decl
}

// parseFunctionAliasBinding parses the function-alias-binding production:
//
//	function-alias-binding = "=", non-block-expression, statement-end
//
// This binds the declared name to an existing callable rather than to a body.
//
// Implements: function-alias-binding
func (p *parser) parseFunctionAliasBinding(decl ast.FunctionDeclarationStmt) ast.Stmt {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	var target ast.Expr
	if p.startsStructuralTypeValue() {
		// Implements: function-type-value-binding
		// Types are values in FoLang. A structural type expression returned by an
		// ordinary function uses the existing type-object expression node; it does
		// not create a separate syntactic category of function.
		valueStart := p.pos
		t := p.parseTypeExpression()
		target = ast.SDTExpr{NodeName: "SDTExpr", Span: p.spanFrom(valueStart), Type_: t.fullType(), Symb: p.exprSymbol(t.actType())}
	} else {
		target = p.parseExpression()
	}
	p.statementEnd("a function alias binding")

	decl.Body = []ast.Stmt{
		ast.ExpressionStmt{NodeName: "ExpressionStmt", Span: p.spanFrom(spanStart), Expression: target, SymbolId: p.statementID("alias")},
	}
	decl.Symb.IsBody = false
	decl.Symb.FunctionObject = true
	return decl
}

// startsStructuralTypeValue recognizes type-value spellings the ordinary
// expression parser cannot consume. This is bounded prefix selection, not a
// scan across the function signature: forall is contextual and a named
// structural derivation has an immediately following arrow.
func (p *parser) startsStructuralTypeValue() bool {
	if p.atKeyword("forall") && p.forallContextGuard() {
		return true
	}
	return p.startsTypeExpression(p.cur()) && p.peek(1).Kind == scanlex.ARROW
}

// parseFunctionSpecification parses the function-specification production:
//
//	function-specification = annotations, [ receiver-clause ], function-name,
//	                         parameter-list, { parameter-list },
//	                         [ return-type-clause ], statement-end
//
// A specification is a signature with no body. It is what an interface, a signature
// and a contract body are made of.
//
// Implements: function-specification
func (p *parser) parseFunctionSpecification(annotations annotationSet) ast.Stmt {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	var receiver *ast.FunctionReceiver
	if p.atReceiverClause() {
		receiver = p.parseReceiverClause()
	}

	funcName := p.parseFunctionName("as a function name")
	symb := p.functionSymbol(funcName.Scanned)
	defer p.pushContext(symboltable.S_FunctionSymbol, symb)()
	p.declareGenericAnnotationTypes(annotations)
	p.declareReceiver(receiver)
	paramLists := p.parseParameterLists(annotations.has("@co.dap.template"))

	var results []ast.Returns
	if p.at(scanlex.ARROW) {
		results = p.parseReturnTypeClause()
	}
	p.validateGenericAliasSignatureNames(annotations, paramLists, results)

	p.statementEnd("a function specification")

	decl := ast.FunctionDeclarationStmt{NodeName: "FunctionDeclarationStmt", Span: p.spanFrom(spanStart), Parameters: paramLists,
		Name:               funcName.Scanned,
		ReturnType:         results,
		AssociatedReceiver: receiver,
		Dapst:              annotations.list(),
		Symb:               symb,
	}
	decl.Symb.IsBody = false
	decl.Symb.OnlyParamTypes = true
	p.applyFunctionFlags(&decl, annotations)
	p.declareFunction(funcName.Tok, &decl)
	return decl
}

// local-function-declaration — section 8.
//
//	local-function-declaration = annotations, function-name, parameter-list,
//	                             { parameter-list }, return-type-clause,
//	                             function-definition
//
// A function declared inside a block requires BOTH a return-type clause and a block
// body. That is what keeps `foo();` an expression statement rather than a forward
// declaration, and it is what admits the reference's inner-function form
// (docs/language-ref.md, "Inner Function"):
//
//	someother()->()={
//	    co.out.println(p);
//	}
//
// The body comes through function-definition, so the "=" is part of the shape rather
// than an optional flourish.

// atLocalFunctionDeclaration reports whether the cursor begins a
// local-function-declaration.
//
// The return-type clause, the "=" and the body are all required, so the probe checks
// for the full `name ( … ) -> ( … ) = {` shape. Probing for the "=" rather than
// accepting it optionally is what keeps a body written without one from being
// recognised here and then reported by a rule that no longer applies to it.
func (p *parser) atLocalFunctionDeclaration() bool {
	if !p.atIdentifier() && !p.atLifecycleName() {
		return false
	}
	if p.peek(1).Kind != scanlex.OPEN_PAREN {
		return false
	}

	// The ordinary decision is LL(3): `name ( parameter-name type` commits to a
	// declaration as soon as the explicit type is seen. The rest of the signature
	// is parsed once by parseLocalFunctionDeclaration; a malformed declaration is
	// diagnosed as such and is never retried as a call expression.
	//
	// Consecutive parameter lists are the one documented exception. With currying,
	// `f()(x)` is a chained call while `f()(x T)->(R)` is a declaration. Empty
	// leading lists therefore postpone the decision until the first non-empty list
	// or an immediate arrow. This probe walks only those empty `()` pairs; it never
	// scans a non-empty parameter list or rewinds parser state. Removing the loop
	// would deliberately drop support for curried declarations whose earlier lists
	// are empty.
	offset := 1 // the first "(" after the function name
	for p.peek(offset).Kind == scanlex.OPEN_PAREN {
		if p.peek(offset+1).Kind == scanlex.CLOSE_PAREN {
			offset += 2
			if p.peek(offset).Kind == scanlex.ARROW {
				return true
			}
			continue
		}
		return p.startsTypedParameterPrefix(offset + 1)
	}
	return false
}

// startsTypedParameterPrefix recognizes only enough of the first parameter to
// distinguish a declaration from a call. Qualified names are already folded by
// scanlex, so the common `(a T` form is three tokens including "(". Optional,
// named and variadic markers add their own explicit tokens but no ambiguity.
func (p *parser) startsTypedParameterPrefix(offset int) bool {
	if p.peek(offset).Kind == scanlex.DOT_DOT_DOT {
		offset++
	}
	if p.peek(offset).Value == "~" {
		offset++
	}
	if !isIdentifierToken(p.peek(offset)) {
		return false
	}
	offset++
	if p.peek(offset).Kind == scanlex.QUESTION {
		offset++
	}
	return p.startsTypeUse(p.peek(offset))
}

func isIdentifierToken(tok scanlex.Token) bool {
	return tok.IsOneOfMany(scanlex.IDENTIFIER, scanlex.COMPOSITE_IDENTIFER) ||
		(tok.IsOneOfMany(scanlex.KEYWORD, scanlex.CONTEXT_KEYWORD) && contextualKeywords[tok.Value])
}

// parseLocalFunctionDeclaration parses the local-function-declaration production.
//
// Implements: local-function-declaration
func (p *parser) parseLocalFunctionDeclaration(annotations annotationSet) ast.Stmt {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	funcName := p.parseFunctionName("as a local function name")
	if annotations.has("@co.dap.generic") {
		p.reportNamed(p.cur(), helpers.DiagnosticInvalidMetadataPlacement, "Invalid Metadata Placement", "an inner function cannot be generic; declare the generic function directly in a unit, class, module, mixin, or signature")
	}

	// As for a top-level function, the name is declared in the enclosing block and
	// the signature and body are the inner function's own context.
	symb := p.functionSymbol(funcName.Scanned)
	defer p.pushContext(symboltable.S_FunctionSymbol, symb)()
	p.declareGenericAnnotationTypes(annotations)

	paramLists := p.parseParameterLists(annotations.has("@co.dap.template"))
	results := p.parseReturnTypeClause()
	p.validateGenericAliasSignatureNames(annotations, paramLists, results)

	decl := ast.FunctionDeclarationStmt{NodeName: "FunctionDeclarationStmt", Span: p.spanFrom(spanStart), Parameters: paramLists,
		Name:       funcName.Scanned,
		ReturnType: results,
		Scope:      "local",
		Dapst:      annotations.list(),
		Symb:       symb,
	}
	decl.Symb.InnerFunction = true
	decl.Symb.IsInner = true
	p.applyFunctionFlags(&decl, annotations)
	p.declareFunction(funcName.Tok, &decl)

	p.expectOp("=", "before a local function's block body")
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
		case "@co.dap.class":
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
		case "@co.dap.defer":
			symb.Defer = true
		case "@co.dap.callable":
			symb.Callback = true
		case "@co.dap.operator":
			symb.IsOperator = true
		case "@co.dap.executionmodel":
			applyExecutionModelFlags(symb, annotations)
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

// applyExecutionModelFlags records the execution semantics of
// `@co.dap.executionmodel(type=…, kind=…)` on the symbol.
//
// FoLang has ONE execution-model decorator rather than a spelling per kind, so
// the symbol's per-kind flags are set from the decorator's fields. Sequential is
// deliberately absent: it is the default and has no decorator form, so an
// undecorated function already has sequential semantics
// (docs/language-ref.md, "Default Sequential Execution").
//
// A field the reference has not defined a flag for still reaches later phases:
// the complete decorator payload stays on the declaration's Dapst and on the
// ExecutionModelFunctionStmt the classifier produces.
func applyExecutionModelFlags(symb *symboltable.FunctionSymbol, annotations annotationSet) {
	model := annotations.optionString("@co.dap.executionmodel", "type")
	kind := annotations.optionString("@co.dap.executionmodel", "kind")

	switch model {
	case "concurrent":
		symb.Concurrent = true
	case "parallel":
		symb.Parallel = true
	case "async":
		symb.Async = true
		switch annotations.optionString("@co.dap.executionmodel", "completion") {
		case "promise":
			symb.Promise = true
		case "future":
			symb.Future = true
		}
	case "continuation":
		symb.Coroutine = true
	}

	// `kind=` narrows the family to the runtime shape the model requires. It is
	// meaningful only under a family that defines it, which is why it is read
	// after the family rather than instead of it.
	switch kind {
	case "task":
		symb.Task = true
	case "thread":
		symb.Thread = true
	case "fiber":
		symb.Fiber = true
	case "process":
		symb.Process = true
	case "actor":
		symb.Actor = true
	case "channel":
		symb.Channel = true
	case "generator":
		symb.Generator = true
	case "iterator":
		symb.Iterator = true
	}
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
