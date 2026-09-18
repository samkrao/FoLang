package ast

import symboltable "github.com/samkrao/fo-lang/src/context"

// WildcardPattern matches any value without introducing a binding.
type WildcardPattern struct {
	Span
}

func (WildcardPattern) NodeKind() string { return "WildcardPattern" }
func (WildcardPattern) pattern()         {}

// BindingPattern matches any value and introduces Symbol in the enclosing
// match-arm scope.
type BindingPattern struct {
	Span
	Symbol symboltable.SymbolID
}

func (BindingPattern) NodeKind() string { return "BindingPattern" }
func (BindingPattern) pattern()         {}
func (p BindingPattern) def()           {}
func (p BindingPattern) BoundNames() []Name {
	return []Name{{Symbol: p.Symbol}}
}

// LiteralPattern matches a literal value.
type LiteralPattern struct {
	Span
	Value Expr
}

func (LiteralPattern) NodeKind() string { return "LiteralPattern" }
func (LiteralPattern) pattern()         {}

// VariantPattern matches a variant or enum state. Its arguments may contain
// nested BindingPattern values, wildcards, literals, or other patterns.
type VariantPattern struct {
	Span
	State symboltable.SymbolID
	Args  []Pattern
}

func (VariantPattern) NodeKind() string { return "VariantPattern" }
func (VariantPattern) pattern()         {}

func (p VariantPattern) BoundNames() []Name {
	var names []Name
	for _, arg := range p.Args {
		if binder, ok := arg.(Binder); ok {
			names = append(names, binder.BoundNames()...)
		}
	}
	return names
}
