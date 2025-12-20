# FoLang docs
  
  1. [Design Choices](#Design)
  2. [Novel and Innovative Ideas](./NovelIdeas.md)
  3. [Language Docs for Application developers](./../frontend/docs/README.md)
  4. [Backend Docs for Language AST to Binary Code developers ](./../backend/docs/README.md)


## Design


<p align="center">
  <img src="./design.png" alt="Design" width="300" height="300"/>
</p>


# FoLang Design Overview

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

## 3. Shared Plugin Interfaces

The Shared layer defines stable interfaces for extensibility and integration.

### Capabilities

Using the shared plugin interfaces, third parties can:

1. Extend or enhance the Frontend (parsing, analysis, or tooling)
2. Provide custom Backend implementations that integrate with the Frontend

### Purpose

- Treats the Backend itself as a pluggable component
- Enables multiple backends to coexist or be swapped
- Acts as a strict integration boundary between Frontend and Backend
- Supports independent evolution of all components

### License

- **MIT License**

---

## Licensing Summary

| Layer     | Responsibility                          | Implementation                     | License        |
|----------|------------------------------------------|------------------------------------|----------------|
| Frontend | Parsing and semantic analysis             | Go                                 | GPLv3          |
| Backend  | IR processing and native code generation | Go (orchestration) + C++ (target)  | BSD 3-Clause   |
| Shared   | Plugin interfaces and contracts          | Go                                 | MIT            |
