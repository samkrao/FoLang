// Package helpers provides shared utility functions for error handling,
// position tracking, serialization, hashing, and code formatting.
package helpers

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"os"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"

	"fmt"
	"os/exec"
	"path/filepath"

	uniuri "github.com/dchest/uniuri"
	"gopkg.in/mgo.v2/bson"
)

// SerializeArtifact encodes the language-neutral .folenc logical model using
// protobuf, the default FoLang artifact wire format.
func SerializeArtifact(value any) ([]byte, error) {
	return MarshalProtobuf(value)
}

// SerializeArtifactJSON encodes the same logical artifact model as JSON.
func SerializeArtifactJSON(value any) ([]byte, error) {
	if value == nil {
		return nil, errors.New("cannot serialize a nil .folenc artifact")
	}
	encoded, err := Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encoding .folenc artifact: %w", err)
	}
	return encoded, nil
}

// DeserializeArtifact accepts both supported .folenc wire encodings.
//
// The two are told apart by the first byte, which is unambiguous rather than a
// guess. A JSON artifact is an object or an array, so it opens with "{" or "[" —
// 0x7b or 0x5b. A protobuf artifact is a folang.artifact.Value, whose first byte
// is the tag of one of its seven fields: 0x08, 0x11, 0x1a, 0x20, 0x2a, 0x32 or
// 0x38 — the last being int_value, which google.protobuf.Value does not have.
// The two sets do not meet, and TestProtobufArtifactNeverOpensLikeJSON keeps it
// that way if the message ever changes.
func DeserializeArtifact(data []byte, out any) error {
	if len(bytes.TrimSpace(data)) == 0 {
		return errors.New("decoding .folenc artifact: empty input")
	}
	if out == nil || reflect.ValueOf(out).Kind() != reflect.Pointer || reflect.ValueOf(out).IsNil() {
		return errors.New("decoding .folenc artifact requires a non-nil pointer destination")
	}
	trimmed := bytes.TrimSpace(data)
	if trimmed[0] != '{' && trimmed[0] != '[' {
		return UnmarshalProtobuf(data, out)
	}
	return decodeArtifactJSON(data, out)
}

// decodeArtifactJSON reads one artifact document into a typed destination.
//
// A field the destination does not know is IGNORED rather than refused. The
// artifact is an interchange contract between a frontend and a backend that are
// versioned separately, and protobuf — the format this contract names — ignores
// unknown fields by design, so a producer may add one without breaking every
// reader built before it. Refusing here also contradicted the wire reader one
// layer down, which already skips an unrecognized protobuf field.
//
// Nothing is lost by being tolerant, because a genuine incompatibility has its
// own gate: the artifact carries SymbolFormatVersion, and a mismatch is reported
// as a version error naming both numbers. That is the check that should fail a
// stale artifact — not the accident of a field name the reader has not heard of.
//
// A configuration file is the opposite case and stays strict. backend-conf.json
// and the debug-trace configuration are written by a person or an installer, so
// an unrecognized key there is a typo worth reporting rather than a newer
// producer to accommodate.
func decodeArtifactJSON(data []byte, out any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	// UseNumber matters only for an UNTYPED destination, and there it is the
	// difference between reading the artifact and misreading it. Decoding into
	// map[string]any without it turns every number into a float64, so a consumer
	// walking the tree generically — a tool, a test, a backend that has no Go
	// structs for the model — would see 9007199254740993 as 9007199254740992,
	// undoing at the reader exactly what the wire schema preserves. A typed
	// destination is unaffected: an int64 field is parsed as an int64 either way.
	decoder.UseNumber()
	if err := decoder.Decode(out); err != nil {
		return fmt.Errorf("decoding .folenc artifact: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("decoding .folenc artifact: trailing JSON document")
		}
		return fmt.Errorf("decoding .folenc artifact trailing data: %w", err)
	}
	return nil
}

