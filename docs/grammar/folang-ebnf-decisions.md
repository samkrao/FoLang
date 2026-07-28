# FoLang EBNF decision register — revision 4

This register lists explicit lexical, parsing, and C++-lowering decisions used by `folang-complete-v4.ebnf`. These are proposed normative rules for incorporation into `language-ref.md`, not hidden parser assumptions.

## 1. Statement termination

### DECISION-SYN-001 — mandatory semicolons

- Every **simple statement** ends with `;`.
- Every standalone built-in directive such as `@co.ddap.import(...)`, `@co.ddap.alias(...)`, and `@co.ddap.use(...)` ends with `;`.
- Newlines are whitespace only. FoLang performs no automatic semicolon insertion.
- A block statement and a block-bodied declaration do not require another semicolon after their closing `}`.
- A complete condition, loop, or iterator chain used as an expression statement ends with `;`.

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

A final expression without `;` is a block's value-producing tail expression. It is not a statement.

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

Exponentiation binds more tightly than prefix operators, so `-2 ** 2` means `-(2 ** 2)`. Both prefix and postfix forms of `++` and `--` are accepted.

### DECISION-OP-002 — right-associative assignment

```folang
a = b = c;
```

is parsed as:

```folang
a = (b = c);
```

An assignment expression yields the value assigned. Runtime assignment operators are:

```text
=  +=  -=  *=  /=  %=  **=  &=  ^=  |=
```

### DECISION-OP-004 — increment and decrement fixity

`++` and `--` are accepted in prefix and postfix positions. Prefix form mutates the target and yields the new value. Postfix form yields the previous value and then mutates the target. Their operand must be mutable and assignable.

### DECISION-OP-003 — definition operators are statement-only

- `:=` declares and initializes an inferred variable and errors when the name already exists.
- `?=` declares and initializes when absent; otherwise it reassigns the existing binding.
- Neither operator is a general expression operator or may be chained.
- `::=` remains reserved until its semantics are defined.

### DECISION-EXT-001 — custom operators

- Overloading a built-in symbol retains that symbol's built-in fixity, precedence, and associativity.
- A new symbol requires explicit `fixity`, numeric `precedence`, `associativity`, and `arity` metadata.
- Higher numeric precedence binds more tightly.
- Associativity is `left`, `right`, or `none`.
- The compiler collects operator declarations before parsing operator expressions.
- Maximal munch selects the longest valid operator token.
- A custom operator may not redefine structural delimiters, comment delimiters, arrows, assignment/definition operators, or other reserved grammar tokens.

## 3. Lexical contract and C++ identifier lowering

### DECISION-LEX-001 — source encoding and FoLang identifiers

- Source files are UTF-8.
- An optional U+FEFF BOM is allowed only at the beginning of the file.
- Invalid UTF-8 is a lexical error.
- A FoLang identifier begins with `A-Z` or `a-z`.
- Remaining characters may be `A-Z`, `a-z`, `0-9`, or `_`.
- Two consecutive underscores are prohibited.
- A trailing underscore is prohibited. This prevents appending `_fo` from creating a C++-reserved `__` sequence.
- A lone `_` is a contextual wildcard, discard, or filename-derived-name token and is not an identifier.
- Identifier comparison is byte-for-byte and case-sensitive; Unicode normalization is irrelevant because non-ASCII identifier characters are not accepted.
- FoLang reserved words are emitted as their reserved token kinds, not as identifiers.

Equivalent regular-language description:

```text
[A-Za-z][A-Za-z0-9]*(?:_[A-Za-z0-9]+)*
```

Valid:

```folang
employee
Employee1
employee_name
x_1_y2
```

Invalid:

```folang
_employee       // leading underscore
employee__name // consecutive underscores
employee_      // trailing underscore
1employee      // leading digit
_              // dedicated contextual token
नाम            // non-ASCII identifier characters
```

### DECISION-BACKEND-001 — `_fo` C++ suffix

Each **resolved user-defined FoLang identifier** is lowered to the C++ IR by appending `_fo`:

```text
employee       -> employee_fo
class          -> class_fo
employee_name  -> employee_name_fo
```

This makes C++ keywords harmless after lowering. Built-in FoLang names such as `co.*`, reserved words, externally linked names, and compiler-generated internal names follow their separately defined lowering or linkage rules.

The symbol table and AST retain the original FoLang spelling. The C++ name is a backend lowering result; source-level name resolution never uses the suffixed spelling.

