# FoLang grammar decision register — revision 6

This document is the complete companion decision register for `folang.ebnf` revision 6.
It records lexical, syntactic, parsing, ambiguity-resolution, and C++-lowering decisions
that are intentionally represented by the grammar.

The FoLang language reference remains authoritative for language semantics. The EBNF
specifies the complete syntax represented by the reference, including planned constructs.
A construct documented in `language-ref.md` remains part of the parser grammar unless it
is explicitly removed from the language reference. Parser support is not restricted merely
because a construct is marked planned or is scheduled for a later public compiler release.

## Status conventions

- **Normative syntax decision** — represented directly by an EBNF production.
- **Contextual constraint** — parsed by the grammar and validated by a later compiler phase.
- **Backend decision** — affects C++ lowering, not FoLang source name resolution.
- **Soft boundary** — separates related items while permitting another item to follow.
- **Hard boundary** — terminates a complete statement or structural body.

## Decision index

| Decision | Subject |
|---|---|
| `DECISION-SYN-001` | Simple-statement semicolons and structural termination |
| `DECISION-DIR-001` | Self-delimiting built-in directives |
| `DECISION-SYN-002` | Comma-grouped variable declarations |
| `DECISION-SYN-003` | Statement-level local function disambiguation |
| `DECISION-SYN-004` | Annotations on expression statements |
| `DECISION-SYN-005` | Bare blocks as statements |
| `DECISION-BLK-001` | Block tail expressions |
| `DECISION-LEX-001` | UTF-8 and ASCII-letter-first identifiers |
| `DECISION-LEX-002` | Whitespace and comments |
| `DECISION-LEX-003` | Maximal-munch tokenization |
| `DECISION-BACKEND-001` | `_fo` suffix for user-defined C++ names |
| `DECISION-OP-001` | Built-in precedence |
| `DECISION-OP-002` | Right-associative runtime assignment |
| `DECISION-OP-003` | Statement-only definition operators |
| `DECISION-OP-004` | Prefix and postfix increment/decrement |
| `DECISION-OP-005` | Reserved operator tokens |
| `DECISION-EXT-001` | Registered user-defined operators |
| `DECISION-LIT-000` | C++-compatible built-in literal spelling |
| `DECISION-LIT-001` | Integer literals |
| `DECISION-LIT-002` | Floating literals |
| `DECISION-LIT-003` | Character and string literals |
| `DECISION-LIT-004` | FoLang user-defined literals |
| `DECISION-LIT-005` | `co.const` Boolean and none literals |
| `DECISION-COL-001` | Collection, enum, map, and object punctuation |
| `DECISION-ANN-001` | Annotation binders and flag entries |
| `DECISION-TYP-001` | Attributes on every type derivation |
| `DECISION-TYP-002` | Type-constructor body parsing |
| `DECISION-TYP-003` | Elided array dimensions |
| `DECISION-GEN-001` | Higher-kinded generic arity |
| `DECISION-KIND-001` | General built-in-kind declarations |
| `DECISION-FUN-001` | Optional `=` before function blocks |
| `DECISION-FUN-002` | Curried and arrow closure declarations |

---

## 1. Statement and structural termination

### DECISION-SYN-001 — simple-statement semicolons

A semicolon is mandatory for every production that explicitly ends in
`statement-end`.

```ebnf
statement-end = ";";
```

This includes ordinary declarations and executable statements such as:

```folang
value co.lang.int = 10;
value = calculate();
this.return value;
co.out.println(value);
```

Rules:

1. Newlines are whitespace; FoLang performs no automatic semicolon insertion.
2. A semicolon is a **hard statement end**.
3. A closing brace is a **hard structural end** for the containing block or body.
4. A block-bodied declaration is not followed by a semicolon merely because its
   final token is `}`.
5. Built-in directives are the deliberate exception defined by
   `DECISION-DIR-001`.
6. Enum commas are soft item boundaries defined by `DECISION-COL-001`; enum
   variants are not ordinary semicolon-terminated statements.

```folang
Employee co.lang.struct = {
    id   co.lang.int;
    name co.lang.string;
}
```

The two fields end with semicolons. The struct body ends at `}`.

### DECISION-DIR-001 — built-in directives are self-delimiting

Standalone built-in directives do not require or accept a trailing semicolon.
The directive ends when its complete directive form ends. For a directive with
arguments, the closing `)` is sufficient.

