package ast

import (
	symboltable "github.com/samkrao/fo-lang/src/context"
)

// DummyStmt represents a no-op dummy statement.
type DummyStmt struct {
	Span
	NodeName string
}

func (n DummyStmt) GetName() string { return "DummyStmt" }

func (d DummyStmt) stmt() {}

type PragmaStatement struct {
	PragmaSymb symboltable.PDADSymbol
}

func (p PragmaStatement) GetName() string {
	return p.PragmaSymb.Name_
}

func (p PragmaStatement) stmt() {}

type DirectiveStatement struct {
	DirectiveSymb symboltable.PDADSymbol
}

func (p DirectiveStatement) stmt() {}

func (p DirectiveStatement) GetName() string {
	return p.DirectiveSymb.Name_
}

type DecoratorStatement struct {
	DecoratorSymb symboltable.PDADSymbol
}

func (p DecoratorStatement) stmt() {}

func (p DecoratorStatement) GetName() string {
	return p.DecoratorSymb.Name_
}

type AnnotationStatement struct {
	AnnotationSymb symboltable.PDADSymbol
}

func (p AnnotationStatement) GetName() string {
	return p.AnnotationSymb.Name_
}

func (p AnnotationStatement) stmt() {}

type ProjectStatment struct {
	EntryStmt      EntryStatement
	PragmaStmts    []PragmaStatement
	DirectiveStmts []DirectiveStatement
	Sets           []SET
}

func (p ProjectStatment) stmt() {
}

type EntryStatement interface {
	Stmt
	Kind() string
}

type ComponentStatement struct {
	Symbol           symboltable.ComponentSymbol
	AnnotationSymb   []symboltable.PDADSymbol
	IsPackagedExport bool
	ComponentSymb    symboltable.ComponentSymbol
}

func (p ComponentStatement) stmt() {}
func (p ComponentStatement) Kind() string {
	if p.IsPackagedExport {
		return "packaged"
	} else {
		return "Library"
	}
}

func (p ComponentStatement) GetName() string {
	return p.Symbol.Name_
}

type ApplicationStatement struct {
	Symbol symboltable.ApplicationSymbol
}

func (p ApplicationStatement) stmt() {}

func (p ApplicationStatement) GetName() string {
	return p.Symbol.Name_
}

func (p ApplicationStatement) Kind() string {
	return "application"
}

type FunctionDeclarrationStatement struct {
	Symbol         symboltable.FunctionSymbol
	AnnotationStmt []AnnotationStatement
	DecoratorStmt  []DecoratorStatement
	Sets           []ValidFuncInnerStmts //statemetns expressions type aliases etc
}

func (p FunctionDeclarrationStatement) GetName() string {
	return p.Symbol.Name_
}
func (p FunctionDeclarrationStatement) IsValid() bool {
	return true
}

func (p FunctionDeclarrationStatement) Stmt() {

}

type CStructStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         symboltable.CStructSymbol
}

func (p CStructStatement) GetName() string {
	return p.Symbol.Name_
}

func (p CStructStatement) Stmt() {

}

type StructStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         symboltable.StructSymbol
}

func (p StructStatement) GetName() string {
	return p.Symbol.Name_
}

func (p StructStatement) Stmt() {

}

type InterfaceSatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         symboltable.InterfaceSymbol
}

func (p InterfaceSatement) GetName() string {
	return p.Symbol.Name_
}

func (p InterfaceSatement) Stmt() {

}

type ClassStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         symboltable.ClassSymbol
}

func (p ClassStatement) GetName() string {
	return p.Symbol.Name_
}

func (p ClassStatement) Stmt() {

}

type SignatureStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         symboltable.SignatureSymbol
}

func (p SignatureStatement) GetName() string {
	return p.Symbol.Name_
}

func (p SignatureStatement) Stmt() {

}

type ModuleStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         symboltable.ModuleSymbol
}

func (p ModuleStatement) GetName() string {
	return p.Symbol.Name_
}

func (p ModuleStatement) Stmt() {

}

type ObjectStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         symboltable.ObjectSymbol
}

func (p ObjectStatement) GetName() string {
	return p.Symbol.Name_
}

func (p ObjectStatement) Stmt() {

}

type MixinStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         symboltable.MixinSymbol
}

func (p MixinStatement) GetName() string {
	return p.Symbol.Name_
}

func (p MixinStatement) Stmt() {

}

type TraitStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         symboltable.TraitSymbol
}

func (p TraitStatement) GetName() string {
	return p.Symbol.Name_
}

func (p TraitStatement) Stmt() {

}

type InstanceStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         symboltable.InstanceSymbol
}

func (p InstanceStatement) GetName() string {
	return p.Symbol.Name_
}

func (p InstanceStatement) Stmt() {

}

type TypeClassStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         symboltable.TypeClassSymbol
}

func (p TypeClassStatement) GetName() string {
	return p.Symbol.Name_
}

func (p TypeClassStatement) Stmt() {

}

type MatcherStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         symboltable.MatcherSymbol
}

func (p MatcherStatement) GetName() string {
	return p.Symbol.Name_
}

func (p MatcherStatement) Stmt() {

}

type IndexerStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         symboltable.IndexerSymbol
}

func (p IndexerStatement) GetName() string {
	return p.Symbol.Name_
}

func (p IndexerStatement) Stmt() {

}

type MacroStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         symboltable.MacroSymbol
}

func (p MacroStatement) GetName() string {
	return p.Symbol.Name_
}

func (p MacroStatement) Stmt() {

}

type ExtensionStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         symboltable.ExtensionSymbol
}

func (p ExtensionStatement) GetName() string {
	return p.Symbol.Name_
}

func (p ExtensionStatement) Stmt() {

}

type ExtensionMethodStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         symboltable.ExensionMethodSymbol
}

func (p ExtensionMethodStatement) GetName() string {
	return p.Symbol.Name_
}

func (p ExtensionMethodStatement) Stmt() {

}

type NativeMethodStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         symboltable.NativeFunctionSymbol
}

func (p NativeMethodStatement) GetName() string {
	return p.Symbol.Name_
}

func (p NativeMethodStatement) Stmt() {

}

type DelegateStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         symboltable.DelegateType
}

func (p DelegateStatement) GetName() string {
	return p.Symbol.Name_
}

func (p DelegateStatement) Stmt() {

}

type ChainedMethodStatement struct {
	AnnotationStmt     []AnnotationStatement
	DecoratorStatement []DecoratorStatement
	Symbol             symboltable.SignatureSymbol
}

func (p ChainedMethodStatement) GetName() string {
	return p.Symbol.Name_
}

func (p ChainedMethodStatement) Stmt() {

}

type CurriedFunctionStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         symboltable.CurryingFunctionSymbol
}

func (p CurriedFunctionStatement) GetName() string {
	return p.Symbol.Name_
}

func (p CurriedFunctionStatement) Stmt() {

}

type NamedFunctionStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         symboltable.NamedParameterFunctionSymbol
}

func (p NamedFunctionStatement) GetName() string {
	return p.Symbol.Name_
}

func (p NamedFunctionStatement) Stmt() {

}

type OptionalParameterFuncStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         symboltable.OptionalParameterFunctionSymbol
}

func (p OptionalParameterFuncStatement) GetName() string {
	return p.Symbol.Name_
}

func (p OptionalParameterFuncStatement) Stmt() {

}

type DefaultParameterFuncStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         symboltable.DefaultParameterFunctionSymbol
}

func (p DefaultParameterFuncStatement) GetName() string {
	return p.Symbol.Name_
}

func (p DefaultParameterFuncStatement) Stmt() {

}

type VariadicFunctionStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         symboltable.VariadicFunctionSymbol
}

func (p VariadicFunctionStatement) GetName() string {
	return p.Symbol.Name_
}

func (p VariadicFunctionStatement) Stmt() {

}

type LetVarStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         symboltable.LetVarSymbol
}

func (p LetVarStatement) GetName() string {
	return p.Symbol.Name_
}

func (p LetVarStatement) Stmt() {

}

type LetFuncStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         symboltable.LetfunSymbol
}

func (p LetFuncStatement) GetName() string {
	return p.Symbol.Name_
}

func (p LetFuncStatement) Stmt() {

}
