package parser_test

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/samkrao/fo-lang/src/parser"
)

// NodeName names a node's own grammatical form — "IfStmt", "ConditionalStmt" —
// so that a tree printed as JSON says what each object IS. Without it a reader
// meeting `{Condition, Consequent, Alternate}` has to infer the node from its
// shape, and the artifact records a node as its exported fields alone.
//
// It is a FIELD rather than a method so that it survives any marshalling, which
// means it has to be populated at every construction site. That is the same
// hazard TestEveryNodeCarriesASpan exists for and it fails the same silent way:
// a missed site still compiles, still parses, and simply reports "".
//
// Two failures are possible and both matter. An EMPTY name is a construction site
// that was never stamped. A WRONG name is worse — nine statement nodes embed
// FunctionDeclarationStmt and inherit its field, so a copy-paste that stamps the
// embedded name leaves a decorator claiming to be a function declaration, and
// every reader downstream believes it.
func TestEveryNodeCarriesItsNodeName(t *testing.T) {
	for _, path := range conformanceFixtures(t, "accepted") {
		path := path
		t.Run(fixtureName(path), func(t *testing.T) {
			result := parser.ParseFile(
				readFixture(t, path), "nodenames",
				filepath.Dir(path), fixtureBasename(path), "",
			)
			if len(result.Diagnostics) != 0 {
				t.Fatalf("accepted fixture produced diagnostics: %v", result.Diagnostics)
			}

			missing, wrong := map[string]int{}, map[string]int{}
			walkNodeNames(reflect.ValueOf(result.Root), map[uintptr]bool{}, func(type_, name string) {
				switch {
				case name == "":
					missing[type_]++
				case name != type_:
					wrong[type_+" claims "+name]++
				}
			})
			if len(missing) != 0 {
				t.Errorf("nodes with no NodeName:\n%s", formatCounts(missing))
			}
			if len(wrong) != 0 {
				t.Errorf("nodes naming the wrong form:\n%s", formatCounts(wrong))
			}
		})
	}
}

// walkNodeNames mirrors walkNodes: reflection rather than the ast visitor, so a
// node shape the visitor does not handle cannot slip past unchecked.
func walkNodeNames(v reflect.Value, seen map[uintptr]bool, visit func(type_, name string)) {
	if !v.IsValid() {
		return
	}

	switch v.Kind() {
	case reflect.Interface, reflect.Ptr:
		if v.IsNil() {
			return
		}
		if v.Kind() == reflect.Ptr {
			if seen[v.Pointer()] {
				return
			}
			seen[v.Pointer()] = true
		}
		walkNodeNames(v.Elem(), seen, visit)
		return

	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			walkNodeNames(v.Index(i), seen, visit)
		}
		return

	case reflect.Map:
		for _, key := range v.MapKeys() {
			walkNodeNames(v.MapIndex(key), seen, visit)
		}
		return

	case reflect.Struct:
		// An empty struct is an unset by-value field, not a node that reached
		// the tree; see walkNodes for why that exclusion is safe. Here it must
		// discount NodeName itself, which a stamped placeholder always carries.
		if isNode(v) && !isEmptyNode(v) {
			if field := v.FieldByName("NodeName"); field.IsValid() && field.Kind() == reflect.String {
				visit(v.Type().Name(), field.String())
			}
		}
		// Symbol records are metadata: they form cycles and carry no node names.
		if strings.HasSuffix(v.Type().PkgPath(), "/context") {
			return
		}
		for i := 0; i < v.NumField(); i++ {
			if !v.Type().Field(i).IsExported() {
				continue
			}
			walkNodeNames(v.Field(i), seen, visit)
		}
		return
	}
}
