# Reference block conformance

This report is maintained alongside the generated corpora under
`frontend/testdata/refblocks/`. Regenerate the corpora from `frontend/` with:

```text
go run ./cmd/refblocks -write
```

## Current result

All 397 `folang` blocks in `docs/language-ref.md` are classified:

| Corpus | Document blocks | Contract |
|---|---:|---|
| `parsing/` | 223 | every block must parse |
| `invalid/` | 14 | every block must be rejected |
| `excluded/` | 160 | incomplete or schematic by design |

There are currently:

- **0 parser gaps**;
- **1 reference bug** awaiting correction;
- **0 unclassified blocks**.

The one reference bug is `L5836`, the map-entry evaluation-order example, which
writes an untyped braced map literal that both "Canonical Object and Collection
Construction" and Appendix A's normative grammar refuse. The correction it needs
is written out in [UNSORTED-TRIAGE.md](UNSORTED-TRIAGE.md); nothing in
`docs/language-ref.md` was changed to accommodate the parser.

Apart from it the excluded manifest contains only `by-design` entries such as
metasyntax, elided bodies, multi-file layouts, and members shown outside their
required container. Any future complete example that fails to parse must be
classified as a `gap` and fixed in the parser, or corrected in the normative
reference.

The parser tests enforce both executable corpora. The generator also rechecks
known classifications and automatically promotes an excluded block when it
becomes parseable.
