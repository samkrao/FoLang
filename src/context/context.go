// Package symboltable provides symbol table and context management for the fo-lang compiler.
package symboltable

// ResolutionPolicy is the closed frontend resolver-policy vocabulary defined by
// docs/language-ref.md Appendix B.5. It remains string-backed so serialized AST
// artifacts retain the specified spellings.

type SymbolID string
type ContextID string
type SymbolTableID string
type PackageName string
type SymbolName string
type ImportedAlias string
type ResolutionPolicy string

const (
	LexicalOrdered           ResolutionPolicy = "lexical_ordered"
	LexicalCompleteContainer ResolutionPolicy = "lexical_complete_container"
	LateLexicalCallSite      ResolutionPolicy = "late_lexical_call_site"
	LateLexicalFormationSite ResolutionPolicy = "late_lexical_formation_site"
	MacroDefinitionSite      ResolutionPolicy = "macro_definition_site"
	MacroExpansionSite       ResolutionPolicy = "macro_expansion_site"
	RuntimeBound             ResolutionPolicy = "runtime_bound"
	DynamicCallSite          ResolutionPolicy = "dynamic_call_site"
	LexicalCallSite          ResolutionPolicy = "lexical_call_site"
	MixedCallSite            ResolutionPolicy = "mixed_call_site"
)

// SymbolTable represents a hierarchical chain of symbol mappings within a context.

type FolangSymbols struct {
	RootContextId  ContextID
	SymboltableMap map[SymbolTableID]*SymbolTable
	ContextMap     map[ContextID]ContextInfo
	SymbolsById    map[SymbolID]SymbolInfo
}

type ContextKind string

const (
	ContextKindLexical ContextKind = "context"
	ContextKindFol     ContextKind = "fol-context"
)

// ContextInfo is the read-only view shared by lexical contexts and the project
// FolContext. Graph mutation remains centralized on FolangSymbols.
type ContextInfo interface {
	GetId() ContextID
	GetSymbolTableId() []SymbolTableID
	GetContextKind() ContextKind
}

// FolContext identifies the two entry points of one compiled FoLang project.
// It is deliberately not a Context: the link between the published surface and
// the operational root is transparent project structure, not lexical ancestry.
type FolContext struct {
	Id               ContextID
	SymbolTables_    []SymbolTableID
	Context_         ContextID
	Kind             string
	ChildCtxIds      []ContextID               // published surface contexts
	ExportedPackages map[PackageName]ContextID // exported package name -> Context ID
}

func (c *FolContext) GetId() ContextID                  { return c.Id }
func (c *FolContext) GetSymbolTableId() []SymbolTableID { return c.SymbolTables_ }
func (c *FolContext) GetContextKind() ContextKind       { return ContextKindFol }

func (fs *FolangSymbols) AddSymbolTable(st *SymbolTable) {
	fs.SymboltableMap[st.Id] = st
}
func (fs *FolangSymbols) AddContext(ctx *Context) {
	fs.ContextMap[ctx.Id] = ctx
}
func (fs *FolangSymbols) AddFolContext(ctx *FolContext) {
	fs.ContextMap[ctx.Id] = ctx
	fs.RootContextId = ctx.Id
}
func (fs *FolangSymbols) CreateFolangSymbols() {
	fs.SymboltableMap = make(map[SymbolTableID]*SymbolTable)
	fs.ContextMap = make(map[ContextID]ContextInfo)
	fs.SymbolsById = make(map[SymbolID]SymbolInfo)
}

// RegisterSymbol stores the canonical symbol record addressed by its durable ID.
func (fs *FolangSymbols) RegisterSymbol(symbol SymbolInfo) {
	if fs.SymbolsById == nil {
		fs.SymbolsById = make(map[SymbolID]SymbolInfo)
	}
	fs.SymbolsById[symbol.GetSymbolID()] = symbol

}

// Bindings returns the declaration-key view used by semantic passes. The map is
// a transient view; canonical ownership remains in FolangSymbols.SymbolsById.
func (fs *FolangSymbols) Bindings(tableID SymbolTableID) map[SymbolName]SymbolInfo {
	s := fs.GetSymbolTable(tableID)
	if s == nil {
		return nil
	}
	out := make(map[SymbolName]SymbolInfo, len(s.SymbolsByName))
	for key, ids := range s.SymbolsByName {
		if len(ids) != 0 {
			out[key] = fs.GetSymbol(ids[0])
		}
	}
	return out
}

