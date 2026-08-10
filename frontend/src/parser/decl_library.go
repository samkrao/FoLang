package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// library-declaration — section 7.
//
//	library-declaration        = annotations, filename-derived-name,
//	                             "co.lang.library", "=", library-body
//	library-body               = "{", { library-member }, body-close
//	library-member             = import-directive
//	                           | surface-struct-declaration
//	                           | surface-cstruct-declaration
//	                           | function-declaration
//	surface-struct-declaration = annotations, identifier, "co.lang.struct", "=",
//	                             struct-body
//	surface-cstruct-declaration = annotations, identifier, "co.lang.cstruct",
//	                             "=", cstruct-body
//
// A library surface file declares the library's public boundary. Its members are restricted
// to imports, the two struct kinds and functions, because a surface may only expose types
// whose signatures are closed over the boundary (docs/language-ref.md, "Allowed Surface
// Declarations").
//
// DECISION-FILE-003: the surface struct forms are two of the six stated
// exceptions to filename-derived naming. One surface file carries SEVERAL
// declarations, so one filename cannot name them and each keeps an explicit
// identifier in its head. The surface itself is file-backed and still uses "_".

// parseLibraryDeclaration parses the library-declaration production.
//
// Implements: library-declaration
func (p *parser) parseLibraryDeclaration(declName name, annotations annotationSet) ast.Stmt {
	spanStart := p.pos
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	options := p.parseOptionalKindOptions()

	p.expectOp("=", "before a library body")
	members := p.parseBracedBody("a library body", p.parseLibraryMember)

	libType := firstOptionString(options, "type")
	if libType == "" {
		libType = annotations.optionString("@co.dap.library", "type")
	}
	if libType == "" {
		libType = "application"
	}

	// The restricted-import rules are stated in terms of the surface's identity, so it is
	// recorded for the check that runs once the file has been read.
	p.libraryName = declName.Logical
	p.libraryType = libType

	// A source library has to be built before its consumers, so the driver is told.
	p.buildLibs = true

	return ast.Library{Span: p.spanFrom(spanStart), Body: members,
		Symb: p.librarySymbol(declName.Scanned, libType),
	}
}

// parseLibraryMember parses the library-member production.
//
// Implements: library-member
// Implements: surface-struct-declaration
// Implements: surface-cstruct-declaration
func (p *parser) parseLibraryMember() ast.Stmt {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	// Imports are members in their own right, not annotations decorating the
	// declaration that follows. Parsing them through the directive production is
	// essential: it validates the closed field set and records the dependency edge
	// used by restricted-import and cycle checks.
	if p.atFileDirective() {
		if p.lexeme() == "@co.ddap.import" {
			return p.parseImportDirective()
		}
		p.failf(p.cur(), "a library body admits import directives but not %q", p.lexeme())
	}

	annotations := p.parseAnnotations()

	// A surface member is named in the source, not by the filename, because one
	// surface file carries several of them (DECISION-FILE-003).
	declName := p.parseIdentifier("as a library member name")

	if p.at(scanlex.BUILT_IN_KIND) {
		kindTok := p.advance()
		p.rejectOperatorPlacement(annotations, "a library type member")
		switch kindTok.Value {
		case "co.lang.struct":
			return p.parseStructDeclaration(declName, annotations)
		case "co.lang.cstruct":
			return p.parseCStructDeclaration(declName, annotations)
		}
		p.failf(kindTok, "a library surface admits only imports, struct and cstruct declarations, and functions; %q is not allowed here", kindTok.Value)
	}

	// A function declaration: the name has already been consumed, so the declaration
	// continues from its parameter list.
	if annotations.has("@co.dap.operator") {
		p.reportf(declName.Tok, "an operator function cannot be exported from a library surface; declare it in a class or a struct companion unit")
	}
	return p.continueFunctionDeclaration(declName, annotations)
}
