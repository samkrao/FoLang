# FoLang Examples

Example programs for every construct in the FoLang language, written against:

- [`docs/language-ref.md`](../../docs/language-ref.md) — the normative language reference
- [`docs/grammar/folang.ebnf`](../../docs/grammar/folang.ebnf) — the consolidated EBNF grammar

Each file names the construct it demonstrates and the reference section it comes
from. Where the reference marks a feature *planned*, the example says so in a
comment.

---

## Compilation-unit kinds

The grammar defines exactly three compilation units, and every file here is one
of them. This is why the examples are grouped by *project shape* and not only by
feature:

| Unit | Rule | Contents |
|---|---|---|
| Application entry file | `application-entry-file` | directives, entry-local type declarations, function-pattern groups, executable statements |
| Package source file | `package-source-file` | directives plus **exactly one** primary declaration |
| Library surface file | `library-surface-file` | directives plus one `co.lang.library` declaration |

Consequences visible throughout these examples:

- An entry file cannot declare functions, structs, classes, units, or modules.
  Anything that needs one of those lives in a package folder.
- A package source file holds one primary declaration, so each struct, unit,
  class, enum, macro, template, or type constructor gets its own `.fol` file.
- Free functions never float at package-file scope — they are enclosed in a
  `co.lang.unit`.

## Termination model (DECISION-SYN-006)

- `;` ends a simple statement, an expression-bodied declaration, and every
  forward declaration.
- `}` ends a declaration body, a function body, a function-pattern body, or a
  standalone block. No `;` follows a body-closing `}`.
- A `}` that closes object construction or a map expression ends only that
  expression — the enclosing statement still needs its `;`.
- Built-in directives (`@co.ddap.*`, `@co.pdap.*`) are self-delimiting and take
  no terminator.

## Literal subset accepted by the alpha release

- Strings are unprefixed, single-line, and contain no backslash or embedded `"`.
- A character literal holds exactly one non-backslash character.
- A floating literal needs a digit on both sides of the point: `1.0`, not `1.`.
- Booleans and null are `co.const.true`, `co.const.false`, `co.const.none`.
- Numeric literals contain digits only — no digit separators.

---

## Status

These examples target the **specification**, not the subset the frontend
currently implements. A file the parser rejects today is a gap in the frontend
(or a spec/grammar divergence worth reporting), not a defect in the example.

Last swept against the frontend parser: **122 of 137 files parse**. Every
remaining failure is one of the frontend gaps listed below.

### Where the reference and the grammar disagree

The grammar decides syntax, so these examples follow it and note the reference's
spelling in a comment.

- `nums.0` — *Mutation accessors* offers `nums.0` alongside `nums[0]`, but
  `member-suffix` accepts only an identifier or a lifecycle name after `.`.
  These examples use the indexed form.
- `let adjust(0) = offset` — the *Capturing `let` Function-Pattern Group*
  examples show no terminator, but `pattern-result` is `block,
  body-closure-guard | non-block-expression, statement-end`. These examples end
  an expression-bodied clause with `;`, matching the bare `=>` form.
- `funtype co.lang.type = (a co.lang.int, b co.lang.int)->(co.lang.int);` — the
  reference names parameters inside a function *type*, but `function-type` is
  `"(" [type-list] ")", return-type-clause` and a `type-list` holds type
  expressions only. Parameter names belong to a declaration or an
  anonymous-function expression.
- **Receiving multiple return values has no syntax.** *Normal* states that a
  function may return several values, and *Named Returns* declares them — but
  `multiple-assignment-statement` pairs N targets with N expressions in an
  `expression-list`, and `:=` takes one declarator. There is no destructuring
  form for one multi-result call. `07-functions/Basics.fol` uses an out
  parameter instead.
- `a Employee = { Name = "Kamesh" };` — the *Uniform Object Model* section
  brace-constructs a **class**, but the container comparison table marks
  value/literal construction ❌ for classes, and `object-field-initializer` uses
  `:` rather than `=`. These examples construct classes with `init(...)`.

### Frontend gaps

Grammar-legal constructs the parser rejects today:

