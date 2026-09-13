package scanlex

import "testing"

// TestLabelIdentifierIsDistinctFromCharacterLiteral covers label-identifier-guard.
//
// Both forms open with an apostrophe, and the guard is the only thing separating
// them: a character literal has a CLOSING apostrophe, a label does not. The
// scanner implements the guard by ordering — a complete character literal is
// taken first — so the two spellings have to be checked against each other rather
// than each on its own (docs/language-ref.md, "Label Lexing and Character
// Literals").
func TestLabelIdentifierIsDistinctFromCharacterLiteral(t *testing.T) {
	tests := []struct {
		source string
		kind   TokenKind
		value  string
	}{
		{"'outer", LABEL_IDENTIFIER, "'outer"},
		{"'c'", CHAR, "'c'"},
		// A one-letter label is still a label: what decides is the absence of the
		// closing apostrophe, not the length of the name.
		{"'c", LABEL_IDENTIFIER, "'c"},
		// A non-ASCII character literal stays a literal; it is not an apostrophe
		// followed by an identifier at all.
		{"'∪'", CHAR, "'∪'"},
	}

	for _, test := range tests {
		t.Run(test.source, func(t *testing.T) {
			tokens := Tokenize(test.source, "labels.fol")
			if len(tokens) != 1 {
				t.Fatalf("%q tokens = %#v, want exactly one", test.source, tokens)
			}
			if tokens[0].Kind != test.kind || tokens[0].Value != test.value {
				t.Fatalf("%q token = %s %q, want %s %q", test.source,
					TokenKindString(tokens[0].Kind), tokens[0].Value,
					TokenKindString(test.kind), test.value)
			}
		})
	}
}

// TestLabelDeclarationSeparatesFromItsColon keeps the ":" out of the label name.
// The colon belongs to the declaration syntax, not to the label.
func TestLabelDeclarationSeparatesFromItsColon(t *testing.T) {
	tokens := Tokenize("'outer:", "labels.fol")
	if len(tokens) != 2 {
		t.Fatalf("tokens = %#v, want a label and a colon", tokens)
	}
	if tokens[0].Kind != LABEL_IDENTIFIER || tokens[0].Value != "'outer" {
		t.Fatalf("label token = %s %q", TokenKindString(tokens[0].Kind), tokens[0].Value)
	}
	if tokens[1].Kind != COLON {
		t.Fatalf("second token = %s, want a colon", TokenKindString(tokens[1].Kind))
	}
}

// TestLifecycleMarkerIsItsOwnSpelling checks that "::" is a whole symbolic run of
// its own and does not collide with the longer hard-reserved "::=".
//
// The symbolic-run scan takes the longest complete spelling and never splits an
// unrecognized run, so the two have to resolve independently rather than one
// being read as a prefix of the other.
func TestLifecycleMarkerIsItsOwnSpelling(t *testing.T) {
	marker := Tokenize("::", "lifecycle.fol")
	if len(marker) != 1 || marker[0].Kind != LIFECYCLE_MARKER {
		t.Fatalf("\"::\" tokens = %#v, want one LIFECYCLE_MARKER", marker)
	}

	walrus := Tokenize("::=", "lifecycle.fol")
	if len(walrus) != 1 || walrus[0].Kind != COLON_WALRUS {
		t.Fatalf("\"::=\" tokens = %#v, want one COLON_WALRUS", walrus)
	}
}

// TestLifecycleCallTokenizesAsMarkerAndName confirms the invocation name reaches
// the parser as an ordinary identifier. `new` and `init` are not reserved words:
// ordinary methods may carry those names, and only the preceding "::" makes an
// occurrence a lifecycle invocation. The name carries the backend lowering
// suffix like any other identifier, which is what the parser strips before
// testing it against the closed lifecycle-invocation-name set.
func TestLifecycleCallTokenizesAsMarkerAndName(t *testing.T) {
	tokens := Tokenize("Employee::new", "lifecycle.fol")
	if len(tokens) != 3 {
		t.Fatalf("tokens = %#v, want a receiver, a marker and a name", tokens)
	}
	if tokens[1].Kind != LIFECYCLE_MARKER {
		t.Fatalf("marker token = %s, want LIFECYCLE_MARKER", TokenKindString(tokens[1].Kind))
	}
	if tokens[2].Kind != IDENTIFIER || tokens[2].Value != "new_fo" {
		t.Fatalf("name token = %s %q, want the identifier \"new\"",
			TokenKindString(tokens[2].Kind), tokens[2].Value)
	}
}
