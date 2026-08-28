package parser

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/samkrao/fo-lang/src/ast"
	symboltable "github.com/samkrao/fo-lang/src/context"
	"github.com/samkrao/fo-lang/src/foerrors"
	"github.com/samkrao/fo-lang/src/helpers"
	"github.com/samkrao/fo-lang/src/importcheck"
	"github.com/samkrao/fo-lang/src/project"
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
// It returns the file's base name, the path of the JSON artifact it wrote (empty
// for binary output), the serialized AST, whether libraries must be built, and any
// error.
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
	if DEBUG_TRACE {
		resetDebugTraceEvents()
	}

	// An explicit project-root compilation prepares isolated components and
	// compiled artifacts before any primary-src import scan or target parse.
	if rootDir != "" && stopAt != "Tokens" {
		bootstrap := loadProjectOperatorBootstrap(rootDir)
		if targetErr := operatorSourceTargetError(sourceFile, bootstrap); targetErr != nil {
			return filename, "", "", false, targetErr
		}
		prepared, prepareErr := PrepareProjectRoot(sourceFile, rootDir)
		if prepareErr != nil {
			return filename, "", "", false, fmt.Errorf("preparing project: %w", prepareErr)
		}
		if len(prepared.Findings) > 0 {
			messages := make([]string, 0, len(prepared.Findings))
			for _, finding := range prepared.Findings {
				messages = append(messages, finding.Error())
			}
			return filename, "", "", false, fmt.Errorf("preparing project:\n%s", strings.Join(messages, "\n"))
		}
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

	// Pass 2: build the tree. With an explicit project root the unit is the
	// PROJECT — every file parsed into one scope model and arranged by the layout
	// that produced them — because a package spans its folder and a struct spans
	// its companion, so no single file is a complete declaration. Without one
	// there is no evidence of the project's extent, and the requested file is all
	// there is to parse.
	start := time.Now()
	if rootDir != "" {
		return compileProject(rootDir, proj, filename, binary, buildLibs, start, basename)
	}
	// The collecting entry point rather than the batch one, because the artifact
	// needs the whole scope graph. parseIntoConfigured hands back only the ROOT
	// context, and a context carries its symbol table by id rather than by value,
	// so an artifact built from it would name a symbol table it does not contain.
	// Diagnostics stay fatal here exactly as before.
	parsed := parseCollecting(
		importcheck.NewGraph(),
		string(sourceBytes),
		projectRootLabel(proj, rootDir),
		filename,
		basename,
		packagePath,
		true,
		parseConfiguration{locationKnown: proj != nil, atRoot: atRoot, operators: projectOperators},
	)
	if len(parsed.Diagnostics) > 0 {
		foerrors.HandleErrors(parsed.Diagnostics...)
	}
	root, ctx, fileBuildLibs := parsed.Root, parsed.Context, parsed.BuildLibraries
	// Progress goes to stderr. This is a library package: a consumer that speaks a
	// protocol over stdout — a language server, most obviously — must be able to
	// call it without the frontend corrupting that stream.
	fmt.Fprintf(os.Stderr, "parsed %s in %v\n", basename, time.Since(start))

	serialized, artifactPath, err := serializeAST(root, ctx, parsed.Symbols, binary, astArtifact{
		Root: projectArtifactRoot(proj, rootDir),
		Stem: filename,
	})
	if err != nil {
		return filename, "", "", buildLibs || fileBuildLibs, err
	}
	if artifactPath != "" {
		fmt.Fprintf(os.Stderr, "wrote %s\n", artifactPath)
	}
	return filename, artifactPath, serialized, buildLibs || fileBuildLibs, nil
}

