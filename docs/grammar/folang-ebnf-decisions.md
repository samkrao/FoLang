# FoLang EBNF decision register — revision 2

This register lists the rules added where `language-ref(32).md` did not yet make a complete lexical or parsing decision. These are proposed normative rules, not hidden assumptions. The matching `DECISION-*` identifiers also appear as comments in `folang-complete-v2.ebnf`.

## 1. Statement termination

### DECISION-SYN-001 — mandatory semicolons

- Every **simple statement** ends with `;`.
- Every standalone built-in directive such as `@co.ddap.import(...)`, `@co.ddap.alias(...)`, and `@co.ddap.use(...)` ends with `;`.
- Newlines are whitespace only. FoLang performs no automatic semicolon insertion.
- A block statement and a block-bodied declaration do not require a second semicolon after their closing `}`.
- The complete condition/loop/iterator chain is an expression statement and therefore ends with `;`.

Examples:

```folang
value := calculate();
value = other = source;

condition.do({
    run();
}).otherwise.do({
    recover();
});

Employee co.lang.struct = {
    id co.lang.int;
    name co.lang.string;
}
```

The struct declaration itself has no semicolon after `}`; its field declarations do.

### DECISION-BLK-001 — block tail expressions

A final expression without `;` is a block's value-producing tail expression. It is not a statement and therefore does not violate mandatory statement termination.

```folang
classify(n) => {
    n + 1
}
```

Writing `n + 1;` makes it an expression statement and the block has no tail value.

## 2. Assignment and operators

### DECISION-OP-001 — built-in precedence

| Precedence | Operators/forms | Associativity |
|---:|---|---|
| 100 | call `()`, index `[]`, member `.`, postfix `!`, `++`, `--` | left |
| 90 | `**` | right |
| 80 | prefix `+`, `-`, `!`, `~`, `@`, `#`, `^`, `++`, `--` | right |
| 70 | `*`, `/`, `%` | left |
| 60 | `+`, `-` | left |
| 55 | `..`, `<..`, `..<`, `<..<` | non-associative |
| 50 | `<`, `<=`, `>`, `>=` | left |
| 45 | `==`, `!=` | left |
| 40 | `&` | left |
| 38 | `^` | left |
| 36 | `|` | left |
| 30 | `&&` | left, short-circuit |
| 20 | `||` | left, short-circuit |
| 10 | assignment operators | right |

Precedence determines grouping. It does not replace FoLang's normative left-to-right operand evaluation or target-first assignment evaluation.

Exponentiation binds more tightly than prefix operators, so `-2 ** 2` means `-(2 ** 2)`. Both prefix and postfix forms of `++` and `--` are accepted. Prefix form mutates and yields the new value; postfix form yields the previous value and then mutates.

### DECISION-OP-002 — right-associative assignment

```folang
a = b = c;
```

is parsed as:

```folang
a = (b = c);
```

An assignment expression yields the value assigned. The assignment target is still resolved/evaluated before its right-hand side, as required by the existing evaluation-order section.

Runtime assignment operators are:

```text
=  +=  -=  *=  /=  %=  **=  &=  ^=  |=
```

### DECISION-OP-004 — increment and decrement fixity

`++` and `--` are accepted in both prefix and postfix positions, following mainstream C/Java behavior. Prefix form mutates the target and yields the new value. Postfix form yields the previous value and then mutates the target. Their operand must be a mutable assignable target.

### DECISION-OP-003 — definition operators are statement-only

- `:=` declares and initializes an inferred variable and errors when the name already exists.
- `?=` declares and initializes when absent; otherwise it reassigns the existing binding.
- Neither operator is a general expression operator, returns a chainable assignment value, or may appear in `a = b := c`.
- `::=` remains reserved until its semantics are defined and is rejected by this grammar.

### DECISION-EXT-001 — custom operators

- Overloading a built-in symbol retains that symbol's built-in fixity, precedence, and associativity.
- Defining a new symbol requires explicit `fixity`, numeric `precedence`, `associativity`, and `arity` metadata.
- Higher numeric precedence binds more tightly.
- Associativity is `left`, `right`, or `none`.
- The compiler collects operator declarations before parsing operator expressions, so use is not source-order dependent.
- Maximal munch selects the longest valid operator token.
- A custom operator may not redefine structural delimiters, comment delimiters, arrows, assignment/definition operators, or other reserved grammar tokens.

Because arbitrary registered operators make the expression grammar depend on declarations, their exact expansion is context-sensitive. The EBNF includes a named contextual production for this compiler-generated precedence grammar rather than pretending that a fixed context-free production can enumerate all future symbols.

## 3. Lexical contract

