//go:build !partrace

package main

// collectTrace reports that tracing is not compiled in.
//
// The parser records spans only in a partrace build, so without the tag there is
// nothing to write. Returning ok=false rather than an empty map keeps docgen
// from replacing a good trace.json with an empty one.
func collectTrace(corpus string, limit int) (map[string][]string, bool, error) {
	return nil, false, nil
}
