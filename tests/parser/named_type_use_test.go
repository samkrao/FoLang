package parser_test

import "testing"

func TestNamedDerivedTypesAreAcceptedAtOrdinaryUseSites(t *testing.T) {
	source := `_ co.unit = {
    Binary co.type = (co.int, co.int)->(co.int);
    IntPtr co.type = co.int->(*);
    TenInts co.type = co.int->([10]);

    apply(operation Binary)->(co.int) = {
        pointer IntPtr;
        values TenInts;
        items co.List(co.int);
        this => operation(10, 20);
    }
}`
	mustNotPanic(t, func() { parseRegressionFile(t, source, "named_types.unit.fol") })
}

func TestInlineDerivedTypesAreRejectedAtOrdinaryUseSites(t *testing.T) {
	tests := []struct {
		name     string
		basename string
		source   string
	}{
		{"function parameter", "inline_types.unit.fol", `_ co.unit = { apply(operation (co.int)->(co.int))->() = {} }`},
		{"function result", "inline_types.unit.fol", `_ co.unit = { make()->((co.int)->(co.int)) = {} }`},
		{"pointer variable", "inline_types.unit.fol", `_ co.unit = { run()->() = { pointer co.int->(*); } }`},
		{"array field", "InlineTypes.fol", `_ co.class = { values co.int->([10]); }`},
		{"derived receiver", "Employee.comp.unit.fol", `_ co.unit = { (emp Employee->(&)) method()->() = {} }`},
		{"derived lambda parameter", "inline_types.unit.fol", `_ co.unit = { run(items Values)->() = { items.each(|value co.int->(*)| => value); } }`},
		{"derived type argument", "inline_types.unit.fol", `_ co.unit = { run(value Vector(co.int->(*)))->() = {} }`},
		{"derived embedded field", "InlineTypes.fol", `_ co.struct = { (Element->(*))->(&); }`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mustPanic(t, func() { parseRegressionFile(t, tc.source, tc.basename) })
		})
	}
}

func TestOrdinaryFunctionSignatureRemainsDirect(t *testing.T) {
	source := `_ co.unit = {
    calculate(a co.int, b co.int)->(co.int) = {
        this => a + b;
    }
}`
	mustNotPanic(t, func() { parseRegressionFile(t, source, "ordinary_function.unit.fol") })
}
