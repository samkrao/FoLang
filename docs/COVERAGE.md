# Parser coverage

Generated from `docs/trace.json` and `docs/grammar-map.json`. Do not hand-edit.

A report only: nothing here is fixed or worked around.

Three independent signals, each answering a different question.

| Signal | Count | Question |
|---|---:|---|
| Functions with no recorded snippets | 39 | which parse functions did the corpus never exercise? |
| Productions marked MISSING | 182 | which grammar productions does no function claim? |
| Functions marked EXTRA | 60 | which parse functions claim no production? |

Totals for context: 212 parse functions are instrumented, 173 of them recorded at
least one snippet, the grammar defines 334 productions, and 152 of those are
claimed by an `Implements:` doc comment.

## Functions with no recorded snippets

39 of 212 instrumented parse functions produced no span during the
`tests/parser/examples/accepted/` run that wrote `trace.json`.

The list does not distinguish why. A function lands here for any of:

- the corpus contains no source that reaches it — `parseImportDirective` is here
  because no accepted fixture uses `@co.ddap.import`;
- it is reachable only on a path the language rejects, so it never returns
  successfully — `parseLambdaExpression` handles a lambda outside a call
  argument, which is a rejected form (`standalone-lambda.fol`), while lambdas in
  argument position go through `parseDirectLambdaArgument`, which is traced;
- no valid source can reach it at all — `parseBooleanToken` is dispatched on the
  scanner's `BOOL` kind, but FoLang's booleans are `co.const.true`/`co.const.false`,
  which scan as `BUILT_IN_CONSTANTS` and route to `parseBuiltinConstant` instead
  (DECISION-LIT-005 makes bare `true`/`false` ordinary names, not literals);
- the instrumentation discarded every span it produced — an aborted parse, a
  rewound speculative parse, or a call that consumed no token.

Separating these requires reading each function; only the three named above
were checked.

```
parseAliasDirective
parseAnnotatedContractDeclaration
parseAnonymousClassExpression
parseBareAttributeDerivation
parseBlockExpression
parseBooleanToken
parseBuiltinStatementExpression
parseClassMembers
parseCoPath
parseDeclarationReference
parseEmbeddedFieldDeclaration
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
parseOptionalKindOptions
parsePackageAliasDeclaration
parseParenthesizedExpression
parsePrefixRange
parseReservedOperatorError
parseTrailingItems
parseTupleAssignmentTarget
parseTypeAsExpression
```

## Productions marked MISSING

182 of 334 grammar productions have no function claiming them with an
`Implements: <production>` doc comment.

The 152 claimed productions are those whose implementing function is named for
them, verified individually against the production before the comment was
added. What remains falls into three groups, which this list does not separate:
productions implemented below recursive-descent level (scanner token rules,
Pratt precedence layers, zero-width guards), productions whose implementing
function carries a different name, and productions with no implementation.

```
additive-expression
additive-operator
address-type-specification
alpha-basic-c-character
alpha-basic-s-character
annotation-arrow-pair
annotation-binder
annotation-map-entry
array-dimension
array-dimension-group
ascii-alphanumeric
ascii-letter
assignment-expression
backslash
binary-digit
binary-digit-sequence
binary-exponent-part
binary-integer-literal
binding-pattern
bitwise-and-expression
bitwise-or-expression
bitwise-xor-expression
block-comment
block-comment-character
block-item
block-tail-expression
body-close
body-closure-guard
boolean-literal
builtin-literal
class-body
compound-assignment-operator
contextual-keyword
contract-body
cstruct-body
cstruct-declaration
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
floating-literal
floating-point-suffix
forward-declarable-kind
fractional-constant
function-definition
function-object-binding
general-declarable-kind
general-kind-block
generic-directive
grouped-expression
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
integer-literal
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
match-case-body
member-suffix
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
none-literal
nonzero-digit
object-body
object-field-initializer
octal-digit
octal-digit-sequence
octal-integer-literal
operator-arity
operator-associativity
operator-body
operator-declaration
operator-fixity
operator-identity-value
operator-library-body
operator-library-declaration
operator-library-marker
operator-property
operator-source-file
operator-symbol
operator-symbol-list
operator-symbol-reference
parenthesized-type-list
pattern-result
pointer-stars
postfix-expression
postfix-operator
postfix-suffix
power-expression
power-operator
pragma-directive
prefix-operator
primary-expression
qualified-function-reference
range-expression
range-operator
range-type-specification
record-pattern-field
relational-expression
relational-operator
reserved-future-operator
reserved-operator
reserved-prefix-operator
result-binding
return-item-list
runtime-assignment-operator
self-binding
sign
signature-body
single-quote
size-suffix
slice-type-specification
special-method
statement-end
string-literal
struct-body
symbolic-token
thunk-type-specification
token
token-separator
tuple-expression
type-constructor-result-kind
type-constructor-return-clause
type-declaration-kind
unary-expression
union-body
unit-body
unsigned-suffix
use-field
where-clause
white-space
```

## Functions marked EXTRA

60 of 212 parse functions carry no `Implements:` annotation, because their name
is not the camelCase form of a production. Some implement a production under
another name, some are helpers shared by several productions, and some are
dispatch or context-guard functions the grammar does not name.

```
parseAddressSpecification
parseAnnotationNameValue
parseAnnotationStringOrArrowPair
parseArrowTypeResults
parseBareAttributeDerivation
parseBlockExpression
parseBooleanToken
parseBracedBody
parseBuiltinConstant
parseBuiltinStatementExpression
parseCStructDeclaration
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
parseGroupedOrTupleExpression
parseHeapReferenceSpecification
parseInfixRightOperand
parseInstanceMember
parseLambdaExpressionWithPermission
parseLocalKindDeclaration
parseMatchChain
parseMemberList
parseMemberOrMatchSuffix
parseNameExpression
parseNamePattern
parseNamedTypeAtom
parseNumericLiteral
parseOptionalAttributeTail
parseOptionalGenericParameterClause
parseOptionalKindOptions
parseOptionalWhereClause
parseParameterLists
parseParenthesizedExpression
parseParenthesizedReturnList
parseParenthesizedTypeItems
parsePostfix
parsePrefixRange
parsePrimary
parseQualifiedNameWith
parseQualifiedTypeName
parseRangeSpecification
parseReservedOperatorError
parseSelfReference
parseSliceSpecification
parseThunkSpecification
parseTopLevel
parseTrailingItems
parseTypeAsExpression
parseUnary
parseVariableInitializer
parseWhereSuffix
```
