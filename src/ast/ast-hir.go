// Package ast defines the high-level intermediate representation (HIR) AST node types
// used by the fo-lang frontend parser.
package ast

import symboltable "github.com/samkrao/fo-lang/src/context"

// SET is the base interface for all AST nodes that can be visited and annotated.
type SET interface {
	NodeKind() string
}

// Def is implemented by every declaration that introduces a symbol.
// Definitions include variables, functions, types, components, and metadata
// declarations. A definition is not automatically an executable statement.
type Def interface {
	SET
	def()
}

// Stmt is implemented by executable or control-flow instructions.
type Stmt interface {
	SET
	stmt()
}

// Expr is implemented by constructs that evaluate a value or produce an effect
// as part of a larger expression or statement.
type Expr interface {
	SET
	expr()
}

// Type is implemented by constructs that describe the shape of a value.
type Type interface {
	SET
	_type()
}

// Pattern is implemented by nodes used to test or destructure a value in a
// match case. Patterns are neither executable statements nor value expressions.
type Pattern interface {
	SET
	pattern()
}

// Name is the source-level identity of a name introduced by a pattern. The
// canonical semantic record is identified by Symbol.
type Name struct {
	Symbol symboltable.SymbolID
}

// Binder is an optional capability implemented by patterns that can introduce
// names into the scope of a match arm. It is not an AST node category.
type Binder interface {
	BoundNames() []Name
}

// FunctionBody is the ordered collection of definitions and statements in a
// callable or direct block. SET is intentional: definitions and statements are
// separate semantic categories but may occur together in a body.
type FunctionBody []SET

// Entry is a source-level entry definition such as an application or library.
type Entry interface {
	Def
	Kind() string
}
