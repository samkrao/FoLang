package scanlex

import (
	"fmt"
	"slices"
	"strings"

	"github.com/samkrao/fo-lang/frontend/src/foerrors"
	"github.com/samkrao/fo-lang/frontend/src/helpers"
)

type lexer struct {
	fn         string
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

		src := lex.remainder()
		result, ok := lex.scanToken(src)
		if !ok {
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
			}
		case actionError:
			start := helpers.NewPosition(lex.pos, lex.line, lex.col, lex.pos, lex.fn, lex.currentLineText(), false)
			lex.advanceN(result.length)
			end := helpers.NewPosition(lex.pos, lex.line, lex.col, lex.pos, lex.fn, lex.currentLineText(), false)
			if result.lines > 0 {
				lex.advanceline(result.lines)
			}
			foerrors.HandleErrors(lex.errorException(result.message, result.errType, *start, *end))
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
	}

}
