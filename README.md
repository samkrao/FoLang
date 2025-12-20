# ⚙️ In Active Development

> [!Note]
> This repository is currently a public placeholder. 
>
> The language and its toolchain are under active research and implementation in private repositories.
>
> Development is progressing little by little toward a functional implementation.
>
> A stable version—either complete or a working subset—will be made available here when it’s ready.
>
> Follow this repository for release notifications.
>
-----

<p align="center">
  <img src="Banner_52.png" width="400" alt="Foλang Logo"/>
</p>

<p align="center">
  <a href="https://github.com/samkrao/folang/releases"><img src="https://img.shields.io/github/v/release/samkrao/folang?color=3cb4ac&style=flat-square" /></a>
  <a href="LICENSE.txt"><img src="https://img.shields.io/badge/license-GPLv3-blue?style=flat-square" /></a>
  <a href="shared/LICENSE.txt"><img src="https://img.shields.io/badge/plugin%20API-MIT-green?style=flat-square" /></a>
  <img src="https://img.shields.io/badge/spec-CC%20BY%204.0-orange?style=flat-square" />
</p>

# Foλang Programming Language

Foλang is a general-purpose programming language designed to be **expressive, consistent, and extensible**, merging functional fluency with object-centric abstractions.

---

## 📌 Table of Contents
1. [Overview](#overview)
2. [Licensing Model](#licensing-model)
3. [Documentation](#documentation)
4. [Downloads](#downloads)
5. [Building From Source](#building-from-source)
6. [Dependencies](#dependencies)
7. [Plugin Ecosystem & Legal Docs](#plugin-ecosystem--legal-docs)
8. [Acknowledgments](#acknowledgments)

---

## Overview

Foλang combines:

✨ Functional programming fluency  
✨ Object semantics  
✨ Modern syntax theory  

This project originated in **2025**, evolving into a structured language platform.

---

## 📜 Licensing Model

FoLang uses a **multi-license architecture**:

### 📘 Language & Specification — CC BY 4.0  
✔ Reusable with attribution  
🔗 https://creativecommons.org/licenses/by/4.0/


### 🔧 Compiler Frontend — GPLv3 + Plugin Exception  
See [LICENSE](frontend/LICENSE.txt) and [PLUGIN_EXCEPTION](shared/PLUGIN_EXCEPTION.md)

### 🧱 Backend Binary Code Generator — BSD 3-Clause  
See [LICENSE](backend/LICENSE-BSD-3-CLAUSE.txt)
🔗 https://opensource.org/licenses/BSD-3-Clause

### 🔌 Plugin / Shared API Layer — MIT License
✔ Allows open, commercial, or closed plugins  
📌 See [LICENSE](shared/LICENSE.txt)  
🔗 https://opensource.org/licenses/MIT

---

## Documentation 

[FoLang Guide](docs/README.md)


## Downloads

➡ Official Releases  
https://github.com/samkrao/folang/releases

---

## Building From Source

```sh
git clone https://github.com/samkrao/folang.git
go get -u ./...
```

---

## Dependencies

### Windows
✔ MinGW / MSYS2 / Winlibs / TDM-GCC

### Linux
✔ Default GCC toolchain

### Clang
🔧 Port in progress

---

## Plugin Ecosystem & Legal Docs

FoLang supports third‑party plugins under flexible licensing.

Documents available in `plugin_license/`:

- [CLA](plugin_license/CLA.txt) — Contributor License Agreement  
- [EULA](plugin_license/PLUGIN_EULA.txt) — Commercial license template  
- [BADGE](plugin_license/CERT_BADGE.txt) — Label text for certified plugins  
- [PLUGIN POLICY](plugin_license/PLUGIN_POLICY_README.md) — Rules for plugin authors

---

## Acknowledgments

Inspired by:

- Bob Nystrom
- David Callanan
- Tyler Laceby
- ChatGPT

See [CREDITS](docs/CREDITS.md) for full attribution.

See:  
➡️ [MILESTONES.md](./ROADMAP.md)  
(A complete milestone checklist with progress tracking)




---
> © 2025 — FoLang Project  
---