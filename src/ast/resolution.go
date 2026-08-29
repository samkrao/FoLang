package ast

// ResolutionState describes how much semantic identity or type information is
// known for one AST occurrence. It is deliberately not symbol metadata: uses of
// the same declaration may reach different states during one compilation.
type ResolutionState string

// The states an occurrence may be in, spelled as the artifact writes them
// (docs/language-ref.md, Appendix B.10).
//
// Each constant repeats the explicit type because each also supplies its own
// expression. Go inherits the preceding expression list and its type only when
// a later ConstSpec omits the expression list; writing `= "RESOLVING"` without
// a type instead declares an untyped string constant.
const (
	// ResolutionUnresolved is an occurrence whose identity or type is not yet known.
	ResolutionUnresolved ResolutionState = "UNRESOLVED"
	// ResolutionResolving marks an occurrence a later pass is working through,
	// which is how that pass detects a cycle rather than recursing on it.
	ResolutionResolving ResolutionState = "RESOLVING"
	// ResolutionPartiallyResolved is an occurrence whose spelling and lookup
	// domain are known while canonical lookup remains.
	ResolutionPartiallyResolved ResolutionState = "PARTIALLY_RESOLVED"
	// ResolutionResolved is an occurrence needing nothing further.
	ResolutionResolved ResolutionState = "RESOLVED"
	// ResolutionError is an occurrence a pass failed to resolve, kept so the
	// failure travels with the tree instead of only in a diagnostic.
	ResolutionError ResolutionState = "ERROR"
)
