package scanlex

import (
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/samkrao/fo-lang/frontend/src/foerrors"
	"github.com/samkrao/fo-lang/frontend/src/helpers"
)

type lexer struct {
	fn         string
	custom     *CustomOperators
	quiet      bool
	Tokens     []Token
	source     string
	sourcearr  []string
	pos        int
	line       int
	currentPos int
	col        int
	posi       *helpers.Position
}

// utf8BOM is the U+FEFF byte-order mark in its UTF-8 encoding.
var utf8BOM = string(rune(0xFEFF))

// Tokenize lexes the given source string into a slice of Tokens, performing folding and cleanup.
func Tokenize(source string, fn string) []Token {
	return TokenizeWith(source, fn, nil)
}

// TokenizeQuiet lexes source without reporting diagnostics. It supports
// best-effort project surface scans whose callers report any relevant errors
// during their authoritative parse.
func TokenizeQuiet(source string, fn string) []Token {
	return tokenize(source, fn, nil, true)
}

// TokenizeWith lexes source with the user-defined symbols in a project operator catalog.
//
// A custom operator cannot be recognised from one source file alone: a symbol may be
// declared elsewhere in the project. The caller therefore supplies the precomputed
// project operator catalog. Semantic name resolution remains responsible for deciding
// whether a catalogued spelling is visible at a use site. Tokenize is the same call with
// no custom operators, which consumers that do not need project operators use.
func TokenizeWith(source string, fn string, custom *CustomOperators) []Token {
	return tokenize(source, fn, custom, false)
}

// tokenize is the scanning loop shared by the exported entry points.
func tokenize(source string, fn string, custom *CustomOperators, quiet bool) []Token {
	if !quiet {
		if err := validateSourceEncoding(source, fn); err != nil {
			foerrors.HandleErrors(err)
			return nil
		}
	}

	// DECISION-LEX-001: a U+FEFF byte-order mark is permitted only as the first code
	// point. Removing it here accepts the editors that emit one; anywhere else the
	// character has already been rejected by the complete-source validation above.
	source = strings.TrimPrefix(source, utf8BOM)

	lex := createLexer(source, fn)
	lex.custom = custom
	lex.quiet = quiet

	for !lex.at_eof() {
		if length, message, unsupported := detectUnsupportedAlphaLiteral(lex.remainder()); unsupported {
			if lex.quiet {
				lex.advanceN(length)
				continue
			}
			rejectUnsupportedAlphaLiteral(lex, length, message)
			continue
		}

		src := lex.remainder()
		result, ok := lex.scanToken(src)
		if !ok {
			if lex.quiet {
				lex.advanceN(1)
				continue
			}
			err_ := lex.errorObj(nil, fmt.Sprintf("lexer error: unrecognized token near '%v'", src))
			foerrors.HandleErrors(err_)
			// HandleErrors returns when diagnostics are collected rather than
			// fatal, so the offending byte is consumed to guarantee progress.
			lex.advanceN(1)
			continue
		}

		switch result.action {
		case actionNewline:
			lex.emitNewline(src[:result.length])
		case actionSkip:
			lex.advanceN(result.length)
			if result.lines > 0 {
				lex.advanceline(result.lines)
				lex.col = result.endColumn
			}
		case actionError:
			start := helpers.NewPosition(lex.pos, lex.line, lex.col, lex.pos, lex.fn, lex.currentLineText(), false)
			lex.advanceN(result.length)
			if result.lines > 0 {
				lex.advanceline(result.lines)
				lex.col = result.endColumn
			}
			end := helpers.NewPosition(lex.pos, lex.line, lex.col, lex.pos, lex.fn, lex.currentLineText(), false)
			if !lex.quiet {
				foerrors.HandleErrors(lex.errorException(result.message, result.errType, *start, *end))
			}
		default:
			lex.emitToken(result.kind, src[:result.length])
		}
	}
	startPos := helpers.NewPosition(1, 0, 1, 0, "", "", false)
	endPos := helpers.NewPosition(1, 0, 1, 0, "", "", false)
	lex.push(newUniqueToken(EOF, "EOF", startPos, endPos))
	cleanupLB(lex)
	foldTokens(lex)
	return lex.Tokens
}

