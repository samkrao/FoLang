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

	// Tokenize-only mode skips import validation and AST construction, but it still
	// performs operator bootstrap. A project-local symbolic spelling cannot be
	// tokenized correctly without the project operator bootstrap catalog.
	if stopAt == "Tokens" {
		configuration := parseConfiguration{}
		packagePath := ""
		projectLabel := rootDir
		proj, discoverErr := project.Discover(sourceFile, rootDir)
		if discoverErr != nil {
			return filename, "", "", false, fmt.Errorf("discovering project for tokenization: %w", discoverErr)
		}
		bootstrap := loadProjectOperatorBootstrap(proj.Root)
		if targetErr := operatorSourceTargetError(sourceFile, bootstrap); targetErr != nil {
			return filename, "", "", false, targetErr
		}
		if len(bootstrap.Findings) > 0 {
			reportFindings(bootstrap.Findings)
		}
		configuration.locationKnown = true
		configuration.operators = bootstrap.Declarations
		projectLabel = projectRootLabel(proj, rootDir)
		for _, f := range proj.Files {
			if f.Path == sourceFile {
				packagePath = f.PackagePath
				configuration.atRoot = f.AtRoot
				break
			}
		}
		_, tokens, _, _ := parseIntoConfigured(nil, string(sourceBytes), projectLabel, filename, basename, packagePath, "program", "program", false, configuration)
		encoded, marshalErr := json.Marshal(tokens)
		if marshalErr != nil {
			return filename, "", "", false, marshalErr
		}
		fmt.Println(string(encoded))
		return filename, "", string(encoded), false, nil
	}

	// Pass 1: the project-wide import checks.
	proj, packagePath, atRoot, projectOperators, buildLibs, discoverErr := checkProjectImports(sourceFile, rootDir)
	if discoverErr != nil {
		return filename, "", "", false, discoverErr
	}

	// Pass 2: fully parse the requested file. The graph is passed so that the parser records
	// its edges rather than re-running the per-file import checks that pass 1 already did.
	start := time.Now()
	root, _, ctx, fileBuildLibs := parseIntoConfigured(
		importcheck.NewGraph(),
		string(sourceBytes),
		projectRootLabel(proj, rootDir),
		filename,
		basename,
		packagePath,
		"program",
		"program",
		true,
		parseConfiguration{locationKnown: proj != nil, atRoot: atRoot, operators: projectOperators},
	)
	// Progress goes to stderr. This is a library package: a consumer that speaks a
	// protocol over stdout — a language server, most obviously — must be able to
	// call it without the frontend corrupting that stream.
	fmt.Fprintf(os.Stderr, "parsed %s in %v\n", basename, time.Since(start))

	serialized, err := serializeAST(root, ctx, binary)
	if err != nil {
		return filename, "", "", buildLibs || fileBuildLibs, err
	}
	return filename, "", serialized, buildLibs || fileBuildLibs, nil
}

// checkProjectImports runs the whole-project import checks and returns the discovered project,
// the requested file's package identity, and whether any library must be built.
//
// Project discovery also succeeds for a loose source by returning a one-file project.
// Therefore an error here is a real configuration, filesystem, or root/target
// relationship failure and must reach the caller rather than silently disabling
// bootstrap and project-wide validation.
func checkProjectImports(sourceFile string, rootDir string) (*project.Project, string, bool, []operatorDeclaration, bool, error) {

	proj, err := project.Discover(sourceFile, rootDir)
	if err != nil {
		return nil, "", false, nil, false, fmt.Errorf("discovering project: %w", err)
	}

	bootstrap := loadProjectOperatorBootstrap(proj.Root)
	if targetErr := operatorSourceTargetError(sourceFile, bootstrap); targetErr != nil {
		return nil, "", false, nil, false, targetErr
	}
	scanned := make([]importcheck.File, 0, len(proj.Files))
	surfaces := make([]declarationSurface, 0, len(proj.Files))
	packagePath := ""
	atRoot := false
	operators := append([]operatorDeclaration(nil), bootstrap.Declarations...)
	buildLibs := false

	for _, f := range proj.Files {
		// srclib/operators/ is a second grammar root, not a package and not an
		// importable library. Exclude the whole slot even when its library.fol is
		// absent, so nothing there is ever read as ordinary source.
		if pathWithin(f.Path, bootstrap.Area) {
			continue
		}
		source, readErr := os.ReadFile(f.Path)
		if readErr != nil {
			continue // an unreadable file is reported when it is compiled
		}

		record := ScanImportSurface(string(source), f.Base, f.Stem, f.PackagePath, f.AtRoot, f.LibrarySlot)
		scanned = append(scanned, record)
		surfaces = append(surfaces, scanDeclarationSurface(string(source), f))
		if f.Path == sourceFile {
			packagePath = f.PackagePath
			atRoot = f.AtRoot
		}
		if record.IsLibrarySurface || hasSourceLibraryImport(record) {
			buildLibs = true
		}
	}

	// The standardized domains are a project-wide fact, so they are checked here
	// alongside the import relationships rather than by any one file's parse.
	findings := append([]error(nil), proj.Layout.Findings...)
	findings = append(findings, bootstrap.Findings...)
	findings = append(findings, importcheck.ValidateProject(scanned)...)
	findings = append(findings, validateOperatorCompanions(surfaces)...)
	if len(findings) > 0 {
		reportFindings(findings)
	}
	return proj, packagePath, atRoot, operators, buildLibs, nil
}

// operatorSourceTargetError keeps srclib/operators/ on its dedicated grammar root.
// Files there are bootstrap inputs, never ordinary token or compilation targets, and
// produce no artifact.
func operatorSourceTargetError(sourceFile string, bootstrap projectOperatorBootstrap) error {

	if !pathWithin(sourceFile, bootstrap.Area) {
		return nil
	}
	return fmt.Errorf(
		"%s is inside the operator-source area %s; source-only bootstrap files cannot be compiled or tokenized directly",
		sourceFile,
		bootstrap.Area,
	)
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
