package parser

import (
	"github.com/samkrao/fo-lang/src/ast"
	symboltable "github.com/samkrao/fo-lang/src/context"
	"github.com/samkrao/fo-lang/src/helpers"
)

// Symbol records for AST nodes.
//
// Every ast node carries a Symb pointer and its GetName method dereferences it,
// so a node with a nil Symb panics the moment anything walks the tree. This file
// is the single place that mints those records, which keeps two things true:
//
//   - No node is ever constructed without one.
//   - Minting is separate from BINDING. Nothing here touches a symbol table: a
//     record is per-node until a declaration site hands it to bind.go, so the
//     records made for references, and for a speculative parse that is thrown
//     away, leave no trace in the shared model. What every record does take here
//     is the id of the segment active at its source position, which is the table
//     its declaration will bind into.
//
// The name stored is the scanned spelling, i.e. the backend-lowered one carrying
// the "_fo" suffix (DECISION-BACKEND-001), because that is what later phases
// emit. Grammar-level comparisons use logicalName instead.

// details builds the SymbolDetails common to every symbol record.
func (p *parser) details(name string, kind symboltable.SymbolsToString, typ string) symboltable.SymbolDetails {
	return symboltable.SymbolDetails{
		SymbolId_:     helpers.NewSymbolId(),
		SymbolType_:   string(kind),
		Name_:         name,
		State:         symboltable.Unresolved,
		Type_:         typ,
		SymbolTableId: p.symtab.Id,
	}
}

// variableDetails builds the VariableDetails shared by every variable-like
// symbol: plain variables, parameters, returns, fields and the derived forms
// (pointer, array, reference, address, thunk, slice, range).
func (p *parser) variableDetails(name string, kind symboltable.SymbolsToString, actType string) symboltable.VariableDetails {
	return symboltable.VariableDetails{
		SymbolDetails: p.details(name, kind, actType),
		ActType_:      actType,
		Type_:         actType,
	}
}

func (p *parser) varSymbol(name, actType string) *symboltable.VarSymbol {
	return &symboltable.VarSymbol{
		VariableDetails: p.variableDetails(name, symboltable.S_VarSymbol, actType),
	}
}

func (p *parser) pointerSymbol(name, actType string) *symboltable.PointerSymbol {
	return &symboltable.PointerSymbol{
		VariableDetails: p.variableDetails(name, symboltable.S_PointerSymbol, actType),
	}
}

func (p *parser) arraySymbol(name, actType string) *symboltable.ArraySymbol {
	return &symboltable.ArraySymbol{
		VariableDetails: p.variableDetails(name, symboltable.S_ArraySymbol, actType),
	}
}

func (p *parser) referenceSymbol(name, actType string) *symboltable.ReferenceSymbol {
	return &symboltable.ReferenceSymbol{
		VariableDetails: p.variableDetails(name, symboltable.S_ReferenceSymbol, actType),
	}
}

func (p *parser) addressSymbol(name, actType string) *symboltable.AddressSymbol {
	return &symboltable.AddressSymbol{
		VariableDetails: p.variableDetails(name, symboltable.S_AddressSymbol, actType),
	}
}

func (p *parser) thunkSymbol(name, actType string) *symboltable.ThunkSymbol {
	return &symboltable.ThunkSymbol{
		VariableDetails: p.variableDetails(name, symboltable.S_ThunkSymbol, actType),
	}
}

func (p *parser) rangeSymbol(name, actType string) *symboltable.RangeSymbol {
	return &symboltable.RangeSymbol{
		VariableDetails: p.variableDetails(name, symboltable.S_RangeSymbol, actType),
	}
}

// declFor builds the ast.SymbolDeclStmt that ast.Parameter and ast.Returns embed.
//
// Both types embed the SymbolDeclStmt INTERFACE, and their promoted GetActType method resolves
// through it. Leaving it nil therefore makes any walk that asks a function type or a function
// declaration for its type signature dereference a nil interface — ast.FunctionType.GetActType
// iterates its parameters and results and calls GetActType on each. Every Parameter and
// Returns the parser builds must have this populated.
func (p *parser) declFor(name string, actType string, t ast.Type) ast.SymbolDeclStmt {
	return ast.VarDeclarationStmt{
		// This declaration is synthesized to carry a parameter's or result's
		// type, so it covers the same source that type does. It has no
		// declarator of its own to measure.
		Span: spanOfNode(t, p.spanFrom(p.pos)),
		BasicVarStmt: ast.BasicVarStmt{
			Identifier: name,
			Type_:      t,
			VarType:    actType,
		},
		Symb: p.varSymbol(name, actType),
	}
}

