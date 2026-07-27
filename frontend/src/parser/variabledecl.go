package parser

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/samkrao/fo-lang/frontend/src/ast"
	symboltable "github.com/samkrao/fo-lang/frontend/src/context"
	"github.com/samkrao/fo-lang/frontend/src/helpers"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
	"golang.org/x/exp/slices"
)

// parse_decl_stmt parses a variable-, field-, parameter-, or result-shaped
// declaration.
//
//	what can be
//	  * normal variables
//	  * poinnters
//	  * thunks
//	  * arrays (jagged, multidimensions, zero lengh, zero dimension, variable length)
//	  * References (Heap references, single reference (Lvalues), Double reference (RValues))
//	  * address variables
//	  * walrus
//	  * generic types
//	  * forall types
//	  * function parameters ( Named, optional, default, varargs, normal)
//	  * function returns ( named and no named)
//
// Feature examples:
//
//	name co.lang.string = "Rao";
//	age  co.lang.int;
//	nums co.lang.arr->(dim=1, size=10) = [1,2,3];
//	ptr  co.lang.ptr->(kind=raw) = &value;
//	x := 10;
//	y ?= 20;
//	fun1(~k co.lang.int, value? co.lang.int, ...rest co.lang.int)->() = {}
//
// The type_ parameter selects the declaration flavour: "var", "param",
// "result", "auto", "adhoc", "fieldormemberorprop", "fovar", "let", "field",
// "walrus", "redeclare", or "range".
func parse_decl_stmt(p *parser, type_ string, isConstant bool, allowinit bool, allowNoName bool, varType Vartype, vartypes_ any, decl bool, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	pr := ParseResult{
		Node:   nil,
		Errors: []helpers.ErrorInterface{},
	}
	redeclarable := false

	// Feature example:
	//   x co.lang.int = 10, y co.lang.int = 20;
	//
	// Consecutive variable declarations can open a fresh symbol table segment so
	// the parser can model declaration grouping without mutating unrelated outer
	// bindings.
	if !p.IsPrevDeclSymbol && type_ == "var" {
		currSymbolTable := p.SymbolTable_

		p.SymbolTable_ = createNewSymbolTable(p.Context_.Id, p.SymbolTable_.Id)
		p.SymbolTable_.Prev = currSymbolTable.Id
		currSymbolTable.Next = p.SymbolTable_.Id
		p.Fs.AddSymbolTable(p.SymbolTable_)
	}
	subType_ := "co.lang.var"

	isDyanamicRedec := false
	var explicitType ast.Type
	var errs []helpers.ErrorInterface
	var symbolName scanlex.Token
	deduceType := false
	localbinding := false
	optionalParam := false
	thunkArg := false
	variadicParam := false
	namedArgs := false

	if type_ == "var" {
		// Feature example:
		//   name co.lang.string = "Rao";
		symbolName_, err_ := p.expectError(scanlex.IDENTIFIER,
			fmt.Sprintf("Following %s expected variable name however instead recieved %s instead\n",
				scanlex.TokenKindString(p.previousToken(1).Kind), scanlex.TokenKindString(p.currentTokenKind())))

		p.addErr(err_)
		symbolName = symbolName_
	} else if type_ == "param" {
		// Feature examples:
		//   fun1(~k co.lang.int)->()
		//   fun1(value? co.lang.int)->()
		//   fun1(...rest co.lang.int)->()
		//   fun1(^lazyValue co.lang.int)->()
		//
		// Parameter parsing also records parameter modifiers such as named,
		// optional, variadic, and thunk forms.
		if !allowNoName {
			if p.currentToken().Kind == scanlex.TILD {
				namedArgs = true
			} else if p.currentTokenKind() == scanlex.POW {
				thunkArg = true
			} else if p.currentTokenKind() == scanlex.DOT_DOT_DOT {
				variadicParam = true
			}

			symbolName_, err_ := p.expectError(scanlex.IDENTIFIER,
				fmt.Sprintf("Expected Parameter/Argument name however instead recieved %s instead\n",
					scanlex.TokenKindString(p.currentTokenKind())))

			p.addErr(err_)
			symbolName = symbolName_
			if p.currentTokenKind() == scanlex.QUESTION {
				optionalParam = true
				p.advance()
			}
		} else {
			symN := "_co_internal_" + helpers.GenUnique(4)
			symbolName = scanlex.NewUniqueToken(scanlex.IDENTIFIER, symN, p.currentToken().StartPos, p.currentToken().EndPos)
		}
	} else if type_ == "result" {
		symbolName_, err_ := p.expectError(scanlex.IDENTIFIER,
			fmt.Sprintf("Eexpected Return/Result name however instead recieved %s instead\n",
				scanlex.TokenKindString(p.currentTokenKind())))

		p.addErr(err_)
		symbolName = symbolName_
	} else if type_ == "auto" {
		// Feature example:
		//   value co.lang.dynamic = 10;
		startToken := p.currentToken().Kind
		isConstant = false
		isDyanamicRedec = true
		symbolName_, err_ := p.expectError(scanlex.IDENTIFIER,
			fmt.Sprintf("Following %s expected variable name however instead recieved %s instead\n",
				scanlex.TokenKindString(startToken), scanlex.TokenKindString(p.currentTokenKind())))

		p.addErr(err_)
		symbolName = symbolName_
	} else if type_ == "adhoc" {
		// Feature example:
		//   internally synthesized helper bindings used by parser helpers.
		m, b := tryCastToStringMap(vartypes_)
		if b {
			name := m["Varname"]
			symbolName = scanlex.NewUniqueToken(scanlex.IDENTIFIER, name, p.currentToken().StartPos, p.currentToken().EndPos)

		} else {
			err := p.errorExpection("Something Went wrong....", helpers.IllegalChar)
			p.addErr(err)
		}
	} else if type_ == "fieldormemberorprop" {
		// no define or declare or let just variable name and type
	} else if type_ == "fovar" {
		// Feature example:
		//   value := someFunctionResult
		isConstant = false
		deduceType = true
		symbolName_, err_ := p.expectError(scanlex.IDENTIFIER,
			fmt.Sprintf("Following %s expected variable name however instead recieved %s instead\n",
				scanlex.TokenKindString(p.previousToken(1).Kind), scanlex.TokenKindString(p.currentTokenKind())))

		p.addErr(err_)
		symbolName = symbolName_
	} else if type_ == "let" {
		// Feature example:
		//   let x := 10
		isConstant = false
		deduceType = true
		symbolName_, err_ := p.expectError(scanlex.IDENTIFIER,
			fmt.Sprintf("Following %s expected variable name however instead recieved %s instead\n",
				scanlex.TokenKindString(p.previousToken(1).Kind), scanlex.TokenKindString(p.currentTokenKind())))

		p.addErr(err_)
		symbolName = symbolName_
	} else if type_ == "field" {
		token := p.currentToken()

		startToken := token.Kind
		symbolName_, err_ := p.expectError(scanlex.IDENTIFIER,
			fmt.Sprintf("Following %s expected variable name however instead recieved %s instead\n",
				scanlex.TokenKindString(startToken), scanlex.TokenKindString(p.currentTokenKind())))

		p.addErr(err_)
		symbolName = symbolName_
	} else if type_ == "walrus" || type_ == "redeclare" {

	} else if type_ == "range" || type_ == "range_redeclare" {

	}

	if type_ == "redeclare" || type_ == "range_redeclare" {
		redeclarable = true
	}
	if (symbolName.Value == "__fo" || symbolName.Value == "_") && !allowNoName {
		err_ := p.errorObj(nil, "found Underscore")
		p.addErr(err_)
	}
	isexplicitType := true
	if isDyanamicRedec {
		// Dynamic declarations defer type identity to runtime behavior.
		isexplicitType = false
	} else if deduceType {
		// Feature examples:
		//   x := 10
		//   result := someCall()
		//
		// Deduction means the parser skips an explicit type and lets the assigned
		// value drive the inferred type later.
		p.advance()
		isexplicitType = false
	} else if type_ == "adhoc" {
		m, _ := tryCastToStringMap(vartypes_)
		typee_ := scanlex.BUILT_IN_TYPE
		typ_ := m["Type"]
		if slices.Contains(scanlex.Builtin_types, typ_) {

		} else if strings.Contains(typ_, ".") {
			typee_ = scanlex.COMPOSITE_IDENTIFER
		} else {
			typee_ = scanlex.IDENTIFIER
		}
		tk := scanlex.NewUniqueToken(typee_, typ_, p.currentToken().StartPos, p.currentToken().EndPos)
		explicitType, errs = parse_adhoc_type(p, defalt_bp, tk, ddaps)
		if errs != nil {

			err_ := p.errorObj(nil, "Invalid Type")
			p.addErr(err_)
		}
	} else if v, ok := p.AdditionalInfo["genericfunction"]; ok && v.(bool) {
		// Feature examples:
		//   identity(x T)->(T) = { ... }
		//   apply(f forall(T).(T)->(T))->(co.lang.int) = { ... }
		//
		// Generic function mode changes how identifier-like type syntax is
		// interpreted inside declarations.
		if p.currentToken().Value == "forall" {
			// Rank-2: parameter type is itself universally quantified, e.g.
			//   f forall(T) (T)->(T)
			// Parse the ForAllType directly; skip the "must be in type param list"
			// validation that applies only to generic type variables like T or R.
			explicitType, errs = parse_forall_type(p, ddaps)
			if errs != nil {
				err_ := p.errorObj(nil, "Invalid ForAll type")
				p.addErr(err_)
			}
		} else {
			explicitType, errs = parse_generic_types(p, defalt_bp, ddaps)

			if errs != nil {
				err_ := p.errorObj(nil, "Invalid Type")
				p.addErr(err_)
			}
			typeinList := false
			for _, name := range p.AdditionalInfo["generrictypes"].([]string) {
				if name == explicitType.GetName() {
					typeinList = true
					break
				}
			}
			if !typeinList {
				err_ := p.errorObj(nil, "Generic type used in variable declaration not defined in generic type list")
				p.addErr(err_)
			}
		}
	} else {
		// Feature example:
		//   name co.lang.string = "Rao";
		//   nums co.lang.arr->(...) = ...
		//
		// Ordinary declarations parse an explicit type (including compound and
		// special storage forms) before reading any initializer.
		if p.currentTokenKind() == scanlex.SEMI_COLON || p.currentTokenKind() == scanlex.COMMA || p.currentTokenKind() == scanlex.ASSIGNMENT {
			var tk scanlex.Token = p.currentToken()
			err_ := p.errorObj(&tk, "Expected type")
			p.addErr(err_)
		}
		explicitType, errs = parse_compound_type(p, defalt_bp, ddaps)

		if errs != nil {

			err_ := p.errorObj(nil, "Invalid Type")
			p.addErr(err_)
		}

	}

	thunk := false
	heapallocref := false
	var assignmentValue ast.Expr

	var arrVar ast.ArrayVariableDeclStmt = ast.ArrayVariableDeclStmt{}
	var addrVar ast.AddressVariableDeclStmt = ast.AddressVariableDeclStmt{}
	var ptrVar ast.PointerVariableDeclStmt = ast.PointerVariableDeclStmt{}
	var refVar ast.RefVariableDeclStmt = ast.RefVariableDeclStmt{}
	var thunkVar ast.ThunkVariableDeclStmt = ast.ThunkVariableDeclStmt{}
	var sliceVar ast.SliceVariableDeclStmt = ast.SliceVariableDeclStmt{}
	var heapAllRef ast.HeapAllocatedRefStmt = ast.HeapAllocatedRefStmt{}
	var rangeVar ast.RangeVariableDeclStmt = ast.RangeVariableDeclStmt{}
	actType := ""
	if isDyanamicRedec {
	}
	if isexplicitType {
		switch st := explicitType.(type) {
		case ast.SymbolTypeNode:
			actType = st.Value
		case ast.BuiltInDataType:
			actType = st.Value
		}
	}

	vd := symboltable.VariableDetails{
		ExplicitType: isexplicitType,
		ActType_:     actType,
		Dynamic:      isDyanamicRedec,
		DuckType:     isDyanamicRedec,
		Inferred:     deduceType,
		LocalBinding: localbinding,
		SymbolDetails: symboltable.SymbolDetails{
			SymbolType_: string(symboltable.S_VarSymbol),
		},
	}
	var1 := ast.BasicVarStmt{
		Identifier:    symbolName.Value,
		AssignedValue: assignmentValue,
		Type_:         explicitType,
		VarType:       "",
	}
	if _, ok := isAnnDec(ddaps, "@co.dap.mutable"); ok {
		vd.Mutable = true
	}

	var su_ Varsubtype = Normal

	if type_ != "adhoc" {
		golook := ((type_ == "param" || type_ == "result") && p.currentTokenKind() == scanlex.CLOSE_PAREN)
		if p.currentTokenKind() != scanlex.SEMI_COLON && p.currentTokenKind() != scanlex.COMMA && !golook {
			_, err_ := p.expectAny(scanlex.ASSIGNMENT, scanlex.ARROW)
			p.addErr(err_)
			if p.previousToken(1).Kind == scanlex.ARROW {

				_, err_ := p.expect(scanlex.OPEN_PAREN)
				p.addErr(err_)
				if p.currentTokenKind() == scanlex.DOT_DOT {
					su_ = Range
					vd.SubID = "RANGE"
					vd.SubType_ = "co.lang.range"
					rangeVar = ast.RangeVariableDeclStmt{
						Symb: &symboltable.RangeSymbol{
							VariableDetails: vd,
						},
						BasicVarStmt: var1,
					}

				} else if p.currentTokenKind() == scanlex.TILD {
					su_ = HeapAllocRef

					vd.SubID = "HeapAllocRef"
					vd.SubType_ = "co.lang.heapallocref"
					heapAllRef = ast.HeapAllocatedRefStmt{
						BasicVarStmt: var1,
						Symb: &symboltable.ReferenceSymbol{
							VariableDetails: vd,
						},
					}
					heapallocref = true

				} else if p.currentToken().Kind == scanlex.POW {
					su_ = Thunk
					vd.SubID = "Thunk"
					vd.SubType_ = "co.lang.thunk"
					thunkVar = ast.ThunkVariableDeclStmt{
						Symb: &symboltable.ThunkSymbol{
							VariableDetails: vd,
						},
						BasicVarStmt: var1,
					}
					thunk = true
				} else if p.currentToken().Kind == scanlex.OPEN_BRACKET {
					if p.nextTokenSafe(1).Kind == scanlex.COLON {
						p.advance()
						p.advance()
						su_ = Slice
						vd.SubID = "Slice"
						vd.SubType_ = "co.lang.slice"
						sliceVar = ast.SliceVariableDeclStmt{
							Symb: &symboltable.ArraySymbol{
								VariableDetails: vd,
							},
							BasicVarStmt: var1,
						}
						_, err_ := p.expect(scanlex.CLOSE_BRACKET)
						p.addErr(err_)

					} else {
						su_ = Array
						temp := maker_array(p, &var1, &vd, ddaps)

						arrVar = temp.Node.(ast.ArrayVariableDeclStmt)

					}

				} else if p.currentToken().Kind == scanlex.STAR {
					su_ = Pointer

					kindVal := "thin"
					meta := map[string]any{}
					count := 0
					for p.currentToken().Kind == scanlex.STAR {
						p.advance()
						count = count + 1
					}
					switch count {
					case 1:
						subType_ = "co.lang.ptr"
					case 2:
						subType_ = "co.lang.dbl.ptr"
					case 3:
						subType_ = "co.lang.tpl.ptr"
					default:
						err_ := p.errorExpection("More than 3 level pointer indirection not supported directly", helpers.InvalidSyntax)
						p.addErr(err_)
					}
					if p.currentToken().Kind == scanlex.COMMA {
						p.advance()
						// define x int ->(*, meta={},kind=}
						/*
						  valid kind is
						  fat
						  thin
						  region
						  slice
						  trait
						  buffer
						  view
						  opaque
						  custome
						*/
						/* meta

						 */
						p.advance()
						if p.currentToken().Value == "meta" {
							meta = parse_pointer_meta(p, ddaps)

						}
						if p.currentToken().Kind == scanlex.COMMA {
							p.advance()
							if p.currentToken().Value == "kind" {
								p.advance()
								_, err_ := p.expect(scanlex.ASSIGNMENT)
								p.addErr(err_)
								kindTok := p.currentToken()
								kindVal = kindTok.Value
								validKinds := []string{"fat", "thin", "region", "slice", "trait", "buffer", "view", "opaque", "custom", "relative", "mem", "nullptr", "sptr", "uptr", "ptrdiff", "usize", "ssize"}
								if !slices.Contains(validKinds, kindVal) {
									err_ := p.errorExpection("Invalid pointer kind "+kindVal, helpers.InvalidSyntax)
									p.addErr(err_)
								}
								p.advance()
							}
						}
					}
					vd.SubType_ = subType_
					symb := symboltable.PointerSymbol{
						VariableDetails:     vd,
						IsRaw:               true,
						Count:               count,
						PtrToConstType:      false,
						ConstPtrToType:      false,
						ConstPtrToConstType: false,
						MetaData:            meta,
						ISFatPointer:        false,
						Kind_:               kindVal,
					}
					ptrVar = ast.PointerVariableDeclStmt{
						BasicVarStmt: var1,
						Symb:         &symb,
					}
				} else if p.currentToken().Kind == scanlex.AND {
					su_ = Reference
					//C++ <type>&& t = 20; rvalue
					_, err_ := p.expect(scanlex.AND)
					p.addErr(err_)

					vd.SubID = "DBL_REF"
					vd.SubType_ = "co.lang.dbl.ref"
					symb := symboltable.ReferenceSymbol{
						VariableDetails: vd,
						Lref:            true,
						Heap:            false,
						Ref:             true,
						Count:           2,
					}
					refVar = ast.RefVariableDeclStmt{
						BasicVarStmt: var1,
						Symb:         &symb,
					}

				} else if p.currentToken().Kind == scanlex.AMPS {
					su_ = Reference
					// C++ <type>& t = a; lvalue
					_, err_ := p.expect((scanlex.AMPS))
					p.addErr(err_)

					vd.SubID = "REF"
					vd.SubType_ = "co.lang.ref"
					symb := symboltable.ReferenceSymbol{
						VariableDetails: vd,
						Lref:            false,
						Heap:            false,
						Ref:             true,
						Count:           1,
					}
					refVar = ast.RefVariableDeclStmt{
						BasicVarStmt: var1,
						Symb:         &symb,
					}
				} else if p.currentToken().Kind == scanlex.AT {
					su_ = Address
					_, err_ := p.expect(scanlex.AT)
					p.addErr(err_)

					vd.SubID = "ADDR"
					vd.SubType_ = "co.lang.addr"
					symb := symboltable.AddressSymbol{
						VariableDetails: vd,
						Addressop:       true,
						Wordtype:        false,
					}
					addrVar = ast.AddressVariableDeclStmt{
						BasicVarStmt: var1,
						Symb:         &symb,
					}

				} else if p.currentToken().Kind == scanlex.IDENTIFIER {
					// Parse key=value repr/sign pairs: ->(repr=intptr) or ->(sign=unsigned, repr=uintptr)
					su_ = Pointer

					subType_ = "co.lang.word.repr"
					meta := map[string]any{}
					kindVal := ""

					for p.currentToken().Kind == scanlex.IDENTIFIER {
						keyTok := p.currentToken()
						p.advance()
						_, err_ := p.expect(scanlex.ASSIGNMENT)
						p.addErr(err_)
						valTok := p.currentToken()
						p.advance()

						if keyTok.Value == "repr" {
							validReprs := []string{"intptr", "uintptr", "ptrdiff", "usize", "isize", "nullptr"}
							if !slices.Contains(validReprs, valTok.Value) {
								err_ := p.errorExpection("Invalid repr value "+valTok.Value, helpers.InvalidSyntax)
								p.addErr(err_)
							}
							kindVal = valTok.Value
							meta["repr"] = valTok.Value
						} else if keyTok.Value == "sign" {
							validSigns := []string{"unsigned", "signed"}
							if !slices.Contains(validSigns, valTok.Value) {
								err_ := p.errorExpection("Invalid sign value "+valTok.Value, helpers.InvalidSyntax)
								p.addErr(err_)
							}
							meta["sign"] = valTok.Value
						} else {
							err_ := p.errorExpection("Unknown repr attribute "+keyTok.Value, helpers.InvalidSyntax)
							p.addErr(err_)
							meta[keyTok.Value] = valTok.Value
						}

						if p.currentToken().Kind == scanlex.COMMA {
							p.advance()
						}
					}
					PtrSymbol := symboltable.PointerSymbol{
						VariableDetails:     symboltable.VariableDetails{SubType_: subType_},
						Count:               0,
						PtrToConstType:      false,
						ConstPtrToType:      false,
						ConstPtrToConstType: false,
						IsRaw:               false,
						MetaData:            meta,
					}
					ptrVar = ast.PointerVariableDeclStmt{
						Symb:         &PtrSymbol,
						BasicVarStmt: var1,
						Kind_:        kindVal,
					}
				}

				_, err_ = p.expect(scanlex.CLOSE_PAREN)
				p.addErr(err_)
				if p.currentToken().Kind != scanlex.SEMI_COLON && p.currentTokenKind() != scanlex.COMMA {
					_, err_ := p.expect(scanlex.ASSIGNMENT)
					p.addErr(err_)
					if p.currentToken().Value == "let" {
						tr := parse_let_binding(p, ddaps)
						assignmentValue = tr.Node.(ast.Expr)
					} else {
						expr := parse_expr(p, assignment, ddaps)
						assignmentValue = expr.Node.(ast.Expr)
					}
				}
				if su_ == Address {
					addrVar.AssignedValue = assignmentValue
				}
				if su_ == Pointer {
					ptrVar.AssignedValue = assignmentValue
				}
				if su_ == Reference {
					refVar.AssignedValue = assignmentValue
				}
				if su_ == Array {
					arrVar.AssignedValue = assignmentValue
				}

			} else if p.previousToken(1).Kind == scanlex.ASSIGNMENT {
				if !allowinit {
					err := p.errorObj(nil, "Initialization not allowed")
					p.addErr(err)
				}
				if p.currentToken().Value == "let" {
					tr := parse_let_binding(p, ddaps)
					assignmentValue = tr.Node.(ast.Expr)
				} else {
					expr := parse_expr(p, assignment, ddaps)
					assignmentValue = expr.Node.(ast.Expr)
				}
				var1.AssignedValue = assignmentValue
				vd.HasInitValue = true
			}
		} else if explicitType == nil {
			isexplicitType = false
		}
		if type_ == "var" || type_ == "field" || type_ == "let" || type_ == "auto" || type_ == "fovar" {
			_, err_ := p.expectAny(scanlex.SEMI_COLON, scanlex.COMMA)
			p.addErr(err_)
		} else if type_ == "adhoc" {

		} else {
			_, err_ := p.expectAnyWoAdv(scanlex.CLOSE_PAREN, scanlex.COMMA)
			p.addErr(err_)

		}
		if isConstant && assignmentValue == nil {
			if type_ == "param" {
				err_ := p.errorObj(nil, "Optional Parameters succeeded by non optional parameter ")
				p.addErr(err_)

			} else if deduceType || isDyanamicRedec {
				err_ := p.errorObj(nil, "Cannot declare let/auto variable without providing default value.")
				p.addErr(err_)

			} else {
				err_ := p.errorObj(nil, "Cannot define constant variable without providing default value.")
				p.addErr(err_)
			}
		}

		var1.AssignedValue = assignmentValue
		p.IsPrevDeclSymbol = true
		p.ChangeCtx = false
		pr.OtherDetails = map[string]any{
			"name":            var1.Identifier,
			"internalnonamme": allowNoName,
		}
		vd.Optional = optionalParam
		vd.NamedArg = namedArgs
		vd.ThunkVar = thunkArg
		vd.VariadicParam = variadicParam
		var1.VarType = GetVarType(varType)
		if heapallocref {
			heapAllRef.Symb.SymbolType_ = string(symboltable.S_ReferenceSymbol)
			pr.Node = heapAllRef
		} else if thunk {
			thunkVar.Symb.SymbolType_ = string(symboltable.S_ThunkSymbol)
			pr.Node = thunkVar

		} else if rangeVar != (ast.RangeVariableDeclStmt{}) {
			rangeVar.Symb.SymbolType_ = string(symboltable.S_RangeSymbol)
			pr.Node = rangeVar

		} else if sliceVar != (ast.SliceVariableDeclStmt{}) {
			sliceVar.Symb.SymbolType_ = string(symboltable.S_ArraySymbol)
			pr.Node = sliceVar
		} else if addrVar != (ast.AddressVariableDeclStmt{}) {
			addrVar.Symb.SymbolType_ = string(symboltable.S_AddressSymbol)
			pr.Node = addrVar

		} else if !reflect.ValueOf(arrVar).IsZero() {

			if !arrVar.Symb.SizeFromInit {
				if arrVar.AssignedValue == nil {
					err_ := p.errorExpection("Array initialization not provided for "+strings.ReplaceAll(arrVar.Identifier, "_fo", ""), helpers.InvalidSyntax)
					p.addErr(err_)
				}
			}
			arrVar.Symb.SymbolType_ = string(symboltable.S_ArraySymbol)
			pr.Node = arrVar
		} else if !reflect.ValueOf(ptrVar).IsZero() {
			ptrVar.Symb.SymbolType_ = string(symboltable.S_PointerSymbol)
			pr.Node = ptrVar

		} else if !reflect.ValueOf(refVar).IsZero() {
			refVar.Symb.SymbolType_ = string(symboltable.S_ReferenceSymbol)
			pr.Node = refVar
		} else {

			symb := symboltable.VarSymbol{
				VariableDetails: vd,
			}

			var_ := ast.VarDeclarationStmt{
				BasicVarStmt: var1,
				Symb:         &symb,
			}
			var_.Symb.SymbolType_ = string(symboltable.S_VarSymbol)
			pr.Node = var_
		}
	} else {
		symb := symboltable.VarSymbol{
			VariableDetails: vd,
		}
		var_ := ast.VarDeclarationStmt{
			BasicVarStmt: var1,
			Symb:         &symb,
		}
		pr.Node = var_
		//adhoc symbols like in for each or else where
	}
	updateContext(p, pr.Node, false, redeclarable)
	return pr
}

