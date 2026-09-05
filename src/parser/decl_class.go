package parser

import (
	"github.com/samkrao/fo-lang/src/ast"
	symboltable "github.com/samkrao/fo-lang/src/context"
	"github.com/samkrao/fo-lang/src/helpers"
	"github.com/samkrao/fo-lang/src/scanlex"
)

// class-declaration — section 7.
//
//	class-declaration            = annotations, filename-derived-name,
//	                               "co.lang.class", [ kind-options ], "=", class-body,
//	                               class-lifecycle-capability-guard
//	class-body                   = "{", { class-member }, body-close
//	class-member                 = field-declaration
//	                             | function-declaration
//	                             | lifecycle-method-declaration
//	lifecycle-method-declaration = annotations, lifecycle-declaration-name,
//	                               parameter-list, [ return-type-clause ],
//	                               function-definition,
//	                               lifecycle-declaration-context-guard
//
// A class body mixes state and behaviour, so its member loop has to decide between a
// field and a method on every iteration. The kind options carry the class's
// relationships — extends, implements, uses and so on (docs/language-ref.md, "Class
// Declaration Relationships").
//
// The lifecycle members are @@new and @@init (docs/language-ref.md, "Lifecycle
// Members"); they are spelled with the "@@" prefix, which is what distinguishes
// them from ordinary methods.
//
// Every class already HAS lifecycle machinery: the compiler owns inherited @@new
// and @@init implementations for each co.lang.class, and ordinary construction
// uses them without any source declaration. What the two guards below control is
// narrower — whether source may OVERRIDE or OVERLOAD that family — and the answer
// is no unless the class is generic and its generic metadata opts in with
// `lifecycle=true`. A source `@@new` therefore never creates a lifecycle name;
// it only adds a signature to one the language already owns.

