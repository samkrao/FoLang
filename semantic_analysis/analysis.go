// Package semantic_analysis implements the post-parse FoLang frontend passes.
//
// The parser owns syntax recognition and builds the AST/context graph. This
// package deliberately consumes that graph without changing it: semantic facts
// are kept in Result so serialized parser artifacts remain stable.
package semantic_analysis

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/samkrao/fo-lang/src/ast"
	symboltable "github.com/samkrao/fo-lang/src/context"
)

// Phase identifies the semantic stage which produced a diagnostic.
type Phase string

const (
	PhaseResolution Phase = "resolution"
	PhaseTyping     Phase = "typing"
	PhaseFlow       Phase = "flow"
)

// Severity is intentionally independent from parser diagnostics. The entry
// driver can adapt both diagnostic forms at their shared boundary later.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

type Diagnostic struct {
	Phase    Phase
	Severity Severity
	Code     string
	Message  string
	Span     ast.Span
}

// Type is the semantic type of an expression occurrence. Name is canonical
// when resolution succeeded; Unknown keeps later passes running after an error.
type Type struct {
	Name    string
	Unknown bool
}

var UnknownType = Type{Name: "co.unknown", Unknown: true}

// Occurrence is keyed by source position rather than an AST pointer because
// parser nodes currently use value semantics.
type Occurrence struct {
	Node                   string
	StartLine, StartColumn int
	EndLine, EndColumn     int
}

func occurrence(node ast.SET) Occurrence {
	span := node.(ast.Spanned).GetSpan()
	typeOfNode := reflect.TypeOf(node)
	if typeOfNode.Kind() == reflect.Pointer {
		typeOfNode = typeOfNode.Elem()
	}
	name := typeOfNode.Name()
	if symbol, ok := node.(ast.SymbolExpr); ok {
		name += ":" + symbol.Value
	}
	return Occurrence{Node: name, StartLine: span.Start.Ln, StartColumn: span.Start.Col,
		EndLine: span.End.Ln, EndColumn: span.End.Col}
}

type Resolution struct {
	State    ast.ResolutionState
	SymbolID string
}

// Result is the typed semantic side table consumed by lowering/backend work.
type Result struct {
	Root        ast.ProjectStmt
	Symbols     *symboltable.FolangSymbols
	Resolutions map[Occurrence]Resolution
	Types       map[Occurrence]Type
	Diagnostics []Diagnostic
}

// Artifact is the portable semantic portion of the frontend/backend envelope.
// Slices are used instead of maps keyed by Occurrence because JSON object keys
// must be strings and deterministic protobuf output benefits from explicit
// source ordering.
type Artifact struct {
	Resolutions []ResolutionFact `json:"resolutions,omitempty"`
	Types       []TypeFact       `json:"types,omitempty"`
}

type ResolutionFact struct {
	Occurrence
	State    ast.ResolutionState `json:"state"`
	SymbolID string              `json:"symbolId,omitempty"`
}

type TypeFact struct {
	Occurrence
	Type Type `json:"type"`
}

func (r *Result) Artifact() Artifact {
	artifact := Artifact{}
	for at, resolution := range r.Resolutions {
		artifact.Resolutions = append(artifact.Resolutions, ResolutionFact{Occurrence: at, State: resolution.State, SymbolID: resolution.SymbolID})
	}
	for at, typ := range r.Types {
		artifact.Types = append(artifact.Types, TypeFact{Occurrence: at, Type: typ})
	}
	sort.Slice(artifact.Resolutions, func(i, j int) bool {
		return occurrenceLess(artifact.Resolutions[i].Occurrence, artifact.Resolutions[j].Occurrence)
	})
	sort.Slice(artifact.Types, func(i, j int) bool { return occurrenceLess(artifact.Types[i].Occurrence, artifact.Types[j].Occurrence) })
	return artifact
}

func occurrenceLess(left, right Occurrence) bool {
	if left.StartLine != right.StartLine {
		return left.StartLine < right.StartLine
	}
	if left.StartColumn != right.StartColumn {
		return left.StartColumn < right.StartColumn
	}
	if left.EndLine != right.EndLine {
		return left.EndLine < right.EndLine
	}
	if left.EndColumn != right.EndColumn {
		return left.EndColumn < right.EndColumn
	}
	return left.Node < right.Node
}

