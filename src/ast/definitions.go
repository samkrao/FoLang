package ast

import symboltable "github.com/samkrao/fo-lang/src/context"

// DefinitionHeader contains the source span, canonical symbol identity, and
// metadata attached to any symbol-introducing definition.
type DefinitionHeader struct {
	Span
	Symbol   symboltable.SymbolID
	Metadata []SET
}

// DefinitionKind records source syntax. The resolved context symbol remains
// authoritative for richer semantic distinctions and validation.
type DefinitionKind string

const (
	DefinitionVariable       DefinitionKind = "variable"
	DefinitionLetVariable    DefinitionKind = "let_variable"
	DefinitionFunction       DefinitionKind = "function"
	DefinitionLetFunction    DefinitionKind = "let_function"
	DefinitionType           DefinitionKind = "type"
	DefinitionTypeComponent  DefinitionKind = "type_component"
	DefinitionAssociatedType DefinitionKind = "associated_type"
	DefinitionStruct         DefinitionKind = "struct"
	DefinitionCStruct        DefinitionKind = "cstruct"
	DefinitionClass          DefinitionKind = "class"
	DefinitionInterface      DefinitionKind = "interface"
	DefinitionModule         DefinitionKind = "module"
	DefinitionSignature      DefinitionKind = "signature"
	DefinitionObject         DefinitionKind = "object"
	DefinitionMixin          DefinitionKind = "mixin"
	DefinitionTrait          DefinitionKind = "trait"
	DefinitionInstance       DefinitionKind = "instance"
	DefinitionTypeClass      DefinitionKind = "typeclass"
	DefinitionMatcher        DefinitionKind = "matcher"
	DefinitionIndexer        DefinitionKind = "indexer"
	DefinitionMacro          DefinitionKind = "macro"
	DefinitionExtension      DefinitionKind = "extension"
	DefinitionNativeMethod   DefinitionKind = "native_method"
	DefinitionDelegate       DefinitionKind = "delegate"
	DefinitionVariant        DefinitionKind = "variant"
	DefinitionVariantState   DefinitionKind = "variant_state"
	DefinitionEnum           DefinitionKind = "enum"
	DefinitionEnumState      DefinitionKind = "enum_state"
	DefinitionApplication    DefinitionKind = "application"
	DefinitionLibrary        DefinitionKind = "library"
	DefinitionProject        DefinitionKind = "project"
)

// FunctionShapeDefinition represents callable declarations. The context
// symbol carries the detailed callable shape; this node keeps only callable
// syntax shared by functions, methods, macros, operators, and indexers.
type FunctionShapeDefinition struct {
	DefinitionHeader
	Kind DefinitionKind

	Signature         Type
	Parameters        []Def
	Results           []Type
	GenericParameters []Def
	Body              Block
}

func (FunctionShapeDefinition) NodeKind() string { return "FunctionShapeDefinition" }
func (FunctionShapeDefinition) def()             {}

// Definition represents non-callable symbol-introducing forms. Optional
// fields are populated according to Kind; semantic meaning is resolved through
// Symbol in the context package rather than by creating a node per symbol kind.
type Definition struct {
	DefinitionHeader
	Kind DefinitionKind

	GenericParameters []Def
	DeclaredType      Type
	Underlying        Type
	Binding           Type
	Base              Type
	Predicate         Expr
	Value             Expr

	Fields  []Def
	Members []Def
	States  []Def
	Body    Block

	Exported bool
}

func (Definition) NodeKind() string { return "Definition" }
func (Definition) def()             {}

// EntryKind implements Entry without duplicating entry-specific definition
// node types.
func (d Definition) EntryKind() string { return string(d.Kind) }
