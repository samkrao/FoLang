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

func (n DummyStmt) GetName() string { return "DummyStmt" }

func (d DummyStmt) stmt() {}

type PragmaStatement struct {
	PragmaSymb string //symboltable.PDADSymbol
}

func (p PragmaStatement) GetName() string {
	return reflect.TypeOf(p).Name()
}

func (p PragmaStatement) stmt() {}

type DirectiveStatement struct {
	DirectiveSymb string //symboltable.PDADSymbol
}

func (p DirectiveStatement) stmt() {}

func (p DirectiveStatement) GetName() string {
	return reflect.TypeOf(p).Name()
}

type DecoratorStatement struct {
	DecoratorSymb string //symboltable.PDADSymbol
}

func (p DecoratorStatement) stmt() {}

func (p DecoratorStatement) GetName() string {
	return reflect.TypeOf(p).Name()
}

type AnnotationStatement struct {
	AnnotationSymb string //symboltable.PDADSymbol
}

func (p AnnotationStatement) GetName() string {
	return reflect.TypeOf(p).Name()
}

func (p AnnotationStatement) stmt() {}

type ProjectStatment struct {
	EntryStmt      EntryStatement
	PragmaStmts    []PragmaStatement
	DirectiveStmts []DirectiveStatement
	Sets           []SET
	Symbol         string //symboltable.ProgramSymbol
}

func (p ProjectStatment) stmt() {
}

func (p ProjectStatment) GetName() string {
	return reflect.TypeOf(p).Name()
}

type EntryStatement interface {
	Stmt
	Kind() string
}

type ComponentStatement struct {
	Symbol           string //symboltable.ComponentSymbol
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
	return reflect.TypeOf(p).Name()
}

type ApplicationStatement struct {
	Symbol string //symboltable.ApplicationSymbol
}

func (p ApplicationStatement) stmt() {}

func (p ApplicationStatement) GetName() string {
	return reflect.TypeOf(p).Name()
}

func (p ApplicationStatement) Kind() string {
	return "application"
}

type FunctionDeclarrationStatement struct {
	Symbol         string //symboltable.FunctionSymbol
	AnnotationStmt []AnnotationStatement
	DecoratorStmt  []DecoratorStatement
	Sets           []ValidFuncInnerStmts //statemetns expressions type aliases etc
}

func (p FunctionDeclarrationStatement) GetName() string {
	return reflect.TypeOf(p).Name()
}
func (p FunctionDeclarrationStatement) IsValid() bool {
	return true
}

func (p FunctionDeclarrationStatement) Stmt() {

}

type CStructStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         string // symboltable.CStructSymbol
}

func (p CStructStatement) GetName() string {
	return reflect.TypeOf(p).Name()
}

func (p CStructStatement) Stmt() {

}

type StructStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         string //symboltable.StructSymbol
}

func (p StructStatement) GetName() string {
	return reflect.TypeOf(p).Name()
}

func (p StructStatement) Stmt() {

}

type InterfaceSatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         string //symboltable.InterfaceSymbol
}

func (p InterfaceSatement) GetName() string {
	return reflect.TypeOf(p).Name()
}

func (p InterfaceSatement) Stmt() {

}

type ClassStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         string //symboltable.ClassSymbol
}

func (p ClassStatement) GetName() string {
	return reflect.TypeOf(p).Name()
}

func (p ClassStatement) Stmt() {

}

type SignatureStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         string //symboltable.SignatureSymbol
}

func (p SignatureStatement) GetName() string {
	return reflect.TypeOf(p).Name()
}

func (p SignatureStatement) Stmt() {

}

type ModuleStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         string //symboltable.ModuleSymbol
}

func (p ModuleStatement) GetName() string {
	return reflect.TypeOf(p).Name()
}

func (p ModuleStatement) Stmt() {

}

type ObjectStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         string //symboltable.ObjectSymbol
}

func (p ObjectStatement) GetName() string {
	return reflect.TypeOf(p).Name()
}

func (p ObjectStatement) Stmt() {

}

type MixinStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         string //symboltable.MixinSymbol
}

func (p MixinStatement) GetName() string {
	return reflect.TypeOf(p).Name()
}

func (p MixinStatement) Stmt() {

}

type TraitStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         string //symboltable.TraitSymbol
}

