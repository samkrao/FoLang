package ast

import symboltable "github.com/samkrao/fo-lang/src/context"

// DefinitionHeader is shared source structure for symbol-introducing nodes.
// The concrete record in FolangSymbols determines the semantic declaration
// kind; the AST does not repeat that classification.
type DefinitionHeader struct {
	Span
	Symbol   symboltable.SymbolID
	Metadata []MetadataRef
	Kind_    NodeKind
}

// Declaration represents a declaration whose resolved information is fully
// carried by its symbol, such as a bodyless specification.
type Declaration struct {
	DefinitionHeader
}

func (Declaration) NodeKind() string { return "Declaration" }
func (Declaration) def()             {}

// ValueDefinition represents variables, fields, enum states, and other
// declarations that may have an initializer.
type ValueDefinition struct {
	DefinitionHeader
	Initializer Expr
}

func (ValueDefinition) NodeKind() string { return "ValueDefinition" }
func (ValueDefinition) def()             {}

// TypeDefinition retains the source RHS of a named type declaration. The
// symbol identifies whether it is an alias, newtype, refinement, dependent,
// parameterized, or another semantic type category.
type TypeDefinition struct {
	DefinitionHeader
	Value Type
}

func (TypeDefinition) NodeKind() string { return "TypeDefinition" }
func (TypeDefinition) def()             {}

// ContextDefinition represents a declaration with a member body, including
// structs, classes, modules, objects, components, and related kinds.
type ContextDefinition struct {
	DefinitionHeader
	Body Block
}

func (ContextDefinition) NodeKind() string { return "ContextDefinition" }
func (ContextDefinition) def()             {}

// ParameterDefault keeps executable default-value syntax outside the symbol
// record while linking it to the canonical parameter symbol.
type ParameterDefault struct {
	Span
	Parameter symboltable.SymbolID
	Value     Expr
}

func (ParameterDefault) NodeKind() string { return "ParameterDefault" }

// CallableDefinition represents every named callable shape. Signature,
// parameters, results, and callable classification live on its function-shape
// symbol. A nil Body denotes a forward/bodyless declaration.
type CallableDefinition struct {
	DefinitionHeader
	Defaults []ParameterDefault
	Body     *Block
}

func (CallableDefinition) NodeKind() string { return "CallableDefinition" }
func (CallableDefinition) def()             {}