```folang
@co.ddap.import(package="hr.employee", as="emp")
@co.ddap.alias(co.out, as="out")
@co.ddap.use(from="stringextension", extensions=[equals, upperCase])
@co.ddap.dynamicruntime
```

This applies to the active directive productions:

- `import-directive`
- `alias-directive`
- `use-directive`
- `dynamic-runtime-directive`
- `pragma-directive`
- `generic-directive`

A directive is not an ordinary simple statement even when it occurs among entry
items or in a file preamble.

### DECISION-SYN-002 — comma-separated variable declarations form one statement

A comma may join multiple declarators under one final semicolon:

```folang
x co.lang.int = 10, y co.lang.string = "Hello", z co.lang.bool = co.const.true;
```

The commas separate declarators inside one statement. The final semicolon is the
hard statement end.

The same principle applies to inferred declarations:

```folang
x := 10, y := 20;
```

### DECISION-SYN-003 — local function declarations are unambiguous

A function declaration used directly as a block statement must contain:

- a function name;
- at least one parameter list;
- a return-type clause; and
- a block body.

```folang
outer()->() = {
    inner()->() = {
        co.out.println("inside");
    }
    inner();
}
```

Requiring a return-type clause and body prevents an ordinary call such as
`inner();` from being misparsed as a forward declaration.

### DECISION-SYN-004 — annotations may prefix expression statements

An expression statement may carry annotations:

```folang
@co.dap.lazy
x = add(1, 2);
```

The annotation belongs to the expression statement rather than creating a new
standalone declaration form.

### DECISION-SYN-005 — a block is a statement

A bare block may appear where a statement is permitted:

```folang
{
    first();
    second();
}
```

The block does not take a trailing semicolon. Its closing brace is the hard
structural end.

### DECISION-BLK-001 — block tail expressions

A block may end with one expression that has no semicolon. That expression is
the block's value-producing tail expression and is not an expression statement.

```folang
classify(n) => {
    n + 1
}
```

Writing `n + 1;` instead makes it an expression statement and removes its role
as the block value.

---

## 2. Names, source text, and C++ lowering

### DECISION-LEX-001 — UTF-8 and ASCII-letter-first identifiers

- Source files are UTF-8.
- An optional U+FEFF BOM is permitted only at the beginning of a source file.
- Invalid UTF-8 is a lexical error.
- An ordinary identifier must begin with `A-Z` or `a-z`.
- Later characters may be ASCII letters, digits, or separated underscore
  segments.
- Consecutive underscores are prohibited.
- A trailing underscore is prohibited.
- A leading underscore is prohibited.
- A lone `_` is a dedicated contextual token, not an identifier.
- Identifier comparison is case-sensitive and byte-for-byte.

Equivalent regular-language description:

```text
[A-Za-z][A-Za-z0-9]*(?:_[A-Za-z0-9]+)*
```

Valid:

```text
employee
Employee1
employee_name
x_1_y2
```

Invalid:

```text
_employee
_1
employee__name
employee_
1employee
_
```

The lone `_` remains available only where a grammar production explicitly
admits it, including wildcard/discard patterns, iterator positions, generic
arity placeholders, and filename-derived primary declaration names.

### DECISION-BACKEND-001 — user-defined names receive `_fo`

Each resolved user-defined FoLang identifier is lowered to C++ by appending
`_fo`:

```text
employee       -> employee_fo
Employee       -> Employee_fo
employee_name  -> employee_name_fo
```

The original FoLang name remains in the source AST and symbol table. The C++
name is a backend result. Built-in names, externally linked names, compiler
internals, and reserved words follow their own lowering/linkage rules.

Because ordinary identifiers cannot end with `_` or contain `__`, appending
`_fo` does not create a C++-reserved double-underscore identifier.

### DECISION-LEX-002 — whitespace and comments

- Spaces, tabs, form feeds, and line terminators separate tokens.
- `//` introduces a line comment.
- `/* ... */` introduces a non-nesting block comment.
- Newlines do not terminate statements.
- Documentation comments remain ordinary comments until a separate
  documentation-comment model is specified.

### DECISION-LEX-003 — maximal munch

The lexer selects the longest valid token. Reserved multi-character tokens are
recognized before shorter prefixes.

```text
<..<  before  <.. or <
**=   before  ** or *
=>>   before  => or =
..<   before  .. or .
```

Comment openers are recognized before `/` is considered an arithmetic operator.

---

## 3. Operators and assignment

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
| 10 | runtime assignment operators | right |

