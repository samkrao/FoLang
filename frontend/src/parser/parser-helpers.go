package parser

import (
	"fmt"
	"strings"

	"github.com/samkrao/fo-lang/frontend/src/helpers"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
	"golang.org/x/exp/slices"
)

// Contains reports whether the string slice s contains str.
func Contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
}

// currentToken returns the token under the parser cursor, or a synthetic EOF
// token when the cursor has moved past the token slice.
func (p *parser) currentToken() scanlex.Token {
	if p.pos < len(p.tokens) {
		return p.tokens[p.pos]
	}
	return scanlex.NewUniqueToken(scanlex.EOF, "EOF", helpers.NilPosition, helpers.NilPosition)
}

// advance returns the current token and moves the parser cursor forward by one
// token when possible.
func (p *parser) advance() scanlex.Token {
	tk := p.currentToken()
	if p.pos < len(p.tokens) && p.currentToken().Kind != scanlex.EOF {
		p.pos++
	} else {
		err_ := p.errorObj(nil, "Already at the End")
		p.addErr(err_)
	}

	return tk
}

// advanceN advances the parser by `n` tokens.
func (p *parser) advanceN(n int) {
	for i := 0; i < n; i++ {
		p.advance()
	}
}

// backtrack moves the parser cursor back by one token and returns the new
// current token.
func (p *parser) backtrack() scanlex.Token {
	if p.pos > 0 {
		p.pos--
	} else {
		err_ := p.errorObj(nil, "Already at the Start")
		p.addErr(err_)
	}
	tk := p.currentToken()
	return tk

}

// hasTokens reports whether the parser can still consume non-EOF tokens.
func (p *parser) hasTokens() bool {
	return p.pos < len(p.tokens) && p.currentTokenKind() != scanlex.EOF
}

// nextToken returns the token `i` steps ahead, clamping to the available token
// range when needed.
func (p *parser) nextToken(i int) scanlex.Token {
	if p.pos+i < len(p.tokens) {
		return p.tokens[p.pos+i]
	} else if p.pos < len(p.tokens) {
		return p.tokens[p.pos]
	} else {
		return p.tokens[p.pos-1]
	}
}

// nextTokenSafe returns lookahead token `n`, or EOF when the requested token is
// out of range.
func (p *parser) nextTokenSafe(n int) scanlex.Token {
	if n <= 0 {
		return p.currentToken()
	}
	t := p.nextToken(n)
	if t.Kind == scanlex.EOF {
		return t
	}
	if t.Kind == 0 {
		return scanlex.Token{Kind: scanlex.EOF}
	}
	return t
}

// previousToken returns the token `i` steps behind the current cursor.
func (p *parser) previousToken(i int) scanlex.Token {
	if p.pos-i < 0 {
		return p.currentToken()
	}

	return p.tokens[p.pos-i]
}

// currentTokenKind is a convenience wrapper returning currentToken().Kind with
// EOF fallback.
func (p *parser) currentTokenKind() scanlex.TokenKind {
	if p.pos < len(p.tokens) {
		return p.tokens[p.pos].Kind
	}
	return scanlex.EOF
}

// madeProgress reports whether parsing advanced past the previously observed
// token position.
func (p *parser) madeProgress(prev scanlex.Token) bool {
	cur := p.currentToken()
	return cur.StartPos != prev.StartPos || cur.EndPos != prev.EndPos || cur.Kind != prev.Kind
}

// advanceIfStuck forces one-token progress when the parser is still at the same
// token after an attempted parse branch.
func (p *parser) advanceIfStuck(prev scanlex.Token) {
	if !p.madeProgress(prev) {
		p.advance()
	}
}

// expectError consumes the current token if it matches `expectedKind`,
// otherwise it returns a detailed syntax error without consuming.
func (p *parser) expectError(expectedKind scanlex.TokenKind, err any) (scanlex.Token, helpers.ErrorInterface) {
	token := p.currentToken()
	kind := token.Kind

	if token.Kind == scanlex.EOF {
		ct := p.previousToken(1)
		newEnd := &helpers.Position{
			Idx:  ct.EndPos.Idx + 1,
			Ln:   ct.EndPos.Ln,
			Col:  ct.EndPos.Col + 1,
			Fn:   ct.EndPos.Fn,
			Ftxt: ct.EndPos.Ftxt,
		}

		err := helpers.NewExpectedCharError(*ct.EndPos, *newEnd, scanlex.TokenKindString(expectedKind))
		return token, err
	}
	if kind != expectedKind {
		if err == nil {
			err = fmt.Sprintf("Expected %s %s instead\n", scanlex.TokenKindString(expectedKind), scanlex.TokenKindString(kind))
		}

		str := fmt.Sprintf("Expected %s but recieved %s instead\n", scanlex.TokenKindString(expectedKind), scanlex.TokenKindString(kind))
		return scanlex.NewUniqueToken(scanlex.INVALID, "INVALID", token.StartPos, token.EndPos), helpers.NewExpectedCharError(*token.StartPos, *token.EndPos, str)

	}
	v := p.advance()

	return v, nil
}

