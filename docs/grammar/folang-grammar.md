# FoLang — Complete EBNF Grammar

Derived from `language-ref.md` (the normative FoLang language reference). This
grammar aims for full syntactic coverage of every construct shown in the
reference. Constructs the reference itself marks **(planned)** / **not
finalized** / **conceptual stage** are included but flagged, since they are
part of the document's surface syntax even though semantics are not final.

---

## 0. Notation

Standard EBNF (ISO/W3C style):

```
::=      "is defined as"
|        alternative
[ x ]    x is optional (0 or 1)
{ x }    x repeated zero or more times
( x )    grouping
'x'      literal terminal (exact text)
"x"      literal terminal (exact text)
A - B    everything A allows, minus B
?x?      informal / prose-constrained token (defined by comment, not by
         further production) — used sparingly for things the reference
         does not formally lexicalize (e.g. arbitrary Unicode math glyphs)
```

Whitespace and comments are insignificant except as token separators, and are
defined in §1. The grammar is presented top-down: compilation units first,
then declarations, then statements/expressions, then the lexical grammar.

---

## 1. Compilation Units

FoLang source files carry the extension `.fol`. Three source-file shapes
exist, each with its own grammar entry point, plus the special
`package.fol` renaming file.

```ebnf
CompilationUnit
    ::= ApplicationEntryFile
      | PackageSourceFile
      | LibrarySurfaceFile
      | PackageAliasFile

(* --- 1.1 Application entry file / single-source application file --- *)
(* A single-source application file and an application entry file share *)
(* this grammar, context, and restrictions.                            *)
ApplicationEntryFile
    ::= { EntryFileItem }

EntryFileItem
    ::= EntryDirective
      | ImportDirective
      | AliasDirective
      | UseDirective
      | EntryLocalTypeAliasDecl
      | EntryLocalNewTypeDecl
      | EntryLocalOpaqueTypeDecl
      | DependentTypeAliasDecl
      | DependentTypeUsageStmt
      | SubtypeDecl
      | SupertypeDecl
      | BareFunctionPatternGroup
      | LetFunctionPatternGroup
      | VariableDeclStmt
      | Statement
      | Expression ';'

(* Forbidden in the entry file (documented negative constraints, not     *)
(* separate productions): ordinary `let` value bindings, ordinary named  *)
(* function declarations, anonymous functions used as first-class        *)
(* values/general closures, currying, classes, structs, cstructs, enums, *)
(* unions, type constructors, generics, macros, templates, units, and    *)
(* other reusable behavioral declarations.                               *)

(* --- 1.2 Package source file --- *)
(* Exactly one primary top-level declaration per file, optionally        *)
(* preceded by import/alias/annotation directives.                       *)
PackageSourceFile
    ::= { FileLevelDirective } PrimaryDeclaration

FileLevelDirective
    ::= ImportDirective
      | AliasDirective
      | UseDirective
      | PragmaDirective
      | Annotation                      (* annotation attached to the   *)
                                         (* following primary decl       *)

PrimaryDeclaration
    ::= ClassDecl
      | StructDecl
      | CStructDecl
      | EnumDecl
      | UnionDecl
      | InterfaceDecl
      | SignatureDecl
      | ModuleDecl
      | UnitDecl
      | TypeClassDecl
      | TypeClassInstanceDecl
      | TypeConstructorDecl
      | MatcherDecl
      | MacroDecl
      | TemplateDecl
      | AnnotationObjectDecl
      | DecoratorDecl
      | TypeAliasDecl
      | NewTypeDecl
      | OpaqueTypeDecl
      | SubtypeDecl
      | SupertypeDecl
      | ForwardOrExternDecl
      | LibrarySurfaceUnitDecl          (* @co.dap.library-annotated `unit`
                                            when this file *is* a library
                                            surface file, see §1.3 *)

(* --- 1.3 Library surface file --- *)
LibrarySurfaceFile
    ::= { FileLevelDirective }
        LibraryAnnotation
        LibrarySurfaceUnitDecl

LibraryAnnotation
    ::= '@' 'co' '.' 'dap' '.' 'library' '(' 'type' '=' StringLiteral ')'

LibrarySurfaceUnitDecl
    ::= DeclName KindKeyword('co.lang.library') '=' '{' { SurfaceMember } '}'

SurfaceMember
    ::= BoundaryStructDecl
      | BoundaryCStructDecl
      | BoundaryAdapterFunctionDecl

(* --- 1.4 package.fol — package aliasing file (Planned, not finalized) --- *)
PackageAliasFile
    ::= DeclName KindKeyword('co.lang.package') ';'
```

---

## 2. Directives, Annotations, Decorators, Pragmas

FoLang classifies all `@`-prefixed forms into four families, distinguished by
their namespace root:

```ebnf
AtForm
    ::= Pragma | Directive | Annotation | Decorator

Pragma      ::= '@' 'co' '.' 'pdap' '.' Identifier [ AnnotationArgList ]
Directive   ::= '@' 'co' '.' 'ddap' '.' Identifier [ AnnotationArgList ]
Annotation  ::= '@' 'co' '.' 'dap'  '.' Identifier [ AnnotationArgList ]
Decorator   ::= '@' 'co' '.' 'dap'  '.' Identifier [ AnnotationArgList ]
              | '@' 'co' '.' 'fx'   '.' Identifier [ AnnotationArgList ]

LifecycleAtForm
    ::= '@@' Identifier                (* e.g. @@new, @@init — restricted,
                                           compiler-owned lifecycle symbols *)

AnnotationArgList
    ::= '(' [ AnnotationArg { ',' AnnotationArg } [ ',' ] ] ')'

AnnotationArg
    ::= Identifier '=' AnnotationValue     (* named argument, e.g. kind=Functor *)
      | AnnotationValue                    (* positional / bare value      *)

AnnotationValue
    ::= Literal
      | QualifiedName
      | TypeExpr
      | ListLiteral
      | MapLiteral
      | AnnotationObjectLiteral
      | FunctionSignatureRef
      | AtForm                             (* nested annotation value      *)

AnnotationObjectLiteral
    ::= '{' [ AnnotationField { ',' AnnotationField } [ ',' ] ] '}'

AnnotationField
    ::= Identifier ':' AnnotationValue
      | Identifier                          (* bare flag key *)

(* A stack of one or more annotations/decorators/pragmas may precede any  *)
(* declaration; order between annotation kinds is not semantically fixed *)
(* by the reference except where noted (e.g. @co.dap.public together     *)
(* with @co.dap.local — see §5.9).                                       *)
AnnotationStack
    ::= { AtForm }
```

Recognized directive/annotation identifiers (closed, built-in — see the
informative appendix in §12 for the full published list):

