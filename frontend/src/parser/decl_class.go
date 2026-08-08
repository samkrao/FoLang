package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// class-declaration — section 7.
//
//	class-declaration            = annotations, declaration-name,
//	                               [ generic-parameter-clause ], "co.lang.class",
//	                               [ kind-options ], "=", class-body
//	class-body                   = "{", { class-member }, body-close
//	class-member                 = field-declaration
//	                             | function-declaration
//	                             | lifecycle-method-declaration
//	lifecycle-method-declaration = annotations, lifecycle-name,
//	                               parameter-list, [ return-type-clause ],
//	                               function-definition
//
// A class body mixes state and behaviour, so its member loop has to decide between a
// field and a method on every iteration. The kind options carry the class's
// relationships — extends, implements, uses and so on (docs/language-ref.md, "Class
// Declaration Relationships").
//
// The lifecycle methods are @@new and @@init (docs/language-ref.md, "The @@new and @@init
// Methods"); they are spelled with the "@@" prefix, which is what distinguishes them
// from ordinary methods.

// parseClassDeclaration parses the class-declaration production.
//
// Implements: class-declaration
func (p *parser) parseClassDeclaration(declName name, annotations annotationSet) ast.Stmt {
	spanStart := p.pos
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	options := p.parseOptionalKindOptions()

	p.expectOp("=", "before a class body")
	members := p.parseBracedBody("a class body", func() ast.Stmt {
		return p.parseClassMember(&declName)
	})

	symb := p.classSymbol(declName.Scanned)
	symb.IsGeneric = annotations.has("@co.dap.generic")
	symb.Abstract = annotations.has("@co.dap.abstract")
	symb.Virtual = annotations.has("@co.dap.virtual")
	symb.IsSealed = annotations.has("@co.dap.sealed")
	symb.Property = annotations.has("@co.dap.property")
	applyClassRelationships(symb, options)
	applyTypeVisibility(&symb.SymbolDetails, annotations)

	return ast.ClassDeclarationStmt{Span: p.spanFrom(spanStart), Name: declName.Scanned,
		Body:   members,
		SDapst: annotations.list(),
		Symb:   symb,
	}
}

// parseClassMember parses the class-member production.
//
// The three alternatives are separated by their leading tokens: "@@" begins a lifecycle
// method, a name followed by "(" begins a method, and anything else is a field.
//
// Implements: class-member
func (p *parser) parseClassMember(owner *name) ast.Stmt {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	annotations := p.parseAnnotations()

	switch {
	case p.atLifecycleName():
		p.rejectOperatorPlacement(annotations, "a class lifecycle method")
		return p.parseLifecycleMethodDeclaration(annotations)
	case p.atMemberFunctionDeclaration():
		if owner == nil {
			p.rejectOperatorPlacement(annotations, "an anonymous class")
		}
		categoriesValid := p.validateClassMethodCategories(annotations)
		member := p.parseDecoratedFunctionDeclaration(annotations)
		p.markClassMethod(member)
		if owner == nil {
			return member
		}
		if categoriesValid {
			p.validateOperatorOwnership(member, *owner, "class")
		}
		return member
	default:
		p.rejectOperatorPlacement(annotations, "a class field")
		return p.parseFieldDeclaration(annotations)
	}
}

// validateClassMethodCategories enforces the mutually exclusive class-method
// categories before operator ownership derives an implicit `this` operand.
// @co.dap.class and its @co.dap.method.class spelling denote the same category,
// so using both is a duplicate rather than two independent categories.
func (p *parser) validateClassMethodCategories(annotations annotationSet) bool {
	seen := map[string]string{}
	firstCategory := ""
	firstAnnotation := ""
	valid := true

	for _, annotation := range annotations.all {
		category := classMethodCategory(annotation.Name)
		if category == "" {
			continue
		}
		if previous, duplicate := seen[category]; duplicate {
			p.reportf(
				p.cur(),
				"class method category %q is declared more than once by %s and %s",
				category,
				previous,
				annotation.Name,
			)
			valid = false
			continue
		}

		seen[category] = annotation.Name
		if firstCategory == "" {
			firstCategory = category
			firstAnnotation = annotation.Name
			continue
		}

		p.reportf(
			p.cur(),
			"class method categories %s and %s are mutually exclusive",
			firstAnnotation,
			annotation.Name,
		)
		valid = false
	}

	return valid
}

// classMethodCategory normalizes the annotations that choose how a class
// method is associated with its enclosing class.
func classMethodCategory(annotation string) string {
	switch annotation {
	case "@co.dap.static":
		return "static"
	case "@co.dap.class", "@co.dap.method.class":
		return "class"
	case "@co.dap.instance":
		return "instance"
	case "@co.dap.object":
		return "object"
	default:
		return ""
	}
}