// expectAnyWoAdv validates that the current token is one of the requested
// kinds, but does not advance the cursor.
func (p *parser) expectAnyWoAdv(expectedKinds ...scanlex.TokenKind) (scanlex.Token, helpers.ErrorInterface) {
	token := p.currentToken()
	kind := token.Kind
	if slices.Contains(expectedKinds, kind) {
		return p.currentToken(), nil

	} else if token.Kind == scanlex.EOF {
		ct := p.previousToken(1)
		exk := ""
		for _, ex := range expectedKinds {
			exk += scanlex.TokenKindString(ex) + " ,"
		}
		exk = "one of the tokens " + strings.TrimSuffix(exk, ",")
		newEnd := &helpers.Position{
			Idx:  ct.EndPos.Idx + 1,
			Ln:   ct.EndPos.Ln,
			Col:  ct.EndPos.Col + 1,
			Fn:   ct.EndPos.Fn,
			Ftxt: ct.EndPos.Ftxt,
		}

		err := helpers.NewExpectedCharError(*ct.EndPos, *newEnd, exk)

		//helpers.NewInvalidSyntaxError(*token.StartPos, *token.EndPos, err)
		return token, err
	}
	exk1 := ""
	for _, ex := range expectedKinds {
		exk1 += scanlex.TokenKindString(ex) + " ,"
	}
	exk1 = "one of the tokens " + strings.TrimSuffix(exk1, ",")

	err := fmt.Sprintf("Expected %s but recieved %s instead\n", exk1, scanlex.TokenKindString(kind))

	return scanlex.NewUniqueToken(scanlex.INVALID, "INVALID", token.StartPos, token.EndPos), helpers.NewInvalidSyntaxError(*token.StartPos, *token.EndPos, err)
}

// expectAny consumes and returns the current token when it matches one of the
// requested kinds.
func (p *parser) expectAny(expectedKinds ...scanlex.TokenKind) (scanlex.Token, helpers.ErrorInterface) {
	token := p.currentToken()
	kind := token.Kind
	if slices.Contains(expectedKinds, kind) {
		return p.advance(), nil

	} else if token.Kind == scanlex.EOF {
		ct := p.previousToken(1)
		exk := ""
		for _, ex := range expectedKinds {
			exk += scanlex.TokenKindString(ex) + " ,"
		}
		exk = "one of the tokens " + strings.TrimSuffix(exk, ",")
		newEnd := &helpers.Position{
			Idx:  ct.EndPos.Idx + 1,
			Ln:   ct.EndPos.Ln,
			Col:  ct.EndPos.Col + 1,
			Fn:   ct.EndPos.Fn,
			Ftxt: ct.EndPos.Ftxt,
		}

		err := helpers.NewExpectedCharError(*ct.EndPos, *newEnd, exk)

		//helpers.NewInvalidSyntaxError(*token.StartPos, *token.EndPos, err)
		return token, err
	}
	exk1 := ""
	for _, ex := range expectedKinds {
		exk1 += scanlex.TokenKindString(ex) + " ,"
	}
	exk1 = "one of the tokens " + strings.TrimSuffix(exk1, ",")

	err := fmt.Sprintf("Expected %s but recieved %s instead\n", exk1, scanlex.TokenKindString(kind))

	return scanlex.NewUniqueToken(scanlex.INVALID, "INVALID", token.StartPos, token.EndPos), helpers.NewInvalidSyntaxError(*token.StartPos, *token.EndPos, err)
}

// expect is the common single-kind convenience wrapper over expectError.
func (p *parser) expect(expectedKind scanlex.TokenKind) (scanlex.Token, helpers.ErrorInterface) {
	return p.expectError(expectedKind, nil)
}

// errorExpection builds a parser error of the requested type at the current
// token position.
func (p *parser) errorExpection(str string, errType helpers.ErrorType) helpers.ErrorInterface {
	switch errType {
	case helpers.IllegalChar:
		return helpers.NewIllegalCharError(*p.currentToken().StartPos, *p.currentToken().EndPos, str)
	case helpers.IllegalString:
		return helpers.NewIllegalStringException(*p.currentToken().StartPos, *p.currentToken().EndPos, str)
	case helpers.AlreadyDeclared:
		return helpers.NewAlreadyDeclaredException(*p.currentToken().StartPos, *p.currentToken().EndPos, str)
	case helpers.ExpectedChar:
		return helpers.NewExpectedCharError(*p.currentToken().StartPos, *p.currentToken().EndPos, str)
	case helpers.ExpectedToken:
		return helpers.NewExpectedTokenError(*p.currentToken().StartPos, *p.currentToken().EndPos, str)
	case helpers.InvalidSyntax:
		return helpers.NewInvalidSyntaxError(*p.currentToken().StartPos, *p.currentToken().EndPos, str)
	case helpers.IllegalVariableAssignment:
		return helpers.NewIllegalVariableAssignmentException(*p.currentToken().StartPos, *p.currentToken().EndPos, str)
	case helpers.VariableNotDecl:
		return helpers.NewVariableNotDeclared(*p.currentToken().StartPos, *p.currentToken().EndPos, str)
	case helpers.RuntTime:
		return helpers.NewRTError(*p.currentToken().StartPos, *p.currentToken().EndPos, str, nil)
	case helpers.UnSupported:
		return helpers.NewUnSupportedException(*p.currentToken().StartPos, *p.currentToken().EndPos, str)
	case helpers.NotFound:
		return helpers.NewNotFoundException(*p.currentToken().StartPos, *p.currentToken().EndPos, str)
	case helpers.ReservedKeyword:
		return helpers.NewReservedKeywordException(*p.currentToken().StartPos, *p.currentToken().EndPos, str)
	default:
		fmt.Println("Unknow " + p.currentToken().Value + str)
	}
	return helpers.NewError(*p.currentToken().StartPos, *p.currentToken().EndPos, "Unknown Error", str)

}

