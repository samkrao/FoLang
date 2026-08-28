package symboltable

import "strings"

// Declaration binding.
//
// A SymbolTable is both a visibility segment's identity and its CONTENTS: the
// SymbolsByName is what a lookup searches (docs/language-ref.md, B.4). A
// declaration is bound by writing its record into the segment that was active at
// the declaration's own source position, which is the id the record already
// carries in SymbolTableId.
//
// Binding a symbol into the table it was minted in is what makes a name's
// placement fall out of WHERE the parser minted it rather than out of a rule
// repeated at every declaration site. A function's name is minted before its
// context opens, so it binds into the declaring segment; its parameters are
// minted after, so they bind into the function's own.

// SymbolKey builds the SymbolsByName key for a declaration of the given name and
// symbol kind.
//
// The kind is part of the key because one name may be bound once per kind in the
// same segment — a type and a function may share a spelling where the language
// permits it — and because the readers below select on kind as well as on name.
func SymbolKey(name string, symbolType string) string {
	return name + "_" + symbolType
}

// FunctionFamily builds the key prefix shared by every overload of one callable.
//
// The family is the CANONICAL CALLABLE IDENTITY of docs/language-ref.md,
// "Callable Identity and Static Overload Resolution": the owning context, which is
// the table itself; the name; and the callable/receiver category. A class method
// and a static method spelled the same are therefore two families rather than two
// overloads of one.
//
// It extends SymbolKey's `name_Fun` form rather than replacing it, which is what
// lets GetDetails still find a callable by name alone.
func FunctionFamily(name string, category string) string {
	key := SymbolKey(name, string(S_FunctionSymbol))
	if category == "" {
		return key
	}
	return key + "[" + category + "]"
}

// FunctionKey builds the key for one declaration within its family.
//
// Only the PARAMETER signature distinguishes siblings. A return signature never
// participates in overload selection and cannot tell two declarations apart, so it
// is deliberately absent from the key: were it present, two declarations that
// differ only in their result would bind side by side instead of colliding, and
// the reference calls that invalid rather than an overload
// (docs/language-ref.md, "Overload-Family Parameter and Return Rules"). The result
// contract is recorded separately, on the symbol, and checked against the family.
//
// The spellings are the ones WRITTEN at the declaration, because binding happens
// during the parse and no type has been resolved yet. Two spellings that resolve
// to one type are therefore two keys until the semantic pass reconciles them; that
// is a limit of a parse-time family, not a claim about the language's rules.
func FunctionKey(name string, category string, params []string) string {
	return FunctionFamily(name, category) + "(" + strings.Join(params, ",") + ")"
}

// Declare binds info into this segment under key.
//
// It reports whether the binding was made. A key already present is left exactly
// as it was and the record holding it is returned, so that a caller can describe
// the collision — which declaration is the redeclaration and which one owns the
// name — instead of losing one of the two.
func (fs *FolangSymbols) Declare(tableID string, key string, info SymbolInfo) (SymbolInfo, bool) {
	s := fs.GetSymbolTable(tableID)
	if s == nil {
		return nil, false
	}
	if s.SymbolsByName == nil {
		s.SymbolsByName = map[string][]string{}
	}
	if ids := s.SymbolsByName[key]; len(ids) != 0 {
		return fs.GetSymbol(ids[0]), false
	}
	fs.RegisterSymbol(info)
	id := info.GetSymbolID()
	s.SymbolsByName[key] = []string{id}
	s.SymbolIds = append(s.SymbolIds, id)
	return info, true
}

// Undeclare removes a binding. It exists for the parser's speculation rollback,
// which must leave no trace of a branch it threw away; ordinary compilation never
// unbinds a name.
//
// Leaving no trace takes THREE removals, because a declaration is recorded in
// three places: the name index a lookup searches, the declaration order, and the
// registry that holds the record itself. Dropping only the first two would leave
// the record in SymbolsById, where nothing references it and everything reads it —
// the artifact carries the whole registry, so a rolled-back declaration would
// arrive at the backend as a name the program never declared.
//
// Every id under the key is unwound, not just the first. One key holds one id
// today, but SymbolsByName is a slice precisely so that a future overload family
// may bind siblings together, and a partial unwind there would leave the order
// list and the name index disagreeing about what the segment contains.
func (fs *FolangSymbols) Undeclare(tableID string, key string) {
	s := fs.GetSymbolTable(tableID)
	if s == nil {
		return
	}
	ids := s.SymbolsByName[key]
	delete(s.SymbolsByName, key)
	for _, id := range ids {
		for i, candidate := range s.SymbolIds {
			if candidate == id {
				s.SymbolIds = append(s.SymbolIds[:i], s.SymbolIds[i+1:]...)
				break
			}
		}
		fs.UnregisterSymbol(id)
	}
}

// Bindings returns the declaration-key view used by semantic passes. The map is
// a transient view; canonical ownership remains in FolangSymbols.SymbolsById.
func (fs *FolangSymbols) Bindings(tableID string) map[string]SymbolInfo {
	s := fs.GetSymbolTable(tableID)
	if s == nil {
		return nil
	}
	out := make(map[string]SymbolInfo, len(s.SymbolsByName))
	for key, ids := range s.SymbolsByName {
		if len(ids) != 0 {
			out[key] = fs.GetSymbol(ids[0])
		}
	}
	return out
}

// Anchor returns the id of the symbol-table segment that was active where this
// symbol was minted.
//
// It is the use-site visibility anchor B.5 requires: a deferred resolution must
// start here rather than at the owning context's final SymbolTable_, or it would
// see declarations introduced after the symbol's own source position.
func (s SymbolDetails) Anchor() string { return s.SymbolTableId }