// validateSourceEncoding checks the complete byte stream before comments and
// literals can hide malformed input from ordinary token recognition. A single
// leading BOM is metadata and is not counted as a source column; every later
// U+FEFF is a lexical error, including one immediately following that BOM.
func validateSourceEncoding(source, fn string) helpers.ErrorInterface {
	offset := 0
	lineStart := 0
	if strings.HasPrefix(source, utf8BOM) {
		offset = len(utf8BOM)
		lineStart = offset
	}

	line, column := 1, 1
	for offset < len(source) {
		r, size := utf8.DecodeRuneInString(source[offset:])
		if r == utf8.RuneError && size == 1 {
			return sourceEncodingError(
				source, fn, offset, lineStart, line, column, 1,
				fmt.Sprintf("invalid UTF-8 byte sequence at byte %d", offset+1),
			)
		}
		if r == '\uFEFF' {
			return sourceEncodingError(
				source, fn, offset, lineStart, line, column, size,
				fmt.Sprintf("U+FEFF is permitted only once as an optional leading byte-order mark (found at byte %d)", offset+1),
			)
		}

		offset += size
		switch r {
		case '\r':
			if offset < len(source) && source[offset] == '\n' {
				offset++
			}
			line++
			column = 1
			lineStart = offset
		case '\n':
			line++
			column = 1
			lineStart = offset
		default:
			// Scanner positions are byte-oriented, so diagnostic columns follow
			// the same convention and continue to align with source slices.
			column += size
		}
	}
	return nil
}

