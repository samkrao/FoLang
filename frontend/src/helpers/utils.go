// Package helpers provides shared utility functions for error handling,
// position tracking, serialization, hashing, and code formatting.
package helpers

import (
	"bytes"
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

var ctxCounter atomic.Int64
var symCounter atomic.Int64
var parserCounter atomic.Int64

// Context and symbol-table identifiers are a monotonic counter and nothing else.
//
// They used to carry a four-character random suffix, which made two parses of
// the same unchanged source produce different identifiers. That is invisible to
// a batch compile, which parses each file once and exits, but it defeats every
// consumer that compares one parse with the next: an editor cannot diff two
// parses of a buffer to compute a minimal diagnostic update, and nothing keyed
// on symbol identity can be cached across a re-parse.
//
// The counter alone is still unique within a process — which is all these ids
// were ever required to be, since they are per-parse scope handles rather than
// durable references — and it is reproducible for identical input parsed in the
// same order. Reset makes that reproducibility available to a consumer that
// wants two parses of one file to agree exactly.

func NewContextId() string {
	return fmt.Sprintf("ctx_%d", ctxCounter.Add(1))
}

func NewSymbolTableId() string {
	return fmt.Sprintf("sym_%d", symCounter.Add(1))
}

// ResetIdCounters restarts the identifier counters.
//
// A consumer that re-parses one file and wants the new tree's identifiers to
// match the old one's calls this first. It is deliberately explicit rather than
// automatic: the counters are process-wide, so resetting them while another
// parse is in flight would hand out colliding ids.
func ResetIdCounters() {
	ctxCounter.Store(0)
	symCounter.Store(0)
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
