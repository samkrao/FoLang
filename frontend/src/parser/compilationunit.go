package parser

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/samkrao/fo-lang/frontend/src/ast"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// compilation-unit — section 1 of docs/grammar/folang.ebnf.
//
//	compilation-unit             = package-source-file
//	                             | application-entry-file
//	                             | library-surface-file
//	package-source-file          = package-primary-source-file
//	                             | ordinary-unit-source-file
//	                             | companion-unit-source-file
//	                             | package-metadata-source-file
//	package-primary-source-file  = file-preamble, primary-declaration
//	ordinary-unit-source-file    = file-preamble, unit-declaration
//	companion-unit-source-file   = file-preamble, unit-declaration
//	package-metadata-source-file = file-preamble, package-alias-declaration
//	application-entry-file       = file-preamble, { entry-item }
//	library-surface-file         = file-preamble, library-declaration
//	entry-item                   = file-directive
//	                             | entry-type-declaration
//	                             | bare-function-pattern-clause
//	                             | capturing-function-pattern-clause
//	                             | entry-statement
//
// All the forms share one preamble and are then distinguished by what follows it. The
// choice matters beyond structure, because they have different rules: a package source
// file holds exactly ONE declaration
// (docs/language-ref.md, "Package Source Files"), an entry file holds statements and a
// restricted set of declarations (docs/language-ref.md, "Application Entry File"), and a
// library surface file holds one library declaration
// (docs/language-ref.md, "Library Surface file").
//
// Revision 23 split package-source-file into four. Which one applies is decided by
// the FILENAME, not by the body: `_ co.lang.unit` is the same source text in an
// ordinary unit and in a companion, and only the filename says whether its members
// merge into the package namespace or attach to a struct (DECISION-FILE-001,
// DECISION-UNIT-001). See sourcefile.go for the filename grammar.
//
// The unit kind is recorded on the parser so that the context-sensitive rules — chiefly that
// an ordinary `let` value binding is forbidden in an entry file — can be reported where they
// apply.

// parseCompilationUnit parses the compilation-unit production and returns the AST root.
//
// Implements: compilation-unit
func (p *parser) parseCompilationUnit() ast.Stmt {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	// The filename is validated against its siblings before any source text is
	// read, because a name collision is a property of the FOLDER rather than of
	// this file's contents (DECISION-FILE-004).
	p.validateFilenameIsUnique()

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

// validateFilenameIsUnique reports a source filename that denotes the same
// declaration as another file in the same folder.
//
// On a case-INSENSITIVE filesystem the operating system already prevents the
// collision: `Employee.fol` and `employee.fol` cannot both exist. On a
// case-SENSITIVE one they can, and both derive the declaration `Employee`, so
// the package would silently acquire two declarations of one name. That is why
// the reference requires this to be decided from the canonical file key and
// forbids relying on operating-system filename comparison
// (docs/language-ref.md, "Filename Canonicalization").
//
// The check reads the folder rather than the source text, so it also catches the
// collision when only one of the two files is being compiled.
func (p *parser) validateFilenameIsUnique() {
	if p.file.Basedir == "" || p.file.Basename == "" {
		return
	}

	entries, err := os.ReadDir(p.file.Basedir)
	if err != nil {
		// A folder the compiler cannot list is not a source error; the file it
		// was handed is still parsed on its own terms.
		return
	}
	siblings := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			siblings = append(siblings, entry.Name())
		}
	}

	conflicts := duplicateSourceFilenames(p.file.Basename, siblings)
	if len(conflicts) == 0 {
		return
	}
	p.report(p.cur(), fmt.Sprintf(
		"source filename %q denotes the same declaration as %s in this folder; "+
			"filenames are compared after case folding and underscore removal, so these spellings conflict rather than declaring separate types",
		p.file.Basename, strings.Join(quoteAll(conflicts), ", ")))
}

// quoteAll renders filenames for a diagnostic that lists several of them.
func quoteAll(names []string) []string {
	quoted := make([]string, len(names))
	for i, name := range names {
		quoted[i] = strconv.Quote(name)
	}
	return quoted
}

