package helpers

// DiagnosticName is a stable, machine-readable name from the normative
// diagnostic registry in docs/language-ref.md. Display headings and details
// are deliberately separate from this value.
type DiagnosticName string

const (
	DiagnosticUnsupportedFeature           DiagnosticName = "UnsupportedFeature"
	DiagnosticUnsupportedBackendFeature    DiagnosticName = "UnsupportedBackendFeature"
	DiagnosticInvalidIdentifier            DiagnosticName = "InvalidIdentifier"
	DiagnosticInvalidLiteral               DiagnosticName = "InvalidLiteral"
	DiagnosticUnexpectedToken              DiagnosticName = "UnexpectedToken"
	DiagnosticExpectedToken                DiagnosticName = "ExpectedToken"
	DiagnosticInvalidSyntax                DiagnosticName = "InvalidSyntax"
	DiagnosticUnknownOperator              DiagnosticName = "UnknownOperator"
	DiagnosticInvalidProjectLayout         DiagnosticName = "InvalidProjectLayout"
	DiagnosticInvalidSourcePlacement       DiagnosticName = "InvalidSourcePlacement"
	DiagnosticInvalidDeclarationForm       DiagnosticName = "InvalidDeclarationForm"
	DiagnosticReservedPackageShadowing     DiagnosticName = "ReservedPackageShadowing"
	DiagnosticInvalidImport                DiagnosticName = "InvalidImport"
	DiagnosticInvalidDependency            DiagnosticName = "InvalidDependency"
	DiagnosticDependencyCycle              DiagnosticName = "DependencyCycle"
	DiagnosticCapabilityViolation          DiagnosticName = "CapabilityViolation"
	DiagnosticExportBoundaryViolation      DiagnosticName = "ExportBoundaryViolation"
	DiagnosticDuplicateDeclaration         DiagnosticName = "DuplicateDeclaration"
	DiagnosticDuplicateCallableSignature   DiagnosticName = "DuplicateCallableSignature"
	DiagnosticUnresolvedSymbol             DiagnosticName = "UnresolvedSymbol"
	DiagnosticAmbiguousSymbol              DiagnosticName = "AmbiguousSymbol"
	DiagnosticInaccessibleSymbol           DiagnosticName = "InaccessibleSymbol"
	DiagnosticInvalidReceiver              DiagnosticName = "InvalidReceiver"
	DiagnosticUnusedImport                 DiagnosticName = "UnusedImport"
	DiagnosticUnusedSymbol                 DiagnosticName = "UnusedSymbol"
	DiagnosticTypeMismatch                 DiagnosticName = "TypeMismatch"
	DiagnosticConstraintViolation          DiagnosticName = "ConstraintViolation"
	DiagnosticInvalidAssignment            DiagnosticName = "InvalidAssignment"
	DiagnosticUninitialized                DiagnosticName = "Uninitialized"
	DiagnosticInvalidNoneUse               DiagnosticName = "InvalidNoneUse"
	DiagnosticInvalidDependentIndex        DiagnosticName = "InvalidDependentIndex"
	DiagnosticDependentTypeMismatch        DiagnosticName = "DependentTypeMismatch"
	DiagnosticInvalidConstruction          DiagnosticName = "InvalidConstruction"
	DiagnosticMutationNotAllowed           DiagnosticName = "MutationNotAllowed"
	DiagnosticOverloadNotAllowed           DiagnosticName = "OverloadNotAllowed"
	DiagnosticNoApplicableOverload         DiagnosticName = "NoApplicableOverload"
	DiagnosticAmbiguousOverload            DiagnosticName = "AmbiguousOverload"
	DiagnosticInvalidReturn                DiagnosticName = "InvalidReturn"
	DiagnosticMissingImplementation        DiagnosticName = "MissingImplementation"
	DiagnosticInvalidForwardDeclaration    DiagnosticName = "InvalidForwardDeclaration"
	DiagnosticInvalidGenericDeclaration    DiagnosticName = "InvalidGenericDeclaration"
	DiagnosticInvalidGenericMapping        DiagnosticName = "InvalidGenericMapping"
	DiagnosticGenericResolutionFailure     DiagnosticName = "GenericResolutionFailure"
	DiagnosticSignatureConformanceFailure  DiagnosticName = "SignatureConformanceFailure"
	DiagnosticInvalidAssociatedTypeBinding DiagnosticName = "InvalidAssociatedTypeBinding"
	DiagnosticInvalidRelationship          DiagnosticName = "InvalidRelationship"
	DiagnosticInheritedMemberConflict      DiagnosticName = "InheritedMemberConflict"
	DiagnosticInvalidOverride              DiagnosticName = "InvalidOverride"
	DiagnosticInvalidImplementation        DiagnosticName = "InvalidImplementation"
	DiagnosticInvalidLifecycleDeclaration  DiagnosticName = "InvalidLifecycleDeclaration"
	DiagnosticInvalidMemberPolicy          DiagnosticName = "InvalidMemberPolicy"
	DiagnosticUnresolvedMetadataForm       DiagnosticName = "UnresolvedMetadataForm"
	DiagnosticInvalidMetadataPlacement     DiagnosticName = "InvalidMetadataPlacement"
	DiagnosticInvalidMetadataTarget        DiagnosticName = "InvalidMetadataTarget"
	DiagnosticMissingMetadataField         DiagnosticName = "MissingMetadataField"
	DiagnosticInvalidMetadataField         DiagnosticName = "InvalidMetadataField"
	DiagnosticInvalidMetadataValue         DiagnosticName = "InvalidMetadataValue"
	DiagnosticConflictingMetadata          DiagnosticName = "ConflictingMetadata"
	DiagnosticInvalidOperatorDeclaration   DiagnosticName = "InvalidOperatorDeclaration"
	DiagnosticInvalidOperatorOverload      DiagnosticName = "InvalidOperatorOverload"
	DiagnosticInvalidEffectDeclaration     DiagnosticName = "InvalidEffectDeclaration"
	DiagnosticInvalidEffectPolicy          DiagnosticName = "InvalidEffectPolicy"
	DiagnosticInvalidEffectHandler         DiagnosticName = "InvalidEffectHandler"
	DiagnosticInvalidEffectResolution      DiagnosticName = "InvalidEffectResolution"
	DiagnosticInvalidRetryPolicy           DiagnosticName = "InvalidRetryPolicy"
	DiagnosticInvalidExecutionModel        DiagnosticName = "InvalidExecutionModel"
)

