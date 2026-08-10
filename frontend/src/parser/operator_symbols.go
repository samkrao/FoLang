package parser

import (
	"fmt"
	"strconv"

	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// operatorDeclaration is the syntax and optional optimization metadata read
// from the project's dedicated srclib/operators/library.fol bootstrap source. Ordinary FoLang
// files may implement a registered symbol, but never add to this catalog.
type operatorDeclaration struct {
	Options map[string]any
}

// operatorCollection supplies the same immutable project catalog to scanning
// and Pratt parsing.
type operatorCollection struct {
	Custom       *scanlex.CustomOperators
	Declarations []operatorDeclaration
}

// declaredOperatorsIn builds the lexical registry for one ordinary source
// file. source and basename are retained in the signature for compatibility
// with the parse entry point; declarations are intentionally taken only from
// the pre-parsed srclib/operators/library.fol surface, which is the one place a
// project-local symbol may be registered.
func declaredOperatorsIn(_ string, _ string, inherited []operatorDeclaration) operatorCollection {
	declarations := append([]operatorDeclaration(nil), inherited...)
	specs := make([]scanlex.OperatorSpec, 0, len(declarations))
	for _, declaration := range declarations {
		symbol := operatorOptionText(declaration.Options, "symbol")
		fixity := operatorOptionText(declaration.Options, "fixity")
		if symbol != "" && fixity != "" {
			specs = append(specs, scanlex.OperatorSpec{Symbol: symbol, Fixity: fixity})
		}
	}
	return operatorCollection{
		Custom:       scanlex.NewCustomOperatorsWithSpecs(specs),
		Declarations: declarations,
	}
}

// literalText decodes the quoted spelling used by operator-source symbol lists
// and by tests that inspect bootstrap metadata. Alpha literals contain no escape
// sequences, so removing the delimiters preserves the exact operator spelling.
func literalText(tok scanlex.Token) (string, bool) {
	switch tok.Kind {
	case scanlex.CHAR, scanlex.STRING:
		if len(tok.Value) < 2 {
			return "", false
		}
		return tok.Value[1 : len(tok.Value)-1], true
	}
	return "", false
}

// operatorArityCount normalizes the named and numeric spellings accepted by
// operator-arity. It is shared by source validation and implementation arity
// checking so `binary` and `2` cannot acquire different behavior.
func operatorArityCount(value any) (int, bool) {
	switch value := value.(type) {
	case int:
		return value, value > 0
	case int64:
		converted := int(value)
		return converted, value > 0 && int64(converted) == value
	case string:
		switch value {
		case "unary":
			return 1, true
		case "binary":
			return 2, true
		case "ternary":
			return 3, true
		}
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			return parsed, true
		}
	}
	return 0, false
}

func describeOperatorArity(value any) string {
	if count, ok := operatorArityCount(value); ok {
		return fmt.Sprintf("%d", count)
	}
	return fmt.Sprint(value)
}
