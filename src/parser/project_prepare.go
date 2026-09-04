package parser

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/samkrao/fo-lang/src/ast"
	symboltable "github.com/samkrao/fo-lang/src/context"
	"github.com/samkrao/fo-lang/src/helpers"
	"github.com/samkrao/fo-lang/src/project"
)

// PreparedSource keeps one source tree with its own root symbol environment.
// Environments are deliberately not merged: publication happens through the
// component/library boundary records below.
type PreparedSource struct {
	Path            string
	PackagePath     string
	AST             ast.Stmt
	Symbols         *symboltable.Context
	RootSymbolTable *symboltable.SymbolTable
	SymbolGraph     *symboltable.FolangSymbols
}

// PreparedComponent is one isolated project-owned compilation domain.
type PreparedComponent struct {
	Kind            string
	Surface         *PreparedSource
	PrivatePackages map[string][]PreparedSource
	PackagedExports map[string][]PreparedSource
	ProjectedAPI    *symboltable.Context
}

// CompiledArtifact is the logical .folenc payload. Its wire encoding is kept
// behind helpers.DeserializeArtifact and may be implemented later.
type CompiledArtifact struct {
	SymbolFormatVersion int `json:"symbolFormatVersion"`
	Name                string
	ProjectedAPI        *symboltable.Context
	PackagedSymbols     map[string]*symboltable.Context
	// PackagedAST is already projected language-neutral AST JSON. Keeping raw
	// documents here avoids attempting to decode JSON into Go interface values.
	PackagedAST map[string]json.RawMessage
	Operators   []string
	// FolangSymbols is the artifact's complete serialized context/symbol-table
	// graph. It is mandatory for the installed standard-package artifact and may
	// also be retained by ordinary libraries for semantic reconstruction.
	FolangSymbols *symboltable.FolangSymbols
	// RootContextID identifies the exported package root inside FolangSymbols.
	// For the installed standard artifact this context has the reserved prefix co.
	RootContextID string
}

// PreparedLibrary records one successfully decoded and validated artifact.
type PreparedLibrary struct {
	Path     string
	Artifact CompiledArtifact
}

type PublishedArtifactPackage struct {
	Symbols *symboltable.Context
	AST     json.RawMessage
}

// PublishedEnvironment is the only cross-domain state primary src receives.
// Component-private packages are intentionally absent.
type PublishedEnvironment struct {
	ProjectedComponents map[string]*symboltable.Context
	ProjectedLibraries  map[string]*symboltable.Context
	PackagedComponents  map[string][]PreparedSource
	PackagedLibraries   map[string]PublishedArtifactPackage
	Operators           []operatorDeclaration
}

// PreparedProject is the ordered, isolated frontend input environment.
type PreparedProject struct {
	Root                      string
	Kind                      project.CompilationProjectKind
	Components                map[string]*PreparedComponent
	Libraries                 map[string]PreparedLibrary
	Primary                   []PreparedSource
	Operators                 []operatorDeclaration
	Order                     []project.CompilationStage
	Findings                  []error
	Environment               PublishedEnvironment
	StandaloneProjectedAPI    *symboltable.Context
	StandalonePackagedExports map[string][]PreparedSource
	StandardArtifact          *CompiledArtifact
}

// PrepareProjectRoot prepares a discovered project in the normative order:
// components, compiled libraries, then primary src.
func PrepareProjectRoot(target, rootOverride string) (*PreparedProject, error) {
	standardArtifact, _, err := loadInstalledStandardArtifact()
	if err != nil {
		return nil, err
	}
	discovered, err := project.Discover(target, rootOverride)
	if err != nil {
		return nil, err
	}
	inputs, err := project.CompilationInputs(discovered.Root, discovered.Files)
	if err != nil {
		return nil, err
	}

	bootstrap := loadProjectOperatorBootstrap(discovered.Root)
	kind, layoutFindings := project.ValidateCompilationRoot(discovered.Root)
	prepared := &PreparedProject{
		Root: discovered.Root, Kind: kind, Components: map[string]*PreparedComponent{},
		Libraries:                 map[string]PreparedLibrary{},
		Operators:                 append([]operatorDeclaration(nil), bootstrap.Declarations...),
		Findings:                  append([]error(nil), layoutFindings...),
		StandalonePackagedExports: map[string][]PreparedSource{},
		StandardArtifact:          standardArtifact,
	}
	prepared.Findings = append(prepared.Findings, bootstrap.Findings...)

	var last project.CompilationStage
	exportsFinalized := false
	for _, input := range inputs {
		if input.Stage != last {
			if input.Stage > project.StageComponents && !exportsFinalized {
				prepared.finalizeComponentExports()
				exportsFinalized = true
			}
			if input.Stage == project.StagePrimarySource {
				prepared.buildPublishedEnvironment()
			}
			prepared.Order = append(prepared.Order, input.Stage)
			last = input.Stage
		}
		switch input.Stage {
		case project.StageComponents:
			prepared.prepareComponentInput(input)
		case project.StageLibraries:
			prepared.prepareLibraryInput(input)
		case project.StagePrimarySource:
			prepared.preparePrimaryInput(input)
		}
	}
	if !exportsFinalized {
		prepared.finalizeComponentExports()
	}
	prepared.buildPublishedEnvironment()
	prepared.finalizeStandaloneSurface()
	return prepared, nil
}

