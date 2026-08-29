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

// The protobuf wire carries a 64-bit integer exactly, across the whole range.
//
// This is the reason src/shared/folang-artifact.proto exists rather than reusing
// google.protobuf.Value: that message stores every number as a double, and
// FoLang's co.lang.int is 64-bit. Encoded through a double, 9007199254740993
// arrived as 9007199254740992 and nothing reported it, so the backend compiled a
// program the source never wrote.
func TestProtobufWireCarries64BitIntegersExactly(t *testing.T) {
	type payload struct {
		Max      int64 `json:"max"`
		Min      int64 `json:"min"`
		PastReal int64 `json:"past_real"`
		Exact    int64 `json:"exact"`
		Small    int64 `json:"small"`
	}
	original := payload{
		Max:      9223372036854775807, // the whole range, not just what a double reaches
		Min:      -9223372036854775808,
		PastReal: 9007199254740993, // 2^53+1: the value a double rounds
		Exact:    9007199254740992, // 2^53: the largest a double still holds
		Small:    10,
	}
	encoded, err := SerializeArtifact(original)
	if err != nil {
		t.Fatalf("encoding: %v", err)
	}
	var restored payload
	if err := DeserializeArtifact(encoded, &restored); err != nil {
		t.Fatalf("decoding: %v", err)
	}
	if restored != original {
		t.Errorf("round trip = %#v, want %#v", restored, original)
	}
}

// A number written with a fraction or an exponent stays a double; only integers
// take the integer field.
func TestProtobufWireKeepsFloatsAsDoubles(t *testing.T) {
	type payload struct {
		Tenth    float64 `json:"tenth"`
		Negative float64 `json:"negative"`
		Whole    float64 `json:"whole"`
	}
	original := payload{Tenth: 0.1, Negative: -2.5e-8, Whole: 3}
	encoded, err := SerializeArtifact(original)
	if err != nil {
		t.Fatalf("encoding: %v", err)
	}
	var restored payload
	if err := DeserializeArtifact(encoded, &restored); err != nil {
		t.Fatalf("decoding: %v", err)
	}
	if restored != original {
		t.Errorf("round trip = %#v, want %#v", restored, original)
	}
}

// One artifact encodes to one byte sequence. A map has no order of its own, so
// without sorted keys an artifact would change shape between identical builds
// and could not be compared, cached, or checksummed.
func TestProtobufWireIsDeterministic(t *testing.T) {
	value := map[string]any{
		"z": 1, "a": 2, "m": map[string]any{"q": []any{1, "x", true, nil}},
	}
	first, err := SerializeArtifact(value)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 8; i++ {
		again, err := SerializeArtifact(value)
		if err != nil {
			t.Fatal(err)
		}
		if string(again) != string(first) {
			t.Fatalf("encoding %d differs from the first", i)
		}
	}
}

// The JSON wire carries the same values, so the two encodings agree rather than
// one being a lossy convenience.
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

// A field the reader does not know is ignored, not refused.
//
// The artifact is an interchange contract between a frontend and a backend that
// are versioned separately, and protobuf — the format the contract names —
// ignores unknown fields by design. Refusing them would mean a producer could not
// add a field without breaking every reader built before it, and it contradicted
// the wire reader one layer down, which already skips an unrecognized protobuf
// field.
func TestAnUnknownFieldIsIgnoredOnBothWires(t *testing.T) {
	type reader struct {
		Name string `json:"name"`
	}
	// what a newer producer writes
	writer := map[string]any{"name": "artifact", "future": true, "later": []any{1, 2}}

	protobufBytes, err := SerializeArtifact(writer)
	if err != nil {
		t.Fatal(err)
	}
	jsonBytes, err := SerializeArtifactJSON(writer)
	if err != nil {
		t.Fatal(err)
	}
	for name, encoded := range map[string][]byte{"protobuf": protobufBytes, "json": jsonBytes} {
		var restored reader
		if err := DeserializeArtifact(encoded, &restored); err != nil {
			t.Errorf("%s: an older reader could not read a newer artifact: %v", name, err)
			continue
		}
		if restored.Name != "artifact" {
			t.Errorf("%s: name = %q, want the field the reader does know", name, restored.Name)
		}
	}
}

// Tolerating an unknown field costs nothing, because a genuine incompatibility
// has its own gate. A malformed document is still refused.
func TestATolerantReaderStillRefusesAMalformedDocument(t *testing.T) {
	type reader struct {
		Name string `json:"name"`
	}
	for name, test := range map[string]struct{ data, want string }{
		"empty":             {"  ", "empty input"},
		"trailing document": {`{} {}`, "trailing JSON document"},
		"truncated":         {`{"name":`, "decoding .folenc artifact"},
	} {
		t.Run(name, func(t *testing.T) {
			var restored reader
			err := DeserializeArtifact([]byte(test.data), &restored)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want text %q", err, test.want)
			}
		})
	}
}
