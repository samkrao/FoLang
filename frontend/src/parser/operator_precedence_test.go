package parser

import (
	"fmt"
	"strings"
	"testing"

	"github.com/samkrao/fo-lang/frontend/src/ast"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

func TestCustomInfixPrecedenceCrossesPower(t *testing.T) {
	tests := []struct {
		name       string
		precedence int64
		want       string
	}{
		{"above power", 95, "((a <+> b) ** c)"},
		{"below power", 85, "(a <+> (b ** c))"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			declaration := testOperatorDeclaration("<+>", "infix", test.precedence, "left", "binary")
			expr := parseOperatorExpression(t, "a <+> b ** c", declaration)
			if got := operatorExpressionShape(expr); got != test.want {
				t.Fatalf("shape = %s, want %s", got, test.want)
			}
		})
	}
}

func TestPowerAndBuiltInPrefixGroupingIsPreserved(t *testing.T) {
	tests := []struct {
		source string
		want   string
	}{
		{"-a ** b", "(- (a ** b))"},
		{"a ** -b", "(a ** (- b))"},
		{"a ** b ** c", "(a ** (b ** c))"},
	}
	for _, test := range tests {
		t.Run(test.source, func(t *testing.T) {
			if got := operatorExpressionShape(parseOperatorExpression(t, test.source)); got != test.want {
				t.Fatalf("shape = %s, want %s", got, test.want)
			}
		})
	}
}

func TestCustomPrefixPrecedenceCrossesPower(t *testing.T) {
	tests := []struct {
		name       string
		precedence int64
		want       string
	}{
		{"above power", 95, "((!! a) ** b)"},
		{"below power", 85, "(!! (a ** b))"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			declaration := testOperatorDeclaration("!!", "prefix", test.precedence, "right", "unary")
			expr := parseOperatorExpression(t, "!! a ** b", declaration)
			if got := operatorExpressionShape(expr); got != test.want {
				t.Fatalf("shape = %s, want %s", got, test.want)
			}
		})
	}
}

func TestCustomPostfixPrecedenceCrossesPower(t *testing.T) {
	tests := []struct {
		name       string
		precedence int64
		want       string
	}{
		{"above power", 95, "(a ** (b %%))"},
		{"below power", 85, "((a ** b) %%)"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			declaration := testOperatorDeclaration("%%", "postfix", test.precedence, "left", "unary")
			expr := parseOperatorExpression(t, "a ** b %%", declaration)
			if got := operatorExpressionShape(expr); got != test.want {
				t.Fatalf("shape = %s, want %s", got, test.want)
			}
		})
	}
}

func TestFixedPostfixSuffixCanFollowCustomPostfix(t *testing.T) {
	declaration := testOperatorDeclaration("%%", "postfix", 95, "left", "unary")
	for _, source := range []string{
		"value %% .field",
		"value %% ()",
		"value %% [0]",
	} {
		t.Run(source, func(t *testing.T) {
			parseOperatorExpression(t, source, declaration)
		})
	}
}

func TestEnumConstantExpressionRejectsAssignment(t *testing.T) {
	for _, expression := range []string{
		"a = b",
		"(a = b)",
		"a += b",
	} {
		t.Run(expression, func(t *testing.T) {
			findings := parseWithOperatorCatalog(enumWithConstantExpression(expression))
			if len(findings) == 0 || !strings.Contains(joinParserFindings(findings), "not allowed in a constant expression") {
				t.Fatalf("enum constant expression %q findings:\n%s", expression, joinParserFindings(findings))
			}
		})
	}
}

func TestEnumConstantExpressionAllowsLowPrecedenceCustomOperator(t *testing.T) {
	for _, precedence := range []int64{0, 5, 10} {
		t.Run(fmt.Sprintf("precedence-%d", precedence), func(t *testing.T) {
			declaration := testOperatorDeclaration("<+>", "infix", precedence, "left", "binary")
			findings := parseWithOperatorCatalog(enumWithConstantExpression("a <+> b"), declaration)
			if len(findings) != 0 {
				t.Fatalf("custom operator at precedence %d was rejected in a constant expression:\n%s", precedence, joinParserFindings(findings))
			}
		})
	}
}

func enumWithConstantExpression(expression string) string {
	return "Choice co.lang.enum = { Selected = " + expression + " }"
}

func testOperatorDeclaration(symbol, fixity string, precedence int64, associativity, arity string) operatorDeclaration {
	return operatorDeclaration{Options: map[string]any{
		"symbol": symbol, "fixity": fixity, "precedence": precedence,
		"associativity": associativity, "arity": arity,
	}}
}

func parseOperatorExpression(t *testing.T, source string, declarations ...operatorDeclaration) ast.Expr {
	t.Helper()
	collection := declaredOperatorsIn(source, "precedence.fol", declarations)
	toks := normalizeTokens(scanlex.TokenizeWith(source, "precedence.fol", collection.Custom))
	p, _ := newParser(toks)
	p.preRegisterOperatorDeclarations(collection.Declarations)
	expr := p.parseExpression()
	if !p.atEOF() {
		t.Fatalf("expression left unconsumed token %s", describeToken(p.cur()))
	}
	if len(p.diags) != 0 {
		parts := make([]string, len(p.diags))
		for index, diagnostic := range p.diags {
			parts[index] = diagnostic.Error()
		}
		t.Fatalf("expression diagnostics:\n%s", strings.Join(parts, "\n"))
	}
	return expr
}

func operatorExpressionShape(expr ast.Expr) string {
	switch expr := expr.(type) {
	case ast.SymbolExpr:
		return logicalName(expr.Value)
	case ast.BinaryExpr:
		return fmt.Sprintf("(%s %s %s)", operatorExpressionShape(expr.Left), expr.Operator.Value, operatorExpressionShape(expr.Right))
	case ast.PrefixExpr:
		if expr.Symb != nil && strings.HasPrefix(expr.Symb.GetName(), "postfix") {
			return fmt.Sprintf("(%s %s)", operatorExpressionShape(expr.Right), expr.Operator.Value)
		}
		return fmt.Sprintf("(%s %s)", expr.Operator.Value, operatorExpressionShape(expr.Right))
	case ast.GroupingExpr:
		return "(" + operatorExpressionShape(expr.Expr_) + ")"
	default:
		return fmt.Sprintf("<%T>", expr)
	}
}
