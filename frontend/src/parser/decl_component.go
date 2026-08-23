package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// component-surface-file and component-declaration — sections 1 and 7 of
// docs/grammar/folang.ebnf.
//
//	component-surface-file = file-preamble, component-declaration
//	component-declaration  = annotations, filename-derived-name,
//	                         "co.lang.component", "=", component-body,
//	                         component-surface-context-guard
//	component-body         = "{", { component-member }, body-close
//	component-member       = import-directive
//	                       | surface-struct-declaration
//	                       | surface-cstruct-declaration
//	                       | function-declaration
//	                       | component-export-selector
//	                       | operator-declaration
//
// Every project-local surface — `src/component.fol` and each
// `components/<kind>/component.fol` — uses this ONE root. There is no separate
// operator-source root any more: components/operators/component.fol is an
// ordinary component surface whose members happen to be operator declarations
// (docs/grammar/folang.ebnf, section 14).
//
// What distinguishes the surfaces is the FILESYSTEM, not the source text. The
// immediate `components/<kind>/` folder fixes the kind before the file is
// parsed, and `src/component.fol` chooses between two mutually exclusive
// standalone exposure models from its own metadata
// (docs/language-ref.md, "Form Exclusivity"):
//
//	@co.dap.library present                         -> projected standalone library
//	@co.dap.library absent + @co.dap.export in body -> packaged standalone library
//
// A surface that establishes neither, or both, is invalid. That check needs only
// this file, so it is made here; every other structural rule the
// component-surface-context-guard describes — which kinds a project may contain,
// which components may import which — needs the project tree and belongs to
// layout validation.
//
// The member set is why a component surface is not a package file. Its struct
// and cstruct declarations name THEMSELVES rather than taking the filename:
// several boundary declarations share one surface, so no filename could name
// them all. That is surface-struct-declaration's whole difference from
// struct-declaration.

// projectedLibraryKinds is the closed `type=` set of @co.dap.library on a
// standalone src/component.fol. Omitting `type` means application.
var projectedLibraryKinds = map[string]bool{
	componentKindApplication: true,
	componentKindNative:      true,
	componentKindDynamicVMRT: true,
}

// parseComponentSurfaceFile parses the component-surface-file production.
//
// Implements: component-surface-file
func (p *parser) parseComponentSurfaceFile(preamble []ast.Stmt) ast.Stmt {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	// As in parsePackageSourceFile, the surface declaration gets a recovery point
	// so a malformed head does not discard a well-formed body.
	startPos := p.pos
	var component ast.Stmt
	p.recoverItem(startPos, syncStatement, func() {
		annotations := p.parseAnnotations()

		declName := p.parseFilenameDerivedName("a component surface declaration")
		kindTok := p.expect(scanlex.BUILT_IN_KIND, "to declare a component")
		if kindTok.Value != "co.lang.component" {
			p.failf(kindTok, "expected \"co.lang.component\" in a component surface file, found %q", kindTok.Value)
		}

		component = p.parseComponentDeclaration(declName, annotations)
	})
	if component == nil {
		return ast.PackageStmt{Span: p.spanFrom0(), Body: preamble, Symb: p.packageSymbol(p.packageIdentity())}
	}

	if !p.atEOF() {
		p.reportf(p.cur(), "a component surface file holds exactly one component declaration; %s follows it", describeToken(p.cur()))
	}

	// The preamble's imports belong to the component node, which keeps the
	// surface's dependencies with the surface.
	if decl, ok := component.(ast.ComponentDeclarationStmt); ok {
		decl.SurfaceFile = prependSurfacePreamble(decl.SurfaceFile, preamble)
		return decl
	}
	return component
}