```
PRAGMA:      co.pdap.compiler, co.pdap.scale
DIRECTIVE:   co.ddap.movetotop, co.ddap.import, co.ddap.dynamicruntime,
             co.ddap.use, co.ddap.alias
ANNOTATION:  co.dap.template, co.dap.macro, co.dap.operator, co.dap.annotation,
             co.dap.library, co.dap.module, co.dap.pragma, co.dap.directive,
             co.dap.native, co.dap.class, co.dap.static, co.dap.instance,
             co.dap.object, co.dap.inline, co.dap.ctfe, co.dap.friend,
             co.dap.sealed, co.dap.extension, co.dap.override, co.dap.virtual,
             co.dap.abstract, co.dap.delegate, co.dap.dynamicscope,
             co.dap.lexicalscope, co.dap.staticscope, co.dap.mixedscope,
             co.dap.typeclass, co.dap.matcher, co.dap.constructor, co.dap.oops,
             co.dap.hokrt, co.dap.hokrlt, co.dap.indexer, co.dap.generic,
             co.dap.comptime, co.dap.typefromvalue, co.dap.local,
             co.dap.private, co.dap.public, co.dap.package, co.dap.protected,
             co.dap.internal, co.dap.export, co.dap.eager, co.dap.lazy,
             co.dap.packed, co.dap.declare, co.dap.simd, co.dap.reflection,
             co.dap.mop, co.dap.nested, co.dap.inner, co.dap.method.class,
             co.dap.constructor
DECORATOR:   co.dap.before, co.dap.after, co.dap.around, co.fx.onErrExcept,
             co.fx.InvokeAlways, co.fx.HandleEffect, co.dap.callback,
             co.dap.defer, co.dap.continuation, co.dap.event, co.dap.scale,
             co.dap.distributed, co.dap.concurrent, co.dap.parallel,
             co.dap.subroutine, co.dap.generator, co.dap.goroutine,
             co.dap.coroutine, co.dap.async, co.dap.promise, co.dap.future,
             co.dap.thread, co.dap.task, co.dap.fiber, co.dap.process,
             co.dap.spawn, co.dap.exec, co.dap.fork, co.dap.csp, co.dap.actor,
             co.dap.synthetic, co.dap.bridge, co.dap.greenlet, co.dap.channel,
             co.dap.callable, co.dap.iterator
```

> Note: `@co.dap.declare(extern)` / `@co.dap.declare(forward)` and other
> user-authored pragmas/directives/annotations/decorators are explicitly
> disallowed — only the closed built-in set above may be produced; the
> reference states "Directives and Pragmas are not allowed to create as
> they are language internals."

---

## 3. Imports and Aliases

```ebnf
ImportDirective
    ::= '@' 'co' '.' 'ddap' '.' 'import' '(' ImportArgList ')'

ImportArgList
    ::= ImportArg { ',' ImportArg }

ImportArg
    ::= 'package'      '=' StringLiteral
      | 'library'      '=' StringLiteral
      | 'src-library'  '=' BooleanLiteral
      | 'expect'       '=' StringLiteral
      | 'as'           '=' StringLiteral
      | 'realm'        '=' StringLiteral
      | 'parent-realm' '=' StringLiteral

AliasDirective
    ::= '@' 'co' '.' 'ddap' '.' 'alias' '(' CoPath ',' 'as' '=' StringLiteral ')'

UseDirective
    ::= '@' 'co' '.' 'ddap' '.' 'use' '('
            'from' '=' StringLiteral ','
            'extensions' '=' '[' Identifier { ',' Identifier } ']'
        ')'

DynamicRuntimeDirective
    ::= '@' 'co' '.' 'ddap' '.' 'dynamicruntime'   (* entry file only *)

MoveToTopDirective
    ::= '@' 'co' '.' 'ddap' '.' 'movetotop'

CoPath
    ::= 'co' { '.' Identifier }            (* e.g. co.out, co.core.list *)
```

---

## 4. Types

```ebnf
TypeExpr
    ::= QualifiedName [ GenericArgList ]           (* e.g. co.lang.int, List(T) *)
      | PointerTypeExpr
      | ArrayTypeExpr
      | ReferenceTypeExpr
      | RangeTypeExpr
      | SliceTypeExpr
      | FunctionTypeExpr
      | ForallTypeExpr
      | UnionTypeExpr
      | TupleTypeGroupExpr
      | DependentTypeApplication
      | AnonymousStructuralType

GenericArgList
    ::= '(' TypeExpr { ',' TypeExpr } ')'

(* --- 4.1 Fat / plain pointers, references, addresses, thunks, slices --- *)
PointerTypeExpr
    ::= BaseType '->' '(' PointerSuffix { ',' PointerAttr } ')'

PointerSuffix
    ::= '*'                     (* pointer            *)
      | '**'                    (* double pointer     *)
      | '&'                     (* reference          *)
      | '&&'                    (* lvalue reference    *)
      | '~'                     (* heap ref            *)
      | '@'                     (* address             *)
      | '^'                     (* thunk               *)
      | 'kind' '=' PointerKind  (* fat-pointer kind form *)

PointerKind
    ::= 'thin' | 'slice' | 'relative' | 'trait' | 'buffer' | 'view'
      | 'opaque' | 'custom' | 'mem' | 'nullptr' | 'sptr' | 'uptr'
      | 'ptrdiff' | 'usize' | 'ssize' | Identifier  (* region sugar *)

PointerAttr
    ::= Identifier '=' AnnotationValue    (* kind=, meta={...}, repr=,
                                              sign=, region=, len=, cap=,
                                              vtab=, bits=, endian= ... *)

ArrayTypeExpr
    ::= BaseType '->' '(' ArrayDims ')'

ArrayDims
    ::= '[' [ ArrayDim { ',' ArrayDim } ] ']'      (* single/multi-dim,
                                                        '[N]','[N,M]','[]' *)
      | '[' ArrayDim ']' { '[' ArrayDim ']' }        (* jagged: [2][3]   *)
      | '[' '...' ']'                                (* variable length  *)
      | '[' '.' ']'                                  (* zero-dimension   *)

ArrayDim
    ::= IntegerLiteral | Identifier | (* empty *)

ReferenceTypeExpr
    ::= PointerTypeExpr                    (* &, &&, ~, @ share this form *)

RangeTypeExpr
    ::= BaseType '->' '(' '..' ')'

SliceTypeExpr
    ::= BaseType '->' '(' '[' ':' ']' ')'

FunctionTypeExpr
    ::= '(' [ TypeExpr { ',' TypeExpr } ] ')' '->' '(' [ TypeExpr { ',' TypeExpr } ] ')'
      | '(' [ TypeExpr { ',' TypeExpr } ] ')' '->' TypeExpr

ForallTypeExpr
    ::= 'forall' '(' Identifier { ',' Identifier } ')' '.' TypeExpr
        (* type-expression form ONLY — valid at Rank-2/3 parameter and
           return positions, and inside `co.lang.type` alias right-hand
           sides. `forall` at declaration head position is a compiler
           error; see §5.5. *)

UnionTypeExpr
    ::= TypeExpr '|' TypeExpr { '|' TypeExpr }     (* ADT / tagged union
                                                        type expression *)

TupleTypeGroupExpr
    ::= '(' TypeExpr { ',' TypeExpr } ')'          (* comma/grouping form *)

DependentTypeApplication
    ::= QualifiedName '(' DependentArg { ',' DependentArg } ')'
        (* e.g. Vector(3), Vector(n), Matrix(r, c), Stack(10, co.lang.int) *)

DependentArg
    ::= Expression | TypeExpr

AnonymousStructuralType
    ::= KindKeyword('co.lang.class') [ '(' GenericFieldList ')' ] '{' { StructMember } '}'
      | KindKeyword('co.lang.struct') '{' { StructMember } '}'

BaseType
    ::= TypeExpr

KindKeyword(K)
    ::= (* literal token spelled exactly as K, e.g. `co.lang.struct` *)
```

---

## 5. Declarations

### 5.1 Variable Declarations

