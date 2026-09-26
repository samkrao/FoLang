package scanlex

import (
	"slices"
	"strings"

	"github.com/samkrao/fo-lang/src/helpers"
)

// isSpecialBuiltin compares source and internally lowered identifier spellings.
// Folding may already have appended _fo to one or more path segments when this
// decision is made; that implementation suffix must not turn a statement form
// such as a language-owned dotted built-in into an ordinary method call.
func isSpecialBuiltin(name string) bool {
	logical := strings.ReplaceAll(name, "_fo.", ".")
	logical = strings.TrimSuffix(logical, "_fo")
	return slices.Contains(SpecialBuiltins, logical)
}

type lexer struct {
	fn         string
	custom     *CustomOperators
	Tokens     []Token
	source     string
	pos        int
	line       int
	currentPos int
	col        int
	lineText   string
	lineTextAt int
	lineTextOK bool
	posi       *helpers.Position
	// indentLevel is the per-lexer nesting depth of the optional debug trace.
	indentLevel int
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
	return tokenize(source, fn, nil)
}

// TokenizeCollecting is retained for compatibility. Lexical problems are
// returned as UNKNOWN tokens, so the diagnostics result is always empty.
func TokenizeCollecting(source string, fn string, custom *CustomOperators) ([]Token, []helpers.ErrorInterface) {
	return tokenize(source, fn, custom), nil
}

// TokenizeWith lexes source with the user-defined symbols in a project operator catalog.
//
// A custom operator cannot be recognised from one source file alone: a symbol may be
// declared elsewhere in the project. The caller therefore supplies the precomputed
// project operator catalog. Semantic name resolution remains responsible for deciding
// whether a catalogued spelling is visible at a use site. Tokenize is the same call with
// no custom operators, which consumers that do not need project operators use.
func TokenizeWith(source string, fn string, custom *CustomOperators) []Token {
	return tokenize(source, fn, custom)
}

// tokenize drains the lazy lexer for the legacy slice APIs, then applies their
// historical whole-stream folding passes.
func tokenize(source string, fn string, custom *CustomOperators) []Token {
	lazyLexer := newLexer([]byte(source), fn, custom)
	lex := lazyLexer.inner
	if DEBUG_TRACE {
		defer lex.debugTraceEnd(lex.debugTraceBegin("tokenize", INVALID, ""))
	}

	// Keep the historical slice API as an adapter over the lazy scanner. The
	// parser-facing TokenStream does not materialize this complete slice.
	tokens := make([]Token, 0)
	for {
		token := lazyLexer.nextToken()
		tokens = append(tokens, token)
		if token.Kind == EOF {
			break
		}
	}
	lex.Tokens = tokens
	lex.currentPos = 0
	cleanupLB(lex)
	foldTokens(lex)
	foldSpecialStatementBuiltins(lex)
	return lex.Tokens
}

// foldSpecialStatementBuiltins is the final canonicalization pass for the
// statement-only `this.<verb>` spellings. Earlier dotted-name folding may keep
// an invoked member split so ordinary calls preserve their receiver. These
// registered statement heads are the exception and must remain one token even
// when an expression-opening parenthesis follows.
func foldSpecialStatementBuiltins(lex *lexer) {
	in := lex.Tokens
	out := make([]Token, 0, len(in))
	for i := 0; i < len(in); {
		if i+2 < len(in) && logicalFoldedName(in[i].Value) == "this" &&
			in[i+1].Kind == DOT && isSpecialBuiltin("this."+logicalFoldedName(in[i+2].Value)) {
			out = append(out, newUniqueToken(BUIL_IN_STMT_EXPRS, "this."+logicalFoldedName(in[i+2].Value),
				in[i].StartPos.Copy(), in[i+2].EndPos.Copy()))
			i += 3
			continue
		}
		out = append(out, in[i])
		i++
	}
	lex.Tokens = out
}

