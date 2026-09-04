package parser_test

import "testing"

func TestNamedDerivedTypesAreAcceptedAtOrdinaryUseSites(t *testing.T) {
	source := `_ co.lang.unit = {
    Binary co.lang.type = (co.lang.int, co.lang.int)->(co.lang.int);
    IntPtr co.lang.type = co.lang.int->(*);
    TenInts co.lang.type = co.lang.int->([10]);

    apply(operation Binary)->(co.lang.int) = {
        pointer IntPtr;
        values TenInts;
        items co.core.List(co.lang.int);
        this.return operation(10, 20);
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
		{"function parameter", "inline_types.unit.fol", `_ co.lang.unit = { apply(operation (co.lang.int)->(co.lang.int))->() = {} }`},
		{"function result", "inline_types.unit.fol", `_ co.lang.unit = { make()->((co.lang.int)->(co.lang.int)) = {} }`},
		{"pointer variable", "inline_types.unit.fol", `_ co.lang.unit = { run()->() = { pointer co.lang.int->(*); } }`},
		{"array field", "InlineTypes.fol", `_ co.lang.class = { values co.lang.int->([10]); }`},
		{"derived receiver", "Employee.comp.unit.fol", `_ co.lang.unit = { (emp Employee->(&)) method()->() = {} }`},
		{"derived lambda parameter", "inline_types.unit.fol", `_ co.lang.unit = { run(items Values)->() = { items.each(|value co.lang.int->(*)| => value); } }`},
		{"derived type argument", "inline_types.unit.fol", `_ co.lang.unit = { run(value Vector(co.lang.int->(*)))->() = {} }`},
		{"derived embedded field", "InlineTypes.fol", `_ co.lang.struct = { (Element->(*))->(&); }`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mustPanic(t, func() { parseRegressionFile(t, tc.source, tc.basename) })
		})
	}
}

func TestOrdinaryFunctionSignatureRemainsDirect(t *testing.T) {
	source := `_ co.lang.unit = {
    calculate(a co.lang.int, b co.lang.int)->(co.lang.int) = {
        this.return a + b;
    }
}`
	mustNotPanic(t, func() { parseRegressionFile(t, source, "ordinary_function.unit.fol") })
}
