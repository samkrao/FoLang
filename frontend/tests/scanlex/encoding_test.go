package scanlex_test

import (
	"testing"

	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

func TestTokenizeRejectsInvalidEncodingBeforeScanningContexts(t *testing.T) {
	invalid := string([]byte{0xff})
	tests := []struct {
		name   string
		source string
		file   string
	}{
		{name: "ordinary source", source: "value := " + invalid + ";", file: "body.fol"},
		{name: "string", source: `"before` + invalid + `after"`, file: "body.fol"},
		{name: "block comment", source: "/* before " + invalid + " after */", file: "body.fol"},
		{name: "line comment in operator source", source: "// before " + invalid + " after\n", file: "operators.fol"},
		{name: "interior BOM in string", source: "\"before\uFEFFafter\"", file: "body.fol"},
		{name: "interior BOM in comment", source: "/* before \uFEFF after */", file: "body.fol"},
		{name: "second leading BOM", source: "\uFEFF\uFEFFvalue", file: "body.fol"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if recovered := diagnosticPanic(func() {
				scanlex.Tokenize(test.source, test.file)
			}); recovered != "Error" {
				t.Fatalf("Tokenize recovered %#v, want diagnostic panic %q", recovered, "Error")
			}
		})
	}
}

func TestTokenizeWithAlsoRejectsInvalidEncoding(t *testing.T) {
	custom := scanlex.NewCustomOperatorsWithSpecs([]scanlex.OperatorSpec{{Symbol: "+-", Fixity: "infix"}})
	source := "left +- " + string([]byte{0xfe})
	if recovered := diagnosticPanic(func() {
		scanlex.TokenizeWith(source, "body.fol", custom)
	}); recovered != "Error" {
		t.Fatalf("TokenizeWith recovered %#v, want diagnostic panic %q", recovered, "Error")
	}
}

func TestLeadingBOMAndEncodedReplacementRuneRemainValid(t *testing.T) {
	tokens := meaningful(scanlex.Tokenize("\uFEFF\uFFFD", "body.fol"))
	if len(tokens) != 1 {
		t.Fatalf("tokens = %#v, want one replacement-rune token", tokens)
	}
	assertKindValue(t, tokens[0], scanlex.SYMBOLIC_RUN, "\uFFFD")

	if !scanlex.IsOperatorSpelling("\uFFFD") {
		t.Fatal("valid encoded U+FFFD should be an operator spelling")
	}
	if scanlex.IsOperatorSpelling(string([]byte{0xff})) {
		t.Fatal("invalid UTF-8 byte should not be an operator spelling")
	}

	custom := scanlex.NewCustomOperatorsWithSpecs([]scanlex.OperatorSpec{{Symbol: "\uFFFD", Fixity: "prefix"}})
	registered := meaningful(scanlex.TokenizeWith("\uFFFD value", "body.fol", custom))
	if len(registered) == 0 {
		t.Fatal("registered replacement-rune operator produced no token")
	}
	assertKindValue(t, registered[0], scanlex.CUSTOM_OPERATOR, "\uFFFD")
}

func TestTokenizeQuietRemainsBestEffortForEncodingErrors(t *testing.T) {
	source := "/* \uFEFF */ " + string([]byte{0xff}) + " value"
	if recovered := diagnosticPanic(func() {
		scanlex.TokenizeQuiet(source, "surface.fol")
	}); recovered != nil {
		t.Fatalf("TokenizeQuiet recovered %#v, want no diagnostic panic", recovered)
	}
}

func diagnosticPanic(scan func()) (recovered any) {
	defer func() {
		recovered = recover()
	}()
	scan()
	return nil
}
