package parser

import (
	"sort"
	"testing"

	symboltable "github.com/samkrao/fo-lang/src/context"
)

// A compilation unit's own symbol belongs to the segment its file OPENS with.
//
// A context is divided into visibility segments: a declaration following an
// executable item opens a further one (Appendix B.9), and a lookup reaches up
// through ParentId, so a name bound in a later segment is invisible to everything
// anchored in an earlier one. A record binds into the segment it was minted in
// (docs/language-ref.md, B.4), which makes WHEN the parser mints a symbol decide
// where the name lives.
//
// Minting the unit symbol after its body therefore filed the file itself under
// whichever segment the body happened to leave active. `x; println(x); z;` put it
// in the segment `z` opened; the same program without the call put it in the
// first. The compilation unit covers the whole file and cannot move with an
// unrelated statement, so it is minted before the body is read.
func TestCompilationUnitSymbolAnchorsToTheOpeningSegment(t *testing.T) {
	sources := map[string]string{
		"declarations only": `x co.lang.int = 10;
y ?= 20;
z ?= 30;`,
		"executable item last": `x co.lang.int = 10;
co.out.println(x);`,
		"declaration after an executable item": `x co.lang.int = 10;
y ?= 20;

co.out.println(x);

z := x + y;

co.out.println(z);`,
	}

	for name, source := range sources {
		t.Run(name, func(t *testing.T) {
			_, p := parsePackageSource(t, source, "appl.fol")
			if len(p.diags) != 0 {
				t.Fatalf("entry file produced diagnostics: %v", p.diags)
			}

			const unitKey = "appl.fol_" + string(symboltable.S_PackageSymbol)
			opening, holder := openingSegment(t, p), ""
			for id, table := range p.fs.SymboltableMap {
				if _, bound := table.SymbolsByName[unitKey]; bound {
					holder = id
				}
			}
			if holder == "" {
				t.Fatalf("%s is not bound in any segment", unitKey)
			}
			if holder != opening {
				t.Errorf("%s is bound in a later segment (%s), where a lookup from the opening segment (%s) cannot reach it",
					unitKey, holder, opening)
			}
		})
	}
}

// The hoist must not flatten the segments themselves: a declaration that follows
// an executable item still opens a new one, and still lands there.
func TestDeclarationAfterAnExecutableItemStillOpensASegment(t *testing.T) {
	_, p := parsePackageSource(t, `x co.lang.int = 10;

co.out.println(x);

z ?= 20;`, "appl.fol")
	if len(p.diags) != 0 {
		t.Fatalf("entry file produced diagnostics: %v", p.diags)
	}

	opening := openingSegment(t, p)
	for id, table := range p.fs.SymboltableMap {
		_, hasX := table.SymbolsByName["x_fo_Var"]
		_, hasZ := table.SymbolsByName["z_fo_Var"]
		if hasX && id != opening {
			t.Errorf("x was bound outside the opening segment")
		}
		if hasZ && id == opening {
			t.Errorf("z was bound in the opening segment; the interleaving rule stopped applying")
		}
	}
}

// openingSegment returns the id of the segment with no parent — the one the file
// opens with, and the only one every later segment can reach up into.
func openingSegment(t *testing.T, p *parser) string {
	t.Helper()
	roots := []string{}
	for id, table := range p.fs.SymboltableMap {
		if table.ParentId == "" {
			roots = append(roots, id)
		}
	}
	sort.Strings(roots)
	if len(roots) != 1 {
		t.Fatalf("expected exactly one opening segment, found %d: %v", len(roots), roots)
	}
	return roots[0]
}