func sourceEncodingError(source, fn string, offset, lineStart, line, column, width int, message string) helpers.ErrorInterface {
	lineEnd := lineStart
	for lineEnd < len(source) && source[lineEnd] != '\r' && source[lineEnd] != '\n' {
		lineEnd++
	}
	lineText := source[lineStart:lineEnd]
	start := helpers.NewPosition(offset, line, column, offset, fn, lineText, false)
	end := helpers.NewPosition(offset+width, line, column+width, offset+width, fn, lineText, false)
	return helpers.NewInvalidSyntaxError(*start, *end, message)
}
func cleanupLB(lex *lexer) []Token {
	nTokens := make([]Token, 0)
	for {

		if lex.isEof() {
			break
		}

		Token_ := lex.currentToken()
		if Token_.Kind != NEWLINE {

			nTokens = append(nTokens, Token_)
		}
		lex.moveNext()
	}
	lex.resetCurrent()
	lex.Tokens = nTokens
	return nTokens
}
func foldTokens(lex *lexer) []Token {
	nTokens := make([]Token, 0)
	var tempToken = ""
	changed := false
	lastToken := ""
	for {
		if lex.isEof() {
			break
		}

		Token_ := lex.currentToken()

		tempToken = Token_.Value
		//Token_.Println()

		lastToken = ""
		length := 1
		lstTokens := []Token{}
		lstTokens = append(lstTokens, Token_)
		if Token_.Kind == IDENTIFIER || Token_.Kind == KEYWORD || Token_.Kind == RESERVEDWORD || Token_.Kind == CONTEXT_KEYWORD || Token_.Kind == ATDAP {

			/*
				if Token_.Kind == KEYWORD || Token_.Kind == RESERVEDWORD {
					if _, ok := Operator_details[lex.lookBack(1).Kind]; ok {
						if !Contains(AllowedOps, lex.lookBack(1).Kind) {
							tTok := newUniqueToken(NONKEYRESERVEDWORD, "****", Token_.StartPos, Token_.EndPos)
							err_ := lex.errorObj(&tTok, "Operator should not precede KeyWord/ReservedWord ")
							foerrors.HandleErrors(err_)
						}

					}
				}
			*/
			if Token_.Kind == KEYWORD || Token_.Kind == RESERVEDWORD || Token_.Kind == CONTEXT_KEYWORD {
				if lex.lookAhead(1).Kind == DOT {
					check := false
					for _, val := range UnsupportedObjects {
						if Token_.Value == val {
							check = true
							break
						}
					}
					if arr, ok := KeyWords_me[Token_.Value]; ok && check {
						for _, v := range arr {
							if lex.lookAhead(2).Value == v {
								err_ := lex.errorObj(&Token_, "Ah. eventhough it is a valid statement currently we are not supporting it please use decorator kind or type kind things ")
								foerrors.HandleErrors(err_)
							} else {
								err_ := lex.errorObj(&Token_, "Invalid method/stmt after KeyWord/ReservedWord ")
								foerrors.HandleErrors(err_)

							}
						}
					}
				}
			}
			for lex.lookAhead(1).Kind == DOT {

				if lex.lookAhead(2).Kind == IDENTIFIER || lex.lookAhead(2).Kind == KEYWORD || lex.lookAhead(2).Kind == RESERVEDWORD || lex.lookAhead(2).Kind == CONTEXT_KEYWORD || lex.lookAhead(2).Kind == BUILT_IN_METHOD {
					// adding dot and advancing
					lex.moveNext()
					lstTokens = append(lstTokens, lex.currentToken())
					tempToken = tempToken + lex.currentToken().Value
					changed = true
					lex.moveNext()
					lstTokens = append(lstTokens, lex.currentToken())
					lastToken = lex.currentToken().Value
					tempToken = tempToken + lastToken
					length = length + 1
				} else {
					break
				}

			}

		}

		if changed && selfCallChainNeedsSeparation(lstTokens, tempToken, lex.lookAhead(1).Kind == OPEN_PAREN) {
			// `this` and `self` remain self-reference primaries when they are call
			// receivers. Their special return-statement spellings stay folded.
			nTokens = appendSeparatedMemberChain(nTokens, lstTokens, true)
		} else if changed && dottedChainFollowsCompletedExpression(lex, len(lstTokens)) {
			// A chain after a completed receiver is postfix structure, not a
			// qualified name. Preserve every source dot so
			// `factory().service.worker.run()` becomes three MemberExpr suffixes.
			nTokens = appendSeparatedMemberChain(nTokens, lstTokens, lex.lookAhead(1).Kind == OPEN_PAREN)
		} else if changed {
			dirTok := tempToken
			//dirTok = strings.TrimPrefix(dirTok, "@")
			if _, ok := Built_in_directives(dirTok); ok && strings.HasPrefix(tempToken, "@") {
				nTokens = append(nTokens, newUniqueToken(BUILT_IN_DIRECTIVES, tempToken, lstTokens[0].StartPos.Copy(), lstTokens[len(lstTokens)-3].EndPos.Copy()))
			} else if strings.HasPrefix(tempToken, "@") {
				nTokens = append(nTokens, newUniqueToken(CUSTOM_DIRECTIVES, tempToken, lstTokens[0].StartPos.Copy(), lstTokens[len(lstTokens)-3].EndPos.Copy()))

			} else if tempToken == "co.lang.operator" {
				nTokens = append(nTokens, newUniqueToken(OPERATOR_SOURCE_KIND, tempToken, lstTokens[0].StartPos.Copy(), lstTokens[len(lstTokens)-1].EndPos.Copy()))
			} else if slices.Contains(Builtin_Kinds, tempToken) {
				nTokens = append(nTokens, newUniqueToken(BUILT_IN_KIND, tempToken, lstTokens[0].StartPos.Copy(), lstTokens[len(lstTokens)-1].EndPos.Copy()))
			} else if slices.Contains(Builtin_types, tempToken) {
				nTokens = append(nTokens, newUniqueToken(BUILT_IN_TYPE, tempToken, lstTokens[0].StartPos.Copy(), lstTokens[len(lstTokens)-1].EndPos.Copy()))

			} else if _, ok := Built_in_constants[tempToken]; ok {
				nTokens = append(nTokens, newUniqueToken(BUILT_IN_CONSTANTS, tempToken, lstTokens[0].StartPos.Copy(), lstTokens[len(lstTokens)-1].EndPos.Copy()))

			} else if _, ok := Built_in_stmt_exprs[tempToken]; ok {
				// Prefer the complete registered namespace over the shorter root.
				// In particular, co.sys.file is a receiver in its own right.
				nTokens = append(nTokens, newUniqueToken(BUIL_IN_STMT_EXPRS, tempToken, lstTokens[0].StartPos.Copy(), lstTokens[len(lstTokens)-1].EndPos.Copy()))

			} else if _, ok := Built_in_stmt_exprs[Token_.Value]; ok {
				otherFlag := checkBuiltInStExmet(Token_, tempToken, lastToken)
				// A dotted invocation always keeps its receiver, dot and method as
				// separate tokens. Previously this branch folded an ordinary method on
				// a built-in/keyword receiver (for example `this.custom()`) into one
				// BUIL_IN_STMT_EXPRS token, while a Reserved_me spelling such as
				// `this.map()` was split. That made the parse-tree shape depend on the
				// method registry rather than on the source syntax.
				//
				// SpecialBuiltins are statement spellings, not calls. In particular,
				// `this.return (value);` has an OPEN_PAREN after the folded path but the
				// parenthesis begins the returned expression, so it must remain whole.
				if lex.lookAhead(1).Kind == OPEN_PAREN && !slices.Contains(SpecialBuiltins, tempToken) {
					receiver := strings.TrimSuffix(tempToken, "."+lastToken)
					receiverEnd := lstTokens[len(lstTokens)-3].EndPos.Copy()

					if separated, ok := appendLongestBuiltInReceiver(nTokens, lstTokens); ok {
						nTokens = separated
					} else if !otherFlag {
						nTokens = append(nTokens, newUniqueToken(
							BUIL_IN_STMT_EXPRS,
							receiver,
							lstTokens[0].StartPos.Copy(),
							receiverEnd,
						))
					} else if length-1 > 1 {
						receiver = strings.ReplaceAll(receiver, ".", "_fo.")
						nTokens = append(nTokens, newUniqueToken(
							COMPOSITE_IDENTIFER,
							receiver,
							lstTokens[0].StartPos.Copy(),
							receiverEnd,
						))
					} else {
						receiver += "_fo"
						nTokens = append(nTokens, newUniqueToken(
							IDENTIFIER,
							receiver,
							lstTokens[0].StartPos.Copy(),
							receiverEnd,
						))
					}

					nTokens = append(nTokens, newUniqueToken(
						DOT,
						".",
						lstTokens[len(lstTokens)-2].StartPos.Copy(),
						lstTokens[len(lstTokens)-2].EndPos.Copy(),
					))

					methodKind := METHOD_CALL
					methodValue := lastToken + "_fo"
					if IsReservedMethod(lastToken) {
						methodKind = BUILT_IN_METHOD
						methodValue = lastToken
					}
					nTokens = append(nTokens, newUniqueToken(
						methodKind,
						methodValue,
						lstTokens[len(lstTokens)-1].StartPos.Copy(),
						lstTokens[len(lstTokens)-1].EndPos.Copy(),
					))
				} else if !otherFlag {
					rmethod := false
					var nTempToken = tempToken
					if slices.Contains(SpecialBuiltins, tempToken) {
						rmethod = false
					} else if IsReservedMethod(lastToken) {
						rmethod = true
						nTempToken = strings.Replace(tempToken, "."+lastToken, "", 1)
					} else {
						rmethod = false
					}
					nTokens = append(nTokens, newUniqueToken(BUIL_IN_STMT_EXPRS, nTempToken, lstTokens[0].StartPos.Copy(), lstTokens[len(lstTokens)-1].EndPos.Copy()))

					if rmethod {
						nTokens = append(nTokens, newUniqueToken(DOT, ".", lstTokens[len(lstTokens)-2].StartPos.Copy(), lstTokens[len(lstTokens)-2].EndPos.Copy()))
						nTokens = append(nTokens, newUniqueToken(BUILT_IN_METHOD, lastToken, lstTokens[len(lstTokens)-1].StartPos.Copy(), lstTokens[len(lstTokens)-1].EndPos.Copy()))
					}
				} else {
					if length > 1 {
						tempToken = strings.ReplaceAll(tempToken, ".", "_fo.")
						nTokens = append(nTokens, newUniqueToken(COMPOSITE_IDENTIFER, tempToken, lstTokens[0].StartPos.Copy(), lstTokens[len(lstTokens)-1].EndPos.Copy()))
					} else {
						nTokens = append(nTokens, Token_)
					}
				}
			} else if IsReservedMethod(lastToken) {

				var nTempToken = strings.Replace(tempToken, "."+lastToken, "", 1)
				dirTok := nTempToken
				dirTok = strings.TrimPrefix(dirTok, "@")
				if _, ok := Built_in_directives(dirTok); ok && strings.HasPrefix(nTempToken, "@") {
					nTokens = append(nTokens, newUniqueToken(BUILT_IN_DIRECTIVES, nTempToken, lstTokens[0].StartPos.Copy(), lstTokens[len(lstTokens)-3].EndPos.Copy()))
				} else if strings.HasPrefix(nTempToken, "@") {
					nTokens = append(nTokens, newUniqueToken(CUSTOM_DIRECTIVES, nTempToken, lstTokens[0].StartPos.Copy(), lstTokens[len(lstTokens)-3].EndPos.Copy()))

				} else if slices.Contains(Builtin_Kinds, tempToken) {
					nTokens = append(nTokens, newUniqueToken(BUILT_IN_KIND, nTempToken, lstTokens[0].StartPos.Copy(), lstTokens[len(lstTokens)-3].EndPos.Copy()))
				} else if slices.Contains(Builtin_types, tempToken) {
					nTokens = append(nTokens, newUniqueToken(BUILT_IN_TYPE, nTempToken, lstTokens[0].StartPos.Copy(), lstTokens[len(lstTokens)-3].EndPos.Copy()))
				} else if _, ok := Built_in_constants[nTempToken]; ok {
					nTokens = append(nTokens, newUniqueToken(BUILT_IN_CONSTANTS, nTempToken, lstTokens[0].StartPos.Copy(), lstTokens[len(lstTokens)-3].EndPos.Copy()))

				} else if lex.lookAhead(1).Kind == OPEN_PAREN && length-1 == 1 {
					nTempToken += "_fo"
					nTokens = append(nTokens, newUniqueToken(IDENTIFIER, nTempToken, lstTokens[0].StartPos.Copy(), lstTokens[len(lstTokens)-3].EndPos.Copy()))
				} else if length > 1 {
					nTempToken = strings.ReplaceAll(nTempToken, ".", "_fo.")
					nTokens = append(nTokens, newUniqueToken(COMPOSITE_IDENTIFER, nTempToken, lstTokens[0].StartPos.Copy(), lstTokens[len(lstTokens)-3].EndPos.Copy()))
				} else {
					nTokens = append(nTokens, Token_)
				}
				nTokens = append(nTokens, newUniqueToken(DOT, ".", lstTokens[len(lstTokens)-2].StartPos.Copy(), lstTokens[len(lstTokens)-2].EndPos.Copy()))
				nTokens = append(nTokens, newUniqueToken(BUILT_IN_METHOD, lastToken, lstTokens[len(lstTokens)-1].StartPos.Copy(), lstTokens[len(lstTokens)-1].EndPos.Copy()))
			} else {
				if lex.lookAhead(1).Kind == OPEN_PAREN {
					var nTempToken = strings.Replace(tempToken, "."+lastToken, "", 1)
					length = length - 1
					if length > 1 {
						nTempToken = strings.ReplaceAll(nTempToken, ".", "_fo.")
						nTokens = append(nTokens, newUniqueToken(COMPOSITE_IDENTIFER, nTempToken, lstTokens[0].StartPos.Copy(), lstTokens[len(lstTokens)-3].EndPos.Copy()))
					} else {
						nTempToken = nTempToken + "_fo"
						nTokens = append(nTokens, newUniqueToken(IDENTIFIER, nTempToken, lstTokens[0].StartPos.Copy(), lstTokens[len(lstTokens)-3].EndPos.Copy()))

					}
					nTokens = append(nTokens, newUniqueToken(DOT, ".", lstTokens[len(lstTokens)-2].StartPos.Copy(), lstTokens[len(lstTokens)-2].EndPos.Copy()))
					lastToken = lastToken + "_fo"
					nTokens = append(nTokens, newUniqueToken(METHOD_CALL, lastToken, lstTokens[len(lstTokens)-1].StartPos.Copy(), lstTokens[len(lstTokens)-1].EndPos.Copy()))

				} else {
					tempToken = strings.ReplaceAll(tempToken, ".", "_fo.")
					nTokens = append(nTokens, newUniqueToken(COMPOSITE_IDENTIFER, tempToken, lstTokens[0].StartPos.Copy(), lstTokens[len(lstTokens)-1].EndPos.Copy()))
				}
			}

		} else {
			// A receiver ending in `)` or `]` is completed before its following
			// `.method(` tail is visited, so that tail cannot participate in the
			// forward dotted-name fold above. Classify it here from its immediate
			// token context to give `factory().work()` the same method token as
			// `factory.work()`.
			if lex.lookBack(1).Kind == DOT && lex.lookAhead(1).Kind == OPEN_PAREN && IsReservedMethod(Token_.Value) {
				Token_.Kind = BUILT_IN_METHOD
				nTokens = append(nTokens, Token_)
			} else if lex.lookBack(1).Kind == DOT && lex.lookAhead(1).Kind == OPEN_PAREN && Token_.Kind == IDENTIFIER {
				Token_.Kind = METHOD_CALL
				Token_.Value += "_fo"
				nTokens = append(nTokens, Token_)
			} else if Token_.Kind == IDENTIFIER {
				Token_.Value = Token_.Value + "_fo"
				nTokens = append(nTokens, Token_)
			} else {
				nTokens = append(nTokens, Token_)
			}
		}
		changed = false
		lex.moveNext()

	}
	lex.Tokens = nTokens
	// tks_b, _ := json.Marshal(lex.Tokens)
	// fmt.Println(string(tks_b[:]))
	return nTokens
}

