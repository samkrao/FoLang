package ast

// Marker methods completing the control-flow node set.
//
// A node joins the tree by implementing the unexported marker of the interface it belongs to:
// stmt() for Stmt, expr() for Expr. ContainsStmt declares GetName, GetSymbolType, SetDap and
// Visit, and Treevistor is prepared to walk it, but it never declared stmt() — so it satisfied
// neither interface and could not actually be stored anywhere in a tree.
//
// The omission looks accidental rather than deliberate: every sibling control-flow node
// (ConditionalStmt, DefaultConditionalStmt, TernaryStmt, ForeachStmt) declares stmt(), and
// ConditionalStmt carries an ISParentArrCont flag whose only purpose is to mark a conditional
// whose test is a containment check. The method is supplied here rather than edited into
// statements.go so the original declaration stays untouched.
func (n ContainsStmt) stmt() {}