// classifyCompilationUnit decides which of the three unit forms this file is.
//
// A reserved or suffixed filename decides outright. Otherwise the decision is lookahead over
// the declaration prefix — annotations then the name — looking for the built-in kind token
// that identifies the declaration. `co.lang.library` makes it a library surface; any other
// declarable kind makes it a package source file; anything else makes it an entry file.
//
// A kindless declaration shape carries no kind token and so cannot be classified by
// lexeme. It is recognised structurally instead, otherwise a file holding one would be
// misread as an entry file and its declaration reparsed as a call.
func (p *parser) classifyCompilationUnit() unitKind {
	// A reserved filename settles the question outright, which is the whole point of
	// reserving it. `appl.fol` is the application entry file and `library.fol` is a
	// library surface, wherever in the project either sits; no other file may be
	// either, and neither is ever anything else.
	switch p.file.Source.Class {
	case sourceClassApplicationEntry:
		return unitEntry
	case sourceClassLibrarySurface:
		return unitLibrary
	case sourceClassPackageMetadata:
		return unitPackage
	}

	// A unit and a package declaration have their own source forms, and a file
	// carrying one is a package source file wherever it sits.
	if p.file.Source.isUnitSourceFile() {
		return unitPackage
	}

	// What is left is `<Name>.fol`, which is a package source file. The syntactic
	// classification is consulted only for legacy Parse callers, which supply no
	// project location and so may hand over an entry file under any name at all.
	detected := p.classifyCompilationUnitBySyntax()
	if !p.file.LocationKnown {
		return detected
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
		// A kindless declaration shape still routes to the package reading, which
		// is where the diagnostic naming its real home lives. A type-level function
		// is a unit member, so a file that begins with one is not a package primary.
		if p.atKindlessPrimaryDeclaration() {
			return unitPackage
		}
		return unitEntry

	case entryFileDeclarationKinds[kind]:
		// entry-type-declaration. Revision 23 removed type-declaration from
		// primary-declaration, so one of these NEVER makes a file a package
		// source file by itself: in a package it is a unit member, and a unit
		// file is already selected by its filename above. What is left is the
		// entry form.
		return unitEntry

	default:
		return unitPackage
	}
}

// atKindlessPrimaryDeclaration reports whether the cursor begins a declaration shape that
// carries no kind token.
//
// primary-declaration has no such alternative any more: every one is selected by a
// built-in kind. The shape is still recognised so that a file written this way is
// classified as a package source file and reaches the diagnostic naming where the
// declaration belongs, rather than leaving an entry parser to fail on a braced body.
//
// It requires at least one annotation, so an unannotated statement is never captured here.
func (p *parser) atKindlessPrimaryDeclaration() bool {
	// A kindless primary is annotated by definition, so without annotations this
	// cannot be one.
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

		// The withdrawn contract shape: name, optional generics, then "=" and a body.
		if p.atOp("=") {
			return p.lookaheadOnly(func() bool {
				p.advance()
				return p.at(scanlex.OPEN_CURLY)
			})
		}
		// A function shape: a parameter list and a function binding.
		return sawParens && (p.at(scanlex.ARROW) || p.at(scanlex.OPEN_CURLY) ||
			p.atOp("=>") || p.atOp("=>>") || p.at(scanlex.SEMI_COLON))
	})
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
// Exactly one declaration is allowed. Anything after it is reported, because in
// FoLang each user-defined type, function group, macro, extension, template,
// typeclass and unit must be in its own file.
//
// The filename classification chooses the alternative. It is what makes the four
// roots distinguishable at all: three of them begin with the identical token
// sequence `_ co.lang.<kind>`.
//
// Implements: package-source-file
// Implements: package-primary-source-file
// Implements: ordinary-unit-source-file
// Implements: companion-unit-source-file
// Implements: package-metadata-source-file
func (p *parser) parsePackageSourceFile(preamble []ast.Stmt) ast.Stmt {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	// The one declaration gets its own recovery point.
	//
	// Without it a malformed declaration HEAD — a missing kind token, an
	// explicitly named file-backed primary — bails out past every sync point to
	// parseTopLevel, which discards the whole tree. That is the common case in an
	// editor rather than an edge case: the head is exactly what is incomplete
	// while the user is still typing it, and the body below it is often already
	// well formed. Recovering here costs the head instead of the file.
	startPos := p.pos
	var declaration ast.Stmt
	p.recoverItem(startPos, syncStatement, func() {
		switch {
		case p.file.Source.isUnitSourceFile():
			declaration = p.parseUnitSourceFile()
		case p.file.Source.Class == sourceClassPackageMetadata:
			declaration = p.parsePackageMetadataSourceFile()
		default:
			declaration = p.parsePrimaryDeclaration()
		}
	})

	body := preamble
	if declaration != nil {
		body = append(body, declaration)
	}

	if !p.atEOF() {
		p.reportf(p.cur(), "a package source file holds exactly one declaration; %s follows the declaration and must move to its own file", describeToken(p.cur()))
		body = append(body, p.parseTrailingItems()...)
	}

	symb := p.packageSymbol(p.packageIdentity())
	return ast.PackageStmt{Span: p.spanFrom0(), Body: body,
		Symb: symb,
	}
}