// classifyBuiltInName applies the main folding precedence to a complete dotted
// receiver. Looking at the receiver rather than only its first segment preserves
// the longest registered prefix in co.sys.file.open() and co.const.true.to_str().
func classifyBuiltInName(name string) (TokenKind, bool) {
	if slices.Contains(Builtin_Kinds, name) {
		return BUILT_IN_KIND, true
	}
	if slices.Contains(Builtin_types, name) {
		return BUILT_IN_TYPE, true
	}
	if _, ok := Built_in_constants[name]; ok {
		return BUILT_IN_CONSTANTS, true
	}
	if _, ok := Built_in_stmt_exprs[name]; ok {
		return BUIL_IN_STMT_EXPRS, true
	}
	return EOF, false
}

// appendLongestBuiltInReceiver selects the longest registered prefix of the
// receiver portion of a dotted invocation. Any receiver segments after that
// prefix remain individual postfix members; the final gathered segment is the
// called method and is emitted by the caller.
func appendLongestBuiltInReceiver(out []Token, gathered []Token) ([]Token, bool) {
	receiverSegments := (len(gathered)+1)/2 - 1
	for count := receiverSegments; count > 0; count-- {
		parts := make([]string, 0, count)
		for segment := 0; segment < count; segment++ {
			parts = append(parts, gathered[segment*2].Value)
		}
		name := strings.Join(parts, ".")
		kind, ok := classifyBuiltInName(name)
		if !ok {
			continue
		}

		prefixEnd := gathered[(count-1)*2].EndPos.Copy()
		out = append(out, newUniqueToken(kind, name, gathered[0].StartPos.Copy(), prefixEnd))
		for segment := count; segment < receiverSegments; segment++ {
			dot := gathered[segment*2-1]
			out = append(out, newUniqueToken(DOT, ".", dot.StartPos.Copy(), dot.EndPos.Copy()))
			out = append(out, normalizedMemberToken(gathered[segment*2]))
		}
		return out, true
	}
	return out, false
}

