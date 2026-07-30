# 08 — Generics

Spec: *Generics*, *Generic Functions — Parameters and Return Values*,
*Generic Types*, *forall*.

`@co.dap.generic` is the **one and only** way to declare a generic function,
struct, class, or other named declaration. `forall` is a type-level expression
only — `forall(T) name ...` at declaration level is a compiler error.

## Annotation fields

| Field | Values |
|---|---|
| `at` | `runtime`, `compiletime` (behaves like a C++ template) |
| `refied` | `true`, `false` |
| `where` | `usesite`, `callsite` |

## Type-parameter attributes

| Attribute | Values |
|---|---|
| `variance` | `covariant`, `invariant`, `contravariant` |
| `bound` | the type to bind |
| `kind` | `param`, `result`, `var`, `arg` |
| `default` | default type |
| `nullable`, `inclusive`, `impredicative` | boolean |
| `typekind` | `type`, `class`, `function`, `module`, `unit`, `package` |
| `types` | list of allowed types for a constraint |

## Rank support

| Scenario | Status |
|---|---|
| Rank-1 parameter and return, all syntaxes | ✅ |
| Rank-2 via `forall` — inline signature or `co.lang.type` alias | ✅ |
| Rank-2 or Rank-3 via a `co.lang.function` object | ❌ function objects are concrete values |
| Rank-3 via `forall` nesting | ✅ no new constructs needed |
| Impredicativity — wrap the `forall` type in `co.lang.type` | ✅ v1 workaround |
| Impredicativity — `impredicative:true` opt-in | 🔜 v2 |
