# Parser Conformance Audit

This audit compares `docs/grammar/folang.ebnf` with the recursive-descent
parser in `frontend/src/parser` and the full-parser fixtures in
`frontend/tests/parser/examples`.

## Scope and method

The grammar currently defines 303 productions. A production is considered
implementation-traceable when either:

1. its EBNF name appears in parser source beside the code implementing it; or
2. it is explicitly classified as scanner trivia/token syntax, a Pratt
   precedence layer, an informative semantic shape, or a zero-width guard.

`TestGrammarProductionsHaveImplementationTrace` enforces this inventory. It is
deliberately not called behavioral coverage: a comment can establish where a
production is intended to be implemented, but only a full parse can establish
that its accepted and rejected forms behave correctly.

## Traceability result

- 303 grammar productions inventoried.
- 248 productions are named directly in parser source.
- 55 productions are explicitly classified as scanner, Pratt, informative,
  or contextual-guard implementations.
- 0 productions are currently left without an implementation trace.

The 55 classified productions are listed in
`frontend/tests/parser/grammar_traceability_test.go`; additions to the EBNF
must either acquire a parser implementation trace or an explicit low-level
classification.

## Behavioral coverage added

The full-parser corpus was expanded beyond termination, operators, pointers,
maps, object construction, and numeric literals. New accepted fixtures cover:

- C structs, enums, unions, algebraic data declarations, interfaces,
  signatures, modules, objects, instances, and matcher instances;
- delegates and annotated primary functions;
- ordinary, curried, arrow, nested-local, and delegated function forms;
- literal, binding, constructor, record, tuple, wildcard, and capturing
  function patterns;
- pointer/reference/address/thunk/slice/range/array derivations, generic,
  union, and function types;
- tuple, array, map, object, grouped, range, logical, bitwise, power, call,
  index, member, lambda, comprehension, and let expressions;
- typed, inferred, grouped, let-value, multiple-assignment, expression,
  labeled-block, block-tail, and empty statements.

The fixture runner parses complete compilation units and rejects a dummy AST,
so these are parser tests rather than tokenizer-only examples.

## Known limits and non-parser responsibilities

The audit does not support the claim that every language feature is complete.
The following remain explicit limits:

- Import-cycle validation is a project-driver/import-graph responsibility,
  not a grammar production. It still needs direct and indirect package,
  source-library, self-import, and realm-cycle integration fixtures.
- The informative control-flow chain productions intentionally parse through
  ordinary postfix/member/call expressions. Their shape restrictions belong
  to semantic analysis and therefore cannot be proven by parser fixtures.
- Operator ownership is a second-pass parser validation, while operator type
  compatibility beyond owner-type matching belongs to later semantic typing.
- Binary AST serialization is explicitly not implemented by the parser
  driver. This is an output feature, not missing EBNF syntax.
- Reserved alpha features are correctly recognized and rejected; rejection is
  conformance, not an implementation omission.

## Completion criterion

Parser syntax can be called grammar-traceable when the traceability test and
all full-parser fixtures pass. The broader language can be called complete
only after separate semantic, import-graph, name-resolution, and type-checking
audits pass as well.