// Error summarizes fatal semantic diagnostics for non-interactive compiler
// entry points. Callers needing individual source spans retain Result directly.
func (r *Result) Error() error {
	if !r.HasErrors() {
		return nil
	}
	messages := make([]string, 0, len(r.Diagnostics))
	for _, diagnostic := range r.Diagnostics {
		if diagnostic.Severity == SeverityError {
			messages = append(messages, fmt.Sprintf("%s %s: %s", diagnostic.Code, diagnostic.Phase, diagnostic.Message))
		}
	}
	return fmt.Errorf("semantic analysis failed:\n%s", strings.Join(messages, "\n"))
}

func (r *Result) HasErrors() bool {
	for _, diagnostic := range r.Diagnostics {
		if diagnostic.Severity == SeverityError {
			return true
		}
	}
	return false
}

// Analyze runs the stages in dependency order. A failed occurrence becomes an
// unknown type; it does not abort the walk and create cascades by omission.
func Analyze(root ast.ProjectStmt) *Result {
	r := &Result{Root: root, Symbols: root.FolangSymbols,
		Resolutions: map[Occurrence]Resolution{}, Types: map[Occurrence]Type{}}
	validateGraph(r)
	resolveNames(r)
	inferTypes(r)
	checkFlow(r)
	return r
}

func (r *Result) report(phase Phase, code, message string, node ast.SET) {
	span := ast.Span{}
	if spanned, ok := node.(ast.Spanned); ok {
		span = spanned.GetSpan()
	}
	r.Diagnostics = append(r.Diagnostics, Diagnostic{Phase: phase, Severity: SeverityError,
		Code: code, Message: message, Span: span})
}

func validateGraph(r *Result) {
	if r.Symbols == nil {
		r.Diagnostics = append(r.Diagnostics, Diagnostic{Phase: PhaseResolution, Severity: SeverityError,
			Code: "SEM0001", Message: "project has no FolangSymbols graph"})
		return
	}
	if r.Symbols.GetFolContext(r.Symbols.RootContextId) == nil {
		r.Diagnostics = append(r.Diagnostics, Diagnostic{Phase: PhaseResolution, Severity: SeverityError,
			Code: "SEM0002", Message: "FolangSymbols.RootContextId does not identify a FolContext"})
	}
	for id, table := range r.Symbols.SymboltableMap {
		if table == nil {
			continue
		}
		if r.Symbols.GetContextInfo(table.ContextId) == nil {
			r.Diagnostics = append(r.Diagnostics, Diagnostic{Phase: PhaseResolution, Severity: SeverityError,
				Code: "SEM0003", Message: fmt.Sprintf("symbol table %q refers to missing context %q", id, table.ContextId)})
		}
		for _, symbolID := range table.SymbolIds {
			if r.Symbols.GetSymbol(symbolID) == nil {
				r.Diagnostics = append(r.Diagnostics, Diagnostic{Phase: PhaseResolution, Severity: SeverityError,
					Code: "SEM0004", Message: fmt.Sprintf("symbol table %q refers to missing symbol %q", id, symbolID)})
			}
		}
	}
}

func resolveNames(r *Result) {
	if r.Symbols == nil {
		return
	}
	Walk(r.Root, func(node ast.SET) bool {
		reference, ok := node.(ast.SymbolExpr)
		if !ok || reference.Value == "" || reference.Value == "_" {
			return true
		}
		// These spellings have already been classified by grammar/context and do
		// not participate in ordinary lexical name lookup.
		if reference.SymbolType_ != "" && reference.SymbolType_ != "reference" {
			return true
		}
		key := occurrence(reference)
		anchor := ""
		if reference.Symb != nil {
			anchor = reference.Symb.SymbolTableId
		}
		matches := lookup(r.Symbols, anchor, reference.Value)
		switch len(matches) {
		case 0:
			r.Resolutions[key] = Resolution{State: ast.ResolutionError}
			r.report(PhaseResolution, "SEM1001", fmt.Sprintf("unresolved name %q", reference.Value), reference)
		case 1:
			r.Resolutions[key] = Resolution{State: ast.ResolutionResolved, SymbolID: matches[0].GetSymbolID()}
		default:
			// Overload families are intentionally retained for the typing pass.
			r.Resolutions[key] = Resolution{State: ast.ResolutionPartiallyResolved}
		}
		return true
	})
}

