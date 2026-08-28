// Package helpers provides shared utility functions for error handling,
// position tracking, serialization, hashing, and code formatting.
package helpers

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"os"
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

// ErrArtifactCodecNotImplemented marks the deliberately deferred .folenc wire
// codec. Project preparation is built against these hooks so choosing protobuf,
// BSON, or another encoding later does not alter compilation ordering or symbol
// ownership.
var ErrArtifactCodecNotImplemented = fmt.Errorf(".folenc artifact codec is not implemented")

// SerializeArtifact is the encoding boundary for a compiled .folenc artifact.
// The logical artifact model lives in the frontend; only its byte encoding is
// intentionally deferred.
func SerializeArtifact(_ any) ([]byte, error) {
	return nil, ErrArtifactCodecNotImplemented
}

// DeserializeArtifact is the decoding boundary for a compiled .folenc artifact.
// out will be populated once the wire codec is selected.
func DeserializeArtifact(_ []byte, _ any) error {
	return ErrArtifactCodecNotImplemented
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
