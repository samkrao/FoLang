# FoLang Compilation Input and Parsing Model

## Project Invocation

FoLang compilation is invoked against a project root:

```text
folangcc <some-parent>/myApp
```

The project root has three compiler-input domains and one compiler-generated output domain:

```text
myApp/
├── src/
├── components/
├── lib/
└── build/
```

The compiler discovers and classifies the input domains before parsing primary project source. `build/` is compiler-managed output and is not a source-discovery domain.

---

## Project Layout

### `src/`

`src/` contains the primary project source surface.

Exactly one of the following top-level surface files is used:

```text
src/
├── appl.fol
```

or:

```text
src/
├── library.fol
```

### `src/appl.fol`

`appl.fol` is the application entry surface.

It identifies the project as an application source entry.

### `src/library.fol`

`library.fol` is the primary surface of a standalone distributable library project.

It has exactly two valid forms.

#### Projected library

```folang
@co.dap.library(type=...)

_ co.lang.library = {
    // public/surface APIs
}
```

`@co.dap.library(type=...)` determines the projected-library kind:

```text
application
ffi
system
advanced
dynamicvmrt
```

The following `_ co.lang.library = { ... }` definition is mandatory for this form and defines the projected public API surface serialized into the resulting `<project-name>.folenc`.

#### Package-export library

```folang
@co.dap.export(
    packages = {
        // package selections relative to src/
    }
)
```

This form selects package contexts under the project's `src/` package root as the distributable library surface. It has no projected `_ co.lang.library` API context.

The two forms are mutually exclusive. `_ co.lang.unit` is not a valid primary `src/library.fol` form.


---

## `components/`

`components/` contains project-owned specialized FoLang source components. Components are not independently built libraries: they are parsed and compiled as part of the owning project and do not produce separate `.folenc` artifacts.

The immediate child folder determines the component kind before its source is parsed. Every standardized component folder uses the same fixed structural surface filename:

```text
component.fol
```

```text
components/
├── application/
│   └── component.fol
├── ffi/
│   └── component.fol
├── system/
│   └── component.fol
├── advanced/
│   └── component.fol
├── dynamicvmrt/
│   └── component.fol
├── operators/
│   └── component.fol
└── exports/
    └── component.fol
```

The valid component surface shapes are:

```folang
// components/application|ffi|system|advanced|dynamicvmrt/component.fol
_ co.lang.library = {
    // projected APIs exposed from this component to the owning project
}
```

```folang
// components/exports/component.fol
@co.dap.export(
    packages = {
        // descendant component packages to expose
    }
)
```

```folang
// components/operators/component.fol
_ co.lang.unit = {
    <symbol> co.lang.operator = {
        // operator parse properties
    };
}
```

No component uses `@co.dap.library(...)`; the folder is the authoritative component-kind discriminator.

For `application`, `ffi`, `system`, `advanced`, and `dynamicvmrt`, `_ co.lang.library = { ... }` defines the component's projected API surface. Implementation source may reside in descendant private packages within that component.

For `exports`, the folder establishes `componentKind=exports`; `@co.dap.export(packages={...})` remains required because it supplies the package-selection data that determines which descendant export-component package contexts are exposed.

For `operators`, the folder establishes `componentKind=operators`; `component.fol` contains one `_ co.lang.unit = { ... }` and its body consists of `<symbol> co.lang.operator = { ... };` declarations. The operator component has no package subdirectories.

All component source uses the same FoLang parser and grammar as `src/` source. Component kind affects semantic/capability validation and compilation ordering, not grammar selection.

---

## `lib/`

`lib/` contains already compiled FoLang library artifacts:

```text
lib/
├── xxx.folenc
├── yyy.folenc
└── ...
```

`.folenc` files are not passed through the FoLang source parser.

Instead, the compiler loads/deserializes the information required from the artifacts, including applicable:

```text
symbol tables
contexts
AST information
library metadata
```

This information becomes part of the compilation environment before dependent FoLang source is parsed.

---

# Common Parsing Model

FoLang uses one common source grammar and one common parser for source under both `src/` and `components/`.

Conceptually:

```text
parseFoLang(source, compilationContext)
```

The parser does not select a different grammar for:

```text
application
ffi
system
advanced
dynamicvmrt
operators
exports
```

Instead, project discovery supplies compilation context to the parser and semantic phases.

A conceptual parsing context may contain information such as:

```text
ParseContext
{
    origin:
        src | components

    sourceRole:
        application-entry |
        library-surface |
        component-surface |
        package-source

    componentKind:
        application |
        ffi |
        system |
        advanced |
        dynamicvmrt |
        operators |
        exports |
        unknown

    
    preloadedSymbols
    preloadedContexts
    preloadedASTs
    operatorTable
}
```

The exact compiler data structures are implementation-defined. The important semantic rule is that component location supplies component context, while `src/library.fol` supplies its standalone library form from source: projected (`@co.dap.library` + `_ co.lang.library`) or package-export (`@co.dap.export`). None of these distinctions selects a different FoLang grammar.

---

# Determining Library and Component Kind

There are two distinct mechanisms.

## Component under `components/`

For any component surface such as:

```text
components/ffi/component.fol
```

the compiler already knows before parsing:

```text
origin        = components
sourceRole    = component-surface
componentKind = ffi
```

because the immediate folder under `components/` determines the component kind.

