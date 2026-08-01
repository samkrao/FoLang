package parser

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/samkrao/fo-lang/frontend/src/ast"
)

// rejectOperatorPlacement enforces the closed set of containers that may own an
// operator function. Only a named class and a struct companion unit call the
// decorated-function path; every other declaration body reports the annotation
// instead of silently retaining it on an ordinary function node.
func (p *parser) rejectOperatorPlacement(annotations annotationSet, container string) {
	if annotations.has("@co.dap.operator") {
		p.reportf(p.cur(), "an operator function cannot be declared in %s; declare it in a named class or a struct companion unit", container)
	}
}

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
	defer p.validateDuplicateOperatorSignature(operator)
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

// validateDuplicateOperatorSignature collapses receiver ownership syntax to the
// operands a call sees. Consequently an instance receiver plus its ordinary
// parameters and the equivalent receiverless-first-operand shorthand acquire the
// same key and cannot both be declared.
func (p *parser) validateDuplicateOperatorSignature(operator ast.OperatorStmt) {
	function := operator.FunctionDeclarationStmt
	symbol := operatorSymbolFromFunction(function)
	if symbol == "" {
		return
	}

	key := normalizedOperatorSignature(function, symbol)
	if _, duplicate := p.operatorSignatures[key]; duplicate {
		p.reportf(p.cur(), "operator %q duplicates an equivalent operator signature already declared in this container", symbol)
		return
	}
	p.operatorSignatures[key] = p.cur()
}

// normalizedOperatorSignature returns the callable shape of an operator while
// erasing the three equivalent ownership spellings:
//
//	(emp Employee) add(other Employee)
//	(Employee) add(other Employee)
//	add(emp Employee, other Employee)
//
// A receiver contributes the first operand regardless of whether it has a
// binder. Receiverless parameters then naturally produce the same flattened
// operand sequence. Parameter names and parameter-list grouping are not part of
// an operator overload signature; operand and result types are.
func normalizedOperatorSignature(function ast.FunctionDeclarationStmt, symbol string) string {
	operands := make([]string, 0)
	if function.AssociatedReceiver != nil {
		operands = append(operands, typeFingerprint(receiverTypeNode(function.AssociatedReceiver)))
	}
	for _, group := range function.Parameters {
		for _, parameter := range group {
			operands = append(operands, typeFingerprint(parameter.Type_))
		}
	}

	results := make([]string, 0, len(function.ReturnType))
	for _, result := range function.ReturnType {
		results = append(results, typeFingerprint(result.Type_))
	}
	return fingerprintParts("operator", symbol, fingerprintParts("operands", operands...), fingerprintParts("results", results...))
}

// receiverTypeNode extracts the complete receiver type without depending on a
// value-receiver binder. Both `(emp Employee)` and `(Employee)` lower to a
// VarDeclarationStmt; only the former has a non-empty Identifier.
func receiverTypeNode(receiver *ast.FunctionReceiver) ast.Type {
	if receiver == nil {
		return nil
	}
	if variable, ok := receiver.SymbolStmt.(ast.VarDeclarationStmt); ok {
		return variable.Type_
	}
	return nil
}

func operatorSymbolFromFunction(function ast.FunctionDeclarationStmt) string {
	list, ok := function.Dapst.(*ast.DirectveList)
	if !ok || list == nil {
		return ""
	}
	for _, group := range list.GetDap() {
		for _, statement := range group {
			if directive, ok := statement.(ast.DirectiveStmt); ok && directive.Name == "@co.dap.operator" {
				return operatorOptionText(directive.Parameters, "symbol")
			}
		}
	}
	return ""
}

