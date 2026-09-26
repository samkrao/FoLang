package scanlex_test

import (
	"testing"

	"github.com/samkrao/fo-lang/src/scanlex"
)

func TestTokenStreamFoldsOnDemandAndReturnsUnknown(t *testing.T) {
	lexer := scanlex.NewLexerCollecting([]byte("\nfirst ''"), "lazy.fol", nil)
	stream := scanlex.NewTokenStream(lexer)

	first := stream.Next()
	if first.Kind != scanlex.IDENTIFIER || first.Value != "first_fo" {
		t.Fatalf("first token = %s(%q), want IDENTIFIER(%q)", scanlex.TokenKindString(first.Kind), first.Value, "first_fo")
	}
	if diagnostics := lexer.Diagnostics(); len(diagnostics) != 0 {
		t.Fatalf("lexer unexpectedly reported diagnostics: %v", diagnostics)
	}

	if got := stream.Peek(0); got.Kind != scanlex.UNKNOWN || got.Value != "''" {
		t.Fatalf("Peek(0) for malformed character literal = %s(%q), want UNKNOWN(%q)", scanlex.TokenKindString(got.Kind), got.Value, "''")
	}
	if diagnostics := lexer.Diagnostics(); len(diagnostics) != 0 {
		t.Fatalf("malformed lexeme should be represented by UNKNOWN, got diagnostics: %v", diagnostics)
	}
	if got := stream.Next(); got.Kind != scanlex.UNKNOWN || got.Value != "''" {
		t.Fatalf("Next() after Peek() = %s(%q), want UNKNOWN(%q)", scanlex.TokenKindString(got.Kind), got.Value, "''")
	}
}

func TestLexerReturnsUnknownForInvalidLexemes(t *testing.T) {
	for _, source := range []string{
		"a__b",
		"name_",
		"1.",
		"@@notAMethod",
		"\"unterminated",
		"\"escaped\\nstring\"",
		"\"embedded\uFEFFbom\"",
		"R\"(raw)\"",
		string([]byte{0xff}),
		"\"embedded" + string([]byte{0xff}) + "byte\"",
	} {
		t.Run(source, func(t *testing.T) {
			stream := scanlex.NewTokenStream(scanlex.NewLexer([]byte(source), "invalid.fol", nil))
			got := stream.Next()
			if got.Kind != scanlex.UNKNOWN {
				t.Fatalf("token = %s(%q), want UNKNOWN(%q)", scanlex.TokenKindString(got.Kind), got.Value, source)
			}
			if got.Value != source {
				t.Fatalf("UNKNOWN lexeme = %q, want original %q", got.Value, source)
			}
			if got := stream.Next(); got.Kind != scanlex.EOF {
				t.Fatalf("token after unknown lexeme = %s(%q), want EOF", scanlex.TokenKindString(got.Kind), got.Value)
			}
		})
	}
}

func TestLexerPreservesValidTokenAfterUnknown(t *testing.T) {
	stream := scanlex.NewTokenStream(scanlex.NewLexer([]byte("\na__b valid"), "recover.fol", nil))
	if got := stream.Next(); got.Kind != scanlex.UNKNOWN || got.Value != "a__b" {
		t.Fatalf("first token = %s(%q), want UNKNOWN(%q)", scanlex.TokenKindString(got.Kind), got.Value, "a__b")
	}
	if got := stream.Next(); got.Kind != scanlex.IDENTIFIER || got.Value != "valid_fo" {
		t.Fatalf("second token = %s(%q), want IDENTIFIER(%q)", scanlex.TokenKindString(got.Kind), got.Value, "valid_fo")
	}
}

