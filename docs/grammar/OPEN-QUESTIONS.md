# Open questions

Recorded, not decided. Each entry has more than one defensible reading and no
evidence in `language-ref.md` selecting between them. A wrong ruling encoded in
the grammar is harder to reverse than a documented gap, so these stay open.

## OQ-001 — Is a forward-declared function object or data declaration expressible?

`forward-declarable-kind` lists `co.lang.function` and `co.lang.data`, but after
grammar revision 27 there is no syntactic route to either:

- `forward-type-declaration` spells `filename-derived-name`, and neither
  `co.lang.function` nor `co.lang.data` is a primary declaration any more
  (`DECISION-DECL-001`, `DECISION-DECL-002`);
- `function-object-binding` has no bare `statement-end` alternative, so
  `<name> co.lang.function;` is not admitted as a unit member either.

Both entries are therefore unreachable through the production that names them.

**Reading (a) — forward declaration does not apply to these kinds.**
Remove `co.lang.function` and `co.lang.data` from `forward-declarable-kind` and
state that only file-backed primaries may be forward declared.

**Reading (b) — it does apply.**
Add a bare `statement-end` alternative to `function-object-binding`, and the
equivalent for `data-declaration`, so a unit member can be forward declared.

Whichever is chosen must be applied to both kinds; they are in identical
positions and the parser currently treats them alike.

No test encodes either reading. The forward-object case was deliberately
dropped from the conformance suite rather than frozen to a guess.
