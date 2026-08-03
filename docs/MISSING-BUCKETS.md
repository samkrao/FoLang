# MISSING production buckets

Every production that `docs/grammar-map.json` reported as MISSING, bucketed into
exactly one category. 182 productions at the time of the analysis.

Parser behaviour is unchanged: the only edits this produced were
`Implements:` doc comments on the `renamed` bucket.

| Bucket | Count | Meaning |
|---|---:|---|
| `lexical` | 55 | handled by the scanner; no parse function expected |
| `inlined` | 42 | a token check or short loop inside its caller |
| `renamed` | 41 | implemented by a function with a different name |
| `pratt-collapsed` | 35 | subsumed by the Pratt expression layer |
| `zero-width-guard` | 9 | implemented as an inline predicate |
| `genuinely-missing` | 0 | no implementation exists |

## genuinely-missing — none

No production in the MISSING list lacks an implementation. Each one is a
scanner token rule, a Pratt precedence layer, a zero-width predicate, a check
inlined into its caller, or a function under another name. MISSING measured the
absence of an `Implements:` comment, not the absence of code.

The closest thing to a gap is `informative-pipeline-chain`. Its five siblings
(`informative-condition-chain`, `-loop-chain`, `-mixed-chain`, `-ternary-chain`,
`-each-chain`, `-contains-chain`) each have a lowering pass that recognises the
canonical shape after parsing; the pipeline chain has none. That costs nothing
syntactically — section 11a marks these productions informative and states they
must not be a second parse path, so all of them parse as ordinary postfix
chains either way — but the pipeline shape is not narrowed into a dedicated
node the way the others are.

## renamed

Implemented by a function whose name is not the camelCase form of the
production. Each was verified against the production before annotation.

23 are `parse*` methods on `*parser` and now carry an `Implements:` comment,
which moved them out of MISSING — `grammar-map.json` went from 152 mapped /
182 missing to 175 mapped / 159 missing.

The remaining 18 also carry the comment but are **not** indexed by
`cmd/docgen`, whose `isParseFunc` filter accepts only `parse*` methods on
`*parser`. Twelve belong to `operatorSourceParser`, the separate grammar root
for `operators.fol`; the rest are `finish*`/`try*` continuation helpers and one
package-level function. They stay MISSING until that filter widens.

| Production | Implementing function |
|---|---|
| `address-type-specification` | parseAddressSpecification (annotated) |
| `annotation-arrow-pair` | parseAnnotationStringOrArrowPair (annotated) |
| `assignment-expression` | finishAssignment |
| `binding-pattern` | parseNamePattern (annotated) |
| `block-tail-expression` | tryBlockTailExpression |
| `boolean-literal` | parseBuiltinConstant (annotated) |
| `builtin-literal` | parseLiteral (annotated) |
| `cstruct-declaration` | parseCStructDeclaration (annotated) |
| `floating-literal` | parseNumericLiteral (annotated) |
| `function-definition` | finishFunctionDefinition |
| `grouped-expression` | parseGroupedOrTupleExpression (annotated) |
| `integer-literal` | parseNumericLiteral (annotated) |
| `match-case-body` | parseMatchCase (annotated) |
| `member-suffix` | parseMemberOrMatchSuffix (annotated) |
| `none-literal` | parseBuiltinConstant (annotated) |
| `operator-arity` | operatorSourceParser method (dedicated operator-source parser) |
| `operator-associativity` | operatorSourceParser method (dedicated operator-source parser) |
| `operator-body` | operatorSourceParser method (dedicated operator-source parser) |
| `operator-declaration` | operatorSourceParser method (dedicated operator-source parser) |
| `operator-fixity` | operatorSourceParser method (dedicated operator-source parser) |
| `operator-identity-value` | operatorSourceParser method (dedicated operator-source parser) |
| `operator-library-body` | operatorSourceParser method (dedicated operator-source parser) |
| `operator-library-declaration` | operatorSourceParser method (dedicated operator-source parser) |
| `operator-property` | operatorSourceParser method (dedicated operator-source parser) |
| `operator-source-file` | parseOperatorSource |
| `operator-symbol` | operatorSourceParser method (dedicated operator-source parser) |
| `operator-symbol-list` | operatorSourceParser method (dedicated operator-source parser) |
| `operator-symbol-reference` | operatorSourceParser method (dedicated operator-source parser) |
| `pattern-result` | finishFunctionPatternClause |
| `postfix-expression` | parsePostfix (annotated) |
| `primary-expression` | parsePrimary (annotated) |
| `qualified-function-reference` | parseDeclarationReference (annotated) |
| `range-expression` | finishRange |
| `range-type-specification` | parseRangeSpecification (annotated) |
| `result-binding` | parseSpecialBinding (annotated) |
| `self-binding` | parseSpecialBinding (annotated) |
| `slice-type-specification` | parseSliceSpecification (annotated) |
| `special-method` | parseLifecycleName (annotated) |
| `thunk-type-specification` | parseThunkSpecification (annotated) |
| `tuple-expression` | parseGroupedOrTupleExpression (annotated) |
| `where-clause` | parseOptionalWhereClause (annotated) |

