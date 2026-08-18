package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
)

// Control-flow lowering — the pass that turns canonical chains into dedicated AST nodes.
//
// Parsing produces a uniform postfix chain for every control construct, because that is what the
// grammar requires (see the header of chain.go). This pass walks the finished tree and rewrites
// the chains that match a canonical shape from section 12:
//
//	(c).then({…}).otherwise(c2).then({…}).default({…})   -> ConditionalStmt
//	(c).loop({…})                                        -> ConditionalStmt with Loop set
//	(c).then(a).default(b)                               -> TernaryStmt
//	arr.each(i, v, {…})                                  -> ForeachStmt
//	arr.contains(k).then({…}).default({…})               -> ConditionalStmt over ContainsStmt
//
// Anything that does not match is left exactly as parsed. That is the important property: the
// pass only ever narrows a generic chain into a more specific node, so an unrecognised
// method chain — a pipeline, a user's own `.then`-named method — keeps working.
//
// Lowering is bottom-up: each construct lowers the blocks it contains as it builds, so a
// conditional nested inside a loop body becomes a ConditionalStmt too.

// lowerControlFlow rewrites the control-flow chains in a parsed compilation unit.
//
// The root node types are values, so each is rebuilt with its lowered body.
func (p *parser) lowerControlFlow(root ast.Stmt) ast.Stmt {
	switch n := root.(type) {
	case ast.Application:
		n.Body = p.lowerStatements(n.Body)
		return n
	case ast.PackageStmt:
		n.Body = p.lowerStatements(n.Body)
		return n
	case ast.Library:
		n.Body = p.lowerStatements(n.Body)
		return n
	default:
		return p.lowerStatement(root)
	}
}

// lowerStatements lowers every statement in a list.
func (p *parser) lowerStatements(body []ast.Stmt) []ast.Stmt {
	if body == nil {
		return nil
	}
	out := make([]ast.Stmt, 0, len(body))
	for _, s := range body {
		out = append(out, p.lowerStatement(s))
	}
	return out
}