```ebnf
VariableDeclStmt
    ::= VariableDecl { ',' VariableDecl } ';'
      | '(' VariableDecl { ',' VariableDecl } ')' ';'   (* grouping form *)

VariableDecl
    ::= SimpleVarDecl
      | InferredVarDecl
      | ReassignOrDefineDecl
      | PointerVarDecl
      | ArrayVarDecl
      | ReferenceVarDecl
      | RangeVarDecl
      | AutoVarDecl
      | DynamicVarDecl
      | LazyVarDecl
      | BindVarDecl
      | DiscardVarDecl

SimpleVarDecl
    ::= DeclTargetName TypeExpr [ '=' Expression ]

InferredVarDecl
    ::= DeclTargetName ':=' Expression      (* error if name already defined *)

ReassignOrDefineDecl
    ::= DeclTargetName '?=' Expression      (* define+init if absent,
                                                 else plain reassignment  *)

PointerVarDecl
    ::= DeclTargetName PointerTypeExpr [ '=' Expression ]

ArrayVarDecl
    ::= DeclTargetName ArrayTypeExpr [ '=' ArrayLiteral ]

ReferenceVarDecl
    ::= DeclTargetName ReferenceTypeExpr [ '=' Expression ]

RangeVarDecl
    ::= DeclTargetName RangeTypeExpr [ '=' RangeExpr ]

AutoVarDecl
    ::= DeclTargetName KindKeyword('co.lang.auto') '=' Expression   (* init required *)

DynamicVarDecl
    ::= DeclTargetName KindKeyword('co.lang.dynamic') [ '=' Expression ]

LazyVarDecl
    ::= '@' 'co' '.' 'dap' '.' 'lazy'
        DeclTargetName [ TypeExpr ] '=' Expression

BindVarDecl
    ::= BindVariable                        (* $0, $1, $2, ... — token *)

DiscardVarDecl
    ::= '_'                                  (* wildcard; outside pattern
            matching/contains/iterators, `_` must be suffixed by a letter
            or digit, e.g. `_x`, `_1` *)

DeclTargetName
    ::= Identifier | '_'
```

### 5.2 Type Declarations (aliases, new types, opaque types, ADTs, subtype/supertype)

```ebnf
TypeAliasDecl
    ::= DeclName KindKeyword('co.lang.type') '=' TypeExpr

NewTypeDecl
    ::= DeclName KindKeyword('co.lang.newtype') '=' TypeExpr

OpaqueTypeDecl
    ::= DeclName KindKeyword('co.lang.opaquetype') '=' TypeExpr

AlgebraicDataTypeDecl
    ::= DeclName [ '(' GenericParamList ')' ] KindKeyword('co.lang.data')
        '=' AdtVariantList

AdtVariantList
    ::= AdtVariant { '|' AdtVariant }

AdtVariant
    ::= Identifier [ '(' [ TypeExpr { ',' TypeExpr } ] ')' ]

SubtypeDecl
    ::= DeclName KindKeyword('co.lang.subtype') '=' TypeExpr

SupertypeDecl
    ::= DeclName KindKeyword('co.lang.supertype') '=' TypeExpr

EntryLocalTypeAliasDecl   ::= TypeAliasDecl
EntryLocalNewTypeDecl     ::= NewTypeDecl
EntryLocalOpaqueTypeDecl  ::= OpaqueTypeDecl

(* --- Dependent type constructors: functions returning types --- *)
TypeConstructorDecl
    ::= DeclName '(' DependentParamList ')' '->' '(' DependentReturnType ')'
        '=' TypeExpr

DependentParamList
    ::= DependentParam { ',' DependentParam }

DependentParam
    ::= Identifier TypeExpr                 (* value parameter, e.g. n co.lang.int *)
      | Identifier KindKeyword('co.lang.type')  (* type parameter *)

DependentReturnType
    ::= KindKeyword('co.lang.dependentType')
      | KindKeyword('co.lang.type')

DependentTypeAliasDecl  ::= TypeAliasDecl
DependentTypeUsageStmt  ::= DependentTypeApplication ';'

(* --- Compile-time / built-in type computation forms --- *)
CompileTimeTypeFunctionDecl
    ::= AnnotationStack FunctionDecl     (* carries @co.dap.comptime and/or
                                             @co.dap.typefromvalue *)

DeclTypeExpr
    ::= 'co' '.' 'hokrlt' '.' 'type' '.' 'decltype' '(' Expression ')'
```

### 5.3 User-Defined Data Types (UDTs)

```ebnf
(* --- Struct --- *)
StructDecl
    ::= AnnotationStack DeclName [ '(' GenericParamList ')' ]
        KindKeyword('co.lang.struct') '=' '{' { StructMember } '}'
      | AnnotationStack DeclName KindKeyword('co.lang.struct') ';'  (* forward/extern *)

BoundaryStructDecl ::= StructDecl        (* library-surface struct contract *)

StructMember
    ::= FieldDecl
      | EmbeddedFieldDecl

FieldDecl
    ::= Identifier TypeExpr ';'

EmbeddedFieldDecl
    ::= TypeExpr ';'                      (* bare type reference = embedding *)

(* --- CStruct --- *)
CStructDecl
    ::= AnnotationStack DeclName KindKeyword('co.lang.cstruct')
        '=' '{' { CStructMember } '}'

BoundaryCStructDecl ::= CStructDecl

CStructMember
    ::= Identifier TypeExpr ';'           (* primitives, fixed arrays, or
                                              nested cstructs only *)

(* --- Enum --- *)
EnumDecl
    ::= AnnotationStack DeclName KindKeyword('co.lang.enum')
        '=' '{' EnumVariantList '}'

EnumVariantList
    ::= EnumVariant { ',' EnumVariant } [ ',' ]

EnumVariant
    ::= Identifier

(* --- Union (untagged ADT) --- *)
UnionDecl
    ::= AnnotationStack DeclName KindKeyword('co.lang.union')
        '=' '{' { UnionMember } '}'

UnionMember
    ::= Identifier TypeExpr ';'

(* --- Class --- *)
ClassDecl
    ::= AnnotationStack DeclName [ '(' GenericParamList ')' ]
        KindKeyword('co.lang.class') [ '->' '(' ClassAttrList ')' ]
        '=' '{' { ClassMember } '}'

ClassAttrList
    ::= AnnotationArg { ',' AnnotationArg }   (* e.g. implements=[...] *)

ClassMember
    ::= FieldDecl
      | LifecycleMethodDecl
      | MethodDecl
      | AssignedMethodDecl
      | DelegatedMethodDecl

LifecycleMethodDecl
    ::= AnnotationStack LifecycleAtForm '(' [ ParamList ] ')'
        [ '->' '(' [ TypeExpr { ',' TypeExpr } ] ')' ] '=' Block

MethodDecl
    ::= AnnotationStack Identifier '(' [ ParamList ] ')'
        '->' '(' [ TypeExpr { ',' TypeExpr } ] ')' '=' Block

(* method assigned directly from another callable — "assigning module
   function to class's method" *)
AssignedMethodDecl
    ::= AnnotationStack Identifier '(' [ ParamList ] ')'
        '->' '(' TypeExpr ')' '=' QualifiedName

(* method delegating/chaining via =>> to other calls, with $1,$2,... bind
   variables referring to previous call results *)
DelegatedMethodDecl
    ::= AnnotationStack Identifier '(' [ ParamList ] ')'
        '->' '(' [ TypeExpr { ',' TypeExpr } ] ')'
        '=>>' CallExpr { '=>>' CallExpr } ';'

(* --- Interface --- *)
InterfaceDecl
    ::= DeclName KindKeyword('co.lang.interface') '=' '{' { InterfaceMember } '}'

InterfaceMember
    ::= Identifier '(' [ ParamList ] ')'
        '->' '(' [ TypeExpr { ',' TypeExpr } ] ')' ';'

(* --- Signature (module contract) --- *)
SignatureDecl
    ::= DeclName KindKeyword('co.lang.signature') '=' '{' { SignatureMember } '}'

SignatureMember
    ::= InterfaceMember                     (* function requirement *)
      | Identifier TypeExpr ';'             (* required value        *)
      | Identifier '(' [ GenericParamList ] ')' KindKeyword('co.lang.type') ';'
                                             (* abstract/fixed type-
                                                component requirement  *)

(* --- Module --- *)
ModuleDecl
    ::= AnnotationStack DeclName
        KindKeyword('co.lang.module') '->' '(' ModuleAttrList ')'
        '=' '{' { ModuleMember } '}'

ModuleAttrList
    ::= 'signature' '=' QualifiedName [ ',' 'matches' '=' QualifiedName ]

ModuleMember
    ::= FieldDecl
      | MethodDecl

ModuleBindingDecl
    ::= Identifier QualifiedName '=' QualifiedName ';'    (* mm EmployeeModule = EmployeeModImpl; *)

(* --- Unit (named, non-instantiable function container) --- *)
UnitDecl
    ::= AnnotationStack DeclName KindKeyword('co.lang.unit')
        '=' '{' { UnitMember } '}'

UnitMember
    ::= FunctionDecl                        (* standalone-unit function *)
      | CompanionFunctionDecl                (* companion-unit associated fn *)
      | OperatorFunctionDecl
      | IndexerFunctionDecl
```

