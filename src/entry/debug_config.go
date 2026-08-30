package entry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/samkrao/fo-lang/src/parser"
	"github.com/samkrao/fo-lang/src/project"
	"github.com/samkrao/fo-lang/src/scanlex"
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
// per process, from the project root marked by fol-conf.yaml.
func configureDebugTracingAtStartup(sourceFile string) error {
	debugConfigOnce.Do(func() {
		path, err := projectDebugConfigPath(sourceFile)
		if err != nil {
			debugConfigErr = err
			return
		}
		debugConfigErr = loadDebugTraceConfig(path)
	})
	return debugConfigErr
}

// projectDebugConfigPath locates the nearest fol-conf.yaml above sourceFile
// and returns the sibling folang-debug.json. With no project marker, the
// returned path is deliberately nonexistent so tracing remains disabled.
func projectDebugConfigPath(sourceFile string) (string, error) {
	absSource, err := filepath.Abs(sourceFile)
	if err != nil {
		return "", fmt.Errorf("resolve source file %s: %w", sourceFile, err)
	}

	dir := filepath.Dir(absSource)
	for {
		marker := filepath.Join(dir, project.MarkerFilename)
		if _, statErr := os.Stat(marker); statErr == nil {
			return filepath.Join(dir, debugConfigFilename), nil
		} else if !os.IsNotExist(statErr) {
			return "", fmt.Errorf("check project marker %s: %w", marker, statErr)
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return filepath.Join(absSource, debugConfigFilename), nil
		}
		dir = parent
	}
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