// lowerStatement lowers one statement, recursing into whatever bodies it contains.
//
// The recursion has to cover every node with a statement list, because a control chain can appear
// anywhere a statement can: inside a function body, a class method, a module member, or a block
// nested in another chain's branch.
func (p *parser) lowerStatement(s ast.Stmt) ast.Stmt {
	switch n := s.(type) {
	case nil:
		return nil

	// The statement that can actually BE a chain.
	case ast.ExpressionStmt:
		if lowered, ok := p.lowerControlChain(n.Expression); ok {
			return lowered
		}
		n.Expression = p.lowerExpr(n.Expression)
		return n

	// Containers of statements.
	case *ast.BlockStmt:
		n.Body = p.lowerStatements(n.Body)
		return n
	case ast.BlockStmt:
		n.Body = p.lowerStatements(n.Body)
		return n
	case ast.FunctionDeclarationStmt:
		n.Parameters = p.lowerParameterLists(n.Parameters)
		n.Body = p.lowerStatements(n.Body)
		return n
	case ast.TypeDeclarationStmt:
		n.Parameters = p.lowerParameterLists(n.Parameters)
		n.Body = p.lowerStatements(n.Body)
		return n
	case ast.ClassDeclarationStmt:
		n.Body = p.lowerStatements(n.Body)
		return n
	case ast.ExtensionDeclarationStmt:
		n.Body = p.lowerStatements(n.Body)
		return n
	case ast.ComponentDeclarationStmt:
		n.Body = p.lowerStatements(n.Body)
		return n
	case ast.ModuleStmt:
		n.Body = p.lowerStatements(n.Body)
		return n
	case ast.ObjectDeclStmt:
		n.Body = p.lowerStatements(n.Body)
		return n
	case ast.TypeclassStmt:
		n.Methods = p.lowerStatements(n.Methods)
		return n
	case ast.TypeclassInstanceStmt:
		n.Body = p.lowerStatements(n.Body)
		return n
	case ast.MatcherInstanceStmt:
		n.Body = p.lowerStatements(n.Body)
		return n
	case ast.MacroStmt:
		n.FunctionDeclarationStmt.Body = p.lowerStatements(n.FunctionDeclarationStmt.Body)
		return n
	case ast.TemplateStmt:
		n.FunctionDeclarationStmt.Body = p.lowerStatements(n.FunctionDeclarationStmt.Body)
		return n
	case ast.OperatorStmt:
		n.FunctionDeclarationStmt.Body = p.lowerStatements(n.FunctionDeclarationStmt.Body)
		return n
	case ast.IndexerStmt:
		n.FunctionDeclarationStmt.Body = p.lowerStatements(n.FunctionDeclarationStmt.Body)
		return n
	case ast.ExtensionStmt:
		n.FunctionDeclarationStmt.Body = p.lowerStatements(n.FunctionDeclarationStmt.Body)
		return n
	case ast.DecoratorStmt:
		n.FunctionDeclarationStmt.Body = p.lowerStatements(n.FunctionDeclarationStmt.Body)
		return n
	case ast.NativeFunctionStmt:
		n.FunctionDeclarationStmt.Body = p.lowerStatements(n.FunctionDeclarationStmt.Body)
		return n
	case ast.ExecutionModelFunctionStmt:
		n.FunctionDeclarationStmt.Body = p.lowerStatements(n.FunctionDeclarationStmt.Body)
		return n
	case ast.GenerricFun:
		n.FunctionDeclarationStmt.Body = p.lowerStatements(n.FunctionDeclarationStmt.Body)
		return n
	case ast.FunctionPatternStmt:
		n.PatternArgs = p.lowerExprs(n.PatternArgs)
		n.Guard = p.lowerExpr(n.Guard)
		n.Body = p.lowerStatements(n.Body)
		if n.BodyExpr != nil {
			n.BodyExpr = p.lowerExpr(n.BodyExpr)
		}
		return n
	case ast.CaseStmt:
		n.Expr_ = p.lowerExpr(n.Expr_)
		n.Stmt_ = p.lowerStatement(n.Stmt_)
		return n
	case ast.MatchExprStmt:
		n.Expr_ = p.lowerExpr(n.Expr_)
		n.MatcherExpr = p.lowerExpr(n.MatcherExpr)
		n.Stmt_ = p.lowerStatement(n.Stmt_)
		return n
	case ast.PatternExprStmt:
		n.Expr_ = p.lowerStatement(n.Expr_).(ast.MatchExprStmt)
		n.Stmt_ = p.lowerStatement(n.Stmt_)
		for i := range n.CaseExprStmt {
			n.CaseExprStmt[i] = p.lowerStatement(n.CaseExprStmt[i]).(ast.CaseStmt)
		}
		if n.DefaultExprStmt != nil {
			lowered := p.lowerStatement(*n.DefaultExprStmt).(ast.CaseStmt)
			n.DefaultExprStmt = &lowered
		}
		return n

	// A declaration's initializer can contain a chain, as in an anonymous function body.
	case ast.VarDeclarationStmt:
		n.BasicVarStmt = p.lowerBasicVar(n.BasicVarStmt)
		return n
	case ast.ArrayVariableDeclStmt:
		n.BasicVarStmt = p.lowerBasicVar(n.BasicVarStmt)
		n.Sizes = p.lowerExprs(n.Sizes)
		return n
	case ast.PointerVariableDeclStmt:
		n.BasicVarStmt = p.lowerBasicVar(n.BasicVarStmt)
		return n
	case ast.RefVariableDeclStmt:
		n.BasicVarStmt = p.lowerBasicVar(n.BasicVarStmt)
		return n
	case ast.AddressVariableDeclStmt:
		n.BasicVarStmt = p.lowerBasicVar(n.BasicVarStmt)
		return n
	case ast.ThunkVariableDeclStmt:
		n.BasicVarStmt = p.lowerBasicVar(n.BasicVarStmt)
		return n
	case ast.HeapAllocatedRefStmt:
		n.BasicVarStmt = p.lowerBasicVar(n.BasicVarStmt)
		return n
	case ast.SliceVariableDeclStmt:
		n.BasicVarStmt = p.lowerBasicVar(n.BasicVarStmt)
		return n
	case ast.RangeVariableDeclStmt:
		n.BasicVarStmt = p.lowerBasicVar(n.BasicVarStmt)
		return n

	case ast.ReturnStmt:
		switch payload := n.StmtExpr_.(type) {
		case ast.Stmt:
			n.StmtExpr_ = p.lowerStatement(payload)
		case ast.Expr:
			n.StmtExpr_ = p.lowerExpr(payload)
		}
		return n

	case ast.ConditionalStmt:
		n.IfExpr = p.lowerExpr(n.IfExpr)
		n.IfStmt = p.lowerStatement(n.IfStmt)
		for i := range n.ElifExprStmt {
			n.ElifExprStmt[i] = p.lowerStatement(n.ElifExprStmt[i]).(ast.ConditionalStmt)
		}
		if n.ElseExprStmt != nil {
			lowered := p.lowerStatement(*n.ElseExprStmt).(ast.DefaultConditionalStmt)
			n.ElseExprStmt = &lowered
		}
		return n
	case ast.DefaultConditionalStmt:
		n.Stmt_ = p.lowerStatement(n.Stmt_)
		n.Expr_ = p.lowerExprs(n.Expr_)
		return n
	case ast.TernaryStmt:
		n.Expr_ = p.lowerExpr(n.Expr_)
		n.Stmt_ = p.lowerStatement(n.Stmt_)
		for i := range n.ElifExprStmt {
			n.ElifExprStmt[i] = p.lowerStatement(n.ElifExprStmt[i]).(ast.TernaryStmt)
		}
		if n.ElseExprStmt != nil {
			lowered := p.lowerStatement(*n.ElseExprStmt).(ast.DefaultConditionalStmt)
			n.ElseExprStmt = &lowered
		}
		return n

	default:
		return s
	}
}

