package scanlex

import "testing"

func TestDetectUnsupportedAlphaLiteral(t *testing.T) {
	tests := []struct {
		source  string
		message string
	}{
		{`"quoted: \"text\""`, "escaped string characters are not supported in the alpha release"},
		{`'\n'`, "escaped character literals are not supported in the alpha release"},
		{`u8"hello"`, "encoded string literal prefixes are not supported in the alpha release"},
		{`L'x'`, "encoded character literal prefixes are not supported in the alpha release"},
		{`R"(raw text)"`, "raw string literals are not supported in the alpha release"},
		{`u8R"tag(raw text)tag"`, "raw string literals are not supported in the alpha release"},
		{`'\N{LATIN CAPITAL LETTER A}'`, "named universal character literals are not supported in the alpha release"},
	}

	for _, test := range tests {
		t.Run(test.source, func(t *testing.T) {
			length, message, ok := detectUnsupportedAlphaLiteral(test.source + ";")
			if !ok {
				t.Fatal("future literal was not recognized")
			}
			if length != len(test.source) {
				t.Fatalf("recognized length %d, want %d", length, len(test.source))
			}
			if message != test.message {
				t.Fatalf("diagnostic %q, want %q", message, test.message)
			}
		})
	}
}

func TestDetectUnsupportedAlphaLiteralLeavesSimpleSubsetAlone(t *testing.T) {
	for _, source := range []string{`"hello"`, `'x'`, `u8`, `L`, `u8 "hello"`} {
		t.Run(source, func(t *testing.T) {
			if _, _, unsupported := detectUnsupportedAlphaLiteral(source); unsupported {
				t.Fatalf("%q should remain in the alpha tokenization path", source)
			}
		})
	}
}
