package scanlex

import "testing"

func TestUnicodeReservedWordDoesNotWidenIdentifierGrammar(t *testing.T) {
	tokens, diagnostics := TokenizeCollecting("fΦλ fo ordinary", "reserved.fol", nil)
	if len(diagnostics) != 0 {
		t.Fatalf("exact reserved spelling produced diagnostics: %v", diagnostics)
	}
	if len(tokens) != 3 {
		t.Fatalf("tokens = %#v", tokens)
	}
	if tokens[0].Kind != RESERVEDWORD || tokens[0].Value != "fΦλ" {
		t.Fatalf("first token = %#v, want exact fΦλ RESERVEDWORD", tokens[0])
	}
	for _, at := range []int{1, 2} {
		if tokens[at].Kind != IDENTIFIER {
			t.Errorf("token %q kind = %s, want IDENTIFIER", tokens[at].Value, TokenKindString(tokens[at].Kind))
		}
	}
}

func TestUnicodeReservedWordRequiresAnExactBoundary(t *testing.T) {
	tokens, _ := TokenizeCollecting("fΦλname", "reserved.fol", nil)
	for _, token := range tokens {
		if token.Kind == RESERVEDWORD {
			t.Fatalf("fΦλname tokens = %#v; only the exact fΦλ spelling may become RESERVEDWORD", tokens)
		}
	}
}