### 5.4 Companion-Unit / Associated / Operator / Indexer Functions

```ebnf
CompanionFunctionDecl
    ::= AnnotationStack
        [ '(' Identifier TypeExpr ')' ]        (* optional explicit receiver
                                                    (recv StructType) form  *)
        Identifier '(' [ ParamList ] ')'
        '->' '(' [ TypeExpr { ',' TypeExpr } ] ')' '=' Block
        (* When no explicit receiver clause is used, the first declared
           ordinary parameter's type must match the companion struct. *)

OperatorFunctionDecl
    ::= '@' 'co' '.' 'dap' '.' 'operator' '(' OperatorAttrList ')'
        Identifier '(' ParamList ')'
        '->' '(' TypeExpr ')' '=' Block

OperatorAttrList
    ::= OperatorAttr { ',' OperatorAttr }

OperatorAttr
    ::= 'symbol' '=' CharOrStringLiteral
      | 'mode' '=' ( 'overload' | 'define' )     (* mode=override: rejected *)
      | 'fixity' '=' Fixity
      | 'precedence' '=' IntegerLiteral
      | 'associativity' '=' ( 'left' | 'right' | 'none' )
      | 'arity' '=' ( 'unary' | 'binary' | 'ternary' )
      | 'commutative' '=' BooleanLiteral
      | 'idempotent' '=' BooleanLiteral
      | 'identity' '=' Literal
      | 'foldable' '=' BooleanLiteral
      | 'vectorizable' '=' BooleanLiteral
      | 'distributes_over' '=' ListLiteral
      | 'desugar' '=' StringLiteral

Fixity
    ::= 'infix' | 'postfix' | 'prefix' | 'circumfix' | 'postcircumfix'
      | 'prescircumfix' | 'mixfix' | 'ternary' | 'distfix'

IndexerFunctionDecl
    ::= '@' 'co' '.' 'dap' '.' 'indexer' '(' 'symbol' '=' StringLiteral ')'
        '(' Identifier TypeExpr ')' Identifier '(' [ ParamList ] ')'
        '->' '(' [ TypeExpr { ',' TypeExpr } ] ')' '=' Block
```

### 5.5 Functions (free/unit functions, forms, patterns, closures)

```ebnf
FunctionDecl
    ::= AnnotationStack FunctionHeader FunctionBodyOrAssign

FunctionHeader
    ::= Identifier ParamClauseList '->' '(' [ ReturnList ] ')'

ParamClauseList
    ::= '(' [ ParamList ] ')' { '(' [ ParamList ] ')' }
        (* one clause = normal function; 2+ clauses = curried function,
           e.g. add(first co.lang.int)(second co.lang.int) *)

ParamList
    ::= Param { ',' Param }

Param
    ::= [ '...' ] [ '~' ] Identifier [ '?' ] TypeExpr [ '=' Expression ]
        (* '...' prefix  -> variadic (only on the last declared param);
           '~'   prefix  -> named parameter;
           '?'   suffix on name -> optional parameter, tested via
                 `paramName.omitted` inside the body;
           trailing '= Expression' -> default parameter value.
           Curried and variadic are mutually exclusive on one function. *)

ReturnList
    ::= ReturnItem { ',' ReturnItem }

ReturnItem
    ::= TypeExpr
      | Identifier TypeExpr               (* named return, e.g. r co.lang.int *)

FunctionBodyOrAssign
    ::= '=' Block
      | '=' Expression ';'                (* single-expression body *)
      | '=>>' DelegateChain ';'            (* chained delegation, uses
                                                $1,$2,... bind vars *)
      | ';'                                (* forward declaration *)

DelegateChain
    ::= CallExpr { '=>>' CallExpr }

(* --- Anonymous functions / function literals --- *)
AnonymousFunctionExpr
    ::= '(' [ AnonParamList ] ')' '->' '(' [ TypeExpr { ',' TypeExpr } ] ')'
        Block
      | '(' [ AnonParamList ] ')' '->' '(' [ TypeExpr { ',' TypeExpr } ] ')'
        '=' Block

AnonParamList
    ::= AnonParam { ',' AnonParam }

AnonParam
    ::= Identifier TypeExpr

ImmediatelyInvokedFunctionExpr
    ::= AnonymousFunctionExpr '(' [ ArgList ] ')'

(* --- Lambda: restricted, callback-argument-only inline syntax --- *)
LambdaExpr
    ::= '|' [ LambdaParamList ] '|' '=>' Expression
      | '|' [ LambdaParamList ] '|' '=>' Block

LambdaParamList
    ::= Identifier { ',' Identifier }
        (* Legal only as a direct inline argument to a collection
           operation such as map/filter/reduce/forEach/sortBy/groupBy;
           elsewhere `|...|` is a compile-time/lint error. *)

(* --- Curried lambda / closure sugar --- *)
CurriedFunctionExpr
    ::= Identifier '(' Param ')' '=>' Expression      (* closure(factor int) => (x int) = x*factor -- see below *)

ArrowClosureDecl
    ::= Identifier '(' ParamList ')' '=>' '(' ParamList ')' '=' Expression
      | Identifier ParamClauseList '=' Expression      (* curry(f)(v) = f*v *)

FunctionObjectDecl
    ::= Identifier KindKeyword('co.lang.function') '=' AnonymousFunctionExpr
      | Identifier KindKeyword('co.lang.function') '=' QualifiedName

FunctionDelegateDecl
    ::= '@' 'co' '.' 'dap' '.' 'delegate'
        Identifier KindKeyword('co.lang.delegate') '=' FunctionTypeExpr ';'

(* --- Inner function (nested inside another function body) --- *)
InnerFunctionDecl ::= FunctionDecl              (* appears as a Statement
                                                     inside a Block; captures
                                                     enclosing lexical scope *)

(* --- Function-pattern groups --- *)
BareFunctionPatternGroup
    ::= FunctionPatternClause { FunctionPatternClause }
        (* all clauses share Identifier + arity; no runtime capture *)

LetFunctionPatternGroup
    ::= LetFunctionPatternClause { LetFunctionPatternClause }
        (* at least one surrounding runtime binding must be captured *)

FunctionPatternClause
    ::= Identifier '(' [ PatternParamList ] ')' [ WhereGuard ] '=>' PatternClauseBody

LetFunctionPatternClause
    ::= 'let' Identifier '(' [ PatternParamList ] ')' [ WhereGuard ] '=' PatternClauseBody

PatternParamList
    ::= Pattern { ',' Pattern }

WhereGuard
    ::= '.' 'where' '(' Expression ')'

PatternClauseBody
    ::= Block
      | Expression

(* --- Ordinary `let` value bindings (NOT permitted in entry file) --- *)
LetValueBindingExpr
    ::= 'let' '(' '{' LetBindingList '}' ')' '.' 'in' '(' '{' Expression '}' ')'

LetBindingList
    ::= LetBinding { ',' LetBinding }

LetBinding
    ::= ( Identifier | '$' ) '=' Expression

WhereBindingExpr
    ::= '(' Expression ')' '.' 'where' '(' LetBindingList ')'
```

