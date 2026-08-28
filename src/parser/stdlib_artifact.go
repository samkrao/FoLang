package parser

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	symboltable "github.com/samkrao/fo-lang/src/context"
	"github.com/samkrao/fo-lang/src/helpers"
)

const standardArtifactRelativePath = "stdlib/co.folenc"

var (
	standardExecutablePath = os.Executable
	standardEvalSymlinks   = filepath.EvalSymlinks
	standardArtifactDecode = helpers.DeserializeArtifact
)

// installedStandardArtifactPath derives <install-root>/stdlib/co.folenc from
// the real running compiler executable, never from cwd or argv[0].
func installedStandardArtifactPath() (string, error) {
	executable, err := standardExecutablePath()
	if err != nil {
		return "", fmt.Errorf("locating the running compiler executable: %w", err)
	}
	realExecutable, err := standardEvalSymlinks(executable)
	if err != nil {
		return "", fmt.Errorf("resolving compiler executable %s: %w", executable, err)
	}
	installRoot := filepath.Dir(filepath.Dir(realExecutable))
	return filepath.Join(installRoot, filepath.FromSlash(standardArtifactRelativePath)), nil
}

// loadInstalledStandardArtifact loads the standard package before project
// parsing. A missing file is temporarily tolerated for compiler-bootstrap and
// repository development while co.folenc is not yet distributed. Once the file
// exists, every read/codec/validation failure is a compiler-installation error.
func loadInstalledStandardArtifact() (*CompiledArtifact, string, error) {
	path, err := installedStandardArtifactPath()
	if err != nil {
		return nil, "", err
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, path, nil
	}
	if err != nil {
		return nil, path, fmt.Errorf("reading installed standard package %s: %w", path, err)
	}
	var artifact CompiledArtifact
	if err := standardArtifactDecode(raw, &artifact); err != nil {
		return nil, path, fmt.Errorf("decoding installed standard package %s: %w", path, err)
	}
	if err := validateInstalledStandardArtifact(&artifact); err != nil {
		return nil, path, fmt.Errorf("invalid installed standard package %s: %w", path, err)
	}
	return &artifact, path, nil
}

func validateInstalledStandardArtifact(artifact *CompiledArtifact) error {
	if artifact == nil || artifact.FolangSymbols == nil {
		return errors.New("artifact does not contain FolangSymbols")
	}
	if artifact.SymbolFormatVersion != symboltable.SymbolFormatVersion {
		return fmt.Errorf("symbol format version %d is unsupported; want %d", artifact.SymbolFormatVersion, symboltable.SymbolFormatVersion)
	}
	graph := artifact.FolangSymbols
	if graph.ContextMap == nil || graph.SymboltableMap == nil {
		return errors.New("artifact has an incomplete context/symbol-table graph")
	}
	root := graph.GetContext(artifact.RootContextID)
	if root == nil {
		return fmt.Errorf("root context %q is absent", artifact.RootContextID)
	}
	if root.Prefix != "co" && !strings.HasPrefix(root.Prefix, "co.") {
		return fmt.Errorf("root context %q has prefix %q, want reserved co package identity", root.Id, root.Prefix)
	}
	if graph.GetSymbolTable(root.SymbolTable_) == nil {
		return fmt.Errorf("co root context %q names absent symbol table %q", root.Id, root.SymbolTable_)
	}
	return nil
}

// mergeInstalledStandardSymbols adds the canonical artifact graph to the
// project's FolangSymbols and imports its co root implicitly at project scope.
// Artifact IDs must be stable artifact-owned IDs; collision is rejected rather
// than silently rewriting cross-structure references.
func mergeInstalledStandardSymbols(destination *symboltable.FolangSymbols, projectRoot *symboltable.Context, artifact *CompiledArtifact) error {
	if destination == nil || projectRoot == nil {
		return errors.New("cannot merge standard symbols without a destination graph and project root context")
	}
	if err := validateInstalledStandardArtifact(artifact); err != nil {
		return err
	}
	graph := cloneStandardSymbolGraph(artifact.FolangSymbols)
	coRoot := graph.GetContext(artifact.RootContextID)
	if projectRoot.ImportedContextIds != nil {
		if existing := projectRoot.ImportedContextIds["co"]; existing != "" && existing != coRoot.Id {
			return fmt.Errorf("project root already imports reserved co identity from context %q", existing)
		}
	}
	for id := range graph.ContextMap {
		if destination.GetContext(id) != nil {
			return fmt.Errorf("standard context id %q collides with the project symbol graph", id)
		}
	}
	for id := range graph.SymboltableMap {
		if destination.GetSymbolTable(id) != nil {
			return fmt.Errorf("standard symbol-table id %q collides with the project symbol graph", id)
		}
	}
	for _, table := range graph.SymboltableMap {
		destination.AddSymbolTable(table)
	}
	for _, info := range graph.SymbolsById {
		destination.RegisterSymbol(info)
	}
	for id, context := range graph.ContextMap {
		destination.ContextMap[id] = context
	}

	coRoot.ParentId = projectRoot.Id
	coRoot.ParentCtxSymbolTableId = projectRoot.SymbolTable_
	projectRoot.ChildCtxIds = append(projectRoot.ChildCtxIds, coRoot.Id)
	if projectRoot.ImportedContextIds == nil {
		projectRoot.ImportedContextIds = map[string]string{}
	}
	projectRoot.ImportedContextIds["co"] = coRoot.Id
	return nil
}

func cloneStandardSymbolGraph(source *symboltable.FolangSymbols) *symboltable.FolangSymbols {
	clone := &symboltable.FolangSymbols{}
	clone.CreateFolangSymbols()
	clone.RootContextId = source.RootContextId
	clone.SurfaceSymbols = source.SurfaceSymbols
	for _, symbol := range source.SymbolsById {
		clone.RegisterSymbol(symbol)
	}
	for _, sourceTable := range source.SymboltableMap {
		table := *sourceTable
		table.SymbolIds = append([]string(nil), sourceTable.SymbolIds...)
		table.SymbolsByName = make(map[string][]string, len(sourceTable.SymbolsByName))
		for name, ids := range sourceTable.SymbolsByName {
			table.SymbolsByName[name] = append([]string(nil), ids...)
		}
		clone.AddSymbolTable(&table)
	}
	for id, sourceContext := range source.ContextMap {
		context := *sourceContext
		context.RestrictedSymbolNameReuse = append([]string(nil), sourceContext.RestrictedSymbolNameReuse...)
		context.ChildCtxIds = append([]string(nil), sourceContext.ChildCtxIds...)
		context.ImportedContextIds = make(map[string]string, len(sourceContext.ImportedContextIds))
		for alias, contextID := range sourceContext.ImportedContextIds {
			context.ImportedContextIds[alias] = contextID
		}
		clone.ContextMap[id] = &context
	}
	return clone
}
