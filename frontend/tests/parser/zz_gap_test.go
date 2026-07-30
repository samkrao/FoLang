package parser_test

// Temporary audit harness: probes grammar features not covered by the earlier snippet run,
// plus the lexical forms section 12 requires. Deleted after use.

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/samkrao/fo-lang/frontend/src/foerrors"
	"github.com/samkrao/fo-lang/frontend/src/parser"
)

func try(src string) (out string, crashed bool) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	done := make(chan string)
	go func() {
		buf := make([]byte, 1<<20)
		n, _ := r.Read(buf)
		done <- string(buf[:n])
	}()
	func() {
		defer func() {
			if rec := recover(); rec != nil {
				if fmt.Sprint(rec) != "Error" {
					crashed = true
				}
			}
		}()
		parser.Parse(src, "s", ".", "s.fol", "", "program", "program", true)
	}()
	w.Close()
	os.Stdout = old
	return <-done, crashed
}

func probe(name, src string) {
	out, crashed := try(src)
	bad := crashed
	var first string
	for _, l := range strings.Split(out, "\n") {
		if strings.Contains(l, "Syntax") || strings.Contains(l, "UnSupported") || strings.Contains(l, "Invalid") {
			bad = true
			first = strings.TrimSpace(l)
			break
		}
	}
	switch {
	case crashed:
		fmt.Printf("CRASH   %-34s\n", name)
	case bad:
		fmt.Printf("REJECT  %-34s %s\n", name, first)
	default:
		fmt.Printf("ACCEPT  %-34s\n", name)
	}
}

func TestZZGaps(t *testing.T) {
	foerrors.GenPanic = true

	fmt.Println("--- parser productions not in the earlier run ---")
	probe("receiver-clause (named)", `Gen co.lang.unit = { (e Employee) describe()->(co.lang.string) = { this.return "x"; } }`)
	probe("receiver-clause (bare type)", `Gen co.lang.unit = { (Employee) describe()->(co.lang.string) = { this.return "x"; } }`)
	probe("annotation-arrow-pair", `@co.dap.oops("a" => "b")`+"\nx co.lang.int = 1;")
	probe("annotation-list", `@co.dap.generic(types=[co.lang.int, co.lang.float])`+"\nx co.lang.int = 1;")
	probe("tuple-assignment-target", `(a, b), c = x, y;`)
	probe("qualified-function-reference", `@co.dap.local(target=hr.find(co.lang.int)->(Employee))`+"\nx co.lang.int = 1;")
	probe("lifecycle member access", `x := obj.@@init;`)
	probe("empty-statement", `;`)
	probe("index multi-dim", `v := grid[row, col];`)
	probe("postfix on block", `v := { 1 }.to_str();`)
	probe("forall decl prefix", `forall(T) LinkedList co.lang.struct = { value T; }`)
	probe("nested generic apply", `x F(A)(B) = y;`)
	probe("thunk param", `Gen co.lang.unit = { f(a co.lang.int->(^))->()= { co.nop; } }`)
	probe("variadic + default mix", `Gen co.lang.unit = { f(a co.lang.int = 1, ...b co.lang.char)->()={ co.nop; } }`)
	probe("chained derivation", `p co.lang.int->(*)->(&);`)
	probe("match no cases", `x.match;`)
	probe("nested match in arg", `f(x.match.case(0 => 1).default(2));`)

	fmt.Println("\n--- section 12 lexical forms ---")
	probe("line comment", "// hi\nx co.lang.int = 1;")
	probe("block comment", "/* hi */\nx co.lang.int = 1;")
	probe("block comment inline", `x /* mid */ co.lang.int = 1;`)
	probe("char escape \\n", `c co.lang.char = '\n';`)
	probe("char escape \\t", `c co.lang.char = '\t';`)
	probe("string with escape", `s co.lang.string = "a\nb";`)
	probe("raw string literal", `s co.lang.string = R"(no \n escape)";`)
	probe("encoding prefix u8", `s co.lang.string = u8"text";`)
	probe("encoding prefix L char", `c co.lang.char = L'c';`)
	probe("hex integer", `a := 0xFF;`)
	probe("binary integer", `a := 0b1011;`)
	probe("octal integer", `a := 0755;`)
	probe("integer suffix", `a := 10u;`)
	probe("exponent", `a := 1e5;`)
	probe("signed exponent", `a := 1e-5;`)
	probe("hex float", `a := 0x1.8p3;`)
	probe("float suffix", `a := 3.14f;`)
	probe("adjacent strings", `s := "a" "b";`)
	probe("reserved glyph lambda", "x := λ;")
	probe("rejected float 1.", `a := 1.;`)
	probe("digit separator (must reject)", `a := 1'000;`)
}