// parseComponentDeclaration parses the component-declaration production.
//
// Implements: component-declaration
// Implements: component-body
func (p *parser) parseComponentDeclaration(declName name, annotations annotationSet) ast.Stmt {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	kind := componentKindOf(p.file.Basedir)

	p.expectOp("=", "before a component body")
	members := p.parseBracedBody(symboltable.S_ComponentSymbol, "a component body", p.parseComponentMember)
	if kind == componentKindOperators {
		for _, member := range members {
			operator, ok := member.(ast.DirectiveStmt)
			if !ok || operator.Name != "co.lang.operator" {
				p.reportf(p.cur(), "components/operators/component.fol contains only co.lang.operator declarations")
				break
			}
		}
	}

	symb := p.componentSymbol(declName.Scanned, kind)
	p.declareNamed(declName, symb)

	decl := ast.ComponentDeclarationStmt{Span: p.spanFrom(spanStart), Name: declName.Scanned,
		Kind: kind,
		SurfaceFile: ast.PackageStmt{Span: p.spanFrom(spanStart), Name: kind,
			Body: members,
			Symb: p.packageSymbol(declName.Scanned),
		},
		SubPackage: map[string]ast.Stmt{},
		SDapst:     annotations.list(),
		Symb:       symb,
	}
	decl.Projected, decl.LibraryType = p.componentExposureModel(kind, annotations, members)
	return decl
}

// prependSurfacePreamble puts a surface file's preamble in front of the
// declarations it introduces, so the surface's imports stay with the surface.
func prependSurfacePreamble(surface ast.Stmt, preamble []ast.Stmt) ast.Stmt {
	if len(preamble) == 0 {
		return surface
	}
	file, isPackage := surface.(ast.PackageStmt)
	if !isPackage {
		return surface
	}
	file.Body = append(preamble, file.Body...)
	return file
}

// componentExposureModel applies the Form Exclusivity rule and returns whether
// the surface is a projected library and which projected kind it declares.
//
// The rule applies only to the standalone `src/component.fol`. A project-local
// component under `components/<kind>/` is NOT a library: its kind already comes
// from its folder, so `@co.dap.library` there would state a second, possibly
// contradictory identity, and the reference forbids it outright.
func (p *parser) componentExposureModel(kind string, annotations annotationSet, members []ast.Stmt) (bool, string) {
	projected := annotations.has("@co.dap.library")

	if kind != componentKindStandaloneSrc {
		if projected {
			p.reportf(p.cur(),
				"a project-local %s/%s/component.fol surface takes no %s annotation; the fixed directory is what supplies its kind",
				componentDomain, kind, "@co.dap.library")
		}
		return false, ""
	}

	libraryType := annotations.optionString("@co.dap.library", "type")
	exports := componentDeclaresExportSelector(members)

	switch {
	case projected && exports:
		p.report(p.cur(), "a standalone src/component.fol is either a projected library or a packaged library, never both; remove either the @co.dap.library annotation or the @co.dap.export package selector")
	case !projected && !exports:
		p.report(p.cur(), "a standalone src/component.fol establishes its exposure model with either @co.dap.library for a projected library or an @co.dap.export package selector in its body for a packaged library")
	}

	if projected {
		// Omitting `type` means application, so only a stated kind is checked.
		if libraryType == "" {
			libraryType = componentKindApplication
		} else if !projectedLibraryKinds[libraryType] {
			p.reportf(p.cur(), "%q is not a projected library kind; a projected standalone component is one of application, native or dynamicvmrt", libraryType)
		}
		return true, libraryType
	}
	return false, ""
}

// componentDeclaresExportSelector reports whether the body carries the packaged
// export selector.
func componentDeclaresExportSelector(members []ast.Stmt) bool {
	for _, member := range members {
		if directive, ok := member.(ast.DirectiveStmt); ok && directive.Name == componentExportSelectorName {
			return true
		}
	}
	return false
}

// componentExportSelectorName is the metadata name of component-export-selector.
const componentExportSelectorName = "@co.dap.export"

