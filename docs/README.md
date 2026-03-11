# FoLang docs
  
  1. [FoLang Design Overview](#folang-design-overview)
  2. [Novel and Innovative Ideas](./NovelIdeas.md)
  3. [Language Docs for Application developers](./../frontend/docs/README.md)
  4. [Backend Docs for Language AST to Binary Code developers ](./../backend/docs/README.md)



# FoLang Design Overview


<p align="center">
  <img src="./design.png" alt="Design" width="600" style="max-width:100%;"/>
</p>


FoLang follows a deliberately different approach from conventional programming language designs.  
The system is structured to ensure **clear separation of concerns**, **license isolation**, and **extensibility through well-defined integration boundaries**.

---

## 1. Frontend

The Frontend is responsible for source-level analysis and semantic processing of FoLang programs.

### Components

- Scanner / Lexer  
- Parser  
- AST / Parse Tree Generator  
- Symbol Table Generator  
- Semantic Analyzer  

### Implementation

- Implemented in **Go**
- Generates AST representations in **Go structures** or **plain JSON**

### License

- **GNU General Public License v3 (GPLv3)**

### Why the Frontend Is Not Pluggable

FoLang's syntax is **fixed and closed**. All language extensibility — operators, generics, macros, annotations, type extensions, custom matchers — is expressed through the built-in `@co.dap` and `@co.ddap` annotation system within the fixed syntax. There is no mechanism to introduce new syntax from outside the language.

Because no external component can alter what the frontend parses or how it builds the AST, a frontend plugin architecture would add complexity with no benefit. The frontend is therefore a **single, fixed, non-pluggable component**.

| Extensibility need | How FoLang handles it |
|---|---|
| New syntax | Not allowed — syntax is fixed |
| New operators | `@co.dap.operator` — declared in-language |
| Generics | `@co.dap.generic` — annotation-driven |
| Macros | `@co.dap.macro` — AST manipulation in-language |
| Annotations / decorators | `@co.dap.annotation` etc. — in-language |
| Type extensions | `@co.dap.extension` — in-language |
| Custom matchers | `@co.dap.matcher` — in-language |

---

## 2. Backend

The Backend is responsible for transforming validated frontend output into executable artifacts.

The Backend itself is implemented and integrated as a **plugin**, using the same shared plugin interfaces available to third parties.  
The out-of-the-box (OOTB) backend is provided as a **default plugin implementation**.

### Components

- Intermediate Representation (IR) Generator  
- Native Binary Executable Generation  

### Implementation

- Backend orchestration and plugin integration implemented in **Go**
- Code generation target is **C++**
- Uses **Clang** or **GCC** to generate native binaries from generated C++ IR

### License

- **BSD 3-Clause License**

---

## Why the Backend Is a Plugin

As illustrated in the architecture diagrams, the Backend is intentionally treated as a **pluggable component** rather than a privileged or tightly coupled part of the system.

This design ensures that the Frontend depends only on **stable interfaces**, not on any specific backend implementation.  
As a result, backend implementations can be added, replaced, or evolved independently without requiring changes to the Frontend.

Treating the Backend as a plugin also establishes a clear **integration and licensing boundary**, enabling multiple backend implementations — each with different execution models, targets, or licenses — to coexist behind the same shared interface.

---

## 3. Shared Plugin Interfaces

The Shared layer defines stable interfaces for extensibility and integration across the FoLang ecosystem.

### Capabilities

Using the shared plugin interfaces, third parties can provide custom Backend implementations that integrate with the Frontend.

### Plugin Model

- **Backend plugins**
  - The FoLang project provides reference backend plugins, including:
    - A **Go-based backend plugin** that integrates directly with the Frontend
    - **Language-agnostic backend plugins**, implemented in any language
  - Backend plugins are invoked via a **configuration file**
  - Communication between the Frontend and backend plugins occurs through **JSON and/or Protocol Buffers** over an IPC boundary

### Purpose

- Treats the Backend as a pluggable component
- Enables multiple backends to coexist or be swapped
- Acts as a strict integration boundary between Frontend and Backend
- Supports independent evolution of all components

### License

- **MIT License**

---

## 4. Plugin Configuration

FoLang uses a **minimal and explicit configuration file** to define how the Frontend and Backend are connected at runtime.

Each FoLang binary distribution includes:

- **Exactly one Frontend** — fixed, not configurable
- **Exactly one Backend** — selected via configuration

There is **no runtime plugin selection or discovery**.  
Different backend implementations are achieved by distributing different FoLang binaries or swapping the backend plugin.

---

### Configuration File Structure

```json
{
  "backend": {
    "plugin": "cpp-backend",
    "protocol": "folang-plugin/1.0",
    "hir_schema": "folang-hir/1",
    "wire": "protobuf"
  }
}
```

### Configuration Contract

The configuration establishes the only required compatibility contract between the Frontend and Backend:

- `plugin` — Identifies the backend executable plugin to load
- `protocol` — Specifies the plugin communication protocol version. If the protocol does not match, the Backend is rejected
- `hir_schema` — Declares the HIR (High-level Intermediate Representation) schema version understood by the Backend
- `wire` — Defines the serialization format used for protocol messages (e.g. `protobuf` or `json`)

---

## 5. Plugin Location and Resolution Rules

FoLang enforces strict and predictable plugin loading rules.

### Plugin Directory

All plugins must reside in a directory named: `folang_plugins`

This directory must be located next to the FoLang binary.

### Directory Layout Example

```
folang/
├─ folang                     # FoLang executable
└─ folang_plugins/
    └─ backend/
        └─ cpp-backend
```

### Resolution Rules

- Plugins are referenced by name only
- No absolute or relative filesystem paths are allowed in configuration
- No environment-variable-based plugin discovery is supported
- No fallback or search paths are used

At runtime, FoLang resolves the backend plugin as:

```
<binary-dir>/folang_plugins/backend/<plugin-name>
```

---

## Licensing Summary

| Layer    | Pluggable? | Responsibility                           | Implementation                    | License      |
|----------|------------|------------------------------------------|-----------------------------------|--------------|
| Frontend | ❌ Fixed   | Parsing and semantic analysis            | Go                                | GPLv3        |
| Backend  | ✅ Plugin  | IR processing and native code generation | Go (orchestration) + C++ (target) | BSD 3-Clause |
| Shared   | ✅ Plugin  | Backend plugin interfaces and contracts  | Go                                | MIT          |

---

## 6. Capability Security Model

FoLang's compiler ships with all language features compiled in but **systems and FFI features are disabled by default**. The compiler has no hardcoded keys — capability configuration happens entirely at install time. This moves authorization from source code (developer-controlled) to the compiler installation (organization-controlled).

---

### Feature Tiers

| Tier | Features | Default State |
|---|---|---|
| `application` | All standard language features, `co.net`, `co.core`, `co.encoding`, `co.crypto`, etc. | ✅ Always enabled |
| `systems` | Raw pointers, pointer arithmetic, `co.sys.unsafe`, MMIO, heap allocators | 🔒 Disabled — requires install-time configuration |
| `ffi` | `@co.dap.native`, `co.sys.ffi`, extern types, `co.lang.void` pointers, C ABI | 🔒 Disabled — requires install-time configuration |

---

### Installation Modes

#### Personal Mode

For individual developers. Enables all features with a single checkbox and explicit agreement. No key or password required.

```
┌─────────────────────────────────────────────────────┐
│           FoLang Compiler Installation              │
│                                                     │
│  Installation Mode:                                 │
│                                                     │
│  ○ Managed — configure with cryptographic key       │
│              and password                           │
│                                                     │
│  ● Personal — enable all features                   │
│                                                     │
│  ┌─────────────────────────────────────────────┐   │
│  │ ⚠ Personal mode enables ALL language        │   │
│  │ features including systems-level and FFI    │   │
│  │ capabilities. You are responsible for safe  │   │
│  │ and correct use of these features.          │   │
│  │                                             │   │
│  │ Personal mode is NOT suitable for           │   │
│  │ organizational or pipeline installations.   │   │
│  │                                             │   │
│  │ ☑ I understand and accept responsibility    │   │
│  └─────────────────────────────────────────────┘   │
│                                                     │
│  [ Back ]                    [ Install ]            │
└─────────────────────────────────────────────────────┘
```

- No key, no password
- All features immediately available
- Developer explicitly acknowledges responsibility
- Suitable for personal machines only

#### Managed Mode

For organizations, build pipelines, and corporate assets. Requires a cryptographic key and password at install time. Only declared features are enabled — everything else is disabled at the compiler level.

```
Installation prompts:
    1. Cryptographic key
    2. Password
    3. Which features to enable — systems, ffi, or subsets
            ↓
    Compiler configured with ONLY those features
    No other features accessible regardless of source code
```

---

### Post-Install Feature Management (Managed Mode)

All changes to feature configuration require authentication — preventing unilateral changes by individual developers.

**Add or remove features**
```
Requires: current password + current valid cryptographic key
Then: declare which features to add or remove
```

**Change password**
```
Requires: current password + current cryptographic key
Then: provide new password
```

**Key rotation**
```
Requires: current password + current cryptographic key
Then: provide new cryptographic key
Result: new key replaces old — old key invalidated
```

Rotation ensures that if a key is compromised, a new one can be issued — but only by whoever holds both the current key and the password.

---

### The Two Gates — Why Both Are Needed

Systems and FFI features require **both** gates to be cleared. Each serves a distinct purpose.

**Gate 1 — Annotation `@co.dap.zone(level=xxx)` on package**

The annotation is a **boundary wall and code organization gate** — works everywhere including personal machines. Zone applies to `co.lang.package` only — not individual functions, classes, or files. A package has exactly one zone. Once a package is marked systems, everything inside is systems — no mixing possible.

```folang
// ❌ Compiler error — systems construct in application zone package
hrPackage co.lang.package = {
    calculateSalary()->() = {
        gpio co.lang.word->(*) = ...   // error — application zone cannot use systems constructs
    }
}

// ✅ Correct — systems constructs inside a systems zone package
@co.dap.zone(level=systems)
driversPackage co.lang.package = {
    @co.dap.private
    doGpio()->() = {
        gpio co.lang.word->(*) = ...   // ✅ systems zone — allowed
    }

    @co.dap.public
    init()->(co.lang.bool) = { ... }   // only door out — application-safe type
}
```

The zone is a **boundary wall** — the only way to communicate across zones is through the public interface of the package. Internal systems details never leak out:

```folang
// ✅ Clean public interface — application-safe types only
DriversSignature co.lang.signature = {
    init()                     -> (co.lang.bool);
    readSensor(id co.lang.int) -> (co.lang.float);
}

// ❌ Compiler error — raw pointer cannot cross zone boundary
DriversSignature co.lang.signature = {
    getBuffer() -> (co.lang.byte->(*));   // error — systems type in public interface
}
```

**Gate 2 — Compiler feature switch (install-time)**

The feature switch is an **organizational enforcement gate** — works on managed installations.

```
zone=systems package present in source
        ↓
Compiler checks — is systems feature enabled in this installation?
        ↓ no — managed install, systems not configured
        ↓
COMPILER ERROR — systems feature not enabled
Package zone annotation is present but feature is locked
Source code change cannot fix this
```

Even if a developer correctly annotates their code, the compiler on a managed pipeline machine rejects it if the feature was never enabled at install time.

**Why both together**

```
Gate 1 alone  →  annotation enforces code organization ✅
               →  developer can annotate anything — no hard block ❌

Gate 2 alone  →  coarse on/off per feature ✅
               →  no code organization enforcement ❌

Both together →  annotation enforces segregation everywhere ✅
               →  compiler switch enforces org policy on managed installs ✅
               →  personal developer gets clarity without restriction ✅
               →  pipeline gets hard enforcement without annotation games ✅
```

---

### Import Incompatibility — Compiler Check

The compiler also enforces semantic incompatibility between imports and zone declarations. A systems zone package cannot import application packages — the combination is contradictory and rejected regardless of installation mode.

```folang
// ❌ Compiler error — systems zone package importing application package
@co.ddap.import(path="co/net", package="co.net", as="rest")

@co.dap.zone(level=systems)
driversPackage co.lang.package = { }
// ERROR: zone=systems package cannot import zone=application package co.net
// Systems zone is a boundary wall — no application imports allowed inside
```

| Import | Incompatible With |
|---|---|
| `co.net` | `@co.dap.zone(level=systems)` |
| `co.core` | `@co.dap.zone(level=systems)` |
| `co.encoding` | `@co.dap.zone(level=systems)` |
| `co.sys.unsafe` | `co.net`, `co.core`, `co.encoding`, `co.out` |
| `co.sys.ffi` | `co.net`, `co.core`, `co.encoding` |

---

### Enforcement Chain (Managed Installation)

```
Developer adds systems construct to application zone package
        ↓
Gate 1 — package zone declared?
        ↓ no — no @co.dap.zone on package
        COMPILER ERROR — systems construct in application zone
        ↓ yes — developer adds @co.dap.zone(level=systems) to package
Gate 1 — package importing application packages?
        ↓ yes — co.net imported in systems zone package
        COMPILER ERROR — systems zone cannot import application packages
        ↓ no — package imports are clean
Gate 1 — public interface leaking systems types?
        ↓ yes — raw pointer in signature
        COMPILER ERROR — systems types cannot cross zone boundary
        ↓ no — public interface is application-safe
Gate 2 — systems feature enabled in this compiler installation?
        ↓ no — managed pipeline, systems not configured at install
        COMPILER ERROR — feature not enabled
        ↓
BUILD FAILS — PR cannot merge
        ↓
Developer cannot fix by editing source code
Feature enablement requires password + key — outside developer's reach
```

---

### Where This Model Is Most Effective

| Context | Mode | Enforcement |
|---|---|---|
| Personal machine | Personal | Gate 1 only — code organization |
| Small team, shared machine | Managed | Both gates |
| CI/CD pipeline | Managed | Both gates — strongest |
| Corporate build server | Managed | Both gates — strongest |

The managed installation on a pipeline machine is where the model is most effective — the IT or security team installs the compiler, configures the key and password, enables only the required features, and developers never have access to the installation. Any attempt to use unauthorized features fails at compilation, before code review, before merge.

---

### Honest Limits

```
Personal mode        — developer accepts all responsibility, no controls
Small teams          — key holder and developer may be the same person
Key + password theft — if both are compromised, installation can be reconfigured
Authorized features  — compiler says the feature is enabled, not that code is correct
                       bad systems code in an authorized file is still possible
```

The model is an organizational control, not a correctness guarantee. It prevents unauthorized capability use — it does not prevent incorrect use of authorized capabilities.

---

### Audit Log

Every build on a managed installation logs all systems and FFI feature usage automatically:

```
[2024-06-01T10:32:11Z] BUILD managed:AcmeCorp
  systems.unsafe  →  src/drivers/gpio.fol
  systems.alloc   →  src/allocator/arena.fol
  ffi.c           →  src/bindings/libssl.fol
```

Full audit trail without any developer action — emitted by the compiler on every managed build.