// parseUnitSourceFile parses the one unit-declaration an ordinary or companion
// unit file holds.
//
// Both forms take the same grammar (DECISION-UNIT-001), so the only thing
// checked here is that the file really does declare a unit: a `<Name>.unit.fol`
// filename is a promise about the body, and a body that breaks it would
// otherwise be indexed as a unit it is not.
func (p *parser) parseUnitSourceFile() ast.Stmt {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	annotations := p.parseAnnotations()
	declName := p.parseFilenameDerivedName(p.file.Source.describeClass())

	kindTok := p.expect(scanlex.BUILT_IN_KIND, "to declare a unit")
	if kindTok.Value != "co.lang.unit" {
		p.failf(kindTok, "%s must contain %s; found %q", p.file.Source.describeClass(), "`_ co.lang.unit`", kindTok.Value)
	}
	return p.parseUnitDeclaration(declName, annotations)
}

// parsePackageMetadataSourceFile parses the one package-alias-declaration the
// reserved package.fol source form holds (DECISION-PKG-001).
func (p *parser) parsePackageMetadataSourceFile() ast.Stmt {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	annotations := p.parseAnnotations()

	// `_` here does NOT derive the literal name `package` from the reserved
	// filename; the logical segment comes from the body's name field. So the
	// wildcard is consumed directly rather than through filename derivation,
	// which would report the reserved filename as an unusable component.
	if !p.at(scanlex.DISCARD_WILD_VAR) {
		p.failf(p.cur(), "the reserved package.fol source form contains exactly %s", "`_ co.lang.package = { name: \"segment\" };`")
	}
	declName := nameFrom(p.advance())

	kindTok := p.expect(scanlex.BUILT_IN_KIND, "to declare a package")
	if kindTok.Value != "co.lang.package" {
		p.failf(kindTok, "the reserved package.fol source form declares %q, not %q", "co.lang.package", kindTok.Value)
	}
	return p.parsePackageAliasDeclaration(declName, annotations)
}