Precedence determines expression structure. It does not override FoLang's
separately specified evaluation order.

Exponentiation binds more tightly than prefix negation:

```folang
-2 ** 2    // -(2 ** 2)
```

### DECISION-OP-002 — runtime assignment is right-associative

```folang
a = b = c;
```

parses as:

```folang
a = (b = c);
```

A runtime assignment expression yields the assigned value. Runtime assignment
operators include:

```text
=  +=  -=  *=  /=  %=  **=  &=  ^=  |=
```

### DECISION-OP-003 — definition operators are statement-level

- `:=` declares and initializes an inferred variable and errors if the name
  already exists.
- `?=` declares and initializes when absent; otherwise it reassigns the existing
  binding.
- `:=` and `?=` are not general expression operators.
- They cannot be chained as assignment expressions.
- `::=` remains reserved until assigned a language meaning.

### DECISION-OP-004 — increment and decrement fixity

`++` and `--` are accepted in prefix and postfix positions.

- Prefix form mutates the target and yields the new value.
- Postfix form yields the previous value and then mutates the target.
- The operand must be mutable and assignable.

### DECISION-OP-005 — reserved operator tokens

The lexer recognizes the following as reserved tokens, but the active expression
grammar rejects them until the language assigns semantics:

```text
::=  ->>  <->  `  \
```

The reserved-future Unicode glyph set is handled the same way outside literals
and declared operator-symbol positions. User-defined operators cannot silently
claim these tokens.

### DECISION-EXT-001 — registered custom operators

- Overloading a built-in symbol retains the built-in symbol's precedence,
  fixity, and associativity.
- A new operator symbol requires explicit fixity, numeric precedence,
  associativity, and arity metadata.
- Higher numeric precedence binds more tightly.
- Associativity is `left`, `right`, or `none`.
- Operator declarations are collected before custom-operator expressions are
  parsed.
- Maximal munch selects the longest valid operator token.
- Structural delimiters, comment delimiters, arrows, definition operators, and
  reserved operator tokens cannot be redefined.

---

## 4. Literal syntax

### DECISION-LIT-000 — C++-compatible built-in spelling

FoLang accepts C++-compatible lexical spelling for configured-backend numeric,
character, and string literals. The frontend preserves:

```text
literal category
complete raw source lexeme
```

The C++ backend may emit those compatible raw lexemes unchanged. Prefixes,
suffixes, apostrophe separators, escape spelling, encoding prefixes, and raw
string delimiters are therefore not reconstructed.

This pass-through rule does **not** mean that the resulting value adopts the C++
object model. The value remains a FoLang object/value under FoLang semantics.

Exceptions and exclusions:

- `co.const.true`, `co.const.false`, and `co.const.none` use the FoLang-specific
  forms defined by `DECISION-LIT-005` and require backend lowering.
- C++ `nullptr` is not introduced as a FoLang source literal.
- C++ `operator""` lookup is not imported.
- FoLang user-defined literals follow `DECISION-LIT-004`.

### DECISION-LIT-001 — integer literals

Supported forms include:

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
- octal: leading `0`;
- hexadecimal prefix: `0x` or `0X`;
- digit separator: apostrophe, only between digits;
- C++-permitted integer suffix combinations are accepted for the configured
  backend dialect;
- underscore is not a numeric separator.

### DECISION-LIT-002 — floating literals

Supported forms include:

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

- decimal floating literals use optional `e/E` exponents;
- hexadecimal floating literals require `p/P` exponents;
- apostrophes may separate digits;
- standard floating suffixes are supported;
- extended suffixes such as `f16`, `f32`, `f64`, `f128`, and `bf16` are
  accepted only when supported by the configured backend.

Maximal munch keeps `1..10` as integer `1`, range operator `..`, integer `10`,
rather than a floating literal followed by another dot.

### DECISION-LIT-003 — character and string literals

FoLang accepts configured-backend C++-compatible forms including:

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

The grammar includes encoding prefixes, C++ escapes, numeric escapes,
universal character names, raw strings with matching delimiters, and adjacent
string-literal sequences.

### DECISION-LIT-004 — FoLang user-defined literals

FoLang user-defined literals are resolved through FoLang declarations and type
rules, not C++ `operator""` lookup.

The frontend retains the original spelling and structured components. A backend
may lower a resolved literal to any semantically equivalent representation,
including:

- a native C++ literal;
- aggregate initialization;
- a constructor;
- a static factory;
- a generated helper;
- an ADT/variant constructor;
- a `constexpr` or `consteval` helper.

The selected representation is an implementation detail and does not define
FoLang source syntax.

### DECISION-LIT-005 — Boolean and none literal forms

FoLang Boolean literals are:

```folang
co.const.true
co.const.false
```

The FoLang none/null literal is:

```folang
co.const.none
```

The exact three forms are parsed as built-in constant literals even though their
surface spelling also resembles a qualified name. The literal interpretation
wins for these exact forms.

Bare `true`, `false`, `none`, and `null` are not FoLang literals. Bare names such
as `true`, `false`, or `True` inside annotation metadata remain ordinary
annotation-value names unless separately resolved by that context.

Typical backend lowering is contextual:

```text
co.const.true   -> C++ true or equivalent FoLang bool construction
co.const.false  -> C++ false or equivalent FoLang bool construction
co.const.none   -> pointer null, None/tag object, or another expected-type
                   representation