The same rule applies to `application`, `system`, `advanced`, `dynamicvmrt`, `operators`, and `exports`. No `@co.dap.library(...)` annotation is used in a component.

## Primary `src/library.fol`

For:

```text
src/library.fol
```

the compiler establishes exactly one standalone library form from the source.

Projected library:

```text
@co.dap.library(type=...)
    +
_ co.lang.library = { ... }

-> annotation supplies the library kind
-> co.lang.library supplies the projected public API
```

Package-export library:

```text
@co.dap.export(packages={...})

-> selections are relative to src/
-> selected package contexts form the distributable library surface
-> no projected co.lang.library surface exists
```

The two forms are mutually exclusive. `_ co.lang.unit` is not a valid primary `src/library.fol` form.


---

# Compilation Order

The compiler must establish imported and component compilation state before parsing the primary source under `src/`.

The high-level pipeline is:

```text
folangcc project-root
        │
        ▼
1. Discover project structure
        │
        ├── src/
        ├── components/
        └── lib/
        │
        ▼
2. Load compiled libraries from lib/
        │
        ├── validate .folenc artifacts
        ├── deserialize symbol tables
        ├── deserialize contexts
        └── deserialize applicable AST information
        │
        ▼
3. Process project-owned components under components/
        │
        ├── determine kind from folder
        ├── parse component.fol with the common FoLang parser
        ├── parse applicable package source
        ├── generate ASTs
        ├── generate symbol tables and contexts
        └── resolve as much as possible against already loaded libraries
        │
        ▼
4. Construct the initial project compilation environment
        │
        ├── symbol/context information loaded from lib/
        ├── applicable AST information loaded from lib/
        ├── symbol/context information generated from components/
        └── ASTs generated from components/
        │
        ▼
5. Parse the primary source under src/
        │
        ├── src/appl.fol
        │        OR
        └── src/library.fol
                 ├── @co.dap.library(...) + _ co.lang.library = { ... }
                 │       -> projected library
                 └── @co.dap.export(packages={...})
                         -> package-export library
        │
        ├── use the same FoLang parser
        └── resolve against the pre-established compilation environment
        │
        ▼
6. Complete semantic resolution
        │
        ▼
7. Merge applicable AST material
        │
        ▼
8. Produce Final AST
        │
        ▼
9. Pass Final AST to the backend
```

---

# Compiled Libraries versus Components

The important distinction is:

```text
components/
    contains FoLang source
        ↓
    common FoLang parser
        ↓
    AST + symbol tables + contexts
```

whereas:

```text
lib/
    contains compiled .folenc artifacts
        ↓
    artifact loader/deserializer
        ↓
    AST information + symbol tables + contexts
```

`.folenc` loading therefore occurs outside the normal source parsing function.

Its output is supplied to the compilation environment used by later source parsing and semantic resolution.

A projected-library `.folenc` carries the projected `src/library.fol` API and may also carry package contexts selected by the producer's `components/exports/component.fol`. A package-export-library `.folenc` instead carries the package contexts selected by `src/library.fol`'s `@co.dap.export(...)` form and has no projected `_ co.lang.library` API context. Components themselves never produce independent `.folenc` artifacts.

---

# Operator Component Ordering

`components/operators/component.fol` uses the same FoLang grammar as all other FoLang source.

It does not require a separate operator parser or a separate grammar root.

However, operator declarations may affect Pratt parsing of expressions in other source files.

Therefore the operator component must be processed early enough to establish the project operator table before parsing source that may use those operators.

Conceptually:

```text
load lib/*.folenc
        ↓
process components/operators/component.fol
        ↓
collect/register project operators
        ↓
process remaining component source
        ↓
parse src source
```

The distinction is therefore:

```text
special grammar             -> no
special parser              -> no
special compilation order   -> yes
folder-derived context      -> yes
```

---

# Grammar Principle

FoLang source parsing is grammar-uniform.

Project structure determines source role, component kind, processing order, and semantic/capability restrictions; `@co.dap.library(type=...)` determines the standalone library kind. It does not select a different FoLang grammar.

At the grammar level, the parser operates on the actual FoLang source-file forms rather than on individual library kinds.

For example, a common root may conceptually classify source as:

```ebnf
compilation-unit =
      package-source-file
    | application-entry-file
    | library-surface-file
    | component-surface-file ;
```

The precise productions belong to the normative FoLang grammar.

`ffi`, `system`, `advanced`, `dynamicvmrt`, `operators`, and `exports` are not separate parser grammars. Component contexts are determined by the immediate folder under `components/`; `src/library.fol` is classified from either its projected-library form (`@co.dap.library` + `_ co.lang.library`) or its package-export form (`@co.dap.export`).

---

# Governing Rule

> FoLang uses one common source grammar and parser. Project discovery establishes filesystem-derived component context before component parsing. Compiled `.folenc` libraries are loaded and deserialized before source parsing; project-owned `components/` source is then parsed to produce ASTs, symbol tables, and contexts; these results are supplied to the subsequent parsing and resolution of `src/appl.fol` or `src/library.fol`. `src/library.fol` always represents a standalone distributable library project and has exactly two forms: projected (`@co.dap.library` + `_ co.lang.library`) or package-export (`@co.dap.export`). Component kinds affect compilation order, capabilities, semantic validation, and surface rules—not the FoLang grammar.
