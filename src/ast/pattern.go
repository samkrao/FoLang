package ast

import symboltable "github.com/samkrao/fo-lang/src/context"

type PatternHeader struct {
	Kind_ NodeKind
}
type WildcardPattern struct {
	Span
	PatternHeader
}

func (WildcardPattern) NodeKind() string { return "WildcardPattern" }
func (WildcardPattern) pattern()         {}

type BindingPattern struct {
	Span
	PatternHeader
	Symbol symboltable.SymbolID
}

func (BindingPattern) NodeKind() string { return "BindingPattern" }
func (BindingPattern) pattern()         {}
func (p BindingPattern) BoundSymbols() []symboltable.SymbolID {
	return []symboltable.SymbolID{p.Symbol}
}

type LiteralPattern struct {
	Span
	PatternHeader
	Value Expr
}

func (LiteralPattern) NodeKind() string { return "LiteralPattern" }
func (LiteralPattern) pattern()         {}

type PatternArgument struct {
	Span
	PatternHeader
	Parameter symboltable.SymbolID
	Pattern   Pattern
}

func (PatternArgument) NodeKind() string { return "PatternArgument" }

// ConstructorPattern covers enum states, variants, and data constructors.
type ConstructorPattern struct {
	Span
	PatternHeader
	Constructor symboltable.SymbolID
	Arguments   []PatternArgument
}

func (ConstructorPattern) NodeKind() string { return "ConstructorPattern" }
func (ConstructorPattern) pattern()         {}
func (p ConstructorPattern) BoundSymbols() []symboltable.SymbolID {
	return boundSymbolsFromPatterns(func(yield func(Pattern)) {
		for _, argument := range p.Arguments {
			yield(argument.Pattern)
		}
	})
}

type RecordPatternField struct {
	Span
	PatternHeader
	Field   symboltable.SymbolID
	Pattern Pattern
}

func (RecordPatternField) NodeKind() string { return "RecordPatternField" }

type RecordPattern struct {
	Span
	PatternHeader
	Type   symboltable.SymbolID
	Fields []RecordPatternField
}

func (RecordPattern) NodeKind() string { return "RecordPattern" }
func (RecordPattern) pattern()         {}
func (p RecordPattern) BoundSymbols() []symboltable.SymbolID {
	return boundSymbolsFromPatterns(func(yield func(Pattern)) {
		for _, field := range p.Fields {
			yield(field.Pattern)
		}
	})
}

type TuplePattern struct {
	Span
	PatternHeader
	Elements []Pattern
}

func (TuplePattern) NodeKind() string { return "TuplePattern" }
func (TuplePattern) pattern()         {}
func (p TuplePattern) BoundSymbols() []symboltable.SymbolID {
	return boundSymbolsFromPatterns(func(yield func(Pattern)) {
		for _, element := range p.Elements {
			yield(element)
		}
	})
}

func boundSymbolsFromPatterns(visit func(func(Pattern))) []symboltable.SymbolID {
	var symbols []symboltable.SymbolID
	visit(func(pattern Pattern) {
		if binder, ok := pattern.(Binder); ok {
			symbols = append(symbols, binder.BoundSymbols()...)
		}
	})
	return symbols
}