// normalizedMemberToken applies identifier lowering to an individual member
// without changing contextual keyword kinds.
func normalizedMemberToken(segment Token) Token {
	if segment.Kind == IDENTIFIER {
		segment.Value += "_fo"
	}
	return segment
}

// dottedChainFollowsCompletedExpression reports whether gathered begins after a
// dot whose receiver was already completed by a closing delimiter. Every segment
// of such a tail is a postfix member suffix, not part of a qualified name.
func dottedChainFollowsCompletedExpression(lex *lexer, consumed int) bool {
	if lex.lookBack(consumed).Kind != DOT {
		return false
	}
	switch lex.lookBack(consumed + 1).Kind {
	case CLOSE_PAREN, CLOSE_BRACKET, CLOSE_CURLY:
		return true
	default:
		return false
	}
}

// selfCallChainNeedsSeparation keeps a self receiver visible to the parser. The
// return forms are statements whose established token contract is one folded
// BUIL_IN_STMT_EXPRS token, despite the following parenthesized return value.
func selfCallChainNeedsSeparation(gathered []Token, fullName string, invoked bool) bool {
	if !invoked || len(gathered) == 0 || slices.Contains(SpecialBuiltins, fullName) {
		return false
	}
	return gathered[0].Value == "this" || gathered[0].Value == "self"
}