### 5.6 Generics, `forall`, Templates, Type Classes, Matchers

```ebnf
GenericAnnotation
    ::= '@' 'co' '.' 'dap' '.' 'generic' '(' GenericAnnotationArgs ')'

GenericAnnotationArgs
    ::= GenericAnnotationArg { ',' GenericAnnotationArg }

GenericAnnotationArg
    ::= 'at' '=' ( 'runtime' | 'compiletime' )
      | 'refied' '=' BooleanLiteral
      | 'where' '=' ( 'usesite' | 'callsite' )
      | 'typename' '=' Identifier
      | 'type' '=' '{' TypeParamSpec { ',' TypeParamSpec } '}'

TypeParamSpec
    ::= Identifier ':' '{' TypeParamAttr { ',' TypeParamAttr } '}'
      | Identifier ':' '{' 'typename' '}'

TypeParamAttr
    ::= 'variance' ':' ( 'covariant' | 'invariant' | 'contravariant' )
      | 'bound' '=' TypeExpr
      | 'kind' ':' ( 'param' | 'result' | 'var' | 'arg' )
      | 'default' '=' TypeExpr
      | 'nullable' ':' BooleanLiteral
      | 'inclusive' ':' BooleanLiteral
      | 'impredicative' ':' BooleanLiteral
      | 'typekind' ':' ( 'type' | 'class' | 'function' | 'module' | 'unit' | 'package' )
      | 'types' '=' ListLiteral

GenericParamList
    ::= GenericParam { ',' GenericParam }

GenericParam
    ::= Identifier                          (* type parameter, e.g. T *)
      | Identifier TypeExpr                 (* value/kind parameter    *)

GenericFieldList ::= GenericParamList

(* forall — type-expression-only construct; see ForallTypeExpr in §4. *)
(* `forall(T) name ...` at declaration head position is *invalid*.     *)

TemplateDecl
    ::= '@' 'co' '.' 'dap' '.' 'template' FunctionDecl
        (* typed template: ordinary typed params;
           untyped template: params without types, return type
           co.lang.untyped *)

TypeClassDecl
    ::= '@' 'co' '.' 'dap' '.' 'typeclass' '(' 'kind' '=' TypeClassKind ')'
        DeclName '(' GenericParamList ')' '=' '{' { TypeClassMember } '}'

TypeClassKind
    ::= 'Functor' | 'Applicative' | 'Monad' | 'Monoid' | 'Transformer'
      | Identifier                           (* user-defined kind *)

TypeClassMember
    ::= Identifier '(' [ ParamList ] ')' '->' '(' TypeExpr ')' ';'

TypeClassInstanceDecl
    ::= DeclName KindKeyword('co.lang.instance')
        '->' '(' InstanceAttrList ')' '=' '{' { InstanceMember } '}'

InstanceAttrList
    ::= 'for' '=' QualifiedName
        ( ',' 'type' '=' QualifiedName
        | ',' 'types' '=' ListLiteral )

InstanceMember
    ::= Identifier '(' [ ParamList ] ')' '->' '(' TypeExpr ')' '=' Block

MatcherDecl
    ::= '@' 'co' '.' 'dap' '.' 'matcher'
        DeclName '(' GenericParamList ')' '=' '{' MatcherContract '}'
      | DeclName KindKeyword('co.lang.Matcher') '->' '(' MatcherAttrList ')'
        '=' '{' MatcherImpl '}'

MatcherContract
    ::= Identifier '(' ParamList ')' '->' '(' TypeExpr ',' TypeExpr ')' ';'

MatcherAttrList
    ::= 'for' '=' QualifiedName ',' 'type' '=' TypeExpr

MatcherImpl
    ::= Identifier '(' ParamList ')' '->' '(' TypeExpr ',' TypeExpr ')' '=' Block
```

### 5.7 Macros

```ebnf
MacroDecl
    ::= '@' 'co' '.' 'dap' '.' 'macro' [ '(' MacroAttrList ')' ]
        Identifier '(' [ MacroParamList ] ')' '->' '(' [ TypeExpr ] ')' '=' Block

MacroParamList
    ::= MacroParam { ',' MacroParam }

MacroParam
    ::= Identifier [ TypeExpr ]             (* untyped macro params allowed *)

MacroAttrList
    ::= MacroAttr { ',' MacroAttr }

MacroAttr
    ::= 'group' '=' AnnotationObjectLiteral
      | 'sugarform' '=' AnnotationObjectLiteral
      | 'bind' '=' AnnotationObjectLiteral
      | 'isolate' '=' AnnotationObjectLiteral
      | 'gensym' '=' AnnotationObjectLiteral
      | 'hygienic' '=' BooleanLiteral
      | 'argtransform' '=' AnnotationObjectLiteral
      | 'desugar' '=' '{' 'exprs' ':' '[' DesugarRule { ',' DesugarRule } ']' '}'
      | 'mode' '=' StringLiteral
      | 'chainswith' '=' AnnotationObjectLiteral
      | 'standalone' '=' BooleanLiteral

DesugarRule
    ::= StringLiteral '=>' StringLiteral

QuasiquoteExpr
    ::= 'co' '.' 'macro' '.' 'quote' '(' Block ')'
      | 'co' '.' 'macro' '.' 'unquote' '(' Expression ')'

EscapeAssignStmt
    ::= 'co' '.' 'macro' '.' 'esc' '(' Identifier ')' '=' Expression

GensymExpr
    ::= 'co' '.' 'macro' '.' 'gensym' '(' QualifiedName ',' StringLiteral ')'

MacroUtilityAnnotation
    ::= '@' 'co' '.' 'dap' '.' 'compose' '(' 'using' '=' ListLiteral ')'
      | '@' 'co' '.' 'dap' '.' 'guard' '(' 'expr' '=' StringLiteral ')'
```

### 5.8 Extensions

```ebnf
ExtensionUnitDecl
    ::= DeclName KindKeyword('co.lang.unit') '=' '{' { ExtensionMember } '}'

ExtensionMember
    ::= '@' 'co' '.' 'dap' '.' 'extension' '(' ExtensionAttrList ')' FunctionDecl

ExtensionAttrList
    ::= 'fortype' '=' ( TypeExpr | ListLiteral ) ',' 'what' '=' ( 'extends' | 'overrides' )
```

### 5.9 Local / Nested Declarations

```ebnf
LocalAnnotation
    ::= '@' 'co' '.' 'dap' '.' 'local' '(' 'for' '=' LocalTargetSet ')'

LocalTargetSet
    ::= DeclarationReference
      | '[' DeclarationReference { ',' DeclarationReference } ']'

DeclarationReference
    ::= QualifiedName                                    (* class/struct/
                                                              enum/module   *)
      | QualifiedName '(' [ TypeExpr { ',' TypeExpr } ] ')'
        '->' '(' [ TypeExpr { ',' TypeExpr } ] ')'        (* function
                                                              overload ref  *)

NestedAnnotation
    ::= '@' 'co' '.' 'dap' '.' 'nested' '(' 'target' '=' QualifiedName ')'

(* Both annotations decorate any PrimaryDeclaration eligible under §5.9's
   supported-kind rule (class, struct, enum, module, or a named function
   declared in a normally function-permitting context). *)
```

### 5.10 Forward / Extern Declarations

