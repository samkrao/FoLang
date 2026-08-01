package scanlex

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/samkrao/fo-lang/frontend/src/foerrors"
	"github.com/samkrao/fo-lang/frontend/src/helpers"
)

type regexPattern struct {
	regex   *regexp.Regexp
	handler regexHandler
}

// numericLiteralPattern matches a COMPLETE integer-literal or floating-literal,
// suffix included, in one token.
//
// The grammar requires this of the scanner: with the abbreviated float forms gone, a
// numeric literal can never end at a point, so "the scanner needs no numeric lookahead
// and the parser never re-lexes" (the note withdrawing DECISION-LEX-005). Matching only
// bare digits here and reassembling the rest from adjacent identifier tokens in the
// parser is what left 0xFFu, 1e5f and 0x1p3 unparseable — the reassembly covered some
// combinations and not others.
//
// Alternatives are ordered longest-match-first, which is what DECISION-LEX-003 maximal
// munch needs from Go's leftmost-first alternation: a hexadecimal FLOAT is tried before
// a hexadecimal integer, and a decimal float before a decimal integer.
var numericLiteralPattern = regexp.MustCompile(
	`^(?:` +
		// hexadecimal-floating-literal: prefix, (fractional | digits), binary exponent
		`0[xX](?:[0-9a-fA-F]+\.[0-9a-fA-F]+|[0-9a-fA-F]+)[pP][+-]?[0-9]+` + floatSuffixRE +
		// hexadecimal-integer-literal
		`|0[xX][0-9a-fA-F]+` + intSuffixRE +
		// binary-integer-literal
		`|0[bB][01]+` + intSuffixRE +
		// decimal-floating-literal: fractional-constant with optional exponent
		`|[0-9]+\.[0-9]+(?:[eE][+-]?[0-9]+)?` + floatSuffixRE +
		// decimal-floating-literal: digit-sequence with a mandatory exponent
		`|[0-9]+[eE][+-]?[0-9]+` + floatSuffixRE +
		// octal-integer-literal and decimal-integer-literal share this shape
		`|[0-9]+` + intSuffixRE +
		`)`)

// floatSuffixRE is floating-point-suffix, longest alternative first so "f128" wins
// over "f". intSuffixRE is integer-suffix in its four documented orderings.
const (
	floatSuffixRE = `(?:bf16|BF16|f128|F128|f16|F16|f32|F32|f64|F64|[fFlL])?`
	intSuffixRE   = `(?:[uU](?:ll|LL|[lLzZ])?|(?:ll|LL)[uU]?|[lLzZ][uU]?)?`
)