// appendSeparatedMemberChain emits the gathered identifier/dot pairs without
// collapsing their member boundaries. The dot before the first item was emitted
// by the preceding iteration, so this function appends only internal dots.
func appendSeparatedMemberChain(out []Token, gathered []Token, invoked bool) []Token {
	lastSegment := len(gathered) - 1
	for i := 0; i < len(gathered); i += 2 {
		if i > 0 {
			dot := gathered[i-1]
			out = append(out, newUniqueToken(DOT, ".", dot.StartPos.Copy(), dot.EndPos.Copy()))
		}

		segment := gathered[i]
		if invoked && i == lastSegment {
			if IsReservedMethod(segment.Value) {
				segment.Kind = BUILT_IN_METHOD
			} else {
				segment.Kind = METHOD_CALL
				segment.Value += "_fo"
			}
		} else {
			segment = normalizedMemberToken(segment)
		}
		out = append(out, segment)
	}
	return out
}

func checkBuiltInStExmet(Token_ Token, tempToken string, lastToken string) bool {
	tks := strings.Split(tempToken, ".")
	otherFlag := false
	first := true

	nTk := Token_.Value
	for _, stk := range tks {
		if stmts, ok := Built_in_stmt_exprs[nTk]; ok {

			if first {
				first = false
				continue

			}
			if stk == lastToken {
				break
			} else if !slices.Contains(stmts, stk) {
				otherFlag = true
				break
			}
		} else {
			otherFlag = true
			break
		}

	}
	return otherFlag
}
func (lex *lexer) advanceN(n int) {
	lex.pos += n
	lex.col += n
}
func (lex *lexer) advanceline(n int) {
	lex.line += n
	//lex.pos += n
	lex.col = 0
}

