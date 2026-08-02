package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// compilation-unit — section 1 of docs/grammar/folang.ebnf.
//
//	compilation-unit       = package-source-file
//	                       | application-entry-file
//	                       | library-surface-file
//	package-source-file    = file-preamble, primary-declaration
//	application-entry-file = file-preamble, { entry-item }
//	library-surface-file   = file-preamble, library-declaration
//	entry-item             = file-directive
//	                       | entry-type-declaration
//	                       | bare-function-pattern-clause
//	                       | capturing-function-pattern-clause
//	                       | statement
//
// All three forms share one preamble and are then distinguished by what follows it. The
// choice matters beyond structure, because the three have different rules: a package source
// file holds exactly ONE primary declaration
// (docs/language-ref.md, "Package Source Files"), an entry file holds statements and a
// restricted set of declarations (docs/language-ref.md, "Application Entry File"), and a
// library surface file holds one library declaration
// (docs/language-ref.md, "Library Surface file").
//
// The unit kind is recorded on the parser so that the context-sensitive rules — chiefly that
// an ordinary `let` value binding is forbidden in an entry file — can be reported where they
// apply.

// parseCompilationUnit parses the compilation-unit production and returns the AST root.
func (p *parser) parseCompilationUnit() ast.Stmt {
	preamble := p.parseFilePreamble()

	p.unit = p.classifyCompilationUnit()
	p.validateCompilationUnitDirectives()

	switch p.unit {
	case unitLibrary:
		return p.parseLibrarySurfaceFile(preamble)
	case unitPackage:
		return p.parsePackageSourceFile(preamble)
	default:
		return p.parseApplicationEntryFile(preamble)
	}
}

// classifyCompilationUnit decides which of the three unit forms this file is.
//
// The decision is lookahead over the declaration prefix — annotations, a name, an optional
// generic clause — looking for the built-in kind token that identifies the declaration.
// `co.lang.library` makes it a library surface; any other declarable kind makes it a package
// source file; anything else makes it an entry file.
//
// Three primary declarations carry NO kind token and so cannot be classified by lexeme:
// annotated-contract-declaration, type-constructor-primary and annotated-function-primary.
// They are recognised structurally instead, otherwise a file holding a typeclass or a type
// constructor would be misread as an entry file and its declaration reparsed as a call.
func (p *parser) classifyCompilationUnit() unitKind {
	detected := p.classifyCompilationUnitBySyntax()

	// Project layout settles the otherwise ambiguous cases. A root file is an
	// application entry unless it is the one library surface; a file below the
	// root is a package source unless it is a source-library surface. Legacy
	// Parse callers may not supply layout metadata, in which case the syntactic
	// classification remains the best available answer.
	if !p.file.LocationKnown || detected == unitLibrary {
		return detected
	}
	if p.file.AtRoot {
		return unitEntry
	}
	return unitPackage
}

// classifyCompilationUnitBySyntax distinguishes the three grammar shapes before
// the project-location rule is applied.
func (p *parser) classifyCompilationUnitBySyntax() unitKind {
	if p.atEOF() {
		return unitEntry
	}

	kind := p.lookaheadDeclarationKind()
	switch {
	case kind == "co.lang.library":
		return unitLibrary

	case kind == "":
		// An annotated function-pattern clause is still an entry item. Its
		// `name(...) =>` prefix otherwise resembles an annotated, kindless
		// function primary and would misclassify the entire file as a package.
		if p.atEntryFunctionPatternClause() {
			return unitEntry
		}
		// A kindless primary declaration still makes this a package source file.
		if p.atTypeConstructorPrimary() || p.atKindlessPrimaryDeclaration() {
			return unitPackage
		}
		return unitEntry

	case entryFileDeclarationKinds[kind]:
		// entry-type-declaration: an entry file may declare these type kinds, so one
		// of them alone does not make the file a package source file. Whether anything
		// follows it decides.
		return p.classifyAfterEntryTypeDeclaration()

	default:
		return unitPackage
	}
}

// atKindlessPrimaryDeclaration reports whether the cursor begins one of the two annotated
// primary declarations that have no kind token: annotated-contract-declaration and
// annotated-function-primary.
//
// Both require at least one annotation, which is what promotes them to primary declarations,
// so an unannotated statement is never captured here.
func (p *parser) atKindlessPrimaryDeclaration() bool {
	if !p.atAnnotation() {
		return false
	}
	return p.lookaheadOnly(func() bool {
		for p.atAnnotation() {
			p.advance()
			if p.at(scanlex.OPEN_PAREN) {
				p.skipBalanced(scanlex.OPEN_PAREN, scanlex.CLOSE_PAREN)
			}
		}
		if p.atLifecycleName() {
			p.advance()
		} else if p.atIdentifier() || p.at(scanlex.DISCARD_WILD_VAR) {
			p.advance()
		} else {
			return false
		}

		sawParens := false
		for p.at(scanlex.OPEN_PAREN) {
			p.skipBalanced(scanlex.OPEN_PAREN, scanlex.CLOSE_PAREN)
			sawParens = true
		}

		// annotated-contract-declaration: name, optional generics, then "=" and a body.
		if p.atOp("=") {
			return p.lookaheadOnly(func() bool {
				p.advance()
				return p.at(scanlex.OPEN_CURLY)
			})
		}
		// annotated-function-primary: a parameter list and a function binding.
		return sawParens && (p.at(scanlex.ARROW) || p.at(scanlex.OPEN_CURLY) ||
			p.atOp("=>") || p.atOp("=>>") || p.at(scanlex.SEMI_COLON))
	})
}

