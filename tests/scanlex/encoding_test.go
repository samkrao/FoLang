package scanlex_test

import (
	"testing"

	"github.com/samkrao/fo-lang/src/scanlex"
)

func TestInvalidUTF8BecomesUnknownToken(t *testing.T) {
	invalid := string([]byte{0xff})
	tokens := meaningful(scanlex.Tokenize("value "+invalid+" next", "body.fol"))
	if len(tokens) != 3 {
		t.Fatalf("tokens = %#v, want identifier, UNKNOWN, identifier", tokens)
	}
	assertKindValue(t, tokens[1], scanlex.UNKNOWN, invalid)
}

func TestTokenizeWithPreservesUnknownUTF8AsUnknown(t *testing.T) {
	custom := scanlex.NewCustomOperatorsWithSpecs([]scanlex.OperatorSpec{{Symbol: "+-", Fixity: "infix"}})
	source := "left +- " + string([]byte{0xfe})
	tokens := meaningful(scanlex.TokenizeWith(source, "body.fol", custom))
	if len(tokens) != 3 {
		t.Fatalf("tokens = %#v, want identifier, custom operator, UNKNOWN", tokens)
	}
	assertKindValue(t, tokens[2], scanlex.UNKNOWN, string([]byte{0xfe}))
}

func TestLeadingBOMAndEncodedReplacementRuneRemainValid(t *testing.T) {
	tokens := meaningful(scanlex.Tokenize("\uFEFF\uFFFD", "body.fol"))
	if len(tokens) != 1 {
		t.Fatalf("tokens = %#v, want one replacement-rune token", tokens)
	}
	assertKindValue(t, tokens[0], scanlex.UNKNOWN, "\uFFFD")

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

func TestTokenizeQuietReturnsUnknownForInvalidEncoding(t *testing.T) {
	source := "/* \uFEFF */ " + string([]byte{0xff}) + " value"
	tokens := meaningful(scanlex.TokenizeQuiet(source, "surface.fol"))
	if len(tokens) == 0 || tokens[0].Kind != scanlex.UNKNOWN {
		t.Fatalf("first token = %#v, want UNKNOWN", tokens)
	}
}
