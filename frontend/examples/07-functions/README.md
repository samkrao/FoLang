# 07 — Functions

Spec: *Functions*, *Functions in detail*, *Forward / Extern Declarations*,
*Scoping Rules for Functions*.

FoLang allows no free-flowing functions in a package source file: every
function is enclosed in a `co.lang.unit` (or is a class method, module
function, or companion-unit associated function). Each file below is one
package source file with one primary declaration.

| File | Constructs |
|---|---|
| `Basics.fol` | normal functions, multiple returns, default / variadic / optional / named parameters, named returns |
| `Inline.fol` | `@co.dap.inline` |
| `Anonymous.fol` | anonymous functions, immediately invoked forms, lambdas |
| `Inner.fol` | inner functions and lexical capture |
| `Curried.fol` | curried functions and the abbreviated closure forms |
| `Closures.fol` | closures over enclosing state |
| `LetBindings.fol` | `let(...).in(...)`, `.where(...)`, `$`, plain `let` value declarations |
| `HigherOrder.fol` | functions taking and returning functions — all three syntaxes, both positions, composition, call sites |
| `FnArg.fol`, `FnRet.fol` | Syntax 2 — function types named with `co.lang.type` |
| `Handler.fol`, `Doubler.fol` | Syntax 3 — `co.lang.function` objects with inline bodies |
| `Adder.fol` | Syntax 3 — a function object bound to an existing callable |
| `Chaining.fol` | `=>>` delegation chains and `$1` bind variables |
| `Scoping.fol` | lexical scope, and the scope annotations only associated functions accept |
| `BinaryOp.fol` | `co.lang.delegate` |
| `Forward.fol` | forward and extern declarations |
| `Helper.fol` | `@co.dap.local` target-local function |

## Restrictions on special functions

Curried functions, functions with named / optional / default parameters,
variadic functions, functions that take or return functions or function types,
and dynamically or mixed-scoped functions:

- cannot be overloaded;
- cannot be used as callbacks;
- cannot participate in execution models and control abstractions.

Curried functions may not be variadic, and variadic functions may not be
curried.
