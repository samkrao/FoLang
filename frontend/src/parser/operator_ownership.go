package parser

import (
	"strings"

	"github.com/samkrao/fo-lang/frontend/src/ast"
)

// validateOperatorOwnership enforces the declaration-local half of operator
// ownership for a named unit or class.
//
// The parser can decide this part without resolving any package symbols because
// it knows both the enclosing container name and the complete function shape:
//
//   - an explicit value receiver `(emp Employee)` must have the owner type;
//   - an explicit type receiver `(Employee)` must have the owner type; or
//   - a receiverless operator's first ordinary parameter must have the owner
//     type. A matching later parameter is deliberately insufficient.
//
// Whether a same-named unit really has a same-package struct, whether a class or
// struct declaration is unique, and whether an operator signature is duplicated
// require the package declaration set and remain second-pass checks.
func (p *parser) validateOperatorOwnership(stmt ast.Stmt, owner name, containerKind string) {
	operator, ok := stmt.(ast.OperatorStmt)
	if !ok {
		return
	}

	function := operator.FunctionDeclarationStmt
	if function.AssociatedReceiver != nil {
		actual := symbolDeclarationTypeNode(function.AssociatedReceiver.SymbolStmt)
		if !operatorOwnerTypeMatches(actual, owner) {
			p.reportf(
				p.cur(),
				"operator function %q has receiver type %q, but an operator in %s %q requires receiver type %q",
				logicalName(function.Name),
				logicalTypeName(actual),
				containerKind,
				owner.Logical,
				owner.Logical,
			)
		}
		return
	}

	if len(function.Parameters) == 0 || len(function.Parameters[0]) == 0 {
		p.reportf(
			p.cur(),
			"receiverless operator function %q in %s %q requires a first parameter of type %q",
			logicalName(function.Name),
			containerKind,
			owner.Logical,
			owner.Logical,
		)
		return
	}

	actual := parsedTypeName(function.Parameters[0][0].Type_)
	if !operatorOwnerTypeMatches(actual, owner) {
		p.reportf(
			p.cur(),
			"receiverless operator function %q has first parameter type %q, but an operator in %s %q requires its first parameter to have type %q",
			logicalName(function.Name),
			logicalTypeName(actual),
			containerKind,
			owner.Logical,
			owner.Logical,
		)
	}
}

// symbolDeclarationTypeNode returns the spelling from a receiver's parsed type
// node. Receivers are represented as VarDeclarationStmt by parseReceiverClause.
func symbolDeclarationTypeNode(declaration ast.SymbolDeclStmt) string {
	if declaration == nil {
		return ""
	}
	if variable, ok := declaration.(ast.VarDeclarationStmt); ok {
		return parsedTypeName(variable.Type_)
	}
	actual, _ := declaration.GetActType()
	return actual
}

// parsedTypeName extracts the source type rather than the AST's broad category
// label. In particular, SymbolTypeNode.GetActType returns ("Employee_fo",
// "Type"), and ownership needs the first value, not the category "Type".
func parsedTypeName(node ast.Type) string {
	switch node := node.(type) {
	case ast.SymbolTypeNode:
		return node.Value
	case ast.BuiltInDataType:
		return node.Value
	case ast.GenericType:
		return parsedTypeName(node.Type_)
	case nil:
		return ""
	default:
		actual, declared := node.GetActType()
		if actual != "" {
			return actual
		}
		return declared
	}
}

// operatorOwnerTypeMatches compares a parsed type with the exact enclosing
// declaration name. It removes backend lowering only; it does not shorten a
// qualified name, because an imported `other.Employee` must not establish
// ownership for the local `Employee` container.
func operatorOwnerTypeMatches(actual string, owner name) bool {
	return logicalTypeName(actual) == owner.Logical
}

// logicalTypeName removes the backend suffix from every segment of a type name
// while preserving its qualification.
func logicalTypeName(actual string) string {
	parts := strings.Split(actual, ".")
	for i, part := range parts {
		parts[i] = logicalName(part)
	}
	return strings.Join(parts, ".")
}