var registeredDiagnosticNames = map[DiagnosticName]struct{}{
	DiagnosticUnsupportedFeature: {}, DiagnosticUnsupportedBackendFeature: {}, DiagnosticInvalidIdentifier: {}, DiagnosticInvalidLiteral: {},
	DiagnosticUnexpectedToken: {}, DiagnosticExpectedToken: {}, DiagnosticInvalidSyntax: {}, DiagnosticUnknownOperator: {},
	DiagnosticInvalidProjectLayout: {}, DiagnosticInvalidSourcePlacement: {}, DiagnosticInvalidDeclarationForm: {}, DiagnosticReservedPackageShadowing: {},
	DiagnosticInvalidImport: {}, DiagnosticInvalidDependency: {}, DiagnosticDependencyCycle: {}, DiagnosticCapabilityViolation: {}, DiagnosticExportBoundaryViolation: {},
	DiagnosticDuplicateDeclaration: {}, DiagnosticDuplicateCallableSignature: {}, DiagnosticUnresolvedSymbol: {}, DiagnosticAmbiguousSymbol: {}, DiagnosticInaccessibleSymbol: {}, DiagnosticInvalidReceiver: {}, DiagnosticUnusedImport: {}, DiagnosticUnusedSymbol: {},
	DiagnosticTypeMismatch: {}, DiagnosticConstraintViolation: {}, DiagnosticInvalidAssignment: {}, DiagnosticUninitialized: {}, DiagnosticInvalidNoneUse: {}, DiagnosticInvalidDependentIndex: {}, DiagnosticDependentTypeMismatch: {}, DiagnosticInvalidConstruction: {}, DiagnosticMutationNotAllowed: {},
	DiagnosticOverloadNotAllowed: {}, DiagnosticNoApplicableOverload: {}, DiagnosticAmbiguousOverload: {}, DiagnosticInvalidReturn: {}, DiagnosticMissingImplementation: {}, DiagnosticInvalidForwardDeclaration: {},
	DiagnosticInvalidGenericDeclaration: {}, DiagnosticInvalidGenericMapping: {}, DiagnosticGenericResolutionFailure: {}, DiagnosticSignatureConformanceFailure: {}, DiagnosticInvalidAssociatedTypeBinding: {},
	DiagnosticInvalidRelationship: {}, DiagnosticInheritedMemberConflict: {}, DiagnosticInvalidOverride: {}, DiagnosticInvalidImplementation: {}, DiagnosticInvalidLifecycleDeclaration: {}, DiagnosticInvalidMemberPolicy: {},
	DiagnosticUnresolvedMetadataForm: {}, DiagnosticInvalidMetadataPlacement: {}, DiagnosticInvalidMetadataTarget: {}, DiagnosticMissingMetadataField: {}, DiagnosticInvalidMetadataField: {}, DiagnosticInvalidMetadataValue: {}, DiagnosticConflictingMetadata: {},
	DiagnosticInvalidOperatorDeclaration: {}, DiagnosticInvalidOperatorOverload: {}, DiagnosticInvalidEffectDeclaration: {}, DiagnosticInvalidEffectPolicy: {}, DiagnosticInvalidEffectHandler: {}, DiagnosticInvalidEffectResolution: {}, DiagnosticInvalidRetryPolicy: {}, DiagnosticInvalidExecutionModel: {},
}

// IsRegisteredDiagnosticName reports whether name belongs to the normative
// registry. It is exported for compiler/tooling conformance tests.
func IsRegisteredDiagnosticName(name string) bool {
	_, ok := registeredDiagnosticNames[DiagnosticName(name)]
	return ok
}