// MarshalProtobuf encodes the artifact using the FoLang artifact schema,
// src/shared/folang-artifact.proto.
//
// The schema is google.protobuf.Value plus an int64 field, so an AST integer
// literal — an int64, since co.lang.int is 64-bit — is carried as an integer
// rather than squeezed through a double. Encoding through the well-known type
// turned 9007199254740993 into 9007199254740992 and said nothing.
func MarshalProtobuf(value any) ([]byte, error) {
	if value == nil {
		return nil, errors.New("cannot serialize a nil protobuf artifact")
	}
	tree, err := artifactTree(value)
	if err != nil {
		return nil, err
	}
	encoded, err := appendArtifactValue(nil, tree)
	if err != nil {
		return nil, err
	}
	return encoded, nil
}

// UnmarshalProtobuf restores an artifact message into a typed logical model.
//
// The decoded tree is re-marshalled to JSON and decoded into the destination, so
// one set of struct tags describes both wires. An integer survives that step:
// encoding/json writes an int64 as its digits.
func UnmarshalProtobuf(data []byte, out any) error {
	tree, err := parseArtifactValue(data)
	if err != nil {
		return err
	}
	jsonBytes, err := json.Marshal(artifactJSONTree(tree))
	if err != nil {
		return fmt.Errorf("projecting decoded protobuf artifact: %w", err)
	}
	return decodeArtifactJSON(jsonBytes, out)
}

// artifactPath names where in the artifact a number sits, so a refusal points at
// the value rather than at the document.
func artifactPath(path string) string {
	if path == "" {
		return "the artifact root"
	}
	return strings.TrimPrefix(path, "/")
}

const (
	empty = ""
	tab   = "\t"
)

//type IntLiteralNode backend.IntegerLiteralNode

type typeKey[T any] struct{}

// Format_Specifier maps types to their format specifier strings.
var Format_Specifier = map[interface{}]string{}

// Get returns the format specifier string registered for type T.
func Get[T any]() string {
	if val, ok := Format_Specifier[typeKey[T]{}]; ok {
		return val
	}
	return ""
}

// Set registers a format specifier string for type T.
func Set[T any](v string) {
	Format_Specifier[typeKey[T]{}] = v
}

var parserCounter atomic.Int64

// StableID returns a deterministic, language-neutral identity derived from its
// canonical components. The full SHA-256 digest makes collisions practically
// negligible and avoids dependence on process state, machine ABI, or run order.
func StableID(prefix string, components ...string) string {
	hash := sha256.New()
	for _, component := range components {
		hash.Write([]byte(strconv.Itoa(len(component))))
		hash.Write([]byte{':'})
		hash.Write([]byte(component))
	}
	return prefix + "_" + fmt.Sprintf("%x", hash.Sum(nil))
}

// CanonicalIdentityPath normalizes an existing filesystem identity so relative
// versus absolute spelling, separators, symlinks, and Windows case do not change IDs.
func CanonicalIdentityPath(path string) string {
	if absolute, err := filepath.Abs(path); err == nil {
		path = absolute
	}
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		path = resolved
	}
	path = filepath.ToSlash(filepath.Clean(path))
	if runtime.GOOS == "windows" {
		path = strings.ToLower(path)
	}
	return path
}

// ResetIdCounters retains its compatibility role for ephemeral parser labels.
// Durable symbol, context, and symbol-table IDs are content-derived and need no reset.
func ResetIdCounters() {
	parserCounter.Store(0)
}

// GenUnique generates a unique random string of the specified length.
func GenUnique(len int) string {
	return uniuri.NewLen(len)
}

func GenUniqueName(prefix string) string {
	return fmt.Sprintf("%s_%d,", prefix, parserCounter.Add(1))
}

// JSONMarshal encodes the given value to indented JSON without HTML escaping.
func JSONMarshal(t interface{}) ([]byte, error) {
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent(empty, tab)
	err := encoder.Encode(t)
	return buffer.Bytes(), err
}

