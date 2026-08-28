package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeMarker(t *testing.T, content string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, MarkerFilename), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

// Spans are on unless the project asks otherwise, because the artifact is
// normally expected to carry them.
func TestEmitSpansDefaultsOn(t *testing.T) {
	for name, root := range map[string]string{
		"no span key": writeMarker(t, "project: demo\n"),
		"no marker":   t.TempDir(),
		"single file": "",
	} {
		emit, err := EmitSpans(root)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if !emit {
			t.Errorf("%s: spans were off without being asked for", name)
		}
	}
}

func TestEmitSpansReadsTheSetting(t *testing.T) {
	for value, want := range map[string]bool{
		"on":    true,
		"true":  true,
		"off":   false,
		"OFF":   false,
		"false": false,
		" off ": false,
		`"off"`: false,
		"'off'": false,
	} {
		emit, err := EmitSpans(writeMarker(t, "project: demo\nspan: "+value+"\n"))
		if err != nil {
			t.Errorf("span: %s -> %v", value, err)
			continue
		}
		if emit != want {
			t.Errorf("span: %s -> %v, want %v", value, emit, want)
		}
	}
}

// A comment after the value is ordinary YAML and must not be read as part of it.
func TestEmitSpansIgnoresTrailingComments(t *testing.T) {
	emit, err := EmitSpans(writeMarker(t, "project: demo\nspan: off   # no debug information\n"))
	if err != nil {
		t.Fatal(err)
	}
	if emit {
		t.Error("a trailing comment stopped the setting being read")
	}
}

// An unrecognized value is refused rather than defaulted. Emitting spans for
// `span: no` would hand back an artifact the project did not ask for and say
// nothing about it.
func TestEmitSpansRejectsAnUnknownSetting(t *testing.T) {
	for _, value := range []string{"maybe", "no", "yes", "1", ""} {
		_, err := EmitSpans(writeMarker(t, "project: demo\nspan: "+value+"\n"))
		if err == nil {
			t.Errorf("span: %q was accepted", value)
			continue
		}
		if !strings.Contains(err.Error(), SpanConfigKey) {
			t.Errorf("span: %q -> %v, which does not name the offending key", value, err)
		}
	}
}

func TestEmitSpansRejectsARepeatedKey(t *testing.T) {
	_, err := EmitSpans(writeMarker(t, "project: demo\nspan: on\nspan: off\n"))
	if err == nil {
		t.Fatal("a repeated span key was accepted")
	}
	if !strings.Contains(err.Error(), "more than once") {
		t.Errorf("error does not describe the duplicate: %v", err)
	}
}
