# ⚙️ In Active Development

> [!NOTE]
> Foλang (ASCII: **FoLang**) and its compiler frontend are under active research and development.
>
> The language definition and frontend implementation are still evolving and are not yet ready for production use.
>
> Development is progressing steadily toward a functional, stable, and publicly usable release.
>
> This repository contains the Foλang language specification, documentation, and compiler frontend.
>
> Watch this repository for release announcements and significant project updates.

---

<p align="center">
  <img src="Banner_52.png" width="400" alt="Foλang Logo"/>
</p>

<p align="center">
  <a href="https://github.com/samkrao/FoLang/releases">
    <img src="https://img.shields.io/github/v/release/samkrao/FoLang?color=3cb4ac&style=flat-square" alt="Latest release"/>
  </a>
  <a href="LICENSE.txt">
    <img src="https://img.shields.io/badge/frontend-GPLv3-blue?style=flat-square" alt="Frontend licence: GPLv3"/>
  </a>
  <a href="https://creativecommons.org/licenses/by/4.0/">
    <img src="https://img.shields.io/badge/language%20definition-CC%20BY%204.0-orange?style=flat-square" alt="Language definition licence: CC BY 4.0"/>
  </a>
</p>

# Foλang Programming Language

**Functional Objects Language**

Foλang (ASCII: **FoLang**) is a general-purpose programming language designed to be **expressive, consistent, and extensible**, combining functional fluency with object-centric abstractions.

---

## 📌 Table of Contents

1. [Overview](#overview)
2. [Name](#name)
3. [Repository Scope](#repository-scope)
4. [Repository Contents](#repository-contents)
5. [Licensing](#licensing)
6. [Documentation](#documentation)
7. [Building the Frontend](#building-the-frontend)
8. [Releases and Roadmap](#releases-and-roadmap)
9. [Acknowledgments](#acknowledgments)

---

## Overview

Foλang combines:

- functional programming fluency;
- object-oriented and object-centric semantics;
- expressive and consistent syntax;
- extensible language and compiler architecture.

The project originated in **2025** and continues to evolve through active language research and implementation.

---

## Name

The canonical name of the language is **Foλang**, where the lowercase Greek
lambda (`λ`) replaces the `L` in **FoLang**.

**FoLang** is the canonical ASCII representation of **Foλang**. It is used where
Unicode is unavailable, inconvenient, or unsupported, including repository
names, URLs, package names, command-line tooling, and other ASCII-oriented
contexts.

```text
Canonical name:       Foλang
Canonical ASCII form: FoLang
Expanded name:        Functional Objects Language
```

The two forms identify the same language; **FoLang** is not a separate name or
variant of Foλang.

---

## Repository Scope

This repository is the canonical public repository for:

- the Foλang language definition and specification;
- syntax, grammar, and semantic documentation;
- language examples and reference material;
- the Foλang compiler frontend;
- frontend tests, build files, and development documentation.

The following components are intentionally maintained in separate repositories:

- backend implementations and code generators;
- shared or plugin API layers;
- official and third-party plugins.

Those components may follow their own development schedules and licensing terms. Their licences are documented in their respective repositories.

---

## Repository Contents

| Path | Purpose |
|---|---|
| [`src/`](src/) | Foλang compiler frontend source |
| [`cmd/`](cmd/) | Developer tools — doc generation, corpus extraction |
| [`tests/`](tests/), [`testdata/`](testdata/) | Parser test corpora and fixtures |
| [`docs/`](docs/) | Language specification, guides, and supporting documentation |
| [`ROADMAP.md`](ROADMAP.md) | Development milestones and progress |

---

## 📜 Licensing

This repository contains two separately licensed bodies of work.

### 📘 Foλang Language Definition and Documentation — CC BY 4.0

The copyrightable expression contained in the Foλang language definition and documentation—including its specification, syntax and grammar descriptions, semantic-rule descriptions, examples, diagrams, tables, and explanatory material—is licensed under the [Creative Commons Attribution 4.0 International License](https://creativecommons.org/licenses/by/4.0/), unless otherwise stated.

CC BY 4.0 permits copying, redistribution, and adaptation, including commercial use, provided that appropriate attribution is given, the licence is referenced, and modifications are indicated.

See the [Foλang documentation](docs/README.md) for the complete language-definition and documentation licence notice.

### 🔧 Foλang Compiler Frontend — GPLv3

The Foλang compiler frontend source code is licensed under the GNU General Public License version 3.

See the [frontend licence](LICENSE.txt) for the complete terms.

### Licence Scope

The licences stated in this repository apply only to the language-definition, documentation, and frontend materials contained here.

---

## Related Repositories

- [FoLang Backend](https://github.com/samkrao/folang-backend)
- [FoLang Shared API](https://github.com/samkrao/folang-shared)
- [FoLang Plugins](https://github.com/samkrao/folang-plugins)
---

## Documentation

- [Foλang Language Guide and Specification](docs/README.md)
- [Development Roadmap](ROADMAP.md)
- [Project Credits](docs/CREDITS.md)

---

## Building the Frontend

Clone the repository:

```sh
git clone https://github.com/samkrao/FoLang.git
cd FoLang
```

Download Go module dependencies and build the current frontend sources:

```sh
go mod download
go build ./...
```

The build process may evolve while the frontend remains under active development.

---

## Releases and Roadmap

- [Official releases](https://github.com/samkrao/FoLang/releases)
- [Development roadmap and milestone tracking](ROADMAP.md)

---

## Acknowledgments

FoLang has been informed and inspired by educational material and work from:

- Bob Nystrom;
- David Callanan;
- Tyler Laceby;
- ChatGPT, Gemini, and Claude.

See [CREDITS](docs/CREDITS.md) for full attribution.

---

> © 2025–2026 Foλang Project
