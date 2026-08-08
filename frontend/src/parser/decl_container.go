package parser

import (
	"strings"

	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// Container declarations — section 7.
//
// Each of these declares a named container whose body holds members, and each ends
// structurally at its closing brace with no trailing semicolon (DECISION-SYN-006). They
// differ in which members they admit, which is what each parseXMember function below
// encodes.

// unit-declaration.
//
//	unit-declaration = annotations, filename-derived-name, "co.lang.unit", "=",
//	                   unit-body
//	unit-body        = "{", { unit-member }, body-close
//	unit-member      = function-declaration
//	                 | data-declaration
//	                 | type-declaration
//	                 | type-level-function-declaration
//	                 | annotated-function-primary
//
// A unit is the container FoLang requires package-level functions and non-UDT type
// declarations to live in: the language does not allow free-flowing functions in a
// package source file, so a unit is what groups them
// (docs/language-ref.md, "Units in detail"):
//
//	// arithmetic.unit.fol
//	_ co.lang.unit = {
//	    abs(value co.lang.int)->(co.lang.int) = { … }
//	    Option(T) co.lang.type = Some(T) | None();
//	}
//
// DECISION-UNIT-001: both unit source forms use this same grammar. The FILENAME,
// not the body, decides what the members mean — an ordinary `<Fragment>.unit.fol`
// merges its members into the package namespace, while a
// `<StructName>.comp.unit.fol` companion attaches them to that struct. Which is
// why the owner is taken from p.file rather than from declName: a fragment name
// is organizational only and never owns anything.

// parseUnitDeclaration parses the unit-declaration production.
//
// Implements: unit-declaration
func (p *parser) parseUnitDeclaration(declName name, annotations annotationSet) ast.Stmt {
	spanStart := p.pos
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	p.expectOp("=", "before a unit body")

	members := p.parseBracedBody("a unit body", p.parseUnitMember)

	return ast.TypeDeclarationStmt{Span: p.spanFrom(spanStart), Name: declName.Scanned,
		Body:     members,
		Kind:     "co.lang.unit",
		SubType_: "UNIT",
		Typetype: "UDT",
		SDapst:   annotations.list(),
		KDapst:   annotations.list(),
		Symb:     p.typeSymbol(declName.Scanned),
	}
}

// parseUnitMember parses the unit-member production.
//
// Revision 23 widened a unit body from functions alone to the declarations a
// package's non-UDT surface needs: `co.lang.data` algebraic types, the
// `co.lang.type` alias family including parameterized type constructors, and
// type-level functions. Each of those names ITSELF in its head — a filename
// cannot carry `Option(T)` — which is exactly why they are unit members rather
// than file-backed primaries (DECISION-FILE-003, DECISION-GEN-001).
//
// Implements: unit-member
func (p *parser) parseUnitMember() ast.Stmt {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	annotations := p.parseAnnotations()

	// type-level-function-declaration: a function whose result is a type. It is
	// probed before the ordinary function reading because the two share the
	// `name(…)->(…)` shape and differ only in the result kind.
	if p.atTypeLevelFunctionDeclaration() {
		p.rejectOperatorPlacement(annotations, "a type-level function")
		return p.parseTypeLevelFunctionDeclaration(annotations)
	}

	// data-declaration and type-declaration: a name, an optional parameter
	// clause, then a built-in kind token.
	if p.atUnitTypeMember() {
		return p.parseUnitTypeMember(annotations)
	}

	member := p.parseDecoratedFunctionDeclaration(annotations)
	p.validateOperatorOwnership(member, p.companionOwner(), "unit")
	return member
}

// atUnitTypeMember reports whether the cursor begins the `co.lang.data` or
// `co.lang.type` family member of a unit body.
func (p *parser) atUnitTypeMember() bool {
	if !p.atIdentifier() {
		return false
	}
	return p.lookaheadOnly(func() bool {
		p.advance() // the declared name
		if p.at(scanlex.OPEN_PAREN) && p.looksLikeGenericParameterClause() {
			p.skipBalanced(scanlex.OPEN_PAREN, scanlex.CLOSE_PAREN)
		}
		if !p.at(scanlex.BUILT_IN_KIND) {
			return false
		}
		if p.lexeme() == "co.lang.data" {
			return true
		}
		_, isTypeKind := typeDeclarationKinds[p.lexeme()]
		return isTypeKind
	})
}

// parseUnitTypeMember parses the data-declaration and type-declaration members of
// a unit body. Both keep an explicit identifier in the head, and both are the
// declaration forms DECISION-GEN-001 still allows a declaration-head parameter
// clause.
func (p *parser) parseUnitTypeMember(annotations annotationSet) ast.Stmt {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	p.rejectOperatorPlacement(annotations, "a unit type declaration")

	declName := p.parseIdentifier("as a unit type declaration name")
	clauseTok := p.cur()
	generics := p.parseOptionalGenericParameterClause()
	kindTok := p.expect(scanlex.BUILT_IN_KIND, "to declare a unit type member's kind")

	if len(generics) != 0 && kindTok.Value != "co.lang.type" && kindTok.Value != "co.lang.data" {
		p.failf(clauseTok,
			"%q does not take declaration-head type parameters; only a parameterized co.lang.type or co.lang.data declaration does",
			kindTok.Value)
	}
	return p.dispatchKindDeclaration(declName, generics, kindTok, annotations)
}

// companionOwner returns the struct a companion unit's members attach to.
//
// DECISION-COMP-001: companion ownership ALWAYS comes from the filename. An
// ordinary unit fragment owns nothing — its members merge into the package
// namespace — so it yields an empty owner and only a built-in operator extension
// is legal there.
func (p *parser) companionOwner() name {
	if p.file.Source.Class != sourceClassCompanionUnit {
		return name{}
	}
	owner := p.file.Source.DerivedName
	return name{Scanned: owner + foLoweringSuffix, Logical: owner}
}

// module-declaration.
//
//	module-declaration = annotations, declaration-name,
//	                     [ generic-parameter-clause ], "co.lang.module",
//	                     [ kind-options ], "=", module-body
//	module-body        = "{", { module-member }, body-close
//	module-member      = variable-declaration
//	                   | inferred-variable-declaration
//	                   | function-declaration
//	                   | signature-type-component
//
// A module is a singleton container of state and behaviour that may implement a signature
// (docs/language-ref.md, "Modules"). The signature it satisfies is named either in the kind
// options — `->(matches=Sig)` or `->(signature=Sig)` — or in a @co.dap.module annotation.

// parseModuleDeclaration parses the module-declaration production.
//
// Implements: module-declaration
func (p *parser) parseModuleDeclaration(declName name, annotations annotationSet) ast.Stmt {
	spanStart := p.pos
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	options := p.parseOptionalKindOptions()

	p.expectOp("=", "before a module body")
	members := p.parseBracedBody("a module body", p.parseModuleMember)

	symb := p.moduleSymbol(declName.Scanned)
	applyTypeVisibility(&symb.SymbolDetails, annotations)

	decl := ast.ModuleStmt{Span: p.spanFrom(spanStart), Body: members,
		Extensions: optionNames(options, "extensions"),
		Uses:       optionNames(options, "uses"),
		SDapst:     annotations.list(),
		Symb:       symb,
	}
	decl.MatchesSig = firstOptionString(options, "matches")
	decl.SignatureName = firstOptionString(options, "signature")
	if decl.SignatureName == "" {
		decl.SignatureName = annotations.optionString("@co.dap.module", "signature")
	}
	return decl
}

// parseModuleMember parses the module-member production.
//
// Implements: module-member
func (p *parser) parseModuleMember() ast.Stmt {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	annotations := p.parseAnnotations()

	switch {
	case p.atSignatureTypeComponent():
		p.rejectOperatorPlacement(annotations, "a module type component")
		return p.parseSignatureTypeComponent(annotations)
	case p.atMemberFunctionDeclaration():
		p.rejectOperatorPlacement(annotations, "a module")
		return p.parseFunctionDeclaration(annotations)
	case p.atInferredVariableDeclaration():
		p.rejectOperatorPlacement(annotations, "a module variable")
		return p.parseInferredVariableDeclaration(annotations)
	default:
		p.rejectOperatorPlacement(annotations, "a module variable")
		return p.parseVariableDeclaration(annotations)
	}
}

// object-declaration.
//
//	object-declaration = annotations, declaration-name,
//	                     [ generic-parameter-clause ], "co.lang.object",
//	                     [ kind-options ], "=", object-body
//	object-body        = "{", { field-declaration | function-declaration }, body-close
//
// An object is a single named instance. Its `for=` option marks the objects that define an
// annotation, a directive or a pragma, which is how user-defined metadata is declared.

// parseObjectDeclaration parses the object-declaration production.
//
// Implements: object-declaration
func (p *parser) parseObjectDeclaration(declName name, annotations annotationSet) ast.Stmt {
	spanStart := p.pos
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	options := p.parseOptionalKindOptions()

	p.expectOp("=", "before an object body")
	members := p.parseBracedBody("an object body", func() ast.Stmt {
		memberAnnotations := p.parseAnnotations()
		if p.atMemberFunctionDeclaration() {
			p.rejectOperatorPlacement(memberAnnotations, "an object")
			return p.parseFunctionDeclaration(memberAnnotations)
		}
		p.rejectOperatorPlacement(memberAnnotations, "an object field")
		return p.parseFieldDeclaration(memberAnnotations)
	})

	symb := p.objectSymbol(declName.Scanned)
	symb.ObjectFor = firstOptionString(options, "for")

	return ast.ObjectDeclStmt{Span: p.spanFrom(spanStart), Name: declName.Scanned,
		Body:      members,
		Kind:      "co.lang.object",
		ObjectFor: symb.ObjectFor,
		SDapst:    annotations.list(),
		KDapst:    annotations.list(),
		Symb:      symb,
	}
}

// instance-declaration and matcher-instance-declaration.
//
//	instance-declaration         = annotations, filename-derived-name,
//	                               "co.lang.instance", [ kind-options ], "=",
//	                               instance-body
//	instance-body                = "{", { function-declaration
//	                                     | variable-declaration }, body-close
//	matcher-instance-declaration = annotations, filename-derived-name,
//	                               "co.lang.matcher", matcher-options, "=",
//	                               matcher-body
//	matcher-options              = "->", "(", "type", annotation-binder,
//	                               type-expression, ")"
//	matcher-body                 = "{", { function-declaration }, body-close
//
// Only the lowercase kind is recognised: docs/language-ref.md, "Builtin Kinds"
// lists co.lang.matcher alone, and revision 26 of the grammar dropped the
// alternation with the capitalized spelling that never had a token.
//
// An instance implements a typeclass for a type, which the `for=` and `type=` options name
// (docs/language-ref.md, "Type Classes"):
//
//	// ListFunctor.fol
//	_ co.lang.instance->(for=Functor, type=List) = {
//	    map(value List(A), f (A)->B) -> (List(B)) = { … }
//	}
//
// A matcher is close in shape but its options are NOT optional and NOT open
// (DECISION-MATCH-001). It declares exactly one matched-subject type, and its
// body holds exactly one protocol function named matchCase
// (docs/language-ref.md, "Custom Matcher"):
//
//	// PositiveEvenMatcher.fol
//	@co.dap.matcher
//	_ co.lang.matcher->(type=co.lang.int) = {
//	    matchCase(value co.lang.int, pattern co.lang.untyped)
//	        ->(co.lang.int, co.lang.MatchBindings) = { … }
//	}
//
// Comparing the declared subject type with the resolved first parameter type is a
// semantic check; the parser owns the shape — one `type=` option, one matchCase,
// and functions only in the body.

// parseInstanceDeclaration parses the instance-declaration production.
//
// Implements: instance-declaration
func (p *parser) parseInstanceDeclaration(declName name, annotations annotationSet) ast.Stmt {
	spanStart := p.pos
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	options := p.parseOptionalKindOptions()

	p.expectOp("=", "before an instance body")
	members := p.parseBracedBody("an instance body", p.parseInstanceMember)

	typeclassName := firstOptionString(options, "for")
	forType := firstOptionString(options, "type")

	return ast.TypeclassInstanceStmt{Span: p.spanFrom(spanStart), TypeclassName: typeclassName,
		ForType:  forType,
		TypeArgs: optionNames(options, "typeargs"),
		Body:     members,
		SDapst:   annotations.list(),
		Symb:     p.instanceSymbol(declName.Scanned),
	}
}

// parseMatcherInstanceDeclaration parses the matcher-instance-declaration production.
//
// Implements: matcher-instance-declaration
func (p *parser) parseMatcherInstanceDeclaration(declName name, annotations annotationSet) ast.Stmt {
	spanStart := p.pos
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	subject := p.parseMatcherOptions()

	p.expectOp("=", "before a matcher body")
	members := p.parseBracedBody("a matcher body", p.parseMatcherMember)
	p.validateMatcherProtocol(declName, members)

	return ast.MatcherInstanceStmt{Span: p.spanFrom(spanStart), MatcherName: declName.Logical,
		ForType: subject,
		Body:    members,
		SDapst:  annotations.list(),
		Symb:    p.matcherImplSymbol(declName.Scanned),
	}
}

// parseMatcherOptions parses the matcher-options production and returns the one
// declared subject type.
//
// Unlike kind-options this clause is required and closed: DECISION-MATCH-001
// gives a matcher exactly one `type=` entry, so a missing clause, a second
// entry, or any other key is a grammar error rather than an unknown option the
// semantic phase would have to interpret.
//
// Implements: matcher-options
func (p *parser) parseMatcherOptions() string {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	if !p.at(scanlex.ARROW) {
		p.failf(p.cur(), "a matcher declares its one matched-subject type, as in \"co.lang.matcher->(type=co.lang.int)\"")
	}
	options := p.parseKindOptions()

	for key := range options {
		if key != "type" {
			p.failf(p.cur(), "a matcher takes only the \"type\" option; %q is not allowed", key)
		}
	}
	subject := firstOptionString(options, "type")
	if subject == "" {
		p.failf(p.cur(), "a matcher requires exactly one matched-subject type, written \"type=<type>\"")
	}
	if len(optionNames(options, "type")) != 1 {
		p.failf(p.cur(), "a matcher supports exactly one matched-subject type; declare a separate matcher in its own <Name>.fol file for another type")
	}
	return subject
}

// parseMatcherMember parses one member of a matcher-body, which admits function
// declarations only. An instance body also takes variables; a matcher body does
// not, because it declares a protocol rather than holding state.
func (p *parser) parseMatcherMember() ast.Stmt {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	annotations := p.parseAnnotations()
	p.rejectOperatorPlacement(annotations, "a matcher")
	if !p.atMemberFunctionDeclaration() {
		p.failf(p.cur(), "a matcher body holds function declarations only; found %s", describeToken(p.cur()))
	}
	return p.parseFunctionDeclaration(annotations)
}

// matcherProtocolFunction is the one function name a matcher body must declare.
const matcherProtocolFunction = "matchCase"

// validateMatcherProtocol enforces the cardinality half of DECISION-MATCH-001:
// exactly one matchCase, never overloaded. Whether its parameter and result
// types match the declared subject is a semantic check, because it needs type
// resolution; how MANY there are is decidable here.
func (p *parser) validateMatcherProtocol(declName name, members []ast.Stmt) {
	declared := 0
	for _, member := range members {
		function, ok := member.(ast.FunctionDeclarationStmt)
		if ok && logicalName(function.Name) == matcherProtocolFunction {
			declared++
		}
	}

	switch {
	case declared == 0:
		p.reportf(declName.Tok, "matcher %q must declare one protocol function named %q", declName.Logical, matcherProtocolFunction)
	case declared > 1:
		p.reportf(declName.Tok, "matcher %q declares %q %d times; overloading the matcher protocol function is not permitted", declName.Logical, matcherProtocolFunction, declared)
	}
}

// parseInstanceMember parses one member of an instance body, which may be a method
// implementation or a variable declaration.
func (p *parser) parseInstanceMember() ast.Stmt {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	annotations := p.parseAnnotations()

	if p.atMemberFunctionDeclaration() {
		p.rejectOperatorPlacement(annotations, "an instance")
		return p.parseFunctionDeclaration(annotations)
	}
	if p.atInferredVariableDeclaration() {
		p.rejectOperatorPlacement(annotations, "an instance variable")
		return p.parseInferredVariableDeclaration(annotations)
	}
	p.rejectOperatorPlacement(annotations, "an instance variable")
	return p.parseVariableDeclaration(annotations)
}

// typeclass-declaration and annotated-contract-declaration.
//
//	typeclass-declaration          = annotations, filename-derived-name,
//	                                 typeclass-parameter-clause,
//	                                 "co.lang.typeclass", "=", contract-body
//	typeclass-parameter-clause     = generic-parameter-clause
//	annotated-contract-declaration = one-or-more-annotations,
//	                                 filename-derived-name, "=", contract-body
//	contract-body                  = "{", { function-specification
//	                                       | value-specification }, body-close
//
// Revision 23 gave the typeclass a kind token and a dedicated production. It had
// been a general kind, which left `_ (F(_)) co.lang.typeclass` sharing a shape
// with the ordinary declaration-head generic clause DECISION-GEN-001 has since
// removed everywhere else (docs/language-ref.md, "Type Classes"):
//
//	// Functor.fol
//	@co.dap.typeclass(kind=Functor)
//	_ (F(_)) co.lang.typeclass = {
//	    map(value F(A), f (A)->B) -> (F(B));
//	}
//
// `_` and the parameter clause are separate grammar components, which is why the
// canonical spelling has a space between them. A parameter such as `T` denotes an
// ordinary type; `F(_)` and `G(_, _)` declare unary and binary type constructors
// through generic-arity-clause (DECISION-TCLASS-001).
//
// annotated-contract-declaration remains for the declaration forms whose kind
// comes from an annotation rather than a co.lang.* token. It shares contract-body
// but takes no parameter clause of its own.

// parseTypeclassDeclaration parses the typeclass-declaration production.
//
// Implements: typeclass-declaration
func (p *parser) parseTypeclassDeclaration(declName name, params []symboltable.GenericTypeParam, annotations annotationSet) ast.Stmt {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	p.expectOp("=", "before a typeclass body")
	return p.finishContractDeclaration(declName, params, annotations)
}

// parseTypeclassParameterClause parses the typeclass-parameter-clause production.
//
// It is spelled as generic-parameter-clause, so the arity slots that make a
// parameter higher-kinded are already handled there. The production exists under
// its own name because a typeclass head is now the ONLY primary declaration that
// carries one.
//
// Implements: typeclass-parameter-clause
func (p *parser) parseTypeclassParameterClause() []symboltable.GenericTypeParam {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	return p.parseGenericParameterClause()
}

// parseAnnotatedContractDeclaration parses the annotated-contract-declaration production.
//
// Implements: annotated-contract-declaration
func (p *parser) parseAnnotatedContractDeclaration(declName name, annotations annotationSet) ast.Stmt {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	p.expectOp("=", "before a contract body")
	return p.finishContractDeclaration(declName, nil, annotations)
}

// finishContractDeclaration parses the contract-body the two contract-shaped
// declarations share, from just after their "=".
//
// Implements: contract-body
func (p *parser) finishContractDeclaration(declName name, params []symboltable.GenericTypeParam, annotations annotationSet) ast.Stmt {
	spanStart := p.pos
	members := p.parseBracedBody("a contract body", func() ast.Stmt {
		memberAnnotations := p.parseAnnotations()
		p.rejectOperatorPlacement(memberAnnotations, "a typeclass or annotated contract")
		if p.atMemberFunctionDeclaration() {
			return p.parseFunctionSpecification(memberAnnotations)
		}
		return p.parseValueSpecification(memberAnnotations)
	})

	symb := p.typeclassSymbol(declName.Scanned)
	applyTypeclassKind(symb, annotations)

	return ast.TypeclassStmt{Span: p.spanFrom(spanStart), Name: declName.Scanned,
		TypeParams: params,
		Methods:    members,
		Kind:       typeclassKindOf(annotations),
		SDapst:     annotations.list(),
		Symb:       symb,
	}
}

// typeclassKindNames maps a typeclass annotation to the kind name recorded on the node.
//
// `@co.dap.typeclass(kind=...)` is the single annotation the reference now
// documents for every typeclass definition, and its `kind` argument names the
// algebraic structure. The dedicated per-structure spellings are retained because
// the reference's own examples still use them and they name the same kinds.
var typeclassKindNames = map[string]string{
	"@co.dap.Functor":     "functor",
	"@co.dap.Applicative": "applicative",
	"@co.dap.Monad":       "monad",
	"@co.dap.Monoid":      "monoid",
	"@co.dap.Transformer": "transformer",
	"@co.dap.Foldable":    "foldable",
	"@co.dap.Traversable": "traversable",
	"@co.dap.typeclass":   "typeclass",
}

// typeclassKindOf returns the typeclass kind named by a declaration's annotations.
//
// `@co.dap.typeclass(kind=Functor)` names its structure in an argument, so that
// spelling is preferred over the annotation name itself; a user-defined kind
// passes through unchanged.
func typeclassKindOf(annotations annotationSet) string {
	if kind := annotations.optionString("@co.dap.typeclass", "kind"); kind != "" {
		return strings.ToLower(logicalName(kind))
	}
	for _, d := range annotations.all {
		if kind, ok := typeclassKindNames[d.Name]; ok {
			return kind
		}
	}
	return "contract"
}

// applyTypeclassKind sets the symbol flag matching the declaration's typeclass kind.
func applyTypeclassKind(symb *symboltable.TypeclassSymbol, annotations annotationSet) {
	switch typeclassKindOf(annotations) {
	case "functor":
		symb.ISFunctor = true
	case "applicative":
		symb.ISApplicative = true
	case "monad":
		symb.ISMonad = true
	case "monoid":
		symb.ISMonoid = true
	case "transformer":
		symb.ISTransormer = true
	case "foldable":
		symb.ISFoldeable = true
	}
}

// named-block-declaration.
//
//	named-block-declaration = annotations, declaration-name,
//	                          [ generic-parameter-clause ], "co.lang.block", "=",
//	                          block, body-closure-guard
//
// A named block is a block bound to a name, which can then be expanded at a call site
// (docs/language-ref.md, "Labels and Named Blocks"):
//
//	labelBlock co.lang.block={
//	}
//	labelBlock.expand();

// parseNamedBlockDeclaration parses the named-block-declaration production.
//
// Implements: named-block-declaration
func (p *parser) parseNamedBlockDeclaration(declName name, annotations annotationSet) ast.Stmt {
	spanStart := p.pos
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	p.expectOp("=", "before a named block body")

	block := p.parseBlock("a named block body")
	p.bodyClosureGuard("a named block")

	return &ast.BlockStmt{Span: p.spanFrom(spanStart), Body: statementsOf(block),
		Dapst: annotations.list(),
		Symb:  p.blockSymbol(declName.Scanned, true),
	}
}

// delegate-declaration.
//
//	delegate-declaration = annotations, declaration-name,
//	                       [ generic-parameter-clause ], "co.lang.delegate", "=",
//	                       function-type, statement-end
//
// A delegate names a function signature so it can be used as a type
// (docs/language-ref.md, "Function Delegates"):
//
//	@co.dap.delegate someDelegate co.lang.delegate =
//	    (a co.lang.int, b co.lang.int)->(co.lang.int, co.lang.int);

// parseDelegateDeclaration parses the delegate-declaration production.
//
// Implements: delegate-declaration
func (p *parser) parseDelegateDeclaration(declName name, annotations annotationSet) ast.Stmt {
	spanStart := p.pos
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	p.expectOp("=", "before a delegate signature")

	signature := p.parseFunctionType()
	p.statementEnd("a delegate declaration")

	return ast.DelegateStmt{Span: p.spanFrom(spanStart), Type_: ast.TypeStmt{Span: p.spanFrom(spanStart), Type_: signature,
		Symb: p.typeSymbol(declName.Scanned),
	},
		SDapst: annotations.list(),
		Symb:   p.delegateSymbol(declName.Scanned),
	}
}

// firstOptionString reads a kind option as a single string, taking the first entry when the
// option was written as a list.
func firstOptionString(options map[string]any, key string) string {
	names := optionNames(options, key)
	if len(names) == 0 {
		return ""
	}
	return names[0]
}

// atBuiltinKind reports whether the cursor holds the named built-in kind token.
func (p *parser) atBuiltinKind(kind string) bool {
	return p.at(scanlex.BUILT_IN_KIND) && p.lexeme() == kind
}