func typeFingerprint(node ast.Type) string {
	switch node := node.(type) {
	case nil:
		return fingerprintParts("nil")
	case ast.SymbolTypeNode:
		return fingerprintParts("symbol", logicalTypeName(node.Value), node.SymbolType)
	case ast.BuiltInDataType:
		return fingerprintParts("builtin", logicalTypeName(node.Value), logicalTypeName(node.Type), node.SymbolType)
	case ast.CompoundType:
		return fingerprintParts("compound", node.Op, typeFingerprint(node.Left), typeFingerprint(node.Right))
	case ast.ListType:
		return fingerprintParts("list", typeFingerprint(node.Underlying))
	case ast.FunctionType:
		parameterGroups := make([]string, 0, len(node.Params))
		for _, group := range node.Params {
			parameters := make([]string, 0, len(group))
			for _, parameter := range group {
				parameters = append(parameters, typeFingerprint(parameter.Type_))
			}
			parameterGroups = append(parameterGroups, fingerprintParts("params", parameters...))
		}
		results := make([]string, 0, len(node.Results))
		for _, result := range node.Results {
			results = append(results, typeFingerprint(result.Type_))
		}
		return fingerprintParts(
			"function",
			fingerprintParts("groups", parameterGroups...),
			fingerprintParts("results", results...),
		)
	case ast.GenericType:
		return fingerprintParts("generic", typeFingerprint(node.Type_), typeFingerprint(node.Constraint))
	case ast.ForAllType:
		parameters := make([]string, 0, len(node.TypeParams))
		for _, parameter := range node.TypeParams {
			parameters = append(parameters, fingerprintParts(
				"type-parameter",
				logicalName(parameter.Name),
				logicalTypeName(parameter.Constraint),
				parameter.Variance,
				parameter.Kind_,
				logicalTypeName(parameter.Default),
				strconv.FormatBool(parameter.Nullable),
				strconv.FormatBool(parameter.Inclusive),
				strconv.FormatBool(parameter.Impredicative),
				parameter.TypeKind,
				parameter.Types,
			))
		}
		return fingerprintParts("forall", fingerprintParts("parameters", parameters...), typeFingerprint(node.Inner))
	case ast.DependentType:
		return fingerprintParts("dependent", typeFingerprint(node.Base), expressionFingerprint(node.Expr))
	case ast.DerivedType:
		groups := make([]string, 0, len(node.DimGroups))
		for _, group := range node.DimGroups {
			dimensions := make([]string, 0, len(group))
			for _, dimension := range group {
				dimensions = append(dimensions, expressionFingerprint(dimension))
			}
			groups = append(groups, fingerprintParts("dimensions", dimensions...))
		}
		return fingerprintParts(
			"derived",
			typeFingerprint(node.Underlying),
			string(node.Form),
			strconv.Itoa(node.PointerCount),
			strconv.Itoa(node.RefCount),
			fingerprintParts("dimension-groups", groups...),
			strconv.FormatBool(node.VariableLength),
			strconv.FormatBool(node.ZeroDim),
			valueFingerprint(node.Attrs),
		)
	default:
		// ast.Type is deliberately closed inside the ast package. Keep a
		// type-qualified fallback so a future node cannot alias an existing
		// fingerprint while this switch is being extended.
		return fingerprintParts("unknown-type", fmt.Sprintf("%T", node), logicalTypeName(typeNameOf(node)))
	}
}

// fingerprintParts length-prefixes every component, making the structural
// representation unambiguous even when a type or attribute contains a delimiter.
func fingerprintParts(tag string, parts ...string) string {
	var out strings.Builder
	out.WriteString(tag)
	out.WriteByte('{')
	for _, part := range parts {
		out.WriteString(strconv.Itoa(len(part)))
		out.WriteByte(':')
		out.WriteString(part)
	}
	out.WriteByte('}')
	return out.String()
}

// expressionFingerprint covers the closed dependent-index grammar used inside
// types and array dimensions. These expressions are restricted to non-negative
// integer literals and qualified names, but grouping is retained defensively.
func expressionFingerprint(expression ast.Expr) string {
	switch expression := expression.(type) {
	case nil:
		return fingerprintParts("nil-expression")
	case ast.IntegerLiteral:
		return fingerprintParts("integer", strconv.FormatInt(expression.Value, 10), expression.Type_, expression.ActType_)
	case ast.NumberLiteral:
		return fingerprintParts("number", strconv.FormatFloat(expression.Value, 'g', -1, 64), expression.Type_, expression.ActType_)
	case ast.SymbolExpr:
		return fingerprintParts("name", logicalTypeName(expression.Value), expression.SymbolType_)
	case ast.GroupingExpr:
		return fingerprintParts("group", expressionFingerprint(expression.Expr_))
	default:
		return fingerprintParts("unknown-expression", fmt.Sprintf("%T", expression))
	}
}

// valueFingerprint serializes derivation attributes without map iteration
// order. Type and expression values recurse through the same canonical rules;
// scalar option values retain both their dynamic type and value.
func valueFingerprint(value any) string {
	switch value := value.(type) {
	case nil:
		return fingerprintParts("nil-value")
	case ast.Type:
		return typeFingerprint(value)
	case ast.Expr:
		return expressionFingerprint(value)
	case map[string]any:
		keys := make([]string, 0, len(value))
		for key := range value {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		entries := make([]string, 0, len(keys))
		for _, key := range keys {
			entries = append(entries, fingerprintParts("entry", key, valueFingerprint(value[key])))
		}
		return fingerprintParts("map", entries...)
	case []any:
		items := make([]string, len(value))
		for i, item := range value {
			items[i] = valueFingerprint(item)
		}
		return fingerprintParts("list", items...)
	case []ast.Expr:
		items := make([]string, len(value))
		for i, item := range value {
			items[i] = expressionFingerprint(item)
		}
		return fingerprintParts("expressions", items...)
	case string:
		return fingerprintParts("string", value)
	case bool:
		return fingerprintParts("bool", strconv.FormatBool(value))
	case int:
		return fingerprintParts("int", strconv.Itoa(value))
	case int64:
		return fingerprintParts("int64", strconv.FormatInt(value, 10))
	case float64:
		return fingerprintParts("float64", strconv.FormatFloat(value, 'g', -1, 64))
	default:
		return fingerprintParts("scalar", fmt.Sprintf("%T", value), fmt.Sprint(value))
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
	// One naming rule for the whole parser. This used to be a second, near-identical
	// copy that disagreed with typeRef.actType about which half of GetActType is the
	// name, so an operator's owner and the same type's symbol metadata could be
	// recorded under different strings.
	return typeNameOf(node)
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
