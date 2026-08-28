// Package symboltable provides symbol table and context management for the fo-lang compiler.
package symboltable

// ResolutionPolicy is the closed frontend resolver-policy vocabulary defined by
// docs/language-ref.md Appendix B.5. It remains string-backed so serialized AST
// artifacts retain the specified spellings.
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
	SymboltableMap map[string]*SymbolTable
	ContextMap     map[string]*Context
}

func (fs *FolangSymbols) AddSymbolTable(st *SymbolTable) {
	fs.SymboltableMap[st.Id] = st
}
func (fs *FolangSymbols) AddContext(ctx *Context) {
	fs.ContextMap[ctx.Id] = ctx
}
func (fs *FolangSymbols) CreateFolangSymbols() {
	fs.SymboltableMap = make(map[string]*SymbolTable)
	fs.ContextMap = make(map[string]*Context)
}
func (fs *FolangSymbols) GetSymbolTable(id string) *SymbolTable {
	return fs.SymboltableMap[id]
}
func (fs *FolangSymbols) GetContext(id string) *Context {
	return fs.ContextMap[id]
}

// SurfaceSymbols is what a library PUBLISHES, as opposed to what it contains.
//
// A projected library is reached only through its declared surface — its exports,
// and the APIs a dynamicvmrt, system or application library offers — so a consumer
// resolving a name against it must see the surface and nothing behind it. That is
// a different question from what the library itself is made of, which is why this
// is a separate structure rather than a flag on FolangSymbols: the same library
// has both, and handing out the wrong one either hides its API or exposes its
// internals.
//
// It carries no ContextMap. The context is the library, named by the node that
// holds this structure, so there is no context tree to walk: every table here
// belongs to that one surface.
type SurfaceSymbols struct {
	SymboltableMap map[string]*SymbolTable
}

// CreateSurfaceSymbols initialises the map, as CreateFolangSymbols does for the
// complete model.
func (ss *SurfaceSymbols) CreateSurfaceSymbols() {
	ss.SymboltableMap = make(map[string]*SymbolTable)
}

// AddSymbolTable publishes one table on the surface.
func (ss *SurfaceSymbols) AddSymbolTable(st *SymbolTable) {
	if ss.SymboltableMap == nil {
		ss.CreateSurfaceSymbols()
	}
	ss.SymboltableMap[st.Id] = st
}

// GetSymbolTable returns a published table by id.
func (ss *SurfaceSymbols) GetSymbolTable(id string) *SymbolTable {
	return ss.SymboltableMap[id]
}

type SymbolTable struct {
	Id string // id of the symbol table
	// ParentId is the preceding declaration-order visibility segment in the SAME
	// context, and is empty for a context's first segment (docs/language-ref.md,
	// B.4). Lookup walks it from the newest segment toward the oldest, which is
	// what makes a forward link unnecessary: a Context records its active segment
	// in SymbolTable_, and every earlier one is reachable from there.
	ParentId  string
	ContextId string // holds context id of the symbol table
	Prefix    string

	Symboldetails map[string]SymbolInfo
}

// Context represents a scoping context that holds a symbol table and child contexts.
type Context struct {
	ParentId                  string //holds parent's context id
	ParentCtxSymbolTableId    string //holds symbol table of the parent context from where the current branched out
	Id                        string //id of the context
	RestrictedSymbolNameReuse []string
	ImportedContextIds        map[string]string //holds contextds of imported symbols against their alias name in current context
	Prefix                    string
	ContextType_              SymbolsToString
	SymbolTable_              string   // symbol table id
	ChildCtxIds               []string //holds child context ids
	ResolutionPolicy          ResolutionPolicy
	/*
		     *  lexical_ordered,
			 *  lexical_complete_container
			 *  late_lexical_call_site
			 *  late_lexical_formation_site
			 *  macro_definition_site
			 *  macro_expansion_site
			 *  runtime_bound
			 *  dynamic_call_site
			 *
			 *
	*/
	OwnerSymbolId string // symbol that owns this context; empty only for structural roots

}
