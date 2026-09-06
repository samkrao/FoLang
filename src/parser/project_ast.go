package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/samkrao/fo-lang/src/ast"
	symboltable "github.com/samkrao/fo-lang/src/context"
	"github.com/samkrao/fo-lang/src/helpers"
	"github.com/samkrao/fo-lang/src/importcheck"
	"github.com/samkrao/fo-lang/src/project"
)

// Assembling a project into one tree.
//
// A FILE is not a compilation unit in FoLang. A package spans every file in its
// folder, a struct spans its primary file and its companion unit, and which
// domain a file sits in decides what it may be. So the unit a later phase works
// from is the PROJECT, and this file is what builds it: every source file is
// parsed, and the roots are arranged by the layout that produced them
// (docs/language-ref.md, "Project Layout").
//
// # One scope model, not one per file
//
// Every file is parsed into the SAME FolangSymbols, under one project context.
// Two files in a package must be able to see each other's declarations, which is
// only true if their contexts live in one model; and the ids the tree carries
// resolve against that model, so splitting it would leave the assembled tree
// naming tables it does not contain. The model hangs off the ProjectStmt for the
// same reason.
//
// # Where a file's declarations end up
//
//   - src/appl.fol and src/component.fol are the structural surface, so they become
//     EntryStmt rather than a member of anything.
//   - A file in a subfolder of src/ contributes its declarations to that folder's
//     package. The folder is the package; the file is not a scope.
//   - An ordinary unit is spliced: `_ co.lang.unit` is a file-level wrapper with
//     no scope of its own, so its members become the package's members directly.
//   - A companion unit is folded into the type it belongs to. `Employee.fol`
//     declares the struct and `Employee.comp.unit.fol` carries its associated
//     functions, so the two are one declaration and the members join the struct's
//     own body.
//   - lib/ becomes LibraryStmt, components/ becomes ComponentStmt,
//     each keyed by the name an import uses to reach it.

// ParseProject parses every source file under root and assembles them into one
// ProjectStmt.
//
// Diagnostics are returned rather than reported, so one malformed file does not
// cost the caller the rest of the project. The error return is for a failure to
// discover or read the project at all, which is a different kind of problem: no
// tree can be built from it.
func ParseProject(root string) (ast.Stmt, []helpers.ErrorInterface, error) {
	target, err := projectTarget(root)
	if err != nil {
		return nil, nil, err
	}

	proj, err := project.Discover(target, root)
	if err != nil {
		return nil, nil, fmt.Errorf("discovering project: %w", err)
	}

	assembly, err := newProjectAssembly(proj, root)
	if err != nil {
		return nil, nil, err
	}
	// Discovery already validated the domain layout; those findings are the
	// project-level half of this function's contract and belong with the
	// per-file diagnostics rather than only on the CLI path. A caller given a
	// tree for `src/` holding both structural surfaces, or a malformed lib/,
	// would otherwise be told nothing was wrong with the project's shape.
	//
	// They come first because they describe the project a file was read as part
	// of: a file's own diagnostic is easier to read once the reader knows the
	// layout it was parsed under.
	assembly.diagnostics = append(assembly.diagnostics, layoutDiagnostics(proj.Layout)...)
	inputs, err := project.CompilationInputs(proj.Root, proj.Files)
	if err != nil {
		assembly.diagnostics = append(assembly.diagnostics, projectDiagnostic(err.Error()))
		inputs = fallbackCompilationInputs(proj)
	}
	filesByPath := make(map[string]project.File, len(proj.Files))
	for _, file := range proj.Files {
		filesByPath[filepath.Clean(file.Path)] = file
	}
	// Create every source package context before parsing any source file. Do not
	// publish them to the import resolver yet: components precede src in the
	// dependency order and therefore cannot import application source packages.
	for _, input := range inputs {
		if input.Stage != project.StagePrimarySource || input.PackagePath == "" {
			continue
		}
		assembly.packages.packageOf(input.PackagePath)
	}

	// Compiled libraries are independent bootstrap artifacts. Load their
	// published contexts before parsing any project-owned component.
	for _, input := range inputs {
		if input.Stage == project.StageLibraries {
			assembly.loadCompiledLibrary(input.Path)
		}
	}

	// Components are collected in their fixed dependency order and parsed before
	// primary source. Earlier component surfaces are available to later kinds.
	for _, input := range inputs {
		if input.Stage != project.StageComponents {
			continue
		}
		if file, ok := filesByPath[filepath.Clean(input.Path)]; ok {
			assembly.add(file)
		}
	}
	assembly.parseExternals()
	// Source packages become importable only after every component has been
	// prepared. Their identities were allocated above, so sibling packages can
	// still resolve each other regardless of source-file traversal order.
	for path, pkg := range assembly.packages.byPath {
		assembly.publishImportContext("package:"+path, pkg.context)
	}

	// CompilationInputs orders ordinary file-backed declarations before their
	// companions, ordinary units next, and appl.fol/component.fol last.
	for _, input := range inputs {
		if input.Stage != project.StagePrimarySource {
			continue
		}
		if file, ok := filesByPath[filepath.Clean(input.Path)]; ok {
			assembly.add(file)
		}
	}
	assembly.validatePackageOverloads(assembly.packages)
	assembly.validateStandaloneComponents()
	result := assembly.finish()
	assembly.validateNativeIndirectionAliases(result)
	assembly.validateOrdinarySymbolReferences(result)
	return result, assembly.diagnostics, nil
}

// validateNativeIndirectionAliases enforces the native capability boundary after
// project assembly, where both the source domain and the complete alias RHS are
// known. The syntax parser must not reject co.lang.type itself: arrays, ranges,
// function types and generic specializations use the same declaration form.
func (a *projectAssembly) validateNativeIndirectionAliases(root ast.Stmt) {
	projectNode, ok := root.(ast.ProjectStmt)
	if !ok {
		return
	}
	rootIsNative := false
	if surface, ok := projectNode.EntryStmt.(ast.ComponentDeclarationStmt); ok {
		rootIsNative = surface.Projected && surface.LibraryType == componentKindNative
	}
	if rootIsNative {
		return
	}

	check := func(node ast.Stmt) {
		walkTypeAliases(node, func(decl ast.TypeDeclarationStmt) {
			if decl.Type_ == nil || !containsNativeIndirection(reflect.ValueOf(decl.Type_)) {
				return
			}
			a.diagnostics = append(a.diagnostics, helpers.NewInvalidSyntaxError(
				decl.Span.Start, decl.Span.End,
				fmt.Sprintf("type alias %q contains native pointer/reference/address indirection and is allowed only in a native library or components/native", logicalName(decl.Name))))
		})
	}
	check(projectNode.EntryStmt)
	for _, pkg := range projectNode.PackageStmts {
		check(pkg)
	}
	for kind, component := range projectNode.ComponentStmt {
		if kind != componentKindNative {
			check(component)
		}
	}
}

func walkTypeAliases(root ast.Stmt, visit func(ast.TypeDeclarationStmt)) {
	var walk func(reflect.Value)
	walk = func(value reflect.Value) {
		if !value.IsValid() {
			return
		}
		if value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer {
			if !value.IsNil() {
				walk(value.Elem())
			}
			return
		}
		if value.Kind() == reflect.Struct && value.Type() == reflect.TypeOf(ast.TypeDeclarationStmt{}) {
			visit(value.Interface().(ast.TypeDeclarationStmt))
		}
		switch value.Kind() {
		case reflect.Struct:
			for i := 0; i < value.NumField(); i++ {
				field := value.Type().Field(i).Name
				if field != "FolangSymbols" && field != "Symb" && field != "Span" {
					walk(value.Field(i))
				}
			}
		case reflect.Slice, reflect.Array:
			for i := 0; i < value.Len(); i++ {
				walk(value.Index(i))
			}
		case reflect.Map:
			iter := value.MapRange()
			for iter.Next() {
				walk(iter.Value())
			}
		}
	}
	walk(reflect.ValueOf(root))
}

