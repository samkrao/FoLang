package parser_test

import (
	"testing"

	"github.com/samkrao/fo-lang/frontend/src/ast"
)

func TestClassMembersCarryMethodCategories(t *testing.T) {
	body := parseRegressionFile(t, `_ co.lang.class = {
    ordinary(other Employee)->(Employee) = { this.return other; }

    @co.dap.static
    staticMethod(left Employee, right Employee)->(Employee) = { this.return left; }

    @co.dap.class
    classMethod(left Employee)->(Employee) = { this.return left; }

    @co.dap.instance
    instanceMethod(other Employee)->(Employee) = { this.return other; }

    @co.dap.object
    objectMethod(value Employee)->(Employee) = { this.return value; }

    (Employee) typeReceiver(value Employee)->(Employee) = { this.return value; }
}`, "Employee.fol")

	class, ok := body[0].(ast.ClassDeclarationStmt)
	if !ok {
		t.Fatalf("declaration is %T, want ast.ClassDeclarationStmt", body[0])
	}

	tests := []struct {
		index                           int
		name                            string
		class, static, instance, object bool
	}{
		{0, "ordinary", false, false, true, false},
		{1, "staticMethod", false, true, false, false},
		{2, "classMethod", true, false, false, false},
		{3, "instanceMethod", false, false, true, false},
		{4, "objectMethod", false, false, false, true},
		{5, "typeReceiver", true, false, false, false},
	}
	for _, tc := range tests {
		function, ok := class.Body[tc.index].(ast.FunctionDeclarationStmt)
		if !ok {
			t.Fatalf("member %q is %T, want ast.FunctionDeclarationStmt", tc.name, class.Body[tc.index])
		}
		if !function.Symb.IsMethod {
			t.Errorf("member %q is not marked as a method", tc.name)
		}
		if function.Symb.ClassMethod != tc.class || function.Symb.StaticMethod != tc.static ||
			function.Symb.InstanceMethod != tc.instance || function.Symb.ObjectMethod != tc.object {
			t.Errorf(
				"member %q categories = class:%t static:%t instance:%t object:%t",
				tc.name,
				function.Symb.ClassMethod,
				function.Symb.StaticMethod,
				function.Symb.InstanceMethod,
				function.Symb.ObjectMethod,
			)
		}
	}
}

func TestImplicitClassOperatorReceiverParticipatesInDuplicateSignature(t *testing.T) {
	mustNotPanic(t, func() {
		parseEmployeeClass(t, `_ co.lang.class = {
    @co.dap.operator(symbol='+')
    add(other Employee)->(Employee) = { this.return other; }
}`)
	})

	mustPanic(t, func() {
		parseEmployeeClass(t, `_ co.lang.class = {
    @co.dap.operator(symbol='+')
    addImplicit(other Employee)->(Employee) = { this.return other; }

    @co.dap.operator(symbol='+', mode=overload)
    (emp Employee) addExplicit(other Employee)->(Employee) = { this.return emp; }
}`)
	})
}

func TestDuplicateOperatorAnnotationsAreRejected(t *testing.T) {
	mustPanic(t, func() {
		parseEmployeeClass(t, `_ co.lang.class = {
    @co.dap.operator(symbol='+')
    @co.dap.operator(symbol='-')
    add(other Employee)->(Employee) = { this.return other; }
}`)
	})
}

func TestClassOperatorRejectsConflictingOrDuplicateMethodCategories(t *testing.T) {
	tests := []struct {
		name        string
		annotations string
		parameters  string
		returned    string
	}{
		{"static-instance", "@co.dap.static\n    @co.dap.instance", "other Employee", "other"},
		{"class-instance", "@co.dap.class\n    @co.dap.instance", "other Employee", "other"},
		{"object-instance", "@co.dap.object\n    @co.dap.instance", "other Employee", "other"},
		{"duplicate-static", "@co.dap.static\n    @co.dap.static", "left Employee, right Employee", "left"},
		{"duplicate-class", "@co.dap.class\n    @co.dap.class", "left Employee, right Employee", "left"},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			mustPanic(t, func() {
				parseEmployeeClass(t, `_ co.lang.class = {
    @co.dap.operator(symbol='+')
    `+test.annotations+`
    add(`+test.parameters+`)->(Employee) = { this.return `+test.returned+`; }
}`)
			})
		})
	}
}

func TestBuiltInOperatorCallableArityUsesNormalizedOperands(t *testing.T) {
	mustNotPanic(t, func() {
		parseEmployeeClass(t, `_ co.lang.class = {
    @co.dap.operator(symbol='-')
    negate()->(Employee) = { this.return this; }

    @co.dap.operator(symbol="==")
    @co.dap.static
    equals(left Employee, right Employee)->(co.lang.bool) = { this.return co.const.true; }
}`)
	})

	for _, test := range []struct {
		basename string
		source   string
	}{
		{"Employee.fol", `_ co.lang.class = {
    @co.dap.operator(symbol='+')
    add(left Employee, right Employee)->(Employee) = { this.return left; }
}`},
		{"Employee.comp.unit.fol", `_ co.lang.unit = {
    @co.dap.operator(symbol='!')
    negate(left Employee, right Employee)->(co.lang.bool) = { this.return co.const.false; }
}`},
	} {
		test := test
		mustPanic(t, func() {
			parseRegressionFile(t, test.source, test.basename)
		})
	}
}

