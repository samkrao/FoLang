package project

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SpanConfigKey is the fol-conf.yaml key that controls whether the frontend
// artifact carries source spans.
const SpanConfigKey = "span"

// EmitSpans reports whether this project asks for source spans in the frontend
// artifact. It defaults to true, and a project turns them off with:
//
//	span: off
//
// Spans are on by default because the artifact is normally expected to carry
// them: Appendix B.7 lists Span on the AST nodes it defines, and every diagnostic
// instance must name a primary source span — a rule that reaches the backend,
// which can still reject an accepted program with UnsupportedBackendFeature and
// has to say where. Turning them off is the analogue of compiling without debug
// information: it makes the artifact smaller and gives up the ability to point at
// source, so it has to be asked for rather than fallen into.
//
// An empty root is the legacy single-file path, which has no project marker to
// read and therefore keeps the default.
func EmitSpans(root string) (bool, error) {
	if root == "" {
		return true, nil
	}

	configPath := filepath.Join(root, MarkerFilename)
	content, err := os.ReadFile(configPath)
	if os.IsNotExist(err) {
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("reading project configuration %s: %w", configPath, err)
	}

	emit, seen := true, 0
	for index, raw := range strings.Split(strings.ReplaceAll(string(content), "\r\n", "\n"), "\n") {
		line := strings.TrimSpace(stripConfigComment(raw))
		if line == "" {
			continue
		}
		colon := strings.IndexByte(line, ':')
		if colon < 0 || strings.TrimSpace(line[:colon]) != SpanConfigKey {
			continue
		}
		lineNumber := index + 1
		if seen != 0 {
			return false, fmt.Errorf("project configuration %s:%d: %s occurs more than once (first occurrence at line %d)", configPath, lineNumber, SpanConfigKey, seen)
		}
		seen = lineNumber

		value, ok := decodeConfigPathScalar(strings.TrimSpace(line[colon+1:]))
		if !ok {
			return false, fmt.Errorf("project configuration %s:%d: %s requires one scalar value, %q or %q", configPath, lineNumber, SpanConfigKey, "on", "off")
		}
		// A rejected value is not defaulted. Silently emitting spans for
		// `span: no` would hand back an artifact the project did not ask for and
		// say nothing, and the mistake would only show up as unexplained size.
		emit, ok = decodeSpanSetting(value)
		if !ok {
			return false, fmt.Errorf("project configuration %s:%d: %s = %q is not recognized; use %q or %q", configPath, lineNumber, SpanConfigKey, value, "on", "off")
		}
	}
	return emit, nil
}

// decodeSpanSetting accepts the on/off spelling the key documents and the
// true/false one a YAML reader would produce for the same intent.
func decodeSpanSetting(value string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "on", "true":
		return true, true
	case "off", "false":
		return false, true
	default:
		return false, false
	}
}
