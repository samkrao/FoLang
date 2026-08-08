package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// function-object-declaration — section 7.
//
//	function-object-declaration = annotations, filename-derived-name,
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
//	_ co.lang.function = (a co.lang.int)->(co.lang.int) = {
//	    this.return a * 2;
//	}                                       a BODY: ends at "}", no ";"
//
//	_ co.lang.function = add;               an EXPRESSION: ends at ";"
//
// The first is a direct anonymous function used as the declaration's inline body; the
// second binds an existing callable. Ordered choice alone cannot separate them, so the
// anonymous-function reading is selected by the affirmative guard
// startsAnonymousFunction and the expression reading is everything else.

// parseFunctionObjectDeclaration parses the function-object-declaration production.
//
// Implements: function-object-declaration
func (p *parser) parseFunctionObjectDeclaration(declName name, annotations annotationSet) ast.Stmt {
	spanStart := p.pos
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	// A forward declaration of the function kind ends at ";" with no binding.
	if p.at(scanlex.SEMI_COLON) {
		p.advance()
		symb := p.functionSymbol(declName.Scanned)
		symb.FunctionObject = true
		symb.IsBody = false
		return ast.FunctionDeclarationStmt{Span: p.spanFrom(spanStart), Name: declName.Scanned,
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
	spanStart := p.pos
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

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

	return ast.FunctionDeclarationStmt{Span: p.spanFrom(spanStart), Parameters: [][]ast.Parameter{fnExpr.Parameters},
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
	spanStart := p.pos
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	target := p.parseExpression()
	p.statementEnd("a function-object binding")

	symb := p.functionSymbol(declName.Scanned)
	symb.FunctionObject = true
	symb.IsBody = false

	return ast.FunctionDeclarationStmt{Span: p.spanFrom(spanStart), Name: declName.Scanned,
		Body: []ast.Stmt{
			ast.ExpressionStmt{Span: p.spanFrom(spanStart), Expression: target, Symb: p.stmtSymbol("function-object-binding")},
		},
		Dapst: annotations.list(),
		Symb:  symb,
	}
}

// type-level-function-declaration — section 7.
//
//	type-level-function-declaration = annotations, function-name, parameter-list,
//	                                  { parameter-list }, type-level-return-clause,
//	                                  type-level-binding
//	type-level-return-clause        = "->", "(", type-level-result-kind,
//	                                  { "|", type-level-result-kind }, ")"
//	type-level-binding              = function-definition
//	                                | function-delegation
//	                                | "=", type-expression, statement-end
//	                                | "=", non-block-expression, statement-end
//	                                | statement-end
//
// A type-level function is a function that returns a TYPE, so its binding may be a type
// expression. DECISION-TYP-002 puts the type-expression alternative BEFORE the expression
// alternative, which is what makes
//
//	Vector(n co.lang.int)->(co.lang.dependentType) = co.lang.int->([n]);
//
// parse as a type rather than as an unparseable expression: `co.lang.int->([n])` is a valid
// type but not a valid value expression.
//
// Revision 23 renamed this production and moved it out of primary-declaration.
// It names itself in its head, so it cannot take a filename-derived name; it is a
// unit-member and belongs in an ordinary `<Fragment>.unit.fol` file
// (docs/language-ref.md, "Type-Level Functions — Functions That Return Types").

// parseTypeLevelFunctionDeclaration parses the type-level-function-declaration production.
//
// Implements: type-level-function-declaration
// Implements: type-level-return-clause
func (p *parser) parseTypeLevelFunctionDeclaration(annotations annotationSet) ast.Stmt {
	spanStart := p.pos
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	ctorName := p.parseFunctionName("as a type-level function name")
	paramLists := p.parseParameterLists()
	results := p.parseReturnTypeClause()
	p.validateTypeLevelResult(ctorName, results)

	decl := ast.FunctionDeclarationStmt{Span: p.spanFrom(spanStart), Parameters: paramLists,
		Name:       ctorName.Scanned,
		ReturnType: results,
		Dapst:      annotations.list(),
		Symb:       p.functionSymbol(ctorName.Scanned),
	}
	p.applyFunctionFlags(&decl, annotations)

	return p.parseTypeLevelBinding(ctorName, decl, annotations)
}

// validateTypeLevelResult enforces the dedicated type-level result clause.
// A type-level function produces exactly one type. That type may be a union written with
// `|`, but every arm must itself be one of the type-producing kinds; a comma would
// describe multiple runtime results and therefore cannot define one constructed type.
func (p *parser) validateTypeLevelResult(ctorName name, results []ast.Returns) {
	if len(results) != 1 {
		p.failf(ctorName.Tok, "type-level function %q must return exactly one type-producing result; found %d results", ctorName.Logical, len(results))
	}
	if results[0].IsNamed {
		p.failf(ctorName.Tok, "type-level function %q has one type result and cannot name it as a separate runtime return value", ctorName.Logical)
	}
	if !isTypeLevelResultType(results[0].Type_) {
		p.failf(ctorName.Tok, "type-level function %q must return co.lang.dependentType, co.lang.type, co.lang.typetype, co.lang.typekind, co.lang.kind, or a union of those kinds", ctorName.Logical)
	}
}

// typeLevelResultKinds is the closed set of kinds whose value is itself
// a type. The EBNF uses this production inside type-level-return-clause:
//
//	type-level-result-kind = "co.lang.dependentType" | "co.lang.type"
//	                             | "co.lang.typetype" | "co.lang.typekind"
//	                             | "co.lang.kind"
var typeLevelResultKinds = map[string]struct{}{
	"co.lang.dependentType": {},
	"co.lang.type":          {},
	"co.lang.typetype":      {},
	"co.lang.typekind":      {},
	"co.lang.kind":          {},
}

// isTypeLevelResultType reports whether a parsed result is the single
// type-producing value admitted by type-level-return-clause.
func isTypeLevelResultType(result ast.Type) bool {
	if union, ok := result.(ast.CompoundType); ok {
		return union.Op == "|" &&
			isTypeLevelResultType(union.Left) &&
			isTypeLevelResultType(union.Right)
	}
	_, ok := typeLevelResultKinds[typeNameOf(result)]
	return ok
}

// typeConstructorResultKind preserves the kind produced by a function-shaped
// constructor on the declaration node. A union remains one result, represented by
// a stable `left|right` spelling while its full structure stays in ReturnType.
func typeLevelResultKind(result ast.Type) string {
	if union, ok := result.(ast.CompoundType); ok && union.Op == "|" {
		return typeLevelResultKind(union.Left) + "|" + typeLevelResultKind(union.Right)
	}
	return typeNameOf(result)
}

func typeLevelResultContains(result ast.Type, kind string) bool {
	if union, ok := result.(ast.CompoundType); ok && union.Op == "|" {
		return typeLevelResultContains(union.Left, kind) ||
			typeLevelResultContains(union.Right, kind)
	}
	return typeNameOf(result) == kind
}

// parseTypeLevelBinding parses the type-level-binding production.
//
// Implements: type-level-binding
func (p *parser) parseTypeLevelBinding(ctorName name, decl ast.FunctionDeclarationStmt, annotations annotationSet) ast.Stmt {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

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
		if bound, ok := p.tryTypeLevelTypeBinding(ctorName, decl, annotations); ok {
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

// tryTypeLevelTypeBinding attempts the `"=", type-expression, statement-end`
// alternative of type-level-binding.
//
// The attempt is speculative and is accepted only when the type expression runs right up to
// the statement terminator, so a token sequence that merely starts like a type does not
// hijack an expression binding. The resulting node keeps BOTH halves of the declaration:
// the function-shaped constructor signature and the complete type it constructs. Dropping
// either loses the value/type parameters that a dependent result refers to, or the array/
// pointer derivation that represents the constructed type.
func (p *parser) tryTypeLevelTypeBinding(ctorName name, decl ast.FunctionDeclarationStmt, annotations annotationSet) (ast.Stmt, bool) {
	var bound ast.Stmt

	matched := p.speculate(func() bool {
		spanStart := p.pos
		t := p.parseTypeExpression()
		if !p.at(scanlex.SEMI_COLON) {
			return false
		}
		p.advance()

		resultType := decl.ReturnType[0].Type_
		resultKind := typeLevelResultKind(resultType)
		symb := p.typeSymbol(ctorName.Scanned)
		applyTypeDeclarationKind(symb, resultKind)
		// A union does not have one direct declaration kind on which
		// applyTypeDeclarationKind can dispatch, but its dependent arm still matters
		// to later type-constructor resolution.
		symb.DependentType = typeLevelResultContains(resultType, "co.lang.dependentType")
		symb.IsGenericType = true

		bound = ast.TypeDeclarationStmt{Span: p.spanFrom(spanStart), Name: ctorName.Scanned,
			Parameters: decl.Parameters,
			ReturnType: decl.ReturnType,
			Type_:      t.fullType(),
			Kind:       resultKind,
			SubType_:   "TYPE_CONSTRUCTOR",
			Typetype:   "UDT",
			SDapst:     annotations.list(),
			Symb:       symb,
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
// that a macro, template, extension or indexer can be the single declaration of a package
// source file. The annotation decides which wrapper node the declaration becomes.
//
// An operator is the deliberate exception. The normative "Custom Operator Definition &
// Overloading" section requires operator functions to live in the matching struct's
// same-package companion unit, so an operator annotation at package scope is diagnosed here
// even though the general annotated-function-primary shape can recognize it.

// parseAnnotatedFunctionPrimary parses the annotated-function-primary production.
//
// Implements: annotated-function-primary
func (p *parser) parseAnnotatedFunctionPrimary(annotations annotationSet) ast.Stmt {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	if annotations.has("@co.dap.operator") {
		p.reportf(p.cur(), "an operator function cannot be declared at package scope; declare it in a named class, a struct companion unit, or a built-in extension unit")
	}
	return p.parseDecoratedFunctionDeclaration(annotations)
}

// parseDecoratedFunctionDeclaration parses a function and applies the AST
// wrapper selected by its annotations.
//
// Operator functions are valid inside their companion unit, while
// annotated-function-primary uses the same annotation vocabulary for the
// package-level declarations permitted by the grammar. Both paths must perform
// the wrapping step because it marks macros/templates/operators and registers
// custom operators with the Pratt table. Without it, a unit member could carry
// @co.dap.operator while remaining an ordinary, unregistered function.
func (p *parser) parseDecoratedFunctionDeclaration(annotations annotationSet) ast.Stmt {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	decl := p.parseFunctionDeclaration(annotations)

	fn, ok := decl.(ast.FunctionDeclarationStmt)
	if !ok {
		// Recovery can return a placeholder rather than a function declaration.
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
		p.registerDeclaredOperator(annotations)
		return ast.OperatorStmt{
			FunctionDeclarationStmt: fn,
			Type_:                   "operator",
			ForType:                 annotations.optionString("@co.dap.extension", "fortype"),
			What:                    annotations.optionString("@co.dap.extension", "what"),
			IsExtension:             annotations.has("@co.dap.extension"),
		}

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
func (p *parser) registerDeclaredOperator(annotations annotationSet) {
	options := map[string]any{}
	for _, key := range []string{"symbol", "mode"} {
		if value, ok := annotations.option("@co.dap.operator", key); ok {
			options[key] = value
		}
	}
	p.registerOperatorDeclaration(options, "an @co.dap.operator declaration")
}