// maker_array parses an array type declaration including static, dynamic,
// jagged, multi-dimensional, zero-dimensional, and VLA/FAM array variants.
func maker_array(p *parser, var1 *ast.BasicVarStmt, vd *symboltable.VariableDetails, ddaps map[scanlex.DirectiveKind][]ast.Stmt) ParseResult {
	defer p.traceCurrent()()

	pr := ParseResult{
		Node:   nil,
		Errors: []helpers.ErrorInterface{},
	}
	p.advance()
	subtype := "co.lang.0d.arr"
	multi := false
	jagged := false
	isZeroLen := false
	arguments := make([]ast.Expr, 0)
	isZeroD := false
	isELeLength := true
	isDynamic := false
	isRaw := true
	if p.currentToken().Kind == scanlex.CLOSE_BRACKET {
		isELeLength = false
		subtype = "co.lang.arr"
		_, err_ := p.expect(scanlex.CLOSE_BRACKET)
		p.addErr(err_)
	} else {
		isELeLength = true
		for p.hasTokens() {

			if p.currentToken().Kind == scanlex.NUMBER {
				for p.hasTokens() && p.currentTokenKind() != scanlex.CLOSE_BRACKET {
					expr := parse_expr(p, assignment, ddaps)
					arguments = append(arguments, expr.Node.(ast.Expr))

					if !p.currentToken().IsOneOfMany(scanlex.EOF, scanlex.CLOSE_BRACKET) {
						_, err_ := p.expect(scanlex.COMMA)
						p.addErr(err_)
						multi = true
					}
				}
				_, err_ := p.expect(scanlex.CLOSE_BRACKET)
				p.addErr(err_)
				if p.currentToken().Kind != scanlex.OPEN_BRACKET {
					break
				}
				jagged = true
				p.advance()
			} else if p.currentTokenKind() == scanlex.DOT_DOT_DOT {
				for p.hasTokens() && p.currentTokenKind() != scanlex.CLOSE_BRACKET {
					stmt1 := ast.StatementExpr{}
					p.advance()
					arguments = append(arguments, stmt1)

					if !p.currentToken().IsOneOfMany(scanlex.EOF, scanlex.CLOSE_BRACKET) {
						_, err_ := p.expect(scanlex.COMMA)
						p.addErr(err_)
						multi = true
					}
				}
				_, err_ := p.expect(scanlex.CLOSE_BRACKET)
				p.addErr(err_)
				if p.currentToken().Kind != scanlex.OPEN_BRACKET {
					break
				}
				jagged = true
				isDynamic = true
				p.advance()
			} else if p.currentToken().Kind == scanlex.DOT {
				p.advance()

				_, err_ := p.expect(scanlex.CLOSE_BRACKET)
				p.addErr(err_)
				subtype = "co.lang.0d.arr"
				isZeroD = true
			} else {
				err := "Syntax Error!!!"
				err_ := p.errorObj(nil, err)
				p.addErr(err_)
			}
		}
		if multi && jagged {
			err := "Currently mixing of arrays jagged and multidimension not supported"
			err_ := p.errorObj(nil, err)
			p.addErr(err_)
		}
		if len(arguments) == 1 {
			if il, ok := arguments[0].(ast.IntegerLiteral); ok && il.Value == 0 {
				isZeroLen = true
				subtype = "co.lang.0len.arr"
			} else {
				subtype = "co.lang.arr"
			}

		} else if len(arguments) == 2 && multi && !jagged {
			subtype = "co.lang.mult.dbl.arr"
		} else if len(arguments) == 2 && !multi && jagged {
			subtype = "co.lang.jag.dbl.arr"
		} else if len(arguments) == 3 && multi && !jagged {
			subtype = "co.lang.mult.tpl.arr"
		} else if len(arguments) == 3 && !multi && jagged {
			subtype = "co.lang.jag.tpl.arr"
		} else {
			err_ := p.errorExpection("Unsupported", helpers.UnSupported)
			p.addErr(err_)
		}

	}
	isELeLength = len(arguments) > 0
	vd.SubType_ = subtype
	vd.SubID = "ARR"
	ars := symboltable.ArraySymbol{
		VariableDetails: *vd,
		IsZeroDim:       isZeroD,
		IsJagged:        jagged,
		IsMultiDimesion: multi,
		ElementLenDecl:  isELeLength,
		IsDynamic:       false,
		IsZeroLen:       isZeroLen,
		IsRawArray:      isRaw,
		IsSlice:         false,
		VLA:             isDynamic,
	}

	pr.Node = ast.ArrayVariableDeclStmt{
		BasicVarStmt: *var1,
		Dimensions:   len(arguments),
		Sizes:        arguments,
		Symb:         &ars,
	}
	return pr

}

