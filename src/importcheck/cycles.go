package importcheck

import (
	"fmt"
	"sort"
	"strings"

	"github.com/samkrao/fo-lang/src/helpers"
)

// Cycle detection.
//
// docs/language-ref.md, "Cycles" requires a compiler error when a cycle exists through
// package imports:
//
//	packageA imports packageB, and packageB imports packageA
//
// Package imports are the only relation. The graph carried a second one until revision 25,
// which withdrew the import fields that fed it (DECISION-PKG-005); nothing else referenced
// them, so the walker below is written against one relation but not hard-coded to it.
//
// A cycle is a whole-program property: no single file can rule one out, because the edge that
// closes the loop is declared somewhere else. Graph therefore accumulates edges as files are
// parsed and Validate walks the accumulated result.
//
// The one case a single file CAN decide is a self-import, where a package imports itself. That
// is a one-node cycle and ValidateSelfImports reports it without a graph, so the common typo
// is caught immediately rather than waiting for a whole-project pass.

// edge is a directed dependency between two nodes, remembered with the position that declared
// it so a cycle can be reported where the programmer can act on it.
type edge struct {
	from  string
	to    string
	start helpers.Position
	end   helpers.Position
	file  string
}

// Graph accumulates the import relationships of every parsed file.
//
// It is not safe for concurrent use; a driver parsing files in parallel must add to it under
// its own lock, or build one graph per worker and merge them.
type Graph struct {
	// packageEdges maps an importing package or source-library surface to the
	// package/surface identities it imports.
	packageEdges map[string][]edge
}

// NewGraph creates an empty relationship graph.
func NewGraph() *Graph {
	return &Graph{
		packageEdges: map[string][]edge{},
	}
}

// Add records one file's relationships.
func (g *Graph) Add(f File) {
	if g == nil {
		return
	}

	source := packageCycleSource(f)
	for _, imp := range f.Imports {
		target := packageCycleTarget(imp)

		// A self-edge is a one-node cycle that ValidateSelfImports already reports with a
		// clearer message, so it is left out of the graph rather than reported twice.
		if target != "" && source != "" && target != source {
			g.packageEdges[source] = append(g.packageEdges[source], edge{
				from:  source,
				to:    target,
				start: imp.Start,
				end:   imp.End,
				file:  f.Name,
			})
		}
	}
}

// Validate reports every cycle in the accumulated graph.
//
// Each distinct cycle is reported once. A driver calls this after every file has been parsed
// and added.
func (g *Graph) Validate() []error {
	if g == nil {
		return nil
	}
	return detectCycles(g.packageEdges, relationPackageImport)
}

// relation describes a cyclic relationship for diagnostic purposes: how to title the
// finding, how to name the relationship, how internal node identities are displayed, and what
// remedy to suggest. It travels with the relation rather than being hard-coded so that a
// second relation, should one be specified, needs no change to the cycle walker.
type relation struct {
	diagnosticName helpers.DiagnosticName
	heading        string
	noun           string
	remedy         string
	nodeLabel      func(string) string
}

var relationPackageImport = relation{
	diagnosticName: helpers.DiagnosticDependencyCycle,
	heading:        "Package Import Cycle",
	noun:           "package import",
	remedy:         "break the loop by moving the shared declarations into a third package that both sides can import",
	nodeLabel:      packageCycleNodeLabel,
}

// ValidateSelfImports reports a package that imports itself.
//
// This is the degenerate one-node cycle, and it is the only cycle a single file can detect on
// its own, so it is reported during parsing rather than deferred to the whole-project pass.
func ValidateSelfImports(f File) []error {
	source := packageCycleSource(f)
	if source == "" {
		return nil
	}

	var findings []error
	for _, imp := range f.Imports {
		if packageCycleTarget(imp) == source {
			findings = append(findings, finding(imp, helpers.DiagnosticDependencyCycle, "Package Import Cycle", fmt.Sprintf(
				"package %q imports itself; a package's own declarations are already visible to it",
				packageCycleNodeLabel(source))))
		}
	}
	return findings
}

// sourceLibraryNodePrefix gives a source-library surface a graph identity distinct from
// any ordinary package. A source library is named by its fixed `srclib/<slot>/` directory
// rather than by a package path, and its implementation packages never enter the owning
// project's ordinary index, so the two namespaces cannot be allowed to collide.
const sourceLibraryNodePrefix = "\ue000source-library:"