```

`co.const.none` is not specified as an unconditional alias for C++ `nullptr`.

---

## 5. Collection and annotation punctuation

### DECISION-COL-001 — canonical separators and boundaries

- Enum variants are comma-separated.
- A trailing enum comma is permitted.
- In an enum body, comma is a **soft item end** and `}` is the **hard structural
  end**.
- Array and tuple elements are comma-separated.
- Map entries use `key: value` and commas.
- Object-construction fields use `field: value` and commas.
- Ordinary map/object fields do not substitute semicolons for commas.
- Annotation metadata has the additional binder rules in `DECISION-ANN-001`.

```folang
Status co.lang.enum = {
    active,
    inactive,
}
```

```folang
employee := Employee{
    id: 1001,
    name: "Rao",
};
```

The final semicolon in the object example ends the surrounding variable
statement, not the object field list.

### DECISION-ANN-001 — annotation binders and flags

Annotation arguments and annotation-map entries accept either `=` or `:` as the
key/value binder. A bare key is a flag entry.

```folang
@co.dap.oops(
    A: { inherit:true, virtual:true },
    B: { implements=true }
)
```

```folang
@co.dap.generic(
    type={
        T:{typename},
        R:{variance:invariant, bound=Number}
    }
)
```

A bare flag has Boolean-enabled meaning equivalent to the language's true value
(`co.const.true`) in annotation semantics. It is not a new bare `true` literal.

---

## 6. Type, generic, and kind parsing

### DECISION-TYP-001 — every type derivation may carry attributes

A trailing derivation-attribute list may follow every derivation form, not only
pointer derivations.

```folang
value co.lang.int->(&, meta={type=out});
```

The generic derivation-attribute production also covers representations such as
`repr`, `sign`, `region`, and related metadata.

### DECISION-TYP-002 — a type-constructor body binds a type expression

A type constructor returns a type, so the right-hand side is parsed as a
`type-expression` before considering a general expression reading.

```folang
Vector(n co.lang.int)->(co.lang.dependentType) =
    co.lang.int->([n]);
```

Where a token sequence satisfies both type and value-expression shapes in this
position, the type-expression interpretation wins.

### DECISION-TYP-003 — array dimensions may be elided

Any position in an array-dimension group may be empty.

```folang
co.lang.int->([])
co.lang.int->([,])
co.lang.int->([2,])
co.lang.int->([,3])
```

The parser preserves the number and position of elided dimensions for later
type validation.

### DECISION-GEN-001 — higher-kinded generic arity

A generic parameter may declare arity:

```folang
Transformer(F(_), G(_))
```

An arity slot is `_` or a named placeholder. The `_` in this position is the
contextual underscore token, not an identifier.

### DECISION-KIND-001 — general built-in-kind declarations

A built-in kind without a dedicated declaration production is parsed by
`general-kind-declaration`. It may have:

- a block body;
- a type-expression body;
- an expression body; or
- a forward/empty form ending in `statement-end`.

This prevents a declaration such as:

```folang
blockormacro co.lang.kind = block | macro;
```

from being silently misparsed as an ordinary variable declaration.

Kinds that are deliberately treated as data types in declarator positions are
excluded from this catch-all declaration route.

---

## 7. Function and closure forms

### DECISION-FUN-001 — optional `=` before a function block

Both forms are accepted:

```folang
add(a T)->(T) = {
    this.return a;
}
```

```folang
add(a T)->(T) {
    this.return a;
}
```

The optional `=` applies to a block-bodied function definition. It does not turn
an ordinary call or expression into a declaration.

### DECISION-FUN-002 — curried and arrow closure declarations

The grammar recognizes curried declarations with two or more parameter lists:

```folang
add(first co.lang.int)(second co.lang.int)->(co.lang.int) = {
    this.return first + second;
}
```

It also recognizes arrow closure forms represented in the language reference:

```folang
closure(factor co.lang.int) =>
    (x co.lang.int) = x * factor;
