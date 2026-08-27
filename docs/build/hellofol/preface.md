# Preface

## Welcome to *Hello, Foλang!*

Learning a programming language should begin with the satisfaction of making something work. It should not begin with every feature the language offers, every architectural choice a large application may require, or every rule contained in its specification.

*Hello, Foλang!* therefore begins with one file and one clear goal: help you write, build, and run useful FoLang programs without first learning the complete language.

The file is:

```text
src/appl.fol
```

It is FoLang's fixed application entry file. It needs no entry-point annotation and no ceremonial `main` function. In a single-source application, this file contains the complete application program.

## The promise of this book

By the end of *Hello, Foλang!*, you will be able to:

- create a single-source FoLang application;
- print values and call built-in package methods;
- work with literals and built-in types;
- declare, initialize, infer, and update variables;
- combine values with operators and expressions;
- make decisions with FoLang conditions;
- repeat work with loops;
- traverse arrays and built-in collections;
- use the built-in `co.*` packages;
- add an available third-party library to the project;
- import or alias its published API;
- call that library from `src/appl.fol`; and
- build and run the resulting application.

This is a complete learning goal. When you finish the book, you will have working programs and a useful understanding of FoLang's single-source application model.

It is not the whole language, and it is not intended to be.

## One source file, with useful dependencies

The projects in this book use the following source layout:

```text
<project>/
└── src/
    └── appl.fol
```

There are no user-created package directories beneath `src/`, and the book does not ask you to create packages or components. Everything you write belongs to the application entry file.

A single-source application does not have to be dependency-free. FoLang's built-in `co.*` packages are available to the entry file, and an application may also use a third-party library. The application source remains one file even when the project includes an external library artifact and `src/appl.fol` imports its API.

This distinction is important:

> Single-source means that the application's own program is contained in `src/appl.fol`. It does not mean that the application cannot use libraries.

The book teaches you how to consume those APIs. Creating your own multi-file package or component belongs to a later stage of learning.

## Learning through complete programs

The examples are small enough to understand in one sitting, but each one is a complete application rather than an isolated syntax fragment. You will see a result, change the program, build it again, and observe how that change affects its behavior.

The chapters progress gradually:

1. create and run the first application;
2. print values with built-in package methods;
3. introduce literals, types, and variables;
4. build expressions from values and operators;
5. make decisions with conditions;
6. repeat operations with loops;
7. process arrays and built-in collections;
8. combine the constructs in a small application;
9. add and import a third-party library; and
10. finish with a useful single-source program that builds and runs as one application.

Only the ideas needed for these programs are introduced. A construct is explained when it solves a visible problem, not merely because FoLang contains it.

## How to use this book

Type the examples when you can. Run them before changing them. Then make one change at a time and observe the compiler or the program.

When an example uses a built-in package path such as `co.out`, read the complete path before reaching for an alias. When an example introduces an alias or third-party import, notice what becomes shorter and what remains unchanged. When a condition or loop looks unfamiliar, follow the program's behavior rather than translating it immediately into another language's syntax.

The examples use FoLang's own vocabulary. Learning that vocabulary directly will make the programs easier to read and the compiler's feedback easier to understand.

## This book and the Language Reference

*Hello, Foλang!* is an explanatory and practical guide. It selects the rules needed for its programs and presents them in a learning order. It does not define the language.

The **FoLang Language Reference** is the normative definition of FoLang. If this book and the applicable Language Reference differ, the Language Reference governs. Readers who want the exact grammar, complete semantic rules, or behavior outside this book's scope should consult the reference.

The examples in this edition are written for the applicable current language reference and single-source application rules. Toolchain commands may evolve separately, so the edition should be used with its identified compiler and reference version.