// UnregisterSymbol drops one record from the canonical registry.
//
// It is the inverse RegisterSymbol needs for the parser's speculation rollback.
// The registry, not the symbol table, is now where a record LIVES, so unbinding a
// name no longer erases it: without this the record survives a branch that was
// thrown away, reaches the artifact through SymbolsById, and presents itself
// there as a declaration the program never made.
//
// It removes the named record only. A nested symbol reachable from it is left
// alone, because deleting a record some other binding still points at would turn
// a leak into a dangling reference, which the artifact reader rejects outright.
func (fs *FolangSymbols) UnregisterSymbol(id SymbolID) {
	delete(fs.SymbolsById, id)
}

// GetSymbol resolves an AST SymbolId to its canonical symbol-table record.
func (fs *FolangSymbols) GetSymbol(id SymbolID) SymbolInfo { return fs.SymbolsById[id] }
func (fs *FolangSymbols) GetSymbolTable(id SymbolTableID) *SymbolTable {
	return fs.SymboltableMap[id]
}
func (fs *FolangSymbols) GetContext(id ContextID) *Context {
	ctx, _ := fs.ContextMap[id].(*Context)
	return ctx
}

func (fs *FolangSymbols) GetContextInfo(id ContextID) ContextInfo { return fs.ContextMap[id] }
func (fs *FolangSymbols) GetFolContext(id ContextID) *FolContext {
	ctx, _ := fs.ContextMap[id].(*FolContext)
	return ctx
}
func (fs *FolangSymbols) RootFolContext() *FolContext {
	if fs == nil {
		return nil
	}
	return fs.GetFolContext(fs.RootContextId)
}

// FolContextRootContextID returns the operational root reached through the
// transparent FolContext descriptor. RootContextId remains a compatibility
// fallback for graphs produced before FolContext was added.
func (fs *FolangSymbols) FolContextRootContextID() ContextID {
	if fs != nil {
		if root := fs.RootFolContext(); root != nil && root.Context_ != "" {
			return root.Context_
		}
	}
	if fs == nil {
		return ""
	}
	return fs.RootContextId
}

// SymbolTable is one declaration-order segment owned by a Context.
type SymbolTable struct {
	Id        SymbolTableID // id of the symbol table
	ParentId  SymbolTableID
	ContextId ContextID // holds context id of the symbol table
	Prefix    string
	// SymbolIds preserves declaration order. SymbolsByName indexes declaration
	// keys (including overload signatures) into the canonical SymbolsById map.
	SymbolIds     []SymbolID
	SymbolsByName map[SymbolName][]SymbolID
}

// Context represents a scoping context that holds a symbol table and child contexts.
type Context struct {
	ParentId               ContextID                   //holds parent's context id
	ParentCtxSymbolTableId SymbolTableID               //holds symbol table of the parent context from where the current branched out
	Id                     ContextID                   //id of the context
	ImportedContextIds     map[ImportedAlias]ContextID //holds contextds of imported symbols against their alias name in current context
	Prefix                 string
	ContextType_           SymbolsToString
	SymbolTables_          []SymbolTableID // symbol table id
	ChildCtxIds            []ContextID     //holds child context ids
	ResolutionPolicy       ResolutionPolicy
	/*
		     *  lexical_ordered,
			 *  lexical_complete_container
			 *  late_lexical_call_site
			 *  late_lexical_formation_site
			 *  macro_definition_site
			 *  macro_expansion_site
			 *  runtime_bound
			 *  dynamic_call_site
			 *  call-site lexical-context
			 *
	*/
	OwnerSymbolId SymbolID // symbol that owns this context; empty only for structural roots

}

func (c *Context) GetId() ContextID                  { return c.Id }
func (c *Context) GetSymbolTableId() []SymbolTableID { return c.SymbolTables_ }
func (c *Context) GetContextKind() ContextKind       { return ContextKindLexical }