// genericSymbol is the wrapper used by a plain Symbol field, as on ast.Parameter.
func (p *parser) genericSymbol(name string, kind symboltable.SymbolsToString, actType string) *symboltable.Symbol {
	return &symboltable.Symbol{
		SymbolDetails: p.details(name, kind, actType),
		ResolveState_: symboltable.Unresolved,
	}
}

func (p *parser) exprSymbol(name string) *symboltable.ExpressionSymbol {
	return &symboltable.ExpressionSymbol{
		SymbolDetails: p.details(name, symboltable.S_ExpressionSymbol, ""),
	}
}

func (p *parser) stmtSymbol(name string) *symboltable.StatmentSymbol {
	return &symboltable.StatmentSymbol{
		SymbolDetails: p.details(name, symboltable.S_StatmentSymbol, ""),
	}
}

func (p *parser) typeSymbol(name string) *symboltable.TypeSymbol {
	return &symboltable.TypeSymbol{
		SymbolDetails: p.details(name, symboltable.S_TypeSymbol, name),
	}
}

func (p *parser) functionSymbol(name string) *symboltable.FunctionSymbol {
	return &symboltable.FunctionSymbol{
		SymbolDetails: p.details(name, symboltable.S_FunctionSymbol, ""),
	}
}

func (p *parser) structSymbol(name string) *symboltable.StructSymbol {
	return &symboltable.StructSymbol{
		SymbolDetails: p.details(name, symboltable.S_StructSymbol, name),
	}
}

func (p *parser) enumSymbol(name string) *symboltable.EnumSymbol {
	return &symboltable.EnumSymbol{
		SymbolDetails: p.details(name, symboltable.S_EnumSymbol, name),
	}
}

func (p *parser) unionSymbol(name string) *symboltable.UnionSymbol {
	return &symboltable.UnionSymbol{
		SymbolDetails: p.details(name, symboltable.S_UnionSymbol, name),
	}
}

func (p *parser) classSymbol(name string) *symboltable.ClassSymbol {
	return &symboltable.ClassSymbol{
		SymbolDetails: p.details(name, symboltable.S_ClassSymbol, name),
	}
}

func (p *parser) moduleSymbol(name string) *symboltable.ModuleSymbol {
	s := &symboltable.ModuleSymbol{
		SymbolDetails: p.details(name, symboltable.S_ModuleSymbol, name),
	}
	s.Name = name
	return s
}

func (p *parser) blockSymbol(name string, named bool) *symboltable.BlockSymbol {
	return &symboltable.BlockSymbol{
		SymbolDetails: p.details(name, symboltable.S_BlockSymbol, ""),
		IsNamed:       named,
	}
}

// packageSymbol mints the symbol for a package.
//
// The record is a ComponentSymbol carrying the package kind rather than a
// PackageSymbol, so that a package, a component and a project are described by
// one symbol shape: each names a FOLDER that holds declarations, and a consumer
// walking the tree should not need three record types to ask the same question.
func (p *parser) packageSymbol(name string) *symboltable.ComponentSymbol {
	s := &symboltable.ComponentSymbol{
		SymbolDetails: p.details(name, symboltable.S_PackageSymbol, name),
	}
	s.Name = name
	s.Kind = "package"
	return s
}

func (p *parser) applicationSymbol(name string) *symboltable.ApplicationSymbol {
	s := &symboltable.ApplicationSymbol{
		SymbolDetails: p.details(name, symboltable.S_PackageSymbol, name),
	}
	s.Name = name
	return s
}

func (p *parser) extensionSymbol(name, forType string) *symboltable.ExtensionSymbol {
	s := &symboltable.ExtensionSymbol{
		SymbolDetails: p.details(name, symboltable.S_ExtensionSymbol, name),
	}
	s.ForType = forType
	return s
}

func (p *parser) componentSymbol(name, kind string) *symboltable.ComponentSymbol {
	s := &symboltable.ComponentSymbol{
		SymbolDetails: p.details(name, symboltable.S_ComponentSymbol, name),
	}
	s.Name = name
	s.Kind = kind
	return s
}