func containsNativeIndirection(root reflect.Value) bool {
	if !root.IsValid() {
		return false
	}
	if root.Kind() == reflect.Interface || root.Kind() == reflect.Pointer {
		return !root.IsNil() && containsNativeIndirection(root.Elem())
	}
	if root.Kind() == reflect.Struct && root.Type() == reflect.TypeOf(ast.DerivedType{}) {
		derived := root.Interface().(ast.DerivedType)
		switch derived.Form {
		case ast.DerivePointer, ast.DeriveReference, ast.DeriveHeapReference,
			ast.DeriveAddress, ast.DeriveThunk, ast.DeriveWord:
			return true
		}
	}
	switch root.Kind() {
	case reflect.Struct:
		for i := 0; i < root.NumField(); i++ {
			field := root.Type().Field(i).Name
			if field != "Symb" && field != "Span" && containsNativeIndirection(root.Field(i)) {
				return true
			}
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < root.Len(); i++ {
			if containsNativeIndirection(root.Index(i)) {
				return true
			}
		}
	case reflect.Map:
		iter := root.MapRange()
		for iter.Next() {
			if containsNativeIndirection(iter.Value()) {
				return true
			}
		}
	}
	return false
}

// validateOrdinarySymbolReferences runs after every declaration and import
// context has entered the project graph. Ordinary lexical/package/imported names
// must now resolve. Member dispatch remains a later type-directed operation and
// therefore is not represented by SymbolExpr here.
func (a *projectAssembly) validateOrdinarySymbolReferences(root ast.Stmt) {
	seen := map[string]bool{}
	usedImports := map[string]bool{}
	var imports []ast.ImportStmt
	var walk func(reflect.Value)
	walk = func(value reflect.Value) {
		if !value.IsValid() {
			return
		}
		if value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer {
			if value.IsNil() {
				return
			}
			walk(value.Elem())
			return
		}
		if value.Kind() == reflect.Struct && value.Type() == reflect.TypeOf(ast.SymbolExpr{}) {
			node := value.Interface().(ast.SymbolExpr)
			if node.SymbolType_ == "reference" && resolvedLexicalSymbolID(value, a.symbols) == "" {
				name := logicalName(node.Value)
				key := fmt.Sprintf("%s:%d:%s", node.Span.Start.Fn, node.Span.Start.Idx, name)
				if !seen[key] {
					seen[key] = true
					a.diagnostics = append(a.diagnostics, helpers.NewNamedDiagnostic(
						node.Span.Start, node.Span.End, helpers.DiagnosticUnresolvedSymbol,
						"Unresolved Symbol", fmt.Sprintf("%q is neither built in nor declared in the lexical, package, or imported symbol environment", name)))
				}
			} else if node.SymbolType_ == "reference" && node.Symb != nil {
				if key := importUseKey(node.Symb.SymbolTableId, node.Value, a.symbols); key != "" {
					usedImports[key] = true
				}
			}
			return
		}
		if value.Kind() == reflect.Struct && value.Type() == reflect.TypeOf(ast.MemberExpr{}) {
			node := value.Interface().(ast.MemberExpr)
			if receiver, ok := node.Member.(ast.SymbolExpr); ok && receiver.Symb != nil {
				qualified := receiver.Value + "." + node.Property
				if key := importUseKey(receiver.Symb.SymbolTableId, qualified, a.symbols); key != "" {
					if resolvedNameSymbolID(qualified, receiver.Symb, a.symbols) == "" {
						a.diagnostics = append(a.diagnostics, helpers.NewNamedDiagnostic(
							node.Span.Start, node.Span.End, helpers.DiagnosticUnresolvedSymbol,
							"Unresolved Symbol", fmt.Sprintf("imported symbol %q does not exist", logicalName(qualified))))
					} else {
						usedImports[key] = true
					}
					return
				}
			}
			// A non-namespace member needs receiver-type dispatch later. Its
			// receiver still participates in ordinary lexical resolution.
			walk(value.FieldByName("Member"))
			return
		}
		if value.Kind() == reflect.Struct && value.Type() == reflect.TypeOf(ast.SymbolTypeNode{}) {
			node := value.Interface().(ast.SymbolTypeNode)
			if resolvedTypeSymbolID(node, a.symbols) == "" {
				name := logicalName(node.Value)
				key := fmt.Sprintf("%s:%d:type:%s", node.Span.Start.Fn, node.Span.Start.Idx, name)
				if !seen[key] {
					seen[key] = true
					a.diagnostics = append(a.diagnostics, helpers.NewNamedDiagnostic(
						node.Span.Start, node.Span.End, helpers.DiagnosticUnresolvedSymbol,
						"Unresolved Type", fmt.Sprintf("type %q is neither built in nor declared in the lexical, package, or imported symbol environment", name)))
				}
			} else if node.Symb != nil {
				if key := importUseKey(node.Symb.SymbolTableId, node.Value, a.symbols); key != "" {
					usedImports[key] = true
				}
			}
			return
		}
		if value.Kind() == reflect.Struct && value.Type() == reflect.TypeOf(ast.ImportStmt{}) {
			imports = append(imports, value.Interface().(ast.ImportStmt))
			return
		}
		switch value.Kind() {
		case reflect.Struct:
			for i := 0; i < value.NumField(); i++ {
				field := value.Type().Field(i).Name
				if field == "Span" || field == "Symb" || field == "FolangSymbols" {
					continue
				}
				walk(value.Field(i))
			}
		case reflect.Slice, reflect.Array:
			for i := 0; i < value.Len(); i++ {
				walk(value.Index(i))
			}
		case reflect.Map:
			iter := value.MapRange()
			for iter.Next() {
				walk(iter.Value())
			}
		}
	}
	walk(reflect.ValueOf(root))
	for _, imported := range imports {
		if imported.Symb == nil {
			continue
		}
		name := imported.Name
		if name == "" {
			name = imported.Package
			if name == "" {
				name = imported.From
			}
			if name == "" {
				name = imported.Component
			}
		}
		table := a.symbols.GetSymbolTable(imported.Symb.SymbolTableId)
		if table == nil {
			continue
		}
		owner := importBindingOwner(a.symbols.GetContext(table.ContextId), name, a.symbols)
		if owner == nil {
			continue
		}
		key := owner.Id + "\x00" + name
		if _, prepared := owner.ImportedContextIds[name]; prepared && !usedImports[key] {
			a.diagnostics = append(a.diagnostics, helpers.NewNamedDiagnostic(
				imported.Span.Start, imported.Span.End, helpers.DiagnosticUnusedImport,
				"Unused Import", fmt.Sprintf("import %q contributes no resolved symbol use", name)))
		}
	}
	a.validatePackageReachability(usedImports)
}

func (a *projectAssembly) validatePackageReachability(usedImports map[string]bool) {
	edges := map[string][]string{}
	for key := range usedImports {
		parts := strings.SplitN(key, "\x00", 2)
		if len(parts) != 2 {
			continue
		}
		owner := a.symbols.GetContext(parts[0])
		if owner == nil {
			continue
		}
		if target := owner.ImportedContextIds[parts[1]]; target != "" {
			from := a.importOwnerContext(owner.Id)
			edges[from] = append(edges[from], target)
		}
	}
	reachable := map[string]bool{a.context.Id: true}
	queue := []string{a.context.Id}
	// A packaged-library export is a root-facing use of the selected package,
	// not an import. Seed those packages explicitly so the unused-package rule
	// does not reject the very surface the library publishes.
	for path, recursive := range exportedPackageSelectors(a.entry) {
		for current := path; current != ""; current = parentPackagePath(current) {
			if pkg := a.packages.byPath[current]; pkg != nil && pkg.context != nil {
				reachable[pkg.context.Id] = true
			}
		}
		if recursive {
			for candidate, pkg := range a.packages.byPath {
				if (candidate == path || strings.HasPrefix(candidate, path+".")) && pkg != nil && pkg.context != nil {
					reachable[pkg.context.Id] = true
				}
			}
		}
	}
	for len(queue) != 0 {
		from := queue[0]
		queue = queue[1:]
		for _, target := range edges[from] {
			if !reachable[target] {
				reachable[target] = true
				queue = append(queue, target)
			}
		}
	}
	for _, path := range sortedPackagePaths(a.packages.byPath) {
		pkg := a.packages.byPath[path]
		if pkg == nil || pkg.context == nil || reachable[pkg.context.Id] {
			continue
		}
		a.diagnostics = append(a.diagnostics, helpers.NewNamedDiagnostic(
			helpers.Position{}, helpers.Position{}, helpers.DiagnosticUnusedSymbol,
			"Unreachable Package", fmt.Sprintf("source package %q is not reachable from the project root through a live resolved import", path)))
	}
}

func exportedPackageSelectors(entry ast.Stmt) map[string]bool {
	component, ok := entry.(ast.ComponentDeclarationStmt)
	if !ok || component.Projected {
		return nil
	}
	selected := map[string]bool{}
	for _, member := range ast.ComponentSurfaceBody(component) {
		directive, ok := member.(ast.DirectiveStmt)
		if !ok || directive.Name != componentExportSelectorName {
			continue
		}
		packages, ok := directive.Parameters["packages"].(map[string]any)
		if !ok {
			continue
		}
		for path, options := range packages {
			recurse := false
			if fields, ok := options.(map[string]any); ok {
				switch value := fields["recurse"].(type) {
				case bool:
					recurse = value
				case string:
					recurse = value == "true" || value == "co.const.true"
				}
			}
			selected[path] = recurse
		}
	}
	return selected
}

func (a *projectAssembly) importOwnerContext(contextID string) string {
	for context := a.symbols.GetContext(contextID); context != nil; context = a.symbols.GetContext(context.ParentId) {
		if context.ContextType_ == symboltable.S_PackageSymbol || context.Id == a.context.Id {
			return context.Id
		}
	}
	return contextID
}

func importUseKey(tableID, name string, symbols *symboltable.FolangSymbols) string {
	logicalParts := strings.Split(logicalName(name), ".")
	if len(logicalParts) < 2 {
		return ""
	}
	table := symbols.GetSymbolTable(tableID)
	if table == nil {
		return ""
	}
	for context := symbols.GetContext(table.ContextId); context != nil; context = symbols.GetContext(context.ParentId) {
		for importName := range context.ImportedContextIds {
			logicalImport := logicalName(importName)
			width := strings.Count(logicalImport, ".") + 1
			if len(logicalParts) > width && strings.Join(logicalParts[:width], ".") == logicalImport {
				return context.Id + "\x00" + importName
			}
		}
	}
	if root := symbols.GetContext(symbols.FolContextRootContextID()); root != nil {
		for importName := range root.ImportedContextIds {
			logicalImport := logicalName(importName)
			width := strings.Count(logicalImport, ".") + 1
			if len(logicalParts) > width && strings.Join(logicalParts[:width], ".") == logicalImport {
				return root.Id + "\x00" + importName
			}
		}
	}
	return ""
}

func importBindingOwner(context *symboltable.Context, name string, symbols *symboltable.FolangSymbols) *symboltable.Context {
	for current := context; current != nil; current = symbols.GetContext(current.ParentId) {
		if _, ok := current.ImportedContextIds[name]; ok {
			return current
		}
	}
	root := symbols.GetContext(symbols.FolContextRootContextID())
	if root != nil {
		if _, ok := root.ImportedContextIds[name]; ok {
			return root
		}
	}
	return nil
}

// fallbackCompilationInputs keeps malformed projects diagnosable. The strict
// planner may reject their layout before producing an order; parsing the
// remaining files still gives callers the complete set of source findings.
func fallbackCompilationInputs(proj *project.Project) []project.CompilationInput {
	inputs := make([]project.CompilationInput, 0, len(proj.Files))
	for _, file := range proj.Files {
		rel, _ := filepath.Rel(proj.Root, file.Path)
		parts := strings.Split(filepath.ToSlash(rel), "/")
		stage := project.StagePrimarySource
		kind := ""
		if len(parts) >= 2 && parts[0] == project.ComponentDomain {
			stage, kind = project.StageComponents, parts[1]
		}
		inputs = append(inputs, project.CompilationInput{Path: file.Path, Stage: stage, ComponentKind: kind,
			PackagePath: file.PackagePath})
	}
	sort.Slice(inputs, func(i, j int) bool {
		if inputs[i].Stage != inputs[j].Stage {
			return inputs[i].Stage < inputs[j].Stage
		}
		return inputs[i].Path < inputs[j].Path
	})
	return inputs
}

// loadCompiledLibrary reconstructs a lib/*.folenc dependency before primary
// source parsing. The artifact keeps its own context boundaries; importing it
// later links to its published root rather than copying declarations into the
// importing package.
func (a *projectAssembly) loadCompiledLibrary(path string) {
	raw, err := os.ReadFile(path)
	if err != nil {
		a.diagnostics = append(a.diagnostics, projectDiagnostic(fmt.Sprintf("reading compiled library %s: %v", path, err)))
		return
	}
	var artifact CompiledArtifact
	if err := helpers.DeserializeArtifact(raw, &artifact); err != nil {
		a.diagnostics = append(a.diagnostics, projectDiagnostic(fmt.Sprintf("decoding compiled library %s: %v", path, err)))
		return
	}
	if err := validateCompiledDependencyArtifact(&artifact); err != nil {
		a.diagnostics = append(a.diagnostics, projectDiagnostic(fmt.Sprintf("invalid compiled library %s: %v", path, err)))
		return
	}
	if err := mergeDependencySymbolGraph(a.symbols, artifact.FolangSymbols); err != nil {
		a.diagnostics = append(a.diagnostics, projectDiagnostic(fmt.Sprintf("loading compiled library %s: %v", path, err)))
		return
	}
	if artifact.ProjectedAPI != nil {
		a.publishImportContext("library:"+artifact.Name, a.symbols.GetContext(artifact.ProjectedAPI.Id))
	}
	for packagePath, context := range artifact.PackagedSymbols {
		a.publishImportContext("package:"+packagePath, a.symbols.GetContext(context.Id))
	}
	a.libraries[artifact.Name] = ast.ProjectStmt{
		NodeName:      "ProjectStmt",
		FolangSymbols: artifact.FolangSymbols,
		ProjectKind:   ast.ProjectKindLibrary,
		IsLibrary:     true,
	}
}

func mergeDependencySymbolGraph(destination, source *symboltable.FolangSymbols) error {
	if destination == nil || source == nil {
		return fmt.Errorf("dependency has no symbol graph")
	}
	graph := cloneStandardSymbolGraph(source)
	for id := range graph.ContextMap {
		if destination.GetContextInfo(id) != nil {
			return fmt.Errorf("context id %q collides with an existing dependency", id)
		}
	}
	for id := range graph.SymboltableMap {
		if destination.GetSymbolTable(id) != nil {
			return fmt.Errorf("symbol-table id %q collides with an existing dependency", id)
		}
	}
	for _, table := range graph.SymboltableMap {
		destination.AddSymbolTable(table)
	}
	for _, symbol := range graph.SymbolsById {
		destination.RegisterSymbol(symbol)
	}
	for id, context := range graph.ContextMap {
		destination.ContextMap[id] = context
	}
	return nil
}

// validateStandaloneComponents applies the rule that decides WHICH components a
// standalone library may own.
//
// The filesystem cannot answer it. A standalone library is `src/component.fol`,
// and which components it may keep turns on the exposure model written INSIDE
// that file: "The sole structural exception is a projected application library,
// which may contain exactly one optional project-local component kind,
// components/operators/" — while "standalone packaged, native, and dynamicvmrt
// libraries must not contain a components/ tree at all"
// (docs/language-ref.md, "src/ — Primary Project Source").
//
// The CLI's preparation pass already applies this; doing it here as well is what
// makes ParseProject uphold the rule on its own, since a caller using it directly
// gets no preparation pass.
func (a *projectAssembly) validateStandaloneComponents() {
	if len(a.ownedComponentKinds) == 0 {
		return
	}
	surface, isComponent := a.entry.(ast.ComponentDeclarationStmt)
	if !isComponent {
		return // an application entry; components/ is ordinary there
	}

	if surface.Projected && surface.LibraryType == componentKindApplication {
		for _, kind := range sortedComponentKinds(a.ownedComponentKinds) {
			if kind != componentKindOperators {
				a.diagnostics = append(a.diagnostics, projectDiagnostic(fmt.Sprintf(
					"a projected application library permits only %s/%s, not %s/%s",
					componentDomain, componentKindOperators, componentDomain, kind)))
			}
		}
		return
	}
	for _, kind := range sortedComponentKinds(a.ownedComponentKinds) {
		a.diagnostics = append(a.diagnostics, projectDiagnostic(fmt.Sprintf(
			"a standalone %s library may not contain %s/%s; a project-local component is not a library, and reusable dependencies belong in %s/",
			standaloneKind(surface), componentDomain, kind, project.PackagedLibraryDomain)))
	}
}

// sortedComponentKinds orders the assembled component kinds so a project with
// more than one violation reports them the same way on every run.
func sortedComponentKinds(components map[string]bool) []string {
	kinds := make([]string, 0, len(components))
	for kind := range components {
		kinds = append(kinds, kind)
	}
	sort.Strings(kinds)
	return kinds
}

// layoutDiagnostics carries the discovered layout violations through as
// project-level diagnostics.
//
// A finding is ALREADY a diagnostic — Layout.report builds one — so it is passed
// along rather than re-wrapped, which would stamp a second "Invalid Syntax:" onto
// a message that has one. Anything that somehow is not gets wrapped, so a caller
// never loses a violation to a type it did not expect.
func layoutDiagnostics(layout project.Layout) []helpers.ErrorInterface {
	if len(layout.Findings) == 0 {
		return nil
	}
	diagnostics := make([]helpers.ErrorInterface, 0, len(layout.Findings))
	for _, finding := range layout.Findings {
		if diagnostic, ok := finding.(helpers.ErrorInterface); ok {
			diagnostics = append(diagnostics, diagnostic)
			continue
		}
		diagnostics = append(diagnostics, projectDiagnostic(finding.Error()))
	}
	return diagnostics
}

// ProjectContext returns the non-lexical descriptor of an assembled project.
func ProjectContext(symbols *symboltable.FolangSymbols) *symboltable.FolContext {
	if symbols == nil {
		return nil
	}
	return symbols.RootFolContext()
}

// ProjectRootContext resolves the operational root named by FolContext. The
// descriptor-to-root edge is transparent project structure, never ParentId.
func ProjectRootContext(symbols *symboltable.FolangSymbols) *symboltable.Context {
	project := ProjectContext(symbols)
	if project == nil {
		return nil
	}
	return symbols.GetContext(project.Context_)
}

// projectTarget picks the file discovery is anchored on.
//
// Discovery takes a source file and a root rather than a root alone, because its
// other caller compiles ONE file and needs to know which. A whole-project parse
// has no such file, so the structural surface stands in for it: it is the one file
// every project of that kind has. Failing that, any source file under the root
// will do, since the root is given explicitly and it is the root that decides what
// gets enumerated.
func projectTarget(root string) (string, error) {
	for _, surface := range []string{
		filepath.Join(project.SourceDomain, project.ApplicationEntryFilename),
		filepath.Join(project.SourceDomain, componentSurfaceFilename),
	} {
		candidate := filepath.Join(root, surface)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
	}

	found := ""
	walkErr := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || found != "" || info.IsDir() || filepath.Ext(path) != ".fol" {
			return err
		}
		found = path
		return filepath.SkipAll
	})
	if walkErr != nil {
		return "", fmt.Errorf("reading project %s: %w", root, walkErr)
	}
	if found == "" {
		return "", fmt.Errorf("project %s holds no FoLang source", root)
	}
	return found, nil
}