### DECISION-LEX-002 — whitespace and comments

- Spaces, tabs, form feeds, and line terminators separate tokens.
- `//` introduces a line comment.
- `/* ... */` introduces a non-nesting block comment.
- Documentation comments remain ordinary comments until a separate documentation model is defined.
- Newlines never terminate statements.

### DECISION-LEX-003 — token selection

The lexer uses maximal munch. Reserved multi-character tokens are chosen before shorter prefixes.

```text
<..<  before  <.. or <
**=   before  ** or *
=>>   before  => or =
..<   before  .. or .
```

Comment openers are recognized before `/` is treated as an arithmetic operator.

## 4. C++-compatible built-in literals

### DECISION-LIT-000 — pass-through literal policy

FoLang accepts built-in literal spellings supported by the configured C++ backend dialect. For every literal token, the frontend stores:

```text
literal category
complete raw source lexeme
```

The C++ backend emits the raw lexeme unchanged. Digit separators, prefixes, suffixes, escape spelling, encoding prefixes, and raw-string delimiters are not reconstructed.

The grammar mirrors the current C++ built-in numeric, character, and string literal forms used by this revision. A conditionally supported C++ suffix is accepted only when the configured backend compiler supports it.

This policy reuses C++ **lexical spelling and literal value interpretation**, but the resulting value participates in FoLang's type and object model. It does not automatically import C++ array, pointer, storage-duration, mutability, or identity semantics.

The following are not introduced by this decision:

- the C++ pointer literal `nullptr`;
- C++ user-defined-literal operator lookup such as `operator""_km`;
- arbitrary C++ tokens that are not FoLang literals.

FoLang-specific user-defined literals are governed separately by
`DECISION-LIT-004`.

### DECISION-LIT-001 — integer literals

Supported C++-compatible forms include:

```folang
0
42
1'000'000
0b1010'0110
0755
0xCAFE'BABE
42u
42L
42ULL
42uz
```

Rules:

- binary prefix: `0b` or `0B`;
- octal: leading `0`, not `0o`;
- hexadecimal prefix: `0x` or `0X`;
- digit separator: apostrophe `'`, only between digits;
- suffixes: `u/U`, `l/L`, `ll/LL`, `z/Z`, in C++-permitted combinations.

Underscore is not a numeric separator.

### DECISION-LIT-002 — floating literals

Supported C++-compatible forms include:

```folang
1.0
1.
.5
1e10
1.602'176'565e-19
1.0f
1.0L
0xC.68p+2
0x1p-4
```

Rules:

- decimal floating literals use an optional decimal exponent `e/E`;
- hexadecimal floating literals require a binary exponent `p/P`;
- apostrophes may separate digits;
- `f/F` and `l/L` suffixes are supported;
- `f16/F16`, `f32/F32`, `f64/F64`, `f128/F128`, and `bf16/BF16` are accepted only when supported by the configured C++ backend.

The range expression `1..10` remains unambiguous under maximal munch: it is integer `1`, range operator `..`, integer `10`, rather than floating literal `1.` followed by `.10`.

### DECISION-LIT-003 — character and string literals

FoLang accepts the configured C++ backend's built-in character and string literal spellings, including:

```folang
'a'
'\n'
u8'x'
u'Ω'
U'Ω'
L'x'

"text"
u8"text"
u"text"
U"text"
L"text"
R"(raw text)"
u8R"tag(raw UTF-8 text)tag"
```

Supported syntax includes:

- encoding prefixes `u8`, `u`, `U`, and `L`;
- C++ simple escapes;
- octal and hexadecimal numeric escapes;
- universal character names `\\uXXXX`, `\\UXXXXXXXX`, `\\u{...}`, and `\\N{...}`;
- C++ raw strings with matching delimiters of at most 16 valid delimiter characters;
- adjacent string-literal concatenation.

C++ multicharacter literals are conditionally supported only when accepted by the configured backend. Prefixed character literals must satisfy the configured backend's C++ restrictions.


### DECISION-LIT-004 — FoLang user-defined literals

C++ user-defined-literal operator syntax and C++ `operator""` lookup are not
part of FoLang's source semantics.

FoLang user-defined literals use FoLang-defined syntax and semantics. The
frontend resolves each such literal to a target FoLang type and retains:

```text
literal category
complete original source spelling
target FoLang type
structured literal components
```