// errorObj builds an expected-token style error around the current token.
func (p *parser) errorObj(expectedKind *scanlex.Token, str string) helpers.ErrorInterface {
	var expKind scanlex.Token = scanlex.DummyNode
	if expectedKind != nil {
		expKind = *expectedKind
	}
	err := ""
	token := p.currentToken()
	if expectedKind != nil {
		err = fmt.Sprintf("Expected %s but recieved %s instead\n", scanlex.TokenKindString(expKind.Kind), scanlex.TokenKindString(p.currentTokenKind()))
		return helpers.NewExpectedTokenError(*token.StartPos, *token.EndPos, err)
	} else {
		err = fmt.Sprintf("Found %s Token\n", scanlex.TokenKindString(p.currentTokenKind()))
		return helpers.NewExpectedTokenErrorName(*token.StartPos, *token.EndPos, str, err)

	}
}

// errorObj_frm_st is like errorObj but accepts a string list of expected token
// names rather than a concrete scanlex.Token.
func (p *parser) errorObj_frm_st(expectedKind []string, str string) helpers.ErrorInterface {
	err := ""
	token := p.currentToken()

	if len(expectedKind) > 0 {
		st := ""
		for _, tk := range expectedKind {
			st += tk + ","
		}
		err = fmt.Sprintf("Expected %s but recieved %s instead\n", st, scanlex.TokenKindString(p.currentTokenKind()))
		return helpers.NewExpectedTokenError(*token.StartPos, *token.EndPos, err)
	} else {
		err = fmt.Sprintf("Found %s Token\n", scanlex.TokenKindString(p.currentTokenKind()))
		return helpers.NewExpectedTokenErrorName(*token.StartPos, *token.EndPos, str, err)

	}
}

// checkKindsandTypes tests whether the current token or lookahead token is a
// FoLang built-in kind/type marker and maps it to a statement kind.
//
// Feature examples:
//
//	x co.lang.int;
//	Employee co.lang.struct = { ... }
//	Api co.lang.module = { ... }
func checkKindsandTypes(p *parser, what BuiltIn, index int, val string) (bool, StmtKind) {
	var tok scanlex.Token
	if index == 0 {
		tok = p.currentToken()
	} else {
		tok = p.nextToken(index)
	}
	nameVal := tok.Value
	if what == KIND_ && tok.Kind == scanlex.BUILT_IN_KIND {

	} else if what == TYPE_ && tok.Kind == scanlex.BUILT_IN_TYPE {

	} else {
		return checkFunctionType(p, index), FUN
	}
	switch what {
	case KIND_:
		if slices.Contains(scanlex.Builtin_Kinds, nameVal) {
			if val != "any" {
				if nameVal == val {
					return true, kind_to_StmtKind[nameVal]
				}
			} else {
				return true, kind_to_StmtKind[nameVal]
			}
		}
	case TYPE_:
		if slices.Contains(scanlex.Builtin_Kinds, nameVal) {
			if val != "any" {
				if nameVal == val {
					return true, TYPE
				}
			} else {
				return true, TYPE
			}
		}
	default:
		return false, NASTMT
	}
	return false, NASTMT
}

// checkFunctionType performs lightweight lookahead to decide whether the token
// stream from the current position looks like a function signature ending in
// `->`.
//
// Feature examples:
//
//	(a co.lang.int)->(co.lang.int)
//	(a co.lang.int)(b co.lang.int)->(co.lang.int)
func checkFunctionType(p *parser, idx int) bool {
	index := idx
	//(a int)()->
	flag := false
	for {
		for p.nextToken(index).Kind != scanlex.CLOSE_PAREN && p.nextToken(index).Kind != scanlex.EOF {
			index += 1

		}
		if p.nextToken(index).Kind == scanlex.CLOSE_PAREN {
			flag = true
		}
		if p.nextToken(index+1).Kind == scanlex.OPEN_PAREN {
			index += 1
			flag = false
		}
		if p.nextToken(index+1).Kind == scanlex.EOF {
			break
		}

		if flag {
			break
		}
	}
	return p.nextToken(index+1).Kind == scanlex.ARROW
}

// tryCastToStringMap is a small helper used when metadata values may already
// have been normalized into map[string]string.
func tryCastToStringMap(v any) (map[string]string, bool) {
	m, ok := v.(map[string]string)
	return m, ok
}