// classifyAfterEntryTypeDeclaration decides the unit form for a file whose first declaration
// is one of the type kinds an entry file also admits.
//
// A package source file holds exactly one primary declaration and nothing else, so if the
// file continues past that declaration it is an entry file.
func (p *parser) classifyAfterEntryTypeDeclaration() unitKind {
	if p.lookaheadOnly(func() bool {
		// Skip the declaration and see whether anything follows it.
		p.skipDeclarationPrefix()
		p.skipTo(scanlex.SEMI_COLON)
		p.accept(scanlex.SEMI_COLON)
		return p.atEOF()
	}) {
		return unitPackage
	}
	return unitEntry
}

// lookaheadDeclarationKind returns the built-in kind token that identifies the declaration at
// the cursor, or "" when the cursor does not begin a kind-identified declaration.
func (p *parser) lookaheadDeclarationKind() string {
	kind := ""
	p.lookaheadOnly(func() bool {
		p.skipDeclarationPrefix()
		if p.at(scanlex.BUILT_IN_KIND) {
			kind = p.lexeme()
		}
		return true
	})
	return kind
}

// skipDeclarationPrefix consumes the annotations, name and optional generic or parameter lists
// that precede a declaration's kind token. It is only ever called inside a lookahead.
func (p *parser) skipDeclarationPrefix() {
	for p.atAnnotation() {
		p.advance()
		if p.at(scanlex.OPEN_PAREN) {
			p.skipBalanced(scanlex.OPEN_PAREN, scanlex.CLOSE_PAREN)
		}
	}
	if p.atLifecycleName() {
		p.advance()
	}
	if p.atIdentifier() || p.at(scanlex.DISCARD_WILD_VAR) {
		p.advance()
	}
	for p.at(scanlex.OPEN_PAREN) {
		p.skipBalanced(scanlex.OPEN_PAREN, scanlex.CLOSE_PAREN)
	}
}

// parsePackageSourceFile parses the package-source-file production.
//
// Exactly one primary declaration is allowed. Anything after it is reported, because in
// FoLang each user-defined type, function group, macro, extension, template, typeclass, type
// constructor and unit must be in its own file.
func (p *parser) parsePackageSourceFile(preamble []ast.Stmt) ast.Stmt {
	declaration := p.parsePrimaryDeclaration()

	body := append(preamble, declaration)

	if !p.atEOF() {
		p.reportf(p.cur(), "a package source file holds exactly one declaration; %s follows the declaration and must move to its own file", describeToken(p.cur()))
		body = append(body, p.parseTrailingItems()...)
	}

	symb := p.packageSymbol(p.packageIdentity())
	return ast.PackageStmt{
		Body: body,
		Symb: symb,
	}
}

// parseLibrarySurfaceFile parses the library-surface-file production.
func (p *parser) parseLibrarySurfaceFile(preamble []ast.Stmt) ast.Stmt {
	annotations := p.parseAnnotations()

	declName := p.parseDeclarationName("as a library name")
	kindTok := p.expect(scanlex.BUILT_IN_KIND, "to declare a library")
	if kindTok.Value != "co.lang.library" {
		p.failf(kindTok, "expected \"co.lang.library\" in a library surface file, found %q", kindTok.Value)
	}
	declName = p.resolveFilenameDerivedName(declName, kindTok)

	library := p.parseLibraryDeclaration(declName, annotations)

	if !p.atEOF() {
		p.reportf(p.cur(), "a library surface file holds exactly one library declaration; %s follows it", describeToken(p.cur()))
	}

	// The preamble's imports belong to the library node, which keeps the surface's
	// dependencies with the surface.
	if lib, ok := library.(ast.Library); ok {
		lib.Body = append(preamble, lib.Body...)
		return lib
	}
	return library
}

// parseApplicationEntryFile parses the application-entry-file production.
//
// An entry file is a sequence of entry-items, which is the form a single-source application
// uses (docs/language-ref.md, "Single Source Application File").
func (p *parser) parseApplicationEntryFile(preamble []ast.Stmt) ast.Stmt {
	body := preamble

	for !p.atEOF() {
		startPos := p.pos

		var item ast.Stmt
		ok := p.recoverItem(startPos, syncStatement, func() {
			item = p.parseEntryItem()
		})
		if ok && item != nil {
			body = append(body, item)
		}
	}

	return ast.Application{
		Body: body,
		Symb: p.applicationSymbol(p.applicationName()),
	}
}

