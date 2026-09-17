// Package ast defines the high-level intermediate representation (HIR) AST node types
// used by the fo-lang frontend parser.
package ast

// SET is the base interface for all AST nodes that can be visited and annotated.
type SET interface {
	NodeKind() string
}

// Stmt is the interface for all statement AST nodes.
type Stmt interface {
	stmt()
	SET
}

// Expr is the interface for all expression AST nodes.
type Expr interface {
	expr()
	SET
}

// Type is the interface for all type AST nodes.
type Type interface {
	_type()
	SET
}

type ValidFuncInnerStmts interface {
	SET
	IsValid() bool
}
