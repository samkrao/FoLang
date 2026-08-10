# Parser conformance audit

This audit compares the normative `docs/language-ref.md`, the consolidated
`docs/grammar/folang.ebnf`, and the parser under `frontend/src/parser`.

## Current status

As of 2026-08-10, the parser has no known syntax-conformance gaps.

- The grammar contains **352 productions**.
- Every production has an implementation trace or an explicit scanner, Pratt,
  filename-classification, contextual-guard, informative-shape, or operator-source
  classification.
- All curated accepted compilation units parse to a non-dummy AST.
- All curated rejected compilation units produce a syntax diagnostic.
- All **205** complete normative reference blocks parse.
- All **18** normative invalid examples are rejected.
- The remaining **109** reference blocks are incomplete or schematic by design;
  the excluded manifest contains no `gap`, `ref-bug`, or `unsorted` entries.

## Fixed during this audit

The expression/type ambiguity in `co.lang.tag(co.lang.string, "Hello")` was
resolved. In expression position, a built-in type followed by `(` is now parsed
as a call target; in type position, the same shape remains a type application.
A dedicated accepted fixture protects this distinction.

Reference examples were also corrected to follow the grammar's mandatory
statement terminators, self-delimiting directive syntax, filename rules,
unit-member declaration forms, and prohibition on nested named kind declarations.
The reference-block corpora were regenerated after those corrections.

## Evidence and limits

The following tests form the executable syntax audit:

```text
go test ./...
go test ./tests/parser -count=1 -run "Test(EBNFConformance|GrammarProductionsHaveImplementationTrace|RefBlocks)"
```

`TestGrammarProductionsHaveImplementationTrace` is an inventory guard, not by
itself behavioral proof. Behavioral evidence comes from complete accepted and
rejected compilation units and the extracted normative reference blocks.

This audit covers parsing and parser-owned structural validation. Type
equivalence, definite initialization, capture resolution, import graphs, and
other semantic requirements require their respective later-phase audits.