// projectAssembly accumulates parsed files into the project tree.
type projectAssembly struct {
	proj *project.Project
	root string

	// symbols and context are the one scope model every file is parsed into.
	symbols *symboltable.FolangSymbols
	context *symboltable.Context

	// operators is the project's custom operator catalog. It is loaded once,
	// before any file is parsed, because a project-local spelling cannot be
	// tokenized correctly without it.
	operators []operatorDeclaration
	// graph collects every file's import edges, so the whole-project import
	// checks see the edges no single file could show.
	graph *importcheck.Graph

	entry ast.Stmt
	// entryContext owns the independent appl.fol/component.fol surface table.
	// It is selected by FolContext.SymbolTable_ and is not a child of context.
	entryContext *symboltable.Context
	// ownedComponentKinds is every components/<kind>/ the project holds source
	// for, including the operator component that is excluded from assembly.
	ownedComponentKinds map[string]bool
	// externals holds each library slot and component kind while its files are
	// gathered, and externalOrder keeps the order they were first seen so the
	// parse is deterministic.
	externals     map[string]*externalUnit
	externalOrder []string
	packages      *packageTree
	libraries     map[string]ast.Stmt
	compnents     map[string]ast.Stmt
	// importContexts is populated before primary source parsing and addressed by
	// the closed import subjects package:, component:, and library:.
	importContexts map[string]*symboltable.Context
	ownedPackages  map[string]string

	diagnostics []helpers.ErrorInterface
}

