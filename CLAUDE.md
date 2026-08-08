# FoLang parser

## Spec — normative, do not modify to match code
- Grammar: docs/grammar/folang.ebnf
- Decisions: docs/grammar/grammar-decisions.md
- Validation report: docs/grammar/grammar-validation.json
- Reference: docs/language-ref.md

If the code disagrees with the grammar, the code is wrong.

## Testing
- `go test ./...` after every change
## Corpora
Paths are relative to `frontend/`.
- tests/parser/examples/accepted/  curated, all must parse
- tests/parser/examples/rejected/  curated, all must be rejected
- testdata/refblocks/parsing/      doc blocks that parse
- testdata/refblocks/invalid/      doc blocks that must be rejected
- testdata/refblocks/excluded/     not parseable as written, or a tracked gap
                                   (MANIFEST.tsv gives a reason per block;
                                   rows marked `gap` are docs/REFBLOCK-GAPS.md,
                                   rows marked `unsorted` need classifying)

### refblocks — generated, do not hand-edit
Extracted from every ```folang block in docs/language-ref.md.
Regenerate, from `frontend/`:
    go run ./cmd/refblocks          report what would change
    go run ./cmd/refblocks -write   rewrite the corpora

Each block is a folder `L<line>/` named after the line it opens on, holding one
file under the name the reference gives it (`// Employee.fol`). The folder
exists so a block can keep a reserved exact name such as `package.fol` or
`operators.fol`; FoLang classifies a source file BY ITS NAME, so parsing a
block under a synthesized name misclassifies it.

Classification is carried across a re-extraction by block CONTENT, not by
filename, because editing the reference renumbers every block below the edit.
A block that neither parses nor matches a known one is reported for a person to
classify rather than guessed into a bucket.

Hand-written fixtures for rules no reference block covers sit flat beside the
`L<line>/` folders and are left untouched by regeneration.

## Generated files — never hand-edit
docs/trace.json, docs/callgraph.json, docs/grammar-map.json
Regenerate, from `frontend/`: `go run -tags partrace ./cmd/docgen`

trace.json needs the tag on docgen itself, because the spans come from the
parser instrumentation docgen links in. Without it docgen still writes
callgraph.json and grammar-map.json and leaves trace.json alone.

Snippets and call edges in docs/api/ come from those files only.