type lexer struct {
	fn         string
	patterns   []regexPattern
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
	// DECISION-LEX-001: a U+FEFF byte-order mark is permitted only as the first code
	// point. Removing it here accepts the editors that emit one; anywhere else the
	// character still reaches the scanner's no-match path and is reported.
	source = strings.TrimPrefix(source, utf8BOM)

	lex := createLexer(source, fn)

	for !lex.at_eof() {
		if length, message, unsupported := detectUnsupportedAlphaLiteral(lex.remainder()); unsupported {
			rejectUnsupportedAlphaLiteral(lex, length, message)
			continue
		}

		matched := false

		for _, pattern := range lex.patterns {
			loc := pattern.regex.FindStringIndex(lex.remainder())
			if loc != nil && loc[0] == 0 {
				pattern.handler(lex, pattern.regex)
				matched = true
				break // Exit the loop after the first match
			}
		}

		if !matched {
			err_ := lex.errorObj(nil, fmt.Sprintf("lexer error: unrecognized token near '%v'", lex.remainder()))
			foerrors.HandleErrors(err_)

		}
	}
	startPos := helpers.NewPosition(1, 0, 1, 0, "", "", false)
	endPos := helpers.NewPosition(1, 0, 1, 0, "", "", false)
	lex.push(newUniqueToken(EOF, "EOF", startPos, endPos))
	cleanupLB(lex)
	foldTokens(lex)
	return lex.Tokens
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

		if changed {
			dirTok := tempToken
			//dirTok = strings.TrimPrefix(dirTok, "@")
			if _, ok := Built_in_directives(dirTok); ok && strings.HasPrefix(tempToken, "@") {
				nTokens = append(nTokens, newUniqueToken(BUILT_IN_DIRECTIVES, tempToken, lstTokens[0].StartPos.Copy(), lstTokens[len(lstTokens)-3].EndPos.Copy()))
			} else if strings.HasPrefix(tempToken, "@") {
				nTokens = append(nTokens, newUniqueToken(CUSTOM_DIRECTIVES, tempToken, lstTokens[0].StartPos.Copy(), lstTokens[len(lstTokens)-3].EndPos.Copy()))

			} else if slices.Contains(Builtin_Kinds, tempToken) {
				nTokens = append(nTokens, newUniqueToken(BUILT_IN_KIND, tempToken, lstTokens[0].StartPos.Copy(), lstTokens[len(lstTokens)-1].EndPos.Copy()))
			} else if slices.Contains(Builtin_types, tempToken) {
				nTokens = append(nTokens, newUniqueToken(BUILT_IN_TYPE, tempToken, lstTokens[0].StartPos.Copy(), lstTokens[len(lstTokens)-1].EndPos.Copy()))

			} else if _, ok := Built_in_constants[tempToken]; ok {
				nTokens = append(nTokens, newUniqueToken(BUILT_IN_CONSTANTS, tempToken, lstTokens[0].StartPos.Copy(), lstTokens[len(lstTokens)-1].EndPos.Copy()))

			} else if _, ok := Built_in_stmt_exprs[Token_.Value]; ok {
				otherFlag := checkBuiltInStExmet(Token_, tempToken, lastToken)
				rmethod := false

				if !otherFlag {
					var nTempToken = tempToken
					if slices.Contains(SpecialBuiltins, tempToken) {
						rmethod = false
					} else if slices.Contains(Reserved_me, lastToken) {
						rmethod = true
						nTempToken = strings.Replace(tempToken, "."+lastToken, "", 1)
					} else {
						rmethod = false
					}
					nTokens = append(nTokens, newUniqueToken(BUIL_IN_STMT_EXPRS, nTempToken, lstTokens[0].StartPos.Copy(), lstTokens[len(lstTokens)-1].EndPos.Copy()))

				} else {
					if length > 1 {
						tempToken = strings.ReplaceAll(tempToken, ".", "_fo.")
						nTokens = append(nTokens, newUniqueToken(COMPOSITE_IDENTIFER, tempToken, lstTokens[0].StartPos.Copy(), lstTokens[len(lstTokens)-1].EndPos.Copy()))
					} else {
						nTokens = append(nTokens, Token_)
					}
				}
				if rmethod {
					nTokens = append(nTokens, newUniqueToken(DOT, ".", lstTokens[len(lstTokens)-2].StartPos.Copy(), lstTokens[len(lstTokens)-2].EndPos.Copy()))
					nTokens = append(nTokens, newUniqueToken(BUILT_IN_METHOD, lastToken, lstTokens[len(lstTokens)-1].StartPos.Copy(), lstTokens[len(lstTokens)-1].EndPos.Copy()))
				}
			} else if slices.Contains(Reserved_me, lastToken) {

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
					nTokens = append(nTokens, newUniqueToken(IDENTIFIER, lastToken, lstTokens[len(lstTokens)-1].StartPos.Copy(), lstTokens[len(lstTokens)-1].EndPos.Copy()))

				} else {
					tempToken = strings.ReplaceAll(tempToken, ".", "_fo.")
					nTokens = append(nTokens, newUniqueToken(COMPOSITE_IDENTIFER, tempToken, lstTokens[0].StartPos.Copy(), lstTokens[len(lstTokens)-1].EndPos.Copy()))
				}
			}

		} else {
			if Token_.Kind == IDENTIFIER {
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
		patterns: []regexPattern{
			//{regexp.MustCompile(`[\n\r\f]`), newLineHandler},
			{regexp.MustCompile(`\r\n|\n|\r`), newLineHandler},
			//{regexp.MustCompile(`(?m)^[ \t]*\r?\n`), newLineHandler},
			//{regexp.MustCompile(`\s+`), skipHandler},
			//{regexp.MustCompile(`\t+`), skipHandler},
			//{regexp.MustCompile(`^[ \t]*\r?\n?$`), newLineHandler},
			// horizontal-white-space = " " | "\t" | "\f". [[:blank:]] covers only
			// space and tab, so a form feed was an unrecognized character.
			{regexp.MustCompile(`[ \t\f]+`), skipHandler},
			{regexp.MustCompile(`\/\/.*`), commentHandler},
			// block-comment = "/*", { block-comment-character }, "*/". Non-greedy so
			// the comment ends at the FIRST "*/", and (?s) lets it span lines.
			{regexp.MustCompile(`(?s)/\*.*?\*/`), blockCommentHandler},
			// alpha-basic-s-character excludes CR and LF, so a plain quoted string
			// never spans a line break. Allowing one swallowed whole statements when
			// a closing quote was missing.
			{regexp.MustCompile(`"[^"\r\n]*"`), stringHandler},
			// alpha-basic-c-character is any translation character EXCEPT the
			// apostrophe, the backslash, CR and LF. A space and a tab are ordinary
			// characters, so `' '` and a literal tab are well-formed; excluding all
			// whitespace here rejected them. The backslash stays out because an
			// escape is a reserved post-alpha spelling (DECISION-LEX-008).
			{regexp.MustCompile(`'[^'\\\r\n]'`), characterHandler},
			{numericLiteralPattern, numberHandler},
			{regexp.MustCompile(`\__`), defaultHandler(DBL_UNDERSCORE, "__")},
			{regexp.MustCompile(`_`), discardVarHandler},
			{regexp.MustCompile(`[a-zA-Z_][a-zA-Z0-9_]*`), symbolHandler},
			{regexp.MustCompile(`@@`), defaultHandler(DOUBLE_AT, "@@")},
			{regexp.MustCompile(`@([a-zA-Z_][a-zA-Z0-9_]*)`), dapHandler},
			{regexp.MustCompile(`[$][0-9]*`), bindVarHandler},
			{regexp.MustCompile(`\[:\]`), defaultHandler(OB_COLON_CB, "[:]")},
			{regexp.MustCompile(`\[`), defaultHandler(OPEN_BRACKET, "[")},
			{regexp.MustCompile(`\]`), defaultHandler(CLOSE_BRACKET, "]")},
			{regexp.MustCompile(`\{`), defaultHandler(OPEN_CURLY, "{")},
			{regexp.MustCompile(`\}`), defaultHandler(CLOSE_CURLY, "}")},
			{regexp.MustCompile(`\(`), defaultHandler(OPEN_PAREN, "(")},
			{regexp.MustCompile(`\)`), defaultHandler(CLOSE_PAREN, ")")},
			{regexp.MustCompile(`==>>`), defaultHandler(EQEQGTGT, "==>>")},
			{regexp.MustCompile(`==`), defaultHandler(EQUALS, "==")},
			{regexp.MustCompile(`=>`), defaultHandler(EQGT, "=>")},
			{regexp.MustCompile(`=>>`), defaultHandler(EQGTGT, "=>>")},
			{regexp.MustCompile(`!=`), defaultHandler(NOT_EQUALS, "!=")},
			{regexp.MustCompile(`=`), defaultHandler(ASSIGNMENT, "=")},
			{regexp.MustCompile(`!`), defaultHandler(NOT, "!")},
			{regexp.MustCompile(`<=`), defaultHandler(LESS_EQUALS, "<=")},
			{regexp.MustCompile(`<\.\.<`), defaultHandler(LT_DOT_DOT_LT, "<..<")},
			{regexp.MustCompile(`<\.\.`), defaultHandler(LT_DOT_DOT, "<..")},
			{regexp.MustCompile(`<->`), defaultHandler(BIDIR_ARROW, "<->")},
			{regexp.MustCompile(`<-`), defaultHandler(LEFT_ARROW, "<-")},
			{regexp.MustCompile(`<`), defaultHandler(LESS, "<")},
			{regexp.MustCompile(`>=`), defaultHandler(GREATER_EQUALS, ">=")},
			{regexp.MustCompile(`>`), defaultHandler(GREATER, ">")},
			{regexp.MustCompile(`\|\|`), defaultHandler(OR, "||")},
			{regexp.MustCompile(`&&`), defaultHandler(AND, "&&")},
			{regexp.MustCompile(`\.\.<`), defaultHandler(DOT_DOT_LT, "..<")},
			{regexp.MustCompile(`\.\.\.`), defaultHandler(DOT_DOT_DOT, "...")},

			{regexp.MustCompile(`\.\.`), defaultHandler(DOT_DOT, "..")},
			{regexp.MustCompile(`\.`), defaultHandler(DOT, ".")},
			{regexp.MustCompile(`;`), defaultHandler(SEMI_COLON, ";")},
			{regexp.MustCompile(`::=`), defaultHandler(COLON_WALRUS, "::=")},
			{regexp.MustCompile(`:=`), defaultHandler(WALRUS, ":=")},
			{regexp.MustCompile(`:`), defaultHandler(COLON, ":")},
			{regexp.MustCompile(`->>`), defaultHandler(MINUS_ARROW_GT, "->>")},
			{regexp.MustCompile(`->`), defaultHandler(ARROW, "->")},
			{regexp.MustCompile(`\?\?=`), defaultHandler(NULLISH_ASSIGNMENT, "??=")},
			{regexp.MustCompile(`\?=`), defaultHandler(QEQ, "?=")},
			{regexp.MustCompile(`\?`), defaultHandler(QUESTION, "?")},
			{regexp.MustCompile(`,`), defaultHandler(COMMA, ",")},
			{regexp.MustCompile(`\+\+`), defaultHandler(PLUS_PLUS, "++")},
			{regexp.MustCompile(`--`), defaultHandler(MINUS_MINUS, "--")},
			{regexp.MustCompile(`\+=`), defaultHandler(PLUS_EQUALS, "+=")},
			{regexp.MustCompile(`-=`), defaultHandler(MINUS_EQUALS, "-=")},
			{regexp.MustCompile(`\+`), defaultHandler(PLUS, "+")},
			{regexp.MustCompile(`-`), defaultHandler(MINUS, "-")},
			{regexp.MustCompile(`/`), defaultHandler(SLASH, "/")},
			{regexp.MustCompile(`\*`), defaultHandler(STAR, "*")},
			{regexp.MustCompile(`%`), defaultHandler(PERCENT, "%")},
			{regexp.MustCompile(`"`), defaultHandler(DOUBL_QUOTE, "\"")},
			{regexp.MustCompile(`'`), defaultHandler(SINGLE_QUOTE, "'")},
			{regexp.MustCompile(`@`), defaultHandler(AT, "@")},
			{regexp.MustCompile(`\|`), defaultHandler(PIPE, "|")},
			{regexp.MustCompile(`\^`), defaultHandler(POW, "^")},
			{regexp.MustCompile(`\&`), defaultHandler(AMPS, "&")},
			// DECISION-OP-005: the backtick is a reserved operator token. The lexer
			// recognizes it and the parser rejects it. The old handler silently
			// stripped the backticks and re-emitted the quoted word as an ordinary
			// identifier, which is precisely the silent reuse the decision forbids.
			{regexp.MustCompile("`"), defaultHandler(BACK_TICK, "`")},
			{regexp.MustCompile(`~~`), defaultHandler(TILD_TILD, "~~")},
			{regexp.MustCompile(`~`), defaultHandler(TILD, "~")},
			// "#" is the length/count prefix of prefix-operator. The parser's prefix
			// table has always listed it, but with no rule here the scanner rejected
			// the character outright, so the operator was unreachable.
			{regexp.MustCompile(`#`), defaultHandler(HASH, "#")},
			// DECISION-OP-005: the reserved-future-operator glyph set. Outside
			// literals — string, character and comment rules all match earlier —
			// these spellings are reserved, so they get the reserved diagnostic
			// rather than the generic "unrecognized token" failure.
			{regexp.MustCompile(`[λ⒪âŤ∀∃○ö∪ṠŜṁ𝚷⇛𝑓𝒯𝘷𝓕↓∂⊥↧⇓]`), reservedGlyphHandler},
		},
	}

}