### DECISION-LEX-001 — source encoding and identifiers

- Source files are UTF-8.
- An optional U+FEFF BOM is allowed only at the start of the file.
- Invalid UTF-8 is a lexical error.
- Identifiers follow Unicode `XID_Start` and `XID_Continue`, with `_` also allowed.
- Identifiers are normalized to NFC before comparison and symbol-table insertion.
- A lone `_` is the wildcard/discard token, not an identifier.
- Keywords are case-sensitive ASCII spellings.
- `self` remains contextual; the other listed reserved words are hard-reserved.

### DECISION-LEX-002 — whitespace and comments

- Spaces, tabs, form feeds, and line terminators separate tokens.
- `//` introduces a line comment.
- `/* ... */` introduces a non-nesting block comment.
- Documentation comments are lexically comments until a separate documentation model is specified.
- Newlines never terminate statements.

### DECISION-LEX-003 — token selection

The lexer uses maximal munch. Reserved multi-character tokens are chosen before shorter prefixes. Examples:

```text
<..<  before  <.. or <
**=   before  ** or *
=>>   before  => or =
..<   before  .. or .
```

Comment openers are recognized before `/` is treated as an arithmetic operator.

## 4. Literals

### DECISION-LIT-001 — integer literals

Supported:

```folang
0
42
1_000_000
0b1010_0110
0o755
0xCAFE_BABE
```

Underscores may occur only between digits. Numeric suffixes are excluded initially; explicit types, casts, or contextual typing select the required numeric type.

### DECISION-LIT-002 — floating literals

Supported decimal forms include:

```folang
1.0
3.141_592
1.0e10
6.02e-23
10e3
```

A decimal point must be followed by digits. Therefore `1..10` is unambiguously an integer, a range operator, and another integer. Hexadecimal floating literals are excluded initially.

### DECISION-LIT-003 — strings and characters

The initial grammar supports single-line quoted strings and one-scalar character literals. Escapes are:

```text
\0 \b \t \n \f \r \" \' \\ \u{HEX}
```

The Unicode escape accepts one to six hexadecimal digits and must denote a Unicode scalar value. Raw strings, multiline strings, and interpolation are excluded until separately specified.

## 5. Collection and metadata punctuation

### DECISION-COL-001 — canonical separators

- Enum variants use commas, with an optional trailing comma.
- Array and tuple elements use commas.
- Map entries use `key: value` and commas.
- Object constructors use `field: value` and commas.
- Annotation maps use `key: value` and commas.
- Named annotation arguments continue to use `name=value`.
- `=` inside an object initializer or annotation map is rejected rather than treated as an alternative spelling.

Example:

```folang
employee := Employee{
    id: 1001,
    name: "Rao",
};
```

## 6. Ambiguity-resolution rules

1. `{ ... }` is parsed according to its syntactic position: declaration/block body, map literal, annotation map, or object-construction body.
2. `|x| => expression` starts a lambda only where a primary expression may begin; infix `|` is otherwise bitwise OR.
3. A qualified name followed by `{` is object construction only when the name resolves in type position; otherwise the construct is rejected rather than reinterpreted as a block.
4. Type argument/derivation syntax is parsed in type positions; call syntax is parsed in expression positions.
5. Range operators are non-associative. `a..b..c` is a syntax error unless parentheses create nested range expressions explicitly.
6. Collection/map/object trailing commas are accepted; trailing semicolons are not substitutes for commas.
7. Planned constructs, including comprehensions and package aliasing, remain feature-gated even when represented in the grammar.

## 7. Existing contextual rules retained

The generated grammar does not weaken the reference's contextual rules:

- ordinary package files contain exactly one primary top-level declaration;
- entry files permit their restricted entry-local declarations and executable statements;
- library surfaces permit only boundary declarations and adapter functions;
- units contain functions and companion ownership is checked semantically;
- no physical local/nested named declarations are introduced by the grammar;
- `forall(T).type` remains type-position syntax, while named generics use annotations;
- capability, visibility, realm, capture, and definite-initialization rules remain semantic checks.

## 8. Reference-document edits implied by these decisions

Before declaring the grammar normative, update `language-ref.md` to:

1. add the lexical contract and literal sections above;
2. add the built-in precedence table;
3. state assignment's right associativity and value result;
4. state the mandatory-semicolon/no-ASI rule;
5. normalize examples by adding missing semicolons to simple statements and standalone directives;
6. replace enum semicolon separators with commas;
7. replace `=` with `:` inside object initializers and annotation maps;
8. mark `::=`, raw/multiline/interpolated strings, hex floats, and distinct set-literal syntax as unsupported or reserved;
9. document custom-operator collection and precedence integration.
