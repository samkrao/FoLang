package parser

import (
	"fmt"

	symboltable "github.com/samkrao/fo-lang/src/context"
	"github.com/samkrao/fo-lang/src/helpers"
)

// Contexts and symbol-table segments built during the parse.
//
// docs/language-ref.md, Appendix B, fixes when each of the two structures comes
// into being, and both conditions are decidable from the token stream alone,
// which is why they are recognised here rather than re-derived by a later pass:
//
//   - A NEW CONTEXT begins where a new block opens with a brace that is not a
//     literal expression. A collection, map or object-construction brace carries
//     no declarations and so opens nothing.
//
//   - A NEW SYMBOL-TABLE SEGMENT begins with a new context, and again whenever a
//     variable declaration follows a statement or an expression in the same
//     context. That second rule is what B.2 calls the visibility frontier: `k`
//     and `v` are visible before the first call, and the `j` declared after it
//     is a later frontier that must not be visible to the earlier code.
//
// What this file does NOT do is put names in those tables; bind.go does, and it
// chooses the table by the symbol rather than by the cursor. Every symbol record
// minted for a node carries the id of the segment active at that node's source
// position (see symbolfactory.go), which is both the use-site visibility anchor
// invariants 13 and 14 require and the table its declaration binds into. Without
// it a deferred reference would have to restart from the context's final segment
// and could see declarations that came after it.
//
// # A body block does not open a second context
//
// A function, lambda or pattern clause owns a scope that starts at its parameter
// list, before its brace. Those constructs push their own context and then parse
// the body with parseScopeBlock, so the brace joins the scope already open
// instead of nesting a block context inside it. That is what B.1 draws: the body
// of `firstfun` is `firstfun_context`, not a block context within it.
//
// # Speculation
//
// A speculative parse that is thrown away must leave no context behind, or the
// same source would be recorded twice. Every mutation made here is therefore
// journalled with its inverse while a speculation is in flight, and speculate
// runs the inverses in reverse on the path that rewinds the cursor.

// scopeFrame is the parser state one context saves so that leaving it restores
// the enclosing scope exactly.
type scopeFrame struct {
	ctx           *symboltable.Context
	symtab        *symboltable.SymbolTable
	sawExecutable bool
}

// pushContext opens a child context of the current one and returns the function
// that closes it. Call it as `defer p.pushContext(kind)()`, which makes the pop
// run on a bailout as well as on the ordinary path.
//
// The child records two different parents: ParentId, the context that structurally
// contains it, and ParentCtxSymbolTableId, the exact segment of that context which
// was active at the brace. Lexical lookup continues from the second, so a
// declaration written after the child was entered cannot become visible inside it.
func (p *parser) pushContext(kind symboltable.SymbolsToString, owner ...symboltable.SymbolInfo) func() {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	saved := scopeFrame{ctx: p.ctx, symtab: p.symtab, sawExecutable: p.sawExecutable}

	child, table := CreateNewContext(p.ctx.Id, kind, p.identity, fmt.Sprintf("token:%d", p.pos), fmt.Sprintf("child:%d", len(p.ctx.ChildCtxIds)))
	if len(owner) > 0 && owner[0] != nil {
		// OwnerSymbolId is an ID, not a pointer, so the link is only as good as
		// the registry: a reader resolves it through GetSymbol. A scope owner is
		// minted before its body and bound after it, and some are never bound at
		// all — a lowered construct's owner reaches no declaration — so waiting
		// for Declare to register it leaves the link dangling for exactly the
		// symbols the field exists to name.
		child.OwnerSymbolId = owner[0].GetSymbolID()
		p.registerSymbol(owner[0])
		previousOwnedContext := owner[0].GetContextID()
		owner[0].SetOwnedContextID(child.Id)
		p.journal(func() { owner[0].SetOwnedContextID(previousOwnedContext) })
	}
	child.ParentCtxSymbolTableId = p.ctx.SymbolTable_

	parent := p.ctx
	children := len(parent.ChildCtxIds)
	parent.ChildCtxIds = append(parent.ChildCtxIds, child.Id)
	p.journal(func() { parent.ChildCtxIds = parent.ChildCtxIds[:children] })

	p.fs.AddContext(child)
	p.journal(func() { delete(p.fs.ContextMap, child.Id) })
	p.addSymbolTable(table)

	p.ctx, p.symtab, p.sawExecutable = child, table, false

	return func() {
		p.ctx, p.symtab, p.sawExecutable = saved.ctx, saved.symtab, saved.sawExecutable
	}
}

