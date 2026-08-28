package ast

// NodeName reports the AST node's own kind as a bare name — "IfStmt",
// "DefaultConditionalStmt", "DummyStmt" — with no package qualifier and no
// decoration.
//
// It answers a question the other accessors on a node cannot. GetName reports
// what a node is CALLED in the source, which is a property of the program:
// a variable's spelling, a function's identifier, "" where a node names nothing.
// NodeName reports what a node IS, which is a property of the grammar and the
// same for every instance of the form.
//
// The distinction matters most where a node is looked at without being type
// switched. The serialized AST records a node as its exported fields alone, so
// nothing in the artifact says which production produced an object; a reader
// meeting `{Condition, Consequent, Alternate}` has to infer the node from its
// shape. A diagnostic, a trace line and a tree dump have the same problem in
// smaller form. Naming the form on the node itself gives all of them one answer
// rather than four reconstructions of it.
//
// Every name here is the Go type's own identifier, and it is written as a
// constant rather than derived by reflection so that reading a node's kind costs
// nothing on paths that print or serialize a whole tree. TestNodeNameMatchesTheGoTypeName
// holds the two spellings together.
//
// The method is required by SET, so a node type added without one does not
// compile. That is deliberate: the roster below is only complete while the
// compiler keeps it complete.

// Statements.
func (n AddressVariableDeclStmt) NodeName() string       { return "AddressVariableDeclStmt" }
func (n Application) NodeName() string                   { return "Application" }
func (n Argument) NodeName() string                      { return "Argument" }
func (n ArrayVariableDeclStmt) NodeName() string         { return "ArrayVariableDeclStmt" }
func (n ArrowFunction) NodeName() string                 { return "ArrowFunction" }
func (n BlockStmt) NodeName() string                     { return "BlockStmt" }
func (n BreakStmt) NodeName() string                     { return "BreakStmt" }
func (n BuiltInConstantStmt) NodeName() string           { return "BuiltInConstantStmt" }
func (n BuiltInStmt) NodeName() string                   { return "BuiltInStmt" }
func (n CaseStmt) NodeName() string                      { return "CaseStmt" }
func (n ClassDeclarationStmt) NodeName() string          { return "ClassDeclarationStmt" }
func (n CodeStmt) NodeName() string                      { return "CodeStmt" }
func (n ComponentDeclarationStmt) NodeName() string      { return "ComponentDeclarationStmt" }
func (n ConditionalStmt) NodeName() string               { return "ConditionalStmt" }
func (n ContainsStmt) NodeName() string                  { return "ContainsStmt" }
func (n ContinueStmt) NodeName() string                  { return "ContinueStmt" }
func (n DecoratorStmt) NodeName() string                 { return "DecoratorStmt" }
func (n DefaultConditionalStmt) NodeName() string        { return "DefaultConditionalStmt" }
func (n DelegateStmt) NodeName() string                  { return "DelegateStmt" }
func (n DependentTypeDeclarationStmt) NodeName() string  { return "DependentTypeDeclarationStmt" }
func (n DirectiveStmt) NodeName() string                 { return "DirectiveStmt" }
func (n DirectveList) NodeName() string                  { return "DirectveList" }
func (n DummyStmt) NodeName() string                     { return "DummyStmt" }
func (n ExecutionModelFunctionStmt) NodeName() string    { return "ExecutionModelFunctionStmt" }
func (n ExpressionStmt) NodeName() string                { return "ExpressionStmt" }
func (n ExtensionDeclarationStmt) NodeName() string      { return "ExtensionDeclarationStmt" }
func (n ExtensionStmt) NodeName() string                 { return "ExtensionStmt" }
func (n ForAllStmt) NodeName() string                    { return "ForAllStmt" }
func (n ForeachStmt) NodeName() string                   { return "ForeachStmt" }
func (n FunctionDeclarationStmt) NodeName() string       { return "FunctionDeclarationStmt" }
func (n FunctionPatternStmt) NodeName() string           { return "FunctionPatternStmt" }
func (n FunctionReceiver) NodeName() string              { return "FunctionReceiver" }
func (n GenerricFun) NodeName() string                   { return "GenerricFun" }
func (n HeapAllocatedRefStmt) NodeName() string          { return "HeapAllocatedRefStmt" }
func (n IfStmt) NodeName() string                        { return "IfStmt" }
func (n ImportStmt) NodeName() string                    { return "ImportStmt" }
func (n IndexerStmt) NodeName() string                   { return "IndexerStmt" }
func (n LabeledStmt) NodeName() string                   { return "LabeledStmt" }
func (n Library) NodeName() string                       { return "Library" }
func (n LockStmt) NodeName() string                      { return "LockStmt" }
func (n MacroStmt) NodeName() string                     { return "MacroStmt" }
func (n MatchExprStmt) NodeName() string                 { return "MatchExprStmt" }
func (n MatcherInstanceStmt) NodeName() string           { return "MatcherInstanceStmt" }
func (n ModuleStmt) NodeName() string                    { return "ModuleStmt" }
func (n NativeFunctionStmt) NodeName() string            { return "NativeFunctionStmt" }
func (n ObjectDeclStmt) NodeName() string                { return "ObjectDeclStmt" }
func (n OperatorStmt) NodeName() string                  { return "OperatorStmt" }
func (n PackageStmt) NodeName() string                   { return "PackageStmt" }
func (n Parameter) NodeName() string                     { return "Parameter" }
func (n PatternExprStmt) NodeName() string               { return "PatternExprStmt" }
func (n PointerVariableDeclStmt) NodeName() string       { return "PointerVariableDeclStmt" }
func (n PredicateTypeDeclarationStmt) NodeName() string  { return "PredicateTypeDeclarationStmt" }
func (n Prog) NodeName() string                          { return "Prog" }
func (n ProjectStmt) NodeName() string                   { return "ProjectStmt" }
func (n RangeVariableDeclStmt) NodeName() string         { return "RangeVariableDeclStmt" }
func (n RefVariableDeclStmt) NodeName() string           { return "RefVariableDeclStmt" }
func (n RefinementTypeDeclarationStmt) NodeName() string { return "RefinementTypeDeclarationStmt" }
func (n ReturnStmt) NodeName() string                    { return "ReturnStmt" }
func (n Returns) NodeName() string                       { return "Returns" }
func (n SliceVariableDeclStmt) NodeName() string         { return "SliceVariableDeclStmt" }
func (n TemplateStmt) NodeName() string                  { return "TemplateStmt" }
func (n TernaryStmt) NodeName() string                   { return "TernaryStmt" }
func (n ThunkVariableDeclStmt) NodeName() string         { return "ThunkVariableDeclStmt" }
func (n TraversableStmt) NodeName() string               { return "TraversableStmt" }
func (n TypeComposeStmt) NodeName() string               { return "TypeComposeStmt" }
func (n TypeConstructorStmt) NodeName() string           { return "TypeConstructorStmt" }
func (n TypeDeclarationStmt) NodeName() string           { return "TypeDeclarationStmt" }
func (n TypeStmt) NodeName() string                      { return "TypeStmt" }
func (n TypeclassInstanceStmt) NodeName() string         { return "TypeclassInstanceStmt" }
func (n TypeclassStmt) NodeName() string                 { return "TypeclassStmt" }
func (n UseStmtDirective) NodeName() string              { return "UseStmtDirective" }
func (n VarDeclarationStmt) NodeName() string            { return "VarDeclarationStmt" }

