package ast

import (
	"reflect"

	symboltable "github.com/samkrao/fo-lang/src/context"
)

// DummyStmt represents a no-op dummy statement.
type DummyStmt struct {
	Span
	NodeName string
}

func (n DummyStmt) NodeKind() string { return "DummyStmt" }

func (d DummyStmt) stmt() {}

type PragmaStatement struct {
	PragmaSymb symboltable.SymbolID //symboltable.PDADSymbol
}

func (p PragmaStatement) NodeKind() string {
	return reflect.TypeOf(p).Name()
}

func (p PragmaStatement) stmt() {}

type DirectiveStatement struct {
	DirectiveSymb symboltable.SymbolID //symboltable.PDADSymbol
}

func (p DirectiveStatement) stmt() {}

func (p DirectiveStatement) NodeKind() string {
	return reflect.TypeOf(p).Name()
}

type DecoratorStatement struct {
	DecoratorSymb symboltable.SymbolID //symboltable.PDADSymbol
}

func (p DecoratorStatement) stmt() {}

func (p DecoratorStatement) NodeKind() string {
	return reflect.TypeOf(p).Name()
}

type AnnotationStatement struct {
	AnnotationSymb symboltable.SymbolID //symboltable.PDADSymbol
}

func (p AnnotationStatement) NodeKind() string {
	return reflect.TypeOf(p).Name()
}

func (p AnnotationStatement) stmt() {}

type ProjectStatment struct {
	EntryStmt      EntryStatement
	PragmaStmts    []PragmaStatement
	DirectiveStmts []DirectiveStatement
	Sets           []SET
	Symbol         symboltable.SymbolID //symboltable.ProgramSymbol
}

func (p ProjectStatment) stmt() {
}

func (p ProjectStatment) NodeKind() string {
	return reflect.TypeOf(p).Name()
}

type EntryStatement interface {
	Stmt
	Kind() string
}

type ComponentStatement struct {
	Symbol           symboltable.SymbolID //symboltable.ComponentSymbol
	AnnotationStmt   []AnnotationStatement
	IsPackagedExport bool
	ComponentSymb    symboltable.SymbolID //symboltable.ComponentSymbol
}

func (p ComponentStatement) stmt() {}
func (p ComponentStatement) Kind() string {
	if p.IsPackagedExport {
		return "packaged"
	} else {
		return "Library"
	}
}

func (p ComponentStatement) NodeKind() string {
	return reflect.TypeOf(p).Name()
}

type ApplicationStatement struct {
	Symbol symboltable.SymbolID //symboltable.ApplicationSymbol
}

func (p ApplicationStatement) stmt() {}

func (p ApplicationStatement) NodeKind() string {
	return reflect.TypeOf(p).Name()
}

func (p ApplicationStatement) Kind() string {
	return "application"
}

type FunctionDeclarrationStatement struct {
	Symbol         symboltable.SymbolID //symboltable.FunctionSymbol
	AnnotationStmt []AnnotationStatement
	DecoratorStmt  []DecoratorStatement
	Sets           []ValidFuncInnerStmts //statemetns expressions type aliases etc
}

func (p FunctionDeclarrationStatement) NodeKind() string {
	return reflect.TypeOf(p).Name()
}
func (p FunctionDeclarrationStatement) IsValid() bool {
	return true
}

func (p FunctionDeclarrationStatement) stmt() {

}

type CStructStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         symboltable.SymbolID // symboltable.CStructSymbol
}

func (p CStructStatement) NodeKind() string {
	return reflect.TypeOf(p).Name()
}

func (p CStructStatement) stmt() {

}

type StructStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         symboltable.SymbolID //symboltable.StructSymbol
}

func (p StructStatement) NodeKind() string {
	return reflect.TypeOf(p).Name()
}

func (p StructStatement) stmt() {

}

type InterfaceSatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         symboltable.SymbolID //symboltable.InterfaceSymbol
}

func (p InterfaceSatement) NodeKind() string {
	return reflect.TypeOf(p).Name()
}

func (p InterfaceSatement) stmt() {

}

type ClassStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         symboltable.SymbolID //symboltable.ClassSymbol
}

func (p ClassStatement) NodeKind() string {
	return reflect.TypeOf(p).Name()
}

func (p ClassStatement) stmt() {

}

type SignatureStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         symboltable.SymbolID //symboltable.SignatureSymbol
}

func (p SignatureStatement) NodeKind() string {
	return reflect.TypeOf(p).Name()
}

func (p SignatureStatement) stmt() {

}

type ModuleStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         symboltable.SymbolID //symboltable.ModuleSymbol
}

func (p ModuleStatement) NodeKind() string {
	return reflect.TypeOf(p).Name()
}

func (p ModuleStatement) stmt() {

}

type ObjectStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         symboltable.SymbolID //symboltable.ObjectSymbol
}

func (p ObjectStatement) NodeKind() string {
	return reflect.TypeOf(p).Name()
}

func (p ObjectStatement) stmt() {

}

type MixinStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         symboltable.SymbolID //symboltable.MixinSymbol
}

func (p MixinStatement) NodeKind() string {
	return reflect.TypeOf(p).Name()
}

func (p MixinStatement) stmt() {

}

type TraitStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         symboltable.SymbolID //symboltable.TraitSymbol
}

func (p TraitStatement) NodeKind() string {
	return reflect.TypeOf(p).Name()
}

func (p TraitStatement) stmt() {

}

type InstanceStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         symboltable.SymbolID //symboltable.InstanceSymbol
}

func (p InstanceStatement) NodeKind() string {
	return reflect.TypeOf(p).Name()
}

func (p InstanceStatement) stmt() {

}

type TypeClassStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         symboltable.SymbolID //symboltable.TypeClassSymbol
}

func (p TypeClassStatement) NodeKind() string {
	return reflect.TypeOf(p).Name()
}

func (p TypeClassStatement) stmt() {

}

type MatcherStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         symboltable.SymbolID //symboltable.MatcherSymbol
}

func (p MatcherStatement) NodeKind() string {
	return reflect.TypeOf(p).Name()
}

func (p MatcherStatement) stmt() {

}

type IndexerStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         symboltable.SymbolID //symboltable.IndexerSymbol
}

func (p IndexerStatement) NodeKind() string {
	return reflect.TypeOf(p).Name()
}

func (p IndexerStatement) stmt() {

}

type MacroStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         symboltable.SymbolID //symboltable.MacroSymbol
}

func (p MacroStatement) NodeKind() string {
	return reflect.TypeOf(p).Name()
}

func (p MacroStatement) stmt() {

}

type ExtensionStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         symboltable.SymbolID //symboltable.ExtensionSymbol
}

func (p ExtensionStatement) NodeKind() string {
	return reflect.TypeOf(p).Name()
}

func (p ExtensionStatement) stmt() {

}

type ExtensionMethodStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         symboltable.SymbolID //symboltable.ExensionMethodSymbol
}

func (p ExtensionMethodStatement) NodeKind() string {
	return reflect.TypeOf(p).Name()
}

func (p ExtensionMethodStatement) stmt() {

}

type NativeMethodStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         symboltable.SymbolID //symboltable.NativeFunctionSymbol
}

func (p NativeMethodStatement) NodeKind() string {
	return reflect.TypeOf(p).Name()
}

func (p NativeMethodStatement) stmt() {

}

type DelegateStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         symboltable.SymbolID //symboltable.DelegateType
}

func (p DelegateStatement) NodeKind() string {
	return reflect.TypeOf(p).Name()
}

func (p DelegateStatement) stmt() {

}

type ChainedMethodStatement struct {
	AnnotationStmt     []AnnotationStatement
	DecoratorStatement []DecoratorStatement
	Symbol             symboltable.SymbolID //symboltable.SignatureSymbol
}

func (p ChainedMethodStatement) NodeKind() string {
	return reflect.TypeOf(p).Name()
}

func (p ChainedMethodStatement) stmt() {

}

type CurriedFunctionStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         symboltable.SymbolID //symboltable.CurryingFunctionSymbol
}

func (p CurriedFunctionStatement) NodeKind() string {
	return reflect.TypeOf(p).Name()
}

func (p CurriedFunctionStatement) stmt() {

}

type NamedFunctionStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         symboltable.SymbolID //symboltable.NamedParameterFunctionSymbol
}

func (p NamedFunctionStatement) NodeKind() string {
	return reflect.TypeOf(p).Name()
}

func (p NamedFunctionStatement) stmt() {

}

type OptionalParameterFuncStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         symboltable.SymbolID //symboltable.OptionalParameterFunctionSymbol
}

func (p OptionalParameterFuncStatement) NodeKind() string {
	return reflect.TypeOf(p).Name()
}

func (p OptionalParameterFuncStatement) stmt() {

}

type DefaultParameterFuncStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         symboltable.SymbolID //symboltable.DefaultParameterFunctionSymbol
}

func (p DefaultParameterFuncStatement) NodeKind() string {
	return reflect.TypeOf(p).Name()
}

func (p DefaultParameterFuncStatement) stmt() {

}

type VariadicFunctionStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         symboltable.SymbolID //symboltable.VariadicFunctionSymbol
}

func (p VariadicFunctionStatement) NodeKind() string {
	return reflect.TypeOf(p).Name()
}

func (p VariadicFunctionStatement) stmt() {

}

type LetVarStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         symboltable.SymbolID //symboltable.LetVarSymbol
}

func (p LetVarStatement) NodeKind() string {
	return reflect.TypeOf(p).Name()
}

func (p LetVarStatement) stmt() {

}

type LetFuncStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         symboltable.SymbolID //symboltable.LetfunSymbol
}

func (p LetFuncStatement) NodeKind() string {
	return reflect.TypeOf(p).Name()
}

func (p LetFuncStatement) stmt() {

}
