package ast

func (tcs TypeclassStmt) visit(t any) SET {
	node := t.(SET)

	return node
}
func (od ObjectDeclStmt) visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n DirectveList) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n UseStmtDirective) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (d DummyStmt) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Treevistor visits a statement AST node and returns the corresponding MIR node.
func Treevistor(node SET) SET {
	return node.Visit(node)
}

// Visit converts the AST node to a MIR node.
func (n TypeDeclarationStmt) Visit(t any) SET {
	node := t.(SET)

	return node
}

func (n RefinementTypeDeclarationStmt) Visit(t any) SET { return t.(SET) }
func (n PredicateTypeDeclarationStmt) Visit(t any) SET  { return t.(SET) }
func (n DependentTypeDeclarationStmt) Visit(t any) SET  { return t.(SET) }

// Visit converts the AST node to a MIR node.
func (n ClassDeclarationStmt) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n TypeConstructorStmt) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n ExpressionStmt) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n AddressVariableDeclStmt) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n RefVariableDeclStmt) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n PointerVariableDeclStmt) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n SliceVariableDeclStmt) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (th ThunkVariableDeclStmt) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n ArrayVariableDeclStmt) Visit(t any) SET {
	node := t.(SET)

	return node

}

func (a Application) Visit(t any) SET {
	node := t.(SET)
	return node
}

// Visit converts the AST node to a MIR node.
func (n VarDeclarationStmt) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (b BlockStmt) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n ArrayLiteral) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n NumberLiteral) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n IntegerLiteral) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n StringLiteral) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n CharacterLiteral) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n ProjectStmt) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n PackageStmt) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n SymbolExpr) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n StatementExpr) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n PrefixExpr) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n AssignmentExpr) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (tr TernaryStmt) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n DefaultConditionalStmt) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n ConditionalStmt) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n ConditionalExpr) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (con ContinueStmt) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (br BreakStmt) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (lbl LabeledStmt) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (ret ReturnStmt) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (g GroupingExpr) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n CommaExpr) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (l LetExpr) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (pst CaseStmt) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (m MatchExprStmt) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (pes PatternExprStmt) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n BinaryExpr) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n SymbolTypeNode) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n SDTExpr) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n BuiltInDataType) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n CompoundType) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n GenericType) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n CallExpr) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n MemberExpr) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n ParentSelectorExpr) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n RelationshipSelectorExpr) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n LockStmt) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n LifecycleCallExpr) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n ForeachStmt) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (m ModuleStmt) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n ComputedExpr) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n NewExpr) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n FunctionExpr) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n RangeExpr) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n LambdaExpr) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n FunctionPatternStmt) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n ListType) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n ForAllType) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n ContainsStmt) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n DirectiveStmt) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n ImportStmt) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n Parameter) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n Returns) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n FunctionType) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n TypeStmt) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (d DelegateStmt) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n FunctionReceiver) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n FunctionDeclarationStmt) Visit(t any) SET {
	node := t.(SET)

	return node
}

// ForComprehensionExpr is lowered to ForComprehensionNode.
func (n ForComprehensionExpr) Visit(t any) SET {
	node := t.(SET)

	return node
}

// ------------------------------------------------------------------ //
// Function-declaration flavors
// Each wrapper extracts the embedded FunctionDeclarationStmt, calls its
// Visit to get a FunctionDeclarationNode, then wraps it in the correct
// MIR specialised node.  Without these explicit methods the promoted
// FunctionDeclarationStmt.Visit would panic because it type-asserts
// `node` as FunctionDeclarationStmt, which fails for wrapper types.
// ------------------------------------------------------------------ //

// Visit converts the AST node to a MIR node.
func (n MacroStmt) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n TemplateStmt) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n OperatorStmt) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n IndexerStmt) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n GenerricFun) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n DecoratorStmt) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n NativeFunctionStmt) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n ExecutionModelFunctionStmt) Visit(t any) SET {
	node := t.(SET)

	return node
}

// ExtensionStmt, LambdaStmt, AnonymousFunctionStmt:
// no dedicated MIR node yet — delegate to the embedded function declaration.

// Visit converts the AST node to a MIR node.
func (n ExtensionStmt) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n ExtensionDeclarationStmt) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n ComponentDeclarationStmt) Visit(t any) SET {
	node := t.(SET)

	return node
}

// ------------------------------------------------------------------ //
// New literal / statement nodes
// ------------------------------------------------------------------ //

// Visit converts the AST node to a MIR node.
func (n BooleanLiteral) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n HeapAllocatedRefStmt) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n RangeVariableDeclStmt) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n BindVariableExpr) Visit(t any) SET {
	node := t.(SET)

	return node
}

// Visit converts the AST node to a MIR node.
func (n DependentType) Visit(t any) SET {
	node := t.(SET)

	return node
}
func (n ObjectDeclStmt) Visit(t any) SET {
	node := t.(SET)
	return node
}
