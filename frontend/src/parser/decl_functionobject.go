package parser

import (
	"slices"
	"strings"

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
	if traceEnabled || DEBUG_TRACE {
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
	} else {
		symb.IsBody = false
		decl.Body = []ast.Stmt{
			ast.ExpressionStmt{Span: p.spanFrom(spanStart), Expression: target, Symb: p.stmtSymbol("function-object-binding")},
		}
	}

	// Both forms bind under the signature the declaration ended up with: the
	// literal's own, or the empty one an expression binding carries until the
	// callable it names is resolved.
	p.declareFunction(declName.Tok, &decl)
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
	if traceEnabled || DEBUG_TRACE {
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
	p.declareFunction(ctorName.Tok, &decl)

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
	if traceEnabled || DEBUG_TRACE {
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
		if p.startsDirectBody() {
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
	if traceEnabled || DEBUG_TRACE {
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

// Function-Shaped Declaration Classification — docs/language-ref.md.
//
// FoLang deliberately reuses one callable surface syntax for several declarations
// with different semantics, so the parse alone cannot say what a
// `name(params)->(results) = { … }` declaration IS. The reference resolves that
// with a closed table of nine metadata forms, each of which selects the AST
// declaration kind that owns the declaration's semantics:
//
//	@co.dap.generic        -> GenericFunctionDecl
//	@co.dap.decorator      -> DecoratorDecl
//	@co.dap.extension      -> ExtensionMethodDecl
//	@co.dap.macro          -> MacroDecl
//	@co.dap.template       -> TemplateDecl
//	@co.dap.native         -> NativeFunctionDecl
//	@co.dap.executionmodel -> ExecutionModelFunctionDecl
//	@co.dap.operator       -> OperatorOverloadDecl
//	@co.dap.indexer        -> IndexerDecl
//
// Three rules govern the table, and each one is a rule this code previously broke:
//
//   - The table is CLOSED. A function-shaped declaration carrying any OTHER
//     metadata is an ordinary FunctionDecl "irrespective of other annotations,
//     directives, pragmas, or decorators attached to it". @co.dap.annotation is
//     an ordinary function with metadata, and @co.dap.matcher belongs to the
//     `_ co.lang.matcher` declaration rather than to a function at all, so
//     neither selects a node kind.
//
//   - The classification is LOCAL to function-shaped declarations. @co.dap.generic
//     on a co.lang.struct does not make a GenericFunctionDecl; the struct's own
//     declaration kind stays authoritative. That is why this runs only on the
//     function-shaped path and never from a kind-identified declaration.
//
//   - The classifying forms are mutually exclusive. More than one on the same
//     function-shaped declaration is a compiler error because one declaration
//     cannot have two declaration kinds.

// functionShapeClassifications is the reference's table in the priority order
// this parser resolves a multi-classified declaration with.
//
// The reference states that such a declaration stays specialized but does not
// name which specialization wins, so the order is chosen and documented here
// rather than left to map iteration. It runs from the most specific declaration
// identity to the least:
//
//	operator        an operator overload is identified by the symbol it implements,
//	                which no other classification supplies. The one combination the
//	                reference writes — @co.dap.operator with @co.dap.extension —
//	                resolves here, and OperatorStmt carries the extension target.
//	indexer         likewise identified by the symbol it implements, "[]" or "[]=",
//	                rather than by its ordinary function name; it ranks beside
//	                operator and above every form that identifies a declaration by
//	                what its body IS.
//	macro,          a macro and a template are compile-time expansion forms whose
//	template        body is not an ordinary runtime body.
//	decorator       a decorator's parameter and result are the target it rewrites.
//	executionmodel  the execution semantics are properties of an otherwise ordinary
//	                callable, so they rank below the forms that change what the
//	                body IS.
//	native          likewise: the body is supplied outside FoLang.
//	extension       method-level extension placement.
//	generic         type parameterization, which every other form may also carry.
var functionShapeClassifications = []string{
	"@co.dap.operator",
	"@co.dap.indexer",
	"@co.dap.macro",
	"@co.dap.template",
	"@co.dap.decorator",
	"@co.dap.executionmodel",
	"@co.dap.native",
	"@co.dap.extension",
	"@co.dap.generic",
}

// parseDecoratedFunctionDeclaration parses a function-shaped declaration and
// classifies it.
//
// Every context that admits function-declaration routes through here, because the
// classification rule is a property of the declaration's shape and metadata rather
// than of the container it sits in: a class method, a unit member and a component
// member carrying @co.dap.native are all NativeFunctionDecl.
func (p *parser) parseDecoratedFunctionDeclaration(annotations annotationSet) ast.Stmt {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	decl := p.parseFunctionDeclaration(annotations)

	fn, ok := decl.(ast.FunctionDeclarationStmt)
	if !ok {
		// Recovery can return a placeholder rather than a function declaration.
		return decl
	}
	return p.classifyFunctionShapedDeclaration(fn, annotations)
}

// classifyFunctionShapedDeclaration returns the AST declaration kind the
// Function-Shaped Declaration Classification table selects for fn.
//
// An unclassified declaration is returned unchanged, which is the rule's default:
// a function-shaped declaration outside the table is an ordinary FunctionDecl.
//
// The metadata has to be attached DIRECTLY to this declaration for the guard to
// fire, which is what the annotationSet passed in already means: metadata reaching
// the declaration by any other route does not classify it.
//
// Implements: function-shaped-declaration-classification-guard
func (p *parser) classifyFunctionShapedDeclaration(fn ast.FunctionDeclarationStmt, annotations annotationSet) ast.Stmt {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	classifiers := functionShapeClassifiers(annotations)
	if len(classifiers) == 0 {
		return fn
	}
	fn.Classifiers = classifiers
	if annotations.has("@co.dap.extension") && p.file.Source.Class != sourceClassOrdinaryUnit {
		p.reportf(p.cur(), "a function-level @co.dap.extension declaration is valid only inside an ordinary <Fragment>.unit.fol file, not a companion unit, class, or other source form")
	}
	if annotations.has("@co.dap.operator") && annotations.has("@co.dap.generic") {
		p.reportf(p.cur(), "an operator declaration cannot carry @co.dap.generic because operators never introduce operator-level generic parameters")
	} else if len(classifiers) > 1 && !validFunctionShapeClassifierCombination(classifiers) {
		p.reportf(p.cur(), "function-shape classifiers %s are mutually exclusive", strings.Join(classifiers, " and "))
	}

	switch classifiers[0] {
	case "@co.dap.operator":
		fn.Symb.IsOperator = true
		p.registerDeclaredOperator(annotations)
		return ast.OperatorStmt{
			FunctionDeclarationStmt: fn,
			Type_:                   "operator",
			ForType:                 annotations.optionString("@co.dap.extension", "fortype"),
			What:                    annotations.optionString("@co.dap.extension", "what"),
			IsExtension:             annotations.has("@co.dap.extension"),
		}

	case "@co.dap.indexer":
		fn.Symb.Type_ = "indexer"
		symbol := annotations.optionString("@co.dap.indexer", "symbol")
		p.validateIndexerDeclaration(fn, symbol)
		return ast.IndexerStmt{
			FunctionDeclarationStmt: fn,
			Type_:                   "indexer",
			Symbol:                  symbol,
		}

	case "@co.dap.macro":
		fn.Symb.Type_ = "macro"
		return ast.MacroStmt{
			FunctionDeclarationStmt: fn,
			Type_:                   "macro",
			IsExportable:            annotations.has("@co.dap.export"),
		}

	case "@co.dap.template":
		fn.Symb.Type_ = "template"
		return ast.TemplateStmt{FunctionDeclarationStmt: fn, Type_: "template"}

	case "@co.dap.decorator":
		fn.Symb.Type_ = "decorator"
		return ast.DecoratorStmt{FunctionDeclarationStmt: fn, Type_: "decorator"}

	case "@co.dap.executionmodel":
		fn.Symb.Type_ = "executionmodel"
		return ast.ExecutionModelFunctionStmt{
			FunctionDeclarationStmt: fn,
			Type_:                   "executionmodel",
			ExecutionModel:          annotations.optionString("@co.dap.executionmodel", "type"),
			Kind:                    annotations.optionString("@co.dap.executionmodel", "kind"),
			Completion:              annotations.optionString("@co.dap.executionmodel", "completion"),
			Control:                 annotations.optionString("@co.dap.executionmodel", "control"),
		}

	case "@co.dap.native":
		fn.Symb.Native = true
		fn.Symb.Type_ = "native"
		return ast.NativeFunctionStmt{FunctionDeclarationStmt: fn, Type_: "native"}

	case "@co.dap.extension":
		return ast.ExtensionStmt{
			FunctionDeclarationStmt: fn,
			ForType:                 annotations.optionString("@co.dap.extension", "fortype"),
			What:                    annotations.optionString("@co.dap.extension", "what"),
		}

	case "@co.dap.generic":
		fn.Symb.IsGeneric = true
		return ast.GenerricFun{
			FunctionDeclarationStmt: fn,
			Type_:                   "generic",
			Generic:                 *p.genericDetails(fn.Name, nil),
		}
	}

	return fn
}

// validFunctionShapeClassifierCombination recognizes the operator-specific
// composition described by the reference. An operator remains OperatorDecl and
// extension supplies its existing target/owner; operators themselves are never
// generic declarations.
func validFunctionShapeClassifierCombination(classifiers []string) bool {
	return len(classifiers) == 2 &&
		slices.Contains(classifiers, "@co.dap.operator") &&
		slices.Contains(classifiers, "@co.dap.extension")
}

func (p *parser) validateIndexerDeclaration(fn ast.FunctionDeclarationStmt, symbol string) {
	if symbol != "[]" && symbol != "[]=" {
		p.reportf(p.cur(), "@co.dap.indexer requires symbol=\"[]\" or symbol=\"[]=\"; found %q", symbol)
	}
	if p.file.Source.Class != sourceClassCompanionUnit {
		p.reportf(p.cur(), "an indexer must be declared inside <StructName>.comp.unit.fol")
		return
	}
	if fn.AssociatedReceiver == nil {
		p.reportf(p.cur(), "an indexer requires an explicit receiver of its companion struct type")
		return
	}
	actual := symbolDeclarationTypeNode(fn.AssociatedReceiver.SymbolStmt)
	owner := p.file.Source.DerivedName
	if owner != "" && logicalTypeName(actual) != owner {
		p.reportf(p.cur(), "indexer receiver type %q does not match companion owner %q", logicalTypeName(actual), owner)
	}
}

// functionShapeClassifiers returns the classifying metadata attached to a
// declaration, in the priority order of functionShapeClassifications.
//
// The result is both the selector and the record: its first entry chooses the node
// kind, and the whole slice is what a multi-classified declaration keeps so that no
// classification is lost by the one the node's type names.
func functionShapeClassifiers(annotations annotationSet) []string {
	var classifiers []string
	for _, name := range functionShapeClassifications {
		if annotations.has(name) {
			classifiers = append(classifiers, name)
		}
	}
	return classifiers
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