// packageAssembly is one folder's package while it is still being filled.
type packageAssembly struct {
	// path is the package's dot path relative to the domain root that owns it.
	path string
	body []ast.Stmt
	// declared indexes body by declaration name, which is what lets a companion
	// unit find the type it belongs to however the folder's files were ordered.
	declared map[string]int
	// pending holds companion members whose type has not been read yet, keyed by
	// the type's name. A folder is walked in path order, so `Employee.comp.unit.fol`
	// can arrive before `Employee.fol`.
	pending         map[string][]ast.Stmt
	symbol          *symboltable.ComponentSymbol
	context         *symboltable.Context
	contexts        map[string]*symboltable.Context
	pendingContexts map[string][]*symboltable.Context
	symbols         *symboltable.FolangSymbols
}

func newProjectAssembly(proj *project.Project, root string) (*projectAssembly, error) {
	symbols := &symboltable.FolangSymbols{}
	symbols.CreateFolangSymbols()

	context, table := CreateNewContext("", symboltable.S_Program, helpers.CanonicalIdentityPath(root))
	symbols.AddContext(context)
	symbols.AddSymbolTable(table)
	symbols.RootContextId = context.Id

	bootstrap := loadProjectOperatorBootstrap(proj.Root)

	assembly := &projectAssembly{
		proj:                proj,
		root:                root,
		symbols:             symbols,
		context:             context,
		operators:           bootstrap.Declarations,
		graph:               importcheck.NewGraph(),
		packages:            newPackageTree(symbols, context),
		externals:           map[string]*externalUnit{},
		libraries:           map[string]ast.Stmt{},
		compnents:           map[string]ast.Stmt{},
		importContexts:      map[string]*symboltable.Context{},
		ownedPackages:       map[string]string{},
		ownedComponentKinds: map[string]bool{},
	}
	standard, _, err := loadInstalledStandardArtifact()
	if err != nil {
		return nil, err
	}
	if standard != nil {
		if err := mergeInstalledStandardSymbols(symbols, context, standard); err != nil {
			return nil, err
		}
	}
	for _, finding := range bootstrap.Findings {
		assembly.diagnostics = append(assembly.diagnostics, projectDiagnostic(finding.Error()))
	}
	return assembly, nil
}

// add takes one file, parsing it now when it belongs to src/ and setting it aside
// when it belongs to an external unit.
//
// An external cannot be parsed file by file in discovery order, because what its
// implementation files contribute depends on its SURFACE: a projected library
// publishes only the surface, so its own files must not be parsed into the
// project's symbol model at all. The surface has to be read before that is known.
func (a *projectAssembly) add(file project.File) {
	// Recorded before the bootstrap is set aside, because the operator component
	// is a component the project OWNS even though it is never assembled: its
	// declarations reach the project as the operator catalog instead. Reading the
	// owned kinds off a.compnents would therefore miss the one kind a standalone
	// library is most likely to have.
	if a.domainOf(file) == componentDomain {
		if kind := a.componentKindOf(file); kind != "" {
			a.ownedComponentKinds[kind] = true
		}
	}
	if a.isOperatorBootstrap(file) {
		return
	}

	// lib/ is absent here on purpose. It holds compiled .folenc artifacts, source
	// discovery excludes it, and a .fol file found there is a layout error rather
	// than something to assemble. A library reaches the project by being read back
	// from its artifact into ProjectStmt.LibraryStmt, not by being parsed.
	switch a.domainOf(file) {
	case componentDomain:
		a.bucket(componentDomain, a.componentKindOf(file)).take(file, a)
	default:
		parent := a.context
		if file.PackagePath != "" {
			parent = a.packages.packageOf(file.PackagePath).context
		}
		result, ok := a.parse(file, projectScope{symbols: a.symbols, parent: parent})
		if ok {
			a.addSourceFile(file, result)
		}
	}
}

// parse reads and parses one file into the given scope model.
func (a *projectAssembly) parse(file project.File, scope projectScope) (Result, bool) {
	source, err := os.ReadFile(file.Path)
	if err != nil {
		a.diagnostics = append(a.diagnostics, projectDiagnostic(fmt.Sprintf("reading %s: %v", file.Base, err)))
		return Result{}, false
	}

	// The DIRECTORY is what tells a component surface which component it is, and
	// what the duplicate-filename check reads, so the file's own folder is passed
	// rather than the project root.
	result := parseCollecting(a.graph, string(source), filepath.Base(a.proj.Root), filepath.Dir(file.Path), file.Base, file.PackagePath, true,
		parseConfiguration{
			locationKnown:  true,
			atRoot:         a.isSurface(file),
			operators:      a.operators,
			scope:          scope,
			importContexts: a.importContexts,
		})
	a.diagnostics = append(a.diagnostics, result.Diagnostics...)
	return result, result.Root != nil
}

// What a library or a component publishes to the project.
//
// An external unit is a whole little project of its own: one surface and the
// packages below it. Its AST always reaches the project in full, so a later phase
// can see what it is made of. Its SYMBOLS do not, and what decides how much of
// them arrives is the surface's exposure model
// (docs/language-ref.md, "Standalone Library Forms"):
//
//   - A PROJECTED library — its surface carries @co.dap.library — publishes the
//     surface and nothing else. Its implementation is reached only through that
//     declared API, so parsing its files into the project's symbol model would
//     make internals visible that the form exists to hide. They are parsed into a
//     model of the unit's own instead, which the unit's node carries.
//   - Anything else publishes its whole symbol table, because there is no surface
//     standing between the project and its contents.
//
// Two component kinds are outside the rule. `operators` is the bootstrap catalog
// and is never parsed as ordinary source. `packaged` exports selected package
// contexts directly into the application's graph, so it publishes its whole table
// whatever its surface says.

