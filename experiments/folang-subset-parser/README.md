# FoLang subset parser written in FoLang

This directory is an intentionally non-compiling design prototype. It shows how
the FoLang tokenizer and parser can read when expressed in FoLang itself, without
depending on the Go frontend, filesystem APIs, logging, serialization, or
external libraries.

The prototype tokenizes source text and covers this subset:

- identifiers, qualified built-ins, annotations, literals, punctuation, arrows,
  definitions, assignments, comments, and operators;
- file-backed `struct`, `class`, `enum`, and `unit` declarations;
- unit functions and companion receiver functions;
- variable declarations and `this.return` statements;
- calls, member selection, literals, names, grouping, and infix expressions;
- named enum-state declarations and invocations;
- context-first dispatch with bounded lookahead only where context is insufficient.

`co.out.println(...)` is used only by `printNode`, standing in for eventual AST
output. There is no trace or debug logging.

The files intentionally favor clarity over completeness. Unicode normalization,
numeric bases, escape decoding, imports, symbol persistence, recovery, and
backend serialization are outside the prototype.

Suggested reading order:

1. `TokenKind.fol`
2. `Token.fol`
3. `Tokenizer.fol`
4. `SyntaxKind.fol`
5. `SyntaxNode.fol`
6. `ParseContext.fol`
7. `CollectionTypes.unit.fol`
8. `Parser.fol`
9. `Demo.unit.fol`