```ebnf
ForwardOrExternDecl
    ::= ExternVariableDecl
      | ForwardFunctionDecl
      | ExternTypeDecl

ExternVariableDecl
    ::= '@' 'co' '.' 'dap' '.' 'declare' '(' 'extern' ')'
        Identifier TypeExpr ';'

ForwardFunctionDecl
    ::= [ '@' 'co' '.' 'dap' '.' 'declare' '(' 'forward' ')' ]
        FunctionHeader ';'

ExternTypeDecl
    ::= [ '@' 'co' '.' 'dap' '.' 'declare' '(' 'extern' ')' ]
        DeclName ( KindKeyword('co.lang.struct') | KindKeyword('co.lang.class')
                 | QualifiedName ) ';'
```

### 5.11 Reflection

```ebnf
ReflectionEnableAnnotation
    ::= '@' 'co' '.' 'dap' '.' 'reflection' '(' 'enable' '=' BooleanLiteral
        ',' 'package' '=' StringLiteral ')'

ReflectExpr
    ::= Expression '.' 'reflect' '(' ')' '.' ReflectMethod '(' ')'

ReflectMethod ::= 'getType' | 'getValue' | 'getKind'
```

### 5.12 Annotations / Decorator Declarations (user-visible, non-built-in objects)

```ebnf
AnnotationObjectDecl
    ::= DeclName KindKeyword('co.lang.object') '->' '(' 'for' '=' 'annotation' ')'
        '=' '{' { FieldDecl } '}'

DecoratorDecl
    ::= '@' 'co' '.' 'dap' '.' 'decorator'
        Identifier '(' 'target' TypeExpr ')' '->' '(' TypeExpr ')' '=' Block
```

### 5.13 Labels and Named Blocks

```ebnf
LabelStmt
    ::= Identifier ':' Block

Block
    ::= '{' { Statement } '}'

NamedBlockDecl
    ::= DeclName KindKeyword('co.lang.block') '=' Block

NamedBlockExpandExpr
    ::= Identifier '.' 'expand' '(' ')'
```

---

## 6. Statements

```ebnf
Statement
    ::= VariableDeclStmt
      | TypeAliasDecl ';'
      | NewTypeDecl ';'
      | OpaqueTypeDecl ';'
      | SubtypeDecl ';'
      | SupertypeDecl ';'
      | InnerFunctionDecl
      | BareFunctionPatternGroup
      | LabelStmt
      | Block
      | ExpressionStmt
      | AssignmentStmt
      | ReturnStmt

ExpressionStmt
    ::= Expression ';'

ReturnStmt
    ::= ( 'this' | 'self' ) '.' 'return' [ ExpressionList ] ';'

ExpressionList
    ::= Expression { ',' Expression }
```

FoLang's statement taxonomy, as named in the reference, groups into:
Declaration Statement, Initialization Statement, Expression Statement,
Conditional Statement, Loop Statement (and, per the reference, "etc." —
the list is explicitly non-exhaustive). Conditional and loop "statements"
are expression-shaped (`.do`/`.loop`/`.otherwise` chains — see §7.6) rather
than dedicated keyword statements, since FoLang has no `if`/`else`/`for`/
`while`/`foreach` keywords.

---

## 7. Expressions

### 7.1 Grammar

```ebnf
Expression
    ::= AssignmentExpr

AssignmentExpr
    ::= TargetList AssignOp RhsList
      | ConditionalOrExpr

TargetList
    ::= LValue { ',' LValue }

RhsList
    ::= Expression { ',' Expression }        (* multiple assignment *)

AssignOp
    ::= '=' | ':=' | '?=' | '::='
      | '+=' | '-=' | '*=' | '/=' | '%=' | '**='                (* compound *)

LValue
    ::= Identifier
      | LValue '.' Identifier
      | LValue '[' Expression ']'
      | DeclTargetName TypeExpr              (* declaring assignment target *)

ConditionalOrExpr  ::= LogicalOrExpr
LogicalOrExpr       ::= LogicalAndExpr  { '||' LogicalAndExpr }
LogicalAndExpr       ::= BitOrExpr       { '&&' BitOrExpr }
BitOrExpr             ::= BitAndExpr      { '|' BitAndExpr }
BitAndExpr             ::= EqualityExpr    { '&' EqualityExpr }
EqualityExpr            ::= RelationalExpr  { ( '==' | '!=' ) RelationalExpr }
RelationalExpr           ::= RangeExpr       { ( '<' | '>' | '<=' | '>=' ) RangeExpr }
RangeExpr                 ::= AdditiveExpr   [ RangeOp [ AdditiveExpr ] ]
RangeOp   ::= '..' | '<..' | '..<' | '<..<'
AdditiveExpr        ::= MultiplicativeExpr { ( '+' | '-' ) MultiplicativeExpr }
MultiplicativeExpr    ::= PowerExpr           { ( '*' | '/' | '%' ) PowerExpr }
PowerExpr               ::= UnaryExpr          { '**' UnaryExpr }
UnaryExpr
    ::= ( '!' | '-' | '+' | '~' | '++' | '--' ) UnaryExpr
      | PostfixExpr

PostfixExpr
    ::= PrimaryExpr { PostfixOp }

PostfixOp
    ::= '.' Identifier                        (* member access            *)
      | '.' Identifier '(' [ ArgList ] ')'     (* method call              *)
      | '(' [ ArgList ] ')'                    (* function/curried call    *)
      | '[' Expression ']'                     (* indexing                 *)
      | '++' | '--'                            (* postfix incr/decr        *)
      | '!'                                    (* postfix unary operator,
                                                    e.g. factorial-style,
                                                    or error-unwrap style   *)

PrimaryExpr
    ::= Literal
      | QualifiedName
      | BindVariable
      | '_'
      | '(' Expression ')'
      | '(' ExpressionList ')'                 (* tuple / grouping *)
      | AnonymousFunctionExpr
      | LambdaExpr
      | CollectionLiteral
      | AnonymousStructuralType
      | LiteralExpressionObjectExpr
      | LetValueBindingExpr
      | WhereBindingExpr
      | ConditionChainExpr
      | LoopChainExpr
      | TernaryChainExpr
      | MatchChainExpr
      | ComprehensionExpr
      | QuasiquoteExpr
      | ReflectExpr
      | NamedBlockExpandExpr
      | ForallTypeExpr

CallExpr ::= PostfixExpr                       (* any postfix expr ending
                                                    in a call *)

ArgList
    ::= Arg { ',' Arg }

Arg
    ::= Expression
      | '~' Identifier ':' Expression          (* named argument at call site *)
```

### 7.2 Literals and collection literals

```ebnf
Literal
    ::= IntegerLiteral | FloatLiteral | StringLiteral | CharLiteral
      | BooleanLiteral | NoneLiteral

BooleanLiteral ::= 'co' '.' 'const' '.' 'true' | 'co' '.' 'const' '.' 'false'
NoneLiteral    ::= 'co' '.' 'const' '.' 'none'

CollectionLiteral
    ::= ArrayLiteral | MapLiteral | ListLiteral | SetLiteral | StructLiteral

ArrayLiteral
    ::= '[' [ Expression { ',' Expression } ] ']'
      | '[' ArrayLiteral { ',' ArrayLiteral } ']'      (* nested/multi-dim *)

ListLiteral ::= ArrayLiteral

SetLiteral  ::= '{' [ Expression { ',' Expression } ] '}'

MapLiteral
    ::= '{' [ MapEntry { ',' MapEntry } ] '}'

MapEntry
    ::= Expression ':' Expression

StructLiteral
    ::= QualifiedName '{' [ StructFieldInit { ',' StructFieldInit } ] '}'

StructFieldInit
    ::= Identifier ':' Expression

(* "Literal expression objects" per the philosophy section: source-level
   literals with no handle, immutable by nature, deeply so. They use the
   ordinary literal / collection-literal grammar above; there is no
   distinct surface syntax beyond it. *)
LiteralExpressionObjectExpr ::= Literal | CollectionLiteral

RangeExprStandalone
    ::= [ AdditiveExpr ] RangeOp [ AdditiveExpr ]
        (* both bounds omittable independently: ..100 / 1.. / 1..10 *)
```

