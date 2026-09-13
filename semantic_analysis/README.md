# FoLang semantic analysis

This package is the post-parser frontend boundary. `Analyze` consumes an
`ast.ProjectStmt` and its canonical `FolangSymbols` graph; it does not tokenize,
reparse, or mutate parser-owned AST nodes.

The pipeline currently runs in this order:

1. Validate the context, symbol-table, and symbol graph.
2. Resolve ordinary lexical and qualified symbol occurrences. Overload families
   remain partially resolved for type-directed selection.
3. Infer primitive literal and resolved-symbol types and validate assignments
   when both sides have known types.
4. Perform callable-local reachability and definite-assignment analysis,
   intersecting assignment state across conditional branches.

Semantic facts are side tables keyed by source occurrence because the existing
AST deliberately uses value nodes. `Result` is therefore suitable as the input
to later lowering without changing the serialized parser artifact.

This is the executable foundation, not a claim that every advanced rule is
already implemented. Generic constraint solving, overload ranking, member
accessibility, typeclass-instance satisfaction, effects, and static
refinement/dependent proof obligations belong in additional passes in this
folder. Unknown types are preserved after an earlier error so those passes can
continue and produce useful diagnostics without panicking.
