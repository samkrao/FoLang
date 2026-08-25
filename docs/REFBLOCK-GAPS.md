# Reference block conformance

This report is maintained alongside the generated corpora under
`testdata/refblocks/`. Regenerate the corpora from the repo root with:

```text
go run ./cmd/refblocks -write
```

## Current result

All 424 `folang` blocks in `docs/language-ref.md` are classified:

| Corpus | Document blocks | Contract |
|---|---:|---|
| `parsing/` | 225 | every block must parse |
| `invalid/` | 14 | every block must be rejected |
| `excluded/` | 185 | incomplete or schematic by design, or a tracked parser gap |

There are currently:

- **0 parser gaps**;
- **0 reference bugs** awaiting correction;
- **0 unclassified blocks**.

The excluded manifest contains only `by-design` entries such as
metasyntax, elided bodies, multi-file layouts, and members shown outside their
required container. Any future complete example that fails to parse must be
classified as a `gap` and fixed in the parser, or corrected in the normative
reference.

The parser tests enforce both executable corpora. The generator also rechecks
known classifications and automatically promotes an excluded block when it
becomes parseable.