// parse_walrus_decl_impl handles `x := value` (walrus) and `x ?= value` (redeclare/optional-assign).
// These are type-inferred variable declarations whose type is deduced from the assigned value.
func parse_walrus_decl_impl(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt, isRedeclare bool) ParseResult {
	defer p.traceCurrent()()

	pr := ParseResult{Node: nil, Errors: []helpers.ErrorInterface{}}

	if !p.IsPrevDeclSymbol {
		currSymbolTable := p.SymbolTable_
		p.SymbolTable_ = createNewSymbolTable(p.Context_.Id, p.SymbolTable_.Id)
		p.SymbolTable_.Prev = currSymbolTable.Id
		currSymbolTable.Next = p.SymbolTable_.Id

		p.Fs.AddSymbolTable(p.SymbolTable_)
	}

	symbolName, err_ := p.expectError(scanlex.IDENTIFIER,
		fmt.Sprintf("Expected variable name, got %s", scanlex.TokenKindString(p.currentTokenKind())))
	p.addErr(err_)

	if isRedeclare {
		_, err_ = p.expect(scanlex.QEQ)
	} else {
		_, err_ = p.expect(scanlex.WALRUS)
	}
	p.addErr(err_)

	var assignmentValue ast.Expr
	if p.currentToken().Value == "let" {
		tr := parse_let_binding(p, ddaps)
		assignmentValue = tr.Node.(ast.Expr)
	} else if p.currentToken().Value == "for" && isForComprehension(p) {
		tr := parse_for_comprehension_expr(p, ddaps)
		assignmentValue = tr.Node.(ast.Expr)
	} else {
		expr := parse_expr(p, assignment, ddaps)
		assignmentValue = expr.Node.(ast.Expr)
	}

	_, err_ = p.expectAny(scanlex.SEMI_COLON, scanlex.COMMA)
	p.addErr(err_)

	sd := symboltable.SymbolDetails{
		Name_:       symbolName.Value,
		SymbolType_: string(symboltable.S_VarSymbol),
	}
	vd := symboltable.VariableDetails{
		IsSealed:      false,
		SymbolDetails: sd,
		ExplicitType:  false,
		Inferred:      true,
		HasInitValue:  true,
		SubType_:      "co.lang.var",
		SubID:         "WALRUS_OR_REDCL",
		VarType:       GetVarType(VAR),
	}
	symb := symboltable.VarSymbol{
		VariableDetails: vd,
	}
	var1 := ast.BasicVarStmt{
		AssignedValue: assignmentValue,
		Identifier:    symbolName.Value,
	}
	var_ := ast.VarDeclarationStmt{
		BasicVarStmt: var1,
		Symb:         &symb,
	}
	var_.Symb.SymbolType_ = string(symboltable.S_VarSymbol)
	if _, ok := isAnnDec(ddaps, "@co.dap.immutable"); ok {
		vd.IsSealed = true
	}
	p.IsPrevDeclSymbol = true
	p.ChangeCtx = false

	pr.Node = var_
	pr.OtherDetails = map[string]any{
		"name":            var1.Identifier,
		"internalnonamme": false,
	}
	updateContext(p, pr.Node, false, isRedeclare)
	return pr
}