The backend may lower a FoLang user-defined literal to any semantically
equivalent C++ representation, including:

- a native C++ built-in literal when the FoLang value has a direct equivalent;
- aggregate initialization for a generated struct;
- constructor invocation for a generated class or scalar wrapper;
- a static factory invocation;
- a generated helper function;
- an ADT or variant-case constructor;
- a `constexpr` or `consteval` construction helper.

For a struct-like FoLang literal, lowering may produce:

```cpp
Point_fo{
    .x_fo = 10,
    .y_fo = 20
}
```

For a class whose invariants must be enforced, lowering may instead produce:

```cpp
Point_fo::create_fo(10, 20)
```

The selected C++ representation is a backend implementation detail. It does
not define FoLang source syntax, overload resolution, type checking, or literal
semantics.

The EBNF currently represents this category with a contextual production until
the complete FoLang declaration and invocation syntax for user-defined literals
is finalized. A parser must not approximate this category as a C++ token suffix
such as `12_km` unless FoLang independently defines that exact form.

## 5. Collection and metadata punctuation

### DECISION-COL-001 — canonical separators

- Enum variants use commas, with an optional trailing comma.
- Array and tuple elements use commas.
- Map entries use `key: value` and commas.
- Object constructors use `field: value` and commas.
- Annotation maps use `key: value` and commas.
- Named annotation arguments continue to use `name=value`.
- `=` inside an object initializer or annotation map is rejected.

```folang
employee := Employee{
    id: 1001,
    name: "Rao",
};
```

## 6. Ambiguity-resolution rules

1. `{ ... }` is parsed according to its syntactic position: declaration/block body, map literal, annotation map, or object-construction body.
2. `|x| => expression` starts a lambda only where a primary expression may begin; infix `|` is otherwise bitwise OR.
3. A qualified name followed by `{` is object construction only when the name resolves in type position.
4. Type argument/derivation syntax is parsed in type positions; call syntax is parsed in expression positions.
5. Range operators are non-associative. `a..b..c` is invalid unless parentheses explicitly create nested ranges.
6. Collection/map/object trailing commas are accepted; semicolons are not substitutes for commas.
7. Planned constructs remain feature-gated even when represented in the grammar.
8. C++ literal maximal munch is applied before FoLang operator parsing, so hexadecimal exponents, suffixes, and raw strings remain single literal tokens.
9. A C++ user-defined-literal spelling is not resolved through C++ `operator""` lookup. It is accepted only when the same source form is independently defined by FoLang's user-defined-literal mechanism.
10. FoLang user-defined literals are resolved by FoLang type and literal declarations before C++ lowering; the selected constructor, aggregate, factory, helper, or native-literal representation does not affect source parsing.

## 7. Existing contextual rules retained

The generated grammar does not weaken the reference's contextual rules:

- ordinary package files contain exactly one primary top-level declaration;
- entry files permit their restricted entry-local declarations and executable statements;
- library surfaces permit only boundary declarations and adapter functions;
- units contain functions and companion ownership is checked semantically;
- no physical local or nested named declarations are introduced;
- `forall(T).type` remains type-position syntax;
- capability, visibility, realm, capture, and definite-initialization rules remain semantic checks.

## 8. Reference-document edits implied by these decisions

Before declaring the grammar normative, update `language-ref.md` to:

1. add the ASCII identifier contract and `_fo` C++ lowering rule;
2. explicitly prohibit consecutive and trailing underscores;
3. add the C++-compatible built-in literal contract and raw-lexeme preservation rule;
4. replace `_` numeric separators with apostrophes and replace `0o` octal examples with leading-zero octal;
5. add C++ integer and floating suffixes, hexadecimal floating literals, encoding-prefixed literals, universal character names, and raw strings;
6. explicitly exclude C++ `operator""` lookup and `nullptr` from built-in literal pass-through;
7. add the FoLang-specific user-defined-literal model and document its declaration and invocation syntax when finalized;
8. state that user-defined literals may lower to native C++ literals, constructors, aggregate initialization, factories, generated helpers, or ADT/variant construction;
9. add the built-in precedence table;
10. state assignment's right associativity and value result;
11. state the mandatory-semicolon/no-ASI rule;
12. normalize examples by adding missing semicolons to simple statements and standalone directives;
13. replace enum semicolon separators with commas;
14. replace `=` with `:` inside object initializers and annotation maps;
15. document custom-operator collection and precedence integration.