// parseClassDeclaration parses the class-declaration production.
//
// Implements: class-declaration
func (p *parser) parseClassDeclaration(declName name, annotations annotationSet) ast.Stmt {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	options := p.parseOptionalKindOptions()
	relationships := p.classRelationships(annotations)
	symb := p.classSymbol(declName.Scanned)

	p.expectOp("=", "before a class body")

	popRelationships := p.pushDirectRelationships(relationships)
	popLifecycle := p.pushLifecycleCapability(classLifecycleCapability(annotations))
	members := p.parseBracedBodyWithSetup(symboltable.S_ClassSymbol, "a class body", func() {
		p.declareGenericAnnotationTypes(annotations)
	}, func() ast.Stmt {
		return p.parseClassMember(&declName)
	}, symb)
	popLifecycle()
	popRelationships()

	symb.IsGeneric = annotations.has("@co.dap.generic")
	symb.Abstract = annotations.has("@co.dap.abstract")
	symb.Virtual = annotations.has("@co.dap.virtual")
	symb.IsSealed = annotations.has("@co.dap.sealed")
	symb.Property = annotations.has("@co.dap.property")
	applyClassRelationships(symb, options, annotations)
	applyTypeVisibility(&symb.SymbolDetails, annotations)
	p.declareNamed(declName, symb)

	return ast.ClassDeclarationStmt{NodeName: "ClassDeclarationStmt", Span: p.spanFrom(spanStart), Name: declName.Scanned,
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
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	annotations := p.parseAnnotations()

	switch {
	case p.atTypeDeclarationMember():
		p.rejectOperatorPlacement(annotations, "a class type declaration")
		return p.parseUnitKindMember(annotations)
	case p.atLifecycleName():
		p.rejectOperatorPlacement(annotations, "a class lifecycle method")
		popReceiver := p.pushThisReceiverContext()
		member := p.parseLifecycleMethodDeclaration(annotations)
		popReceiver()
		return member
	case p.atMemberFunctionDeclaration():
		if owner == nil {
			p.rejectOperatorPlacement(annotations, "an anonymous class")
		}
		categoriesValid := p.validateClassMethodCategories(annotations)
		if annotations.has("@co.dap.abstract") || annotations.has("@co.dap.virtual") {
			p.reportf(p.cur(), "@co.dap.abstract and @co.dap.virtual are permitted only on trait or mixin methods, not class methods")
			categoriesValid = false
		}
		var member ast.Stmt
		if annotations.has("@co.dap.static") {
			member = p.parseDecoratedFunctionDeclaration(annotations)
		} else {
			popReceiver := p.pushThisReceiverContext()
			member = p.parseDecoratedFunctionDeclaration(annotations)
			popReceiver()
		}
		p.markClassMethod(member)
		if owner == nil {
			return member
		}
		if categoriesValid {
			p.validateOperatorOwnership(member, *owner, "class")
		}
		return member
	default:
		p.rejectNestedKindDeclaration("a class body")
		p.rejectOperatorPlacement(annotations, "a class field")
		return p.parseClassInstanceFieldDeclaration(annotations, "class")
	}
}

// validateClassMethodCategories enforces the mutually exclusive class-method
// categories before operator ownership derives an implicit `this` operand.
//
// A method carries at most one category. Two different categories contradict
// each other, and the same category twice is a duplicate; both are reported here
// so ownership derives its receiver from one unambiguous category.
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
	case "@co.dap.class":
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

// functionDeclarationOf unwraps the function-shaped statement variants
// classifyFunctionShapedDeclaration emits. The declaration is returned by value,
// but its symbol is a shared pointer, which is the metadata class ownership must
// update without changing the wrapper's structural identity.
//
// The cases are exactly the reference's classification table plus the ordinary
// declaration, so a caller that needs the callable signature reaches it whichever
// kind owns the declaration — which is what the reference means by saying the
// specialization does not remove the callable function/method interface.
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
	case ast.IndexerStmt:
		return stmt.FunctionDeclarationStmt, true
	case ast.ExtensionStmt:
		return stmt.FunctionDeclarationStmt, true
	case ast.DecoratorStmt:
		return stmt.FunctionDeclarationStmt, true
	case ast.NativeFunctionStmt:
		return stmt.FunctionDeclarationStmt, true
	case ast.ExecutionModelFunctionStmt:
		return stmt.FunctionDeclarationStmt, true
	case ast.GenerricFun:
		return stmt.FunctionDeclarationStmt, true
	default:
		return ast.FunctionDeclarationStmt{NodeName: "FunctionDeclarationStmt"}, false
	}
}

// parseClassMembers reads a class body's members without the surrounding braces, which is
// what the anonymous class expression needs.
func (p *parser) parseClassMembers() []ast.Stmt {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	// An anonymous class expression carries no declaration metadata of its own, so
	// it can never be a generic class with `lifecycle=true`. The capability is
	// pushed as the zero value rather than inherited, because otherwise a
	// `co.lang.class { … }` written inside a lifecycle-enabled class's method would
	// pick up that class's permission.
	popLifecycle := p.pushLifecycleCapability(lifecycleCapability{inClassBody: true})
	defer popLifecycle()

	return p.parseMemberList("an anonymous class body", func() ast.Stmt {
		return p.parseClassMember(nil)
	})
}

// lifecycleCapability is what class-lifecycle-capability-guard establishes for a
// class body and lifecycle-declaration-context-guard then tests each source
// lifecycle declaration against.
//
// The three fields are kept apart rather than collapsed into one boolean so the
// diagnostic can name the actual reason a declaration is refused. "This class is
// not generic" and "this generic class did not set lifecycle=true" are different
// mistakes with different fixes, and a single `enabled` flag could report neither.
type lifecycleCapability struct {
	// inClassBody reports whether a class body is being parsed at all. Outside
	// one, a lifecycle declaration is refused for a third reason again: no
	// non-class declaration can source-declare @@new or @@init.
	inClassBody bool
	// generic reports whether the class carries @co.dap.generic with an explicit
	// non-empty types=[...] list. The list must be explicit: the guard requires
	// "valid co.dap.generic metadata with an explicit types=[...] list", so a bare
	// @co.dap.generic does not make the class generic for this purpose.
	generic bool
	// enabled reports whether that same generic metadata carries lifecycle=true.
	enabled bool
}

