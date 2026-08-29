package ast

// ResolutionState describes how much semantic identity or type information is
// known for one AST occurrence. It is deliberately not symbol metadata: uses of
// the same declaration may reach different states during one compilation.
type ResolutionState string

// The states an occurrence may be in, spelled as the artifact writes them
// (docs/language-ref.md, Appendix B.10).
//
// The explicit type is repeated on every line to keep the public API apparent
// beside each value. Go would also preserve the type if a later ConstSpec omitted
// it, because an empty ConstSpec repeats the complete preceding specification.
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
