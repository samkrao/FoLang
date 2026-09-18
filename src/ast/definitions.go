package ast

import symboltable "github.com/samkrao/fo-lang/src/context"

// DefinitionHeader contains the only semantic identity an AST definition
// needs. The canonical symbol record lives in the context package.
type DefinitionHeader struct {
	Span
	Symbol   symboltable.SymbolID
	Metadata []SET
}

// VariableDefinition introduces a binding such as x co.int or x := value.
type VariableDefinition struct {
	DefinitionHeader
	DeclaredType Type
	Initializer  Expr
}

func (VariableDefinition) NodeKind() string { return "VariableDefinition" }
func (VariableDefinition) def()             {}

// LetVariableDefinition introduces an immutable or let-style binding.
type LetVariableDefinition struct {
	DefinitionHeader
	DeclaredType Type
	Initializer  Expr
}

func (LetVariableDefinition) NodeKind() string { return "LetVariableDefinition" }
func (LetVariableDefinition) def()             {}

// TypeDefinition introduces a named type or type alias.
type TypeDefinition struct {
	DefinitionHeader
	Underlying Type
}

func (TypeDefinition) NodeKind() string { return "TypeDefinition" }
func (TypeDefinition) def()             {}

// AssociatedTypeDefinition represents a signature requirement or matching
// module binding such as T co.associatedType or T co.associatedType = co.int.
// The symbol identifies the canonical context.AssociatedType record; the
// optional Binding is the concrete type supplied by a matching module.
type AssociatedTypeDefinition struct {
	DefinitionHeader
	Binding Type
}

func (AssociatedTypeDefinition) NodeKind() string { return "AssociatedTypeDefinition" }
func (AssociatedTypeDefinition) def()             {}

// TypeComponentDefinition represents a required or supplied co.type alias
// component such as Stack co.type or Stack co.type = co.List(T).
type TypeComponentDefinition struct {
	DefinitionHeader
	Underlying Type
}

func (TypeComponentDefinition) NodeKind() string { return "TypeComponentDefinition" }
func (TypeComponentDefinition) def()             {}

// RefinementTypeDefinition introduces a named type constrained by a
// predicate, for example Positive co.refinementType = (co.int).where(...).
type RefinementTypeDefinition struct {
	DefinitionHeader
	Base      Type
	Predicate Expr
}

func (RefinementTypeDefinition) NodeKind() string { return "RefinementTypeDefinition" }
func (RefinementTypeDefinition) def()             {}

// PredicateTypeDefinition introduces a named predicate type. Its predicate
// operates on type values rather than ordinary runtime values.
type PredicateTypeDefinition struct {
	DefinitionHeader
	Base      Type
	Predicate Expr
}

func (PredicateTypeDefinition) NodeKind() string { return "PredicateTypeDefinition" }
func (PredicateTypeDefinition) def()             {}

// DependentTypeDefinition introduces a named value-indexed type object.
type DependentTypeDefinition struct {
	DefinitionHeader
	Parameters []VariableDefinition
	Underlying Type
}

func (DependentTypeDefinition) NodeKind() string { return "DependentTypeDefinition" }
func (DependentTypeDefinition) def()             {}

// ParameterizedTypeDefinition introduces a type constructor such as
// Option(T) co.type = co.variants(Some(T), None).
type ParameterizedTypeDefinition struct {
	DefinitionHeader
	Parameters []symboltable.SymbolID
	Underlying Type
}

func (ParameterizedTypeDefinition) NodeKind() string {
	return "ParameterizedTypeDefinition"
}
func (ParameterizedTypeDefinition) def() {}

// VariantTypeDefinition introduces the enclosing type and owns its states.
type VariantTypeDefinition struct {
	DefinitionHeader
	Parameters []symboltable.SymbolID
	States     []VariantStateDefinition
}

func (VariantTypeDefinition) NodeKind() string { return "VariantTypeDefinition" }
func (VariantTypeDefinition) def()             {}

// VariantStateDefinition declares a state owned by a variant type. A state is
// not an independent type; parameterized states are compiler-provided state
// functions and zero-parameter states are state values.
type VariantStateDefinition struct {
	DefinitionHeader
	Parameters []Type
}

func (VariantStateDefinition) NodeKind() string { return "VariantStateDefinition" }
func (VariantStateDefinition) def()             {}

// EnumDefinition introduces a closed tagged type whose body consists only of
// enum states.
type EnumDefinition struct {
	DefinitionHeader
	States []EnumStateDefinition
}

func (EnumDefinition) NodeKind() string { return "EnumDefinition" }
func (EnumDefinition) def()             {}

type EnumStateDefinition struct {
	DefinitionHeader
	Parameters []VariableDefinition
}