| Construct | Rule | Seen in |
|---|---|---|
| `' '` — space in a character literal | DECISION-LIT-007 names space, `;`, `,` as ordinary c-characters; the scanner pattern excludes whitespace | `01-basics/literals.fol` |
| `self.return x;` | `return-statement = ( "this" \| "self" ), ".return", …` — only `this` is handled | `06-udt/Account.fol`, `08-generics/Employee.fol` |
| `(v T) name()` value receiver | `function-declaration = annotations, [receiver-clause], …`. Works as the first member of a body; after a preceding member the leading `(` is taken as a call suffix on the previous `}`, and after a no-argument annotation it is taken as that annotation's argument list. A type receiver `(T) name()` is fine. | `05-packages/…/Employee.unit.fol`, `06-udt/Vector.unit.fol` |
| parameter typed with a bare built-in kind, e.g. `target co.lang.function`, `T co.lang.type` | `parameter = … identifier … [ type-expression ]` | `09-types/Stack.fol`, `11-metaprogramming/annotations/MyDecorator.fol` |
| unparenthesized arrow tail `f (A)->B` | `arrow-type-tail = type-derivation \| parenthesized-type-list \| type-expression` | all six `10-typeclasses` instance/definition files |
| dependent-type application in a parameter or result, `Matrix(r, n)` | `type-postfix-expression = type-atom, { type-argument-list }` | `09-types/Geometry.fol` |
| `co.lang.Matcher->( … )` kind options | `matcher-instance-declaration = …, ( "co.lang.Matcher" \| "co.lang.matcher" ), [ kind-options ], …` — `co.lang.instance->( … )` works, this path does not | `10-typeclasses/PositiveEvenMatcher.fol` |

### Two rules that are easy to get wrong

Both bit these examples before they were corrected:

- **`_` is not universal.** Filename inference works through `declaration-name`.
  A type constructor, macro, decorator, or any `annotated-function-primary` uses
  `function-name = identifier | lifecycle-name`, and DECISION-LEX-001 makes a
  lone `_` a contextual token that is never an identifier. Those declarations
  must be named explicitly.
- **A forward type declaration is a primary declaration.** Pairing
  `Employee co.lang.struct;` with a unit in one file puts two primary
  declarations in a package source file.

## Directory map

| Directory | Unit kind | Constructs |
|---|---|---|
| [`01-basics/`](01-basics/) | entry files | hello world, variables, literals, operators, assignment, evaluation order, the uniform object model, built-in methods |
| [`02-variable-kinds/`](02-variable-kinds/) | entry files + a `system` source library | arrays, ranges, slices, auto/dynamic, pointers, references, addresses, thunks, fat pointers |
| [`03-control-flow/`](03-control-flow/) | entry files + one package file | conditions, loops, mixed chains, ternary, `each`, `contains`, pattern matching, comprehensions, labels, named blocks |
| [`04-entry-file/`](04-entry-file/) | entry files | function-pattern groups, every pattern kind, arity grouping, guards, desugaring, capturing `let` groups, entry-local types, imports and aliases |
| [`05-packages/`](05-packages/) | application project | package identity, multi-file packages, visibility, companion units, package alias |
| [`06-udt/`](06-udt/) | package files | struct, cstruct (packed/simd), enum, union, ADT, classes with inheritance and virtual dispatch, interface, signature, module, unit, companion unit, operators, indexer, `@co.dap.local` |
| [`07-functions/`](07-functions/) | package files | every function form, delegates, chaining, closures, currying, higher-order, scoping |
| [`08-generics/`](08-generics/) | package files | generic functions and types, ranks 1–3, `forall`, impredicativity workaround |
| [`09-types/`](09-types/) | entry file + package files | aliases, newtype, opaque, subtype/supertype, dependent types, type constructors, forward/extern declarations |
| [`10-typeclasses/`](10-typeclasses/) | package files | functor, applicative, monad, monoid, transformer, custom matcher |
| [`11-metaprogramming/`](11-metaprogramming/) | `advanced` source library | macros, templates, annotations, decorators, extensions, execution models, continuations |
| [`12-libraries/`](12-libraries/) | library surfaces + consumer | `application`, `system`, `ffi`, `dynamicvmrt` surfaces, boundary adapters, three import forms |

## Reading order

1. `01-basics` → `03-control-flow` — everything that fits in one source file.
2. `04-entry-file` → `05-packages` — how a real project is laid out.
3. `06-udt` → `07-functions` — the declaration model.
4. `08-generics` → `10-typeclasses` — the type system.
5. `11-metaprogramming` → `12-libraries` — capability-restricted contexts.

## Capability restrictions honoured by these examples

Some constructs are legal only inside particular library kinds
(*Variable Kinds Support*, *Library Kinds*):

| Construct | Allowed in |
|---|---|
| pointers, references, addresses, native code | `system` and `ffi` libraries |
| macros, templates, execution models, continuations | `advanced` libraries |
| `co.meta` reflection, dynamic loading | `dynamicvmrt` libraries |
| thunks / lazy | `application`, `advanced`, `dynamicvmrt`, `system`, `ffi` |
| normal variables, arrays, ranges, slices | everywhere |

That is why, for example, the pointer examples sit inside a `system` source
library rather than in an entry file.