// parseComponentMember parses the component-member production.
//
// Implements: component-member
func (p *parser) parseComponentMember() ast.Stmt {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	// operator-declaration heads with a symbolic run rather than a name, so it is
	// recognised before annotations are read: nothing may precede it.
	if p.atOperatorDeclaration() {
		return p.parseComponentOperatorDeclaration()
	}

	// import-directive and component-export-selector are both bare metadata
	// applications, distinguished by their name.
	if p.atAnnotation() && p.atComponentSurfaceMetadata() {
		return p.parseComponentSurfaceMetadata()
	}

	annotations := p.parseAnnotations()

	// surface-struct-declaration and surface-cstruct-declaration: an identifier,
	// then the kind token. They name themselves because one surface carries
	// several of them.
	if p.atComponentBoundaryDeclaration() {
		return p.parseComponentBoundaryDeclaration(annotations)
	}

	if !p.atMemberFunctionDeclaration() {
		p.failf(p.cur(), "a component surface holds imports, boundary struct/cstruct declarations, public API functions and — in the operators component — operator declarations; found %s", describeToken(p.cur()))
	}
	return p.parseDecoratedFunctionDeclaration(annotations)
}

// atOperatorDeclaration reports whether the cursor begins an
// operator-declaration.
//
// The head is a symbolic run followed by the co.lang.operator kind token, which
// no other component member can be: every other member heads with an annotation,
// an identifier or "_".
func (p *parser) atOperatorDeclaration() bool {
	if !scanlex.IsOperatorSpelling(p.cur().Value) {
		return false
	}
	return p.lookaheadOnly(func() bool {
		p.advance() // the operator symbol
		return p.at(scanlex.OPERATOR_SOURCE_KIND) && p.lexeme() == "co.lang.operator"
	})
}

// parseComponentOperatorDeclaration parses the operator-declaration production
// as a component member.
//
// The declarations themselves are read a second time here, having already been
// read by the bootstrap pre-pass that built the project's operator table
// (operator_bootstrap.go). That is deliberate: the pre-pass produces the TABLE
// the lexer needs before any source is tokenized, while this pass produces the
// AST entry for the surface being compiled. Reusing the same reader keeps one
// definition of the declaration's shape.
//
// Implements: operator-declaration
func (p *parser) parseComponentOperatorDeclaration() ast.Stmt {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	if kind := componentKindOf(p.file.Basedir); kind != componentKindOperators {
		p.reportf(p.cur(),
			"a co.lang.operator declaration belongs to %s/%s/component.fol; an @co.dap.operator function implements an already declared symbol elsewhere",
			componentDomain, componentKindOperators)
	}

	symbolTok := p.advance()
	p.expect(scanlex.OPERATOR_SOURCE_KIND, "to declare an operator")
	p.expectOp("=", "before an operator metadata body")

	options := p.parseOperatorMetadataBody(symbolTok.Value)
	p.statementEnd("an operator declaration")

	return ast.DirectiveStmt{Span: p.spanFrom(spanStart), Name: "co.lang.operator",
		Parameters: options,
		Symb:       p.directiveSymbol("co.lang.operator", false),
	}
}

// parseOperatorMetadataBody reads the operator-body property list.
//
// The braces are MAP-shaped rather than declaration-shaped — the body is a
// property list — which is why the enclosing declaration still takes its
// terminating ";" and body-closure-guard does not apply here.
//
// Implements: operator-body
// Implements: operator-property
func (p *parser) parseOperatorMetadataBody(symbol string) map[string]any {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	p.expect(scanlex.OPEN_CURLY, "to open an operator metadata body")

	options := map[string]any{"symbol": symbol}
	for !p.at(scanlex.CLOSE_CURLY) && !p.atEOF() {
		keyTok := p.cur()
		key := logicalName(keyTok.Value)
		p.advance()
		if !operatorSourcePropertyKeys[key] {
			p.reportf(keyTok, "unknown operator property %q", key)
		}
		p.expectOp("=", "after operator property "+key)
		if p.at(scanlex.OPERATOR_SOURCE_CONSTANT) {
			constant := p.advance()
			options[key] = scanlex.Operator_source_constants[constant.Value]
		} else {
			options[key] = p.parseAnnotationValue()
		}

		if !p.accept(scanlex.COMMA) {
			break
		}
	}

	p.expect(scanlex.CLOSE_CURLY, "to close an operator metadata body")
	return options
}