// parse_range_decl_impl handles `x := 1..10` and variants.
// Range variable declarations use type inference and the assigned value is a RangeExpr.
func parse_range_decl_impl(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt, isRedeclare bool) ParseResult {
	defer p.traceCurrent()()

	pr := ParseResult{Node: nil, Errors: []helpers.ErrorInterface{}}

	if !p.IsPrevDeclSymbol {
		currSymbolTable := p.SymbolTable_
		p.SymbolTable_ = createNewSymbolTable(p.Context_.Id, p.SymbolTable_.Id)
		p.SymbolTable_.Prev = currSymbolTable.Id
		currSymbolTable.Next = p.SymbolTable_.Id
		p.Fs.AddSymbolTable(p.SymbolTable_)
	}

	symbolName, err_ := p.expectError(scanlex.IDENTIFIER,
		fmt.Sprintf("Expected variable name, got %s", scanlex.TokenKindString(p.currentTokenKind())))
	p.addErr(err_)

	// Range declarations always use := (WALRUS)
	_, err_ = p.expect(scanlex.WALRUS)
	p.addErr(err_)

	expr := parse_expr(p, assignment, ddaps)
	assignmentValue := expr.Node.(ast.Expr)

	_, err_ = p.expectAny(scanlex.SEMI_COLON, scanlex.COMMA)
	p.addErr(err_)

	sd := symboltable.SymbolDetails{
		Name_:       symbolName.Value,
		SymbolType_: string(symboltable.S_VarSymbol),
	}
	vd := symboltable.VariableDetails{
		IsSealed:      false,
		SymbolDetails: sd,
		ExplicitType:  false,
		Inferred:      true,
		HasInitValue:  true,
		SubType_:      "co.lang.range",
		SubID:         "range",
		VarType:       GetVarType(VAR),
	}
	symb := symboltable.VarSymbol{
		VariableDetails: vd,
	}
	var1 := ast.BasicVarStmt{
		AssignedValue: assignmentValue,
		Identifier:    symbolName.Value,
	}
	var_ := ast.VarDeclarationStmt{
		Symb:         &symb,
		BasicVarStmt: var1,
	}
	var_.Symb.SymbolType_ = string(symboltable.S_RangeSymbol)
	p.IsPrevDeclSymbol = true
	p.ChangeCtx = false

	pr.Node = var_
	pr.OtherDetails = map[string]any{
		"name":            var1.Identifier,
		"internalnonamme": false,
	}
	updateContext(p, pr.Node, false, false)
	return pr
}