func (p *PreparedProject) finalizeStandaloneSurface() {
	if p.Kind != project.CompilationStandaloneComponent {
		return
	}
	var surface *PreparedSource
	packages := map[string][]PreparedSource{}
	for index := range p.Primary {
		source := &p.Primary[index]
		if filepath.Base(source.Path) == project.ComponentSurfaceFilename && source.PackagePath == "" {
			surface = source
			continue
		}
		if source.PackagePath != "" {
			packages[source.PackagePath] = append(packages[source.PackagePath], *source)
		}
	}
	if surface == nil {
		return
	}
	decl, ok := surface.AST.(ast.ComponentDeclarationStmt)
	if !ok {
		return
	}
	if decl.Projected {
		p.StandaloneProjectedAPI = surface.Symbols
	} else {
		p.selectPackagedSources(exportSelectors(decl), packages, p.StandalonePackagedExports, "standalone component")
	}

	if len(p.Components) == 0 {
		return
	}
	if decl.Projected && decl.LibraryType == componentKindApplication {
		for kind := range p.Components {
			if kind != componentKindOperators {
				p.Findings = append(p.Findings, fmt.Errorf("a projected application library permits only components/operators, not components/%s", kind))
			}
		}
		return
	}
	for kind := range p.Components {
		p.Findings = append(p.Findings, fmt.Errorf("a standalone %s component project may not contain components/%s", standaloneKind(decl), kind))
	}
}

func standaloneKind(decl ast.ComponentDeclarationStmt) string {
	if !decl.Projected {
		return "packaged"
	}
	if decl.LibraryType == "" {
		return componentKindApplication
	}
	return decl.LibraryType
}

func (p *PreparedProject) buildPublishedEnvironment() {
	environment := PublishedEnvironment{
		ProjectedComponents: map[string]*symboltable.Context{},
		ProjectedLibraries:  map[string]*symboltable.Context{},
		PackagedComponents:  map[string][]PreparedSource{},
		PackagedLibraries:   map[string]PublishedArtifactPackage{},
		Operators:           append([]operatorDeclaration(nil), p.Operators...),
	}
	for kind, component := range p.Components {
		if component.ProjectedAPI != nil {
			environment.ProjectedComponents[kind] = component.ProjectedAPI
		}
		for packagePath, sources := range component.PackagedExports {
			environment.PackagedComponents[packagePath] = append([]PreparedSource(nil), sources...)
		}
	}
	for _, library := range p.Libraries {
		name := library.Artifact.Name
		if library.Artifact.ProjectedAPI != nil {
			environment.ProjectedLibraries[name] = library.Artifact.ProjectedAPI
		}
		for packagePath, symbols := range library.Artifact.PackagedSymbols {
			environment.PackagedLibraries[packagePath] = PublishedArtifactPackage{
				Symbols: symbols, AST: library.Artifact.PackagedAST[packagePath],
			}
		}
	}
	p.Environment = environment
}

func (p *PreparedProject) finalizeComponentExports() {
	component := p.Components[componentKindPackaged]
	if component == nil || component.Surface == nil {
		return
	}
	decl, ok := component.Surface.AST.(ast.ComponentDeclarationStmt)
	if !ok {
		return
	}
	selectors := exportSelectors(decl)
	p.selectPackagedSources(selectors, component.PrivatePackages, component.PackagedExports, "packaged component")
}