### 7.3 Function / method calls, member access, indexing — see §7.1 `PostfixExpr`.

### 7.4 Pattern matching expression

```ebnf
MatchChainExpr
    ::= Expression '.' 'match' [ '(' MatchModeOrMatcher ')' ]
        MatchCaseChain

MatchModeOrMatcher
    ::= 'co' '.' 'pattern' '.' MatchMode
      | QualifiedName                          (* custom matcher type *)

MatchMode
    ::= 'Type' | 'Value' | 'Instance' | 'Object' | 'Shape' | 'Any'

MatchCaseChain
    ::= { '.' 'case' '(' MatchCase ')' } [ '.' 'default' '(' Expression ')' ]

MatchCase
    ::= Pattern [ WhereGuard ] '=>' MatchResult
      | Pattern ':' Expression '=>' MatchResult   (* `n: n > 10 => ...` form *)

MatchResult
    ::= Expression
      | Block
```

### 7.5 Patterns

```ebnf
Pattern
    ::= LiteralPattern
      | WildcardPattern
      | BindingPattern
      | ConstructorPattern
      | TuplePattern
      | TypePattern
      | GuardedPattern

LiteralPattern     ::= Literal
WildcardPattern     ::= '_'
BindingPattern       ::= Identifier
ConstructorPattern    ::= QualifiedName [ '(' [ Pattern { ',' Pattern } ] ')' ]
TuplePattern           ::= '(' Pattern { ',' Pattern } ')'
TypePattern             ::= TypeExpr
GuardedPattern           ::= Pattern WhereGuard
```

### 7.6 Conditions, loops, ternary (all associated-function chains — no keywords)

```ebnf
ConditionChainExpr
    ::= '(' Expression ')' '.' 'do' '(' Block ')'
        { '.' 'otherwise' '(' Expression ')' '.' 'do' '(' Block ')' }
        [ '.' 'otherwise' '.' 'do' '(' Block ')' ]

LoopChainExpr
    ::= '(' Expression ')' '.' 'loop' '(' Block ')'
        { '.' 'otherwise' '(' Expression ')' '.' 'loop' '(' Block ')' }
        [ '.' 'otherwise' '.' 'loop' '(' Block ')' ]

(* Conditions and loops may freely mix within one otherwise-chain: *)
ConditionOrLoopChainExpr
    ::= '(' Expression ')' '.' ( 'do' | 'loop' ) '(' Block ')'
        { '.' 'otherwise' '(' Expression ')' '.' ( 'do' | 'loop' ) '(' Block ')' }
        [ '.' 'otherwise' '.' ( 'do' | 'loop' ) '(' Block ')' ]

TernaryChainExpr
    ::= '(' Expression ')' '.' 'return' '(' Expression ')'
        { '.' 'otherwise' '(' Expression ')' '.' 'return' '(' Expression ')' }
        '.' 'otherwise' '.' 'return' '(' Expression ')'

ContainsExpr
    ::= Expression '.' 'contains' '(' Expression ')'
        '.' 'do' '(' Block ')' [ '.' 'otherwise' '.' 'do' '(' Block ')' ]

EachExpr
    ::= Expression '.' 'each' '(' ( Identifier | '_' ) ',' Identifier ')'
        '.' 'do' '(' Block ')'
```

### 7.7 Comprehensions *(planned)*

```ebnf
ComprehensionExpr
    ::= PipelineComprehension
      | ForYieldComprehension

PipelineComprehension
    ::= Expression '.' 'filter' '(' LambdaExpr ')'
      | Expression '.' 'map' '(' LambdaExpr ')'
      | PipelineComprehension '.' ( 'filter' | 'map' ) '(' LambdaExpr ')'

ForYieldComprehension
    ::= 'for' '(' ComprehensionBinding ')' '.' 'yield' '(' ExpressionList ')'

ComprehensionBinding
    ::= Pattern '<-' Expression
      | '(' Pattern { ',' Pattern } ')' '<-' Expression
```

### 7.8 Lazy expressions

```ebnf
LazyExpr ::= LazyVarDecl                        (* @co.dap.lazy binding, §5.1 *)
```

### 7.9 Concurrency / execution-model expressions (library type = advanced)

```ebnf
ConcurrencyAnnotatedFunction
    ::= AnnotationStack FunctionDecl
        (* AnnotationStack may include @co.dap.thread, @co.dap.task,
           @co.dap.process, @co.dap.continuation, @co.dap.async,
           @co.dap.goroutine, @co.dap.coroutine, @co.dap.actor, etc. *)

SubmitExpr
    ::= 'co' '.' 'cpca' '.' ( 'submitToPool' | 'submitThread'
                             | 'submitToEventLoop' ) '(' ArgList ')'

SendReceiveExpr
    ::= Expression '.' ( 'send' | 'receive' ) '(' [ ArgList ] ')'
```

### 7.10 Operator precedence table (highest to lowest binding)

```
1.  Postfix:      . () [] ++ --  !(postfix)
2.  Prefix unary: ! - + ~ ++ -- (prefix)
3.  Power:        **
4.  Multiplicative: * / %
5.  Additive:     + -
6.  Range:        .. <.. ..< <..<
7.  Relational:   < > <= >=
8.  Equality:     == !=
9.  Bitwise AND:  &
10. Bitwise OR:   |
11. Logical AND:  &&
12. Logical OR:   ||
13. Assignment:   = := ?= ::= += -= *= /= %= **=
```

Method-chain forms (`.do`, `.otherwise`, `.loop`, `.return`, `.match`,
`.case`, `.default`, `.each`, `.contains`, `.filter`, `.map`, `.reduce`,
`.where`) are ordinary postfix method calls and therefore bind at postfix
level (§7.1), left-associatively, chained left to right — the precedence
table above governs only the classic operator symbols.

---

## 8. Function objects, closures, currying — recap of surface forms

```ebnf
ClosureSyntaxForms
    ::= Identifier '(' ParamList ')' '=' Block                     (* ordinary named fn *)
      | Identifier '(' Param ')' '(' Param ')' '->' '(' TypeExpr ')' '=' Block   (* curried *)
      | Identifier '(' ParamList ')' '->' '(' TypeExpr ')' '=' AnonymousFunctionExpr
      | Identifier '(' Param ')' '=>' '(' Param ')' '=' Expression  (* closure(...) => (...) = ... *)
      | Identifier '(' Param ')' '(' Param ')' '=' Expression       (* curry(a)(b) = expr *)
```

---

## 9. Package, Import, and Access Declarations (structural, non-syntactic-file rules folded into grammar)

```ebnf
AccessAnnotation
    ::= '@' 'co' '.' 'dap' '.' ( 'public' | 'package' | 'protected' | 'private' )

MethodKindAnnotation
    ::= '@' 'co' '.' 'dap' '.' ( 'static' | 'instance' | 'class' | 'object' )

ScopeAnnotation
    ::= '@' 'co' '.' 'dap' '.' ( 'lexicalscope' | 'dynamicscope' | 'mixedscope'
                               | 'staticscope' )

OopsAnnotation
    ::= '@' 'co' '.' 'dap' '.' 'oops' '(' OopsEntry { ',' OopsEntry } [ ',' ] ')'

OopsEntry
    ::= Identifier ':' '{' OopsFlag { ',' OopsFlag } '}'

OopsFlag
    ::= ( 'inherit' | 'virtual' | 'implements' | 'inherits' | 'abstract'
        | 'uses' | 'composes' | 'extends' | 'with' | 'assiociate' )
        ( ':' | '=' ) BooleanLiteral
```