func (p TraitStatement) GetName() string {
	return reflect.TypeOf(p).Name()
}

func (p TraitStatement) Stmt() {

}

type InstanceStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         string //symboltable.InstanceSymbol
}

func (p InstanceStatement) GetName() string {
	return reflect.TypeOf(p).Name()
}

func (p InstanceStatement) Stmt() {

}

type TypeClassStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         string //symboltable.TypeClassSymbol
}

func (p TypeClassStatement) GetName() string {
	return reflect.TypeOf(p).Name()
}

func (p TypeClassStatement) Stmt() {

}

type MatcherStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         string //symboltable.MatcherSymbol
}

func (p MatcherStatement) GetName() string {
	return reflect.TypeOf(p).Name()
}

func (p MatcherStatement) Stmt() {

}

type IndexerStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         string //symboltable.IndexerSymbol
}

func (p IndexerStatement) GetName() string {
	return reflect.TypeOf(p).Name()
}

func (p IndexerStatement) Stmt() {

}

type MacroStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         string //symboltable.MacroSymbol
}

func (p MacroStatement) GetName() string {
	return reflect.TypeOf(p).Name()
}

func (p MacroStatement) Stmt() {

}

type ExtensionStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         string //symboltable.ExtensionSymbol
}

func (p ExtensionStatement) GetName() string {
	return reflect.TypeOf(p).Name()
}

func (p ExtensionStatement) Stmt() {

}

type ExtensionMethodStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         string //symboltable.ExensionMethodSymbol
}

func (p ExtensionMethodStatement) GetName() string {
	return reflect.TypeOf(p).Name()
}

func (p ExtensionMethodStatement) Stmt() {

}

type NativeMethodStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         string //symboltable.NativeFunctionSymbol
}

func (p NativeMethodStatement) GetName() string {
	return reflect.TypeOf(p).Name()
}

func (p NativeMethodStatement) Stmt() {

}

type DelegateStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         string //symboltable.DelegateType
}

func (p DelegateStatement) GetName() string {
	return reflect.TypeOf(p).Name()
}

func (p DelegateStatement) Stmt() {

}

type ChainedMethodStatement struct {
	AnnotationStmt     []AnnotationStatement
	DecoratorStatement []DecoratorStatement
	Symbol             string //symboltable.SignatureSymbol
}

func (p ChainedMethodStatement) GetName() string {
	return reflect.TypeOf(p).Name()
}

func (p ChainedMethodStatement) Stmt() {

}

type CurriedFunctionStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         string //symboltable.CurryingFunctionSymbol
}

func (p CurriedFunctionStatement) GetName() string {
	return reflect.TypeOf(p).Name()
}

func (p CurriedFunctionStatement) Stmt() {

}

type NamedFunctionStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         string //symboltable.NamedParameterFunctionSymbol
}

func (p NamedFunctionStatement) GetName() string {
	return reflect.TypeOf(p).Name()
}

func (p NamedFunctionStatement) Stmt() {

}

type OptionalParameterFuncStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         string //symboltable.OptionalParameterFunctionSymbol
}

func (p OptionalParameterFuncStatement) GetName() string {
	return reflect.TypeOf(p).Name()
}

func (p OptionalParameterFuncStatement) Stmt() {

}

type DefaultParameterFuncStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         string //symboltable.DefaultParameterFunctionSymbol
}

func (p DefaultParameterFuncStatement) GetName() string {
	return reflect.TypeOf(p).Name()
}

func (p DefaultParameterFuncStatement) Stmt() {

}

type VariadicFunctionStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         string //symboltable.VariadicFunctionSymbol
}

func (p VariadicFunctionStatement) GetName() string {
	return reflect.TypeOf(p).Name()
}

func (p VariadicFunctionStatement) Stmt() {

}

type LetVarStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         string //symboltable.LetVarSymbol
}

func (p LetVarStatement) GetName() string {
	return reflect.TypeOf(p).Name()
}

func (p LetVarStatement) Stmt() {

}

type LetFuncStatement struct {
	AnnotationStmt []AnnotationStatement
	Symbol         string //symboltable.LetfunSymbol
}

func (p LetFuncStatement) GetName() string {
	return reflect.TypeOf(p).Name()
}

func (p LetFuncStatement) Stmt() {

}
