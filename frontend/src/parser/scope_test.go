package parser

import (
	"testing"

	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
)

// The scope model the parse produces — docs/language-ref.md, Appendix B.
//
// B.1 is a worked example with a drawing of the contexts and symbol-table
// segments it must produce, so it is parsed here and compared against that
// drawing directly. The invariant checks in B.7 are then applied to the whole
// map, which is what catches a context that was created but never linked and a
// segment left behind by a speculative parse that was thrown away.

// referenceUnit is the source of docs/language-ref.md B.1, verbatim.
const referenceUnit = `_ co.lang.unit = {
    firstfun()->() = {
        k co.lang.int = 10;
        v := 20;

        co.out.println(k + v);

        j ?= 30;

        {
            j co.lang.char = 'A';
            co.out.println(j);
        }

        co.out.println(j);
    }

    secondfun()->() = {
        k co.lang.int = 10;
        v := 20;

        co.out.println(k + v);

        j ?= 30;

        {
            j co.lang.char = 'A';
            co.out.println(j);
        }

        co.out.println(j);
    }
}`

// TestReferenceUnitBuildsTheAppendixBScopeModel checks the parse against the tree
// drawn in B.1: a function context per function, TWO visibility segments in each
// because a declaration follows a call, and a block context that branches from
// the SECOND segment rather than from the function's first or final one.
func TestReferenceUnitBuildsTheAppendixBScopeModel(t *testing.T) {
	_, p := parsePackageSource(t, referenceUnit, "some.unit.fol")
	if len(p.diags) != 0 {
		t.Fatalf("the reference unit produced diagnostics: %v", p.diags)
	}

	unit := onlyChild(t, p.fs, p.ctx)
	if unit.ContextType_ != string(symboltable.S_ModuleSymbol) {
		t.Errorf("the unit body context is %q, want %q", unit.ContextType_, symboltable.S_ModuleSymbol)
	}
	if len(unit.ChildCtxIds) != 2 {
		t.Fatalf("the unit body has %d child contexts, want one per function", len(unit.ChildCtxIds))
	}

	for _, id := range unit.ChildCtxIds {
		function := p.fs.GetContext(id)
		if function.ContextType_ != string(symboltable.S_FunctionSymbol) {
			t.Fatalf("a unit member context is %q, want %q", function.ContextType_, symboltable.S_FunctionSymbol)
		}

		// The function's segments, newest first. `k` and `v` are the first
		// frontier; the call closes it and `j ?= 30` opens the second.
		segments := segmentChain(p.fs, function)
		if len(segments) != 2 {
			t.Fatalf("the function context owns %d symbol-table segments, want 2: a declaration written after a call is a new visibility frontier", len(segments))
		}
		if segments[1].ParentId != "" {
			t.Errorf("the first segment has ParentId %q, want it empty", segments[1].ParentId)
		}
		if segments[0].ParentId != segments[1].Id {
			t.Errorf("the second segment chains to %q, want the first segment %q", segments[0].ParentId, segments[1].Id)
		}

		block := onlyChild(t, p.fs, function)
		if block.ContextType_ != string(symboltable.S_BlockSymbol) {
			t.Errorf("the nested block context is %q, want %q", block.ContextType_, symboltable.S_BlockSymbol)
		}
		if block.ParentCtxSymbolTableId != segments[0].Id {
			t.Errorf("the block branched from segment %q, want the second segment %q; branching from an earlier one would hide the outer j, and from a later one would show it declarations it must not see",
				block.ParentCtxSymbolTableId, segments[0].Id)
		}
		if len(segmentChain(p.fs, block)) != 1 {
			t.Errorf("the block context owns %d segments, want 1: nothing in it follows a statement", len(segmentChain(p.fs, block)))
		}
	}
}

// TestDeclarationSymbolsAreAnchoredToTheirOwnSegment covers what the segments are
// FOR.
//
// A symbol's anchor is the visibility state at its source position: the segment its
// declaration binds into, and the point a deferred resolution must start from. `k`
// is declared before the call and `j` after it, so a shared anchor would let a
// reference to `k` see `j`, which is exactly what B.5 says must not happen.
// bind_test.go checks the bindings those anchors produce.
func TestDeclarationSymbolsAreAnchoredToTheirOwnSegment(t *testing.T) {
	root, p := parsePackageSource(t, referenceUnit, "some.unit.fol")
	if len(p.diags) != 0 {
		t.Fatalf("the reference unit produced diagnostics: %v", p.diags)
	}

	unit := root.(ast.PackageStmt).Body[0].(ast.TypeDeclarationStmt)
	body := unit.Body[0].(ast.FunctionDeclarationStmt).Body

	early := body[0].(ast.VarDeclarationStmt).Symb.SymbolTableId
	late := body[3].(ast.VarDeclarationStmt).Symb.SymbolTableId

	if early == "" || late == "" {
		t.Fatalf("a declaration was minted with no symbol-table anchor: k=%q j=%q", early, late)
	}
	if early == late {
		t.Errorf("k and j are both anchored to segment %q; the declaration written after the call belongs to a later frontier", early)
	}

	function := p.fs.GetContext(onlyChild(t, p.fs, p.ctx).ChildCtxIds[0])
	segments := segmentChain(p.fs, function)
	if late != segments[0].Id || early != segments[1].Id {
		t.Errorf("anchors are k=%q j=%q, want the first and second segments %q and %q", early, late, segments[1].Id, segments[0].Id)
	}
}

