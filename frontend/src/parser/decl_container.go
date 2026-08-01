package parser

import (
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
//	unit-declaration = annotations, declaration-name, "co.lang.unit", "=", unit-body
//	unit-body        = "{", { function-declaration }, body-close
//
// A unit is the container FoLang requires functions to live in: the language does not
// allow free-flowing functions in a package source file, so a unit is what groups them
// (docs/language-ref.md, "Functions"):
//
//	General co.lang.unit = {
//	    fun1(k co.lang.int)->(co.lang.int) = { … }
//	}

// parseUnitDeclaration parses the unit-declaration production.
func (p *parser) parseUnitDeclaration(declName name, annotations annotationSet) ast.Stmt {
	p.expectOp("=", "before a unit body")

	members := p.parseBracedBody("a unit body", func() ast.Stmt {
		memberAnnotations := p.parseAnnotations()
		member := p.parseDecoratedFunctionDeclaration(memberAnnotations)
		p.validateOperatorOwnership(member, declName, "unit")
		return member
	})

	return ast.TypeDeclarationStmt{
		Name:     declName.Scanned,
		Body:     members,
		Kind:     "co.lang.unit",
		SubType_: "UNIT",
		Typetype: "UDT",
		SDapst:   annotations.list(),
		KDapst:   annotations.list(),
		Symb:     p.typeSymbol(declName.Scanned),
	}
}

// module-declaration.
//
//	module-declaration = annotations, declaration-name, "co.lang.module",
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
func (p *parser) parseModuleDeclaration(declName name, annotations annotationSet) ast.Stmt {
	options := p.parseOptionalKindOptions()

	p.expectOp("=", "before a module body")
	members := p.parseBracedBody("a module body", p.parseModuleMember)

	symb := p.moduleSymbol(declName.Scanned)
	applyTypeVisibility(&symb.SymbolDetails, annotations)

	decl := ast.ModuleStmt{
		Body:       members,
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
func (p *parser) parseModuleMember() ast.Stmt {
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
//	object-declaration = annotations, declaration-name, "co.lang.object",
//	                     [ kind-options ], "=", object-body
//	object-body        = "{", { field-declaration | function-declaration }, body-close
//
// An object is a single named instance. Its `for=` option marks the objects that define an
// annotation, a directive or a pragma, which is how user-defined metadata is declared.

// parseObjectDeclaration parses the object-declaration production.
func (p *parser) parseObjectDeclaration(declName name, annotations annotationSet) ast.Stmt {
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

	return ast.ObjectDeclStmt{
		Name:      declName.Scanned,
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
//	instance-declaration         = annotations, declaration-name, "co.lang.instance",
//	                               [ kind-options ], "=", instance-body
//	instance-body                = "{", { function-declaration
//	                                     | variable-declaration }, body-close
//	matcher-instance-declaration = annotations, declaration-name,
//	                               ( "co.lang.Matcher" | "co.lang.matcher" ),
//	                               [ kind-options ], "=", instance-body
//
// An instance implements a typeclass for a type, which the `for=` and `type=` options name
// (docs/language-ref.md, "Type Classes"):
//
//	ListFunctor co.lang.instance->(for=Functor, type=List) = {
//	    map(value List(A), f (A)->B) -> (List(B)) = { … }
//	}
//
// A matcher instance is the same shape for a custom pattern matcher
// (docs/language-ref.md, "Custom Matcher").

// parseInstanceDeclaration parses the instance-declaration production.
func (p *parser) parseInstanceDeclaration(declName name, annotations annotationSet) ast.Stmt {
	options := p.parseOptionalKindOptions()

	p.expectOp("=", "before an instance body")
	members := p.parseBracedBody("an instance body", p.parseInstanceMember)

	typeclassName := firstOptionString(options, "for")
	forType := firstOptionString(options, "type")

	return ast.TypeclassInstanceStmt{
		TypeclassName: typeclassName,
		ForType:       forType,
		TypeArgs:      optionNames(options, "typeargs"),
		Body:          members,
		SDapst:        annotations.list(),
		Symb:          p.instanceSymbol(declName.Scanned),
	}
}

// parseMatcherInstanceDeclaration parses the matcher-instance-declaration production.
func (p *parser) parseMatcherInstanceDeclaration(declName name, annotations annotationSet) ast.Stmt {
	options := p.parseOptionalKindOptions()

	p.expectOp("=", "before a matcher instance body")
	members := p.parseBracedBody("a matcher instance body", p.parseInstanceMember)

	return ast.MatcherInstanceStmt{
		MatcherName: firstOptionString(options, "for"),
		ForType:     firstOptionString(options, "type"),
		Body:        members,
		SDapst:      annotations.list(),
		Symb:        p.matcherImplSymbol(declName.Scanned),
	}
}

// parseInstanceMember parses one member of an instance body, which may be a method
// implementation or a variable declaration.
func (p *parser) parseInstanceMember() ast.Stmt {
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

// annotated-contract-declaration.
//
//	annotated-contract-declaration = one-or-more-annotations, declaration-name,
//	                                 [ generic-parameter-clause ], "=", contract-body
//	contract-body                  = "{", { function-specification
//	                                       | value-specification }, body-close
//
// This is the declaration form that has NO kind token: the annotation supplies the kind
// instead. It is what a typeclass declaration uses
// (docs/language-ref.md, "Type Classes"):
//
//	@co.dap.Functor
//	Functor(F) = {
//	    map(value F(A), f (A)->B) -> (F(B));
//	}

// parseAnnotatedContractDeclaration parses the annotated-contract-declaration production.
func (p *parser) parseAnnotatedContractDeclaration(declName name, generics []symboltable.GenericTypeParam, annotations annotationSet) ast.Stmt {
	p.expectOp("=", "before a contract body")

	members := p.parseBracedBody("a contract body", func() ast.Stmt {
		memberAnnotations := p.parseAnnotations()
		p.rejectOperatorPlacement(memberAnnotations, "an annotated contract")
		if p.atMemberFunctionDeclaration() {
			return p.parseFunctionSpecification(memberAnnotations)
		}
		return p.parseValueSpecification(memberAnnotations)
	})

	typeParams := make([]string, 0, len(generics))
	for _, g := range generics {
		typeParams = append(typeParams, g.Name)
	}

	symb := p.typeclassSymbol(declName.Scanned)
	applyTypeclassKind(symb, annotations)

	return ast.TypeclassStmt{
		Name:       declName.Scanned,
		TypeParams: typeParams,
		Methods:    members,
		Kind:       typeclassKindOf(annotations),
		SDapst:     annotations.list(),
		Symb:       symb,
	}
}

// typeclassKindNames maps a typeclass annotation to the kind name recorded on the node.
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
func typeclassKindOf(annotations annotationSet) string {
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
//	named-block-declaration = annotations, declaration-name, "co.lang.block", "=",
//	                          block, body-closure-guard
//
// A named block is a block bound to a name, which can then be expanded at a call site
// (docs/language-ref.md, "Labels and Named Blocks"):
//
//	labelBlock co.lang.block={
//	}
//	labelBlock.expand();

// parseNamedBlockDeclaration parses the named-block-declaration production.
func (p *parser) parseNamedBlockDeclaration(declName name, annotations annotationSet) ast.Stmt {
	p.expectOp("=", "before a named block body")

	block := p.parseBlock("a named block body")
	p.bodyClosureGuard("a named block")

	return &ast.BlockStmt{
		Body:  statementsOf(block),
		Dapst: annotations.list(),
		Symb:  p.blockSymbol(declName.Scanned, true),
	}
}

// delegate-declaration.
//
//	delegate-declaration = annotations, declaration-name, "co.lang.delegate", "=",
//	                       function-type, statement-end
//
// A delegate names a function signature so it can be used as a type
// (docs/language-ref.md, "Function Delegates"):
//
//	@co.dap.delegate someDelegate co.lang.delegate =
//	    (a co.lang.int, b co.lang.int)->(co.lang.int, co.lang.int);

// parseDelegateDeclaration parses the delegate-declaration production.
func (p *parser) parseDelegateDeclaration(declName name, annotations annotationSet) ast.Stmt {
	p.expectOp("=", "before a delegate signature")

	signature := p.parseFunctionType()
	p.statementEnd("a delegate declaration")

	return ast.DelegateStmt{
		Type_: ast.TypeStmt{
			Type_: signature,
			Symb:  p.typeSymbol(declName.Scanned),
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
