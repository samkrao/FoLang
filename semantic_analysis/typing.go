package semantic_analysis

import (
	"github.com/samkrao/fo-lang/src/ast"
	symboltable "github.com/samkrao/fo-lang/src/context"
)

func inferTypes(r *Result) {
	Walk(r.Root, func(node ast.SET) bool {
		if expression, ok := node.(ast.Expr); ok {
			r.Types[occurrence(expression)] = typeOf(r, expression)
		}
		return true
	})
}

func typeOf(r *Result, expression ast.Expr) Type {
	switch value := expression.(type) {
	case ast.IntegerLiteral:
		return concrete(value.ActType_, "co.lang.int")
	case ast.NumberLiteral:
		return concrete(value.ActType_, "co.lang.float")
	case ast.StringLiteral:
		return concrete(value.ActType_, "co.lang.string")
	case ast.CharacterLiteral:
		return concrete(value.ActType_, "co.lang.char")
	case ast.BooleanLiteral:
		return concrete(value.ActType_, "co.lang.bool")
	case ast.GroupingExpr:
		return knownType(r, value.Expr_)
	case ast.SymbolExpr:
		resolved := r.Resolutions[occurrence(value)]
		if resolved.SymbolID == "" || r.Symbols == nil {
			return UnknownType
		}
		return symbolType(r.Symbols.GetSymbol(resolved.SymbolID))
	case ast.AssignmentExpr:
		left, right := knownType(r, value.Assigne), knownType(r, value.AssignedValue)
		if !left.Unknown && !right.Unknown && left.Name != right.Name {
			r.report(PhaseTyping, "SEM2001", "assignment has incompatible types "+left.Name+" and "+right.Name, value)
		}
		return left
	case ast.ConditionalExpr:
		return Type{Name: "co.lang.bool"}
	}
	return UnknownType
}

func knownType(r *Result, expression ast.Expr) Type {
	if expression == nil {
		return UnknownType
	}
	if typ, ok := r.Types[occurrence(expression)]; ok {
		return typ
	}
	return typeOf(r, expression)
}

func concrete(candidate, fallback string) Type {
	if candidate != "" {
		return Type{Name: candidate}
	}
	return Type{Name: fallback}
}

func symbolType(symbol symboltable.SymbolInfo) Type {
	if symbol == nil || symbol.GetType() == "" {
		return UnknownType
	}
	return Type{Name: symbol.GetType()}
}