func TestTokenStreamPeekBuffersRequestedLookahead(t *testing.T) {
	stream := scanlex.NewTokenStream(scanlex.NewLexer([]byte("\na b c"), "peek.fol", nil))
	if got := stream.Peek(2); got.Value != "c_fo" {
		t.Fatalf("Peek(2) = %q, want %q", got.Value, "c_fo")
	}
	if got := stream.Next(); got.Value != "a_fo" {
		t.Fatalf("Next() after Peek(2) = %q, want %q", got.Value, "a_fo")
	}
	if got := stream.Peek(1); got.Value != "c_fo" {
		t.Fatalf("Peek(1) after consuming first token = %q, want %q", got.Value, "c_fo")
	}
	if got := stream.Next(); got.Value != "b_fo" {
		t.Fatalf("second Next() = %q, want %q", got.Value, "b_fo")
	}
	if got := stream.Next(); got.Value != "c_fo" {
		t.Fatalf("third Next() = %q, want %q", got.Value, "c_fo")
	}
	if got := stream.Next(); got.Kind != scanlex.EOF {
		t.Fatalf("Next() after final token = %s, want EOF", scanlex.TokenKindString(got.Kind))
	}
	if got := stream.Next(); got.Kind != scanlex.EOF {
		t.Fatalf("repeated Next() at EOF = %s, want EOF", scanlex.TokenKindString(got.Kind))
	}
}

func TestTokenStreamMatchesLegacyFolding(t *testing.T) {
	source := "co.lang.int service.worker @custom.value co.out"
	stream := scanlex.NewTokenStream(scanlex.NewLexer([]byte(source), "folded.fol", nil))
	want := []struct {
		kind  scanlex.TokenKind
		value string
	}{
		{scanlex.BUILT_IN_TYPE, "co.lang.int"},
		{scanlex.COMPOSITE_IDENTIFER, "service_fo.worker"},
		{scanlex.CUSTOM_DIRECTIVES, "@custom.value"},
		{scanlex.BUIL_IN_STMT_EXPRS, "co.out"},
	}

	for i, expected := range want {
		got := stream.Next()
		if got.Kind != expected.kind || got.Value != expected.value {
			t.Fatalf("token %d = %s(%q), want %s(%q)", i,
				scanlex.TokenKindString(got.Kind), got.Value,
				scanlex.TokenKindString(expected.kind), expected.value)
		}
	}
	if got := stream.Next(); got.Kind != scanlex.EOF {
		t.Fatalf("token after folded stream = %s(%q), want EOF", scanlex.TokenKindString(got.Kind), got.Value)
	}
}

func TestEmbeddedBOMInSymbolicRunIsOneUnknownLexeme(t *testing.T) {
	source := "+\uFEFF+"
	stream := scanlex.NewTokenStream(scanlex.NewLexer([]byte(source), "encoding.fol", nil))
	got := stream.Next()
	if got.Kind != scanlex.UNKNOWN || got.Value != source {
		t.Fatalf("token = %s(%q), want UNKNOWN(%q)", scanlex.TokenKindString(got.Kind), got.Value, source)
	}
}

func TestMultilineUnknownPreservesFollowingPosition(t *testing.T) {
	source := "R\"(first\nsecond)\" next"
	stream := scanlex.NewTokenStream(scanlex.NewLexer([]byte(source), "position.fol", nil))
	unknown := stream.Next()
	if unknown.Kind != scanlex.UNKNOWN || unknown.Value != "R\"(first\nsecond)\"" {
		t.Fatalf("first token = %s(%q), want complete raw literal as UNKNOWN", scanlex.TokenKindString(unknown.Kind), unknown.Value)
	}
	next := stream.Next()
	if next.Kind != scanlex.IDENTIFIER || next.Value != "next_fo" {
		t.Fatalf("second token = %s(%q), want IDENTIFIER(%q)", scanlex.TokenKindString(next.Kind), next.Value, "next_fo")
	}
	if next.StartPos.Ln != 2 || next.StartPos.Col != len(`second)" `) {
		t.Fatalf("token after multiline UNKNOWN starts at line %d, column %d; want line 2, column %d",
			next.StartPos.Ln, next.StartPos.Col, len(`second)" `))
	}
}