// TestScopeModelHoldsTheStructuralInvariants applies B.7 to a file that exercises
// the constructs which open a scope, including the ones a speculative parse has
// to reconsider.
func TestScopeModelHoldsTheStructuralInvariants(t *testing.T) {
	source := `_ co.lang.unit = {
    outer(seed co.lang.int)->(co.lang.int) = {
        total := seed;

        helper(step co.lang.int)->(co.lang.int) = {
            this.return step * 2;
        }

        total = helper(total);

        {
            scratch co.lang.int = 1;
            total = total + scratch;
        }

        widen = (a co.lang.int)(b co.lang.int) ==>> a + b;

        this.return widen(total)(1);
    }
}`

	_, p := parsePackageSource(t, source, "shapes.unit.fol")
	if len(p.diags) != 0 {
		t.Fatalf("the source produced diagnostics: %v", p.diags)
	}
	checkScopeInvariants(t, p.fs, p.ctx)
}

// TestEveryNonVariableItemClosesADeclarationRun covers the complete B.9 rule.
// Earlier coverage exercised calls and child blocks, but non-variable local
// declarations and empty statements are intervening context-level items too.
func TestEveryNonVariableItemClosesADeclarationRun(t *testing.T) {
	tests := []struct {
		name        string
		intervening string
	}{
		{name: "empty statement", intervening: ";"},
		{name: "local function", intervening: "helper()->() = {}"},
		{name: "closure declaration", intervening: "helper = () ==>> 1;"},
		{name: "named block", intervening: "helper co.lang.block = {}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := `_ co.lang.unit = {
    subject()->() = {
        before := 1;
        ` + tt.intervening + `
        after := 2;
    }
}`
			_, p := parsePackageSource(t, source, "frontier.unit.fol")
			if len(p.diags) != 0 {
				t.Fatalf("the source produced diagnostics: %v", p.diags)
			}

			unit := onlyChild(t, p.fs, p.ctx)
			function := p.fs.GetContext(unit.ChildCtxIds[0])
			if got := len(segmentChain(p.fs, function)); got != 2 {
				t.Errorf("the function owns %d segments, want 2: %s must close the variable-declaration run", got, tt.name)
			}
			checkScopeInvariants(t, p.fs, p.ctx)
		})
	}
}

func TestClosureBodyStatementUsesTheClosureContext(t *testing.T) {
	root, p := parsePackageSource(t, `_ co.lang.unit = {
    subject()->() = {
        helper = (value co.lang.int) ==>> value + 1;
    }
}`, "closure.unit.fol")
	if len(p.diags) != 0 {
		t.Fatalf("the source produced diagnostics: %v", p.diags)
	}

	unitStmt := root.(ast.PackageStmt).Body[0].(ast.TypeDeclarationStmt)
	functionStmt := unitStmt.Body[0].(ast.FunctionDeclarationStmt)
	closureStmt := functionStmt.Body[0].(ast.FunctionDeclarationStmt)
	bodyStmt := closureStmt.Body[0].(ast.ExpressionStmt)

	unitCtx := onlyChild(t, p.fs, p.ctx)
	functionCtx := p.fs.GetContext(unitCtx.ChildCtxIds[0])
	closureCtx := onlyChild(t, p.fs, functionCtx)
	if bodyStmt.Symb.SymbolTableId != closureCtx.SymbolTable_ {
		t.Errorf("closure body is anchored to %q, want its closure context segment %q", bodyStmt.Symb.SymbolTableId, closureCtx.SymbolTable_)
	}
}

// TestSpeculationLeavesNoContextBehind covers the one way the model can grow
// entries that describe nothing.
//
// A block whose last item carries no ";" is read twice: once speculatively as a
// tail expression, and again as a statement when that reading is rejected. Any
// context the abandoned reading opened would be a second, unreachable copy of a
// scope in the file.
func TestSpeculationLeavesNoContextBehind(t *testing.T) {
	source := `_ co.lang.unit = {
    subject()->(co.lang.int) = {
        base := 1;
        values := [1, 2, 3];
        values.map(|v| => { v + base })
    }
}`

	_, p := parsePackageSource(t, source, "tail.unit.fol")
	if len(p.diags) != 0 {
		t.Fatalf("the source produced diagnostics: %v", p.diags)
	}
	checkScopeInvariants(t, p.fs, p.ctx)

	if left := len(p.scopeJournal); left != 0 {
		t.Errorf("the scope journal still holds %d undo entries after the parse, so speculation state outlived it", left)
	}
}

// segmentChain returns a context's symbol-table segments from the active one back
// to the first, following SymbolTable.ParentId.
func segmentChain(fs *symboltable.FolangSymbols, ctx *symboltable.Context) []*symboltable.SymbolTable {
	var chain []*symboltable.SymbolTable
	for id := ctx.SymbolTable_; id != ""; {
		table := fs.GetSymbolTable(id)
		if table == nil {
			return chain
		}
		chain = append(chain, table)
		id = table.ParentId
	}
	return chain
}

// onlyChild returns a context's single child, failing when it has any other
// number of them.
func onlyChild(t *testing.T, fs *symboltable.FolangSymbols, ctx *symboltable.Context) *symboltable.Context {
	t.Helper()
	if len(ctx.ChildCtxIds) != 1 {
		t.Fatalf("context %s has %d children, want exactly 1", ctx.Id, len(ctx.ChildCtxIds))
	}
	child := fs.GetContext(ctx.ChildCtxIds[0])
	if child == nil {
		t.Fatalf("context %s names a child %s that is absent from the context map", ctx.Id, ctx.ChildCtxIds[0])
	}
	return child
}

// checkScopeInvariants asserts the structural invariants of B.7 over a whole
// parse, plus reachability: a context that no parent names describes a reading
// the parser abandoned and must not be in the map at all.
func checkScopeInvariants(t *testing.T, fs *symboltable.FolangSymbols, root *symboltable.Context) {
	t.Helper()

	for id, table := range fs.SymboltableMap {
		if table.Id != id {
			t.Errorf("symbol table keyed %q reports id %q", id, table.Id)
		}
		owner := fs.GetContext(table.ContextId)
		if owner == nil {
			t.Errorf("symbol table %s names context %q, which is absent from the context map", table.Id, table.ContextId)
			continue
		}
		if table.ParentId == "" {
			continue
		}
		previous := fs.GetSymbolTable(table.ParentId)
		if previous == nil {
			t.Errorf("symbol table %s chains to %q, which is absent from the table map", table.Id, table.ParentId)
			continue
		}
		if previous.ContextId != table.ContextId {
			t.Errorf("symbol table %s chains to %s, which is owned by another context: a segment chain never leaves its context", table.Id, previous.Id)
		}
	}

	for id, ctx := range fs.ContextMap {
		if ctx.Id != id {
			t.Errorf("context keyed %q reports id %q", id, ctx.Id)
		}
		if active := fs.GetSymbolTable(ctx.SymbolTable_); active == nil {
			t.Errorf("context %s names active segment %q, which is absent from the table map", ctx.Id, ctx.SymbolTable_)
		} else if active.ContextId != ctx.Id {
			t.Errorf("context %s names active segment %s, which is owned by context %s", ctx.Id, active.Id, active.ContextId)
		}
		if ctx.ParentId == "" {
			continue
		}
		parent := fs.GetContext(ctx.ParentId)
		if parent == nil {
			t.Errorf("context %s names parent %q, which is absent from the context map", ctx.Id, ctx.ParentId)
			continue
		}
		if anchor := fs.GetSymbolTable(ctx.ParentCtxSymbolTableId); anchor == nil {
			t.Errorf("context %s names branch point %q, which is absent from the table map", ctx.Id, ctx.ParentCtxSymbolTableId)
		} else if anchor.ContextId != parent.Id {
			t.Errorf("context %s branches from segment %s, which belongs to %s rather than to its parent %s", ctx.Id, anchor.Id, anchor.ContextId, parent.Id)
		}
	}

	reachable := map[string]bool{}
	var walk func(ctx *symboltable.Context)
	walk = func(ctx *symboltable.Context) {
		if ctx == nil || reachable[ctx.Id] {
			return
		}
		reachable[ctx.Id] = true
		for _, id := range ctx.ChildCtxIds {
			child := fs.GetContext(id)
			if child == nil {
				t.Errorf("context %s names child %q, which is absent from the context map", ctx.Id, id)
				continue
			}
			if child.ParentId != ctx.Id {
				t.Errorf("context %s lists %s as a child, but that context's parent is %s", ctx.Id, child.Id, child.ParentId)
			}
			walk(child)
		}
	}
	walk(root)

	for id := range fs.ContextMap {
		if !reachable[id] {
			t.Errorf("context %s is in the map but reachable from no parent", id)
		}
	}
	for id, table := range fs.SymboltableMap {
		if !reachable[table.ContextId] {
			t.Errorf("symbol table %s belongs to unreachable context %s", id, table.ContextId)
		}
	}
}