// externalUnit is one library slot or component kind, gathered before it is
// parsed.
type externalUnit struct {
	// key is the name an import reaches this unit by: a library slot, a
	// component kind.
	key string
	// domain distinguishes a library from a component, which decides which of the
	// project's two maps the unit lands in.
	domain string
	// surface is the unit's fixed structural file, and is nil for a unit that has
	// none — a malformed tree, or a lib/ artifact folder with no source surface.
	surface *project.File
	files   []project.File
}

// bucket returns the unit for a domain and key, creating it on first sight.
func (a *projectAssembly) bucket(domain, key string) *externalUnit {
	id := domain + "/" + key
	if existing, ok := a.externals[id]; ok {
		return existing
	}

	unit := &externalUnit{key: key, domain: domain}
	a.externals[id] = unit
	a.externalOrder = append(a.externalOrder, id)
	return unit
}

// take sets a file aside for later, remembering the surface when it sees it.
func (u *externalUnit) take(file project.File, a *projectAssembly) {
	if a.isSurface(file) {
		copied := file
		u.surface = &copied
		return
	}
	file.PackagePath = a.packagePathIn(file)
	u.files = append(u.files, file)
}

// packagePathIn is a file's package path relative to the domain root that owns
// it.
//
// Discovery computes that for src/, whose domain it knows. It knows
// nothing of components/, so a file there arrives with a path measured from the
// PROJECT root — `components.packaged.internals` rather than `internals` — which
// would make `components` a package and the component's own folder another.
func (a *projectAssembly) packagePathIn(file project.File) string {
	if a.domainOf(file) != componentDomain {
		return file.PackagePath
	}

	// components / <kind> / <package…> / <file>
	segments := a.segmentsOf(file)
	if len(segments) <= 3 {
		return ""
	}
	return strings.Join(segments[2:len(segments)-1], ".")
}

// parseExternals parses every external unit and files its node.
func (a *projectAssembly) parseExternals() {
	for _, id := range a.externalOrder {
		unit := a.externals[id]
		node := a.parseExternal(unit)
		if node == nil {
			continue
		}
		if unit.domain == componentDomain {
			a.compnents[unit.key] = node
		} else {
			a.libraries[unit.key] = node
		}
	}
}

// parseExternal parses one unit's surface, decides how much of it the project may
// see, and assembles its files into a node of its own.
//
// The SURFACE is parsed into the project's model either way, because it is what a
// projected unit publishes and part of what an unprojected one does. Only its
// implementation files move, and only when the surface says they must.
func (a *projectAssembly) parseExternal(unit *externalUnit) ast.Stmt {
	projectScopeOfUnit := projectScope{symbols: a.symbols, parent: a.context}

	var surfaceRoot ast.Stmt
	var surfaceContext *symboltable.Context
	projected := false
	if unit.surface != nil {
		if result, ok := a.parse(*unit.surface, projectScopeOfUnit); ok {
			surfaceRoot, surfaceContext = result.Root, result.Context
			if unit.domain == componentDomain {
				a.context.ChildCtxIds = removeContextID(a.context.ChildCtxIds, surfaceContext.Id)
				surfaceContext.ParentId = ""
				surfaceContext.ParentCtxSymbolTableId = ""
			}
			projected = a.publishesSurfaceOnly(unit, surfaceRoot)
		}
	}

	// An unprojected unit publishes its whole table, so its files go straight into
	// the project's model and its node shares it. A projected one gets a model of
	// its own, rooted outside the project's tree, so that nothing below its
	// surface is reachable from the project.
	scope, published := projectScopeOfUnit, a.symbols
	if projected {
		symbols := &symboltable.FolangSymbols{}
		symbols.CreateFolangSymbols()
		context, table := CreateNewContext("", symboltable.S_Program, helpers.CanonicalIdentityPath(a.root), unit.key)
		symbols.AddContext(context)
		symbols.AddSymbolTable(table)
		symbols.RootContextId = context.Id

		// The unit's own model still indexes its surface, so the node carries a
		// complete view of the unit. The record is shared, not copied: one
		// declaration seen from two places.
		a.publishInto(symbols, surfaceContext)

		scope, published = projectScope{symbols: symbols, parent: context}, symbols
	}

	tree := newPackageTree(scope.symbols, scope.parent)
	for _, file := range unit.files {
		pkg := tree.packageOf(file.PackagePath)
		result, ok := a.parse(file, projectScope{symbols: scope.symbols, parent: pkg.context})
		if !ok {
			continue
		}
		if file.PackagePath == "" {
			// A domain root is not a package either, so the surface is the only
			// thing that may sit directly in it.
			a.diagnostics = append(a.diagnostics, projectDiagnostic(
				fmt.Sprintf("%s sits directly in %s, which holds only its surface file and its packages", file.Base, unit.key)))
			continue
		}
		pkg.absorb(file, result, a)
	}
	a.validatePackageOverloads(tree)

	packages, pending := tree.build()
	a.reportPending(unit.key, pending)
	if surfaceRoot == nil && len(packages) == 0 {
		return nil
	}

	// A component describes a folder that has a surface and packages below it,
	// which is the node the parser already built for its surface file; assembly
	// only has to hand it the packages.
	if unit.domain == componentDomain {
		component, isComponent := surfaceRoot.(ast.ComponentDeclarationStmt)
		if !isComponent {
			component = ast.ComponentDeclarationStmt{NodeName: "ComponentDeclarationStmt", Name: unit.key, Kind: unit.key,
				SurfaceFile: surfaceRoot,
				Symb:        externalSymbol(unit),
			}
		}
		component.SubPackage = packages
		if unit.key == componentKindPackaged {
			for selected, recursive := range exportedPackageSelectors(surfaceRoot) {
				matched := false
				for path, pkg := range tree.byPath {
					if path == selected || (recursive && strings.HasPrefix(path, selected+".")) {
						a.publishImportContext("package:"+path, pkg.context)
						matched = true
					}
				}
				if !matched {
					a.diagnostics = append(a.diagnostics, projectDiagnostic(
						fmt.Sprintf("packaged component exports unknown package %q", selected)))
				}
			}
		} else if surfaceContext != nil {
			a.publishImportContext("component:"+unit.key, surfaceContext)
		}
		return component
	}

	// A library IS a project: its own entry, its own packages, its own scope
	// model, and — when it publishes a surface — the surface a consumer resolves
	// against instead of that model.
	kind := projectKind(surfaceRoot)
	var descriptor *symboltable.FolContext
	if surfaceContext != nil && scope.parent != nil {
		descriptor = &symboltable.FolContext{
			Id: helpers.StableID("project", unit.domain, unit.key), SymbolTable_: surfaceContext.SymbolTable_,
			Context_: scope.parent.Id, Kind: kind, ChildCtxIds: []string{surfaceContext.Id},
		}
		surfaceContext.ParentId = descriptor.Id
		if published != a.symbols {
			published.AddFolContext(descriptor)
		}
	}
	return ast.ProjectStmt{NodeName: "ProjectStmt",
		EntryStmt:     surfaceRoot,
		PackageStmts:  packages,
		ProjectKind:   kind,
		FolangSymbols: published,
		IsLibrary:     true, // a reconstructed external unit is a library by construction
		Symb:          externalSymbol(unit),
	}
}

// publishImportContext adds one prepared dependency to the closed import
// resolver namespace. Equal bindings are idempotent; two providers publishing
// the same subject are a project-layout error that the developer must resolve.
func (a *projectAssembly) publishImportContext(subject string, context *symboltable.Context) {
	if context == nil {
		return
	}
	if existing := a.importContexts[subject]; existing != nil && existing.Id != context.Id {
		a.diagnostics = append(a.diagnostics, projectDiagnostic(
			fmt.Sprintf("import subject %q is published by both context %q and context %q", subject, existing.Id, context.Id)))
		return
	}
	a.importContexts[subject] = context
}

// validatePackageOverloads applies callable-family rules after ordinary unit
// wrappers have been collapsed into their package. Individual file parses can
// validate siblings written in one file, but two ordinary units initially bind
// into separate temporary table chains. Once merged, their semantic owner is the
// package and declarations across those chains are sibling overloads too.
func (a *projectAssembly) validatePackageOverloads(tree *packageTree) {
	for _, pkg := range tree.byPath {
		validateMergedOverloadFamilies(pkg, a)
	}
}

type mergedFunctionBinding struct {
	key     string
	tableID string
	symbol  *symboltable.FunctionSymbol
}

