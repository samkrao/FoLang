package parser

import (
	"reflect"
	"strings"
	"testing"

	"github.com/samkrao/fo-lang/frontend/src/ast"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// Built-in metadata parsing — docs/language-ref.md, "Built-in Metadata Parsing".
//
// The registry closes the metadata NAME and deliberately not its fields. Once a
// built-in form is recognized the parser "must collect and preserve the complete
// metadata application, including every supplied positional argument, named
// argument, field, attribute, and argument expression", and a field the frontend
// has no handling for "is still accepted, collected, and preserved AS PARSED".
//
// Asserting only that such a file parses is not enough, and is how a
// string-flattening bug first escaped review: `extensions=[a, b]` parsed happily
// and reached the AST as the text "[a b]". These tests read the preserved value
// back, because preserving the SHAPE is the requirement — a rendered value cannot
// be recovered by a later stage.

// parseEntryDirective parses one entry file and returns its first statement,
// failing the test if the file produced any diagnostic.
func parseEntryDirective(t *testing.T, source string) ast.Stmt {
	t.Helper()
	root, p := parseEntrySourceCollecting(t, source)
	if len(p.diags) != 0 {
		t.Fatalf("directive produced diagnostics: %v", p.diags)
	}
	body := root.(ast.Application).Body
	if len(body) == 0 {
		t.Fatal("entry file produced no statements")
	}
	return body[0]
}

// parseEntrySourceCollecting parses one entry file and hands back the parser, so
// a caller can inspect the diagnostics instead of failing on them.
func parseEntrySourceCollecting(t *testing.T, source string) (ast.Stmt, *parser) {
	t.Helper()
	toks := normalizeTokens(scanlex.Tokenize(source, "appl.fol"))
	p, _ := newParser(toks)
	p.file = fileinfo{Basename: "appl.fol", LocationKnown: true, Source: classifySourceFilename("appl.fol")}
	return p.parseCompilationUnit(), p
}

func TestUseDirectivePreservesUnknownFieldValuesAsParsed(t *testing.T) {
	for _, tc := range []struct {
		name  string
		field string
		want  any
	}{
		{"enabled", `enabled=co.const.true`, true},
		{"retries", `retries=3`, int64(3)},
		{"scope", `scope="file"`, "file"},
		{"extensions", `extensions=[upperCase, lowerCase]`, []any{"upperCase", "lowerCase"}},
		{"options", `options={mode: eager}`, map[string]any{"mode": "eager"}},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			stmt := parseEntryDirective(t, `@co.ddap.use(from="tu", `+tc.field+`)`)

			directive, ok := stmt.(ast.UseStmtDirective)
			if !ok {
				t.Fatalf("use directive = %T, want ast.UseStmtDirective", stmt)
			}

			// The two known fields keep their own reduced representation.
			if got := directive.Type["from"]; !reflect.DeepEqual(got, []string{"tu"}) {
				t.Errorf(`Type["from"] = %#v, want ["tu"]`, got)
			}

			got, kept := directive.Preserved[tc.name]
			if !kept {
				t.Fatalf("field %q was dropped; Preserved = %#v", tc.name, directive.Preserved)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Preserved[%q] = %#v (%T), want %#v (%T)", tc.name, got, got, tc.want, tc.want)
			}

			// An unknown field must not also land in the string map the semantic
			// phase reads, where its structure would be unrecoverable.
			if _, flattened := directive.Type[tc.name]; flattened {
				t.Errorf("field %q was also flattened into Type, which loses its shape", tc.name)
			}
		})
	}
}

// The alias directive keeps an unknown field with its shape too, and never lets
// one displace the target or the alias name the frontend already resolved.
func TestAliasDirectivePreservesUnknownFieldValuesAsParsed(t *testing.T) {
	stmt := parseEntryDirective(t, `@co.ddap.alias(co.out, as="out", tags=[fast, quiet])`)

	directive, ok := stmt.(ast.DirectiveStmt)
	if !ok {
		t.Fatalf("alias directive = %T, want ast.DirectiveStmt", stmt)
	}
	if got := directive.Parameters["target"]; got != "co.out" {
		t.Errorf(`Parameters["target"] = %#v, want "co.out"`, got)
	}
	if got := directive.Parameters["as"]; got != "out" {
		t.Errorf(`Parameters["as"] = %#v, want "out"`, got)
	}
	want := []any{"fast", "quiet"}
	if got := directive.Parameters["tags"]; !reflect.DeepEqual(got, want) {
		t.Errorf(`Parameters["tags"] = %#v (%T), want %#v`, got, got, want)
	}
}

// A repeated field would overwrite the value already collected, which loses part
// of the application the reference requires to be preserved. Both directives that
// build their own field map report it rather than resolving it silently — and for
// the alias directive that includes a field that repeats what the positional
// target or the "as" field already bound.
func TestDirectivesRejectARepeatedField(t *testing.T) {
	for _, tc := range []struct {
		name   string
		source string
		want   string
	}{
		{"use-unknown", `@co.ddap.use(from="tu", scope="file", scope="package")`,
			`the use field "scope" is given more than once`},
		{"alias-unknown", `@co.ddap.alias(co.out, as="out", scope="file", scope="package")`,
			`the alias field "scope" is already supplied by this directive`},
		{"alias-as", `@co.ddap.alias(co.out, as="out", as="other")`,
			`the alias field "as" is already supplied by this directive`},
		{"alias-target", `@co.ddap.alias(co.out, as="out", target=co.in)`,
			`the alias field "target" is already supplied by this directive`},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			_, p := parseEntrySourceCollecting(t, tc.source)

			if len(p.diags) == 0 {
				t.Fatalf("a repeated field parsed without a diagnostic: %s", tc.source)
			}
			if first := p.diags[0].AsString(); !strings.Contains(first, tc.want) {
				t.Errorf("first diagnostic does not name the repeated field\n  want contains: %s\n  got: %s",
					tc.want, first)
			}
		})
	}
}

// Nothing above may weaken the field shapes the frontend does understand.
func TestKnownDirectiveFieldsStayValidated(t *testing.T) {
	for _, tc := range []struct {
		name   string
		source string
	}{
		{"alias-target-not-co", `@co.ddap.alias(myapp.thing, as="t")`},
		{"alias-name-unspellable", `@co.ddap.alias(co.out, as="1out")`},
		{"use-methods-unbracketed", `@co.ddap.use(methods=upperCase)`},
		{"use-malformed-unknown-value", `@co.ddap.use(methods=[a], scope=)`},
		{"alias-trailing-comma", `@co.ddap.alias(co.out, as="out",)`},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			defer func() { _ = recover() }()
			_, p := parseEntrySourceCollecting(t, tc.source)
			if len(p.diags) == 0 {
				t.Errorf("expected a diagnostic for %s", tc.source)
			}
		})
	}
}