---

## 10. Lexical Grammar

```ebnf
(* --- 10.1 Whitespace and comments --- *)
Whitespace  ::= ( ' ' | '\t' | '\r' | '\n' )+
LineComment ::= '//' { AnyCharExceptNewline } NewLine

(* --- 10.2 Identifiers --- *)
Identifier
    ::= IdentStart { IdentPart }
      - ReservedWord

IdentStart ::= Letter | '_'
IdentPart  ::= Letter | Digit | '_'
Letter     ::= 'a'..'z' | 'A'..'Z'
Digit      ::= '0'..'9'

QualifiedName
    ::= Identifier { '.' Identifier }

(* --- 10.3 Special identifiers --- *)
BindVariable ::= '$' { Digit }              (* $, $0, $1, $2, ... *)

(* --- 10.4 Reserved words --- *)
ReservedWord
    ::= 'co' | 'let' | 'this' | 'self' | 'for' | 'forall' | 'fo'
        (* `self` is documented as a contextual keyword. *)

(* --- 10.5 Literals --- *)
IntegerLiteral ::= Digit { Digit }
FloatLiteral   ::= Digit { Digit } '.' Digit { Digit }
StringLiteral  ::= '"' { StringChar } '"'
CharLiteral    ::= "'" CharChar "'"
CharOrStringLiteral ::= CharLiteral | StringLiteral
StringChar     ::= AnyCharExcept('"', '\') | EscapeSeq
CharChar       ::= AnyCharExcept("'", '\') | EscapeSeq
EscapeSeq      ::= '\' AnyChar

(* --- 10.6 Punctuation / operator tokens (closed set, per reference) --- *)
ArithmeticOp ::= '+' | '-' | '*' | '/' | '%' | '**' | '++' | '--'
LogicalOp    ::= '&&' | '||' | '!' | '&' | '|'
ComparisonOp ::= '==' | '!=' | '<' | '>' | '<=' | '>='
OtherOp      ::= '@' | '#' | '~' | '$' | '^' | '(' | ')' | '_' | '`' | '?'
              | '{' | '[' | ']' | '}' | '\' | ':' | ';' | '"' | "'"
              | '=' | '.' | '?=' | ':=' | '::=' | ',' | '..' | '...'
              | '<..' | '..<' | '<..<' | '=>>' | '=>' | '->' | '<-'
              | '->>' | '<->' | '@@'

(* Reserved for future use — recognized as tokens but not currently bound
   to any production: *)
SpecialOp
    ::= 'λ' | '⒪' | 'â' | 'Ť' | '∀' | '∃' | '○' | 'ö' | '∪' | 'Ṡ' | 'Ŝ'
      | 'ṁ' | '𝚷' | '⇛' | '𝑓' | '𝒯' | '𝘷' | '𝓕' | '↓' | '∂' | '⊥' | '↧' | '⇓'
```

---

## 11. Start Symbols (entry points, restated)

```ebnf
Start
    ::= ApplicationEntryFile              (* app.fol / single-source app *)
      | PackageSourceFile                 (* ordinary <Name>.fol under a package *)
      | LibrarySurfaceFile                (* <lib>.fol at a library boundary *)
      | PackageAliasFile                  (* package.fol — planned feature *)
```

---

## 12. Informative Appendix — Closed Vocabulary Referenced by the Grammar

These tables are not themselves grammar productions; they enumerate the
terminal values that `KindKeyword(...)`, `AnnotationValue` built-in
identifiers, `TypeExpr` base names, and `CoPath` may take, as published by
the reference. Implementations should treat any `co.*` name outside this
list, used in a position requiring a *known* built-in, as unresolved rather
than silently accepted — the reference states `co.*` is always in lexical
scope but not always semantically permitted.

**Builtin Data Types** (`co.lang.*`):
`string, int, bit, double, float, long, byte, char, any, dynamic, auto,
infer, bool, void, data, value, typed, untyped, nothing, word,
MatchBindings, tag, typevalue, uninit, Literal, pointer, address,
reference, thunk, array, slice, range` — plus `co.mem.region`.

**Builtin Kinds** (`co.lang.*`, used as `KindKeyword`):
`type, struct, cstruct, realm, loader, class, interface, union, role,
record, property, indexer, object, instance, matcher, trait, mixin,
extension, delegate, typeclass, concept, typealias, module, unit, macro,
template, lambda, block, behavior, package, signature, function, method,
operator, namespace, stex, kind, level, order, rank, newtype, opaquetype,
subtype, supertype, dependentType, refinementType, associatedtype, hokrlt,
data, enum, typetype, typekind, alias, value, just, nothing`.

**Builtin Methods** (dot-called on values/objects):
`to_str, to_int, to_float, to_double, classof, typeof, new, prototype,
proto, make, objectof, instanceof, is, as, iskindof, has, hasown, uses,
match, matchall, matchany, matchnone, matchtype, case, with, print,
println, printsp, echo, contains, cast, to, dummy, clone, of, for, when,
where, then, callback, getAttr, inject, isinstance, cast_to, cast_from,
do, map, flatMap, orElse, filter, fold, recover, peek, loop, istrue,
isfalse, if, elif, else, return, otherwise, each, containsVal, in,
decltype, replace, send, receive, submitToPool, submitToEventLoop`.

**Builtin (root) Package** — `co` is the only default package; the
principal sub-packages named in the reference are: `co.lang, co.sys, co.os,
co.meta, co.core, co.native, co.in, co.out, co.regex, co.crypto, co.dap,
co.ddap, co.pdap, co.net, co.const, co.encoding, co.utils, co.dynamic,
co.runtime, co.compiletime, co.macro, co.pattern, co.control, co.cpca,
co.hokrtl, co.hokrt, co.fx, co.mem`.

---

## 13. Notes on Completeness and Open Areas

The reference document itself flags the following areas as incomplete,
planned, or explicitly out of scope for the initial release. The grammar
above still models their demonstrated surface syntax, but implementers
should treat these productions as provisional:

1. **Comprehensions** — marked *(planned)* throughout; `ForYieldComprehension`
   and the `.filter/.map` pipeline forms are both shown, without a
   reconciling unified grammar in the source.
2. **Package aliasing (`package.fol`)** — explicitly "Planned, not
   finalized to be part of initial release."
3. **Generics inheritance / path-dependent types** — explicitly
   "conceptual stage, not supported."
4. **Impredicative generics (`impredicative:true` full semantics)** —
   marked "v2," only the `co.lang.type`-wrapping workaround (v1) is
   currently legal.
5. **`@co.dap.hokrt` / `@co.dap.hokrlt`** (higher-kinded / higher-rank-
   order-kind-level types) appear only as annotation names on ADT type
   constructors; no further surface grammar is published for them beyond
   their use on `Name(T) co.lang.data = ...` declarations.
6. **Realms** (`realm=`, `parent-realm=`) are syntactically valid on every
   import but are only *semantically* active for `dynamicvmrt`-kind
   libraries; the grammar accepts them unconditionally per §3, with the
   activation constraint left as a semantic (non-syntactic) rule.
7. **Special/reserved-for-future operator glyphs** (§10.6 `SpecialOp`) are
   lexically reserved but carry no defined grammar productions yet.
8. The document's own **Statements** section explicitly leaves the
   statement-category list open-ended ("etc,."); §6 therefore should be
   read as the minimal statement grammar directly evidenced by the
   reference, not a closed set.

This grammar is licensed for reuse under the same terms as the source
language reference (CC BY 4.0) as it is a derivative organization of that
document's syntax and grammar rules.
