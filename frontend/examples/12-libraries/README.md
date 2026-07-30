# 12 — Libraries

Spec: *Libraries*, *Library Surface file*, *Library Kinds*, *Boundary-Adapter
Functions*, *Consumer API Projection*, *Dependency Direction*, *imports*.

## The unified surface model

Every library kind has the same conceptual shape:

```text
library surface
├── boundary data contracts
└── public boundary-adapter functions
```

The kind changes the permitted boundary representation and transfer semantics —
never the shape of the API.

| Kind | Boundary data | Transfer |
|---|---|---|
| `application` | `co.lang.struct` | automatic deep snapshot |
| `dynamicvmrt` | `co.lang.struct` | automatic deep snapshot |
| `advanced` | `co.lang.struct` | automatic deep snapshot |
| `system` | `co.lang.cstruct` | system ABI value |
| `ffi` | `co.lang.cstruct` | C ABI value |

`struct` is a FoLang semantic data contract; `cstruct` is a physical
ABI-compatible value contract. Value transfer is not the same thing as
C-compatible layout, which is why the application family uses expressive
structs snapshotted deeply rather than restricted cstructs.

## Layout

```
12-libraries/
├── hrlib/          @co.dap.library(type="application")
│   ├── hrlib.fol       the surface
│   └── emp/            internal package, invisible to consumers
├── driver/         @co.dap.library(type="system")
├── clib/           @co.dap.library(type="ffi")
├── metart/         @co.dap.library(type="dynamicvmrt")
└── consumer/app.fol    imports all of them
```

## What a surface may contain

Only: the imports its adapters need, `struct` boundary declarations
(application family) or `cstruct` boundary declarations (system/ffi), and
public free-function API declarations with boundary-adapter definitions.

Forbidden in **every** surface: classes, modules, interfaces, signatures, units
and companion units, associated functions, operator functions, enums, unions
and other ADTs, type aliases, newtypes and opaque types, objects, instances,
type classes, dependent types, macros, templates, annotations, decorators, and
any global variable, pointer, reference, address, or mutable surface state.

## Public signature type closure

Every public signature must be closed over the library's exported boundary
types, recursively through their fields. An internal package type may never
appear in a public signature or surface field, and pointers, references, and
addresses may never cross any public surface.

Forbidden in public fields and signatures: `co.lang.auto`, `co.lang.infer`,
`co.lang.dynamic`, `co.lang.any`, `co.lang.typed`, `co.lang.untyped`, function
/ closure / delegate / loader / realm / AST / reflection / runtime values,
pointer / reference / address / thunk / handle types, and anything whose
reachable representation contains one of those.

## Dependency direction

```text
application → dynamicvmrt → advanced → system → ffi
```

Dependencies flow downward only, through projected public surfaces. A library
may depend on the same or a lower level when that creates no cycle; any reverse
dependency is a compiler error.

## Consumer API projection

A consumer's symbol table receives the library identity and kind, complete
public boundary type definitions, public function names, parameter names and
types, result types, and linkage metadata.

It does **not** receive function bodies, the surface's imports, local
conversion variables, delegate targets, internal package paths, compiler
helpers, or internal business types. A definition can therefore exist in the
surface source and in the compiled artifact while the consumer sees only its
signature — and having the source available does not weaken that boundary.
