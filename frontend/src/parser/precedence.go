package parser

// Operator precedence tables.
//
// This file encodes the built-in precedence of DECISION-OP-001 and the registry
// for user-defined operators of DECISION-EXT-001. It holds data only; pratt.go
// holds the algorithm that consumes it.
//
// Dispatch is keyed by operator LEXEME, not by token kind, because the scanner
// maps several spellings onto one kind: "^" and the fused "**" both arrive as
// POW, and every compound assignment arrives as ASSIGNMENT. The lexeme is the
// operator's identity, so it is what the tables are indexed by. See
// tokenstream.go for the fusion pass that guarantees each operator is one token.

// bindingPower is an operator's precedence. Values are taken verbatim from the
// DECISION-OP-001 table so the source can be diffed against the grammar:
//
//	100  postfix: calls, indexing, member access, postfix !, ++, --   left
//	 90  exponentiation: **                                          right
//	 80  prefix: +, -, !, ~, @, #, ^, ++, --                          right
//	 70  multiplicative: *, /, %                                      left
//	 60  additive: +, -                                               left
//	 55  ranges: .., <.., ..<, <..<                                   none
//	 50  relational: <, <=, >, >=                                     left
//	 45  equality: ==, !=                                             left
//	 40  bitwise AND: &                                               left
//	 38  bitwise XOR: ^                                               left
//	 36  bitwise OR: |                                                left
//	 30  logical AND: &&                                              left
//	 20  logical OR: ||                                               left
//	 10  assignment: =, +=, -=, *=, /=, %=, **=, &=, ^=, |=           right
type bindingPower int

const (
	bpNone           bindingPower = 0
	bpAssignment     bindingPower = 10
	bpLogicalOr      bindingPower = 20
	bpLogicalAnd     bindingPower = 30
	bpBitwiseOr      bindingPower = 36
	bpBitwiseXor     bindingPower = 38
	bpBitwiseAnd     bindingPower = 40
	bpEquality       bindingPower = 45
	bpRelational     bindingPower = 50
	bpRange          bindingPower = 55
	bpAdditive       bindingPower = 60
	bpMultiplicative bindingPower = 70
	bpPrefix         bindingPower = 80
	bpPower          bindingPower = 90
	bpPostfix        bindingPower = 100
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
	// operator, so `1..5..9` is a diagnostic rather than a nested range.
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
	"==": {"==", bpEquality, leftAssoc, roleArithmetic},
	"!=": {"!=", bpEquality, leftAssoc, roleArithmetic},
	"<":  {"<", bpRelational, leftAssoc, roleArithmetic},
	"<=": {"<=", bpRelational, leftAssoc, roleArithmetic},
	">":  {">", bpRelational, leftAssoc, roleArithmetic},
	">=": {">=", bpRelational, leftAssoc, roleArithmetic},

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
}

// prefixOperators is the prefix set of DECISION-OP-001. All prefix operators bind
// at bpPrefix and are right-associative, so `- - x` and `!!ready` parse.
//
// Several spellings are shared with infix operators; position decides which
// table applies, which is the ordinary Pratt null-denotation/left-denotation
// split. "@" takes an address, "^" forces a thunk, and "#" is the length/count
// prefix.
var prefixOperators = map[string]struct{}{
	"+":  {},
	"-":  {},
	"!":  {},
	"~":  {},
	"@":  {},
	"#":  {},
	"^":  {},
	"++": {},
	"--": {},
}

// postfixOperators is the postfix set of DECISION-OP-004. "!" is the
// unwrap/assert postfix; "++" and "--" exist in both prefix and postfix form.
var postfixOperators = map[string]struct{}{
	"!":  {},
	"++": {},
	"--": {},
}

// reservedOperators are the spellings the scanner recognises as single tokens and
// the parser must refuse (DECISION-OP-005). Rejecting them explicitly stops a
// user-defined operator from claiming a spelling before the language assigns it
// a meaning, and gives a better diagnostic than "unexpected token".
var reservedOperators = map[string]string{
	"::=": "reserved for a future definition operator",
	"->>": "reserved for a future pipeline operator",
	"<->": "reserved for a future bidirectional operator",
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
	infix   map[string]infixOp
	prefix  map[string]struct{}
	postfix map[string]struct{}
}

// newOperatorTable creates an empty user-defined operator registry.
func newOperatorTable() *operatorTable {
	return &operatorTable{
		infix:   map[string]infixOp{},
		prefix:  map[string]struct{}{},
		postfix: map[string]struct{}{},
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
func (t *operatorTable) registerPrefix(lexeme string) { t.prefix[lexeme] = struct{}{} }

// registerPostfix adds a user-defined postfix operator.
func (t *operatorTable) registerPostfix(lexeme string) { t.postfix[lexeme] = struct{}{} }

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
