# Parser coverage

Snapshot generated from `docs/trace.json`, `docs/callgraph.json`, and
`docs/grammar-map.json`. Regenerate those inputs with:

```text
go run -tags partrace ./cmd/docgen
```

| Signal | Count |
|---|---:|
| Indexed parser functions | 253 |
| Indexed functions with a recorded snippet | 213 |
| Indexed functions with no recorded snippet | 40 |
| Grammar productions claimed by an `Implements:` comment | 286 |
| Productions marked `MISSING` | 133 |
| Functions marked `EXTRA` | 68 |

`trace.json` currently contains 222 function entries. Nine are instrumentation
entries outside the 253 functions indexed into `callgraph.json`, so the
coverage intersection is 213 rather than 222.

The authoritative production and function lists live in
`docs/grammar-map.json`; `docs/MISSING-BUCKETS.md` explains why unclaimed
productions can still have an implementation (for example scanner rules, Pratt
levels, and prose guards).

## Indexed functions without a recorded snippet

```text
parseAnnotationKeySegment
parseAnnotationStringOrArrowPair
parseAnonymousClassExpression
parseBlock
parseBlockExpression
parseBooleanToken
parseClassMembers
parseComponentBoundaryDeclaration
parseComponentDeclaration
parseComponentMember
parseComponentOperatorDeclaration
parseComponentSurfaceFile
parseComponentSurfaceMetadata
parseEffectHandledCallExpression
parseEmbeddedFieldDeclaration
parseExternVariableDeclaration
parseFoldedMatchChain
parseFunctionAliasBinding
parseImportField
parseImportStringField
parseLambdaExpression
parseLetValueDeclaration
parseLocalDeclarationName
parseLocalKindDeclaration
parseLockStatement
parseMatchDefault
parseObjectAssociationOptions
parseOperatorMetadataBody
parseOptionalLabelReference
parseOrdinaryAttributeValue
parseParentSelectorExpression
parsePredicateTypeDeclaration
parsePrefixRange
parseRelationshipSelectorExpression
parseReservedOperatorError
parseTrailingItems
parseTupleAssignmentTarget
parseUnitMemberName
parseUseField
parseUseMethodList
```