// atComponentSurfaceMetadata reports whether the annotation at the cursor is one
// the component body carries as a MEMBER rather than as decoration of a
// following declaration.
//
// The distinction matters because both are spelled as a bare annotation. An
// import directive and the export selector stand alone; every other annotation
// decorates the declaration after it. The reference is explicit for the
// selector: in this structural context `@co.dap.export(...)` applies to the
// containing `_ co.lang.component` "and is not waiting for a following
// declaration" (docs/language-ref.md, "Packaged Library Form").
func (p *parser) atComponentSurfaceMetadata() bool {
	name := p.cur().Value
	return name == componentExportSelectorName || isImportDirectiveName(name)
}

// parseComponentSurfaceMetadata parses an import-directive or a
// component-export-selector standing as a component-body entry.
//
// Implements: component-export-selector
func (p *parser) parseComponentSurfaceMetadata() ast.Stmt {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	if isImportDirectiveName(p.cur().Value) {
		return p.parseImportDirective()
	}
	directive := p.parseAnnotation()
	if directive.Name == componentExportSelectorName {
		p.validateComponentExportSelector(directive)
	}
	return directive
}

// validateComponentExportSelector checks the half of
// component-export-selector-guard that one file can decide.
//
// The parser collects the complete field payload whatever it is; what it checks
// here is that the selector is in a structural context that HAS a packaged
// exposure model. The packages selector's own contents, and the project kind,
// are semantic (docs/grammar/folang.ebnf, component-export-selector-guard).
func (p *parser) validateComponentExportSelector(directive ast.DirectiveStmt) {
	kind := componentKindOf(p.file.Basedir)
	if kind == componentKindStandaloneSrc || kind == componentKindPackaged {
		return
	}
	p.reportf(p.cur(),
		"the %s package selector belongs to a packaged surface — src/component.fol or %s/%s/component.fol — and not to the %s component",
		componentExportSelectorName, componentDomain, componentKindPackaged, kind)
}

// atComponentBoundaryDeclaration reports whether the cursor begins a
// surface-struct-declaration or surface-cstruct-declaration.
func (p *parser) atComponentBoundaryDeclaration() bool {
	if !p.atIdentifier() && !p.at(scanlex.DISCARD_WILD_VAR) {
		return false
	}
	return p.lookaheadOnly(func() bool {
		p.advance() // the declared name
		return p.at(scanlex.BUILT_IN_KIND) &&
			(p.lexeme() == "co.lang.struct" || p.lexeme() == "co.lang.cstruct")
	})
}

// parseComponentBoundaryDeclaration parses the surface-struct-declaration and
// surface-cstruct-declaration productions.
//
// They differ from struct-declaration and cstruct-declaration in exactly one
// place: the head is an identifier rather than "_". A component surface may
// carry several boundary declarations, so the filename — which is the fixed
// `component.fol` for every surface in the project — cannot name any of them.
//
// Implements: surface-struct-declaration
// Implements: surface-cstruct-declaration
func (p *parser) parseComponentBoundaryDeclaration(annotations annotationSet) ast.Stmt {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	if p.at(scanlex.DISCARD_WILD_VAR) {
		p.failf(p.cur(), "a component surface boundary declaration is named in its own head, not by the fixed component.fol filename, so write an identifier here rather than \"_\"")
	}
	declName := p.parseIdentifier("as a component surface boundary declaration name")
	kindTok := p.advance()

	if kindTok.Value == "co.lang.cstruct" {
		return p.parseCStructDeclaration(declName, annotations)
	}
	return p.parseStructDeclaration(declName, annotations)
}
