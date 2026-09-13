package scanlex

import (
	"strings"
	"testing"
)

func TestValidateSourceEncodingReportsUsefulPosition(t *testing.T) {
	err := validateSourceEncoding("first\nsecond \uFEFF rest", "nested.fol")
	if err == nil {
		t.Fatal("interior U+FEFF was accepted")
	}
	diagnostic := err.AsString()
	for _, want := range []string{"U+FEFF", "File nested.fol, line 2", "second"} {
		if !strings.Contains(diagnostic, want) {
			t.Errorf("diagnostic %q does not contain %q", diagnostic, want)
		}
	}
}

func TestValidateSourceEncodingDistinguishesReplacementRuneFromInvalidByte(t *testing.T) {
	if err := validateSourceEncoding("\uFEFF\uFFFD", "valid.fol"); err != nil {
		t.Fatalf("valid encoded U+FFFD was rejected: %s", err.AsString())
	}
	if err := validateSourceEncoding(string([]byte{'x', 0xff}), "invalid.fol"); err == nil {
		t.Fatal("invalid UTF-8 byte was accepted")
	}
}