// compileProject parses a whole project and serializes it as one artifact.
//
// The artifact keeps the name of the REQUESTED file, so that a build invoked per
// file still writes where its caller expects; what changed is its contents, which
// now describe the project the file belongs to rather than the file alone.
func compileProject(rootDir string, proj *project.Project, filename string, binary bool, buildLibs bool, start time.Time, basename string) (string, string, string, bool, error) {
	root, diagnostics, err := ParseProject(rootDir)
	if err != nil {
		return filename, "", "", buildLibs, err
	}
	if len(diagnostics) > 0 {
		foerrors.HandleErrors(diagnostics...)
	}

	stmt, isProject := root.(ast.ProjectStmt)
	if !isProject {
		return filename, "", "", buildLibs, fmt.Errorf("parsing project %s: no project statement was produced", rootDir)
	}
	fmt.Fprintf(os.Stderr, "parsed project %s in %v\n", rootDir, time.Since(start))

	serialized, artifactPath, err := serializeAST(root, ProjectContext(stmt.FolangSymbols), stmt.FolangSymbols, binary, astArtifact{
		Root: projectArtifactRoot(proj, rootDir),
		Stem: filename,
	})
	if err != nil {
		return filename, "", "", buildLibs, err
	}
	if artifactPath != "" {
		fmt.Fprintf(os.Stderr, "wrote %s\n", artifactPath)
	}
	return filename, artifactPath, serialized, buildLibs, nil
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
		// components/operators/ is the bootstrap area, not a package and not an
		// importable component. Exclude the whole directory even when its
		// component.fol is absent, so nothing there is ever read as ordinary source.
		if pathWithin(f.Path, bootstrap.Area) {
			continue
		}
		source, readErr := os.ReadFile(f.Path)
		if readErr != nil {
			continue // an unreadable file is reported when it is compiled
		}

		record := ScanImportSurface(string(source), f.Base, f.Stem, f.PackagePath, f.AtRoot)
		scanned = append(scanned, record)
		surfaces = append(surfaces, scanDeclarationSurface(string(source), f))
		if f.Path == sourceFile {
			packagePath = f.PackagePath
			atRoot = f.AtRoot
		}
		if record.IsLibrarySurface {
			buildLibs = true
		}
	}

	// The standardized domains are a project-wide fact, so they are checked here
	// alongside the import relationships rather than by any one file's parse.
	_, currentLayoutFindings := project.ValidateCompilationRoot(proj.Root)
	findings := append([]error(nil), currentLayoutFindings...)
	findings = append(findings, bootstrap.Findings...)
	findings = append(findings, importcheck.ValidateProject(scanned)...)
	findings = append(findings, validateOperatorCompanions(surfaces)...)
	if len(findings) > 0 {
		reportFindings(findings)
	}
	return proj, packagePath, atRoot, operators, buildLibs, nil
}

// operatorSourceTargetError keeps components/operators/ on the bootstrap path.
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

