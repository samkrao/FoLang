# 11 — Metaprogramming and Execution Models

Spec: *Macros*, *Templates*, *Annotations and Decorators*, *Extensions*,
*Execution Models and Control Abstractions*, *Library Kinds > `advanced`*.

Macros, templates, and execution machinery are **internally forbidden** in an
`application` library. They belong to an `advanced` library, so this directory
is a source-library project:

```
11-metaprogramming/
├── metalib.fol              @co.dap.library(type="advanced") — the surface
├── macros/                  package "macros"
├── templates/               package "templates"
├── annotations/             package "annotations"
├── extensions/              package "extensions"
└── execution/               package "execution"
```

Consumers import it with:

```folang
@co.ddap.import(package="metalib", src-library=true,
                expect="advanced", as="meta")
```

Only the projected surface API is visible; every internal package stays hidden
even though its source is physically present.

## What an `advanced` library may do internally

Allowed: all ordinary safe language features, macros and templates, async,
parallel, continuation, scheduling and transformation machinery.

Forbidden: raw pointers, references, addresses, and native functions — those
require a `system` or `ffi` library.

## Notes

- Directives and pragmas cannot be user-defined; they are language internals.
  Annotations (static objects that carry data) and decorators (functions that
  transform a target and return it) can be.
- Extensions are block-scoped and must be explicitly activated with
  `@co.ddap.use` before any extended method is callable.