func (p *parser) delegateSymbol(name string) *symboltable.DelegateSymbol {
	s := &symboltable.DelegateSymbol{
		SymbolDetails: p.details(name, symboltable.S_DelegateSymbol, name),
	}
	s.Name = name
	return s
}

func (p *parser) lambdaSymbol(name string) *symboltable.LambdaSymbol {
	return &symboltable.LambdaSymbol{
		SymbolDetails: p.details(name, symboltable.S_LambdaSymbol, ""),
	}
}

func (p *parser) letSymbol(name string) *symboltable.LetBindings {
	return &symboltable.LetBindings{
		SymbolDetails: p.details(name, symboltable.S_LetBindings, ""),
	}
}

func (p *parser) comprehensionSymbol(name string) *symboltable.ForComprehension {
	return &symboltable.ForComprehension{
		SymbolDetails: p.details(name, symboltable.S_ForComprehension, ""),
	}
}

func (p *parser) patternSymbol(name string, letForm bool) *symboltable.FunctionPattern {
	return &symboltable.FunctionPattern{
		SymbolDetails: p.details(name, symboltable.S_FunctionPattern, ""),
		IsLetForm:     letForm,
	}
}

func (p *parser) matcherSymbol(name string) *symboltable.MatcherSymbol {
	return &symboltable.MatcherSymbol{
		SymbolDetails: p.details(name, symboltable.S_MatcherSymbol, ""),
	}
}

func (p *parser) matcherImplSymbol(name string) *symboltable.MatcherImplSymbol {
	return &symboltable.MatcherImplSymbol{
		SymbolDetails: p.details(name, symboltable.S_MatcherImplSymbol, ""),
	}
}

func (p *parser) instanceSymbol(name string) *symboltable.InstanceSymbol {
	return &symboltable.InstanceSymbol{
		SymbolDetails: p.details(name, symboltable.S_InstanceSymbol, name),
	}
}

func (p *parser) objectSymbol(name string) *symboltable.ObjectSymbol {
	return &symboltable.ObjectSymbol{
		SymbolDetails: p.details(name, symboltable.S_ObjectSymbol, name),
	}
}

func (p *parser) interfaceSymbol(name string) *symboltable.InterfaceSymbol {
	return &symboltable.InterfaceSymbol{
		SymbolDetails: p.details(name, symboltable.S_InterfaceSymbol, name),
	}
}

func (p *parser) signatureSymbol(name string) *symboltable.SignatureSymbol {
	return &symboltable.SignatureSymbol{
		SymbolDetails: p.details(name, symboltable.S_SignatureSymbol, name),
	}
}

func (p *parser) typeclassSymbol(name string) *symboltable.TypeclassSymbol {
	return &symboltable.TypeclassSymbol{
		SymbolDetails: p.details(name, symboltable.S_TypeclassSymbol, name),
	}
}

func (p *parser) typeConstructorSymbol(name string) *symboltable.TypeConstructor {
	return &symboltable.TypeConstructor{
		SymbolDetails: p.details(name, symboltable.S_TypeConstructor, name),
	}
}

func (p *parser) variantConstructorSymbol(name string) *symboltable.VariantConstructor {
	return &symboltable.VariantConstructor{
		SymbolDetails: p.details(name, symboltable.S_VariantConstructor, name),
	}
}

func (p *parser) directiveSymbol(name string, isPragma bool) *symboltable.DirectivePragmaDetails {
	return &symboltable.DirectivePragmaDetails{
		SymbolDetails: p.details(name, symboltable.S_DirectivePragmaDetails, ""),
		IsPragma:      isPragma,
	}
}

func (p *parser) useSymbol(name string) *symboltable.UseSymbol {
	return &symboltable.UseSymbol{
		SymbolDetails: p.details(name, symboltable.S_UseSymbol, ""),
	}
}

func (p *parser) genericDetails(name string, params []symboltable.GenericTypeParam) *symboltable.GenericDetails {
	names := make([]string, 0, len(params))
	for _, tp := range params {
		names = append(names, tp.Name)
	}
	return &symboltable.GenericDetails{
		SymbolDetails:    p.details(name, symboltable.S_GenericDetails, ""),
		TypeSymbols:      names,
		GenericTypeParam: params,
	}
}
