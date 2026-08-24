package symboltable

import (
	"strings"
)

// GetVarDetails retrieves symbol info for the given variable name.
func (s *SymbolTable) GetVarDetails(fs FolangSymbols, varName string) SymbolInfo {
	return s.GetDetails(fs, varName, string(S_VarSymbol))
}

// GetFunDetails retrieves symbol info for the given function name.
func (s *SymbolTable) GetFunDetails(fs FolangSymbols, varName string) SymbolInfo {
	return s.GetDetails(fs, varName, string(S_FunctionSymbol))
}

// GetDetails looks up symbol info by name and type, searching parent tables if needed.
func (s *SymbolTable) GetDetails(fs FolangSymbols, varName string, Type_ string) SymbolInfo {
	symbolInfoMap1 := s.Symboldetails
	key := SymbolKey(varName, Type_)

	if Type_ == string(S_FunctionSymbol) {
		for k, v := range symbolInfoMap1 {
			if strings.HasPrefix(k, key) {
				return v
			}
		}
	} else if _, ok := symbolInfoMap1[key]; ok && Type_ != string(S_FunctionSymbol) {
		return symbolInfoMap1[key]
	} else {

		if s.ParentId == "" {
			if fs.GetContext(s.ContextId).ParentId != "" {
				ctx := fs.GetContext(s.ContextId)
				pctx := fs.GetContext(ctx.ParentId)
				return fs.GetSymbolTable(pctx.SymbolTable_).GetDetails(fs, varName, Type_)

			} else {
				return &SymbolDetails{}
			}
		}
		return fs.GetSymbolTable(s.ParentId).GetDetails(fs, varName, Type_)

	}
	return &SymbolDetails{}

}

// ExistsVar reports whether a variable with the given name exists in the symbol table.
func (s SymbolTable) ExistsVar(fs FolangSymbols, varName string) bool {
	return s.Exists(fs, varName, string(S_VarSymbol))
}

// ExistsFun reports whether a function with the given name exists in the symbol table.
func (s SymbolTable) ExistsFun(fs FolangSymbols, varName string) bool {
	return s.Exists(fs, varName, string(S_FunctionSymbol))
}

// Exists reports whether a symbol with the given name and type exists, searching parent tables if needed.
func (s SymbolTable) Exists(fs FolangSymbols, varName string, Type_ string) bool {
	symbolInfoMap1 := s.Symboldetails

	key := SymbolKey(varName, Type_)
	if Type_ == string(S_FunctionSymbol) {
		for k, _ := range symbolInfoMap1 {
			if strings.HasPrefix(k, key) {
				return true
			}
		}
	} else if d, ok := symbolInfoMap1[key]; ok && Type_ != string(S_FunctionSymbol) {
		v := d
		return !v.IsInternal()

	} else {
		if s.ParentId == "" {

			if fs.GetContext(s.ContextId).ParentId != "" {
				ctx := fs.GetContext(s.ContextId)
				pctx := fs.GetContext(ctx.ParentId)
				return fs.GetSymbolTable(pctx.SymbolTable_).Exists(fs, varName, Type_)

			} else {
				return false
			}
		}
		return fs.GetSymbolTable(s.ParentId).Exists(fs, varName, Type_)

	}
	return false
}

func (s SymbolTable) ExistsType(fs FolangSymbols, varName string, Type_ string) bool {
	symbolInfoMap1 := s.Symboldetails

	key := SymbolKey(varName, Type_)
	if Type_ == string(S_FunctionSymbol) {
		for k, _ := range symbolInfoMap1 {
			if strings.HasPrefix(k, key) {
				return true
			}
		}
	} else if d, ok := symbolInfoMap1[key]; ok && Type_ != string(S_FunctionSymbol) {
		v := d
		return !v.IsInternal()

	} else {
		if s.ParentId == "" {

			if fs.GetContext(s.ContextId).ParentId != "" {
				ctx := fs.GetContext(s.ContextId)
				pctx := fs.GetContext(ctx.ParentId)
				return fs.GetSymbolTable(pctx.SymbolTable_).ExistsType(fs, varName, Type_)

			} else {
				return false
			}
		}
		return fs.GetSymbolTable(s.ParentId).ExistsType(fs, varName, Type_)

	}
	return false
}

// IsSymbolNameReusable reports whether the symbol name s is in the restricted reuse list.
func (c *Context) IsSymbolNameReusable(fs FolangSymbols, s string) bool {

	result := false
	for _, v := range c.RestrictedSymbolNameReuse {
		if v == s {
			result = true
			break
		}
	}
	if c.ParentId != "" && !result {
		Parent := fs.GetContext(c.ParentId)
		result = Parent.IsSymbolNameReusable(fs, s)

	}
	return result

}
