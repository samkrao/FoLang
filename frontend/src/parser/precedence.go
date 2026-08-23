package parser

import (
	"fmt"
	"sort"
	"strings"

	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// Operator precedence tables.
//
// This file encodes the built-in precedence of DECISION-OP-001 and the registry
// for user-defined operators of DECISION-EXT-001. It holds data only; pratt.go
// holds the algorithm that consumes it.
//
// Dispatch is keyed by operator LEXEME, not by token kind, because the scanner
// maps several spellings onto one kind: "^" and "**" both arrive as
// POW, and every compound assignment arrives as ASSIGNMENT. The lexeme is the
// operator's identity, so it is what the tables are indexed by. The scanner's
// whole-run pass guarantees that each operator arrives as one token.

// bindingPower is an operator's precedence. Values are taken verbatim from the
// DECISION-OP-001 table so the source can be diffed against the grammar:
//
//		 700  postfix: calls, indexing, member access, postfix !           left
//		 650  exponentiation: **                                          right
//		 600  prefix: +, -, !                                               right
//		 550  multiplicative: *, /, %                                      left
//	     500  union/intersection: ∪, ∩                                     	left
//		 450  additive: +, -                                               left
//		 400  ranges: .., <.., ..<, <..<                                   none
//		 350  relational: <, <=, >, >=, <:, :>                             none
//		 300  equality: ==, !=                                             none
//		 250  bitwise AND: &                                               left
//		 200  bitwise XOR: ^                                               left
//		 150  bitwise OR: |                                                left
//		 100  logical AND: &&                                              left
//		 50  logical OR: ||                                               left
//		 10  assignment: =, +=, -=, *=, /=, %=, **=, &=, ^=, |=           right
type bindingPower int

const (
	bpNone           bindingPower = 0
	bpAssignment     bindingPower = 10
	bpLogicalOr      bindingPower = 50
	bpLogicalAnd     bindingPower = 100
	bpBitwiseOr      bindingPower = 150
	bpBitwiseXor     bindingPower = 200
	bpBitwiseAnd     bindingPower = 250
	bpEquality       bindingPower = 300
	bpRelational     bindingPower = 350
	bpRange          bindingPower = 400
	bpAdditive       bindingPower = 450
	bpUnionInter     bindingPower = 500
	bpMultiplicative bindingPower = 550
	bpPrefix         bindingPower = 600
	bpPower          bindingPower = 650
	bpPostfix        bindingPower = 700
)

// associativity determines how operators of equal precedence group. It does not
// affect the order in which operands are evaluated: FoLang separately specifies
// left-to-right, target-first evaluation, which the semantic phase enforces.
type associativity int

const (
	// leftAssoc groups a op b op c as (a op b) op c.
	leftAssoc associativity = iota
	// rightAssoc groups a op b op c as a op (b op c). Assignment
	// (DECISION-OP-002) and exponentiation are right-associative.
	rightAssoc
	// nonAssoc forbids chaining. A range expression contains at most one range
	// operator, so `1 .. 5 .. 9` is a diagnostic rather than a nested range.
	nonAssoc
)

// operatorRole distinguishes the AST shape an infix operator produces. Several
// operators share a precedence but not a node type, so the role is carried
// alongside the binding power.
type operatorRole int

const (
	// roleArithmetic produces an ast.BinaryExpr.
	roleArithmetic operatorRole = iota
	// roleAssignment produces an ast.AssignmentExpr.
	roleAssignment
	// roleRange produces an ast.RangeExpr.
	roleRange
	// roleTypeUnion produces an ast.ADTExpr. "|" is both bitwise OR and the
	// algebraic-data-type union operator; which one is meant depends on
	// whether a type expression or a value expression is being parsed, and the
	// type parser calls into its own union handling rather than the Pratt loop.
	roleTypeUnion
	// roleCustom produces an ast.BinaryExpr from a user-registered operator.
	roleCustom
)

// infixOp describes one infix operator.
type infixOp struct {
	lexeme string
	bp     bindingPower
	assoc  associativity
	role   operatorRole
}

// builtinInfixOperators is the built-in infix table of DECISION-OP-001, keyed by
// lexeme.
//
// The four range spellings carry the same precedence and are non-associative;
// their inclusive/exclusive bounds are decoded in expr_range.go.
var builtinInfixOperators = map[string]infixOp{
	// Assignment (right-associative, lowest precedence).
	"=":   {"=", bpAssignment, rightAssoc, roleAssignment},
	"+=":  {"+=", bpAssignment, rightAssoc, roleAssignment},
	"-=":  {"-=", bpAssignment, rightAssoc, roleAssignment},
	"*=":  {"*=", bpAssignment, rightAssoc, roleAssignment},
	"/=":  {"/=", bpAssignment, rightAssoc, roleAssignment},
	"%=":  {"%=", bpAssignment, rightAssoc, roleAssignment},
	"**=": {"**=", bpAssignment, rightAssoc, roleAssignment},
	"&=":  {"&=", bpAssignment, rightAssoc, roleAssignment},
	"^=":  {"^=", bpAssignment, rightAssoc, roleAssignment},
	"|=":  {"|=", bpAssignment, rightAssoc, roleAssignment},

	// Logical.
	"||": {"||", bpLogicalOr, leftAssoc, roleArithmetic},
	"&&": {"&&", bpLogicalAnd, leftAssoc, roleArithmetic},

	// Bitwise.
	"|": {"|", bpBitwiseOr, leftAssoc, roleArithmetic},
	"^": {"^", bpBitwiseXor, leftAssoc, roleArithmetic},
	"&": {"&", bpBitwiseAnd, leftAssoc, roleArithmetic},

	// Equality and relational.
	"==": {"==", bpEquality, nonAssoc, roleArithmetic},
	"!=": {"!=", bpEquality, nonAssoc, roleArithmetic},
	"<":  {"<", bpRelational, nonAssoc, roleArithmetic},
	"<=": {"<=", bpRelational, nonAssoc, roleArithmetic},
	">":  {">", bpRelational, nonAssoc, roleArithmetic},
	">=": {">=", bpRelational, nonAssoc, roleArithmetic},
	"<:": {"<:", bpRelational, nonAssoc, roleArithmetic},
	":>": {":>", bpRelational, nonAssoc, roleArithmetic},

	// Ranges (non-associative: at most one per range-expression).
	"..":   {"..", bpRange, nonAssoc, roleRange},
	"<..":  {"<..", bpRange, nonAssoc, roleRange},
	"..<":  {"..<", bpRange, nonAssoc, roleRange},
	"<..<": {"<..<", bpRange, nonAssoc, roleRange},

	// Arithmetic.
	"+": {"+", bpAdditive, leftAssoc, roleArithmetic},
	"-": {"-", bpAdditive, leftAssoc, roleArithmetic},
	"*": {"*", bpMultiplicative, leftAssoc, roleArithmetic},
	"/": {"/", bpMultiplicative, leftAssoc, roleArithmetic},
	"%": {"%", bpMultiplicative, leftAssoc, roleArithmetic},

	// Exponentiation (right-associative, above every other infix operator).
	"**": {"**", bpPower, rightAssoc, roleArithmetic},

	// predeclared-glyph-expression: the two language-owned mathematical glyphs.
	// They are ACTIVE binary infix operators at precedence 500, left associative,
	// which is why they live in this built-in table rather than in the custom
	// registry — they are language-owned and cannot be redeclared with
	// co.lang.operator. A missing overload implementation fails during operator
	// resolution, not during parsing.
	//
	// Implements: predeclared-glyph-expression
	"∪": {"∪", bpUnionInter, leftAssoc, roleArithmetic},
	"∩": {"∩", bpUnionInter, leftAssoc, roleArithmetic},
}

// prefixOperators is the prefix-operator set of DECISION-OP-001:
//
//	prefix-operator = "+" | "-" | "!"
//
// All prefix operators bind at bpPrefix and are right-associative, so `- -x` and
// `! !ready` parse.
//
// Several spellings are shared with infix operators; position decides which table
// applies, which is the ordinary Pratt null-denotation/left-denotation split.
//
// "@" is NOT here: it introduces a directive or an annotation, and it marks the
// address derivation `co.lang.int->(@)`. A variable of an address, pointer or
// reference kind is used like any other variable — the kind lives in the type
// derivation, never at the use site — so there is no address-of prefix to take.
//
// prefix-operator is exactly these three. There is no reserved-prefix-operator
// alternative: "#" is a reserved-operator in its own right, and "~" and "^" simply have
// no prefix meaning — each keeps the roles it already plays elsewhere ("~" marks a named
// parameter and the `->(~)` heap reference, "^" is bitwise xor and the `->(^)` thunk
// derivation), so neither is a prefix spelling being held back.
var prefixOperators = map[string]struct{}{
	"+": {},
	"-": {},
	"!": {},
}

// postfixOperators is the postfix set. "!" is the unwrap/assert postfix.
// Increment/decrement spellings are deliberately absent:
// alpha FoLang uses += 1 and -= 1, and an unregistered ++/-- remains one unknown
// symbolic run rather than falling back to two unary operators.
var postfixOperators = map[string]struct{}{
	"!": {},
}

// reservedOperators is the reserved-operator production: the spellings the scanner
// recognises as single tokens and the parser must refuse. Rejecting them explicitly
// stops a user-defined operator from claiming a spelling before the language assigns it
// a meaning, and gives a better diagnostic than "unexpected token".
//
// The list is CLOSED and matches the grammar exactly. "~" and "^" are deliberately
// absent: neither is reserved, both already carry meanings elsewhere in the language,
// and neither has a prefix reading to hold back. "#" is here because the grammar names
// it, alongside the three multi-symbol spellings and the two single characters.
//
// Implements: reserved-operator
var reservedOperators = map[string]string{
	"::=": "reserved for a future definition operator",
	"->>": "reserved for a future pipeline operator",
	"<->": "reserved for a future bidirectional operator",
	"#":   "reserved",
	"`":   "reserved",
	"\\":  "reserved",
}

// operatorTable is the registry of user-defined operators (DECISION-EXT-001).
//
// A new symbol must declare fixity, numeric precedence and associativity before
// it can be used, and the Pratt engine then treats it exactly like a built-in.
// An overload of a built-in symbol does not enter this table: it keeps the
// built-in precedence, as the decision requires.
type operatorTable struct {
	infix      map[string]infixOp
	prefix     map[string]bindingPower
	postfix    map[string]bindingPower
	syntax     map[string]operatorSyntax
	conflicted map[string]struct{}
}

// operatorSyntax is the project-wide parse contract for one custom spelling.
// A spelling cannot mean prefix in one file and infix in another, or acquire a
// different binding power according to which file happened to be visited first.
// Mode and implementation signatures are semantic overload metadata; the four
// fields here are precisely the metadata that changes token consumption or the
// Pratt tree.
type operatorSyntax struct {
	fixity        string
	precedence    int
	associativity string
	arity         string
}

// String renders syntax metadata in a stable order for diagnostics and tests.
func (s operatorSyntax) String() string {
	return fmt.Sprintf(
		"fixity=%s, precedence=%d, associativity=%s, arity=%s",
		s.fixity,
		s.precedence,
		s.associativity,
		s.arity,
	)
}

// newOperatorTable creates an empty user-defined operator registry.
func newOperatorTable() *operatorTable {
	return &operatorTable{
		infix:      map[string]infixOp{},
		prefix:     map[string]bindingPower{},
		postfix:    map[string]bindingPower{},
		syntax:     map[string]operatorSyntax{},
		conflicted: map[string]struct{}{},
	}
}

// registerInfix adds a user-defined infix operator with the declared precedence
// and associativity. Registering a built-in spelling is ignored, because an
// overload of a built-in symbol retains built-in precedence.
func (t *operatorTable) registerInfix(lexeme string, prec int, assoc associativity) {
	if _, isBuiltin := builtinInfixOperators[lexeme]; isBuiltin {
		return
	}
	t.infix[lexeme] = infixOp{
		lexeme: lexeme,
		bp:     bindingPower(prec),
		assoc:  assoc,
		role:   roleCustom,
	}
}

// registerPrefix adds a user-defined prefix operator.
func (t *operatorTable) registerPrefix(lexeme string, prec int) {
	t.prefix[lexeme] = bindingPower(prec)
}

// registerPostfix adds a user-defined postfix operator.
func (t *operatorTable) registerPostfix(lexeme string, prec int) {
	t.postfix[lexeme] = bindingPower(prec)
}

// registerOperatorDeclaration validates an ordinary implementation annotation.
// Registration belongs exclusively to the configured operator source; an
// @co.dap.operator function may only overload an already language-owned or
// project-registered symbol and cannot alter its parse properties.
func (p *parser) registerOperatorDeclaration(options map[string]any, context string) {
	symbol := operatorOptionText(options, "symbol")
	if symbol == "" {
		p.reportf(p.cur(), "%s requires a symbol option", context)
		return
	}

	mode := operatorOptionText(options, "mode")
	if mode != "" && mode != "overload" {
		p.reportf(p.cur(), "%s implements operator %q and accepts only mode=overload (or an omitted mode), found mode=%s", context, symbol, mode)
		return
	}
	if !scanlex.IsOperatorSpelling(symbol) {
		p.reportf(p.cur(), "%s names %q, which is not a valid operator spelling", context, symbol)
		return
	}
	if scanlex.IsHardReservedOperatorSpelling(symbol) {
		p.reportf(p.cur(), "%s cannot implement %q because it is hard-reserved", context, symbol)
		return
	}
	for _, key := range []string{"fixity", "precedence", "associativity", "arity"} {
		if _, present := options[key]; present {
			p.reportf(p.cur(), "%s cannot specify %s; parse properties belong only in the components/operators/component.fol declaration for %q", context, key, symbol)
			return
		}
	}

	// A pre-declared glyph reaches this check as an ordinary built-in overload.
	// `∪` and `∩` are ENABLED operators whose parse properties the language fixes,
	// and the reference states that their concrete behavior is supplied through
	// ordinary mode=overload implementations under the same ownership and
	// resolution rules as any other language-owned operator. What they cannot do
	// is be REDECLARED with co.lang.operator, which the operator-component reader
	// rejects at the declaration site rather than here at the implementation site.
	if isBuiltinOperatorSymbol(symbol) {
		return
	}
	if scanlex.IsLanguageOwnedOperatorSpelling(symbol) {
		p.reportf(p.cur(), "%s cannot implement language-owned structural spelling %q; its presence in the symbolic inventory does not make it overloadable", context, symbol)
		return
	}
	if _, registered := p.ops.syntax[symbol]; !registered {
		p.reportf(p.cur(), "%s implements unregistered custom operator %q; declare its parse properties once in components/operators/component.fol", context, symbol)
	}
}

// preRegisterOperatorDeclarations installs the validated declarations from the
// one components/operators/component.fol surface before any ordinary body is parsed.
// Grouping is retained defensively for programmatic callers; the dedicated source
// parser itself rejects every duplicate symbol.
//
// This is where the project-local custom registrations are combined with the
// language-owned built-in ones into the tables the Pratt engine then treats as
// immutable for the whole compilation. Imports contribute nothing here, so the
// bootstrap has no import-order or transitive-dependency dependency.
func (p *parser) preRegisterOperatorDeclarations(declarations []operatorDeclaration) {
	bySymbol := map[string]map[operatorSyntax]struct{}{}
	for _, declaration := range declarations {
		symbol, syntax, ok := operatorSyntaxOf(declaration.Options)
		if !ok {
			continue
		}
		if bySymbol[symbol] == nil {
			bySymbol[symbol] = map[operatorSyntax]struct{}{}
		}
		bySymbol[symbol][syntax] = struct{}{}
	}

	symbols := make([]string, 0, len(bySymbol))
	for symbol := range bySymbol {
		symbols = append(symbols, symbol)
	}
	sort.Strings(symbols)

	for _, symbol := range symbols {
		syntaxes := make([]operatorSyntax, 0, len(bySymbol[symbol]))
		for syntax := range bySymbol[symbol] {
			syntaxes = append(syntaxes, syntax)
		}
		sort.Slice(syntaxes, func(i, j int) bool {
			return syntaxes[i].String() < syntaxes[j].String()
		})

		if len(syntaxes) > 1 {
			p.ops.conflicted[symbol] = struct{}{}
			p.ops.remove(symbol)
			formatted := make([]string, len(syntaxes))
			for i, syntax := range syntaxes {
				formatted[i] = "{" + syntax.String() + "}"
			}
			p.reportf(
				p.cur(),
				"custom operator %q has conflicting project syntax declarations: %s; one symbol must have one fixity, precedence, associativity, and arity",
				symbol,
				strings.Join(formatted, " versus "),
			)
			continue
		}

		if assoc, ok := registrableOperatorSyntax(syntaxes[0]); ok {
			p.installOperatorSyntax(symbol, syntaxes[0], assoc, "an operator declaration", false)
		}
	}
}

// operatorSyntaxOf extracts a complete custom-operator syntax declaration. A
// malformed declaration remains the real parser's responsibility; built-in
// overloads are excluded because their syntax is fixed by the language table.
func operatorSyntaxOf(options map[string]any) (string, operatorSyntax, bool) {
	symbol := operatorOptionText(options, "symbol")
	if symbol == "" || scanlex.IsLanguageOwnedOperatorSpelling(symbol) || !scanlex.IsOperatorSpelling(symbol) {
		return "", operatorSyntax{}, false
	}
	if _, reserved := reservedOperators[symbol]; reserved {
		return "", operatorSyntax{}, false
	}
	precedence, ok := operatorOptionInteger(options, "precedence")
	if !ok {
		return "", operatorSyntax{}, false
	}
	for _, key := range []string{"fixity", "associativity", "arity"} {
		if _, present := options[key]; !present {
			return "", operatorSyntax{}, false
		}
	}
	arity, arityOK := operatorArityCount(options["arity"])
	if !arityOK {
		return "", operatorSyntax{}, false
	}
	return symbol, operatorSyntax{
		fixity:        operatorOptionText(options, "fixity"),
		precedence:    precedence,
		associativity: operatorOptionText(options, "associativity"),
		arity:         fmt.Sprint(arity),
	}, true
}

// registrableOperatorSyntax converts validated textual associativity and checks
// the fixity/arity pair needed by the current Pratt implementation.
func registrableOperatorSyntax(syntax operatorSyntax) (associativity, bool) {
	assoc := leftAssoc
	switch syntax.associativity {
	case "left":
	case "right":
		assoc = rightAssoc
	case "none":
		assoc = nonAssoc
	default:
		return leftAssoc, false
	}

	switch syntax.fixity {
	case "prefix", "postfix":
		return assoc, syntax.arity == "1"
	case "infix":
		return assoc, syntax.arity == "2"
	default:
		return leftAssoc, false
	}
}

// installOperatorSyntax makes one custom spelling active after checking the
// invariant recorded by an earlier project prepass or declaration. When called
// without a prepass, a second conflicting declaration still removes the first
// registration, so direct parser users get the same deterministic behavior.
func (p *parser) installOperatorSyntax(symbol string, syntax operatorSyntax, assoc associativity, context string, diagnose bool) {
	if _, conflicted := p.ops.conflicted[symbol]; conflicted {
		return
	}
	if previous, exists := p.ops.syntax[symbol]; exists {
		if previous == syntax {
			return
		}
		p.ops.remove(symbol)
		p.ops.conflicted[symbol] = struct{}{}
		if diagnose {
			p.reportf(
				p.cur(),
				"%s conflicts with another declaration of custom operator %q: {%s} versus {%s}",
				context,
				symbol,
				previous.String(),
				syntax.String(),
			)
		}
		return
	}

	p.ops.syntax[symbol] = syntax
	switch syntax.fixity {
	case "prefix":
		p.ops.registerPrefix(symbol, syntax.precedence)
	case "postfix":
		p.ops.registerPostfix(symbol, syntax.precedence)
	case "infix":
		p.ops.registerInfix(symbol, syntax.precedence, assoc)
	}
}

// remove clears every fixity table entry for a spelling. It is used when a
// conflict is discovered after an earlier declaration was registered.
func (t *operatorTable) remove(symbol string) {
	delete(t.infix, symbol)
	delete(t.prefix, symbol)
	delete(t.postfix, symbol)
	delete(t.syntax, symbol)
}

// isBuiltinOperatorSymbol reports whether symbol already belongs to one of the
// normative built-in fixity tables. Such a declaration is an overload and must
// keep the grammar-defined precedence.
func isBuiltinOperatorSymbol(symbol string) bool {
	if _, ok := builtinInfixOperators[symbol]; ok {
		return true
	}
	if _, ok := prefixOperators[symbol]; ok {
		return true
	}
	_, ok := postfixOperators[symbol]
	return ok
}

// operatorOptionText normalizes string-like metadata to its source spelling.
func operatorOptionText(options map[string]any, key string) string {
	value, ok := options[key]
	if !ok {
		return ""
	}
	switch value := value.(type) {
	case string:
		return value
	case fmt.Stringer:
		return value.String()
	default:
		return fmt.Sprint(value)
	}
}

// operatorOptionInteger extracts a numeric precedence without accepting a
// floating-point value or an arbitrary textual fallback.
func operatorOptionInteger(options map[string]any, key string) (int, bool) {
	value, ok := options[key]
	if !ok {
		return 0, false
	}
	switch value := value.(type) {
	case int:
		return value, true
	case int64:
		return int(value), true
	default:
		return 0, false
	}
}

// infixOperator returns the infix descriptor for the token at the cursor,
// consulting the built-in table first and then the user-defined registry.
func (p *parser) infixOperator() (infixOp, bool) {
	lex := p.lexeme()
	if op, ok := builtinInfixOperators[lex]; ok {
		return op, true
	}
	if op, ok := p.ops.infix[lex]; ok {
		return op, true
	}
	return infixOp{}, false
}

// isPrefixOperator reports whether the token at the cursor may begin a unary
// expression.
func (p *parser) isPrefixOperator() bool {
	lex := p.lexeme()
	if _, ok := prefixOperators[lex]; ok {
		return true
	}
	_, ok := p.ops.prefix[lex]
	return ok
}

// isPostfixOperator reports whether the token at the cursor may follow an
// operand as a postfix operator.
func (p *parser) isPostfixOperator() bool {
	lex := p.lexeme()
	if _, ok := postfixOperators[lex]; ok {
		return true
	}
	_, ok := p.ops.postfix[lex]
	return ok
}

// isBuiltinPostfixOperator reports whether the fixed grammar postfix layer owns
// the current token. Custom postfix operators are handled by parseExpr so their
// declared precedence is respected.
func (p *parser) isBuiltinPostfixOperator() bool {
	_, ok := postfixOperators[p.lexeme()]
	return ok
}

// nextMinBindingPower returns the minimum binding power for the right operand of
// op. A left-associative operator excludes its own precedence so that an equal
// operator terminates the recursive call and is picked up by the enclosing loop;
// a right-associative one includes it so that the recursive call keeps going.
func nextMinBindingPower(op infixOp) bindingPower {
	if op.assoc == rightAssoc {
		return op.bp
	}
	return op.bp + 1
}