// markClassMethod adds the method category supplied by the enclosing class.
// applyFunctionFlags handles explicit receiver and annotation metadata, but an
// ordinary class member has an implicit `this` receiver and therefore cannot be
// identified from its declaration shape alone. Unless an explicit static,
// class, object, or instance category was supplied, a receiverless class member
// is an instance method; a bare type receiver is class-associated.
func (p *parser) markClassMethod(stmt ast.Stmt) {
	function, ok := functionDeclarationOf(stmt)
	if !ok || function.Symb == nil {
		return
	}

	function.Symb.IsMethod = true
	if function.Symb.StaticMethod || function.Symb.ClassMethod ||
		function.Symb.InstanceMethod || function.Symb.ObjectMethod {
		return
	}

	if function.AssociatedReceiver != nil && instanceReceiverType(function.AssociatedReceiver) == nil {
		function.Symb.ClassMethod = true
		return
	}
	function.Symb.InstanceMethod = true
}

// functionDeclarationOf unwraps the function-shaped statement variants emitted
// by parseDecoratedFunctionDeclaration. The declaration is returned by value,
// but its symbol is a shared pointer, which is the metadata class ownership must
// update without changing the wrapper's structural identity.
func functionDeclarationOf(stmt ast.Stmt) (ast.FunctionDeclarationStmt, bool) {
	switch stmt := stmt.(type) {
	case ast.FunctionDeclarationStmt:
		return stmt, true
	case ast.MacroStmt:
		return stmt.FunctionDeclarationStmt, true
	case ast.TemplateStmt:
		return stmt.FunctionDeclarationStmt, true
	case ast.OperatorStmt:
		return stmt.FunctionDeclarationStmt, true
	case ast.ExtensionStmt:
		return stmt.FunctionDeclarationStmt, true
	case ast.IndexerStmt:
		return stmt.FunctionDeclarationStmt, true
	case ast.MatcherStmt:
		return stmt.FunctionDeclarationStmt, true
	case ast.GenerricFun:
		return stmt.FunctionDeclarationStmt, true
	case ast.DDapStmt:
		return stmt.FunctionDeclarationStmt, true
	default:
		return ast.FunctionDeclarationStmt{}, false
	}
}

// parseClassMembers reads a class body's members without the surrounding braces, which is
// what the anonymous class expression needs.
func (p *parser) parseClassMembers() []ast.Stmt {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	return p.parseMemberList("an anonymous class body", func() ast.Stmt {
		return p.parseClassMember(nil)
	})
}

// atMemberFunctionDeclaration reports whether the cursor begins a function declaration
// inside a declaration body.
//
// A method is a name followed by a parameter list, possibly preceded by a receiver clause.
// A field is a name followed by a type, so the "(" is the discriminator.
func (p *parser) atMemberFunctionDeclaration() bool {
	if p.atReceiverClause() {
		return true
	}
	if !p.atIdentifier() && !p.atLifecycleName() {
		return false
	}
	return p.lookaheadOnly(func() bool {
		p.advance() // the name
		return p.at(scanlex.OPEN_PAREN)
	})
}

// parseLifecycleMethodDeclaration parses the lifecycle-method-declaration production.
//
// A lifecycle method always has a block body, so unlike an ordinary function declaration
// it has no forward, delegation or alias form.
//
// Implements: lifecycle-method-declaration
func (p *parser) parseLifecycleMethodDeclaration(annotations annotationSet) ast.Stmt {
	spanStart := p.pos
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	methodName := p.parseLifecycleName()
	params := p.parseParameterList()

	var results []ast.Returns
	if p.at(scanlex.ARROW) {
		results = p.parseReturnTypeClause()
	}

	symb := p.functionSymbol(methodName.Scanned)
	symb.IsMethod = true
	symb.ClassMethod = true

	decl := ast.FunctionDeclarationStmt{Span: p.spanFrom(spanStart), Parameters: [][]ast.Parameter{params},
		Name:       methodName.Scanned,
		ReturnType: results,
		Dapst:      annotations.list(),
		Symb:       symb,
	}
	p.applyFunctionFlags(&decl, annotations)
	decl.Symb.IsMethod = true
	decl.Symb.ClassMethod = true

	// function-definition: the "=" is optional (DECISION-FUN-001).
	p.acceptOp("=")
	return p.finishFunctionDefinition(decl)
}

// applyClassRelationships records the relationships declared in a class's kind options.
//
// FoLang spells inheritance and composition as options rather than as clauses, so
// `co.lang.class->(extends=Base, implements=[Printable])` is where they live.
func applyClassRelationships(symb *symboltable.ClassSymbol, options map[string]any) {
	symb.Extends = optionNames(options, "extends")
	symb.Implements = optionNames(options, "implements")
	symb.Inherits = optionNames(options, "inherits")
	symb.Uses = optionNames(options, "uses")
	symb.Mixin = optionNames(options, "mixin")
	symb.Traits = optionNames(options, "traits")
	symb.Extensions = optionNames(options, "extensions")
	symb.ComposeAssociate = optionNames(options, "compose")

	if with, ok := options["with"]; ok {
		if s, isString := with.(string); isString {
			symb.With = s
		}
	}
}

