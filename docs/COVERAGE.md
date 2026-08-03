# Parser coverage

Generated from `docs/trace.json` and `docs/grammar-map.json`. Do not hand-edit.

A report only: nothing here is fixed or worked around.

| Signal | Count | Question |
|---|---:|---|
| Functions with no recorded snippets | 55 | which parse functions did the corpus never exercise? |
| Productions marked MISSING | 142 | which grammar productions does no indexed function claim? |
| Functions marked EXTRA | 53 | which parse functions claim no production? |

Totals: `cmd/docgen` indexes 228 functions, 173 of which recorded a snippet; the
grammar defines 334 productions, 192 of them claimed by an `Implements:` comment.

docgen indexes `parse*`, `finish*` and `try*` methods on `*parser` and
`*operatorSourceParser`. The partrace instrumentation is narrower — `parse*` on
`*parser` only — so some indexed functions can never carry a snippet no matter
what the corpus contains. The two signals are not directly comparable.

Every MISSING production is bucketed by cause in `docs/MISSING-BUCKETS.md`;
none of them lacks an implementation.

## Functions with no recorded snippets

55 of the 228 indexed functions produced no span during the
`tests/parser/examples/accepted/` run that wrote `trace.json`.

The list does not distinguish why. A function lands here for any of:

- it is outside the instrumented set — every `*operatorSourceParser` method is
  here, because the operator-source grammar is parsed by its own reader before
  ordinary source and partrace does not instrument it;
- the corpus contains no source that reaches it — `parseImportDirective` is here
  because no accepted fixture uses `@co.ddap.import`;
- it is reachable only on a path the language rejects, so it never returns
  successfully — `parseLambdaExpression` handles a lambda outside a call
  argument, a rejected form (`standalone-lambda.fol`), while lambdas in argument
  position go through `parseDirectLambdaArgument`, which is traced;
- no valid source can reach it at all — `parseBooleanToken` is dispatched on the
  scanner's `BOOL` kind, but FoLang's booleans are `co.const.true`/`co.const.false`,
  which scan as `BUILT_IN_CONSTANTS` and route to `parseBuiltinConstant` instead
  (DECISION-LIT-005 makes bare `true`/`false` ordinary names, not literals);
- the instrumentation discarded every span it produced — an aborted parse, a
  rewound speculative parse, or a call that consumed no token.

Separating these requires reading each function; only the cases named above
were checked.

```
finishAssignment
finishFilenameDerivedName
finishFunctionDefinition
finishFunctionPatternClause
finishMatch
finishParenthesizedTypeAtom
finishRange
parseAliasDirective
parseAnnotatedContractDeclaration
parseAnonymousClassExpression
parseBareAttributeDerivation
parseBlockExpression
parseBooleanToken
parseBuiltinStatementExpression
parseClassMembers
parseCoPath
parseDeclaration
parseDeclarationReference
parseEmbeddedFieldDeclaration
parseFile
parseFoldedMatchChain
parseForwardTypeDeclaration
parseFunctionAliasBinding
parseFunctionObjectExpressionBinding
parseGeneralKindBinding
parseGeneralKindDeclaration
parseGeneralKindMember
parseGenericArityClause
parseGenericAritySlot
parseImportDirective
parseKindOptions
parseLambdaExpression
parseLetValueDeclaration
parseLibraryDeclaration
parseLibraryMember
parseLibrarySurfaceFile
parseLocalKindDeclaration
parseMatchDefault
parseNamedBlockDeclaration
parseOneOrMoreAnnotations
parseOperatorSymbolList
parseOptionalKindOptions
parsePackageAliasDeclaration
parseParenthesizedExpression
parsePrefixRange
parsePropertyValue
parseReservedOperatorError
parseTrailingItems
parseTupleAssignmentTarget
parseTypeAsExpression
tryBlockTailExpression
tryGeneralKindTypeBinding
tryParseEntryDeclaration
tryParsePrimaryDeclaration
tryTypeConstructorTypeBinding
```

## Productions marked MISSING

142 of 334 grammar productions are not claimed by a function `cmd/docgen`
indexes. docgen accepts only `parse*` methods on `*parser`, so a production
implemented by a scanner rule, a Pratt layer, a zero-width guard, a check
inlined into its caller, or a differently-shaped function stays here even when
its implementation exists and carries an `Implements:` comment.

`docs/MISSING-BUCKETS.md` sorts all of them by cause.

