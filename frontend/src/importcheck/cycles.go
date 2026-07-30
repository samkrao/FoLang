package importcheck

import (
	"fmt"
	"sort"
	"strings"

	"github.com/samkrao/fo-lang/frontend/src/helpers"
)

// Cycle detection.
//
// docs/language-ref.md, "Cycles" requires a compiler error when a cycle exists through either
// of two relations:
//
//	packageA imports packageB, and packageB imports packageA
//	realm="x", parent-realm="y" and another import uses realm="y", parent-realm="x"
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

// Graph accumulates the import and realm relationships of every parsed file.
//
// It is not safe for concurrent use; a driver parsing files in parallel must add to it under
// its own lock, or build one graph per worker and merge them.
type Graph struct {
	// packageEdges maps an importing package to the packages it imports.
	packageEdges map[string][]edge
	// realmEdges maps a realm to its declared parent realms.
	realmEdges map[string][]edge
}

// NewGraph creates an empty relationship graph.
func NewGraph() *Graph {
	return &Graph{
		packageEdges: map[string][]edge{},
		realmEdges:   map[string][]edge{},
	}
}

// Add records one file's relationships.
//
// A file with no package identity — one at the project root, which is not a package — still
// contributes its realm edges, because those are independent of where the file lives.
func (g *Graph) Add(f File) {
	if g == nil {
		return
	}

	for _, imp := range f.Imports {
		// A self-edge is a one-node cycle that ValidateSelfImports already reports with a
		// clearer message, so it is left out of the graph rather than reported twice.
		if target := imp.Package; target != "" && f.PackagePath != "" && target != f.PackagePath {
			g.packageEdges[f.PackagePath] = append(g.packageEdges[f.PackagePath], edge{
				from:  f.PackagePath,
				to:    target,
				start: imp.Start,
				end:   imp.End,
				file:  f.Name,
			})
		}

		// A realm declares its parent, giving an edge from child to parent.
		if imp.Realm != "" && imp.ParentRealm != "" {
			g.realmEdges[imp.Realm] = append(g.realmEdges[imp.Realm], edge{
				from:  imp.Realm,
				to:    imp.ParentRealm,
				start: imp.Start,
				end:   imp.End,
				file:  f.Name,
			})
		}
	}
}

// Validate reports every cycle in the accumulated graph.
//
// Both relations are checked, and each distinct cycle is reported once. A driver calls this
// after every file has been parsed and added.
func (g *Graph) Validate() []error {
	if g == nil {
		return nil
	}
	var findings []error
	findings = append(findings, detectCycles(g.packageEdges, relationPackageImport)...)
	findings = append(findings, detectCycles(g.realmEdges, relationRealmParent)...)
	return findings
}

// relation describes one cyclic relationship for diagnostic purposes: how to title the
// finding, how to name the relationship, and what remedy to suggest. The remedy differs
// between the two, so it travels with the relation rather than being hard-coded.
type relation struct {
	errorName string
	noun      string
	remedy    string
}

var (
	relationPackageImport = relation{
		errorName: "Package Import Cycle",
		noun:      "package import",
		remedy:    "break the loop by moving the shared declarations into a third package that both sides can import",
	}
	relationRealmParent = relation{
		errorName: "Realm Cycle",
		noun:      "realm parent relationship",
		remedy:    "a realm hierarchy must be a tree, so give these realms a single common parent instead of making each the parent of the other",
	}
)

// ValidateSelfImports reports a package that imports itself.
//
// This is the degenerate one-node cycle, and it is the only cycle a single file can detect on
// its own, so it is reported during parsing rather than deferred to the whole-project pass.
func ValidateSelfImports(f File) []error {
	if f.PackagePath == "" {
		return nil
	}

	var findings []error
	for _, imp := range f.Imports {
		if imp.Package == f.PackagePath {
			findings = append(findings, finding(imp, "Package Import Cycle", fmt.Sprintf(
				"package %q imports itself; a package's own declarations are already visible to it",
				f.PackagePath)))
		}
	}
	return findings
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

	nodes := make([]string, 0, len(cycle)+1)
	for _, e := range cycle {
		nodes = append(nodes, e.from)
	}
	nodes = append(nodes, closing.to)

	return helpers.NewExpectedTokenErrorName(closing.start, closing.end, rel.errorName, fmt.Sprintf(
		"%s cycle: %s. A cycle through %ss is a compiler error; %s",
		rel.noun, strings.Join(nodes, " -> "), rel.noun, rel.remedy))
}