// lowerBasicVar centralizes initializer recursion for every storage-specific
// declaration wrapper. All of those nodes embed the same BasicVarStmt payload.
func (p *parser) lowerBasicVar(n ast.BasicVarStmt) ast.BasicVarStmt {
	n.AssignedValue = p.lowerExpr(n.AssignedValue)
	return n
}

// lowerParameters reaches default expressions on both declared and anonymous functions.
func (p *parser) lowerParameters(params []ast.Parameter) []ast.Parameter {
	for i := range params {
		params[i].Default = p.lowerExpr(params[i].Default)
	}
	return params
}

// lowerParameterLists applies the same recursion to every curried parameter group.
func (p *parser) lowerParameterLists(lists [][]ast.Parameter) [][]ast.Parameter {
	for i := range lists {
		lists[i] = p.lowerParameters(lists[i])
	}
	return lists
}

// lowerControlChain attempts every canonical shape against an expression.
//
// `.each` and `.contains` name the iterator and containment forms, and each is
// keyed by its own leading verb, so neither can be confused with anything else.
//
// The selection forms both open with `.then`, because the ternary and the
// condition chain share the whole selection vocabulary and differ in what their
// branches CARRY: a ternary's are values and a condition's are blocks. The
// ternary is therefore tried first, and it declines any chain whose branches are
// blocks — leaving that chain to the condition matcher — rather than the two
// being told apart by their opening verb.
func (p *parser) lowerControlChain(e ast.Expr) (ast.Stmt, bool) {
	c, ok := decomposeChain(e)
	if !ok {
		return nil, false
	}

	var lowered ast.Stmt
	switch {
	case c.verbAt(0) == verbEach:
		lowered, ok = p.lowerEachChain(c)
	case containsVerbs[c.verbAt(0)]:
		lowered, ok = p.lowerContainsChain(c)
	case isBranchVerb(c.verbAt(0)):
		if lowered, ok = p.lowerTernaryChain(c); ok {
			break
		}
		lowered, ok = p.lowerConditionalChain(c)
	default:
		return nil, false
	}
	if !ok {
		return nil, false
	}
	return retainOriginalControlChain(lowered, e), true
}

// retainOriginalControlChain makes lowering reversible for receiver-aware
// method resolution. Reserved spellings are only built-in candidates during
// parsing: a class/companion method or activated extension may still win. The
// dedicated control nodes remain available to existing consumers, while the
// complete uniform CallExpr/MemberExpr tree remains available to the resolver.
func retainOriginalControlChain(lowered ast.Stmt, original ast.Expr) ast.Stmt {
	switch node := lowered.(type) {
	case ast.ConditionalStmt:
		node.OriginalChain = original
		return node
	case ast.TernaryStmt:
		node.OriginalChain = original
		return node
	case ast.ForeachStmt:
		node.OriginalChain = original
		return node
	default:
		return lowered
	}
}