func (EnumStateDefinition) NodeKind() string { return "EnumStateDefinition" }
func (EnumStateDefinition) def()             {}

// FunctionDefinition introduces a callable symbol and its body.
type FunctionDefinition struct {
	DefinitionHeader
	Parameters []VariableDefinition
	Signature  Type
	Body       FunctionBody
}

func (FunctionDefinition) NodeKind() string { return "FunctionDefinition" }
func (FunctionDefinition) def()             {}

// StructDefinition introduces a struct-like type.
type StructDefinition struct {
	DefinitionHeader
	Fields []VariableDefinition
	Body   FunctionBody
}

func (StructDefinition) NodeKind() string { return "StructDefinition" }
func (StructDefinition) def()             {}

// CStructDefinition introduces a native-layout struct-like type.
type CStructDefinition struct {
	DefinitionHeader
	Fields []VariableDefinition
}

func (CStructDefinition) NodeKind() string { return "CStructDefinition" }
func (CStructDefinition) def()             {}

// ClassDefinition introduces a class-like type.
type ClassDefinition struct {
	DefinitionHeader
	Fields []VariableDefinition
	Body   FunctionBody
}

func (ClassDefinition) NodeKind() string { return "ClassDefinition" }
func (ClassDefinition) def()             {}

// InterfaceDefinition introduces an interface contract.
type InterfaceDefinition struct {
	DefinitionHeader
	Members []Def
}

func (InterfaceDefinition) NodeKind() string { return "InterfaceDefinition" }
func (InterfaceDefinition) def()             {}

// ModuleDefinition introduces a module namespace.
type ModuleDefinition struct {
	DefinitionHeader
	Body FunctionBody
}

func (ModuleDefinition) NodeKind() string { return "ModuleDefinition" }
func (ModuleDefinition) def()             {}

// SignatureDefinition introduces a declarative callable/module contract.
type SignatureDefinition struct {
	DefinitionHeader
	Members []Def
}

func (SignatureDefinition) NodeKind() string { return "SignatureDefinition" }
func (SignatureDefinition) def()             {}

// ObjectDefinition, MixinDefinition, TraitDefinition, and InstanceDefinition
// are symbol-introducing type/relationship definitions.
type ObjectDefinition struct {
	DefinitionHeader
	Body FunctionBody
}

func (ObjectDefinition) NodeKind() string { return "ObjectDefinition" }
func (ObjectDefinition) def()             {}

type MixinDefinition struct {
	DefinitionHeader
	Body FunctionBody
}

func (MixinDefinition) NodeKind() string { return "MixinDefinition" }
func (MixinDefinition) def()             {}

type TraitDefinition struct {
	DefinitionHeader
	Members []Def
}

func (TraitDefinition) NodeKind() string { return "TraitDefinition" }
func (TraitDefinition) def()             {}

type InstanceDefinition struct {
	DefinitionHeader
	Body FunctionBody
}

func (InstanceDefinition) NodeKind() string { return "InstanceDefinition" }
func (InstanceDefinition) def()             {}

type TypeClassDefinition struct {
	DefinitionHeader
	Members []Def
}

func (TypeClassDefinition) NodeKind() string { return "TypeClassDefinition" }
func (TypeClassDefinition) def()             {}

// MatcherDefinition, IndexerDefinition, and MacroDefinition introduce
// callable or compile-time symbols.
type MatcherDefinition struct {
	DefinitionHeader
	Body FunctionBody
}

func (MatcherDefinition) NodeKind() string { return "MatcherDefinition" }
func (MatcherDefinition) def()             {}

type IndexerDefinition struct {
	DefinitionHeader
	Signature Type
	Body      FunctionBody
}

func (IndexerDefinition) NodeKind() string { return "IndexerDefinition" }
func (IndexerDefinition) def()             {}

type MacroDefinition struct {
	DefinitionHeader
	Body FunctionBody
}

func (MacroDefinition) NodeKind() string { return "MacroDefinition" }
func (MacroDefinition) def()             {}

type ExtensionDefinition struct {
	DefinitionHeader
	Body FunctionBody
}

func (ExtensionDefinition) NodeKind() string { return "ExtensionDefinition" }
func (ExtensionDefinition) def()             {}

type ExtensionMethodDefinition struct {
	DefinitionHeader
	Signature Type
	Body      FunctionBody
}

func (ExtensionMethodDefinition) NodeKind() string { return "ExtensionMethodDefinition" }
func (ExtensionMethodDefinition) def()             {}

type NativeMethodDefinition struct {
	DefinitionHeader
	Signature Type
}

func (NativeMethodDefinition) NodeKind() string { return "NativeMethodDefinition" }
func (NativeMethodDefinition) def()             {}

