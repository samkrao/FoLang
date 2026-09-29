package ast

import "testing"

func TestSpanContainsUsesHalfOpenEnd(t *testing.T) {
	span := Span{
		Start: Position{Offset: 2, Line: 1, Column: 3},
		End:   Position{Offset: 7, Line: 1, Column: 8},
	}

	if !span.Contains(1, 3) || !span.Contains(1, 7) {
		t.Fatal("span does not contain its start or last included column")
	}
	if span.Contains(1, 8) {
		t.Fatal("half-open span contains its excluded end")
	}
}

func TestCoreNodesSatisfyTheirCategories(t *testing.T) {
	var _ SET = SourceFile{}
	var _ SET = Block{}
	var _ Def = Declaration{}
	var _ Def = ValueDefinition{}
	var _ Def = TypeDefinition{}
	var _ Def = ContextDefinition{}
	var _ Def = CallableDefinition{}
	var _ Expr = LiteralExpr{}
	var _ Expr = CallExpr{}
	var _ Expr = AssignmentExpr{}
	var _ Stmt = MultipleAssignmentStatement{}
	var _ Stmt = ReturnStatement{}
	var _ Type = NamedType{}
	var _ Type = DerivedType{}
	var _ Pattern = BindingPattern{}
	var _ Pattern = ConstructorPattern{}
}
