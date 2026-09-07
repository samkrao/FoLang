package parser

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	symboltable "github.com/samkrao/fo-lang/src/context"
)

// validateTypeclassInstances resolves every instance's contract after all
// project files and imported symbol graphs are available. Syntax parsing keeps
// the declarations single-pass; contract specialization is necessarily a
// project-level operation because the typeclass and its instance live in
// different files and may live in different imported packages.
func (a *projectAssembly) validateTypeclassInstances() {
	for _, info := range a.symbols.SymbolsById {
		instance, ok := info.(*symboltable.InstanceSymbol)
		if !ok {
			continue
		}
		a.validateTypeclassInstance(instance)
	}
}

func (a *projectAssembly) validateTypeclassInstance(instance *symboltable.InstanceSymbol) {
	table := a.symbols.GetSymbolTable(instance.SymbolTableId)
	contractID := resolveTypeNameAt(table, instance.TypeClassName, a.symbols)
	contract, ok := a.symbols.GetSymbol(contractID).(*symboltable.TypeclassSymbol)
	if !ok {
		a.typeclassConformanceError(instance, "references unknown typeclass %q", instance.TypeClassName)
		return
	}
	if len(instance.ForTypes) != len(contract.TypeParams) {
		a.typeclassConformanceError(instance,
			"binds %d type constructor(s), but typeclass %s requires %d from shape=(...)",
			len(instance.ForTypes), logicalName(contract.GetName()), len(contract.TypeParams))
		return
	}

	replacements := map[string]string{}
	for i, parameter := range contract.TypeParams {
		replacements[logicalName(parameter.Name)] = instance.ForTypes[i]
	}
	rawAliases := contractAliasRepresentations(contract, a.symbols)
	instance.SpecializedAliases = specializeContractAliases(rawAliases, replacements)
	if len(instance.SpecializedAliases) != len(rawAliases) {
		a.typeclassConformanceError(instance, "cannot specialize cyclic or unresolved aliases from typeclass %s", logicalName(contract.GetName()))
	}

	required := contextFunctionSignatures(contract.GetContextID(), a.symbols)
	provided := contextFunctionSignatures(instance.GetContextID(), a.symbols)
	for name, signatures := range required {
		actual := provided[name]
		if len(actual) == 0 {
			a.typeclassConformanceError(instance, "does not implement required typeclass method %s", name)
			continue
		}
		for _, signature := range signatures {
			if !containsString(actual, signature) {
				a.typeclassConformanceError(instance, "method %s has signature %s; required signature is %s", name, strings.Join(actual, " or "), signature)
			}
		}
		for _, signature := range actual {
			if !containsString(signatures, signature) {
				a.typeclassConformanceError(instance, "method %s provides undeclared overload signature %s", name, signature)
			}
		}
	}
	for name := range provided {
		if len(required[name]) == 0 {
			a.typeclassConformanceError(instance, "declares unknown method %s, which is not in typeclass %s", name, logicalName(contract.GetName()))
		}
	}
}

func (a *projectAssembly) typeclassConformanceError(instance *symboltable.InstanceSymbol, format string, args ...any) {
	detail := fmt.Sprintf(format, args...)
	a.diagnostics = append(a.diagnostics, projectDiagnostic(
		fmt.Sprintf("typeclass instance %s %s", logicalName(instance.GetName()), detail)))
}

func contractAliasRepresentations(contract *symboltable.TypeclassSymbol, symbols *symboltable.FolangSymbols) map[string]string {
	out := map[string]string{}
	context := symbols.GetContext(contract.GetContextID())
	if context == nil {
		return out
	}
	for _, name := range contract.AliasNames {
		info := symbols.GetSymbolTable(context.SymbolTable_).GetDetails(*symbols, name, string(symboltable.S_TypeSymbol))
		if info != nil && info.GetSymbolID() != "" {
			out[logicalName(name)] = info.GetType()
		}
	}
	return out
}

func specializeContractAliases(raw, bindings map[string]string) map[string]string {
	resolved := map[string]string{}
	for key, value := range bindings {
		resolved[logicalName(key)] = value
	}
	pending := map[string]string{}
	for key, value := range raw {
		pending[logicalName(key)] = value
	}
	for len(pending) != 0 {
		progress := false
		for name, value := range pending {
			expanded, blocked := substituteTypeNames(value, resolved, pending)
			if blocked {
				continue
			}
			resolved[name] = expanded
			delete(pending, name)
			progress = true
		}
		if !progress {
			break
		}
	}
	result := make(map[string]string, len(raw))
	for name := range raw {
		if value, ok := resolved[logicalName(name)]; ok {
			result[logicalName(name)] = value
		}
	}
	return result
}

func substituteTypeNames(value string, resolved, pending map[string]string) (string, bool) {
	var out strings.Builder
	for i := 0; i < len(value); {
		r := rune(value[i])
		if !unicode.IsLetter(r) && r != '_' {
			out.WriteByte(value[i])
			i++
			continue
		}
		start := i
		for i < len(value) {
			r = rune(value[i])
			if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '.' {
				break
			}
			i++
		}
		name := logicalName(value[start:i])
		if _, blocked := pending[name]; blocked {
			return "", true
		}
		if replacement, ok := resolved[name]; ok {
			out.WriteString(replacement)
		} else {
			out.WriteString(value[start:i])
		}
	}
	return out.String(), false
}

func contextFunctionSignatures(contextID string, symbols *symboltable.FolangSymbols) map[string][]string {
	out := map[string][]string{}
	context := symbols.GetContext(contextID)
	if context == nil {
		return out
	}
	for table := symbols.GetSymbolTable(context.SymbolTable_); table != nil; table = symbols.GetSymbolTable(table.ParentId) {
		for _, info := range symbols.Bindings(table.Id) {
			function, ok := info.(*symboltable.FunctionSymbol)
			if !ok {
				continue
			}
			name := logicalName(function.GetName())
			signature := "(" + function.ParameterSignature + ")->(" + function.ReturnSignature + ")"
			out[name] = append(out[name], signature)
		}
	}
	for name := range out {
		sort.Strings(out[name])
	}
	return out
}

func containsString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
