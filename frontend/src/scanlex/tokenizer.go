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

type lexer struct {
	fn         string
	patterns   []regexPattern
	Tokens     []Token
	source     string
	sourcearr  []string
	pos        int
	line       int
	backtick   bool
	currentPos int
	col        int
	posi       *helpers.Position
}

// Tokenize lexes the given source string into a slice of Tokens, performing folding and cleanup.
func Tokenize(source string, fn string) []Token {
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
		backtick:   false,
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
			{regexp.MustCompile(`[[:blank:]]+`), skipHandler},
			{regexp.MustCompile(`\/\/.*`), commentHandler},
			{regexp.MustCompile(`"[^"]*"`), stringHandler},
			{regexp.MustCompile(`'[^'\s\t\r\n]'`), characterHandler},
			{regexp.MustCompile(`[0-9]+(\.[0-9]+)?`), numberHandler},
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
			{regexp.MustCompile("`"), backTickHandler},
			{regexp.MustCompile(`~~`), defaultHandler(TILD_TILD, "~~")},
			{regexp.MustCompile(`~`), defaultHandler(TILD, "~")},
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

func backTickHandler(lex *lexer, regex *regexp.Regexp) {
	if lex.backtick {
		lex.backtick = false
		lex.advanceN(len("`"))
		return
	}
	var startpos = helpers.NewPosition(lex.pos, lex.line, lex.col, lex.pos, lex.fn, lex.sourcearr[lex.line-1], false)
	lex.posi = startpos
	lex.advanceN(len("`"))
	var regex_n = regexp.MustCompile(`[a-zA-Z_][a-zA-Z0-9_]*`)
	match := regex_n.FindString(lex.remainder())
	lex.advanceN(len(match))
	var endPos = helpers.NewPosition(lex.pos, lex.line, lex.col, lex.pos, lex.fn, lex.sourcearr[lex.line-1], false)
	lex.push(newUniqueToken(IDENTIFIER, match, lex.posi.Copy(), endPos))
	lex.backtick = true
	lex.posi = endPos
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
func dapHandler(lex *lexer, regex *regexp.Regexp) {
	match := regex.FindString(lex.remainder())
	if !strings.HasPrefix(match, "@co") {
		tTk := newDummyToken(match, lex.currentToken().StartPos, lex.currentToken().EndPos)
		err_ := lex.errorObj(&tTk, "Cannot have @ with any other literal except @co")
		foerrors.HandleErrors(err_)

	}
	var startpos = helpers.NewPosition(lex.pos, lex.line, lex.col, lex.pos, lex.fn, lex.sourcearr[lex.line-1], false)
	lex.posi = startpos
	lex.advanceN(len(match))
	var endPos = helpers.NewPosition(lex.pos, lex.line, lex.col, lex.pos, lex.fn, lex.sourcearr[lex.line-1], false)

	lex.push(newUniqueToken(ATDAP, match, lex.posi.Copy(), endPos))

	lex.posi = endPos

}

func symbolHandler(lex *lexer, regex *regexp.Regexp) {
	match := regex.FindString(lex.remainder())
	startPos := helpers.NewPosition(lex.pos, lex.line, lex.col, lex.pos, lex.fn, lex.sourcearr[lex.line-1], false)
	lex.posi = startPos
	lex.advanceN(len(match))
	endPos := helpers.NewPosition(lex.pos, lex.line, lex.col, lex.pos, lex.fn, lex.sourcearr[lex.line-1], false)

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
