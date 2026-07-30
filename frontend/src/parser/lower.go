package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"
)

// Control-flow lowering — the pass that turns canonical chains into dedicated AST nodes.
//
// Parsing produces a uniform postfix chain for every control construct, because that is what the
// grammar requires (see the header of chain.go). This pass walks the finished tree and rewrites
// the chains that match a canonical shape from section 11a:
//
//	(c).do({…}).otherwise(c2).do({…}).otherwise.do({…})   -> ConditionalStmt
//	(c).loop({…})                                          -> ConditionalStmt with Loop set
//	(c).return(a).otherwise.return(b)                      -> TernaryStmt
//	arr.each(i, v).do({…})                                 -> ForeachStmt
//	arr.contains(k).do({…}).otherwise.do({…})              -> ConditionalStmt over ContainsStmt
//
// Anything that does not match is left exactly as parsed. That is the important property: the
// pass only ever narrows a generic chain into a more specific node, so an unrecognised
// method chain — a pipeline, a user's own `.do`-named method — keeps working.
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
		n.Body = p.lowerStatements(n.Body)
		return n
	case ast.TypeDeclarationStmt:
		n.Body = p.lowerStatements(n.Body)
		return n
	case ast.ClassDeclarationStmt:
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
	case ast.ExtensionStmt:
		n.FunctionDeclarationStmt.Body = p.lowerStatements(n.FunctionDeclarationStmt.Body)
		return n
	case ast.IndexerStmt:
		n.FunctionDeclarationStmt.Body = p.lowerStatements(n.FunctionDeclarationStmt.Body)
		return n
	case ast.MatcherStmt:
		n.FunctionDeclarationStmt.Body = p.lowerStatements(n.FunctionDeclarationStmt.Body)
		return n
	case ast.GenerricFun:
		n.FunctionDeclarationStmt.Body = p.lowerStatements(n.FunctionDeclarationStmt.Body)
		return n
	case ast.DDapStmt:
		n.FunctionDeclarationStmt.Body = p.lowerStatements(n.FunctionDeclarationStmt.Body)
		return n
	case ast.FunctionPatternStmt:
		n.Body = p.lowerStatements(n.Body)
		if n.BodyExpr != nil {
			n.BodyExpr = p.lowerExpr(n.BodyExpr)
		}
		return n

	// A declaration's initializer can contain a chain, as in an anonymous function body.
	case ast.VarDeclarationStmt:
		if n.AssignedValue != nil {
			n.AssignedValue = p.lowerExpr(n.AssignedValue)
		}
		return n

	default:
		return s
	}
}

// lowerControlChain attempts every canonical shape against an expression.
//
// The order matters only in that each matcher is keyed by its own leading verb, so at most one can
// match: `.each` and `.contains` name the iterator and containment forms, `.return` the ternary,
// and `.do`/`.loop` the condition and loop forms.
func (p *parser) lowerControlChain(e ast.Expr) (ast.Stmt, bool) {
	c, ok := decomposeChain(e)
	if !ok {
		return nil, false
	}

	switch {
	case c.verbAt(0) == verbEach:
		return p.lowerEachChain(c)
	case containsVerbs[c.verbAt(0)]:
		return p.lowerContainsChain(c)
	case c.verbAt(0) == verbReturn:
		return p.lowerTernaryChain(c)
	case isBranchVerb(c.verbAt(0)):
		return p.lowerConditionalChain(c)
	}
	return nil, false
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

	case ast.AssignmentExpr:
		n.Assigne = p.lowerExpr(n.Assigne)
		n.AssignedValue = p.lowerExpr(n.AssignedValue)
		return n

	case ast.PrefixExpr:
		n.Right = p.lowerExpr(n.Right)
		return n

	case ast.FunctionExpr:
		n.Body = p.lowerStatements(n.Body)
		return n

	case ast.LambdaExpr:
		n.Body = p.lowerExpr(n.Body)
		return n

	case ast.ArrayLiteral:
		n.Contents = p.lowerExprs(n.Contents)
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
