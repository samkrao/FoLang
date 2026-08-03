# FoLang parser

## Spec — normative, do not modify to match code
- Grammar: folang-r22.ebnf
- Decisions: grammar-decisions-r24.md
- Reference: language-ref.md

If the code disagrees with the grammar, the code is wrong.

## Testing
- `go test ./...` after every change
## Corpora
Paths are relative to `frontend/`.
- tests/parser/examples/accepted/  curated, all must parse
- testdata/refblocks/parsing/      doc blocks that parse
- testdata/refblocks/invalid/      doc blocks that must be rejected
- testdata/refblocks/excluded/     prose fragments, not parseable by design
                                   (MANIFEST.tsv gives a reason per file;
                                   rows marked `gap` are docs/REFBLOCK-GAPS.md)

## Generated files — never hand-edit
docs/trace.json, docs/callgraph.json, docs/grammar-map.json
Regenerate, from `frontend/`: `go run -tags partrace ./cmd/docgen`

trace.json needs the tag on docgen itself, because the spans come from the
parser instrumentation docgen links in. Without it docgen still writes
callgraph.json and grammar-map.json and leaves trace.json alone.

Snippets and call edges in docs/api/ come from those files only.