func validateMergedOverloadFamilies(pkg *packageAssembly, assembly *projectAssembly) {
	if pkg == nil || pkg.context == nil || pkg.symbols == nil {
		return
	}

	families := map[string][]mergedFunctionBinding{}
	visited := map[string]bool{}
	for tableID := pkg.context.SymbolTable_; tableID != "" && !visited[tableID]; {
		visited[tableID] = true
		table := pkg.symbols.GetSymbolTable(tableID)
		if table == nil {
			break
		}
		for key, info := range pkg.symbols.Bindings(table.Id) {
			function, ok := info.(*symboltable.FunctionSymbol)
			if !ok || function.IsOperator {
				continue
			}
			open := strings.IndexByte(key, '(')
			if open < 0 {
				continue
			}
			family := key[:open]
			families[family] = append(families[family], mergedFunctionBinding{key: key, tableID: tableID, symbol: function})
		}
		tableID = table.ParentId
	}

nextFamily:
	for _, siblings := range families {
		for left := 0; left < len(siblings); left++ {
			for right := left + 1; right < len(siblings); right++ {
				first, second := siblings[left], siblings[right]
				if first.tableID == second.tableID {
					continue // the file-local binder already checked this pair
				}
				name := logicalName(first.symbol.Name_)
				switch {
				case first.key == second.key:
					assembly.diagnostics = append(assembly.diagnostics, projectDiagnostic(
						fmt.Sprintf("%s is already declared in this package with the same parameter signature", name)))
				case first.symbol.OverloadRestriction != "":
					assembly.diagnostics = append(assembly.diagnostics, projectDiagnostic(
						fmt.Sprintf("%s cannot be overloaded across ordinary unit files: a declaration has %s", name, first.symbol.OverloadRestriction)))
				case second.symbol.OverloadRestriction != "":
					assembly.diagnostics = append(assembly.diagnostics, projectDiagnostic(
						fmt.Sprintf("%s cannot be overloaded across ordinary unit files: a declaration has %s", name, second.symbol.OverloadRestriction)))
				case first.symbol.ReturnSignature != second.symbol.ReturnSignature:
					assembly.diagnostics = append(assembly.diagnostics, projectDiagnostic(
						fmt.Sprintf("every declaration of %s in the package must declare the same return signature", name)))
				default:
					continue
				}
				continue nextFamily // one project diagnostic is sufficient for this invalid family
			}
		}
	}
}

// publishesSurfaceOnly reports whether only this unit's surface reaches the
// project's symbol model.
//
// Only a LIBRARY can hide behind a surface. A project-local component is not a
// library: it is compiled as part of the project that owns it, its kind comes
// from its folder rather than from an annotation, and the reference forbids
// @co.dap.library below components/ outright. It also has no scope model of its
// own to be isolated INTO — a component node carries a surface and its packages,
// not a FolangSymbols — so isolating one would drop the model it was moved to.
func (a *projectAssembly) publishesSurfaceOnly(unit *externalUnit, surface ast.Stmt) bool {
	if unit.domain == componentDomain {
		return false
	}
	return declaresProjectedLibrary(surface)
}

// publishInto indexes a surface's scope in another model, so that a projected
// unit's own model describes the surface it publishes as well as the
// implementation behind it.
//
// The contexts are shared rather than copied. A context is identified by its id
// and a symbol by the table that holds it, so indexing one record in two models
// makes it visible from both without making it two declarations.
func (a *projectAssembly) publishInto(symbols *symboltable.FolangSymbols, surface *symboltable.Context) {
	if surface == nil {
		return
	}

	var index func(ctx *symboltable.Context)
	index = func(ctx *symboltable.Context) {
		if ctx == nil {
			return
		}
		symbols.ContextMap[ctx.Id] = ctx
		for segment := ctx.SymbolTable_; segment != ""; {
			table := a.symbols.GetSymbolTable(segment)
			if table == nil {
				break
			}
			symbols.AddSymbolTable(table)
			for _, id := range table.SymbolIds {
				if info := a.symbols.GetSymbol(id); info != nil {
					symbols.RegisterSymbol(info)
				}
			}
			segment = table.ParentId
		}
		for _, child := range ctx.ChildCtxIds {
			index(a.symbols.GetContext(child))
		}
	}
	index(surface)
}

// declaresProjectedLibrary reports whether a surface carries @co.dap.library,
// which is the annotation that makes a standalone component a projected library
// rather than a packaged one.
func declaresProjectedLibrary(surface ast.Stmt) bool {
	if component, isComponent := surface.(ast.ComponentDeclarationStmt); isComponent && component.Projected {
		return true
	}
	return carriesAnnotation(surface, projectedLibraryAnnotation)
}

// projectedLibraryAnnotation is the annotation whose presence selects the
// projected exposure model.
const projectedLibraryAnnotation = "@co.dap.library"

// carriesAnnotation reports whether a node's attached annotation run holds one of
// the given name.
func carriesAnnotation(node ast.Stmt, name string) bool {
	list, isList := annotationsOf(node).(*ast.DirectveList)
	if !isList || list == nil {
		return false
	}
	for _, run := range list.Dapst {
		for _, item := range run {
			if directive, isDirective := item.(ast.DirectiveStmt); isDirective && directive.Name == name {
				return true
			}
		}
	}
	return false
}

// annotationsOf reaches the annotation run a surface node carries. Each surface
// form keeps it in its own field, and a node with none has nothing to report.
func annotationsOf(node ast.Stmt) ast.Stmt {
	switch surface := node.(type) {
	case ast.ComponentDeclarationStmt:
		return surface.SDapst
	case ast.TypeDeclarationStmt:
		return surface.SDapst
	case ast.RefinementTypeDeclarationStmt:
		return surface.SDapst
	case ast.PredicateTypeDeclarationStmt:
		return surface.SDapst
	case ast.DependentTypeDeclarationStmt:
		return surface.SDapst
	case ast.PackageStmt:
		return surface.SDapst
	}
	return nil
}

// externalSymbol names an external unit after the slot or kind an import reaches
// it by, since neither its folder nor its fixed surface filename carries a name.
func externalSymbol(unit *externalUnit) *symboltable.ComponentSymbol {
	symbol := &symboltable.ComponentSymbol{
		SymbolDetails: symboltable.SymbolDetails{
			SymbolId_:   helpers.StableID("symbol", "external", unit.key, unit.domain),
			SymbolType_: string(symboltable.S_ComponentSymbol),
			Name_:       unit.key,
			Type_:       unit.key,
		},
	}
	symbol.Name = unit.key
	symbol.Kind = unit.domain
	return symbol
}

// isSurface reports whether a file is the fixed structural surface of the domain
// that holds it, rather than a package source file within it.
//
// Discovery answers that for the source and source-library domains; a component
// surface is `components/<kind>/component.fol` and is recognised by its name,
// since discovery does not classify that domain.
func (a *projectAssembly) isSurface(file project.File) bool {
	if a.domainOf(file) == componentDomain {
		return file.Base == componentSurfaceFilename
	}
	return file.AtRoot
}

// isOperatorBootstrap reports whether a file is the project's operator catalog
// rather than one of its program surfaces.
//
// The catalog is read by the operator-source grammar before any file is parsed,
// which is what makes a project-local spelling tokenize at all. Parsing it AGAIN
// as ordinary source would report every `co.lang.operator` declaration in it as
// misplaced, since that form is legal only in the surface being excluded here.
// Its declarations already reach the project as this assembly's operator catalog.
func (a *projectAssembly) isOperatorBootstrap(file project.File) bool {
	return a.domainOf(file) == componentDomain && a.componentKindOf(file) == componentKindOperators
}

// addSourceFile places a file from the src/ domain.
func (a *projectAssembly) addSourceFile(file project.File, result Result) {
	if file.PackagePath != "" {
		a.packages.packageOf(file.PackagePath).absorb(file, result, a)
		return
	}

	// src/ is a domain, not a package, so the only thing that may sit directly in
	// it is the one structural surface. Anything else has nowhere to go: it is in
	// no package, and treating it as a second surface would silently pick one of
	// the two.
	if !isStructuralSurface(file.Base) {
		a.diagnostics = append(a.diagnostics, projectDiagnostic(
			fmt.Sprintf("%s sits directly in %s/, which is a domain rather than a package; move it into a subfolder", file.Base, project.SourceDomain)))
		return
	}

	a.entry = result.Root
	a.entryContext = result.Context
	// The surface and operational root are independent structural roots. Imports
	// written in appl.fol/component.fol belong to the operational root; the
	// surface retains only the automatic co binding used by all contexts.
	for alias, contextID := range result.Context.ImportedContextIds {
		if existing := a.context.ImportedContextIds[alias]; existing != "" && existing != contextID {
			a.diagnostics = append(a.diagnostics, projectDiagnostic(
				fmt.Sprintf("surface import name %q conflicts with the project root binding", alias)))
			continue
		}
		a.context.ImportedContextIds[alias] = contextID
		if alias != "co" {
			delete(result.Context.ImportedContextIds, alias)
		}
	}
	a.context.ChildCtxIds = removeContextID(a.context.ChildCtxIds, result.Context.Id)
	result.Context.ParentId = ""
	result.Context.ParentCtxSymbolTableId = ""
}