func logicalFoldedName(name string) string {
	logical := strings.ReplaceAll(name, "_fo.", ".")
	return strings.TrimSuffix(logical, "_fo")
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
		lastToken = ""
		length := 1
		lstTokens := []Token{Token_}

		if Token_.Kind == IDENTIFIER || Token_.Kind == KEYWORD || Token_.Kind == RESERVEDWORD || Token_.Kind == CONTEXT_KEYWORD || Token_.Kind == ATDAP {
			if (Token_.Kind == KEYWORD || Token_.Kind == RESERVEDWORD || Token_.Kind == CONTEXT_KEYWORD) &&
				lex.lookAhead(1).Kind == DOT && slices.Contains(UnsupportedObjects, Token_.Value) {
				if _, hasKeywordMethods := KeyWords_me[Token_.Value]; hasKeywordMethods {
					unknown := Token_
					unknown.Kind = UNKNOWN
					consumed := 0
					for lex.lookAhead(consumed+1).Kind == DOT {
						part := lex.lookAhead(consumed + 2)
						if part.Kind != IDENTIFIER && part.Kind != KEYWORD && part.Kind != RESERVEDWORD &&
							part.Kind != CONTEXT_KEYWORD && part.Kind != BUILT_IN_METHOD {
							break
						}
						unknown.Value += "." + part.Value
						if part.EndPos != nil {
							unknown.EndPos = part.EndPos.Copy()
						}
						consumed += 2
					}
					if consumed > 0 {
						nTokens = append(nTokens, unknown)
						lex.currentPos += consumed
						lex.moveNext()
						continue
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

		if invoked := lex.lookAhead(1).Kind == OPEN_PAREN; changed && reservedReceiverChainNeedsSeparation(lstTokens, tempToken) {
			// Hard-reserved roots remain visible as primaries for both field access
			// and invocation. They must never be lowered into an ordinary composite
			// identifier merely because a member follows them.
			nTokens = appendSeparatedMemberChain(nTokens, lstTokens, invoked)
		} else if changed && dottedChainFollowsCompletedExpression(lex, len(lstTokens)) {
			// A chain after a completed receiver is postfix structure, not a
			// qualified name. Preserve every source dot so
			// `factory().service.worker.run()` becomes three MemberExpr suffixes.
			nTokens = appendSeparatedMemberChain(nTokens, lstTokens, lex.lookAhead(1).Kind == OPEN_PAREN)
		} else if changed {
			dirTok := tempToken
			if _, ok := Built_in_directives(dirTok); ok && strings.HasPrefix(tempToken, "@") {
				nTokens = append(nTokens, newUniqueToken(BUILT_IN_DIRECTIVES, tempToken, lstTokens[0].StartPos.Copy(), lstTokens[len(lstTokens)-1].EndPos.Copy()))
			} else if IsLanguageOwnedMetadataName(tempToken) {
				nTokens = append(nTokens, newUniqueToken(UNKNOWN, tempToken, lstTokens[0].StartPos.Copy(), lstTokens[len(lstTokens)-1].EndPos.Copy()))
			} else if strings.HasPrefix(tempToken, "@") {
				nTokens = append(nTokens, newUniqueToken(CUSTOM_DIRECTIVES, tempToken, lstTokens[0].StartPos.Copy(), lstTokens[len(lstTokens)-1].EndPos.Copy()))

			} else if tempToken == "co.operator" {
				nTokens = append(nTokens, newUniqueToken(OPERATOR_SOURCE_KIND, tempToken, lstTokens[0].StartPos.Copy(), lstTokens[len(lstTokens)-1].EndPos.Copy()))
			} else if _, ok := Operator_source_constants[tempToken]; ok {
				// A co.operator.* property value belongs to the operator-source
				// grammar alone, so it keeps its exact spelling and takes no
				// backend lowering (DECISION-OPDECL-006).
				nTokens = append(nTokens, newUniqueToken(OPERATOR_SOURCE_CONSTANT, tempToken, lstTokens[0].StartPos.Copy(), lstTokens[len(lstTokens)-1].EndPos.Copy()))
			} else if slices.Contains(Builtin_Kinds, tempToken) {
				nTokens = append(nTokens, newUniqueToken(BUILT_IN_KIND, tempToken, lstTokens[0].StartPos.Copy(), lstTokens[len(lstTokens)-1].EndPos.Copy()))
			} else if slices.Contains(Builtin_types, tempToken) {
				nTokens = append(nTokens, newUniqueToken(BUILT_IN_TYPE, tempToken, lstTokens[0].StartPos.Copy(), lstTokens[len(lstTokens)-1].EndPos.Copy()))

			} else if slices.Contains(Built_In_Collections, tempToken) {
				nTokens = append(nTokens, newUniqueToken(BUILT_IN_COLLECTIONS, tempToken, lstTokens[0].StartPos.Copy(), lstTokens[len(lstTokens)-1].EndPos.Copy()))

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
				// A registered dotted statement spelling may have an OPEN_PAREN after
				// its folded path without being an ordinary method call.
				if lex.lookAhead(1).Kind == OPEN_PAREN && !isSpecialBuiltin(tempToken) {
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
					if isSpecialBuiltin(tempToken) {
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
					if separated, ok := appendLongestBuiltInQualifiedName(nTokens, lstTokens); ok {
						nTokens = separated
					} else if length > 1 {
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
				} else if slices.Contains(Built_In_Collections, tempToken) {
					nTokens = append(nTokens, newUniqueToken(BUILT_IN_COLLECTIONS, nTempToken, lstTokens[0].StartPos.Copy(), lstTokens[len(lstTokens)-3].EndPos.Copy()))
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
	if slices.Contains(Built_In_Collections, name) {
		return BUILT_IN_COLLECTIONS, true
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

// appendLongestBuiltInQualifiedName preserves a registered co.* receiver while
// leaving a non-call final member visible to ordinary name resolution. This is
// important for standard-package API evolution: the lexer owns the package/type
// prefix, not a closed list of every declaration that package may export.
func appendLongestBuiltInQualifiedName(out []Token, gathered []Token) ([]Token, bool) {
	separated, ok := appendLongestBuiltInReceiver(out, gathered)
	if !ok || len(gathered) < 3 {
		return out, false
	}
	dot := gathered[len(gathered)-2]
	separated = append(separated, newUniqueToken(DOT, ".", dot.StartPos.Copy(), dot.EndPos.Copy()))
	separated = append(separated, normalizedMemberToken(gathered[len(gathered)-1]))
	return separated, true
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

// reservedReceiverChainNeedsSeparation keeps hard-reserved receiver roots visible
// to the parser. `this.member` is ordinary member syntax, while fΦλ is the
// private standard-package root; neither spelling may be identifier-lowered.
func reservedReceiverChainNeedsSeparation(gathered []Token, fullName string) bool {
	if len(gathered) == 0 || isSpecialBuiltin(fullName) {
		return false
	}
	return gathered[0].Value == "this" || gathered[0].Value == "fΦλ"
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
	parts := strings.Split(tempToken, ".")
	if len(parts) < 2 || parts[0] != Token_.Value || parts[len(parts)-1] != lastToken {
		return true
	}

	prefix := parts[0]
	for _, part := range parts[1:] {
		members, ok := Built_in_stmt_exprs[prefix]
		if !ok || !slices.Contains(members, part) {
			return true
		}
		prefix += "." + part
	}
	return false
}
func (lex *lexer) advanceN(n int) {
	lex.pos += n
	lex.col += n
}
func (lex *lexer) advanceline(n int) {
	lex.line += n
	lex.invalidateCurrentLineText()
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

func createLexer(source string, fn string) *lexer {
	return &lexer{
		pos:        0,
		line:       1,
		source:     source,
		currentPos: 0,
		fn:         fn,
		col:        1,
		posi:       helpers.NewPosition(0, 1, 0, 0, fn, "", false),
		Tokens:     make([]Token, 0),
	}

}
