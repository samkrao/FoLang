package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/importcheck"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// Import-surface scanning.
//
// Whole-program import checking needs one thing from every file in the project: its import
// edges, plus enough of its header to know whether it is a library surface and of what kind.
// It does not need the file's bodies.
//
// ScanImportSurface reads exactly that much. Parsing only the preamble and the declaration
// header keeps a project-wide pass cheap, and — more importantly — keeps it robust: a syntax
// error in the body of some unrelated file must not prevent the driver from detecting an import
// cycle elsewhere.

// ScanImportSurface parses a file's preamble and declaration header and returns its import
// surface.
//
// packagePath is the file's package identity relative to the domain that owns it, and
// librarySlot is the srclib/ child it belongs to — "ffi", "system", "advanced",
// "dynamicvmrt" — or "" for a file under src/.
//
// The slot is what a source library is IMPORTED BY. The fixed `library.fol` name supplies
// no identity, and a source library's implementation packages never enter the owning
// project's ordinary index, so the slot is the only name a consumer can write
// (docs/language-ref.md, "Import Directive Fields").
//
// Diagnostics are deliberately discarded. This is a lookahead pass over the whole project, so a
// malformed file is reported when it is properly parsed rather than once per project scan. The
// returned record is best-effort: a file whose preamble cannot be read contributes no edges.
func ScanImportSurface(source string, basename string, stem string, packagePath string, atRoot bool, librarySlot string) importcheck.File {
	// The surface pass intentionally ignores bodies and diagnostics. A quiet
	// whole-run scan prevents an unrelated body's custom operator from failing
	// before the configured bootstrap catalog is installed by the full driver.
	toks := normalizeTokens(scanlex.TokenizeQuiet(normalizeLineEndings(source), basename))

	p, _ := newParser(toks)
	p.file = fileinfo{
		Basename:      basename,
		PackagePath:   packagePath,
		LocationKnown: true,
		AtRoot:        atRoot,
		Source:        classifySourceFilename(basename),
	}

	p.scanHeader()

	file := importcheck.File{
		Name:             basename,
		PackagePath:      packagePath,
		IsLibrarySurface: p.unit == unitLibrary,
		LibraryName:      p.libraryName,
		LibraryType:      p.libraryType,
		Imports:          p.imports,
	}

	// A surface's owned subtree depends on which domain it sits in. `src/library.fol`
	// is the standalone packaged-library project surface, so every package below src/
	// is internal to it. `srclib/<slot>/library.fol` owns only its own slot, and is
	// named by that slot rather than by any path.
	if file.IsLibrarySurface {
		if librarySlot != "" {
			file.LibraryPath = librarySlot
		} else {
			file.LibraryPath = importcheck.WholeProject
		}
	}
	return file
}

// scanHeader parses as much of the file as the import surface needs: the preamble, and for a
// library surface the declaration name, kind and options that carry its type.
//
// It runs under a recovery point because a malformed file must yield whatever was read so far
// rather than propagating a bailout to the caller.
func (p *parser) scanHeader() {
	defer func() {
		if r := recover(); r != nil {
			if _, isBailout := r.(bailout); !isBailout {
				panic(r)
			}
		}
	}()

	p.parseFilePreamble()
	p.unit = p.classifyCompilationUnit()

	if p.unit != unitLibrary {
		return
	}

	// Read the surface header far enough to capture its name and declared type. The body is
	// left unparsed.
	annotations := p.parseAnnotations()
	declName := p.parseFilenameDerivedName("a library surface declaration")

	if !p.atBuiltinKind("co.lang.library") {
		return
	}
	p.advance() // "co.lang.library"

	options := p.parseOptionalKindOptions()

	libType := firstOptionString(options, "type")
	if libType == "" {
		libType = annotations.optionString("@co.dap.library", "type")
	}
	if libType == "" {
		libType = "application"
	}

	p.libraryName = declName.Logical
	p.libraryType = libType

	// Directive Placement moved a surface's imports into the file preamble, which
	// parseFilePreamble above has already read. The body walk is kept because the
	// project pass must not lose the dependency edges of a surface still written
	// the old way; a conforming file simply has none to find.
	p.scanLibraryBodyImports()
}

// scanLibraryBodyImports walks only the outer brace level of a library body and
// parses each import directive it finds there. Nested braces are skipped, keeping
// syntax errors in unrelated member bodies out of the project-wide import pass.
func (p *parser) scanLibraryBodyImports() {
	if !p.acceptOp("=") || !p.accept(scanlex.OPEN_CURLY) {
		return
	}

	depth := 1
	for !p.atEOF() && depth > 0 {
		if depth == 1 && p.atFileDirective() && p.lexeme() == "@co.ddap.import" {
			p.parseImportDirective()
			continue
		}

		switch p.kind() {
		case scanlex.OPEN_CURLY:
			depth++
		case scanlex.CLOSE_CURLY:
			depth--
		}
		p.advance()
	}
}

// logicalPathOf joins a folder's package path and a file's stem into the dot path that names
// the file itself, which is how a source library is imported.
func logicalPathOf(packagePath, stem string) string {
	if packagePath == "" {
		return stem
	}
	return packagePath + "." + stem
}