type regexHandler func(lex *lexer, regex *regexp.Regexp)

// Created a default handler which will simply create a token with the matched contents. This handler is used with most simple tokens.
func defaultHandler(kind TokenKind, value string) regexHandler {

	return func(lex *lexer, _ *regexp.Regexp) {

		var startpos = helpers.NewPosition(lex.pos, lex.line, lex.col, lex.pos, lex.fn, lex.sourcearr[lex.line-1], false)
		lex.posi = startpos
		lex.advanceN(len(value))
		var endPos = helpers.NewPosition(lex.pos, lex.line, lex.col, lex.pos, lex.fn, lex.sourcearr[lex.line-1], false)
		lex.push(newUniqueToken(kind, value, lex.posi.Copy(), endPos))
		lex.posi = endPos
	}
}

func stringHandler(lex *lexer, regex *regexp.Regexp) {
	match := regex.FindStringIndex(lex.remainder())
	stringLiteral := lex.remainder()[match[0]:match[1]]
	var startpos = helpers.NewPosition(lex.pos, lex.line, lex.col, lex.pos, lex.fn, lex.sourcearr[lex.line-1], false)
	lex.posi = startpos
	lex.advanceN(len(stringLiteral))
	var endPos = helpers.NewPosition(lex.pos, lex.line, lex.col, lex.pos, lex.fn, lex.sourcearr[lex.line-1], false)

	lex.push(newUniqueToken(STRING, stringLiteral, lex.posi.Copy(), endPos))
	lex.posi = endPos

}

