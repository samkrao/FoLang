package parser_test

import (
	"testing"

	"github.com/samkrao/fo-lang/frontend/src/ast"
)

func TestClassMembersCarryMethodCategories(t *testing.T) {
	body := parseRegressionBody(t, `Employee co.lang.class = {
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
}`)

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
		parseRegressionBody(t, `Employee co.lang.class = {
    @co.dap.operator(symbol='+')
    add(other Employee)->(Employee) = { this.return other; }
}`)
	})

	mustPanic(t, func() {
		parseRegressionBody(t, `Employee co.lang.class = {
    @co.dap.operator(symbol='+')
    addImplicit(other Employee)->(Employee) = { this.return other; }

    @co.dap.operator(symbol='+', mode=overload)
    (emp Employee) addExplicit(other Employee)->(Employee) = { this.return emp; }
}`)
	})
}

func TestBuiltInOperatorCallableArityUsesNormalizedOperands(t *testing.T) {
	mustNotPanic(t, func() {
		parseRegressionBody(t, `Employee co.lang.class = {
    @co.dap.operator(symbol='-')
    negate()->(Employee) = { this.return this; }

    @co.dap.operator(symbol="==")
    @co.dap.static
    equals(left Employee, right Employee)->(co.lang.bool) = { this.return co.const.true; }
}`)
	})

	for _, source := range []string{
		`Employee co.lang.class = {
    @co.dap.operator(symbol='+')
    add(left Employee, right Employee)->(Employee) = { this.return left; }
}`,
		`Employee co.lang.unit = {
    @co.dap.operator(symbol='!')
    negate(left Employee, right Employee)->(co.lang.bool) = { this.return co.const.false; }
}`,
	} {
		source := source
		mustPanic(t, func() {
			parseRegressionBody(t, source)
		})
	}
}

func TestOperatorModesAreClosedBySymbolKind(t *testing.T) {
	builtinAccepted := []string{"", "mode=overload"}
	for _, options := range builtinAccepted {
		options := options
		if options != "" {
			options = ", " + options
		}
		t.Run("builtin-accepted-"+options, func(t *testing.T) {
			mustNotPanic(t, func() {
				parseRegressionBody(t, `Employee co.lang.class = {
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
				parseRegressionBody(t, `Employee co.lang.class = {
    @co.dap.operator(symbol='+', mode=`+mode+`)
    add(other Employee)->(Employee) = { this.return other; }
}`)
			})
		})
	}

	customOptions := `symbol="<+>", fixity=infix, precedence=55, associativity=left, arity=binary`
	mustNotPanic(t, func() {
		parseRegressionBody(t, `Vector co.lang.unit = {
    @co.dap.operator(`+customOptions+`, mode=define)
    merge(left Vector, right Vector)->(Vector) = { this.return left; }
}`)
	})

	for _, suffix := range []string{"", ", mode=overload", ", mode=extends", ", mode=override"} {
		suffix := suffix
		t.Run("custom-rejected-"+suffix, func(t *testing.T) {
			mustPanic(t, func() {
				parseRegressionBody(t, `Vector co.lang.unit = {
    @co.dap.operator(`+customOptions+suffix+`)
    merge(left Vector, right Vector)->(Vector) = { this.return left; }
}`)
			})
		})
	}
}

func TestBuiltInOperatorExtensionRetainsItsOwner(t *testing.T) {
	body := parseRegressionBody(t, `Strings co.lang.unit = {
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
	for _, source := range []string{
		`Strings co.lang.unit = {
    @co.dap.operator(symbol='+')
    @co.dap.extension(what=extends)
    concat(left co.lang.string, right co.lang.string)->(co.lang.string) = { this.return left; }
}`,
		`Employees co.lang.unit = {
    @co.dap.operator(symbol='+')
    @co.dap.extension(fortype=Employee, what=extends)
    add(left Employee, right Employee)->(Employee) = { this.return left; }
}`,
		`Employee co.lang.class = {
    @co.dap.operator(symbol='+')
    @co.dap.extension(fortype=co.lang.string, what=extends)
    add(other Employee)->(Employee) = { this.return other; }
}`,
	} {
		source := source
		mustPanic(t, func() {
			parseRegressionBody(t, source)
		})
	}
}