func TestOperatorModesAreClosedBySymbolKind(t *testing.T) {
	// A one-rune symbol may use either the canonical character spelling or a
	// string spelling; multi-rune symbols require a string.
	mustNotPanic(t, func() {
		parseEmployeeClass(t, `_ co.lang.class = {
    @co.dap.operator(symbol="+")
    add(other Employee)->(Employee) = { this.return other; }
}`)
	})

	builtinAccepted := []string{"", "mode=overload"}
	for _, options := range builtinAccepted {
		options := options
		if options != "" {
			options = ", " + options
		}
		t.Run("builtin-accepted-"+options, func(t *testing.T) {
			mustNotPanic(t, func() {
				parseEmployeeClass(t, `_ co.lang.class = {
    @co.dap.operator(symbol='+'`+options+`)
    add(other Employee)->(Employee) = { this.return other; }
}`)
			})
		})
	}

	for _, mode := range []string{"define", "extends", "override", "unknown"} {
		mode := mode
		t.Run("builtin-rejects-"+mode, func(t *testing.T) {
			mustPanic(t, func() {
				parseEmployeeClass(t, `_ co.lang.class = {
    @co.dap.operator(symbol='+', mode=`+mode+`)
    add(other Employee)->(Employee) = { this.return other; }
}`)
			})
		})
	}

	// Custom spellings are registered only by components/operators/component.fol
	// bootstrap. An ordinary implementation without that catalog is rejected,
	// and implementation annotations cannot repeat source parse properties.
	for _, options := range []string{
		`symbol="<+>"`,
		`symbol="<+>", mode=overload`,
		`symbol="<+>", mode=define`,
		`symbol="<+>", fixity=infix, precedence=55, associativity=left, arity=binary`,
	} {
		options := options
		t.Run("custom-without-bootstrap-rejected-"+options, func(t *testing.T) {
			mustPanic(t, func() {
				parseCompanionUnit(t, "Vector", `_ co.lang.unit = {
	@co.dap.operator(`+options+`)
    merge(left Vector, right Vector)->(Vector) = { this.return left; }
}`)
			})
		})
	}
}

func TestBuiltInOperatorExtensionRetainsItsOwner(t *testing.T) {
	body := parseCompanionUnit(t, "Strings", `_ co.lang.unit = {
    @co.dap.operator(symbol='+')
    @co.dap.extension(fortype=co.lang.string, what=extends)
    concat(left co.lang.string, right co.lang.string)->(co.lang.string) = { this.return left; }
}`)

	unit, ok := body[0].(ast.TypeDeclarationStmt)
	if !ok || len(unit.Body) != 1 {
		t.Fatalf("unit declaration is %T with an unexpected body", body[0])
	}
	operator, ok := unit.Body[0].(ast.OperatorStmt)
	if !ok {
		t.Fatalf("unit member is %T, want ast.OperatorStmt", unit.Body[0])
	}
	if !operator.IsExtension || operator.ForType != "co.lang.string" || operator.What != "extends" {
		t.Fatalf("operator extension metadata = extension:%t fortype:%q what:%q", operator.IsExtension, operator.ForType, operator.What)
	}
}

func TestBuiltInOperatorExtensionRequiresOneBuiltInOwner(t *testing.T) {
	for _, test := range []struct {
		basename string
		source   string
	}{
		// No fortype at all.
		{"Strings.comp.unit.fol", `_ co.lang.unit = {
    @co.dap.operator(symbol='+')
    @co.dap.extension(what=extends)
    concat(left co.lang.string, right co.lang.string)->(co.lang.string) = { this.return left; }
}`},
		// A user-defined fortype: an extension target must be a built-in type.
		{"Employee.comp.unit.fol", `_ co.lang.unit = {
    @co.dap.operator(symbol='+')
    @co.dap.extension(fortype=Employee, what=extends)
    add(left Employee, right Employee)->(Employee) = { this.return left; }
}`},
		// An extension must be declared in a unit, not a class.
		{"Employee.fol", `_ co.lang.class = {
    @co.dap.operator(symbol='+')
    @co.dap.extension(fortype=co.lang.string, what=extends)
    add(other Employee)->(Employee) = { this.return other; }
}`},
	} {
		test := test
		mustPanic(t, func() {
			parseRegressionFile(t, test.source, test.basename)
		})
	}
}

// parseEmployeeClass parses a class declaration as the file-backed primary of
// Employee.fol, which is where its name comes from (DECISION-FILE-001).
func parseEmployeeClass(t *testing.T, source string) []ast.Stmt {
	t.Helper()
	return parseRegressionFile(t, source, "Employee.fol")
}

// parseCompanionUnit parses a unit declaration as owner's companion unit. An
// operator implementation needs the companion form, because DECISION-COMP-001
// takes the operand owner from the companion filename and an ordinary unit
// fragment owns nothing.
func parseCompanionUnit(t *testing.T, owner, source string) []ast.Stmt {
	t.Helper()
	return parseRegressionFile(t, source, owner+".comp.unit.fol")
}
