package parser

import (
	"errors"
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
	Name            string
	ProjectedAPI    *symboltable.Context
	PackagedSymbols map[string]*symboltable.Context
	PackagedAST     map[string]ast.SET
	Operators       []string
	// FolangSymbols is the artifact's complete serialized context/symbol-table
	// graph. It is mandatory for the installed standard-package artifact and may
	// also be retained by ordinary libraries for semantic reconstruction.
	FolangSymbols *symboltable.FolangSymbols
	// RootContextID identifies the exported package root inside FolangSymbols.
	// For the installed standard artifact this context has the reserved prefix co.
	RootContextID string
}

// PreparedLibrary records either a decoded artifact or a pending artifact whose
// codec has deliberately not been implemented yet.
type PreparedLibrary struct {
	Path     string
	Artifact CompiledArtifact
	Pending  bool
}

type PublishedArtifactPackage struct {
	Symbols *symboltable.Context
	AST     ast.SET
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
	for path, library := range p.Libraries {
		if library.Pending {
			continue
		}
		name := library.Artifact.Name
		if name == "" {
			name = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		}
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
	err = helpers.DeserializeArtifact(raw, &artifact)
	if errors.Is(err, helpers.ErrArtifactCodecNotImplemented) {
		p.Libraries[input.Path] = PreparedLibrary{Path: input.Path, Pending: true}
		return
	}
	if err != nil {
		p.Findings = append(p.Findings, fmt.Errorf("decoding compiled library %s: %w", input.Path, err))
		return
	}
	p.Libraries[input.Path] = PreparedLibrary{Path: input.Path, Artifact: artifact}
}

func (p *PreparedProject) preparePrimaryInput(input project.CompilationInput) {
	if source, ok := p.parsePreparedSource(input); ok {
		p.Primary = append(p.Primary, source)
	}
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
