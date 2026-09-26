package scanlex_test

import (
	"testing"

	"github.com/samkrao/fo-lang/src/scanlex"
)

func assertStringSet(t *testing.T, got, want []string) {
	t.Helper()
	gotCounts := make(map[string]int, len(got))
	for _, value := range got {
		gotCounts[value]++
	}
	wantCounts := make(map[string]int, len(want))
	for _, value := range want {
		wantCounts[value]++
	}
	if len(got) != len(want) || len(gotCounts) != len(wantCounts) {
		t.Fatalf("registry = %v, want %v", got, want)
	}
	for value, count := range wantCounts {
		if gotCounts[value] != count {
			t.Fatalf("registry = %v, want %v (mismatch at %q)", got, want, value)
		}
	}
}

func TestReservedWordsAndBuiltinPackageRootsMatchCurrentReference(t *testing.T) {
	if len(scanlex.Reserved_lu) != 3 || scanlex.Reserved_lu["co"] != scanlex.KEYWORD ||
		scanlex.Reserved_lu["this"] != scanlex.KEYWORD ||
		scanlex.Reserved_lu["fΦλ"] != scanlex.RESERVEDWORD {
		t.Fatalf("Reserved_lu = %#v, want only co, this, and fΦλ", scanlex.Reserved_lu)
	}
	if len(scanlex.UnsupportedObjects) != 0 {
		t.Fatalf("UnsupportedObjects = %v, want none", scanlex.UnsupportedObjects)
	}
	assertStringSet(t, scanlex.KeyWords_me["co"], []string{
		"http", "tcp", "udp", "sys", "os", "meta", "native", "in", "out",
		"regex", "crypto", "dap", "ddap", "pdap", "const", "encoding", "utils",
		"dynamic", "runtime", "compiletime", "macro", "pattern", "control", "cpca",
		"hokrlt", "operator", "hw", "stex",
	})

	tokens := meaningful(scanlex.Tokenize("fΦλ", "reference.fol"))
	if len(tokens) != 1 {
		t.Fatalf("Tokenize(fΦλ) = %#v, want one reserved token", tokens)
	}
	assertKindValue(t, tokens[0], scanlex.RESERVEDWORD, "fΦλ")
}

func TestBuiltinRegistriesMatchCurrentReference(t *testing.T) {
	assertStringSet(t, scanlex.Builtin_types, []string{
		"co.string", "co.int", "co.bit", "co.double", "co.float", "co.long",
		"co.byte", "co.char", "co.any", "co.bool", "co.void", "co.value",
		"co.untyped", "co.word", "co.MatchBindings", "co.number", "co.uninit",
		"co.error", "co.AbstractError", "co.literal", "co.delegate", "co.condition",
		"co.variants", "co.tag", "co.hokrlt", "co.newtype", "co.opaquetype",
		"co.subtype", "co.supertype", "co.dependentType", "co.polymorphic",
		"co.refinementType", "co.associatedType", "co.predicateType", "co.data",
		"co.type", "co.generic", "co.shape",
	})
	assertStringSet(t, scanlex.Builtin_Kinds, []string{
		"co.struct", "co.cstruct", "co.class", "co.interface", "co.union",
		"co.object", "co.instance", "co.matcher", "co.loader", "co.trait",
		"co.mixin", "co.extension", "co.typeclass", "co.module", "co.unit",
		"co.block", "co.kind", "co.signature", "co.function", "co.callable",
		"co.boundcallable", "co.enum", "co.symbol", "co.component",
	})
	assertStringSet(t, scanlex.Built_In_Collections, []string{
		"co.List", "co.Set", "co.Map", "co.Tree", "co.Trie", "co.Array",
		"co.Tuple", "co.Comparable", "co.Stack", "co.Queue", "co.StructObject",
		"co.ClassObject", "co.ModuleObject", "co.InstanceObject", "co.ObjectObject",
		"co.Matrix",
	})
}

func TestCurrentReferenceBuiltinsUseDirectCoPaths(t *testing.T) {
	tests := []struct {
		spelling string
		kind     scanlex.TokenKind
	}{
		{"co.int", scanlex.BUILT_IN_TYPE},
		{"co.polymorphic", scanlex.BUILT_IN_TYPE},
		{"co.struct", scanlex.BUILT_IN_KIND},
		{"co.callable", scanlex.BUILT_IN_KIND},
		{"co.List", scanlex.BUILT_IN_COLLECTIONS},
		{"co.operator", scanlex.OPERATOR_SOURCE_KIND},
		{"co.http", scanlex.BUIL_IN_STMT_EXPRS},
		{"co.utils.makeImmutable", scanlex.BUIL_IN_STMT_EXPRS},
		{"co.hw.cpu", scanlex.BUIL_IN_STMT_EXPRS},
	}

	for _, test := range tests {
		t.Run(test.spelling, func(t *testing.T) {
			tokens := meaningful(scanlex.Tokenize(test.spelling, "reference.fol"))
			if len(tokens) != 1 {
				t.Fatalf("tokens = %#v, want one folded token", tokens)
			}
			assertKindValue(t, tokens[0], test.kind, test.spelling)
		})
	}
}