// isStructuralSurface reports whether a filename is one of the fixed surfaces a
// domain root may hold.
func isStructuralSurface(base string) bool {
	switch base {
	case project.ApplicationEntryFilename, componentSurfaceFilename:
		return true
	}
	return false
}

// packageTree is the set of packages below one domain root, keyed by dot path.
//
// A project has one for src/, and each external unit has one of its own, because
// a library's packages are its own namespace rather than the project's.
type packageTree struct {
	byPath  map[string]*packageAssembly
	symbols *symboltable.FolangSymbols
	root    *symboltable.Context
}

func newPackageTree(symbols *symboltable.FolangSymbols, root *symboltable.Context) *packageTree {
	return &packageTree{byPath: map[string]*packageAssembly{}, symbols: symbols, root: root}
}

// packageOf returns the package for a dot path, creating it and every ancestor it
// needs. An ancestor is created even when no file sits directly in it, because a
// folder that only holds folders is still a package in the tree.
func (t *packageTree) packageOf(path string) *packageAssembly {
	if existing, ok := t.byPath[path]; ok {
		return existing
	}

	var parent *symboltable.Context
	if parentPath := parentPackagePath(path); parentPath != "" {
		parent = t.packageOf(parentPath).context
	}
	parentID := ""
	if parent != nil {
		parentID = parent.Id
	}
	ctx, table := CreateNewContext(parentID, symboltable.S_PackageSymbol, path)
	if parent != nil {
		ctx.ParentCtxSymbolTableId = parent.SymbolTable_
		parent.ChildCtxIds = append(parent.ChildCtxIds, ctx.Id)
	} else if t.root != nil {
		if co := t.root.ImportedContextIds["co"]; co != "" {
			ctx.ImportedContextIds["co"] = co
		}
	}
	t.symbols.AddContext(ctx)
	t.symbols.AddSymbolTable(table)

	created := &packageAssembly{
		path:            path,
		declared:        map[string]int{},
		pending:         map[string][]ast.Stmt{},
		context:         ctx,
		contexts:        map[string]*symboltable.Context{},
		pendingContexts: map[string][]*symboltable.Context{},
		symbols:         t.symbols,
	}
	t.byPath[path] = created
	return created
}

// build links the packages into a tree and returns the top-level ones by name,
// together with the companion members no declaration claimed.
func (t *packageTree) build() (map[string]ast.Stmt, map[string][]string) {
	built := map[string]ast.PackageStmt{}
	for path, pkg := range t.byPath {
		built[path] = ast.PackageStmt{NodeName: "PackageStmt",
			Name:       path,
			Body:       pkg.body,
			SubPackage: map[string]ast.Stmt{},
			Symb:       pkg.symbol,
		}
	}

	// Deepest first, so a package is complete before its parent takes it.
	for _, path := range pathsByDepth(built) {
		parent := parentPackagePath(path)
		if parent == "" {
			continue
		}
		owner, ok := built[parent]
		if !ok {
			continue
		}
		owner.SubPackage[lastSegment(path)] = built[path]
		built[parent] = owner
	}

	top := map[string]ast.Stmt{}
	for _, path := range sortedKeys(built) {
		if parentPackagePath(path) == "" {
			top[path] = built[path]
		}
	}

	pending := map[string][]string{}
	for _, path := range sortedPackagePaths(t.byPath) {
		if names := sortedStmtKeys(t.byPath[path].pending); len(names) > 0 {
			pending[path] = names
		}
	}
	return top, pending
}

// absorb takes one file's declarations into this package.
func (p *packageAssembly) absorb(file project.File, result Result, a *projectAssembly) {
	packageRoot, isPackage := result.Root.(ast.PackageStmt)
	if !isPackage {
		// A file whose root is not a package source file still belongs to the
		// folder; keep it whole rather than discarding what was parsed.
		p.body = append(p.body, result.Root)
		return
	}
	if p.symbol == nil {
		p.symbol = packageRoot.Symb
	}

	class := classifySourceFilename(file.Base)
	fileContext := result.Context
	var declarationContext *symboltable.Context
	if fileContext != nil && len(fileContext.ChildCtxIds) == 1 {
		declarationContext = a.symbols.GetContext(fileContext.ChildCtxIds[0])
	}

	for _, item := range packageRoot.Body {
		p.absorbItem(class, item, declarationContext, a)
	}

	switch class.Class {
	case sourceClassOrdinaryUnit:
		a.mergeContext(fileContext, p.context)
		a.mergeContext(declarationContext, p.context)
	case sourceClassCompanionUnit:
		// fold transfers the unit body into the owning type. The remaining file
		// root is a parse-time wrapper and is collapsed into the package.
		a.mergeContext(fileContext, p.context)
	default:
		a.mergeContext(fileContext, p.context)
	}
}

// mergeContext collapses a temporary parse context into its semantic owner while
// preserving every table ID carried by AST nodes. Its oldest segment is chained
// to the owner's current segment, all segments change ownership, and children are
// reparented without changing their exact branch-point table IDs.
func (a *projectAssembly) mergeContext(source, target *symboltable.Context) {
	if err := mergeContext(a.symbols, source, target); err != nil {
		a.diagnostics = append(a.diagnostics, projectDiagnostic(err.Error()))
	}
}

func mergeContext(symbols *symboltable.FolangSymbols, source, target *symboltable.Context) error {
	if source == nil || target == nil || source.Id == target.Id {
		return nil
	}
	for alias, contextID := range source.ImportedContextIds {
		if existing := target.ImportedContextIds[alias]; existing != "" && existing != contextID {
			return fmt.Errorf("import name %q resolves to both context %q and context %q in the same owning context", alias, existing, contextID)
		}
	}
	if target.ImportedContextIds == nil {
		target.ImportedContextIds = map[string]string{}
	}
	for alias, contextID := range source.ImportedContextIds {
		target.ImportedContextIds[alias] = contextID
	}

	targetOriginalTableID := target.SymbolTable_
	oldest := symbols.GetSymbolTable(source.SymbolTable_)
	for oldest != nil && oldest.ParentId != "" {
		next := symbols.GetSymbolTable(oldest.ParentId)
		if next == nil {
			break
		}
		oldest = next
	}
	if oldest != nil {
		oldest.ParentId = target.SymbolTable_
	}
	for id := source.SymbolTable_; id != ""; {
		table := symbols.GetSymbolTable(id)
		if table == nil {
			break
		}
		table.ContextId = target.Id
		id = table.ParentId
		if id == target.SymbolTable_ {
			break
		}
	}
	target.SymbolTable_ = source.SymbolTable_

	for _, childID := range source.ChildCtxIds {
		child := symbols.GetContext(childID)
		if child != nil {
			child.ParentId = target.Id
			target.ChildCtxIds = append(target.ChildCtxIds, childID)
		}
	}
	if parent := symbols.GetContext(source.ParentId); parent != nil {
		parent.ChildCtxIds = removeContextID(parent.ChildCtxIds, source.Id)
	}
	delete(symbols.ContextMap, source.Id)

	// If the target's original segment was only the empty anchor used to attach
	// this temporary parse context, it no longer represents a visibility segment
	// after the merge. Collapse it unless another context still records that
	// exact table as its branch point.
	anchor := symbols.GetSymbolTable(targetOriginalTableID)
	if anchor == nil || len(anchor.SymbolIds) != 0 || len(anchor.SymbolsByName) != 0 || target.SymbolTable_ == targetOriginalTableID {
		return nil
	}
	for _, info := range symbols.ContextMap {
		if context, ok := info.(*symboltable.Context); ok && context.ParentCtxSymbolTableId == targetOriginalTableID {
			return nil
		}
	}
	for _, table := range symbols.SymboltableMap {
		if table != nil && table.ParentId == targetOriginalTableID {
			table.ParentId = anchor.ParentId
		}
	}
	delete(symbols.SymboltableMap, targetOriginalTableID)
	return nil
}

func removeContextID(ids []string, remove string) []string {
	kept := ids[:0]
	for _, id := range ids {
		if id != remove {
			kept = append(kept, id)
		}
	}
	return kept
}

// absorbItem places one of a file's top-level statements.
func (p *packageAssembly) absorbItem(class sourceFilename, item ast.Stmt, declarationContext *symboltable.Context, a *projectAssembly) {
	declaration, isDeclaration := item.(ast.TypeDeclarationStmt)
	if !isDeclaration {
		// Preamble directives are not declarations and belong to no type, so they
		// stay where the file put them.
		p.body = append(p.body, item)
		return
	}

	switch class.Class {
	case sourceClassCompanionUnit:
		p.fold(class.DerivedName, declaration.Body, declarationContext, a)
	case sourceClassOrdinaryUnit:
		// The unit wrapper has no scope of its own; its members are the package's.
		for _, member := range declaration.Body {
			p.declare(member, a)
		}
	default:
		if declarationContext != nil {
			p.contexts[logicalName(declaration.Name)] = declarationContext
		}
		p.declare(declaration, a)
	}
}