func characterHandler(lex *lexer, regex *regexp.Regexp) {
	match := regex.FindStringIndex(lex.remainder())
	charLiteral := lex.remainder()[match[0]:match[1]]
	var startpos = helpers.NewPosition(lex.pos, lex.line, lex.col, lex.pos, lex.fn, lex.sourcearr[lex.line-1], false)
	lex.posi = startpos
	lex.advanceN(len(charLiteral))
	var endPos = helpers.NewPosition(lex.pos, lex.line, lex.col, lex.pos, lex.fn, lex.sourcearr[lex.line-1], false)

	lex.push(newUniqueToken(CHAR, charLiteral, lex.posi.Copy(), endPos))
	lex.posi = endPos
}

func numberHandler(lex *lexer, regex *regexp.Regexp) {
	match := regex.FindString(lex.remainder())
	var startpos = helpers.NewPosition(lex.pos, lex.line, lex.col, lex.pos, lex.fn, lex.sourcearr[lex.line-1], false)
	lex.posi = startpos
	lex.advanceN(len(match))
	var endPos = helpers.NewPosition(lex.pos, lex.line, lex.col, lex.pos, lex.fn, lex.sourcearr[lex.line-1], false)

	lex.push(newUniqueToken(NUMBER, match, lex.posi.Copy(), endPos))
	lex.posi = endPos

}

