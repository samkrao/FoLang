package parser

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/foerrors"
	"github.com/samkrao/fo-lang/frontend/src/helpers"
	"github.com/samkrao/fo-lang/frontend/src/importcheck"
	"github.com/samkrao/fo-lang/frontend/src/project"
)

// The compiler driver.
//
// Focmain compiles one requested file, but import checking is a whole-program concern: a cycle
// cannot be detected from one file because the edge that closes the loop is declared elsewhere,
// and the library-boundary rules are stated in terms of project layout. So the driver runs in
// two passes:
//
//  1. A project pass. Discover the project, scan every source file's import surface — the
//     preamble and declaration header only, not the bodies — and run every import-relationship
//     check over the result.
//
//  2. A file pass. Fully parse the requested file and serialize its AST.
//
// The scan in pass 1 is cheap by construction, which is what makes running it on every
// invocation reasonable. It also means a syntax error inside some unrelated file's body does not
// prevent a cycle elsewhere from being reported.

// Focmain reads, checks and parses a FoLang source file.
//
// It returns the file's base name, a reserved second value, the serialized AST, whether libraries
// must be built, and any error.
//
// stopAt selects an early exit: "Tokens" stops after tokenizing and prints the token stream.
// binary requests protobuf output rather than JSON. rootDir names the project root explicitly;
// when empty the driver locates it from the project marker file.
func Focmain(fname string, binary bool, singleton bool, stopAt string, toast bool, rootDir string) (string, string, string, bool, error) {
	sourceFile, err := filepath.Abs(fname)
	if err != nil {
		return "", "", "", false, err
	}

	sourceBytes, err := os.ReadFile(sourceFile)
	basename := filepath.Base(sourceFile)
	filename := strings.TrimSuffix(basename, filepath.Ext(basename))
	if err != nil {
		return filename, "", "", false, err
	}

	// Tokenize-only mode short-circuits both passes: it is a lexer-level request and has no
	// use for project context.
	if stopAt == "Tokens" {
		_, tokens, _, _ := Parse(string(sourceBytes), rootDir, filename, basename, "", "program", "program", false)
		encoded, marshalErr := json.Marshal(tokens)
		if marshalErr != nil {
			return filename, "", "", false, marshalErr
		}
		fmt.Println(string(encoded))
		return filename, "", string(encoded), false, nil
	}

	// Pass 1: the project-wide import checks.
	proj, packagePath, buildLibs := checkProjectImports(sourceFile, rootDir)

	// Pass 2: fully parse the requested file. The graph is passed so that the parser records
	// its edges rather than re-running the per-file import checks that pass 1 already did.
	start := time.Now()
	root, _, ctx, fileBuildLibs := ParseInto(
		importcheck.NewGraph(),
		string(sourceBytes),
		projectRootLabel(proj, rootDir),
		filename,
		basename,
		packagePath,
		"program",
		"program",
		true,
	)
	fmt.Printf("parsed %s in %v\n", basename, time.Since(start))

	serialized, err := serializeAST(root, ctx, binary)
	if err != nil {
		return filename, "", "", buildLibs || fileBuildLibs, err
	}
	return filename, "", serialized, buildLibs || fileBuildLibs, nil
}

// checkProjectImports runs the whole-project import checks and returns the discovered project,
// the requested file's package identity, and whether any library must be built.
//
// Discovery failure is not fatal. Without a project the driver falls back to checking the single
// requested file, which is the behaviour a one-off invocation on a loose file needs.
func checkProjectImports(sourceFile string, rootDir string) (*project.Project, string, bool) {
	proj, err := project.Discover(sourceFile, rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "skipping project-wide import checks: %v\n", err)
		return nil, "", false
	}

	scanned := make([]importcheck.File, 0, len(proj.Files))
	packagePath := ""
	buildLibs := false

	for _, f := range proj.Files {
		source, readErr := os.ReadFile(f.Path)
		if readErr != nil {
			continue // an unreadable file is reported when it is compiled
		}

		record := ScanImportSurface(string(source), f.Base, f.Stem, f.PackagePath, f.AtRoot)
		scanned = append(scanned, record)

		if f.Path == sourceFile {
			packagePath = f.PackagePath
		}
		if record.IsLibrarySurface || hasSourceLibraryImport(record) {
			buildLibs = true
		}
	}

	if findings := importcheck.ValidateProject(scanned); len(findings) > 0 {
		reportFindings(findings)
	}
	return proj, packagePath, buildLibs
}

// hasSourceLibraryImport reports whether a file imports a library from source, which obliges the
// driver to build that library before its consumers.
func hasSourceLibraryImport(f importcheck.File) bool {
	for _, imp := range f.Imports {
		if imp.SrcLibrary {
			return true
		}
	}
	return false
}

// reportFindings hands import-check findings to the shared error handler, which prints them and
// terminates the build.
func reportFindings(findings []error) {
	diags := make([]helpers.ErrorInterface, 0, len(findings))
	for _, f := range findings {
		if diag, ok := f.(helpers.ErrorInterface); ok {
			diags = append(diags, diag)
		}
	}
	if len(diags) > 0 {
		foerrors.HandleErrors(diags...)
	}
}

// projectRootLabel returns the name the parser records as the compilation root.
func projectRootLabel(proj *project.Project, rootDir string) string {
	if rootDir != "" {
		return rootDir
	}
	if proj != nil {
		return filepath.Base(proj.Root)
	}
	return ""
}

// serializedAST is the envelope written out for a parsed file: the root scope alongside the tree
// itself.
type serializedAST struct {
	Context *symboltable.Context `json:"SymbolTable"`
	AST     ast.SET              `json:"AST"`
}

// serializeAST renders the parsed tree.
//
// The tree is walked through ast.Treevistor first, which is the hook later phases use to lower an
// AST node to its mid-level form.
func serializeAST(root ast.Stmt, ctx *symboltable.Context, binary bool) (string, error) {
	if root == nil {
		return "", nil
	}

	envelope := serializedAST{
		Context: ctx,
		AST:     ast.Treevistor(root),
	}

	if binary {
		// Protobuf output is produced by the serialization layer, which is not part of the
		// parser; JSON is emitted until that path is wired up.
		return "", fmt.Errorf("binary AST output is not implemented by the parser; run without -b")
	}

	encoded, err := helpers.Marshal(envelope)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}