// classLifecycleCapability reads the lifecycle-customization permission out of a
// class declaration's metadata.
//
// `lifecycle` is a FIELD of the class's own @co.dap.generic application and not
// an annotation of its own, which is the shape the reference is explicit about:
// no separate lifecycle annotation exists. Reading it from anywhere else — a
// bare `@co.dap.lifecycle`, or a generic annotation on some other declaration —
// would invent a spelling the language does not have.
func classLifecycleCapability(annotations annotationSet) lifecycleCapability {
	capability := lifecycleCapability{inClassBody: true}
	if !annotations.has("@co.dap.generic") {
		return capability
	}

	types, listed := annotations.option("@co.dap.generic", "types")
	entries, isList := types.([]any)
	capability.generic = listed && isList && len(entries) > 0
	capability.enabled = capability.generic &&
		annotations.optionString("@co.dap.generic", "lifecycle") == "true"

	return capability
}

// pushLifecycleCapability installs the capability of the class body being entered
// and returns the restore for the enclosing one.
func (p *parser) pushLifecycleCapability(capability lifecycleCapability) func() {
	previous := p.lifecycle
	p.lifecycle = capability
	return func() { p.lifecycle = previous }
}

// lifecycleDeclarationContextGuard applies lifecycle-declaration-context-guard to
// one source-declared lifecycle member, and with it the half of
// class-lifecycle-capability-guard that a parse can settle:
//
//	? - the enclosing declaration is a co.lang.class carrying valid
//	    co.dap.generic metadata with an explicit types=[...] list and
//	    lifecycle=true;
//	  - the source declaration is an override of an existing compiler lifecycle
//	    signature or an overload that adds another signature to the same
//	    language-owned lifecycle name; it never creates a new lifecycle name;
//	  - a non-generic class, a generic class with lifecycle absent/false, or any
//	    non-class declaration cannot source-declare @@new or @@init ?
//
// Only the eligibility half is checked here. Whether a particular signature
// overrides an inherited one or overloads the family, and whether the declared
// accessibility lets a given caller reach it, are questions about resolved
// signatures rather than about the token stream.
//
// The declaration is reported and then parsed rather than abandoned, so an
// ineligible lifecycle member costs one diagnostic and the rest of the class body
// still parses.
//
// Implements: lifecycle-declaration-context-guard
// Implements: class-lifecycle-capability-guard
func (p *parser) lifecycleDeclarationContextGuard(methodName name) {
	switch {
	case !p.lifecycle.inClassBody:
		p.reportNamedf(p.cur(), helpers.DiagnosticInvalidLifecycleDeclaration, "Invalid Lifecycle Declaration", "%s is a class lifecycle member and can be declared only inside a co.lang.class", methodName.Logical)

	case !p.lifecycle.generic:
		p.reportNamedf(p.cur(), helpers.DiagnosticInvalidLifecycleDeclaration, "Invalid Lifecycle Declaration", "%s customizes the compiler-owned class lifecycle, which only a generic class may do; give the class @co.dap.generic(types=[...], lifecycle=true) or remove the declaration, since every class already inherits its lifecycle implementations", methodName.Logical)

	case !p.lifecycle.enabled:
		p.reportNamedf(p.cur(), helpers.DiagnosticInvalidLifecycleDeclaration, "Invalid Lifecycle Declaration", "%s customizes the compiler-owned class lifecycle, so the class's @co.dap.generic metadata must carry lifecycle=true; without it the inherited lifecycle remains but developer override and overload are forbidden", methodName.Logical)
	}
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
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	methodName := p.parseLifecycleName()
	p.lifecycleDeclarationContextGuard(methodName)
	params := p.parseParameterList(false)

	var results []ast.Returns
	if p.at(scanlex.ARROW) {
		results = p.parseReturnTypeClause()
	}

	symb := p.functionSymbol(methodName.Scanned)
	symb.IsMethod = true
	symb.ClassMethod = true

	decl := ast.FunctionDeclarationStmt{NodeName: "FunctionDeclarationStmt", Span: p.spanFrom(spanStart), Parameters: [][]ast.Parameter{params},
		Name:       methodName.Scanned,
		ReturnType: results,
		Dapst:      annotations.list(),
		Symb:       symb,
	}
	p.applyFunctionFlags(&decl, annotations)
	decl.Symb.IsMethod = true
	decl.Symb.ClassMethod = true
	p.declareFunction(methodName.Tok, &decl)

	// function-definition binds the block body with "=", the same as any other named
	// function; a lifecycle method is no exception.
	p.expectOp("=", "before a lifecycle method's block body")
	return p.finishFunctionDefinition(decl)
}