func lookup(fs *symboltable.FolangSymbols, anchor, name string) []symboltable.SymbolInfo {
	if anchor == "" {
		return nil
	}

	// A qualified source spelling is still allowed to name a lexical symbol.
	// Imports are considered only after the complete lexical walk fails.
	if found := lexicalMatches(fs, anchor, name); len(found) > 0 {
		return found
	}

	parts := strings.Split(semanticLogicalName(name), ".")
	if len(parts) < 2 {
		return nil
	}
	start := fs.GetSymbolTable(anchor)
	if start == nil {
		return nil
	}
	visitedOperationalRoot := false
	for ctx := fs.GetContext(start.ContextId); ctx != nil; ctx = fs.GetContext(ctx.ParentId) {
		if ctx.Id == fs.FolContextRootContextID() {
			visitedOperationalRoot = true
		}
		if found := importedMatches(fs, ctx.ImportedContextIds, parts); len(found) > 0 {
			return found
		}
	}

	// FolContext.Context_ is a transparent project link, not lexical ancestry.
	// Root imports (including the automatic co projection) remain visible from
	// package/member/function contexts that therefore cannot reach it by ParentId.
	if !visitedOperationalRoot {
		if root := fs.GetContext(fs.FolContextRootContextID()); root != nil {
			return importedMatches(fs, root.ImportedContextIds, parts)
		}
	}
	return nil
}

func lexicalMatches(fs *symboltable.FolangSymbols, anchor, name string) []symboltable.SymbolInfo {
	visited := map[string]bool{}
	for table := fs.GetSymbolTable(anchor); table != nil && !visited[table.Id]; {
		visited[table.Id] = true
		if found := tableMatches(fs, table, name); len(found) > 0 {
			return found
		}
		if table.ParentId != "" {
			table = fs.GetSymbolTable(table.ParentId)
			continue
		}
		ctx := fs.GetContext(table.ContextId)
		if ctx == nil || ctx.ParentId == "" {
			break
		}
		if ctx.ParentCtxSymbolTableId != "" {
			table = fs.GetSymbolTable(ctx.ParentCtxSymbolTableId)
			continue
		}
		parent := fs.GetContext(ctx.ParentId)
		if parent == nil {
			break
		}
		table = fs.GetSymbolTable(parent.SymbolTable_)
	}
	return nil
}

func importedMatches(fs *symboltable.FolangSymbols, imports map[string]string, parts []string) []symboltable.SymbolInfo {
	target, width := longestSemanticImport(fs, imports, parts)
	if target == nil {
		return nil
	}
	member := strings.Join(parts[width:], ".")
	boundary := fs.FolContextRootContextID()
	for ctx := target; ctx != nil && ctx.Id != boundary; ctx = fs.GetContext(ctx.ParentId) {
		for table := fs.GetSymbolTable(ctx.SymbolTable_); table != nil; table = fs.GetSymbolTable(table.ParentId) {
			if found := tableMatches(fs, table, member); len(found) > 0 {
				return found
			}
			if table.ParentId == "" {
				break
			}
		}
	}
	return nil
}

func longestSemanticImport(fs *symboltable.FolangSymbols, imports map[string]string, parts []string) (*symboltable.Context, int) {
	var selected *symboltable.Context
	selectedWidth := 0
	for alias, contextID := range imports {
		logicalAlias := semanticLogicalName(alias)
		width := strings.Count(logicalAlias, ".") + 1
		if width <= selectedWidth || len(parts) <= width || strings.Join(parts[:width], ".") != logicalAlias {
			continue
		}
		if target := fs.GetContext(contextID); target != nil {
			selected, selectedWidth = target, width
		}
	}
	return selected, selectedWidth
}

func tableMatches(fs *symboltable.FolangSymbols, table *symboltable.SymbolTable, name string) []symboltable.SymbolInfo {
	if table == nil {
		return nil
	}
	wanted := semanticLogicalName(name)
	var result []symboltable.SymbolInfo
	for _, id := range table.SymbolIds {
		if symbol := fs.GetSymbol(id); symbol != nil && semanticLogicalName(symbol.GetName()) == wanted {
			result = append(result, symbol)
		}
	}
	return result
}

func semanticLogicalName(scanned string) string {
	logical := strings.ReplaceAll(scanned, "_fo.", ".")
	return strings.TrimSuffix(logical, "_fo")
}