// projectArtifactRoot returns the directory whose build/ domain receives this
// file's frontend artifact, or "" when there is no project to own one.
//
// The root has to be KNOWN, not guessed. Discover succeeds for a loose source
// file by returning a one-file project rooted at that file's own directory, and
// treating that as a project root puts build/ wherever the file happens to sit —
// `src/hr/build/` for a package file, which is a compiler-managed directory
// inside a package. "Project Layout" makes build/ one of four standardized ROOT
// domains and requires every direct entry under src/ to be a package directory,
// so that location is not a build/ domain at all.
//
// MarkerFound is exactly the distinction needed, and project.Layout already draws
// the same line for the same reason: the fallback path "has no evidence of the
// project's extent". With no evidence, no artifact is written and the caller
// still receives the serialized envelope.
func projectArtifactRoot(proj *project.Project, rootDir string) string {

	if rootDir != "" {
		return rootDir
	}
	if proj != nil && proj.MarkerFound {
		return proj.Root
	}
	return ""
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

// serializedAST is the envelope written for a parsed file: one projected AST
// alongside the complete ID-addressable symbol graph.
//
// Both halves are the point. The AST alone does not describe the program a later
// phase has to consume — names, scopes and their relationships live in the
// context — so the two are emitted together under one root rather than as two
// artifacts that could drift apart.
type serializedAST struct {
	SymbolFormatVersion int `json:"symbolFormatVersion"`
	// Symbols is the whole scope graph: every symbol table and every context of
	// this compilation unit, each keyed by the id the tree and the contexts refer
	// to. Without it the ids in Context and in the AST resolve to nothing, and the
	// artifact describes a program whose names cannot be looked up.
	//
	// Its tables are the SCOPES, not yet their contents: declaration binding is
	// the semantic pass's work, so a symbol read out of this artifact carries
	// State "UNRESOLVED". What the parse does fix is the shape — a context per
	// non-literal brace block, and a further symbol-table segment wherever a
	// variable declaration follows a statement (scope.go, docs/language-ref.md
	// Appendix B). AST nodes carry durable SymbolIds. Their canonical records
	// retain the SymbolTableId of the segment active at each source position,
	// which keeps deferred lookup from seeing declarations introduced later.
	Symbols *symboltable.FolangSymbols `json:"FolangSymbols"`
	AST     any                        `json:"AST"`
}

// astArtifact names where the JSON artifact for one parsed file is written.
//
// Root is the project root and Stem the artifact basename without its extension.
// A zero value disables the write, which is what an in-process caller wants: a
// language server parses a buffer per keystroke and must not touch the project
// tree to do it.
type astArtifact struct {
	Root string
	Stem string
}

// astArtifactExtension is the suffix of the JSON frontend artifact.
//
// The AST and the symbol table share one file, so the name says "ast" and the
// envelope carries both.
const astArtifactExtension = ".ast.json"

// serializeAST renders the parsed tree and, for JSON output, writes it to disk.
//
// The tree is walked through ast.Treevistor first, which is the hook later phases use to lower an
// AST node to its mid-level form.
//
// The encoded envelope is both RETURNED and written. The return value is what an
// embedding caller consumes without touching the filesystem; the file is what the
// backend reads, and "Compiler and Backend" fixes its home: "the frontend/backend
// interchange artifact is written beneath the reserved root-level `build/`
// domain". `build/` is compiler-managed and "the compiler may create it when
// absent", which is why the write creates the directory rather than requiring it.
//
// Binary output writes nothing. The protobuf encoding belongs to the
// serialization layer, and emitting a JSON file under a name that promised
// protobuf would hand the backend an artifact its contract does not describe.
func serializeAST(root ast.Stmt, ctx *symboltable.Context, symbols *symboltable.FolangSymbols, binary bool, artifact astArtifact) (string, string, error) {

	if root == nil {
		return "", "", nil
	}

	if symbols != nil {
		if symbols.RootContextId == "" && ctx != nil {
			symbols.RootContextId = ctx.Id
		}
	}
	projectedAST := projectAST(ast.Treevistor(root), symbols)
	envelope := serializedAST{
		SymbolFormatVersion: symboltable.SymbolFormatVersion,
		Symbols:             symbols,
		AST:                 projectedAST,
	}

	if binary {
		// Protobuf output is produced by the serialization layer, which is not part of the
		// parser; JSON is emitted until that path is wired up.
		return "", "", fmt.Errorf("binary AST output is not implemented by the parser; run without -b")
	}

	encoded, err := helpers.Marshal(envelope)
	if err != nil {
		return "", "", err
	}

	written, writeErr := writeASTArtifact(artifact, encoded)
	if writeErr != nil {
		return "", "", writeErr
	}
	if _, traceErr := writeDebugTraceArtifact(artifact); traceErr != nil {
		return "", "", traceErr
	}
	return string(encoded), written, nil
}

// projectAST removes concrete symbol records from AST nodes. Each Symb field is
// represented only by SymbolId; the complete portable record lives once in the
// envelope's SymbolsById registry.
func projectAST(input any, symbols *symboltable.FolangSymbols) any {
	return projectASTValue(reflect.ValueOf(input), symbols)
}

func projectASTValue(value reflect.Value, symbols *symboltable.FolangSymbols) any {
	if !value.IsValid() {
		return nil
	}
	if value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil
		}
		if info, ok := value.Interface().(symboltable.SymbolInfo); ok {
			if symbols != nil {
				symbols.RegisterSymbol(info)
			}
			return info.GetSymbolID()
		}
		return projectASTValue(value.Elem(), symbols)
	}
	switch value.Kind() {
	case reflect.Struct:
		out := map[string]any{}
		type_ := value.Type()
		for i := 0; i < value.NumField(); i++ {
			fieldType := type_.Field(i)
			if !fieldType.IsExported() {
				continue
			}
			name := fieldType.Name
			if name == "Symb" {
				name = "SymbolId"
			}
			out[name] = projectASTValue(value.Field(i), symbols)
		}
		return out
	case reflect.Slice, reflect.Array:
		out := make([]any, value.Len())
		for i := range out {
			out[i] = projectASTValue(value.Index(i), symbols)
		}
		return out
	case reflect.Map:
		out := map[string]any{}
		iter := value.MapRange()
		for iter.Next() {
			out[fmt.Sprint(iter.Key().Interface())] = projectASTValue(iter.Value(), symbols)
		}
		return out
	default:
		if value.CanInterface() {
			return value.Interface()
		}
		return nil
	}
}

// writeASTArtifact writes the encoded envelope beneath the project's build/ domain
// and returns the path it wrote, or "" when the caller supplied no destination.
//
// A failure here is returned rather than swallowed. The artifact is the frontend's
// output, so a compilation that could not produce one has not succeeded, and
// reporting the parse as complete while the backend finds nothing to read would
// move the failure somewhere with no source location to name.
func writeASTArtifact(artifact astArtifact, encoded []byte) (string, error) {

	if artifact.Root == "" || artifact.Stem == "" {
		return "", nil
	}

	directory := filepath.Join(artifact.Root, project.BuildDomain)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return "", fmt.Errorf("creating the %s domain: %w", project.BuildDomain, err)
	}

	path := filepath.Join(directory, artifact.Stem+astArtifactExtension)
	if err := os.WriteFile(path, encoded, 0o644); err != nil {
		return "", fmt.Errorf("writing the frontend artifact: %w", err)
	}
	return path, nil
}
