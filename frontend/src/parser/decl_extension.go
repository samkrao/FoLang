package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// extension-declaration — section 7 of docs/grammar/folang.ebnf.
//
//	extension-declaration    = annotations, filename-derived-name,
//	                           "co.lang.extension", extension-target-options, "=",
//	                           extension-body
//	extension-target-options = "->", "(", "fortype", "=", type-expression, ")"
//	extension-body           = "{", { extension-member }, body-close
//	extension-member         = function-declaration
//
// An extension is a reusable collection of fully implemented methods that adds
// behavior to ONE explicitly selected class, without creating a subclass and
// without changing the target's nominal type identity or inheritance hierarchy
// (docs/language-ref.md, "Extension Declarations"):
//
//	// EmployeeExtension.fol
//	_ co.lang.extension->(fortype=somePkg.Employee) = {
//	    @co.dap.instance someFun()->()      = { co.out.println(this.someName); }
//	    @co.dap.class    someOtherFun()->() = { co.out.println(self.clsVariable); }
//	}
//
// The target options clause is REQUIRED and CLOSED — one `fortype` entry and
// nothing else — which is what separates it from the open kind-options clause
// every other container takes. It has to be required because the target is what
// receiver-dependent references resolve against WHILE the extension is compiled:
// `this` is an instance of the target in an @co.dap.instance method, and `self`
// is the target's class/type context in an @co.dap.class method. An extension
// with no target would have neither, so there would be nothing to defer the
// resolution to.
//
// The body holds function declarations only. An extension contributes callable
// behavior and nothing else: no new nominal type, no is-a relationship, no
// inherited state. A field would be state the target does not inherit, so there
// is no field alternative to admit.

// parseExtensionDeclaration parses the extension-declaration production.
//
// Implements: extension-declaration
// Implements: extension-body
// Implements: extension-member
func (p *parser) parseExtensionDeclaration(declName name, annotations annotationSet) ast.Stmt {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	forType := p.parseExtensionTargetOptions()

	p.expectOp("=", "before an extension body")

	// The extension's mandatory target supplies the class/type context an
	// @co.dap.class method's `self` denotes, which is the second of the two
	// contexts self-context-guard admits.
	popSelf := p.pushSelfReceiverContext()
	members := p.parseBracedBody(symboltable.S_ExtensionSymbol, "an extension body", p.parseExtensionMember)
	popSelf()

	return ast.ExtensionDeclarationStmt{Span: p.spanFrom(spanStart), Name: declName.Scanned,
		ForType: forType,
		Body:    members,
		SDapst:  annotations.list(),
		Symb:    p.extensionSymbol(declName.Scanned, forType),
	}
}

// parseExtensionTargetOptions parses the extension-target-options production and
// returns the one declared target type.
//
// Implements: extension-target-options
func (p *parser) parseExtensionTargetOptions() string {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	if !p.at(scanlex.ARROW) {
		p.failf(p.cur(), "an extension names its one target type, as in \"co.lang.extension->(fortype=somePkg.Employee)\"")
	}
	options := p.parseKindOptions()

	for key := range options {
		if key != "fortype" {
			p.failf(p.cur(), "an extension takes only the \"fortype\" option; %q is not allowed", key)
		}
	}
	target := firstOptionString(options, "fortype")
	if target == "" {
		p.failf(p.cur(), "an extension requires exactly one target type, written \"fortype=<type>\"")
	}
	if len(optionNames(options, "fortype")) != 1 {
		p.failf(p.cur(), "an extension targets exactly one class; declare a separate extension in its own <Name>.fol file for another target")
	}
	return target
}

// parseExtensionMember parses one extension-member.
//
// The classifier runs here as it does in every other container: an extension
// method carrying @co.dap.native or @co.dap.executionmodel is that kind of
// declaration wherever it is written (docs/language-ref.md, "Function-Shaped
// Declaration Classification").
func (p *parser) parseExtensionMember() ast.Stmt {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	annotations := p.parseAnnotations()
	p.rejectNestedKindDeclaration("an extension body")
	p.rejectOperatorPlacement(annotations, "an extension")
	if !p.atMemberFunctionDeclaration() {
		p.failf(p.cur(), "an extension body holds function declarations only; found %s", describeToken(p.cur()))
	}
	return p.parseDecoratedFunctionDeclaration(annotations)
}
