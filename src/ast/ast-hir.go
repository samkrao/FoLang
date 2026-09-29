// Package ast defines the resolved high-level intermediate representation
// produced by the FoLang frontend. Nodes retain source structure and spans;
// canonical semantic identity and classification live in context symbols.
package ast

import symboltable "github.com/samkrao/fo-lang/src/context"

type NodeKind string

// SET is the common interface for every AST/HIR node.
type SET interface {
	Spanned
	NodeKind() string
}

// Def is implemented by declarations that introduce a symbol.
type Def interface {
	SET
	def()
}

// Stmt is implemented by executable or control-flow instructions.
type Stmt interface {
	SET
	stmt()
}

// Expr is implemented by constructs that produce a value or effect.
type Expr interface {
	SET
	expr()
}

// Type is implemented by source type expressions. Symbol resolves to the
// canonical semantic type record once resolution completes.
type Type interface {
	SET
	_type()
}

// Pattern is implemented by match and destructuring patterns.
type Pattern interface {
	SET
	pattern()
}

// Binder is implemented by patterns that introduce symbols.
type Binder interface {
	BoundSymbols() []symboltable.SymbolID
}

// MetadataRef preserves the occurrence span while referring to the canonical
// MetaDataApplication record in FolangSymbols.SymbolsById.
type MetadataRef struct {
	Span
	Symbol symboltable.SymbolID
}

func (MetadataRef) NodeKind() string { return "MetadataRef" }

// Block retains ordered body items and the optional unterminated tail
// expression that supplies the block's value.
type Block struct {
	Span
	Items  []SET
	Result Expr
}

func (Block) NodeKind() string { return "Block" }

// SourceFile is the serialization/parser root. Directives and pragmas attach
// here rather than to the file's primary declaration.
type SourceFile struct {
	Span
	Filename string
	Metadata []MetadataRef
	Items    []SET
}

func (SourceFile) NodeKind() string { return "SourceFile" }