func discardVarHandler(lex *lexer, regex *regexp.Regexp) {
	match := regex.FindString(lex.remainder())
	var startpos = helpers.NewPosition(lex.pos, lex.line, lex.col, lex.pos, lex.fn, lex.sourcearr[lex.line-1], false)
	lex.posi = startpos
	lex.advanceN(len(match))
	var endPos = helpers.NewPosition(lex.pos, lex.line, lex.col, lex.pos, lex.fn, lex.sourcearr[lex.line-1], false)

	lex.push(newUniqueToken(DISCARD_WILD_VAR, match, lex.posi.Copy(), endPos))

	lex.posi = endPos
}
func bindVarHandler(lex *lexer, regex *regexp.Regexp) {
	match := regex.FindString(lex.remainder())
	var startpos = helpers.NewPosition(lex.pos, lex.line, lex.col, lex.pos, lex.fn, lex.sourcearr[lex.line-1], false)
	lex.posi = startpos
	lex.advanceN(len(match))
	var endPos = helpers.NewPosition(lex.pos, lex.line, lex.col, lex.pos, lex.fn, lex.sourcearr[lex.line-1], false)

	lex.push(newUniqueToken(BIND_VAR, match, lex.posi.Copy(), endPos))

	lex.posi = endPos

}
// dapHandler scans an annotation introducer.
//
// The grammar spells an annotation as "@", qualified-name, so a user-defined annotation
// carries an ordinary name and is NOT restricted to the co.* namespace. Which names are
// meaningful — and whether a given annotation may appear where it was written — is
// resolved after parsing, so the scanner classifies every one of them the same way.
func dapHandler(lex *lexer, regex *regexp.Regexp) {
	match := regex.FindString(lex.remainder())
	var startpos = helpers.NewPosition(lex.pos, lex.line, lex.col, lex.pos, lex.fn, lex.sourcearr[lex.line-1], false)
	lex.posi = startpos
	lex.advanceN(len(match))
	var endPos = helpers.NewPosition(lex.pos, lex.line, lex.col, lex.pos, lex.fn, lex.sourcearr[lex.line-1], false)

	lex.push(newUniqueToken(ATDAP, match, lex.posi.Copy(), endPos))

	lex.posi = endPos

}

