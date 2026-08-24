package parser

import (
	"github.com/samkrao/fo-lang/src/importcheck"
	"github.com/samkrao/fo-lang/src/scanlex"
)

// Import-surface scanning.
//
// Whole-program import checking needs one thing from every file in the project: its
// import edges. It does not need the file's bodies.
//
// ScanImportSurface reads exactly that much. Parsing only the preamble and the declaration
// header keeps a project-wide pass cheap, and — more importantly — keeps it robust: a syntax
// error in the body of some unrelated file must not prevent the driver from detecting an import
// cycle elsewhere.

// ScanImportSurface parses a file's preamble and declaration header and returns its import
// surface.
//
// packagePath is the file's package identity relative to the domain that owns it.
//
// Diagnostics are deliberately discarded. This is a lookahead pass over the whole project, so a
// malformed file is reported when it is properly parsed rather than once per project scan. The
// returned record is best-effort: a file whose preamble cannot be read contributes no edges.
func ScanImportSurface(source string, basename string, stem string, packagePath string, atRoot bool) importcheck.File {
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

	// The library-surface fields are left unset for the reason given on
	// parser.importFile: the surface form they described has been withdrawn, and
	// re-pointing them at `src/component.fol` is a separate change.
	return importcheck.File{
		Name:        basename,
		PackagePath: packagePath,
		Imports:     p.imports,
	}
}

// scanHeader parses as much of the file as the import surface needs: the preamble,
// which is where every import directive is written.
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
}