func exportSelectors(decl ast.ComponentDeclarationStmt) map[string]bool {
	selectors := map[string]bool{}
	for _, member := range ast.ComponentSurfaceBody(decl) {
		directive, ok := member.(ast.DirectiveStmt)
		if !ok || directive.Name != componentExportSelectorName {
			continue
		}
		packages, _ := directive.Parameters["packages"].(map[string]any)
		for packagePath, options := range packages {
			recurse := false
			if optionMap, isMap := options.(map[string]any); isMap {
				switch value := optionMap["recurse"].(type) {
				case bool:
					recurse = value
				case string:
					recurse = value == "true" || value == "co.const.true"
				}
			}
			selectors[packagePath] = recurse
		}
	}
	return selectors
}

func (p *PreparedProject) selectPackagedSources(selectors map[string]bool, available, published map[string][]PreparedSource, owner string) {
	for selected, recurse := range selectors {
		matched := false
		for packagePath, sources := range available {
			if packagePath == selected || (recurse && strings.HasPrefix(packagePath, selected+".")) {
				published[packagePath] = append([]PreparedSource(nil), sources...)
				matched = true
			}
		}
		if !matched {
			p.Findings = append(p.Findings, fmt.Errorf("%s exports unknown package %q", owner, selected))
		}
	}
}

func (p *PreparedProject) prepareComponentInput(input project.CompilationInput) {
	component := p.Components[input.ComponentKind]
	if component == nil {
		component = &PreparedComponent{Kind: input.ComponentKind,
			PrivatePackages: map[string][]PreparedSource{}, PackagedExports: map[string][]PreparedSource{}}
		p.Components[input.ComponentKind] = component
	}
	source, ok := p.parsePreparedSource(input)
	if !ok {
		return
	}
	if input.Surface {
		component.Surface = &source
		if input.ComponentKind != componentKindOperators {
			component.ProjectedAPI = source.Symbols
		}
		return
	}
	component.PrivatePackages[input.PackagePath] = append(component.PrivatePackages[input.PackagePath], source)
}

func (p *PreparedProject) prepareLibraryInput(input project.CompilationInput) {
	raw, err := os.ReadFile(input.Path)
	if err != nil {
		p.Findings = append(p.Findings, fmt.Errorf("reading compiled library %s: %w", input.Path, err))
		return
	}
	var artifact CompiledArtifact
	if err := helpers.DeserializeArtifact(raw, &artifact); err != nil {
		p.Findings = append(p.Findings, fmt.Errorf("decoding compiled library %s: %w", input.Path, err))
		return
	}
	if err := validateCompiledDependencyArtifact(&artifact); err != nil {
		p.Findings = append(p.Findings, fmt.Errorf("invalid compiled library %s: %w", input.Path, err))
		return
	}
	p.Libraries[input.Path] = PreparedLibrary{Path: input.Path, Artifact: artifact}
}

func validateCompiledDependencyArtifact(artifact *CompiledArtifact) error {
	if artifact == nil {
		return fmt.Errorf("artifact is nil")
	}
	if artifact.SymbolFormatVersion != symboltable.SymbolFormatVersion {
		return fmt.Errorf("symbol format version %d is unsupported; want %d", artifact.SymbolFormatVersion, symboltable.SymbolFormatVersion)
	}
	if strings.TrimSpace(artifact.Name) == "" {
		return fmt.Errorf("artifact has no logical library name")
	}
	if strings.EqualFold(artifact.Name, "co") {
		return fmt.Errorf("artifact claims the reserved standard-package identity %q", artifact.Name)
	}
	if artifact.FolangSymbols == nil {
		return fmt.Errorf("artifact does not contain FolangSymbols")
	}
	graph := artifact.FolangSymbols
	if graph.ContextMap == nil || graph.SymboltableMap == nil || graph.SymbolsById == nil {
		return fmt.Errorf("artifact has an incomplete context/symbol-table graph")
	}
	if artifact.RootContextID == "" || graph.GetContext(artifact.RootContextID) == nil {
		return fmt.Errorf("root context %q is absent", artifact.RootContextID)
	}
	if graph.RootContextId != "" && graph.RootContextId != artifact.RootContextID {
		return fmt.Errorf("artifact root context %q disagrees with graph root %q", artifact.RootContextID, graph.RootContextId)
	}
	projected := artifact.ProjectedAPI != nil
	packaged := len(artifact.PackagedSymbols) != 0
	if projected == packaged {
		return fmt.Errorf("artifact must expose exactly one projected API or packaged context set")
	}
	if projected && graph.GetContext(artifact.ProjectedAPI.Id) == nil {
		return fmt.Errorf("projected API context %q is absent from FolangSymbols", artifact.ProjectedAPI.Id)
	}
	for packagePath, context := range artifact.PackagedSymbols {
		if strings.TrimSpace(packagePath) == "" || context == nil || graph.GetContext(context.Id) == nil {
			return fmt.Errorf("packaged context %q is absent from FolangSymbols", packagePath)
		}
	}
	return nil
}

