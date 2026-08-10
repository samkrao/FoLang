package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// function-object-declaration — section 7.
//
//	function-object-declaration = annotations, identifier,
//	                              "co.lang.function", "=",
//	                              function-object-binding
//	function-object-binding     = expression, statement-end
//
// The head carries an ordinary identifier rather than "_", because this is a unit member
// and not a file-backed primary. The reference presents it as Syntax 3 of the
// function-type feature, beside the named type alias, which has always named itself.
//
// The binding is ONE alternative: an expression terminated by ";". Both spellings the
// reference shows are expressions, so neither needs its own rule
// (docs/language-ref.md, "Other ways to declare closures/function objects"):
//
//	someFRet co.lang.function = (a co.lang.int)->(co.lang.int){
//	    this.return a * 2;
//	};                                      an anonymous function literal
//
//	oObj co.lang.function = add;            an existing callable
//
// The anonymous function is an EXPRESSION here rather than a declaration body, which is
// why the reference writes the ";" after its closing brace. Treating it as a body — and
// so rejecting that ";" through body-closure-guard — contradicted every example.

// parseFunctionObjectDeclaration parses the function-object-declaration production.
//
// Implements: function-object-declaration
// Implements: function-object-binding
func (p *parser) parseFunctionObjectDeclaration(declName name, annotations annotationSet) ast.Stmt {
	spanStart := p.pos
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	p.expectOp("=", "before a function-object binding")

	target := p.parseExpression()
	p.statementEnd("a function-object binding")

	symb := p.functionSymbol(declName.Scanned)
	symb.FunctionObject = true

	decl := ast.FunctionDeclarationStmt{Span: p.spanFrom(spanStart), Name: declName.Scanned,
		Dapst: annotations.list(),
		Symb:  symb,
	}

	// A function literal supplies the object's own signature and body, so those are
	// lifted onto the declaration; any other expression names a callable declared
	// elsewhere and is carried as the bound expression.
	if fn, isLiteral := target.(ast.FunctionExpr); isLiteral {
		symb.Closure = true
		symb.IsBody = true
		decl.Parameters = [][]ast.Parameter{fn.Parameters}
		decl.Body = fn.Body
		decl.ReturnType = fn.ReturnType
		return decl
	}

	symb.IsBody = false
	decl.Body = []ast.Stmt{
		ast.ExpressionStmt{Span: p.spanFrom(spanStart), Expression: target, Symb: p.stmtSymbol("function-object-binding")},
	}
	return decl
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
		p.failf(ctorName.Tok, "type-level function %q must return co.lang.dependentType, co.lang.type, or a union of those kinds", ctorName.Logical)
	}
}

// typeLevelResultKinds is the closed set of kinds whose value is itself
// a type. The EBNF uses this production inside type-level-return-clause:
//
//	type-level-result-kind = "co.lang.dependentType" | "co.lang.type"
//
// Only these two have a type-level return form in the reference; every type-level
// function it shows returns one of them. The kind-level names alongside them in the
// Builtin Kinds table have no such form and stay reserved.
var typeLevelResultKinds = map[string]struct{}{
	"co.lang.dependentType": {},
	"co.lang.type":          {},
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

	// A "=>" here is the pattern-clause arrow; function-delegation spells "=>>".
	if p.atOp("=>") {
		p.fail(p.cur(), "a delegating type-level function forwards with \"=>>\"; \"=>\" introduces the result of a function-pattern clause")
	}

	switch {
	// function-delegation.
	case p.atOp("=>>"):
		return p.parseFunctionDelegation(decl)

	case p.atOp("="):
		p.advance()

		// function-definition: the "=" just consumed is the one it requires.
		if p.startsDirectBody() && !p.looksLikeMapLiteral() {
			return p.finishFunctionDefinition(decl)
		}

		// The type-expression reading is tried first, so a binding that is a
		// complete type is not consumed as an ordinary expression.
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
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}
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

// parseDecoratedFunctionDeclaration parses a function and applies the AST
// wrapper selected by its annotations.
//
// FoLang declares macros, templates, operators, extensions and indexers as ANNOTATED
// functions rather than with dedicated kinds, so the wrapping step is what marks them and
// what registers a custom operator with the Pratt table. Without it, a unit member could
// carry @co.dap.operator while remaining an ordinary, unregistered function.
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
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}
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
