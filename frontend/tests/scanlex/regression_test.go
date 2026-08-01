package scanlex_test

import (
	"testing"

	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

func TestReservedBackslashReachesParserAsPunctuation(t *testing.T) {
	tokens := meaningful(scanlex.Tokenize(`\`, "test"))
	if len(tokens) != 1 {
		t.Fatalf("backslash produced %d meaningful tokens, want 1", len(tokens))
	}
	assertKindValue(t, tokens[0], scanlex.BACK_SLASH, `\`)
}

func TestBlockCommentPreservesTrailingLineAndColumn(t *testing.T) {
	tokens := meaningful(scanlex.Tokenize("/* first\nsecond */value", "test"))
	if len(tokens) != 1 {
		t.Fatalf("source produced %d meaningful tokens, want 1", len(tokens))
	}
	start := tokens[0].StartPos
	if start.Ln != 2 || start.Col != len("second */") {
		t.Fatalf("token after block comment starts at line %d, column %d; want line 2, column %d",
			start.Ln, start.Col, len("second */"))
	}
}

func TestMalformedAbbreviatedFloatsAreScannerDiagnostics(t *testing.T) {
	for _, source := range []string{"1.", "0x1.p3", "0x.8p3"} {
		t.Run(source, func(t *testing.T) {
			defer func() {
				if recovered := recover(); recovered != "Error" {
					t.Fatalf("Tokenize(%q) recovered %#v, want diagnostic panic %q", source, recovered, "Error")
				}
			}()
			scanlex.Tokenize(source, "test")
		})
	}
}
