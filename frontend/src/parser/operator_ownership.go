package parser

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// rejectOperatorPlacement enforces the closed set of containers that may own an
// operator function. Only a named class, a struct companion unit, and a unit
// function extending a built-in type call the decorated-function path; every
// other declaration body reports the annotation instead of silently retaining
// it on an ordinary function node.
func (p *parser) rejectOperatorPlacement(annotations annotationSet, container string) {
	if annotations.has("@co.dap.operator") {
		p.reportf(p.cur(), "an operator function cannot be declared in %s; declare it in a named class, a struct companion unit, or a built-in extension unit", container)
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
//   - an implicit class instance receiver establishes the class owner; and
//   - a built-in extension uses its fortype as the owner instead of the unit's
//     declaration name.
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
	operatorOwner := owner
	ownerDescription := containerKind
	if operator.IsExtension {
		if containerKind != "unit" {
			p.reportf(p.cur(), "a built-in operator extension must be declared inside a unit, not a %s", containerKind)
			return
		}
		target := logicalTypeName(operator.ForType)
		if target == "" {
			p.reportf(p.cur(), "an operator @co.dap.extension declaration requires one fortype built-in type")
			return
		}
		if !isBuiltinDataTypeName(target) {
			p.reportf(p.cur(), "operator extension target %q is not a built-in type; user-defined struct and class operators belong to their companion unit or class", target)
			return
		}
		operatorOwner = name{Scanned: target, Logical: target}
		ownerDescription = "built-in extension"
	}

	implicitOwner := ast.Type(nil)
	if containerKind == "class" && function.AssociatedReceiver == nil &&
		function.Symb != nil && function.Symb.InstanceMethod &&
		!function.Symb.StaticMethod && !function.Symb.ClassMethod && !function.Symb.ObjectMethod {
		// A receiverless class instance method has the class's implicit `this`
		// operand. It establishes ownership without forcing the first explicit
		// parameter to repeat the class type.
		implicitOwner = operatorOwnerTypeNode(operatorOwner)
	}
	if !p.validateOperatorArity(operator, implicitOwner) {
		return
	}
	defer p.validateDuplicateOperatorSignature(operator, implicitOwner)

	if function.AssociatedReceiver != nil {
		actual := symbolDeclarationTypeNode(function.AssociatedReceiver.SymbolStmt)
		if !operatorOwnerTypeMatches(actual, operatorOwner) {
			p.reportf(
				p.cur(),
				"operator function %q has receiver type %q, but an operator in %s %q requires receiver type %q",
				logicalName(function.Name),
				logicalTypeName(actual),
				ownerDescription,
				operatorOwner.Logical,
				operatorOwner.Logical,
			)
		}
		return
	}
	if implicitOwner != nil {
		return
	}

	if len(function.Parameters) == 0 || len(function.Parameters[0]) == 0 {
		p.reportf(
			p.cur(),
			"receiverless operator function %q in %s %q requires a first parameter of type %q",
			logicalName(function.Name),
			ownerDescription,
			operatorOwner.Logical,
			operatorOwner.Logical,
		)
		return
	}

	actual := parsedTypeName(function.Parameters[0][0].Type_)
	if !operatorOwnerTypeMatches(actual, operatorOwner) {
		p.reportf(
			p.cur(),
			"receiverless operator function %q has first parameter type %q, but an operator in %s %q requires its first parameter to have type %q",
			logicalName(function.Name),
			logicalTypeName(actual),
			ownerDescription,
			operatorOwner.Logical,
			operatorOwner.Logical,
		)
	}
}

// validateDuplicateOperatorSignature collapses receiver ownership syntax to the
// operands a call sees. Consequently an explicit or implicit instance receiver
// plus its ordinary parameters and the equivalent receiverless-first-operand
// shorthand acquire the same key and cannot both be declared.
func (p *parser) validateDuplicateOperatorSignature(operator ast.OperatorStmt, implicitOwner ast.Type) {
	function := operator.FunctionDeclarationStmt
	symbol := operatorSymbolFromFunction(function)
	if symbol == "" {
		return
	}

	key := normalizedOperatorSignature(function, symbol, implicitOwner)
	if _, duplicate := p.operatorSignatures[key]; duplicate {
		p.reportf(p.cur(), "operator %q duplicates an equivalent operator signature already declared in this container", symbol)
		return
	}
	p.operatorSignatures[key] = p.cur()
}

// validateOperatorArity checks the callable operands after ownership
// normalization. An explicit instance receiver and an implicit class `this`
// each contribute one operand; a type receiver contributes none. Symbols such
// as + and - admit both their unary and binary language-owned forms, while a
// registered custom symbol admits only its source-declared arity.
func (p *parser) validateOperatorArity(operator ast.OperatorStmt, implicitOwner ast.Type) bool {
	symbol := operatorSymbolFromFunction(operator.FunctionDeclarationStmt)
	if symbol == "" {
		return true
	}

	operandCount := len(normalizedOperatorOperands(operator.FunctionDeclarationStmt, implicitOwner))
	var allowed []int
	if syntax, custom := p.ops.syntax[symbol]; custom {
		if arity, ok := operatorArityCount(syntax.arity); ok {
			allowed = []int{arity}
		}
	} else if isBuiltinOperatorSymbol(symbol) {
		allowed = builtinOperatorArities(symbol)
	} else {
		// Language-predeclared glyph arities come from their immutable language
		// parse table. Until that table is exposed to this package, registration
		// acceptance is still correct and the semantic resolver owns the check.
		return true
	}
	for _, arity := range allowed {
		if operandCount == arity {
			return true
		}
	}

	want := make([]string, len(allowed))
	for index, arity := range allowed {
		want[index] = strconv.Itoa(arity)
	}
	p.reportf(
		p.cur(),
		"operator %q has %d normalized operands, but its registered callable arity is %s",
		symbol,
		operandCount,
		strings.Join(want, " or "),
	)
	return false
}

func builtinOperatorArities(symbol string) []int {
	var arities []int
	if _, prefix := prefixOperators[symbol]; prefix {
		arities = append(arities, 1)
	} else if _, postfix := postfixOperators[symbol]; postfix {
		arities = append(arities, 1)
	}
	if _, infix := builtinInfixOperators[symbol]; infix {
		arities = append(arities, 2)
	}
	return arities
}

// normalizedOperatorSignature returns the overload-uniqueness key of an
// operator while
// erasing the equivalent ownership spellings:
//
//	(emp Employee) add(other Employee)          instance receiver
//	add(emp Employee, other Employee)           receiverless shorthand
//	add(other Employee)                         implicit class receiver
//
//	(Employee) equals(left Employee, right Employee)   type receiver
//	equals(left Employee, right Employee)             receiverless shorthand
//
// The two receiver forms differ in what the receiver MEANS, which is why only one of
// them contributes an operand. An INSTANCE receiver binds a name and is the first
// operand, so `(emp Employee) add(other)` takes the same two operands as
// `add(emp, other)`. A TYPE receiver binds nothing and marks ownership only, so
// `(Employee) equals(left, right)` takes the two operands its parameters already
// spell — which is precisely the reference's rule that a receiverless operator whose
// first parameter is the owner type is shorthand for the TYPE-associated form.
//
// Counting a type receiver as an operand gave it one more operand than its own
// shorthand, so the two spellings the reference calls equivalent never collided and
// the duplicate went unreported. Parameter names, parameter-list grouping, and
// result types are not part of the duplicate key: overloads are distinguished by
// symbol plus normalized operand types, never by return type alone.
func normalizedOperatorSignature(function ast.FunctionDeclarationStmt, symbol string, implicitOwner ast.Type) string {
	operandTypes := normalizedOperatorOperands(function, implicitOwner)
	operands := make([]string, 0, len(operandTypes))
	for _, operand := range operandTypes {
		operands = append(operands, typeFingerprint(operand))
	}

	return fingerprintParts("operator", symbol, fingerprintParts("operands", operands...))
}

func normalizedOperatorOperands(function ast.FunctionDeclarationStmt, implicitOwner ast.Type) []ast.Type {
	operands := make([]ast.Type, 0)
	if implicitOwner != nil {
		operands = append(operands, implicitOwner)
	}
	if instanceReceiverType(function.AssociatedReceiver) != nil {
		operands = append(operands, receiverTypeNode(function.AssociatedReceiver))
	}
	for _, group := range function.Parameters {
		for _, parameter := range group {
			operands = append(operands, parameter.Type_)
		}
	}
	return operands
}

// operatorOwnerTypeNode builds the same simple type node the type parser would
// produce for an enclosing owner. It is used only for signature normalization of
// an implicit class receiver, so no symbol-table registration is needed.
func operatorOwnerTypeNode(owner name) ast.Type {
	if isBuiltinDataTypeName(owner.Logical) {
		return ast.BuiltInDataType{
			Value:      owner.Scanned,
			Type:       owner.Scanned,
			SymbolType: string(symboltable.S_TypeSymbol),
		}
	}
	return ast.SymbolTypeNode{
		Value:      owner.Scanned,
		SymbolType: string(symboltable.S_TypeSymbol),
	}
}

// isBuiltinDataTypeName uses the scanner's canonical table so ownership checks
// cannot drift from the set of names tokenized as built-in types.
func isBuiltinDataTypeName(typeName string) bool {
	typeName = logicalTypeName(typeName)
	for _, builtin := range scanlex.Builtin_types {
		if typeName == builtin {
			return true
		}
	}
	return false
}

// instanceReceiverType returns the receiver type when the receiver is an INSTANCE
// receiver, and nil for a type receiver or no receiver at all.
//
// Both spellings lower to a VarDeclarationStmt and only the instance form binds a
// name, so the binder is what separates `(emp Employee)` — an operand — from
// `(Employee)` — ownership only.
func instanceReceiverType(receiver *ast.FunctionReceiver) ast.Type {
	if receiver == nil {
		return nil
	}
	variable, ok := receiver.SymbolStmt.(ast.VarDeclarationStmt)
	if !ok || logicalName(variable.Identifier) == "" {
		return nil
	}
	return variable.Type_
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