// packageCycleSource returns the graph node whose imports a file declares.
//
// An ordinary source file contributes edges from its folder-derived package identity. A
// source-library surface contributes them from its SLOT, which is the only name a
// consumer can import it by.
func packageCycleSource(f File) string {
	if f.IsLibrarySurface && f.LibraryPath != "" && f.LibraryPath != WholeProject {
		return sourceLibraryNodePrefix + f.LibraryPath
	}
	return f.PackagePath
}

// packageCycleTarget returns the graph node an import selects.
//
// Only `package=` contributes a package-graph edge. `library=` names the projected
// surface of a prebuilt lib/<name>.folenc artifact and `component=` a same-owner
// projected component; neither is a package context, and their dependency
// direction is checked by direction.go against the library index instead.
func packageCycleTarget(imp Import) string {
	return imp.Package
}

// packageCycleNodeLabel removes graph-only resolution metadata from a diagnostic label.
func packageCycleNodeLabel(node string) string {
	return strings.TrimPrefix(node, sourceLibraryNodePrefix)
}

// nodeColor marks a node's state during the depth-first search.
type nodeColor int

const (
	// white is unvisited.
	white nodeColor = iota
	// grey is on the current search path, so reaching it again closes a cycle.
	grey
	// black is fully explored and known to lead to no cycle.
	black
)

// detectCycles finds every cycle in a directed graph by depth-first search.
//
// A back edge to a node already on the search path — a grey node — closes a cycle, and the
// path from that node to the current one is the cycle itself. Nodes are visited in sorted
// order so the diagnostics are deterministic rather than dependent on map iteration.
//
// rel supplies the diagnostic's title, the relationship's name, and the remedy to suggest.
func detectCycles(edges map[string][]edge, rel relation) []error {
	color := map[string]nodeColor{}
	var path []edge
	var findings []error
	reported := map[string]bool{}

	roots := make([]string, 0, len(edges))
	for node := range edges {
		roots = append(roots, node)
	}
	sort.Strings(roots)

	var visit func(node string)
	visit = func(node string) {
		color[node] = grey

		for _, e := range sortedEdges(edges[node]) {
			switch color[e.to] {
			case grey:
				// e closes a cycle back onto e.to.
				cycle := cycleFrom(append(path, e), e.to)
				if key := cycleKey(cycle); !reported[key] {
					reported[key] = true
					findings = append(findings, cycleFinding(cycle, rel))
				}
			case white:
				path = append(path, e)
				visit(e.to)
				path = path[:len(path)-1]
			}
		}
		color[node] = black
	}

	for _, node := range roots {
		if color[node] == white {
			path = path[:0]
			visit(node)
		}
	}
	return findings
}

// sortedEdges returns a node's outgoing edges ordered by target, for deterministic output.
func sortedEdges(list []edge) []edge {
	out := make([]edge, len(list))
	copy(out, list)
	sort.SliceStable(out, func(i, j int) bool { return out[i].to < out[j].to })
	return out
}

// cycleFrom trims a search path down to the cycle it closes, dropping the prefix that merely
// led into it.
func cycleFrom(path []edge, start string) []edge {
	for i, e := range path {
		if e.from == start {
			return path[i:]
		}
	}
	return path
}

// cycleKey builds a rotation-independent identity for a cycle, so the same loop found from a
// different entry point is reported only once.
//
// The node list is rotated to begin at its smallest member, which makes A→B→A and B→A→B
// produce the same key.
func cycleKey(cycle []edge) string {
	if len(cycle) == 0 {
		return ""
	}
	nodes := make([]string, 0, len(cycle))
	for _, e := range cycle {
		nodes = append(nodes, e.from)
	}

	smallest := 0
	for i, n := range nodes {
		if n < nodes[smallest] {
			smallest = i
		}
	}
	rotated := append(nodes[smallest:], nodes[:smallest]...)
	return strings.Join(rotated, "\x00")
}

// cycleFinding renders a cycle as a diagnostic anchored at the edge that closed it.
//
// The closing edge is the useful anchor: it is the import the programmer most likely just
// added, and the message spells the whole loop so the other participants are visible too.
func cycleFinding(cycle []edge, rel relation) error {
	closing := cycle[len(cycle)-1]
	label := rel.nodeLabel
	if label == nil {
		label = func(node string) string { return node }
	}

	nodes := make([]string, 0, len(cycle)+1)
	for _, e := range cycle {
		nodes = append(nodes, label(e.from))
	}
	nodes = append(nodes, label(closing.to))

	return helpers.NewNamedDiagnostic(closing.start, closing.end, rel.diagnosticName, rel.heading, fmt.Sprintf(
		"%s cycle: %s. A cycle through %ss is a compiler error; %s",
		rel.noun, strings.Join(nodes, " -> "), rel.noun, rel.remedy))
}