func TestRemovedBuiltinNamespacesAreNotFoldedAsBuiltinTypes(t *testing.T) {
	for _, spelling := range []string{"co.lang.int", "co.core.List"} {
		t.Run(spelling, func(t *testing.T) {
			tokens := meaningful(scanlex.Tokenize(spelling, "reference.fol"))
			for _, token := range tokens {
				if token.Kind == scanlex.BUILT_IN_TYPE || token.Kind == scanlex.BUILT_IN_KIND ||
					token.Kind == scanlex.BUILT_IN_COLLECTIONS {
					t.Fatalf("removed spelling %q retained built-in classification: %#v", spelling, tokens)
				}
			}
		})
	}
}

func TestMetadataRegistryMatchesCurrentReference(t *testing.T) {
	assertStringSet(t, scanlex.PDADs[scanlex.PRAGMA], []string{
		"@co.pdap.threadpool", "@co.pdap.schedularpool",
	})
	assertStringSet(t, scanlex.PDADs[scanlex.DIRECTIVE], []string{
		"@co.ddap.import", "@co.ddap.dynamicruntime", "@co.ddap.use",
		"@co.ddap.alias", "@co.ddap.dynamicdispatch", "@co.ddap.overload",
	})
	assertStringSet(t, scanlex.PDADs[scanlex.ANNOTATION], []string{
		"@co.dap.extend", "@co.dap.template", "@co.dap.macro", "@co.dap.operator",
		"@co.dap.annotation", "@co.dap.library", "@co.dap.native", "@co.dap.class",
		"@co.dap.static", "@co.dap.object", "@co.dap.inline", "@co.dap.ctfe",
		"@co.dap.friend", "@co.dap.sealed", "@co.dap.extension", "@co.dap.override",
		"@co.dap.implement", "@co.dap.virtual", "@co.dap.abstract", "@co.dap.delegate",
		"@co.dap.typeclass", "@co.dap.matcher", "@co.dap.constructor", "@co.dap.oops",
		"@co.dap.extends", "@co.dap.hokrlt", "@co.dap.indexer", "@co.dap.generic",
		"@co.dap.comptime", "@co.dap.typefromvalue", "@co.dap.local", "@co.dap.private",
		"@co.dap.public", "@co.dap.compose", "@co.dap.guard", "@co.dap.package",
		"@co.dap.protected", "@co.dap.internal", "@co.dap.export", "@co.dap.eager",
		"@co.dap.lazy", "@co.dap.packed", "@co.dap.declare", "@co.dap.implementation",
		"@co.dap.simd", "@co.dap.reflection", "@co.dap.mop", "@co.dap.nested",
		"@co.dap.inner", "@co.dap.final", "@co.dap.const", "@co.dap.decorator",
		"@co.dap.specialize", "@co.dap.scope", "@co.dap.symbol", "@co.dap.with",
	})
	assertStringSet(t, scanlex.PDADs[scanlex.DECORATOR], []string{
		"@co.dap.before", "@co.dap.after", "@co.dap.around", "@co.dap.effects",
		"@co.dap.onEffect", "@co.dap.defer", "@co.dap.callable", "@co.dap.executionmodel",
	})

	if !scanlex.IsBuiltinAnnotationMetadataName("@co.dap.extend") {
		t.Fatal("@co.dap.extend must be a built-in annotation")
	}
	for _, removed := range []string{"@co.dap.module", "@co.dap.instance"} {
		if scanlex.IsBuiltinMetadataName(removed) {
			t.Fatalf("removed metadata form %q is still registered", removed)
		}
		tokens := meaningful(scanlex.Tokenize(removed, "reference.fol"))
		if len(tokens) != 1 {
			t.Fatalf("Tokenize(%q) = %#v, want one UNKNOWN token", removed, tokens)
		}
		assertKindValue(t, tokens[0], scanlex.UNKNOWN, removed)
	}
}

func TestLibraryKindsMatchCurrentCapabilityDomains(t *testing.T) {
	want := []string{"application", "dynamicvmrt", "native"}
	assertStringSet(t, scanlex.LIB_KINDS, want)
}

func TestRemovedContextualWordsAreIdentifiers(t *testing.T) {
	for _, spelling := range []string{"for", "forall", "let"} {
		tokens := meaningful(scanlex.Tokenize(spelling, "reference.fol"))
		if len(tokens) != 1 {
			t.Fatalf("Tokenize(%q) = %#v, want one identifier", spelling, tokens)
		}
		assertKindValue(t, tokens[0], scanlex.IDENTIFIER, spelling+"_fo")
	}
}

