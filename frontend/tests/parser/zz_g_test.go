package parser_test

import (
	"fmt"
	"testing"

	"github.com/samkrao/fo-lang/frontend/src/foerrors"
)

func TestZZG(t *testing.T) {
	foerrors.GenPanic = true
	for _, src := range []string{
		`x Vector(co.lang.int);`,
		`x Vector(co.lang.int) = y;`,
		`x F(A)(B) = y;`,
		`Gen co.lang.unit = { f(a Vector(co.lang.int))->()={ co.nop; } }`,
		`S co.lang.struct = { items Vector(co.lang.int); }`,
		`Gen co.lang.unit = { f()->(Vector(co.lang.int))={ co.nop; } }`,
	} {
		out, crashed := try(src)
		status := "ACCEPT"
		if crashed {
			status = "CRASH"
		} else if len(out) > 0 {
			status = "REJECT"
		}
		fmt.Printf("%-62s %s\n", src, status)
	}
}
