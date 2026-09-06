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
	if rootDir != "" || (proj != nil && proj.MarkerFound) {
		projectRoot := rootDir
		if projectRoot == "" {
			projectRoot = proj.Root
		}
		return compileProject(projectRoot, proj, filename, binary, buildLibs, start, basename)
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

	serialized, artifactPath, err := serializeAST(root, ProjectRootContext(stmt.FolangSymbols), stmt.FolangSymbols, binary, astArtifact{
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
	Protocol            string `json:"protocol"`
	HIRSchema           string `json:"hir_schema"`
	Wire                string `json:"wire"`
	RuntimeOperations   string `json:"runtime_operations"`
	SymbolFormatVersion int    `json:"symbolFormatVersion"`
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

// astArtifact names where the frontend artifact for one parsed file is written.
//
// Root is the project root and Stem the artifact basename without its extension.
// A zero value disables the write, which is what an in-process caller wants: a
// language server parses a buffer per keystroke and must not touch the project
// tree to do it.
type astArtifact struct {
	Root string
	Stem string
}

const (
	astJSONExtension     = ".ast.json"
	astProtobufExtension = ".ast.pb"
	// Kept for JSON-oriented in-package tests and compatibility helpers.
	astArtifactExtension = astJSONExtension
)

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
		switch projectRoot := root.(type) {
		case ast.ProjectStmt:
			if projectRoot.FolContext != nil {
				symbols.FolContext = projectRoot.FolContext
			}
		case *ast.ProjectStmt:
			if projectRoot.FolContext != nil {
				symbols.FolContext = projectRoot.FolContext
			}
		}
		if symbols.RootContextId == "" && ctx != nil {
			symbols.RootContextId = ctx.Id
		}
	}
	emitSpans, spanErr := project.EmitSpans(artifact.Root)
	if spanErr != nil {
		return "", "", spanErr
	}
	projectedAST := projectAST(ast.Treevistor(root), symbols, emitSpans)
	backendConfig, configErr := project.LoadBackendConfig()
	if configErr != nil {
		return "", "", configErr
	}
	// backend-conf.json is authoritative. The legacy binary parameter is retained
	// in the public signature for compatibility but no longer selects the wire.
	_ = binary
	// In-process callers with no project destination historically consume JSON.
	// They do not generate a backend artifact and have no backend-conf.json.
	if artifact.Root == "" {
		backendConfig.Wire = project.WireJSON
	}
	envelope := serializedAST{
		Protocol:            backendConfig.Protocol,
		HIRSchema:           backendConfig.HIRSchema,
		Wire:                backendConfig.Wire,
		RuntimeOperations:   backendConfig.RuntimeOperations,
		SymbolFormatVersion: symboltable.SymbolFormatVersion,
		Symbols:             symbols,
		AST:                 projectedAST,
	}

	var (
		encoded []byte
		err     error
	)
	extension := astProtobufExtension
	if backendConfig.Wire == project.WireJSON {
		encoded, err = helpers.Marshal(envelope)
		extension = astJSONExtension
	} else {
		encoded, err = helpers.MarshalProtobuf(envelope)
	}
	if err != nil {
		return "", "", fmt.Errorf("encoding %s frontend artifact: %w", backendConfig.Wire, err)
	}

	written, writeErr := writeASTArtifact(artifact, encoded, extension)
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
func projectAST(input any, symbols *symboltable.FolangSymbols, emitSpans bool) any {
	return projectASTValue(reflect.ValueOf(input), symbols, emitSpans)
}

func projectASTValue(value reflect.Value, symbols *symboltable.FolangSymbols, emitSpans bool) any {
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
		return projectASTValue(value.Elem(), symbols, emitSpans)
	}
	switch value.Kind() {
	case reflect.Struct:
		type_ := value.Type()
		if type_ == reflect.TypeOf(helpers.Position{}) {
			return projectPosition(value.Interface().(helpers.Position))
		}
		out := map[string]any{}
		for i := 0; i < value.NumField(); i++ {
			fieldType := type_.Field(i)
			if !fieldType.IsExported() {
				continue
			}
			// ProjectStmt keeps these graph references for in-process consumers.
			// The artifact carries their canonical form once beside the AST.
			if fieldType.Name == "FolangSymbols" {
				continue
			}
			// `span: off` in fol-conf.yaml drops the source region from the
			// artifact, the way a build without debug information drops line
			// tables. Nothing else about the tree changes, and the parser's own
			// diagnostics are unaffected: they were emitted from the live nodes
			// long before this projection ran.
			if !emitSpans && fieldType.Name == "Span" {
				continue
			}
			// Parser-only statement identities and the fixed appl.fol scope
			// anchor are not portable symbols. Their AST nodes already state
			// what they are, so do not emit dangling SymbolIds for records the
			// portable symbol graph intentionally omits.
			if fieldType.Name == "Symb" {
				field := value.Field(i)
				if !field.IsNil() {
					if symbol, ok := field.Interface().(symboltable.SymbolInfo); ok {
						if symbols != nil && !symbols.ArtifactCarriesGraphSymbol(symbol) {
							continue
						}
					}
				}
			}
			if fieldType.Name == "SymbolId" && symbols != nil {
				id := value.Field(i).String()
				if symbol := symbols.GetSymbol(id); symbol != nil && !symbols.ArtifactCarriesGraphSymbol(symbol) {
					continue
				}
			}
			name := fieldType.Name
			if name == "Symb" {
				name = "SymbolId"
			}
			projected := projectASTValue(value.Field(i), symbols, emitSpans)
			if fieldType.Name == "Operator" {
				if operator, ok := projected.(map[string]any); ok {
					operator["NodeType_"] = ast.NodeTypeOperator
				}
			}
			out[name] = projected
		}
		// Preserve the node's explicit name. If a construction site omitted it,
		// derive the concrete struct name so the serialized tree remains useful
		// for debugging. A non-empty incorrect value is deliberately not hidden;
		// NodeName coverage tests reject those in parser-produced trees.
		if nodeType := astNodeTypeName(type_); nodeType != "" {
			if nodeName, _ := out["NodeName"].(string); nodeName == "" {
				out["NodeName"] = nodeType
			}
			if category := astNodeCategory(value, nodeType); category != "" {
				out["NodeType_"] = category
			}
			if nodeType == "SymbolExpr" && symbols != nil {
				if id := resolvedLexicalSymbolID(value, symbols); id != "" {
					out["SymbolId"] = id
				}
			}
			if state := astNodeResolutionState(nodeType, out); state != "" {
				out["ResolutionState_"] = state
			}
		}
		return out
	case reflect.Slice, reflect.Array:
		out := make([]any, value.Len())
		for i := range out {
			out[i] = projectASTValue(value.Index(i), symbols, emitSpans)
		}
		return out
	case reflect.Map:
		out := map[string]any{}
		iter := value.MapRange()
		for iter.Next() {
			out[fmt.Sprint(iter.Key().Interface())] = projectASTValue(iter.Value(), symbols, emitSpans)
		}
		return out
	case reflect.Float32, reflect.Float64:
		return helpers.ArtifactFloat(value.Float())
	default:
		if value.CanInterface() {
			return value.Interface()
		}
		return nil
	}
}

// astNodeResolutionState classifies resolution of an AST occurrence. Canonical
// symbols intentionally do not carry this state: one declaration may have both
// resolved and deferred uses in the same artifact.
func astNodeResolutionState(nodeName string, node map[string]any) string {
	switch nodeName {
	case "IntegerLiteral", "NumberLiteral", "StringLiteral", "CharacterLiteral", "BooleanLiteral":
		return string(ast.ResolutionResolved)
	case "BuiltInDataType":
		return string(ast.ResolutionResolved)
	case "SymbolTypeNode":
		if name, _ := node["Value"].(string); name != "" {
			return string(ast.ResolutionPartiallyResolved)
		}
		return string(ast.ResolutionUnresolved)
	case "CompoundType":
		return combinedResolution(node["Left"], node["Right"])
	case "Parameter", "Returns":
		if state := projectedResolution(node["Type_"]); state != "" {
			return state
		}
		if state := projectedResolution(node["Value"]); state != "" {
			return state
		}
		return string(ast.ResolutionUnresolved)
	case "SymbolExpr", "BindVariableExpr":
		if id, _ := node["SymbolId"].(string); id != "" {
			return string(ast.ResolutionResolved)
		}
		return string(ast.ResolutionUnresolved)
	case "CallExpr":
		kind := reflect.ValueOf(node["CallKind"])
		if kind.IsValid() && kind.Kind() >= reflect.Int && kind.Kind() <= reflect.Int64 && kind.Int() == int64(ast.CallUnresolved) {
			return string(ast.ResolutionUnresolved)
		}
		return string(ast.ResolutionPartiallyResolved)
	case "LifecycleCallExpr", "MemberExpr":
		return string(ast.ResolutionPartiallyResolved)
	case "VarDeclarationStmt":
		variable := node
		if basic, ok := node["BasicVarStmt"].(map[string]any); ok {
			variable = basic
		}
		typeName, _ := variable["VarType"].(string)
		if typeName == "" || strings.EqualFold(typeName, "co.lang.infer") {
			return string(ast.ResolutionUnresolved)
		}
		if deferredType(typeName) {
			return string(ast.ResolutionPartiallyResolved)
		}
		// Built-in co.lang types have a fixed meaning in the frontend. A named
		// user type (including a generic parameter) still needs type lookup.
		if !strings.HasPrefix(strings.ToLower(typeName), "co.lang.") {
			return string(ast.ResolutionPartiallyResolved)
		}
		return string(ast.ResolutionResolved)
	case "GroupingExpr":
		return projectedResolution(node["Expr_"])
	case "BinaryExpr", "CommaExpr":
		return combinedResolution(node["Left"], node["Right"])
	case "AssignmentExpr":
		return combinedResolution(node["Assigne"], node["AssignedValue"])
	}
	return ""
}

func projectedResolution(value any) string {
	projected, ok := value.(map[string]any)
	if !ok {
		return ""
	}
	state, _ := projected["ResolutionState_"].(string)
	return state
}

func combinedResolution(left, right any) string {
	leftState := projectedResolution(left)
	rightState := projectedResolution(right)
	if leftState == string(ast.ResolutionResolved) && rightState == string(ast.ResolutionResolved) {
		return string(ast.ResolutionResolved)
	}
	if leftState == string(ast.ResolutionUnresolved) && rightState == string(ast.ResolutionUnresolved) {
		return string(ast.ResolutionUnresolved)
	}
	if leftState == "" && rightState == "" {
		return ""
	}
	return string(ast.ResolutionPartiallyResolved)
}

// projectPosition narrows a source position to the part a consumer of the
// artifact can act on.
//
// A span has to survive into the artifact: Appendix B.7 lists Span on the AST
// nodes it defines, and a diagnostic instance must carry a primary source span
// (docs/language-ref.md, "Diagnostics"), which binds the backend too — a program
// the frontend accepted can still be rejected with UnsupportedBackendFeature, and
// that rejection has to name a place in the source.
//
// What a span CONTAINS is not specified, and two of the six fields are frontend
// bookkeeping that no reader of the artifact can use:
//
//   - Ftxt is the whole source LINE, stored once in Start and again in End, and
//     copied again for every nested node on that line. Its only readers are the
//     caret-underline in helpers/error.go, which runs in-process while the
//     frontend is still holding the file. Anyone holding the source can recover
//     it from Fn and Ln.
//   - Idx is Pos under a second name. Every scanner call site passes lex.pos for
//     both, and Advance moves them together.
//
// Ln, Col, Pos and Fn are kept, which is everything needed to report against
// source or to emit debug line information.
func projectPosition(position helpers.Position) map[string]any {
	return map[string]any{
		"Ln": position.Ln, "Col": position.Col,
		"Pos": position.Pos, "Fn": position.Fn,
	}
}

// astNodeTypeName returns the AST node type's own name, or "" for anything that
// is not an AST node: a span, a symbol record, a parser-side helper.
func astNodeTypeName(type_ reflect.Type) string {
	if !strings.HasSuffix(type_.PkgPath(), "/ast") {
		return ""
	}
	field, ok := type_.FieldByName("NodeName")
	if !ok || field.Type.Kind() != reflect.String {
		return ""
	}
	return type_.Name()
}

// astNodeCategory classifies an AST node independently of its concrete shape
// and independently of the FoLang data type the program assigns to it.
func astNodeCategory(value reflect.Value, nodeName string) string {
	if !value.CanInterface() {
		return ""
	}
	node := value.Interface()
	if strings.HasSuffix(nodeName, "Literal") {
		return ast.NodeTypeLiteral
	}
	switch nodeName {
	case "SymbolExpr", "BindVariableExpr":
		return ast.NodeTypeSymbol
	}
	if _, ok := node.(ast.Type); ok {
		return ast.NodeTypeType
	}
	if _, ok := node.(ast.Expr); ok {
		return ast.NodeTypeExpression
	}
	if _, ok := node.(ast.Stmt); ok {
		return ast.NodeTypeStatement
	}
	return ""
}

// resolvedLexicalSymbolID resolves an ordinary SymbolExpr from the exact
// declaration-order segment recorded when that occurrence was parsed. It never
// manufactures an occurrence symbol: success returns the declaration's ID;
// failure leaves the node for a later import/overload/dynamic resolution phase.
func resolvedLexicalSymbolID(value reflect.Value, symbols *symboltable.FolangSymbols) string {
	name := value.FieldByName("Value")
	symb := value.FieldByName("Symb")
	if !name.IsValid() || !symb.IsValid() || symb.IsNil() {
		return ""
	}
	occurrence, ok := symb.Interface().(*symboltable.ExpressionSymbol)
	if !ok {
		return ""
	}
	return resolvedNameSymbolID(name.String(), occurrence, symbols)
}

func resolvedNameSymbolID(name string, occurrence *symboltable.ExpressionSymbol, symbols *symboltable.FolangSymbols) string {
	if occurrence == nil {
		return ""
	}
	table := symbols.GetSymbolTable(occurrence.SymbolTableId)
	if table == nil {
		return ""
	}
	// Pass one is exclusively lexical. GetDetails follows declaration-order
	// segments and lexical parent contexts, but never follows import links.
	if id := resolveSymbolFromTable(table, name, symbols); id != "" {
		return id
	}

	// Pass two is available only to a composite name. It walks import bindings in
	// the current lexical context and its parents after lexical lookup failed.
	// Imports retain canonical contexts; they never copy symbol tables, and lookup
	// inside the selected target never traverses that target's imports.
	originalParts := strings.Split(name, ".")
	parts := strings.Split(logicalName(name), ".")
	if len(parts) < 2 {
		return ""
	}
	context := symbols.GetContext(table.ContextId)
	visitedProjectRoot := false
	for context != nil {
		if context.Id == symbols.FolContextRootContextID() {
			visitedProjectRoot = true
		}
		if id := resolveImportedSymbol(context.ImportedContextIds, originalParts, parts, symbols); id != "" {
			return id
		}
		context = symbols.GetContext(context.ParentId)
	}
	if !visitedProjectRoot {
		if root := symbols.GetContext(symbols.FolContextRootContextID()); root != nil {
			return resolveImportedSymbol(root.ImportedContextIds, originalParts, parts, symbols)
		}
	}
	return ""
}

func resolveImportedSymbol(imports map[string]string, originalParts, logicalParts []string, symbols *symboltable.FolangSymbols) string {
	target, width := longestImportedContext(imports, logicalParts, symbols)
	if target == nil {
		return ""
	}
	member := strings.Join(originalParts[width:], ".")
	return resolveImportedDeclaration(target, member, symbols, valueSymbolKinds)
}

// longestImportedContext resolves a declared alias or full package qualifier.
// Longest-prefix selection makes company.hr deterministic when company is also
// imported; map iteration order must never influence name resolution.
func longestImportedContext(imports map[string]string, logicalParts []string, symbols *symboltable.FolangSymbols) (*symboltable.Context, int) {
	var selected *symboltable.Context
	selectedWidth := 0
	for importName, contextID := range imports {
		logicalImport := logicalName(importName)
		width := strings.Count(logicalImport, ".") + 1
		if width <= selectedWidth || len(logicalParts) <= width || strings.Join(logicalParts[:width], ".") != logicalImport {
			continue
		}
		if target := symbols.GetContext(contextID); target != nil {
			selected, selectedWidth = target, width
		}
	}
	return selected, selectedWidth
}

func resolveSymbolFromTable(table *symboltable.SymbolTable, name string, symbols *symboltable.FolangSymbols) string {
	if table == nil {
		return ""
	}
	for _, kind := range []symboltable.SymbolsToString{
		symboltable.S_VarSymbol,
		symboltable.S_FunctionSymbol,
		symboltable.S_ClassSymbol,
		symboltable.S_StructSymbol,
		symboltable.S_EnumSymbol,
		symboltable.S_UnionSymbol,
		symboltable.S_ModuleSymbol,
		symboltable.S_InterfaceSymbol,
		symboltable.S_TypeSymbol,
	} {
		if declaration := table.GetDetails(*symbols, name, string(kind)); declaration != nil && declaration.GetSymbolID() != "" {
			return declaration.GetSymbolID()
		}
	}
	return resolveLogicalName(table, name, symbols, map[string]bool{
		string(symboltable.S_VarSymbol): true, string(symboltable.S_FunctionSymbol): true,
		string(symboltable.S_ClassSymbol): true, string(symboltable.S_StructSymbol): true,
		string(symboltable.S_EnumSymbol): true, string(symboltable.S_UnionSymbol): true,
		string(symboltable.S_ModuleSymbol): true, string(symboltable.S_InterfaceSymbol): true,
		string(symboltable.S_TypeSymbol): true,
	})
}

func resolvedTypeSymbolID(node ast.SymbolTypeNode, symbols *symboltable.FolangSymbols) string {
	if node.Symb == nil {
		return ""
	}
	table := symbols.GetSymbolTable(node.Symb.SymbolTableId)
	// As for values, exhaust lexical symbol lookup before considering a qualified
	// import alias.
	if id := resolveTypeFromTable(table, node.Value, symbols); id != "" {
		return id
	}
	originalParts := strings.Split(node.Value, ".")
	logicalParts := strings.Split(logicalName(node.Value), ".")
	if len(logicalParts) < 2 {
		return ""
	}
	context := symbols.GetContext(table.ContextId)
	visitedProjectRoot := false
	for context != nil {
		if context.Id == symbols.FolContextRootContextID() {
			visitedProjectRoot = true
		}
		if id := resolveImportedType(context.ImportedContextIds, originalParts, logicalParts, symbols); id != "" {
			return id
		}
		context = symbols.GetContext(context.ParentId)
	}
	if !visitedProjectRoot {
		if root := symbols.GetContext(symbols.FolContextRootContextID()); root != nil {
			return resolveImportedType(root.ImportedContextIds, originalParts, logicalParts, symbols)
		}
	}
	return ""
}

func resolveImportedType(imports map[string]string, originalParts, logicalParts []string, symbols *symboltable.FolangSymbols) string {
	target, width := longestImportedContext(imports, logicalParts, symbols)
	if target == nil {
		return ""
	}
	return resolveImportedDeclaration(target, strings.Join(originalParts[width:], "."), symbols, typeSymbolKinds)
}

var valueSymbolKinds = map[string]bool{
	string(symboltable.S_VarSymbol): true, string(symboltable.S_FunctionSymbol): true,
	string(symboltable.S_ClassSymbol): true, string(symboltable.S_StructSymbol): true,
	string(symboltable.S_EnumSymbol): true, string(symboltable.S_UnionSymbol): true,
	string(symboltable.S_ModuleSymbol): true, string(symboltable.S_InterfaceSymbol): true,
	string(symboltable.S_TypeSymbol): true,
}

var typeSymbolKinds = map[string]bool{
	string(symboltable.S_TypeSymbol): true, string(symboltable.S_ClassSymbol): true,
	string(symboltable.S_StructSymbol): true, string(symboltable.S_EnumSymbol): true,
	string(symboltable.S_UnionSymbol): true, string(symboltable.S_InterfaceSymbol): true,
	string(symboltable.S_SignatureSymbol): true, string(symboltable.S_TypeclassSymbol): true,
}

// resolveImportedDeclaration searches only the selected imported context's own
// declaration-order tables and ownership parents. It stops before the importing
// project's operational root and never follows ImportedContextIds, preventing a
// component, library, or package import from acquiring consumer-root symbols.
func resolveImportedDeclaration(target *symboltable.Context, name string, symbols *symboltable.FolangSymbols, kinds map[string]bool) string {
	boundary := symbols.FolContextRootContextID()
	for context := target; context != nil && context.Id != boundary; context = symbols.GetContext(context.ParentId) {
		for table := symbols.GetSymbolTable(context.SymbolTable_); table != nil; table = symbols.GetSymbolTable(table.ParentId) {
			for _, declaration := range symbols.Bindings(table.Id) {
				if declaration != nil && kinds[declaration.GetSymbolType()] && logicalName(declaration.GetName()) == logicalName(name) {
					return declaration.GetSymbolID()
				}
			}
			if table.ParentId == "" {
				break
			}
		}
	}
	return ""
}

func resolveTypeFromTable(table *symboltable.SymbolTable, name string, symbols *symboltable.FolangSymbols) string {
	if table == nil {
		return ""
	}
	for _, kind := range []symboltable.SymbolsToString{
		symboltable.S_TypeSymbol, symboltable.S_ClassSymbol, symboltable.S_StructSymbol,
		symboltable.S_EnumSymbol, symboltable.S_UnionSymbol, symboltable.S_InterfaceSymbol,
		symboltable.S_SignatureSymbol, symboltable.S_TypeclassSymbol,
	} {
		if declaration := table.GetDetails(*symbols, name, string(kind)); declaration != nil && declaration.GetSymbolID() != "" {
			return declaration.GetSymbolID()
		}
	}
	return resolveLogicalName(table, name, symbols, map[string]bool{
		string(symboltable.S_TypeSymbol): true, string(symboltable.S_ClassSymbol): true,
		string(symboltable.S_StructSymbol): true, string(symboltable.S_EnumSymbol): true,
		string(symboltable.S_UnionSymbol): true, string(symboltable.S_InterfaceSymbol): true,
		string(symboltable.S_SignatureSymbol): true, string(symboltable.S_TypeclassSymbol): true,
	})
}

func resolveLogicalName(table *symboltable.SymbolTable, name string, symbols *symboltable.FolangSymbols, kinds map[string]bool) string {
	want := logicalName(name)
	visited := map[string]bool{}
	for table != nil && !visited[table.Id] {
		visited[table.Id] = true
		for _, declaration := range symbols.Bindings(table.Id) {
			if declaration != nil && kinds[declaration.GetSymbolType()] && logicalName(declaration.GetName()) == want {
				return declaration.GetSymbolID()
			}
		}
		if table.ParentId != "" {
			table = symbols.GetSymbolTable(table.ParentId)
			continue
		}
		context := symbols.GetContext(table.ContextId)
		if context == nil || context.ParentId == "" {
			break
		}
		parent := symbols.GetContext(context.ParentId)
		if parent == nil {
			break
		}
		table = symbols.GetSymbolTable(parent.SymbolTable_)
	}
	return ""
}

// writeASTArtifact writes the encoded envelope beneath the project's build/ domain
// and returns the path it wrote, or "" when the caller supplied no destination.
//
// A failure here is returned rather than swallowed. The artifact is the frontend's
// output, so a compilation that could not produce one has not succeeded, and
// reporting the parse as complete while the backend finds nothing to read would
// move the failure somewhere with no source location to name.
func writeASTArtifact(artifact astArtifact, encoded []byte, extension string) (string, error) {

	if artifact.Root == "" || artifact.Stem == "" {
		return "", nil
	}

	directory := filepath.Join(artifact.Root, project.BuildDomain)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return "", fmt.Errorf("creating the %s domain: %w", project.BuildDomain, err)
	}

	path := filepath.Join(directory, artifact.Stem+extension)
	if err := os.WriteFile(path, encoded, 0o644); err != nil {
		return "", fmt.Errorf("writing the frontend artifact: %w", err)
	}
	return path, nil
}