// parse_pointer_meta parses the meta={key:value, ...} block inside a fat
// pointer declaration, returning the key-value pairs as a map.
func parse_pointer_meta(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) map[string]any {
	defer p.traceCurrent()()

	meta := make(map[string]any)
	p.advance() //meta eaten
	_, err := p.expect(scanlex.ASSIGNMENT)
	p.addErr(err)
	_, err = p.expect(scanlex.OPEN_CURLY)
	p.addErr(err)
	for p.currentTokenKind() != scanlex.CLOSE_CURLY && p.hasTokens() {
		keyTok, err_ := p.expectError(scanlex.IDENTIFIER, "Expected meta key")
		p.addErr(err_)
		_, err = p.expect(scanlex.COLON)
		p.addErr(err)
		valueTok, err_ := p.expectAny(scanlex.IDENTIFIER, scanlex.COMPOSITE_IDENTIFER)
		p.addErr(err_)
		if p.currentToken().Kind == scanlex.ARROW {
			p.advance()
			_, err_ = p.expect(scanlex.OPEN_PAREN)
			p.addErr(err_)
			_, err_ = p.expect(scanlex.STAR)
			p.addErr(err_)
			_, err_ = p.expect(scanlex.CLOSE_PAREN)
			p.addErr(err_)

		}

		meta[keyTok.Value] = valueTok.Value
		if p.currentTokenKind() != scanlex.CLOSE_CURLY {
			_, err = p.expect(scanlex.COMMA)
			p.addErr(err)
		}
	}
	_, err = p.expect(scanlex.CLOSE_CURLY)
	p.addErr(err)
	return meta
}
