package main

import (
	"fmt"
	"sort"
	"strings"

	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/callgraph/cha"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

// callEdges is one function's place in the call graph.
type callEdges struct {
	Callers []string `json:"callers"`
	Callees []string `json:"callees"`
}

// buildCallGraph computes the parse-function call graph with CHA.
//
// Class Hierarchy Analysis resolves an interface or method call to every
// concrete method with a matching signature, so it never misses an edge that a
// run could take. It is the right trade for documentation: a reader wants the
// complete set of possible callees, and the parser dispatches through function
// values (parseBracedBody takes a member parser), which a purely syntactic scan
// of call expressions would drop entirely.
//
// Only edges between parse functions are kept. Helper and scanner calls would
// otherwise dominate the output without describing grammar structure.
func buildCallGraph(pkgDir string) (map[string]*callEdges, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
			packages.NeedImports | packages.NeedDeps | packages.NeedTypes |
			packages.NeedSyntax | packages.NeedTypesInfo | packages.NeedModule,
		Dir: ".",
	}
	pkgs, err := packages.Load(cfg, "./"+filepathToSlash(pkgDir))
	if err != nil {
		return nil, err
	}
	if packages.PrintErrors(pkgs) > 0 {
		return nil, fmt.Errorf("package load reported errors")
	}
	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no packages matched %q", pkgDir)
	}

	prog, _ := ssautil.AllPackages(pkgs, ssa.InstantiateGenerics)
	prog.Build()

	graph := cha.CallGraph(prog)
	graph.DeleteSyntheticNodes()

	// Deduplicate with sets: CHA yields one edge per call site, so a function
	// called from three places would otherwise appear three times.
	callers := map[string]map[string]bool{}
	callees := map[string]map[string]bool{}

	if err := callgraph.GraphVisitEdges(graph, func(edge *callgraph.Edge) error {
		// CHA resolves a call through a func value to every function with a
		// matching signature. recoverItem(startPos, sync, body func()) is called
		// from most item loops, so the unfiltered graph links every `func()`
		// closure in the package to every other, which is sound but says nothing
		// about the grammar. Keeping the statically resolved sites leaves the
		// edges a reader can act on: the direct calls, including those made
		// inside a closure, which parseFunctionName charges to its parent.
		if edge.Site == nil || edge.Site.Common().StaticCallee() == nil {
			return nil
		}
		from, fromOK := parseFunctionName(edge.Caller.Func)
		to, toOK := parseFunctionName(edge.Callee.Func)
		if !fromOK || !toOK || from == to {
			return nil
		}
		addEdge(callees, from, to)
		addEdge(callers, to, from)
		return nil
	}); err != nil {
		return nil, err
	}

	names := map[string]bool{}
	for name := range callers {
		names[name] = true
	}
	for name := range callees {
		names[name] = true
	}

	out := make(map[string]*callEdges, len(names))
	for name := range names {
		out[name] = &callEdges{
			Callers: sortedKeys(callers[name]),
			Callees: sortedKeys(callees[name]),
		}
	}
	return out, nil
}

// parseFunctionName returns the short name of fn when it is a parse function on
// *parser, and reports whether it is one.
//
// A closure is attributed to the named function that encloses it. The parser
// passes member parsers as literals — parseBracedBody("a unit body", func() … ) —
// so without walking to the parent, every call made inside such a literal would
// be charged to an anonymous function and dropped, leaving declaration parsers
// looking as though they call nothing.
func parseFunctionName(fn *ssa.Function) (string, bool) {
	for fn != nil && fn.Parent() != nil {
		fn = fn.Parent()
	}
	if fn == nil || fn.Pkg == nil || fn.Signature.Recv() == nil {
		return "", false
	}
	if !strings.HasSuffix(fn.Pkg.Pkg.Path(), "/src/parser") {
		return "", false
	}
	recv := fn.Signature.Recv().Type().String()
	if !strings.HasSuffix(recv, ".parser") || !strings.HasPrefix(recv, "*") {
		return "", false
	}
	if !strings.HasPrefix(fn.Name(), "parse") {
		return "", false
	}
	return fn.Name(), true
}

func addEdge(m map[string]map[string]bool, from, to string) {
	if m[from] == nil {
		m[from] = map[string]bool{}
	}
	m[from][to] = true
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func filepathToSlash(p string) string {
	return strings.ReplaceAll(p, "\\", "/")
}