## pratt-collapsed

Subsumed by the precedence-climbing expression layer: a binding-power level,
an operator table entry, or a postfix suffix loop rather than a function.

| Production | Where |
|---|---|
| `additive-expression` | Pratt binding-power layer, not a function |
| `additive-operator` | Pratt operator table |
| `bitwise-and-expression` | Pratt precedence table |
| `bitwise-or-expression` | Pratt binding-power layer, not a function |
| `bitwise-xor-expression` | Pratt precedence table |
| `compound-assignment-operator` | Pratt operator table |
| `equality-expression` | Pratt precedence table |
| `equality-operator` | Pratt operator table |
| `extended-operator-expression` | registered Pratt operator table |
| `informative-branch-verb` | member name tested by isBranchVerb during lowering |
| `informative-condition-chain` | parsed as an ordinary postfix chain; shape recognised later by lowerControlFlow |
| `informative-contains-chain` | parsed as an ordinary postfix chain; shape recognised later by lowerControlFlow |
| `informative-each-chain` | parsed as an ordinary postfix chain; shape recognised later by lowerControlFlow |
| `informative-loop-chain` | parsed as an ordinary postfix chain; shape recognised later by lowerControlFlow |
| `informative-mixed-chain` | parsed as an ordinary postfix chain; shape recognised later by lowerControlFlow |
| `informative-pipeline-chain` | parsed as an ordinary postfix chain; no lowering pass claims it |
| `informative-ternary-chain` | parsed as an ordinary postfix chain; shape recognised later by lowerControlFlow |
| `logical-and-expression` | Pratt binding-power layer, not a function |
| `logical-and-operator` | Pratt operator table |
| `logical-or-expression` | Pratt binding-power layer, not a function |
| `logical-or-operator` | Pratt operator table |
| `multi-symbol-relational-operator` | Pratt operator table |
| `multiplicative-expression` | Pratt precedence table |
| `multiplicative-operator` | Pratt operator table |
| `postfix-operator` | postfixOperators table consulted by parsePostfix |
| `postfix-suffix` | suffix loop inside parsePostfix |
| `power-expression` | Pratt binding-power layer, not a function |
| `power-operator` | Pratt operator table |
| `prefix-operator` | prefixOperators table consulted by the Pratt null denotation |
| `range-operator` | range spellings in the Pratt infix table |
| `relational-expression` | Pratt precedence table |
| `relational-operator` | Pratt operator table |
| `reserved-prefix-operator` | reservedOperators table; refused in operand position |
| `runtime-assignment-operator` | Pratt operator table |
| `unary-expression` | Pratt binding-power layer, not a function |

## zero-width-guard

Implemented as an inline predicate that consumes no token.

| Production | Where |
|---|---|
| `body-close` | brace plus body-closure-guard, applied by parseBracedBody |
| `body-closure-guard` | semicolon-rejecting predicate after a body-closing brace |
| `definition-operator` | inferred-declaration parser and operand-boundary guard |
| `multi-symbol-infix-operator-boundary-guard` | parser operand-boundary guard |
| `multi-symbol-range-operator-boundary-guard` | range parser operand-boundary guard |
| `non-anonymous-function-expression` | predicate selecting a direct anonymous-function body |
| `non-anonymous-function-expression-guard` | zero-width parser guard |
| `non-block-expression` | predicate selecting a body reading over an expression reading |
| `non-block-expression-guard` | zero-width parser guard |

## inlined

Consumed inside the caller rather than by a function of its own. The container
bodies dominate: each is a `parseBracedBody` call in its parent declaration,
parameterised by a member parser.