// Marshal encodes the given value to indented JSON, trimming trailing newlines.
func Marshal(i interface{}) ([]byte, error) {
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent(empty, tab)
	err := encoder.Encode(i)
	return bytes.TrimRight(buffer.Bytes(), "\n"), err
}

// BsonMarshal encodes the given value to BSON bytes.
func BsonMarshal(i any) ([]byte, error) {
	barr, err := bson.Marshal(i)
	return barr, err
}

// UnescapeUnicodeCharactersInJSON converts escaped Unicode sequences in raw JSON back to literal characters.
func UnescapeUnicodeCharactersInJSON(_jsonRaw json.RawMessage) (json.RawMessage, error) {
	str, err := strconv.Unquote(strings.Replace(strconv.Quote(string(_jsonRaw)), `\\u`, `\u`, -1))
	if err != nil {
		return nil, err
	}
	return []byte(str), nil
}

// FormatCode runs the astyle formatter on the file at filePath.
func FormatCode(filePath *string, parent string, isWindows bool) error {
	backendParent := filepath.Join(parent, "backend-gen", "beautifier")
	var backendFormatter string
	if isWindows {
		backendFormatter = filepath.Join(backendParent, "astyle.exe")
	} else {
		backendFormatter = "astyle"

	}
	if filePath != nil {
		srcFilePath := *filePath
		cmd := exec.Command(backendFormatter, "--style=allman", "-n", srcFilePath)
		//cmd.Dir = srcFilePath
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("error: %v, output: %s", err, output)
		}
		os.Remove(srcFilePath + ".orig")
	}
	return nil
}

// Trace returns the file name, line number, and function name of the caller.
func Trace() (string, int, string) {
	pc, file, line, ok := runtime.Caller(2)
	if !ok {
		return "?", 0, "?"
	}

	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return file, line, "?"
	}

	return file, line, fn.Name()
}

// Trace2 returns the file name, line number, and function name of the caller using runtime.Callers.
func Trace2() (string, int, string) {
	pc := make([]uintptr, 15)
	n := runtime.Callers(2, pc)
	frames := runtime.CallersFrames(pc[:n])
	frame, _ := frames.Next()
	return frame.File, frame.Line, frame.Function
}

// GetFrame returns the runtime.Frame at the specified depth in the call stack.
func GetFrame(skipFrames int) runtime.Frame {
	// We need the frame at index skipFrames+2, since we never want runtime.Callers and getFrame
	targetFrameIndex := skipFrames + 2

	// Set size to targetFrameIndex+2 to ensure we have room for one more caller than we need
	programCounters := make([]uintptr, targetFrameIndex+2)
	n := runtime.Callers(0, programCounters)

	frame := runtime.Frame{Function: "unknown"}
	if n > 0 {
		frames := runtime.CallersFrames(programCounters[:n])
		for more, frameIndex := true, 0; more && frameIndex <= targetFrameIndex; frameIndex++ {
			var frameCandidate runtime.Frame
			frameCandidate, more = frames.Next()
			if frameIndex == targetFrameIndex {
				frame = frameCandidate
			}
		}
	}

	return frame
}

// GetCurrentFunc returns the fully qualified name of the current function.
func GetCurrentFunc() string {
	pc, _, _, _ := runtime.Caller(0)
	return runtime.FuncForPC(pc).Name()
}

// HasElements reports whether any row in the given 2D slice contains elements.
func HasElements[T any](matrix [][]T) bool {
	for _, row := range matrix {
		if len(row) > 0 {
			return true
		}
	}
	return false
}

// RemoveAfterUnderscore truncates s at the first occurrence of theString.
func RemoveAfterUnderscore(s string, theString string) string {
	idx := strings.Index(s, theString)
	if idx == -1 {
		return s // No underscore found
	}
	return s[:idx] // Slice up to (but not including) the underscore
}

func RsplitOnce(s, sep string) []string {
	index := strings.LastIndex(s, sep)
	if index == -1 {
		return []string{s}
	}

	return []string{
		s[:index],
		s[index+len(sep):],
	}
}
