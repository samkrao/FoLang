# FoLang subset parser written in FoLang

This directory is an intentionally non-compiling design prototype. It shows how
the FoLang parser can read when expressed in FoLang itself, without depending on
the Go frontend, filesystem APIs, logging, serialization, or external libraries.

The prototype accepts an already-tokenized input and covers this subset:

- file-backed `struct`, `class`, `enum`, and `unit` declarations;
- unit functions and companion receiver functions;
- variable declarations and `this.return` statements;
- calls, member selection, literals, names, grouping, and infix expressions;
- named enum-state declarations and invocations;
- context-first dispatch with bounded lookahead only where context is insufficient.

`co.out.println(...)` is used only by `printNode`, standing in for eventual AST
output. There is no trace or debug logging.

The files intentionally favor clarity over completeness. Scanner behavior,
imports, symbol persistence, recovery, and backend serialization are outside the
prototype.

Suggested reading order:

1. `TokenKind.fol`
2. `Token.fol`
3. `SyntaxKind.fol`
4. `SyntaxNode.fol`
5. `ParseContext.fol`
6. `CollectionTypes.unit.fol`
7. `Parser.fol`
8. `Demo.unit.fol`