type DelegateDefinition struct {
	DefinitionHeader
	Signature Type
}

func (DelegateDefinition) NodeKind() string { return "DelegateDefinition" }
func (DelegateDefinition) def()             {}

// These definitions represent callable parameter/result variants. Their
// differences belong in the resolved symbol, so the AST only carries shape.
type ChainedMethodDefinition struct {
	DefinitionHeader
	Signature Type
	Body      FunctionBody
}

func (ChainedMethodDefinition) NodeKind() string { return "ChainedMethodDefinition" }
func (ChainedMethodDefinition) def()             {}

type CurriedFunctionDefinition struct {
	DefinitionHeader
	Signature Type
	Body      FunctionBody
}

func (CurriedFunctionDefinition) NodeKind() string { return "CurriedFunctionDefinition" }
func (CurriedFunctionDefinition) def()             {}

type NamedFunctionDefinition struct {
	DefinitionHeader
	Signature Type
	Body      FunctionBody
}

func (NamedFunctionDefinition) NodeKind() string { return "NamedFunctionDefinition" }
func (NamedFunctionDefinition) def()             {}

type OptionalParameterFunctionDefinition struct {
	DefinitionHeader
	Signature Type
	Body      FunctionBody
}

func (OptionalParameterFunctionDefinition) NodeKind() string {
	return "OptionalParameterFunctionDefinition"
}
func (OptionalParameterFunctionDefinition) def() {}

type DefaultParameterFunctionDefinition struct {
	DefinitionHeader
	Signature Type
	Body      FunctionBody
}

func (DefaultParameterFunctionDefinition) NodeKind() string {
	return "DefaultParameterFunctionDefinition"
}
func (DefaultParameterFunctionDefinition) def() {}

type VariadicFunctionDefinition struct {
	DefinitionHeader
	Signature Type
	Body      FunctionBody
}

func (VariadicFunctionDefinition) NodeKind() string { return "VariadicFunctionDefinition" }
func (VariadicFunctionDefinition) def()             {}

type LetFunctionDefinition struct {
	DefinitionHeader
	Signature Type
	Body      FunctionBody
}

func (LetFunctionDefinition) NodeKind() string { return "LetFunctionDefinition" }
func (LetFunctionDefinition) def()             {}

func (d ComponentDefinition) Kind() string {
	if d.Exported {
		return "packaged"
	}
	return "library"
}

// ComponentDefinition introduces a library or application component.
type ComponentDefinition struct {
	DefinitionHeader
	Exported bool
	Body     FunctionBody
}

func (ComponentDefinition) NodeKind() string { return "ComponentDefinition" }
func (ComponentDefinition) def()             {}

// ApplicationDefinition is an executable entry definition.
type ApplicationDefinition struct {
	DefinitionHeader
	Body FunctionBody
}

func (ApplicationDefinition) NodeKind() string { return "ApplicationDefinition" }
func (ApplicationDefinition) def()             {}
func (ApplicationDefinition) Kind() string     { return "application" }

// LibraryDefinition is a library entry definition.
type LibraryDefinition struct {
	DefinitionHeader
	Exported bool
	Body     FunctionBody
}

func (LibraryDefinition) NodeKind() string { return "LibraryDefinition" }
func (LibraryDefinition) def()             {}
func (LibraryDefinition) Kind() string     { return "library" }

// ProjectDefinition is the root definition for a source project.
type ProjectDefinition struct {
	DefinitionHeader
	Entry       Entry
	Definitions []Def
}

func (ProjectDefinition) NodeKind() string { return "ProjectDefinition" }
func (ProjectDefinition) def()             {}

// Metadata nodes are attached to definitions and are not executable
// statements. Their referenced symbols are resolved in the context registry.
type Annotation struct {
	Span
	Symbol symboltable.SymbolID
}

func (Annotation) NodeKind() string { return "Annotation" }

type Decorator struct {
	Span
	Symbol symboltable.SymbolID
}

func (Decorator) NodeKind() string { return "Decorator" }

type Directive struct {
	Span
	Symbol symboltable.SymbolID
}

func (Directive) NodeKind() string { return "Directive" }

type Pragma struct {
	Span
	Symbol symboltable.SymbolID
}

func (Pragma) NodeKind() string { return "Pragma" }

// Compatibility aliases for names used by the in-progress rewrite.
type VariableDeclarationStatement = VariableDefinition
type StructStatement = StructDefinition
type CStructStatement = CStructDefinition
type ClassStatement = ClassDefinition
type FunctionDeclarrationStatement = FunctionDefinition
type ProjectStatment = ProjectDefinition
type ApplicationStatement = ApplicationDefinition