// Expressions.
func (n ADTExpr) NodeName() string                  { return "ADTExpr" }
func (n ArrayLiteral) NodeName() string             { return "ArrayLiteral" }
func (n AssignmentExpr) NodeName() string           { return "AssignmentExpr" }
func (n BinaryExpr) NodeName() string               { return "BinaryExpr" }
func (n BindVariableExpr) NodeName() string         { return "BindVariableExpr" }
func (n BooleanLiteral) NodeName() string           { return "BooleanLiteral" }
func (n CallExpr) NodeName() string                 { return "CallExpr" }
func (n CharacterLiteral) NodeName() string         { return "CharacterLiteral" }
func (n CommaExpr) NodeName() string                { return "CommaExpr" }
func (n ComputedExpr) NodeName() string             { return "ComputedExpr" }
func (n ConditionalExpr) NodeName() string          { return "ConditionalExpr" }
func (n DefaultExpr) NodeName() string              { return "DefaultExpr" }
func (n ForComprehensionExpr) NodeName() string     { return "ForComprehensionExpr" }
func (n FunctionExpr) NodeName() string             { return "FunctionExpr" }
func (n GroupingExpr) NodeName() string             { return "GroupingExpr" }
func (n IntegerLiteral) NodeName() string           { return "IntegerLiteral" }
func (n LambdaExpr) NodeName() string               { return "LambdaExpr" }
func (n LetExpr) NodeName() string                  { return "LetExpr" }
func (n LifecycleCallExpr) NodeName() string        { return "LifecycleCallExpr" }
func (n MemberExpr) NodeName() string               { return "MemberExpr" }
func (n NewExpr) NodeName() string                  { return "NewExpr" }
func (n NumberLiteral) NodeName() string            { return "NumberLiteral" }
func (n ParentSelectorExpr) NodeName() string       { return "ParentSelectorExpr" }
func (n PlaceHolderExpr) NodeName() string          { return "PlaceHolderExpr" }
func (n PrefixExpr) NodeName() string               { return "PrefixExpr" }
func (n RangeExpr) NodeName() string                { return "RangeExpr" }
func (n RelationshipSelectorExpr) NodeName() string { return "RelationshipSelectorExpr" }
func (n SDTExpr) NodeName() string                  { return "SDTExpr" }
func (n StatementExpr) NodeName() string            { return "StatementExpr" }
func (n StringLiteral) NodeName() string            { return "StringLiteral" }
func (n SymbolExpr) NodeName() string               { return "SymbolExpr" }

// Types.
func (n BuiltInDataType) NodeName() string { return "BuiltInDataType" }
func (n CompoundType) NodeName() string    { return "CompoundType" }
func (n DependentType) NodeName() string   { return "DependentType" }
func (n DerivedType) NodeName() string     { return "DerivedType" }
func (n ForAllType) NodeName() string      { return "ForAllType" }
func (n FunctionType) NodeName() string    { return "FunctionType" }
func (n GenericType) NodeName() string     { return "GenericType" }
func (n ListType) NodeName() string        { return "ListType" }
func (n SymbolRefExpr) NodeName() string   { return "SymbolRefExpr" }
func (n SymbolTypeNode) NodeName() string  { return "SymbolTypeNode" }