```
additive-expression
additive-operator
alpha-basic-c-character
alpha-basic-s-character
annotation-binder
annotation-map-entry
array-dimension
array-dimension-group
ascii-alphanumeric
ascii-letter
backslash
binary-digit
binary-digit-sequence
binary-exponent-part
binary-integer-literal
bitwise-and-expression
bitwise-or-expression
bitwise-xor-expression
block-comment
block-comment-character
block-item
body-close
body-closure-guard
class-body
compound-assignment-operator
contextual-keyword
contract-body
cstruct-body
decimal-digit
decimal-digit-sequence
decimal-floating-literal
decimal-integer-literal
definition-operator
delimiter-token
derivation-attribute
digit
double-quote
dynamic-runtime-directive
empty-statement
entry-type-declaration
enum-body
enum-separator
equality-expression
equality-operator
exponent-part
extended-operator-expression
floating-point-suffix
forward-declarable-kind
fractional-constant
function-object-binding
general-declarable-kind
general-kind-block
generic-directive
hard-reserved-word
hexadecimal-digit
hexadecimal-digit-sequence
hexadecimal-floating-literal
hexadecimal-fractional-constant
hexadecimal-integer-literal
hexadecimal-prefix
horizontal-white-space
identifier-head
identifier-segment
identifier-trailing-guard
import-field
informative-branch-verb
informative-condition-chain
informative-contains-chain
informative-each-chain
informative-loop-chain
informative-mixed-chain
informative-pipeline-chain
informative-ternary-chain
instance-body
integer-suffix
interface-body
keyword-token
library-body
line-break
line-comment
literal-pattern
logical-and-expression
logical-and-operator
logical-or-expression
logical-or-operator
long-long-suffix
long-suffix
map-entry
module-body
multi-symbol-infix-operator-boundary-guard
multi-symbol-range-operator-boundary-guard
multi-symbol-relational-operator
multiplicative-expression
multiplicative-operator
non-anonymous-function-expression
non-anonymous-function-expression-guard
non-block-expression
non-block-expression-guard
nonzero-digit
object-body
object-field-initializer
octal-digit
octal-digit-sequence
octal-integer-literal
operator-library-marker
operator-source-file
parenthesized-type-list
pointer-stars
postfix-operator
postfix-suffix
power-expression
power-operator
pragma-directive
prefix-operator
range-operator
record-pattern-field
relational-expression
relational-operator
reserved-future-operator
reserved-operator
reserved-prefix-operator
return-item-list
runtime-assignment-operator
sign
signature-body
single-quote
size-suffix
statement-end
string-literal
struct-body
symbolic-token
token
token-separator
type-constructor-result-kind
type-constructor-return-clause
type-declaration-kind
unary-expression
union-body
unit-body
unsigned-suffix
use-field
white-space
```

## Functions marked EXTRA

53 of 228 parse functions carry no `Implements:` annotation. Some are helpers
shared by several productions, some are dispatch or context-guard functions the
grammar does not name.

```
finishFilenameDerivedName
finishMatch
finishParenthesizedTypeAtom
parseAnnotationNameValue
parseArrowTypeResults
parseBareAttributeDerivation
parseBlockExpression
parseBooleanToken
parseBracedBody
parseBuiltinStatementExpression
parseCaseResult
parseClassMembers
parseControlStatement
parseDecoratedFunctionDeclaration
parseDirectLambdaArgument
parseEntryFunctionPatternClause
parseExpr
parseExprWithContext
parseExpressionQualifiedName
parseFoldedMatchChain
parseFunctionObjectExpressionBinding
parseFunctionObjectInlineBody
parseHeapReferenceSpecification
parseInfixRightOperand
parseInstanceMember
parseLambdaExpressionWithPermission
parseLocalKindDeclaration
parseMatchChain
parseMemberList
parseNameExpression
parseNamedTypeAtom
parseOptionalAttributeTail
parseOptionalGenericParameterClause
parseOptionalKindOptions
parseParameterLists
parseParenthesizedExpression
parseParenthesizedReturnList
parseParenthesizedTypeItems
parsePrefixRange
parseQualifiedNameWith
parseQualifiedTypeName
parseReservedOperatorError
parseSelfReference
parseTopLevel
parseTrailingItems
parseTypeAsExpression
parseUnary
parseVariableInitializer
parseWhereSuffix
tryGeneralKindTypeBinding
tryParseEntryDeclaration
tryParsePrimaryDeclaration
tryTypeConstructorTypeBinding
```