| Production | Where |
|---|---|
| `annotation-binder` | "=" or ":" accepted inside the annotation parsers |
| `annotation-map-entry` | entry loop inside parseAnnotationMap |
| `array-dimension` | branch inside parseArrayDimensionContent |
| `array-dimension-group` | bracket group loop inside parseArraySpecification |
| `block-item` | statement call inside parseBlock |
| `class-body` | parseBracedBody(parseClassMember) inside parseClassDeclaration |
| `contract-body` | parseBracedBody inside the annotated-contract parser |
| `cstruct-body` | parseBracedBody inside parseCStructDeclaration |
| `derivation-attribute` | key/value pair inside parseDerivationAttributeList |
| `dynamic-runtime-directive` | spelling branch inside parseFileDirective |
| `empty-statement` | bare ";" consumed by parseStatement |
| `entry-type-declaration` | entryFileDeclarationKinds gate in tryParseEntryDeclaration |
| `enum-body` | brace loop inside parseEnumDeclaration |
| `enum-separator` | "," consumed by the enum body loop |
| `forward-declarable-kind` | isForwardDeclarableKind set membership |
| `function-object-binding` | branch between parseFunctionObjectInlineBody and ...ExpressionBinding |
| `general-declarable-kind` | isGeneralDeclarableKind set membership |
| `general-kind-block` | parseBracedBody(parseGeneralKindMember) inside parseGeneralKindBinding |
| `generic-directive` | "@co.ddap." prefix branch inside parseFileDirective |
| `import-field` | assignImportField switch |
| `instance-body` | parseBracedBody inside parseInstanceDeclaration |
| `interface-body` | parseBracedBody inside parseInterfaceDeclaration |
| `library-body` | parseBracedBody(parseLibraryMember) inside parseLibraryDeclaration |
| `literal-pattern` | sign-and-literal branch inside parsePattern |
| `map-entry` | entry loop inside parseMapLiteral |
| `module-body` | parseBracedBody(parseModuleMember) inside parseModuleDeclaration |
| `object-body` | parseBracedBody inside parseObjectDeclaration |
| `object-field-initializer` | field loop inside parseObjectConstruction |
| `operator-library-marker` | fixed token sequence inside operatorSourceParser.parseFile |
| `parenthesized-type-list` | parenthesised list inside the arrow-tail and type-atom parsers |
| `pragma-directive` | spelling branch inside parseFileDirective |
| `record-pattern-field` | field loop inside parseRecordPattern |
| `return-item-list` | comma loop inside parseReturnTypeClause |
| `signature-body` | parseBracedBody(parseSignatureMember) inside parseSignatureDeclaration |
| `statement-end` | statementEnd helper called at each terminator site |
| `struct-body` | parseBracedBody(parseStructMember) inside parseStructDeclaration |
| `type-constructor-result-kind` | isTypeConstructorResultType set membership |
| `type-constructor-return-clause` | result-kind loop inside the type-constructor parser |
| `type-declaration-kind` | applyTypeDeclarationKind switch |
| `union-body` | parseBracedBody inside parseUnionDeclaration |
| `unit-body` | parseBracedBody inside parseUnitDeclaration |
| `use-field` | assignUseField switch |

## lexical

Recognised by the scanner. The parser sees one finished token, so no parse
function corresponds to these.

| Production | Where |
|---|---|
| `alpha-basic-c-character` | scanner literal token |
| `alpha-basic-s-character` | scanner literal token |
| `ascii-alphanumeric` | scanner identifier token |
| `ascii-letter` | scanner identifier token |
| `backslash` | scanner character class |
| `binary-digit` | scanner numeric token |
| `binary-digit-sequence` | scanner numeric token |
| `binary-exponent-part` | part of the scanner hexadecimal float token |
| `binary-integer-literal` | scanner numeric token |
| `block-comment` | scanner trivia |
| `block-comment-character` | scanner trivia |
| `contextual-keyword` | scanner token classification |
| `decimal-digit` | scanner numeric token |
| `decimal-digit-sequence` | scanner numeric token |
| `decimal-floating-literal` | scanner numeric token |
| `decimal-integer-literal` | scanner numeric token |
| `delimiter-token` | scanner delimiter token |
| `digit` | scanner numeric literal character |
| `double-quote` | scanner literal delimiter |
| `exponent-part` | part of the scanner numeric literal token |
| `floating-point-suffix` | scanned into the NUMBER token; stripped by trimFloatSuffix |
| `fractional-constant` | scanner numeric token |
| `hard-reserved-word` | scanner token classification |
| `hexadecimal-digit` | scanner numeric token |
| `hexadecimal-digit-sequence` | scanner numeric token |
| `hexadecimal-floating-literal` | scanner numeric token |
| `hexadecimal-fractional-constant` | scanner numeric token |
| `hexadecimal-integer-literal` | scanner numeric token |
| `hexadecimal-prefix` | scanner numeric token |
| `horizontal-white-space` | scanner trivia |
| `identifier-head` | scanner identifier token |
| `identifier-segment` | scanner identifier token |
| `identifier-trailing-guard` | scanner identifier token |
| `integer-suffix` | scanned into the NUMBER token; stripped by parseIntegerLexeme |
| `keyword-token` | scanner token classification |
| `line-break` | scanner trivia |
| `line-comment` | scanner trivia |
| `long-long-suffix` | scanner numeric token |
| `long-suffix` | scanner numeric token |
| `nonzero-digit` | scanner numeric token |
| `octal-digit` | scanner numeric token |
| `octal-digit-sequence` | scanner numeric token |
| `octal-integer-literal` | scanner numeric token |
| `pointer-stars` | whole symbolic run classified by isPointerStarRun |
| `reserved-future-operator` | scanner/reserved operator table |
| `reserved-operator` | scanner/reserved operator table |
| `sign` | scanner numeric literal character |
| `single-quote` | scanner literal delimiter |
| `size-suffix` | scanner numeric token |
| `string-literal` | scanner STRING token; parseStringLiteralSequence consumes a run of them |
| `symbolic-token` | scanner whole symbolic-run token |
| `token` | informative token-class summary |
| `token-separator` | scanner trivia boundary |
| `unsigned-suffix` | scanner numeric token |
| `white-space` | scanner trivia |
