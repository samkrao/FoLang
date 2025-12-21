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

Treating the Backend as a plugin also establishes a clear **integration and licensing boundary**, enabling multiple backend implementations—each with different execution models, targets, or licenses—to coexist behind the same shared interface.

---

## 3. Shared Plugin Interfaces

The Shared layer defines stable interfaces for extensibility and integration across the FoLang ecosystem.

### Capabilities

Using the shared plugin interfaces, third parties can:

1. Extend or enhance the Frontend (parsing, analysis, or tooling)
2. Provide custom Backend implementations that integrate with the Frontend

### Plugin Model

- **Frontend plugins**
  - Must be implemented in **Go**
  - Run in-process with the Frontend
  - Extend or customize frontend behavior such as parsing, analysis, or tooling

- **Backend plugins**
  - The FoLang project provides reference backend plugins, including:
    - A **Go-based backend plugin** that integrates directly with the Frontend
    - **Language-agnostic backend plugins**, implemented in any language
  - Backend plugins are invoked via a **configuration file**
  - Communication between the Frontend and backend plugins occurs through **JSON and/or Protocol Buffers** over an IPC boundary

### Purpose

- Treats the Backend itself as a pluggable component
- Enables multiple backends to coexist or be swapped
- Acts as a strict integration boundary between Frontend and Backend
- Supports independent evolution of all components

### License

- **MIT License**


---

## 4. Plugin Configuration

FoLang uses a **minimal and explicit configuration file** to define how the Frontend and Backend are connected at runtime.

Each FoLang binary distribution includes:

- **Exactly one Frontend**
- **Exactly one Backend**

There is **no runtime plugin selection or discovery**.  
Different frontend/backend combinations are achieved by distributing different FoLang binaries.

---

### Configuration File Structure

```json
{
  "frontend": {
    "plugin": "default-frontend"
  },

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

    plugin: Identifies the executable plugin to load
    protocol: Specifies the plugin communication protocol version. If the protocol does not match, the Backend is rejected
    hir_schema: Declares the HIR (High-level Intermediate Representation) schema version understood by the Backend
    wire:  Defines the serialization format used for protocol messages (e.g. protobuf or json)

---

## 5. Plugin Location and Resolution Rules

    FoLang enforces strict and predictable plugin loading rules.

    Plugin Directory

    All plugins must reside in a directory named: folang_plugins
    
    This directory must be located next to the FoLang binary.

    Directory Layout Example

      folang/
      ├─ folang                     # FoLang executable
      └─ folang_plugins/
          ├─ frontend/
          │   └─ default-frontend
          └─ backend/
              └─ cpp-backend

      Resolution Rules

          Plugins are referenced by name only
          No absolute or relative filesystem paths are allowed in configuration
          No environment-variable-based plugin discovery is supported
          No fallback or search paths are used
      
      At runtime, FoLang resolves plugins as:
        
        Frontend plugin: <binary-dir>/folang_plugins/frontend/<plugin-name>
        Backend plugin: <binary-dir>/folang_plugins/backend/<plugin-name>

---

## Licensing Summary

| Layer     | Responsibility                          | Implementation                     | License        |
|----------|------------------------------------------|------------------------------------|----------------|
| Frontend | Parsing and semantic analysis             | Go                                 | GPLv3          |
| Backend  | IR processing and native code generation | Go (orchestration) + C++ (target)  | BSD 3-Clause   |
| Shared   | Plugin interfaces and contracts          | Go                                 | MIT            |
