package helpers

import (
	"strings"
	"testing"
)

type artifactCodecFixture struct {
	Version int               `json:"version"`
	Name    string            `json:"name"`
	Values  map[string]string `json:"values"`
}

func TestArtifactCodecRoundTripsBothWires(t *testing.T) {
	want := artifactCodecFixture{Version: 1, Name: "example", Values: map[string]string{"answer": "42"}}
	for name, encode := range map[string]func(any) ([]byte, error){
		"protobuf": SerializeArtifact,
		"json":     SerializeArtifactJSON,
	} {
		t.Run(name, func(t *testing.T) {
			encoded, err := encode(want)
			if err != nil {
				t.Fatal(err)
			}
			var got artifactCodecFixture
			if err := DeserializeArtifact(encoded, &got); err != nil {
				t.Fatal(err)
			}
			if got.Version != want.Version || got.Name != want.Name || got.Values["answer"] != "42" {
				t.Fatalf("decoded artifact = %#v, want %#v", got, want)
			}
		})
	}
}

func TestArtifactDecoderRejectsInvalidDocuments(t *testing.T) {
	for name, test := range map[string]struct {
		data string
		out  any
		want string
	}{
		"empty":             {data: "  ", out: &artifactCodecFixture{}, want: "empty input"},
		"non pointer":       {data: `{}`, out: artifactCodecFixture{}, want: "pointer destination"},
		"unknown field":     {data: `{"future":true}`, out: &artifactCodecFixture{}, want: "unknown field"},
		"trailing document": {data: `{} {}`, out: &artifactCodecFixture{}, want: "trailing JSON document"},
	} {
		t.Run(name, func(t *testing.T) {
			err := DeserializeArtifact([]byte(test.data), test.out)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want text %q", err, test.want)
			}
		})
	}
}