// parseEntryItem parses the entry-item production.
//
// The order matters: a directive and a declaration are both recognised before falling through
// to a statement, because several statement forms begin the same way.
func (p *parser) parseEntryItem() ast.Stmt {
	// file-directive.
	if p.atFileDirective() {
		return p.parseFileDirective()
	}

	// Function-pattern clauses are entry items, not general statements. Keep
	// this dispatch here so the same spelling in a nested block is rejected.
	if p.atEntryFunctionPatternClause() {
		return p.parseEntryFunctionPatternClause()
	}

	// entry-type-declaration, and the other declarations an entry file admits.
	if decl, ok := p.tryParseEntryDeclaration(); ok {
		return decl
	}

	return p.parseStatement()
}

// tryParseEntryDeclaration parses a declaration in an entry file, if one begins here.
//
// An entry file admits a narrower set than a package source file: type aliases, new types,
// opaque types, subtypes, supertypes and dependent-type aliases, but no type-constructor
// function and no user-defined type with a body
// (docs/language-ref.md, "Allowed Constructs"). A declaration outside that set is parsed
// anyway and reported, which gives a better message than a cascade of statement errors.
func (p *parser) tryParseEntryDeclaration() (ast.Stmt, bool) {
	kind := p.lookaheadDeclarationKind()
	if kind == "" {
		return nil, false
	}
	// Generic declarations are reusable type definitions and therefore belong
	// in package source files. Entry files may still use polymorphic types on a
	// declaration's right-hand side (for example a forall type alias); this guard
	// rejects only the declaration-level generic-parameter clause.
	if genericStart, ok := p.entryDeclarationGenericClauseStart(); ok {
		p.reportf(genericStart, "generic parameter clauses are not allowed in an application entry file; remove this clause or move the declaration into a package source file")
	}
	// Names shared by the type and kind registries use the type reading first in
	// an entry file. Package files are already selected by their project location,
	// so this only resolves an otherwise ambiguous executable declaration such as
	// `x co.lang.value = value;`.
	if isTypeFirstKind(kind) && !entryFileDeclarationKinds[kind] {
		return nil, false
	}

	if !entryFileDeclarationKinds[kind] {
		p.reportf(p.cur(), "%q may not be declared in an application entry file; move it into a package source file", kind)
	}

	return p.tryParsePrimaryDeclaration()
}

// entryDeclarationGenericClauseStart finds the opening token of a declaration's
// generic-parameter clause without consuming any input. A function parameter
// list has the same opening delimiter, so the shared generic-clause lookahead is
// used to distinguish Name(T) from name(value SomeType).
func (p *parser) entryDeclarationGenericClauseStart() (scanlex.Token, bool) {
	var start scanlex.Token
	found := p.lookaheadOnly(func() bool {
		for p.atAnnotation() {
			p.advance()
			if p.at(scanlex.OPEN_PAREN) {
				p.skipBalanced(scanlex.OPEN_PAREN, scanlex.CLOSE_PAREN)
			}
		}

		if p.atLifecycleName() {
			p.advance()
		}
		if !p.atIdentifier() && !p.at(scanlex.DISCARD_WILD_VAR) {
			return false
		}
		p.advance() // declaration name

		if !p.at(scanlex.OPEN_PAREN) || !p.looksLikeGenericParameterClause() {
			return false
		}
		start = p.cur()
		return true
	})
	return start, found
}

// entryFileDeclarationKinds is the set of kinds an application entry file may declare.
var entryFileDeclarationKinds = map[string]bool{
	"co.lang.type":          true,
	"co.lang.typealias":     true,
	"co.lang.newtype":       true,
	"co.lang.opaquetype":    true,
	"co.lang.subtype":       true,
	"co.lang.supertype":     true,
	"co.lang.dependentType": true,
}

// parseTrailingItems consumes whatever follows a complete package source file, so that a file
// with extra declarations still produces one diagnostic per item rather than stalling.
func (p *parser) parseTrailingItems() []ast.Stmt {
	var items []ast.Stmt

	for !p.atEOF() {
		startPos := p.pos

		var item ast.Stmt
		ok := p.recoverItem(startPos, syncStatement, func() {
			if decl, isDecl := p.tryParsePrimaryDeclaration(); isDecl {
				item = decl
				return
			}
			item = p.parseStatement()
		})
		if ok && item != nil {
			items = append(items, item)
		}
	}
	return items
}

// packageIdentity returns the package path this file belongs to.
//
// A subfolder containing .fol files IS a package, and the root itself is not, so a file at the
// root has an empty package path (docs/language-ref.md, "Package Identity"). Legacy Parse callers
// do not provide project-location metadata; for those callers only, retain the historical
// basename fallback instead of silently changing the public API's root symbol identity.
func (p *parser) packageIdentity() string {
	if !p.file.LocationKnown && p.file.PackagePath == "" {
		return p.file.Basename
	}
	return p.file.PackagePath
}

// applicationName returns the name recorded on an application entry file's root node.
func (p *parser) applicationName() string {
	if p.file.Basename != "" {
		return p.file.Basename
	}
	return p.file.Filename
}
