package symboltable

import "sort"

const (
	legacyVariableSymbolType = "Var"
	legacyFunctionSymbolType = "Fun"
)

// GetVarDetails retrieves the nearest visible variable with the given name.
func (s *SymbolTable) GetVarDetails(fs FolangSymbols, varName string) SymbolInfo {
	return s.getDetails(fs, SymbolName(varName), isVariable)
}

// GetFunDetails retrieves the nearest visible function with the given name.
// Overload signatures may make the declaration keys differ, so lookup is based
// on the canonical symbol records rather than a declaration-key prefix.
func (s *SymbolTable) GetFunDetails(fs FolangSymbols, funName string) SymbolInfo {
	return s.getDetails(fs, SymbolName(funName), isFunction)
}

// SymbolKey returns the legacy declaration key used by callers that still
// address a symbol by name and category. New code should prefer the symbol's
// durable SymbolID and the table's SymbolsByName index.
func SymbolKey(name string, symbolType string) SymbolName {
	return SymbolName(name + "_" + symbolType)
}

// GetDetails looks up the nearest symbol with the requested name and symbol
// type. Var and Fun remain accepted for compatibility with serialized tables;
// all other values are matched against SymbolInfo.GetSymbolType.
func (s *SymbolTable) GetDetails(fs FolangSymbols, name string, symbolType string) SymbolInfo {
	match := func(symbol SymbolInfo) bool {
		switch symbolType {
		case legacyVariableSymbolType:
			return isVariable(symbol)
		case legacyFunctionSymbolType:
			return isFunction(symbol)
		default:
			return symbol != nil && symbol.GetSymbolType() == SymbolID(symbolType)
		}
	}
	return s.getDetails(fs, SymbolName(name), match)
}

func (s *SymbolTable) getDetails(fs FolangSymbols, name SymbolName, match func(SymbolInfo) bool) SymbolInfo {
	visited := make(map[*SymbolTable]struct{})
	for table := s; table != nil; table = parentSymbolTable(&fs, table) {
		if _, seen := visited[table]; seen {
			break
		}
		visited[table] = struct{}{}

		for _, symbol := range symbolsNamed(&fs, table, name) {
			if match(symbol) {
				return symbol
			}
		}
	}
	return &SymbolDetails{}
}

// ExistsVar reports whether a non-internal variable with the given name exists.
func (s SymbolTable) ExistsVar(fs FolangSymbols, varName string) bool {
	return s.exists(fs, SymbolName(varName), isVariable)
}

// ExistsFun reports whether a non-internal function with the given name exists.
func (s SymbolTable) ExistsFun(fs FolangSymbols, funName string) bool {
	return s.exists(fs, SymbolName(funName), isFunction)
}

// Exists reports whether a non-internal symbol with the given name and symbol
// type exists in this table or an enclosing table.
func (s SymbolTable) Exists(fs FolangSymbols, name string, symbolType string) bool {
	match := func(symbol SymbolInfo) bool {
		switch symbolType {
		case legacyVariableSymbolType:
			return isVariable(symbol)
		case legacyFunctionSymbolType:
			return isFunction(symbol)
		default:
			return symbol != nil && symbol.GetSymbolType() == SymbolID(symbolType)
		}
	}
	return s.exists(fs, SymbolName(name), match)
}

// ExistsType is retained as a compatibility alias for Exists.
func (s SymbolTable) ExistsType(fs FolangSymbols, name string, symbolType string) bool {
	return s.Exists(fs, name, symbolType)
}

func (s SymbolTable) exists(fs FolangSymbols, name SymbolName, match func(SymbolInfo) bool) bool {
	details := (&s).getDetails(fs, name, match)
	return details.GetSymbolID() != "" && !details.IsInternal()
}

func isVariable(symbol SymbolInfo) bool {
	identifier, ok := symbol.(IIdentifier)
	return ok && identifier.IdentifierType() == "Variable"
}

func isFunction(symbol SymbolInfo) bool {
	_, ok := symbol.(IFunctionShape)
	return ok
}

// symbolsNamed returns all canonical records for a source-level name in stable
// declaration order. SymbolIds is authoritative; the declaration-key index is
// also inspected so partially constructed and deserialized tables remain usable.
func symbolsNamed(fs *FolangSymbols, table *SymbolTable, name SymbolName) []SymbolInfo {
	if fs == nil || table == nil {
		return nil
	}

	seen := make(map[SymbolID]struct{})
	result := make([]SymbolInfo, 0)
	appendSymbol := func(id SymbolID) {
		if id == "" {
			return
		}
		if _, exists := seen[id]; exists {
			return
		}
		seen[id] = struct{}{}
		symbol := fs.GetSymbol(id)
		if symbol != nil && symbol.GetName() == name {
			result = append(result, symbol)
		}
	}

	for _, id := range table.SymbolIds {
		appendSymbol(id)
	}

	keys := make([]string, 0, len(table.SymbolsByName))
	for key := range table.SymbolsByName {
		keys = append(keys, string(key))
	}
	sort.Strings(keys)
	for _, key := range keys {
		for _, id := range table.SymbolsByName[SymbolName(key)] {
			appendSymbol(id)
		}
	}

	return result
}

func parentSymbolTable(fs *FolangSymbols, table *SymbolTable) *SymbolTable {
	if fs == nil || table == nil {
		return nil
	}
	if table.ParentId != "" {
		return fs.GetSymbolTable(table.ParentId)
	}
	ctx := fs.GetContext(table.ContextId)
	if ctx == nil || ctx.ParentCtxSymbolTableId == "" {
		return nil
	}
	return fs.GetSymbolTable(ctx.ParentCtxSymbolTableId)
}
