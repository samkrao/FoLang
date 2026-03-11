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

## 3. Shared Interfaces

The Shared layer defines stable **contracts and interfaces** that any backend plugin must conform to. It is not itself a plugin — it is the integration boundary between the Frontend and any Backend implementation.

### Purpose

- Defines the HIR schema that the Frontend produces and any Backend must consume
- Defines the IPC protocol and wire format contract
- Enables third parties to build custom backend implementations against a stable interface
- Acts as the strict boundary that allows Frontend and Backend to evolve independently

### What Third Parties Can Do

Using the shared interfaces, third parties can provide custom Backend implementations in any language — as long as they conform to the HIR schema and IPC protocol declared here.

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

### Backend Kinds

There are two kinds of backend:

**Kind 1 — Plugin backend** implemented in the same language as the frontend (Go). Integrates directly via shared interfaces which are already versioned for backward compatibility. The frontend does not emit protobuf — it communicates through the shared interfaces directly. Config needs only plugin path and version.

**Kind 2 — Independent backend** implemented in any language. The frontend emits HIR over an IPC boundary in the declared wire format. Config declares the full protocol, schema, and wire format.

---

### Configuration File Structure

**Kind 1 — Plugin backend**

```json
{
  "plugin":  "libs/folang-plugin",
  "version": "v1"
}
```

**Kind 2 — Independent backend**

```json
{
  "protocol":   "folang-plugin/1.0",
  "hir_schema": "folang-hir/1",
  "wire":       "protobuf"
}
```

When encrypted libraries are used, both kinds add `vault` and `libraries` blocks:

```json
{
  "vault": {
    "endpoint":    "https://vault.internal/folang",
    "auth":        "mtls",
    "client_cert": "certs/consumer.crt",
    "client_key":  "certs/consumer.key",
    "ca_cert":     "certs/vault-ca.crt"
  },
  "libraries": [
    {
      "path":     "libs/mylib.folenc",
      "key_name": "mylib-decryption-key"
    }
  ]
}
```

### Configuration Contract

**Kind 1 — Plugin backend fields:**
- `plugin` — path to the backend plugin, must be implemented in Go using the shared interfaces
- `version` — shared interface version — frontend rejects if incompatible

**Kind 2 — Independent backend fields:**
- `protocol` — plugin communication protocol version — frontend rejects if mismatched
- `hir_schema` — HIR schema version the backend understands
- `wire` — serialization format for HIR messages (`protobuf` or `json`)

**Shared optional fields (both kinds):**
- `vault` — single global config for all encrypted library key fetches — mTLS credentials declared once, applies to all libraries
- `libraries` — list of encrypted libraries, each needing only `path` and `key_name`

`vault` fields:
- `endpoint` — internal vault server URL
- `auth` — authentication mode (`mtls`)
- `client_cert` / `client_key` — build server's mTLS identity for vault authentication
- `ca_cert` — vault CA certificate for server verification

---

## 4a. Why Protobuf Alone Does Not Protect Proprietary Code

Protobuf is a serialization format, not encryption. It produces compact binary output but:

- Anyone with the `.proto` schema can deserialize it completely
- The HIR schema is published — it must be, for backend plugins to work
- Tools like `protoc` decode any protobuf message back to readable JSON given the schema
- It is no more protected than a ZIP file

What travels over the IPC boundary is the **HIR** — the entire semantic structure of the program. Types, functions, logic, all of it. A proprietary company's business logic is fully visible in the HIR regardless of serialization format.

### What wire format and transport each do

```
wire=protobuf   →  compact, structured, schema-validated
                   NOT confidential — schema is public

transport=tls   →  encrypted channel — HIR not visible in transit
                   server authenticated — consumer knows it is the real backend

transport=mtls  →  both sides authenticated
                   backend knows which consumer is connecting
                   consumer knows it is the real backend
```

Both are needed together. Protobuf without TLS — structured but visible. TLS without protobuf — encrypted but unstructured. Together — structured, validated, and confidential.

---

## 4b. Proprietary Library Protection Model

For vendors who ship compiled FoLang libraries and need to protect their IP from reverse engineering, FoLang supports an **in-memory decryption model at link time**. The decrypted code never touches disk.

### The Problem

A vendor ships their compiled library as encrypted code. Without protection:
- Distributing source → fully readable
- Distributing compiled HIR → readable with `protoc` + schema
- Distributing native binary → reversible with tools like Ghidra

The protection must happen at link time — decrypt in memory, link, discard — so no plaintext is ever written to disk.

---

### What The Backend Does — And Only This

The backend's responsibility is narrow and simple. All operational complexity of how the key was obtained, stored, and managed is outside the backend's concern entirely.

```
At link time:

    1. Connect to vault
    2. Authenticate with password
    3. Fetch key by name
    4. Decrypt library in memory
    5. Link with consumer's compiled code
    6. Discard key and plaintext
    7. Done
```

The backend knows nothing about license servers, token brokers, key rotation, or mTLS. It only knows: vault endpoint, password, key name.

---

### Backend Configuration

For encrypted libraries, add `vault` and `libraries` blocks to whichever backend kind config is in use. The vault is configured once globally — all library key fetches use the same connection. Refer to Section 4 for the full configuration structure of both backend kinds.

---



### What This Protects Against

| Attack | Protected? | How |
|---|---|---|
| Reading `.folenc` directly | ✅ | Encrypted at rest |
| Intercepting IPC channel | ✅ | mTLS on backend transport |
| Key visible in config | ✅ | Password from env var, key fetched from vault |
| Disk forensics after build | ✅ | Plaintext never written to disk |
| Key visible in build logs | ✅ | Key fetched at runtime, not logged |

---

### Honest Limits

```
Memory forensics    — plaintext exists in process memory during the linking window
                      build process should run in an isolated, minimal environment

Compiled binary     — final native binary is still reversible with tools like Ghidra
                      obfuscation, legal protection (EULA/copyright), and SaaS
                      deployment are complementary layers for runtime protection

Vault availability  — if vault is unreachable, build fails
                      vault availability is a consumer operational concern
```

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
| Shared   | ✅ Interfaces  | Backend integration contracts and HIR schema  | Go                                | MIT          |

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
