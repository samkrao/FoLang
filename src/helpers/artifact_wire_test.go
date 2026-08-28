package helpers

import (
	"strings"
	"testing"
)

// The wire encodings are told apart by their first byte, so the two sets of
// opening bytes must never meet. A JSON artifact opens with "{" or "["; a
// protobuf google.protobuf.Value opens with the tag of one of its six fields.
//
// This is checked rather than assumed: if the protobuf message ever changes
// shape, a document that opens like JSON would be handed to the JSON decoder and
// fail as corrupt, naming neither the real encoding nor the real problem.
func TestProtobufArtifactNeverOpensLikeJSON(t *testing.T) {
	for name, value := range map[string]any{
		"an object":       map[string]any{"a": 1},
		"an array":        []any{1, 2, 3},
		"a string":        "text",
		"a number":        42,
		"a bool":          true,
		"a nested tree":   map[string]any{"a": map[string]any{"b": []any{"c"}}},
		"an empty object": map[string]any{},
		"an empty array":  []any{},
	} {
		encoded, err := MarshalProtobuf(value)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if len(encoded) == 0 {
			t.Fatalf("%s: encoded to nothing", name)
		}
		if encoded[0] == '{' || encoded[0] == '[' {
			t.Errorf("%s: protobuf artifact opens with %q, which the reader takes for JSON", name, encoded[0])
		}
	}
}

// Each encoding decodes back to what it carried, and each is recognised without
// being told which it is.
func TestBothWireEncodingsRoundTrip(t *testing.T) {
	type payload struct {
		Name  string `json:"name"`
		Count int64  `json:"count"`
	}
	original := payload{Name: "artifact", Count: 1234567}

	protobufBytes, err := SerializeArtifact(original)
	if err != nil {
		t.Fatal(err)
	}
	jsonBytes, err := SerializeArtifactJSON(original)
	if err != nil {
		t.Fatal(err)
	}
	for name, encoded := range map[string][]byte{"protobuf": protobufBytes, "json": jsonBytes} {
		var restored payload
		if err := DeserializeArtifact(encoded, &restored); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if restored != original {
			t.Errorf("%s round trip = %#v, want %#v", name, restored, original)
		}
	}
}

// An integer past 2^53 is refused by the protobuf wire rather than rounded.
//
// google.protobuf.Value stores every number as a double. FoLang's co.lang.int is
// 64-bit and an AST integer literal is an int64, so a literal past 2^53 would
// otherwise reach the backend as a DIFFERENT number, silently: 9007199254740993
// arrives as 9007199254740992 and nothing reports it.
func TestProtobufRefusesIntegersADoubleCannotHold(t *testing.T) {
	type payload struct {
		Big int64 `json:"big"`
	}

	// 2^53 is the largest integer a double holds exactly, so it still encodes.
	exact := payload{Big: 9007199254740992}
	encoded, err := SerializeArtifact(exact)
	if err != nil {
		t.Fatalf("2^53 must still encode: %v", err)
	}
	var restored payload
	if err := DeserializeArtifact(encoded, &restored); err != nil {
		t.Fatal(err)
	}
	if restored.Big != exact.Big {
		t.Errorf("2^53 round trip = %d, want %d", restored.Big, exact.Big)
	}

	// 2^53+1 is not, and must be refused rather than rounded down to 2^53.
	_, err = SerializeArtifact(payload{Big: 9007199254740993})
	if err == nil {
		t.Fatal("an integer a double cannot hold was encoded anyway")
	}
	for _, want := range []string{"9007199254740993", "big", `"wire": "json"`} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("refusal does not mention %q: %v", want, err)
		}
	}
}

// The JSON wire carries the same value unchanged, which is what the refusal
// above points the caller at.
func TestJSONWireCarries64BitIntegersExactly(t *testing.T) {
	type payload struct {
		Big int64 `json:"big"`
	}
	original := payload{Big: 9007199254740993}
	encoded, err := SerializeArtifactJSON(original)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), "9007199254740993") {
		t.Fatalf("the written JSON lost the value: %s", encoded)
	}
	var restored payload
	if err := DeserializeArtifact(encoded, &restored); err != nil {
		t.Fatal(err)
	}
	if restored.Big != original.Big {
		t.Errorf("json round trip = %d, want %d", restored.Big, original.Big)
	}
}