// declare appends a declaration and indexes it by name, taking any companion
// members that arrived before it.
func (p *packageAssembly) declare(item ast.Stmt, a *projectAssembly) {
	p.body = append(p.body, item)

	declaration, isDeclaration := item.(ast.TypeDeclarationStmt)
	if !isDeclaration {
		return
	}

	name := logicalName(declaration.Name)
	p.declared[name] = len(p.body) - 1
	if waiting, ok := p.pending[name]; ok {
		delete(p.pending, name)
		contexts := p.pendingContexts[name]
		delete(p.pendingContexts, name)
		declaration.Body = append(declaration.Body, waiting...)
		p.body[len(p.body)-1] = declaration
		for _, ctx := range contexts {
			a.mergeContext(ctx, p.contexts[name])
		}
	}
}

// fold adds a companion unit's members to the body of the type they belong to.
//
// The type may not have been read yet, since a folder is walked in path order and
// `Employee.comp.unit.fol` sorts before `Employee.fol`; such members are held
// until it arrives, and reported at the end if it never does.
func (p *packageAssembly) fold(typeName string, members []ast.Stmt, companionContext *symboltable.Context, a *projectAssembly) {
	at, known := p.declared[typeName]
	if !known {
		p.pending[typeName] = append(p.pending[typeName], members...)
		p.pendingContexts[typeName] = append(p.pendingContexts[typeName], companionContext)
		return
	}

	declaration, isDeclaration := p.body[at].(ast.TypeDeclarationStmt)
	if !isDeclaration {
		return
	}
	declaration.Body = append(declaration.Body, members...)
	p.body[at] = declaration
	if a != nil && companionContext != nil {
		a.mergeContext(companionContext, p.contexts[typeName])
	}
}

// finish links the project's own packages into a tree and returns the project.
func (a *projectAssembly) finish() ast.Stmt {
	packages, pending := a.packages.build()
	a.reportPending(project.SourceDomain, pending)
	kind := projectKind(a.entry)
	surfaceTableID := ""
	if a.entryContext != nil {
		surfaceTableID = a.entryContext.SymbolTable_
	}
	projectDescriptor := &symboltable.FolContext{
		Id:           helpers.StableID("project", helpers.CanonicalIdentityPath(a.root)),
		SymbolTable_: surfaceTableID,
		Context_:     a.context.Id,
		Kind:         kind,
	}
	if a.entryContext != nil {
		a.entryContext.ParentId = projectDescriptor.Id
		a.entryContext.ParentCtxSymbolTableId = ""
		projectDescriptor.ChildCtxIds = []string{a.entryContext.Id}
	}
	a.symbols.AddFolContext(projectDescriptor)

	return ast.ProjectStmt{NodeName: "ProjectStmt",
		EntryStmt:     a.entry,
		PackageStmts:  packages,
		LibraryStmt:   a.libraries,
		ComponentStmt: a.compnents,
		ProjectKind:   kind,
		FolangSymbols: a.symbols,
		IsLibrary:     isStandaloneLibrary(kind),
		Symb:          projectSymbol(a.proj.Root),
	}
}

// reportPending reports a companion unit whose type no file in its folder
// declares. Its members would otherwise be dropped silently.
func (a *projectAssembly) reportPending(domain string, pending map[string][]string) {
	for _, path := range sortedNameKeys(pending) {
		for _, name := range pending[path] {
			a.diagnostics = append(a.diagnostics, projectDiagnostic(
				fmt.Sprintf("companion unit %s.comp.unit.fol has no %s.fol in package %s of %s to attach to", name, name, path, domain)))
		}
	}
}

// sortedNameKeys orders a map of package path to names.
func sortedNameKeys(items map[string][]string) []string {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// projectDiagnostic reports something the project's SHAPE is wrong about rather
// than its source text: an unreadable file, a companion with nothing to attach to.
// Such a finding has no token to point at, so it carries the zero position.
func projectDiagnostic(details string) helpers.ErrorInterface {
	return helpers.NewInvalidSyntaxError(helpers.Position{}, helpers.Position{}, details)
}

// domainOf reports which project-root domain a file belongs to. componentDomain
// is matched by position, because discovery classifies only the source and
// source-library domains and gives a component surface no Domain of its own.
func (a *projectAssembly) domainOf(file project.File) string {
	if file.Domain != "" {
		return file.Domain
	}
	if segments := a.segmentsOf(file); len(segments) > 0 {
		return segments[0]
	}
	return project.SourceDomain
}

// componentKindOf names the components/ child that owns a file, which is the key
// an import uses to reach it.
func (a *projectAssembly) componentKindOf(file project.File) string {
	if segments := a.segmentsOf(file); len(segments) > 1 {
		return segments[1]
	}
	return ""
}

// segmentsOf splits a file's path relative to the project root.
func (a *projectAssembly) segmentsOf(file project.File) []string {
	relative, err := filepath.Rel(a.proj.Root, file.Path)
	if err != nil {
		return nil
	}
	return strings.Split(filepath.ToSlash(relative), "/")
}

// isProjectedLibrary reports whether a surface declares itself a standalone
// library rather than an application.
// projectKind names the project from the structural surface src/ holds.
//
// The surface is the only thing that decides it. src/ holds exactly one primary
// surface, and src/component.fol then picks between the two mutually exclusive
// standalone exposure models from its own metadata: @co.dap.library makes it a
// projected library, and its absence — with the @co.dap.export selector in the
// body — makes it a packaged one (docs/language-ref.md, "Form Exclusivity").
//
// A surface this does not recognize returns "" rather than guessing. A project
// whose entry failed to parse has no kind to report, and naming it "application"
// would put a claim in the artifact that nothing established.
func projectKind(entry ast.Stmt) string {
	switch surface := entry.(type) {
	case ast.Application:
		return ast.ProjectKindApplication
	case ast.ComponentDeclarationStmt:
		if surface.Projected {
			return ast.ProjectKindLibrary
		}
		return ast.ProjectKindPackagedLibrary
	default:
		return ""
	}
}

// isStandaloneLibrary reports whether a project kind is one of the two standalone
// library forms.
//
// Both are libraries. "The source then selects one of two mutually exclusive
// STANDALONE exposure models" — projected, carrying @co.dap.library, and packaged,
// exposing selected package contexts through @co.dap.export
// (docs/language-ref.md, "Form Exclusivity"). Appendix B.7.1 defines IsLibrary as
// "true when this ProjectStatement represents a standalone library", so a packaged
// library answers true exactly as a projected one does.
func isStandaloneLibrary(kind string) bool {
	return kind == ast.ProjectKindLibrary || kind == ast.ProjectKindPackagedLibrary
}

func isProjectedLibrary(root ast.Stmt) bool {
	component, isComponent := root.(ast.ComponentDeclarationStmt)
	return isComponent && component.Projected
}

// projectSymbol mints the symbol naming the project itself, after the root
// directory. A project root is not a package and contributes no namespace
// component, so the name is for diagnostics and tooling rather than for lookup.
func projectSymbol(root string) *symboltable.ComponentSymbol {
	name := filepath.Base(root)
	symbol := &symboltable.ComponentSymbol{
		SymbolDetails: symboltable.SymbolDetails{
			SymbolId_:   helpers.StableID("symbol", "project", helpers.CanonicalIdentityPath(root)),
			SymbolType_: string(symboltable.S_ComponentSymbol),
			Name_:       name,
			Type_:       name,
		},
	}
	symbol.Name = name
	symbol.Kind = "project"
	return symbol
}

// parentPackagePath drops a package's last segment, and is empty for a top-level
// package.
func parentPackagePath(path string) string {
	if at := strings.LastIndex(path, "."); at >= 0 {
		return path[:at]
	}
	return ""
}

// lastSegment is a package's own name, without its parents.
func lastSegment(path string) string {
	if at := strings.LastIndex(path, "."); at >= 0 {
		return path[at+1:]
	}
	return path
}

// pathsByDepth orders package paths deepest first.
func pathsByDepth(packages map[string]ast.PackageStmt) []string {
	paths := sortedKeys(packages)
	sort.SliceStable(paths, func(i, j int) bool {
		return strings.Count(paths[i], ".") > strings.Count(paths[j], ".")
	})
	return paths
}

func sortedKeys(packages map[string]ast.PackageStmt) []string {
	keys := make([]string, 0, len(packages))
	for key := range packages {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedPackagePaths(packages map[string]*packageAssembly) []string {
	keys := make([]string, 0, len(packages))
	for key := range packages {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedStmtKeys(items map[string][]ast.Stmt) []string {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
