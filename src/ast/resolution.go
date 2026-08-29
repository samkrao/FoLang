package ast

// ResolutionState describes how much semantic identity or type information is
// known for one AST occurrence. It is deliberately not symbol metadata: uses of
// the same declaration may reach different states during one compilation.
type ResolutionState string

const (
	ResolutionUnresolved        ResolutionState = "UNRESOLVED"
	ResolutionResolving                         = "RESOLVING"
	ResolutionPartiallyResolved                 = "PARTIALLY_RESOLVED"
	ResolutionResolved                          = "RESOLVED"
	ResolutionError                             = "ERROR"
)
