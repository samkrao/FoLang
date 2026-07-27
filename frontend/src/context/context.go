// Package symboltable provides symbol table and context management for the fo-lang compiler.
package symboltable

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

type SymbolTable struct {
	Id        string //id of the symbol table
	Prev      string //holds parent's symbol table id
	ContextId string //holds context id of the symbol table
	Prefix    string
	Next      string //Why we need this field when we have parent id?
	//specifically inner functions nd funtion calls are separated by
	// other statements and variable declarrations which were used in function.
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
	ContextType_              string
	SymbolTable_              string   // symbol table id
	ChildCtxIds               []string //holds child context ids
	ResolutionPolicy          string
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

}