// scoped runs parse inside a fresh child context.
//
// Use it where the scope covers only PART of a parse function, so that the
// enclosing construct's own symbol — a function's name, a let expression's node —
// is still minted in the scope that declares it. The context closes on a bailout
// as well as on the ordinary path.
func (p *parser) scoped(kind symboltable.SymbolsToString, parse func(), owner ...symboltable.SymbolInfo) {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	defer p.pushContext(kind, owner...)()
	parse()
}

// newSymbolSegment starts a further visibility segment in the current context and
// makes it the active one. The context's SymbolTable_ advances to the new segment
// and the previous one stays reachable through the new segment's ParentId.
func (p *parser) newSymbolSegment() {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	ctx := p.ctx
	previous := ctx.SymbolTable_

	table := &symboltable.SymbolTable{
		Id:            helpers.StableID("symtab", ctx.Id, "segment", fmt.Sprintf("%d", contextSegmentCount(p.fs, ctx))),
		ParentId:      previous,
		ContextId:     ctx.Id,
		Prefix:        ctx.Id,
		SymbolsByName: map[string][]string{},
	}

	ctx.SymbolTable_ = table.Id
	p.journal(func() { ctx.SymbolTable_ = previous })
	p.addSymbolTable(table)

	p.symtab = table
}

func contextSegmentCount(fs *symboltable.FolangSymbols, ctx *symboltable.Context) int {
	count := 0
	for id := ctx.SymbolTable_; id != ""; {
		count++
		table := fs.GetSymbolTable(id)
		if table == nil {
			break
		}
		id = table.ParentId
	}
	return count
}

// addSymbolTable registers a table with the parse's symbol model.
func (p *parser) addSymbolTable(table *symboltable.SymbolTable) {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	p.fs.AddSymbolTable(table)
	p.journal(func() { delete(p.fs.SymboltableMap, table.Id) })
}

// noteExecutableItem records that a non-variable context-level item has been read
// in the current context. The next variable declaration here is therefore an
// interleaved one and opens a new visibility segment. The historical name says
// "executable", but Appendix B.9 also includes non-variable declarations and
// empty statements.
//
// Variable declarations deliberately do not set this: a consecutive run of them
// is one frontier, however many names it introduces.
func (p *parser) noteExecutableItem() {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	p.sawExecutable = true
}

// beginDeclarationSegment is called immediately before a variable declaration is
// parsed, so that the declaration's symbol is anchored to the segment that will
// own it rather than to the one the preceding statement saw.
func (p *parser) beginDeclarationSegment() {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	if !p.sawExecutable {
		return
	}
	p.sawExecutable = false
	p.newSymbolSegment()
}

// journal records the inverse of a scope mutation while a speculation is running.
//
// Outside a speculation nothing can be rewound, so nothing is recorded and the
// log does not grow over a whole file.
func (p *parser) journal(undo func()) {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	if p.speculating == 0 {
		return
	}
	p.scopeJournal = append(p.scopeJournal, undo)
}

// rollbackScopes undoes every scope mutation recorded since mark, most recent
// first, so that composite changes come apart in the order they were made.
func (p *parser) rollbackScopes(mark int) {
	if traceEnabled || DEBUG_TRACE {
		defer p.traceEnd(p.traceBegin())
	}

	for i := len(p.scopeJournal) - 1; i >= mark; i-- {
		p.scopeJournal[i]()
	}
	p.scopeJournal = p.scopeJournal[:mark]
}
