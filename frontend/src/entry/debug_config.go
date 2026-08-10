package entry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/samkrao/fo-lang/frontend/src/parser"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

/*  folang-debug.json
{
	"debug": {
		"trace": {
			"lexer": true,
			"parser": true
		}
	}
}
*/

const debugConfigFilename = "folang-debug.json"

type debugConfig struct {
	Debug struct {
		Trace struct {
			Lexer  bool `json:"lexer"`
			Parser bool `json:"parser"`
		} `json:"trace"`
	} `json:"debug"`
}

var (
	debugConfigOnce sync.Once
	debugConfigErr  error
)

// configureDebugTracingAtStartup reads the tracing configuration exactly once
// per process. The working directory is intentionally resolved at startup so
// the same file controls both command-line and editor-launched executions.
func configureDebugTracingAtStartup() error {
	debugConfigOnce.Do(func() {
		workingDirectory, err := os.Getwd()
		if err != nil {
			debugConfigErr = fmt.Errorf("resolve working directory: %w", err)
			return
		}
		debugConfigErr = loadDebugTraceConfig(filepath.Join(workingDirectory, debugConfigFilename))
	})
	return debugConfigErr
}

func loadDebugTraceConfig(path string) error {
	// Missing files and omitted properties both mean tracing is disabled.
	parser.DEBUG_TRACE = false
	scanlex.DEBUG_TRACE = false

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}

	var config debugConfig
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&config); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fmt.Errorf("decode %s: unexpected trailing JSON value", path)
		}
		return fmt.Errorf("decode %s: %w", path, err)
	}

	parser.DEBUG_TRACE = config.Debug.Trace.Parser
	scanlex.DEBUG_TRACE = config.Debug.Trace.Lexer
	return nil
}