// reservedGlyphHandler reports one character from the reserved-future-operator glyph
// set (DECISION-OP-005). The glyphs are reserved outside literals and declared
// operator symbols; none has a meaning yet, so encountering one is always an error,
// and naming it beats the generic "unrecognized token" failure.
func reservedGlyphHandler(lex *lexer, regex *regexp.Regexp) {
	match := regex.FindString(lex.remainder())
	start := helpers.NewPosition(lex.pos, lex.line, lex.col, lex.pos, lex.fn, lex.sourcearr[lex.line-1], false)
	lex.advanceN(len(match))
	end := helpers.NewPosition(lex.pos, lex.line, lex.col, lex.pos, lex.fn, lex.sourcearr[lex.line-1], false)
	foerrors.HandleErrors(lex.errorException(
		fmt.Sprintf("the glyph %q is reserved for a future FoLang operator and cannot be used yet", match),
		helpers.ReservedKeyword, *start, *end))
}

func symbolHandler(lex *lexer, regex *regexp.Regexp) {
	match := regex.FindString(lex.remainder())
	startPos := helpers.NewPosition(lex.pos, lex.line, lex.col, lex.pos, lex.fn, lex.sourcearr[lex.line-1], false)
	lex.posi = startPos
	lex.advanceN(len(match))
	endPos := helpers.NewPosition(lex.pos, lex.line, lex.col, lex.pos, lex.fn, lex.sourcearr[lex.line-1], false)

	// DECISION-LEX-001/006: an identifier contains single underscores between
	// nonempty alphanumeric segments and never ends in one (grammar: identifier,
	// identifier-segment, identifier-trailing-guard). The match regex is wider so
	// the whole malformed name is consumed and reported as one unit rather than
	// splitting into confusing fragments.
	if strings.Contains(match, "__") {
		foerrors.HandleErrors(lex.errorException(
			fmt.Sprintf("identifier %q contains consecutive underscores; FoLang identifiers use single underscores between alphanumeric segments", match),
			helpers.InvalidSyntax, *startPos, *endPos))
	}
	if strings.HasSuffix(match, "_") {
		foerrors.HandleErrors(lex.errorException(
			fmt.Sprintf("identifier %q ends in an underscore; a FoLang identifier must end in a letter or digit", match),
			helpers.InvalidSyntax, *startPos, *endPos))
	}

	// Consult Reserved_lu so reserved words get their proper kind (KEYWORD /
	// RESERVEDWORD) instead of falling through as IDENTIFIER. Identifiers that
	// are not reserved emit IDENTIFIER as before — foldTokens will append _fo.
	kind := IDENTIFIER
	if k, ok := Reserved_lu[match]; ok {
		kind = k
	}
	lex.push(newUniqueToken(kind, match, startPos.Copy(), endPos))
	lex.posi = endPos
}

func skipHandler(lex *lexer, regex *regexp.Regexp) {
	match := regex.FindStringIndex(lex.remainder())
	lex.advanceN(match[1])
}

func newLineHandler(lex *lexer, regex *regexp.Regexp) {
	startPos := helpers.NewPosition(lex.pos, lex.line, lex.col, lex.pos, lex.fn, lex.sourcearr[lex.line-1], false)
	lex.posi = startPos
	lex.advanceN(1)
	var endPos = helpers.NewPosition(lex.pos, lex.line, lex.col, lex.pos, lex.fn, lex.sourcearr[lex.line-1], false)
	lex.push(newUniqueToken(NEWLINE, "LSP", startPos.Copy(), endPos))
	lex.posi = endPos
	lex.advanceline(1)

}

func commentHandler(lex *lexer, regex *regexp.Regexp) {
	match := regex.FindString(lex.remainder())
	lex.advanceN(len(match))
}

// blockCommentHandler skips a block-comment. Unlike a line comment it may span line
// breaks, so the lines it covers are counted to keep diagnostics after it on the right
// line. An unterminated comment does not match the pattern at all and is reported by
// the scanner's no-match path.
func blockCommentHandler(lex *lexer, regex *regexp.Regexp) {
	match := regex.FindString(lex.remainder())
	lines := strings.Count(match, "\n")
	lex.advanceN(len(match))
	if lines > 0 {
		lex.advanceline(lines)
	}
}