func (lex *lexer) at() byte {
	return lex.source[lex.pos]
}

func (lex *lexer) advance() {
	lex.pos += 1
	lex.col += 1
}

func (lex *lexer) remainder() string {
	return lex.source[lex.pos:]
}
func (lex *lexer) lookAhead(n int) Token {
	pos := lex.currentPos + n
	if pos <= len(lex.Tokens)-1 {
		return lex.Tokens[pos]
	}
	return Token{}
}

func (lex *lexer) lookBack(n int) Token {
	pos := lex.currentPos - n
	if pos > 0 {
		return lex.Tokens[pos]
	}
	return Token{}
}
func (lex *lexer) moveNext() {
	if lex.currentPos == len(lex.Tokens) || lex.Tokens[lex.currentPos].Kind == EOF {
		lex.currentPos = lex.currentPos + 0
		return
	}
	lex.currentPos = lex.currentPos + 1
}
func (lex *lexer) movePrev() {
	if lex.currentPos == 0 {
		lex.currentPos = lex.currentPos - 0
		return
	}
	lex.currentPos = lex.currentPos - 1
}
func (lex *lexer) resetCurrent() {
	lex.currentPos = 0
}
func (lex *lexer) currentToken() Token {
	if lex.currentPos < len(lex.Tokens) {
		return lex.Tokens[lex.currentPos]
	} else {
		return DummyNode
	}
}
func (lex *lexer) isEof() bool {
	return lex.currentPos >= len(lex.Tokens) || lex.Tokens[lex.currentPos].Kind == EOF
}

func (lex *lexer) push(token Token) {
	lex.Tokens = append(lex.Tokens, token)
}

func (lex *lexer) at_eof() bool {
	return lex.pos >= len(lex.source)
}

// USE FSM instead of regex
func sourceLines(source string) []string {
	if source == "" {
		return nil
	}
	sar1 := strings.Split(source, "\r\n")

	var sar2 []string = []string{}
	if len(sar1) >= 1 {
		for _, line := range sar1 {
			sar2 = append(sar2, strings.Split(line, "\n\r")...)
		}
	}
	sar1 = []string{}
	if len(sar2) >= 1 {
		for _, line := range sar2 {
			sar1 = append(sar1, strings.Split(line, "\n")...)
		}
	}
	sar2 = []string{}
	if len(sar1) >= 1 {
		for _, line := range sar1 {
			sar2 = append(sar2, strings.Split(line, "\r")...)
		}
	}
	return sar2

}

func createLexer(source string, fn string) *lexer {
	return &lexer{
		pos:        0,
		line:       1,
		source:     source,
		sourcearr:  sourceLines(source),
		currentPos: 0,
		fn:         fn,
		col:        1,
		posi:       helpers.NewPosition(0, 1, 0, 0, fn, "", false),
		Tokens:     make([]Token, 0),
	}

}