// applyClassRelationships records the relationships declared in a class's kind options.
//
// FoLang spells inheritance and composition as options rather than as clauses, so
// `co.lang.class->(extends=Base, implements=[Printable])` is where they live.
func applyClassRelationships(symb *symboltable.ClassSymbol, options map[string]any, annotations annotationSet) {
	symb.Extends = optionNames(options, "extends")
	symb.Implements = optionNames(options, "implements")
	symb.Inherits = optionNames(options, "inherits")
	symb.Uses = optionNames(options, "uses")
	symb.Mixin = optionNames(options, "mixin")
	symb.Traits = optionNames(options, "traits")
	symb.Extensions = optionNames(options, "extensions")
	symb.ComposeAssociate = optionNames(options, "compose")

	// @co.dap.oops is the normative relationship surface. Keep the legacy kind
	// option fields above represented when encountered, but the fixed oops fields
	// take precedence and map to the existing symbol-table relationship slots.
	if names, present := annotationNames(annotations, "@co.dap.oops", "classes"); present {
		symb.Extends = names
	}
	if names, present := annotationNames(annotations, "@co.dap.oops", "interfaces"); present {
		symb.Implements = names
	}
	if names, present := annotationNames(annotations, "@co.dap.oops", "mixins"); present {
		symb.Mixin = names
	}
	if names, present := annotationNames(annotations, "@co.dap.oops", "traits"); present {
		symb.Traits = names
	}

	if with, ok := options["with"]; ok {
		if s, isString := with.(string); isString {
			symb.With = s
		}
	}
}

func annotationNames(annotations annotationSet, annotation, key string) ([]string, bool) {
	value, present := annotations.option(annotation, key)
	if !present {
		return nil, false
	}
	return optionNames(map[string]any{key: value}, key), true
}

func classRelationshipNames(p *parser, annotations annotationSet, key string, maximum int) []string {
	value, present := annotations.option("@co.dap.oops", key)
	if !present {
		return nil
	}
	items, isList := value.([]any)
	if !isList {
		p.reportf(p.cur(), "@co.dap.oops %s must be a list of declaration names", key)
		return nil
	}
	if maximum >= 0 && len(items) > maximum {
		p.reportf(p.cur(), "@co.dap.oops %s permits at most %d direct entries, found %d", key, maximum, len(items))
	}
	names := optionNames(map[string]any{key: value}, key)
	if len(names) != len(items) {
		p.reportf(p.cur(), "every @co.dap.oops %s entry must be a declaration name", key)
	}
	seen := map[string]bool{}
	for _, item := range names {
		if seen[item] {
			p.reportf(p.cur(), "@co.dap.oops %s repeats relationship target %q", key, item)
		}
		seen[item] = true
	}
	return names
}

func (p *parser) classRelationships(annotations annotationSet) map[string][]string {
	relationships := map[string][]string{
		"classes":    classRelationshipNames(p, annotations, "classes", 2),
		"interfaces": classRelationshipNames(p, annotations, "interfaces", -1),
		"mixins":     classRelationshipNames(p, annotations, "mixins", -1),
		"traits":     classRelationshipNames(p, annotations, "traits", -1),
	}
	owner := map[string]string{}
	for _, field := range []string{"classes", "interfaces", "mixins", "traits"} {
		for _, target := range relationships[field] {
			if previous, exists := owner[target]; exists && previous != field {
				p.reportf(p.cur(), "@co.dap.oops relationship target %q appears in incompatible fields %s and %s", target, previous, field)
			}
			owner[target] = field
		}
	}
	return relationships
}