// lowerExpr recurses into an expression so that chains nested inside one are lowered too.
//
// A chain can hide inside a block used as an argument, an anonymous function body, or an
// assignment's right-hand side. Where a nested expression turns out to BE a control chain, it is
// rewrapped in ast.StatementExpr so it still occupies an expression slot.
func (p *parser) lowerExpr(e ast.Expr) ast.Expr {
	if e == nil {
		return nil
	}

	// A nested expression may itself be a control chain. The common case is a ternary, which
	// is almost always the right-hand side of an assignment:
	//
	//	s = (truth).return(a).otherwise.return(b);
	//
	// Here the statement is an assignment, not a chain, so the chain is only reachable by
	// attempting the match during expression recursion. A matched chain becomes a statement,
	// rewrapped so it still occupies an expression slot.
	if lowered, ok := p.lowerControlChain(e); ok {
		return ast.StatementExpr{
			// The rewrapped chain occupies the same source the original
			// expression did.
			Span:      spanOfNode(e, ast.Span{}),
			Statement: lowered,
			Symb:      p.exprSymbol("control-flow"),
		}
	}

	switch n := e.(type) {
	case ast.StatementExpr:
		n.Statement = p.lowerStatement(n.Statement)
		return n

	case ast.CallExpr:
		n.Method = p.lowerExpr(n.Method)
		n.Arguments = p.lowerExprs(n.Arguments)
		return n

	case ast.MemberExpr:
		n.Member = p.lowerExpr(n.Member)
		return n

	case ast.ComputedExpr:
		n.Member = p.lowerExpr(n.Member)
		n.Property = p.lowerExpr(n.Property)
		return n

	case ast.GroupingExpr:
		n.Expr_ = p.lowerExpr(n.Expr_)
		return n

	case ast.BinaryExpr:
		n.Left = p.lowerExpr(n.Left)
		n.Right = p.lowerExpr(n.Right)
		return n

	case ast.CommaExpr:
		n.Left = p.lowerExpr(n.Left)
		n.Right = p.lowerExpr(n.Right)
		return n

	case ast.ADTExpr:
		n.Left = p.lowerExpr(n.Left)
		n.Right = p.lowerExpr(n.Right)
		return n

	case ast.ConditionalExpr:
		n.Left = p.lowerExpr(n.Left)
		n.Right = p.lowerExpr(n.Right)
		n.ArrayVar = p.lowerStatement(n.ArrayVar)
		n.CondVarStmt = p.lowerExpr(n.CondVarStmt)
		n.CondValStmt = p.lowerExpr(n.CondValStmt)
		return n

	case ast.AssignmentExpr:
		n.Assigne = p.lowerExpr(n.Assigne)
		n.AssignedValue = p.lowerExpr(n.AssignedValue)
		return n

	case ast.PrefixExpr:
		n.Right = p.lowerExpr(n.Right)
		return n

	case ast.FunctionExpr:
		n.Parameters = p.lowerParameters(n.Parameters)
		n.Body = p.lowerStatements(n.Body)
		return n

	case ast.LambdaExpr:
		n.Body = p.lowerExpr(n.Body)
		return n

	case ast.ArrayLiteral:
		n.Contents = p.lowerExprs(n.Contents)
		return n

	case ast.RangeExpr:
		n.Lower = p.lowerExpr(n.Lower)
		n.Upper = p.lowerExpr(n.Upper)
		return n

	case ast.NewExpr:
		n.Instantiation.Method = p.lowerExpr(n.Instantiation.Method)
		n.Instantiation.Arguments = p.lowerExprs(n.Instantiation.Arguments)
		return n

	case ast.LetExpr:
		n.Stmt_ = p.lowerStatement(n.Stmt_)
		n.Expr_ = p.lowerExpr(n.Expr_)
		return n

	case ast.ForComprehensionExpr:
		n.Source = p.lowerExpr(n.Source)
		n.Yield = p.lowerExpr(n.Yield)
		return n

	default:
		return e
	}
}

// lowerExprs lowers every expression in a list.
func (p *parser) lowerExprs(list []ast.Expr) []ast.Expr {
	if list == nil {
		return nil
	}
	out := make([]ast.Expr, 0, len(list))
	for _, e := range list {
		out = append(out, p.lowerExpr(e))
	}
	return out
}
