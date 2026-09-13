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
- variable declarations and symbolic `this =>`, `this ^=>`, `this ->>`, and
  `this ->|` control statements, including the anonymous-argument-block
  restriction on `this ^=>`;
- calls, member selection, typed JSON-like composite construction, scalar
  literals, names, grouping, and infix expressions;
- contextual compiler relationships through `this->parent`, `this->super`,
  `this->parents[Type]`, `this->classes[Type]`, `this->mixins[Type]`,
  `this->traits[Type]`, and `this->interfaces[Type]`; ordinary `this.member`
  remains dot-member access and `value->member` is not admitted;
- the binder boundary between `Type{key: value}` composite entries and
  `@metadata(field=value, nested={field=value})` metadata fields;
- named enum-state declarations and `State(name=value)` invocations;
- context-first dispatch with bounded lookahead only where context is insufficient.

`co.out.println(...)` is used only by `printNode`, standing in for eventual AST
output. There is no trace or debug logging.

The files intentionally favor clarity over completeness. Unicode normalization,
numeric bases, escape decoding, imports, symbol persistence, recovery, and
backend serialization are outside the prototype.

Because this prototype deliberately has no symbol table, it validates the
closed syntax of `this->` selectors but cannot prove that a selected type is a
direct parent, mixin, trait, or interface. The Go frontend performs that
semantic relationship check. Likewise, it retains named call arguments and
leaves the enum-state-only restriction to later symbol resolution.

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