func TestDollarContextAndControlProductionsRemainSeparated(t *testing.T) {
	tokens := meaningful(scanlex.Tokenize("$ $1 $=> $->> $->|", "reference.fol"))
	want := []struct {
		kind  scanlex.TokenKind
		value string
	}{
		{scanlex.CONTEXT_SIGIL_DOLLAR, "$"},
		{scanlex.BIND_VAR, "$1"},
		{scanlex.CONTEXT_SIGIL_DOLLAR, "$"},
		{scanlex.EQGT, "=>"},
		{scanlex.CONTEXT_SIGIL_DOLLAR, "$"},
		{scanlex.ARROW_GT, "->>"},
		{scanlex.CONTEXT_SIGIL_DOLLAR, "$"},
		{scanlex.ARROW_PIPE, "->|"},
	}
	if len(tokens) != len(want) {
		t.Fatalf("tokens = %#v, want %#v", tokens, want)
	}
	for i, expected := range want {
		assertKindValue(t, tokens[i], expected.kind, expected.value)
	}
}

func TestOperatorSourceConstantsExcludeRemovedFutureForms(t *testing.T) {
	want := map[string]string{
		"co.operator.fixity.infix":        "infix",
		"co.operator.fixity.prefix":       "prefix",
		"co.operator.fixity.postfix":      "postfix",
		"co.operator.associativity.left":  "left",
		"co.operator.associativity.right": "right",
		"co.operator.associativity.none":  "none",
		"co.operator.arity.unary":         "unary",
		"co.operator.arity.binary":        "binary",
	}
	if len(scanlex.Operator_source_constants) != len(want) {
		t.Fatalf("Operator_source_constants = %#v, want %#v", scanlex.Operator_source_constants, want)
	}
	for spelling, value := range want {
		if scanlex.Operator_source_constants[spelling] != value {
			t.Fatalf("Operator_source_constants[%q] = %q, want %q", spelling, scanlex.Operator_source_constants[spelling], value)
		}
	}

	for _, spelling := range []string{
		"co.operator.fixity.circumfix",
		"co.operator.fixity.postcircumfix",
		"co.operator.fixity.precircumfix",
		"co.operator.fixity.mixfix",
		"co.operator.fixity.ternary",
		"co.operator.fixity.distfix",
		"co.operator.arity.ternary",
	} {
		if _, exists := scanlex.Operator_source_constants[spelling]; exists {
			t.Fatalf("removed operator-source constant %q is still registered", spelling)
		}
	}
}

func TestRemovedControlOperatorsBecomeUnknownWholeRuns(t *testing.T) {
	for _, spelling := range []string{"^=>", "==>>"} {
		tokens := meaningful(scanlex.Tokenize(spelling, "reference.fol"))
		if len(tokens) != 1 {
			t.Fatalf("Tokenize(%q) = %#v, want one UNKNOWN token", spelling, tokens)
		}
		assertKindValue(t, tokens[0], scanlex.UNKNOWN, spelling)
	}
}

func TestPointerDegreeRemainsContextualSymbolicRun(t *testing.T) {
	tokens := meaningful(scanlex.Tokenize("*****", "reference.fol"))
	if len(tokens) != 1 {
		t.Fatalf("tokens = %#v, want one pointer-degree token", tokens)
	}
	assertKindValue(t, tokens[0], scanlex.SYMBOLIC_RUN, "*****")
}

func TestMalformedNumericSpellingsRemainOneUnknownToken(t *testing.T) {
	for _, spelling := range []string{
		"123hr",
		"08",
		"0b102",
		"0xG",
		"0x1.8",
		"1'000",
		"0x1'a",
		"0b1011'0010",
		".10",
		"1e+",
	} {
		t.Run(spelling, func(t *testing.T) {
			tokens := meaningful(scanlex.Tokenize(spelling, "reference.fol"))
			if len(tokens) != 1 {
				t.Fatalf("tokens = %#v, want one UNKNOWN token", tokens)
			}
			assertKindValue(t, tokens[0], scanlex.UNKNOWN, spelling)
		})
	}
}

func TestCurrentNumericFamiliesRemainNumbers(t *testing.T) {
	for _, spelling := range []string{
		"0",
		"077",
		"0b1010",
		"0x1a",
		"42uL",
		"1.0",
		"08.5",
		"1e5",
		"0x1.8p3",
		"1.0f32",
	} {
		t.Run(spelling, func(t *testing.T) {
			tokens := meaningful(scanlex.Tokenize(spelling, "reference.fol"))
			if len(tokens) != 1 {
				t.Fatalf("tokens = %#v, want one NUMBER token", tokens)
			}
			assertKindValue(t, tokens[0], scanlex.NUMBER, spelling)
		})
	}
}