func (p *parser) pushDirectRelationships(relationships map[string][]string) func() {
	previous := p.directRelationships
	p.directRelationships = relationships
	p.classRelationDepth++
	return func() {
		p.classRelationDepth--
		p.directRelationships = previous
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
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	p.expectOp("=", "before an interface body")
	symb := p.typeSymbol(declName.Scanned)

	members := p.parseBracedBody(symboltable.S_InterfaceSymbol, "an interface body", func() ast.Stmt {
		memberAnnotations := p.parseAnnotations()
		if p.atTypeDeclarationMember() {
			p.rejectOperatorPlacement(memberAnnotations, "an interface type declaration")
			return p.parseUnitKindMember(memberAnnotations)
		}
		p.rejectNestedKindDeclaration("an interface body")
		p.rejectOperatorPlacement(memberAnnotations, "an interface")
		return p.parseFunctionSpecification(memberAnnotations)
	}, symb)

	// ast.TypeDeclarationStmt stores its symbol as an ITypeSymbol, which among the
	// symbol kinds only TypeSymbol satisfies, so the interface kind is recorded on the
	// type symbol rather than through a dedicated InterfaceSymbol.
	symb.TypeType = "co.lang.interface"
	applyTypeVisibility(&symb.SymbolDetails, annotations)

	return ast.TypeDeclarationStmt{NodeName: "TypeDeclarationStmt", Span: p.spanFrom(spanStart), Name: declName.Scanned,
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
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	p.expectOp("=", "before a signature body")
	symb := p.typeSymbol(declName.Scanned)
	members := p.parseBracedBody(symboltable.S_SignatureSymbol, "a signature body", p.parseSignatureMember, symb)

	// As with an interface, the signature kind is recorded on a TypeSymbol because that
	// is the symbol kind ast.TypeDeclarationStmt accepts.
	symb.TypeType = "co.lang.signature"
	applyTypeVisibility(&symb.SymbolDetails, annotations)

	return ast.TypeDeclarationStmt{NodeName: "TypeDeclarationStmt", Span: p.spanFrom(spanStart), Name: declName.Scanned,
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
//	signature-member = value-specification
//	                 | function-specification
//	                 | signature-type-component
//	                 | associated-type-requirement
//
// The alternatives are separated by lookahead: a name followed by "co.lang.type"
// is a type component, a name followed by "co.lang.associatedType" is an
// associated-type requirement, a name followed by "(" is a function
// specification, and anything else is a value specification.
//
// Implements: signature-member
func (p *parser) parseSignatureMember() ast.Stmt {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	annotations := p.parseAnnotations()

	switch {
	case p.atSignatureTypeComponent():
		p.rejectOperatorPlacement(annotations, "a signature type component")
		return p.parseSignatureTypeComponent(annotations)
	case p.atAssociatedTypeDeclaration():
		// A signature states the requirement and never binds it; each matching
		// module supplies its own binding.
		p.rejectOperatorPlacement(annotations, "a signature associated type")
		return p.parseAssociatedTypeDeclaration(annotations, false)
	case p.atTypeDeclarationMember():
		p.rejectOperatorPlacement(annotations, "a signature type declaration")
		return p.parseUnitKindMember(annotations)
	case p.atMemberFunctionDeclaration():
		p.rejectOperatorPlacement(annotations, "a signature")
		return p.parseFunctionSpecification(annotations)
	default:
		p.rejectOperatorPlacement(annotations, "a signature value specification")
		return p.parseValueSpecification(annotations)
	}
}

// atSignatureTypeComponent reports whether the cursor begins a
// signature-type-component, whose fixed prefix is name, "co.lang.type".
func (p *parser) atSignatureTypeComponent() bool {
	return (p.atIdentifier() || p.at(scanlex.DISCARD_WILD_VAR)) &&
		p.peek(1).Kind == scanlex.BUILT_IN_KIND && p.peek(1).Value == "co.lang.type"
}