// parseLibrarySurfaceFile parses the library-surface-file production:
//
//	library-surface-file            = standalone-library-surface-file
//	                                | source-library-surface-file
//	standalone-library-surface-file = file-preamble,
//	                                  standalone-library-kind-annotation,
//	                                  library-declaration
//	source-library-surface-file     = file-preamble, library-declaration
//
// The two alternatives differ by exactly one thing: where the library's KIND comes
// from. A standalone `src/library.fol` has no enclosing directory to say what it is, so
// it declares its kind in an annotation. A `srclib/<slot>/library.fol` takes its kind
// from the fixed slot, and so carries no annotation at all — the reference states
// plainly that `@co.dap.library(type=...)` is not used on project-local `library.fol`
// surfaces (docs/language-ref.md, "Import Directive Fields").
//
// Which alternative applies is therefore a fact about the file's LOCATION, not its
// contents, which is why it is settled before the declaration is read.
//
// Implements: library-surface-file
// Implements: standalone-library-surface-file
// Implements: source-library-surface-file
func (p *parser) parseLibrarySurfaceFile(preamble []ast.Stmt) ast.Stmt {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	// As in parsePackageSourceFile, the surface declaration gets a recovery point
	// so a malformed head does not discard a well-formed body.
	startPos := p.pos
	var library ast.Stmt
	p.recoverItem(startPos, syncStatement, func() {
		annotations := p.parseLibraryKindAnnotation()

		declName := p.parseFilenameDerivedName("a library surface declaration")
		kindTok := p.expect(scanlex.BUILT_IN_KIND, "to declare a library")
		if kindTok.Value != "co.lang.library" {
			p.failf(kindTok, "expected \"co.lang.library\" in a library surface file, found %q", kindTok.Value)
		}

		library = p.parseLibraryDeclaration(declName, annotations)
	})
	if library == nil {
		return ast.PackageStmt{Span: p.spanFrom0(), Body: preamble, Symb: p.packageSymbol(p.packageIdentity())}
	}

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

// libraryKindStrings is the library-kind-string production: the closed set of decoded
// values a standalone surface's `type=` may name.
//
// "operator" is deliberately absent. The operator bootstrap surface is selected by its
// `srclib/operators/` position and is read by a different grammar root entirely, so it
// never carries — and never could carry — a kind annotation.
//
// Implements: library-kind-string
var libraryKindStrings = map[string]bool{
	"application": true,
	"dynamicvmrt": true,
	"advanced":    true,
	"system":      true,
	"ffi":         true,
}

// parseLibraryKindAnnotation parses the standalone-library-kind-annotation production:
//
//	standalone-library-kind-annotation = "@co.dap.library", "(", "type", "=",
//	                                     library-kind-string, ")"
//
// It is REQUIRED before a standalone `src/library.fol` declaration and FORBIDDEN before
// a `srclib/<slot>/library.fol` one, because the slot already fixes that library's kind.
// Both halves are enforced here: a missing annotation leaves the surface with no kind at
// all, and a present one on a source library states a kind that could contradict the
// directory it sits in.
//
// Implements: standalone-library-kind-annotation
func (p *parser) parseLibraryKindAnnotation() annotationSet {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	slot := sourceLibrarySlotOf(p.file.Basedir)
	annotations := p.parseAnnotations()
	kind := annotations.optionString("@co.dap.library", "type")

	if slot != "" {
		if annotations.has("@co.dap.library") {
			p.reportf(p.cur(), "a project-local source library takes no %s annotation; the fixed %s/%s/ directory is what supplies its identity and kind",
				"@co.dap.library", "srclib", slot)
		}
		return annotations
	}

	switch {
	case !annotations.has("@co.dap.library"):
		p.reportf(p.cur(), "a standalone library surface declares its kind, as in \"@co.dap.library(type=\\\"system\\\")\"; only a project-local srclib/<kind>/library.fol may omit it")
	case !libraryKindStrings[kind]:
		p.reportf(p.cur(), "%q is not a library kind; a standalone library surface is one of application, dynamicvmrt, advanced, system or ffi", kind)
	}
	return annotations
}

// parseApplicationEntryFile parses the application-entry-file production.
//
// An entry file is a sequence of entry-items, which is the form a single-source application
// uses (docs/language-ref.md, "Single Source Application File").
//
// Implements: application-entry-file
func (p *parser) parseApplicationEntryFile(preamble []ast.Stmt) ast.Stmt {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

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

	return ast.Application{Span: p.spanFrom0(), Body: body,
		Symb: p.applicationSymbol(p.applicationName()),
	}
}

// parseEntryItem parses the entry-item production.
//
// The order matters: a directive and a declaration are both recognised before falling through
// to a statement, because several statement forms begin the same way.
//
// Implements: entry-item
func (p *parser) parseEntryItem() ast.Stmt {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

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

	return p.parseEntryStatement()
}

// parseEntryStatement parses the entry-statement production:
//
//	entry-statement = variable-declaration
//	                | inferred-variable-declaration
//	                | grouped-variable-declaration
//	                | multiple-assignment-statement
//	                | expression-statement
//	                | empty-statement
//
// An entry file's top level is NARROWER than a block's. It admits declarations of
// values and expressions, and nothing that introduces a nested named scope: no
// named block, no labeled or bare block, no `let` value binding, no local function or
// closure declaration, and no return — an entry file is the program, so there is no
// enclosing function to return from (docs/language-ref.md, "Allowed Constructs").
//
// Conditions and loops are not missing from that list. FoLang spells them as method
// chains — `(x > y).do({ … });` — so they arrive here as expression statements, and
// their braces are lambda arguments rather than statement blocks.
//
// The rejected shape is still parsed after the report, so one misplaced construct
// yields one diagnostic rather than a cascade through the rest of the file.
//
// Implements: entry-statement
func (p *parser) parseEntryStatement() ast.Stmt {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

	if what := p.entryForbiddenStatement(); what != "" {
		p.reportf(p.cur(), "%s is not one of the statements an application entry file admits; an entry file holds variable declarations, assignments and expressions", what)
	}
	return p.parseStatement()
}

// entryForbiddenStatement names the statement form at the cursor when it is one the
// entry-statement production leaves out, and returns "" otherwise.
//
// It runs BEFORE parseStatement's annotation run is consumed, so the probes have to
// look past any annotations themselves; a `let` binding and a labeled block are the
// two that can carry them.
func (p *parser) entryForbiddenStatement() string {
	switch {
	case p.at(scanlex.OPEN_CURLY):
		return "a bare block"
	case p.atLabeledBlock():
		return "a labeled block"
	case p.atKeyword("let"):
		return "a let value binding"
	case p.at(scanlex.BUIL_IN_STMT_EXPRS) && isControlStatementBuiltin(p.lexeme()):
		return "a " + logicalControlVerb(p.lexeme()) + " statement"
	case p.atNamedBlockDeclaration():
		return "a named block declaration"
	case p.atLocalFunctionDeclaration():
		return "an ordinary function declaration"
	case p.atClosureDeclaration():
		return "a closure declaration"
	}
	return ""
}

// tryParseEntryDeclaration parses an entry-type-declaration, if one begins here.
//
//	entry-type-declaration               = entry-parameterized-type-declaration
//	                                     | entry-simple-type-declaration
//	entry-parameterized-type-declaration = annotations, identifier,
//	                                       generic-parameter-clause,
//	                                       "co.lang.type", [ kind-options ],
//	                                       [ "=", type-expression ],
//	                                       statement-end
//	entry-simple-type-declaration        = annotations, identifier,
//	                                       entry-simple-type-kind,
//	                                       [ kind-options ],
//	                                       [ "=", type-expression ],
//	                                       statement-end
//
// An entry file admits a narrower set than a package source file: the type-alias
// family only — no ordinary or type-level function declaration, no file-backed
// UDT or container primary, no unit, instance or matcher
// (DECISION-ENTRY-001, docs/language-ref.md, "Allowed Constructs"). A declaration
// outside that set is parsed anyway and reported, which gives a better message
// than a cascade of statement errors.
//
// Revision 23 admitted the parameterized form. An entry file could already USE a
// polymorphic type, and `Option(T) co.lang.type = …` is a type declaration like
// any other in this family; refusing only its parameter clause drew a line the
// reference does not draw. Its name is written explicitly because the entry file
// is not file-backed and has no name to derive (DECISION-FILE-003).
//
// Implements: entry-type-declaration
// Implements: entry-parameterized-type-declaration
// Implements: entry-simple-type-declaration
func (p *parser) tryParseEntryDeclaration() (ast.Stmt, bool) {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}
	kind := p.lookaheadDeclarationKind()
	if kind == "" {
		return nil, false
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
		return p.tryParsePrimaryDeclaration()
	}

	annotations := p.parseAnnotations()
	declName := p.parseIdentifier("as an entry type declaration name")

	// Only co.lang.type has a parameterized form; every other kind in the family
	// is entry-simple-type-declaration and has no clause slot at all.
	// parseTypeDeclaration reports the mismatch, which keeps one rule in one
	// place for the entry, unit and signature contexts alike.
	generics := p.parseOptionalGenericParameterClause()

	kindTok := p.expect(scanlex.BUILT_IN_KIND, "to declare an entry type declaration's kind")
	return p.parseTypeDeclaration(declName, generics, kindTok, annotations), true
}

// entryFileDeclarationKinds is the kind set of entry-simple-type-declaration, plus the
// co.lang.type that entry-parameterized-type-declaration shares with it.
//
// It is entry-simple-type-declaration's own list rather than type-declaration-kind's:
// `co.lang.kind` is a unit-level kind declaration with no entry-file form, and the
// reference's "Allowed Constructs" for the entry file names exactly these six.
var entryFileDeclarationKinds = map[string]bool{
	"co.lang.type":          true,
	"co.lang.newtype":       true,
	"co.lang.opaquetype":    true,
	"co.lang.subtype":       true,
	"co.lang.supertype":     true,
	"co.lang.dependentType": true,
}

// parseTrailingItems consumes whatever follows a complete package source file, so that a file
// with extra declarations still produces one diagnostic per item rather than stalling.
func (p *parser) parseTrailingItems() []ast.Stmt {
	if traceEnabled {
		defer p.traceEnd(p.traceBegin())
	}

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
