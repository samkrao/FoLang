# MISSING production audit

Current against `docs/grammar-map.json` generated from
`docs/grammar/folang.ebnf`.

| Signal | Count |
|---|---:|
| Grammar productions | 427 |
| Productions claimed by indexed functions | 286 |
| Productions reported as `MISSING` | 141 |
| Parser functions reported as `EXTRA` | 68 |
| Productions with conflicting claims | 5 |
| Confirmed implementation gaps | 0 |

`MISSING` means that no function indexed by `cmd/docgen` claims the production;
it does not by itself mean that parser behaviour is absent. The current 141
entries fall into these implementation shapes:

- lexical productions implemented by `src/scanlex`;
- Pratt precedence levels and operator-table entries;
- delimiters, separators, body forms, and other short checks inlined into their
  enclosing parser;
- zero-width context and boundary guards;
- informative control-chain shapes parsed as ordinary postfix chains and
  recognized during lowering;
- built-in name registries implemented as scanner/parser tables.

Representative non-obvious entries were checked directly:

| Production | Implementation |
|---|---|
| `map-literal` | collection parsing in `parseCollectionBody` |
| `matcher-body` | matcher member loop in `parseMatcherMember` |
| `predeclared-glyph-expression` | reserved/predeclared operator handling in the expression parser |
| `operator-declaration-context-guard` | operator source/ownership validation |
| `component-surface-context-guard` | component source classification and surface validation |
| `builtin-metadata-name` and related registries | built-in annotation/directive/pragma registry tables |

The authoritative complete `MISSING` and `EXTRA` name lists are stored in
`docs/grammar-map.json`, avoiding a second hand-maintained copy here. A newly
missing production must be either given an `Implements:` claim or audited into
one of the implementation shapes above; a production with no implementation is
a parser gap and must not be described as a mapping-only omission.

`EXTRA` is a coverage signal too: it lists parse-shaped functions that have no
`Implements:` claim. `conflicts` is stricter: two functions claim the same
production, so at least one claim is stale. The current conflicts are `block`,
`operator-body`, `operator-declaration`, `operator-property`, and
`ordinary-relationship-selector-exclusion-guard`; they remain audit work and
are not included in the zero confirmed implementation-gap count.