// optionNames reads a kind option as a list of names.
//
// An option may be written as a single name or as a list, so both `extends=Base` and
// `implements=[A, B]` decode to a slice.
func optionNames(options map[string]any, key string) []string {
	value, ok := options[key]
	if !ok {
		return nil
	}

	switch v := value.(type) {
	case string:
		return []string{v}
	case []any:
		names := make([]string, 0, len(v))
		for _, item := range v {
			if s, isString := item.(string); isString {
				names = append(names, s)
			}
		}
		return names
	}
	return nil
}

// interface-declaration — section 7.
//
//	interface-declaration = annotations, declaration-name,
//	                        [ generic-parameter-clause ], "co.lang.interface", "=",
//	                        interface-body
//	interface-body        = "{", { function-specification }, body-close
//
// An interface body holds only function specifications — signatures with no bodies — which
// is what distinguishes it from a signature, whose body may also require values and types
// (docs/language-ref.md, "Interface vs Signature").

// parseInterfaceDeclaration parses the interface-declaration production.
//
// Implements: interface-declaration
func (p *parser) parseInterfaceDeclaration(declName name, annotations annotationSet) ast.Stmt {
	spanStart := p.pos
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	p.expectOp("=", "before an interface body")

	members := p.parseBracedBody("an interface body", func() ast.Stmt {
		memberAnnotations := p.parseAnnotations()
		p.rejectOperatorPlacement(memberAnnotations, "an interface")
		return p.parseFunctionSpecification(memberAnnotations)
	})

	// ast.TypeDeclarationStmt stores its symbol as an ITypeSymbol, which among the
	// symbol kinds only TypeSymbol satisfies, so the interface kind is recorded on the
	// type symbol rather than through a dedicated InterfaceSymbol.
	symb := p.typeSymbol(declName.Scanned)
	symb.TypeType = "co.lang.interface"
	applyTypeVisibility(&symb.SymbolDetails, annotations)

	return ast.TypeDeclarationStmt{Span: p.spanFrom(spanStart), Name: declName.Scanned,
		Body:     members,
		Kind:     "co.lang.interface",
		SubType_: "INTERFACE",
		Typetype: "UDT",
		SDapst:   annotations.list(),
		KDapst:   annotations.list(),
		Symb:     symb,
	}
}

// signature-declaration — section 7.
//
//	signature-declaration = annotations, declaration-name,
//	                        [ generic-parameter-clause ], "co.lang.signature", "=",
//	                        signature-body
//	signature-body        = "{", { signature-member }, body-close
//	signature-member      = value-specification
//	                      | function-specification
//	                      | signature-type-component
//
// A signature is a module's contract: it may require values, functions and types, which is
// strictly more than an interface can (docs/language-ref.md, "Module Signature Contents").

// parseSignatureDeclaration parses the signature-declaration production.
//
// Implements: signature-declaration
func (p *parser) parseSignatureDeclaration(declName name, annotations annotationSet) ast.Stmt {
	spanStart := p.pos
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	p.expectOp("=", "before a signature body")
	members := p.parseBracedBody("a signature body", p.parseSignatureMember)

	// As with an interface, the signature kind is recorded on a TypeSymbol because that
	// is the symbol kind ast.TypeDeclarationStmt accepts.
	symb := p.typeSymbol(declName.Scanned)
	symb.TypeType = "co.lang.signature"
	applyTypeVisibility(&symb.SymbolDetails, annotations)

	return ast.TypeDeclarationStmt{Span: p.spanFrom(spanStart), Name: declName.Scanned,
		Body:     members,
		Kind:     "co.lang.signature",
		SubType_: "SIGNATURE",
		Typetype: "UDT",
		SDapst:   annotations.list(),
		KDapst:   annotations.list(),
		Symb:     symb,
	}
}

// parseSignatureMember parses the signature-member production.
//
// The three alternatives are separated by lookahead: a name followed by "co.lang.type" is
// a type component, a name followed by "(" is a function specification, and anything else
// is a value specification.
//
// Implements: signature-member
func (p *parser) parseSignatureMember() ast.Stmt {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	annotations := p.parseAnnotations()

	switch {
	case p.atSignatureTypeComponent():
		p.rejectOperatorPlacement(annotations, "a signature type component")
		return p.parseSignatureTypeComponent(annotations)
	case p.atMemberFunctionDeclaration():
		p.rejectOperatorPlacement(annotations, "a signature")
		return p.parseFunctionSpecification(annotations)
	default:
		p.rejectOperatorPlacement(annotations, "a signature value specification")
		return p.parseValueSpecification(annotations)
	}
}

// atSignatureTypeComponent reports whether the cursor begins a
// signature-type-component, which is a name — optionally generic — followed by
// "co.lang.type".
func (p *parser) atSignatureTypeComponent() bool {
	if !p.atIdentifier() && !p.at(scanlex.DISCARD_WILD_VAR) {
		return false
	}
	return p.lookaheadOnly(func() bool {
		p.advance() // the name
		if p.at(scanlex.OPEN_PAREN) {
			p.skipBalanced(scanlex.OPEN_PAREN, scanlex.CLOSE_PAREN)
		}
		return p.at(scanlex.BUILT_IN_KIND) && p.lexeme() == "co.lang.type"
	})
}
