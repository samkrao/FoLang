package helpers

import (
	"errors"
	"testing"
)

func TestArtifactCodecHooksRemainExplicitlyDeferred(t *testing.T) {
	if encoded, err := SerializeArtifact(struct{}{}); encoded != nil || !errors.Is(err, ErrArtifactCodecNotImplemented) {
		t.Fatalf("SerializeArtifact = %v, %v", encoded, err)
	}
	target := struct{ Value string }{Value: "unchanged"}
	if err := DeserializeArtifact([]byte("future codec"), &target); !errors.Is(err, ErrArtifactCodecNotImplemented) {
		t.Fatalf("DeserializeArtifact error = %v", err)
	}
	if target.Value != "unchanged" {
		t.Fatal("deferred decoder mutated its destination")
	}
}