func (p *PreparedProject) preparePrimaryInput(input project.CompilationInput) {
	if strings.HasSuffix(filepath.Base(input.Path), ".comp.unit.fol") {
		ownerName := strings.TrimSuffix(filepath.Base(input.Path), ".comp.unit.fol") + ".fol"
		ownerPath := filepath.Clean(filepath.Join(filepath.Dir(input.Path), ownerName))
		if !p.hasParsedStructOwner(ownerPath) {
			p.Findings = append(p.Findings, fmt.Errorf("companion unit %s requires %s to parse successfully as a co.lang.struct declaration", input.Path, ownerPath))
			return
		}
	}
	if source, ok := p.parsePreparedSource(input); ok {
		p.Primary = append(p.Primary, source)
	}
}

func (p *PreparedProject) hasParsedStructOwner(ownerPath string) bool {
	for _, source := range p.Primary {
		if filepath.Clean(source.Path) != ownerPath {
			continue
		}
		pkg, ok := source.AST.(ast.PackageStmt)
		if !ok {
			return false
		}
		for _, statement := range pkg.Body {
			if declaration, ok := statement.(ast.TypeDeclarationStmt); ok && declaration.Kind == "co.lang.struct" {
				return true
			}
		}
		return false
	}
	return false
}

func (p *PreparedProject) parsePreparedSource(input project.CompilationInput) (PreparedSource, bool) {
	raw, err := os.ReadFile(input.Path)
	if err != nil {
		p.Findings = append(p.Findings, fmt.Errorf("reading source %s: %w", input.Path, err))
		return PreparedSource{}, false
	}
	base := filepath.Base(input.Path)
	configuration := parseConfiguration{locationKnown: true, atRoot: input.Surface, operators: p.Operators}
	if input.Stage == project.StagePrimarySource {
		configuration.environment = &p.Environment
	}
	result := parseCollecting(nil, string(raw), filepath.Base(p.Root), filepath.Dir(input.Path), base, input.PackagePath, true, configuration)
	for _, diagnostic := range result.Diagnostics {
		p.Findings = append(p.Findings, diagnostic)
	}
	if p.StandardArtifact != nil {
		if err := mergeInstalledStandardSymbols(result.Symbols, result.Context, p.StandardArtifact); err != nil {
			p.Findings = append(p.Findings, fmt.Errorf("adding installed co packages to %s: %w", input.Path, err))
			return PreparedSource{}, false
		}
	}
	if input.Stage == project.StageComponents || (input.Stage == project.StagePrimarySource && p.Kind == project.CompilationStandaloneComponent) {
		for _, imported := range componentImports(result.Root) {
			p.Findings = append(p.Findings, fmt.Errorf("%s imports project-local component %q; component= is available only to executable application primary src", input.Path, imported.Component))
		}
	}
	return PreparedSource{Path: input.Path, PackagePath: input.PackagePath, AST: result.Root, Symbols: result.Context,
		RootSymbolTable: result.RootSymbolTable, SymbolGraph: result.Symbols}, true
}

func componentImports(root ast.Stmt) []ast.ImportStmt {
	var statements []ast.Stmt
	switch node := root.(type) {
	case ast.Application:
		statements = node.Body
	case ast.PackageStmt:
		statements = node.Body
	case ast.ComponentDeclarationStmt:
		statements = ast.ComponentSurfaceBody(node)
	}
	var imports []ast.ImportStmt
	for _, statement := range statements {
		if imported, ok := statement.(ast.ImportStmt); ok && imported.Component != "" {
			imports = append(imports, imported)
		}
	}
	return imports
}
