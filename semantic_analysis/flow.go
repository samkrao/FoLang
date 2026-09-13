package semantic_analysis

import (
	"fmt"

	"github.com/samkrao/fo-lang/src/ast"
)

// checkFlow starts a fresh definite-assignment state for every callable. Nested
// callables are checked independently; their execution is not a path through
// the declaring callable.
func checkFlow(r *Result) {
	Walk(r.Root, func(node ast.SET) bool {
		switch function := node.(type) {
		case ast.FunctionDeclarationStmt:
			state := assignmentState{}
			for _, group := range function.Parameters {
				for _, parameter := range group {
					state[parameter.Name_] = true
				}
			}
			analyzeStatements(r, function.Body, state)
			return false
		case ast.FunctionExpr:
			state := assignmentState{}
			for _, parameter := range function.Parameters {
				state[parameter.Name_] = true
			}
			analyzeStatements(r, function.Body, state)
			return false
		}
		return true
	})
}

type assignmentState map[string]bool

func (s assignmentState) clone() assignmentState {
	out := assignmentState{}
	for key, value := range s {
		out[key] = value
	}
	return out
}

func analyzeStatements(r *Result, statements []ast.Stmt, state assignmentState) bool {
	reachable := true
	for _, statement := range statements {
		if !reachable {
			r.report(PhaseFlow, "SEM3001", "unreachable statement", statement)
			continue
		}
		switch value := statement.(type) {
		case ast.VarDeclarationStmt:
			checkReads(r, value.AssignedValue, state)
			state[value.Identifier] = value.AssignedValue != nil || value.IsInitialize()
		case ast.ArrayVariableDeclStmt:
			checkReads(r, value.AssignedValue, state)
			state[value.Identifier] = value.AssignedValue != nil || value.IsInitialize()
		case ast.PointerVariableDeclStmt:
			checkReads(r, value.AssignedValue, state)
			state[value.Identifier] = value.AssignedValue != nil || value.IsInitialize()
		case ast.RefVariableDeclStmt:
			checkReads(r, value.AssignedValue, state)
			state[value.Identifier] = value.AssignedValue != nil || value.IsInitialize()
		case ast.ExpressionStmt:
			applyExpression(r, value.Expression, state)
		case ast.ReturnStmt:
			if expression, ok := value.StmtExpr_.(ast.Expr); ok {
				checkReads(r, expression, state)
			}
			reachable = false
		case ast.BlockStmt:
			analyzeStatements(r, value.Body, state.clone())
		case ast.ConditionalStmt:
			checkReads(r, value.IfExpr, state)
			branches := []assignmentState{}
			thenState := state.clone()
			branchReturns := analyzeOne(r, value.IfStmt, thenState)
			branches = append(branches, thenState)
			allReturn := branchReturns
			for _, elif := range value.ElifExprStmt {
				checkReads(r, elif.IfExpr, state)
				next := state.clone()
				returns := analyzeOne(r, elif.IfStmt, next)
				branches = append(branches, next)
				allReturn = allReturn && returns
			}
			if value.ElseExprStmt != nil {
				next := state.clone()
				returns := analyzeOne(r, value.ElseExprStmt.Stmt_, next)
				branches = append(branches, next)
				allReturn = allReturn && returns
			} else {
				branches = append(branches, state.clone())
				allReturn = false
			}
			intersectAssignments(state, branches)
			reachable = !allReturn
		}
	}
	return !reachable
}

func analyzeOne(r *Result, statement ast.Stmt, state assignmentState) bool {
	if block, ok := statement.(ast.BlockStmt); ok {
		return analyzeStatements(r, block.Body, state)
	}
	return analyzeStatements(r, []ast.Stmt{statement}, state)
}

func intersectAssignments(target assignmentState, branches []assignmentState) {
	for name := range target {
		assigned := true
		for _, branch := range branches {
			assigned = assigned && branch[name]
		}
		target[name] = assigned
	}
}

func applyExpression(r *Result, expression ast.Expr, state assignmentState) {
	if assignment, ok := expression.(ast.AssignmentExpr); ok {
		checkReads(r, assignment.AssignedValue, state)
		if symbol, ok := assignment.Assigne.(ast.SymbolExpr); ok {
			state[symbol.Value] = true
			return
		}
	}
	checkReads(r, expression, state)
}

func checkReads(r *Result, expression ast.Expr, state assignmentState) {
	if expression == nil {
		return
	}
	Walk(expression, func(node ast.SET) bool {
		symbol, ok := node.(ast.SymbolExpr)
		if !ok {
			return true
		}
		if assigned, local := state[symbol.Value]; local && !assigned {
			r.report(PhaseFlow, "SEM3002", fmt.Sprintf("%q may be read before it is initialized", symbol.Value), symbol)
		}
		return true
	})
}