```

```folang
curry(factor co.lang.int)(value co.lang.int) = factor * value;
```

Requiring multiple parameter lists for abbreviated curried forms avoids
confusing an ordinary assignment to a call result with a curried declaration.

---

## 8. Planned constructs and parser coverage

The complete grammar retains all planned constructs represented by
`language-ref.md`, including package aliasing and comprehension syntax.

Policy:

1. **Planned does not mean absent from parsing.**
2. A syntax production remains active unless the construct is explicitly
   removed from the language reference.
3. Early public compiler releases may omit later semantic/runtime support, but
   the parser may still recognize the complete language shape.
4. Availability, implementation maturity, and semantic completeness are not
   expressed by deleting the production from the consolidated grammar.
5. A compiler may emit an implementation-status diagnostic after parsing when
   execution or lowering is not yet implemented.

Consequently, the grammar intentionally retains:

```ebnf
package-alias-declaration = declaration-name, "co.lang.package", statement-end;
```

and the active `comprehension-expression` route.

---

## 9. Ambiguity-resolution rules

1. `{ ... }` is interpreted from syntactic position: declaration body, block,
   map, annotation map, or object-construction body.
2. `|x| => expression` starts a lambda only in a valid lambda/primary-expression
   context; infix `|` is otherwise bitwise OR.
3. A qualified name followed by `{` is object construction only when the name
   resolves in type position.
4. Type derivation syntax is parsed in type positions; call syntax is parsed in
   expression positions.
5. Range operators are non-associative; `a..b..c` requires explicit grouping or
   is rejected.
6. Collection, enum, map, object, and annotation lists may permit trailing
   commas according to their productions.
7. C++-compatible literal maximal munch occurs before FoLang operator parsing.
8. C++ user-defined-literal lookup is never used.
9. Exact `co.const.true`, `co.const.false`, and `co.const.none` spellings select
   built-in literal interpretation before ordinary qualified-name resolution.
10. A local function declaration requires its complete declaration shape so an
    ordinary call remains an expression statement.

---

## 10. Contextual rules intentionally outside pure EBNF

The parser recognizes syntax shape. Later phases remain responsible for:

- source-file kind and primary-declaration legality;
- package, entry-file, and library-surface restrictions;
- planned-feature implementation status;
- type checking and dependent-type evaluation;
- visibility and access control;
- import, realm, and symbol resolution;
- annotation applicability;
- operator ownership and overload legality;
- closure capture and definite initialization;
- lambda placement restrictions;
- wildcard, bind-variable, `$`, and `_` contextual legality;
- companion-unit ownership;
- ABI and capability restrictions;
- backend availability and lowering support.

Parsing a construct does not by itself make every contextual placement legal.

---

## 11. Reference normalization implied by revision 6

The following documentation corrections make examples agree with the grammar
without removing any planned work:

1. Enum variants should use commas rather than semicolons.
2. Built-in directives should be shown without trailing semicolons.
3. Ordinary statements whose productions use `statement-end` should retain
   their semicolons.
4. Block-bodied declarations should end at `}` without an automatic trailing
   semicolon.
5. Bare Boolean examples should use `co.const.true` or `co.const.false`.
6. None/null examples should use `co.const.none`.
7. Ordinary identifiers should begin with an ASCII letter; `_x` and `_1` should
   not be shown as valid identifiers.
8. A lone `_` should be documented as a contextual token rather than an
   identifier.
9. Object fields should use `:` and commas.
10. Annotation metadata may use `=` or `:` and may contain bare flag entries.
11. Planned constructs should remain documented and represented in the parser
    grammar unless explicitly removed from the language reference.

---

## 12. Validation status

The companion `grammar-validation.json` records the mechanical validation of
this revision. It distinguishes productions reachable from the syntactic root
`compilation-unit` from separately rooted lexical and informative productions.
Those separate roots are intentional and are not classified as undefined or
duplicate productions.
