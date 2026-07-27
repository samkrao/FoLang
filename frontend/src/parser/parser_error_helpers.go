package parser

import (
	"errors"
	"fmt"

	"github.com/samkrao/fo-lang/frontend/src/foerrors"
	"github.com/samkrao/fo-lang/frontend/src/helpers"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// HasErrors reports whether the parser has accumulated any errors.
func (p *parser) HasErrors() bool { return len(p.Errors) > 0 }

// ParserError aggregates one or more parse errors into a single error value.
type ParserError struct{ Errors []error }

// Error returns a human-readable summary of the parser errors.
func (e ParserError) Error() string {
	if len(e.Errors) == 0 {
		return "parser: no errors"
	}
	if len(e.Errors) == 1 {
		return fmt.Sprintf("parser: %v", e.Errors[0])
	}
	return fmt.Sprintf("parser: %d errors (first: %v)", len(e.Errors), e.Errors[0])
}

// AsError converts accumulated parse errors into a single Go error, or nil if none.
func (p *parser) AsError() error {
	if !p.HasErrors() {
		return nil
	}
	arr := make([]error, 0, len(p.Errors))
	for _, e := range p.Errors {
		if e == nil {
			continue
		}
		if err, ok := any(e).(error); ok {
			arr = append(arr, err)
		} else {
			arr = append(arr, errors.New(fmt.Sprint(e)))
		}
	}
	return ParserError{Errors: arr}
}

func (p *parser) errorLimitReached() bool {
	return len(p.Errors) >= foerrors.MaxParseErrors
}

func (p *parser) addErrCap(err any) {
	p.addErr(err)
}

func (p *parser) ensureErrorList() {
	if p.Errors == nil {
		p.Errors = make([]helpers.ErrorInterface, 0, 4)
	}
}

func (p *parser) addErrs(errs ...any) {
	for _, err := range errs {
		p.addErr(err)
	}
}
func (p *parser) addErr(err any) {
	if err == nil {
		return
	}
	if len(p.Errors) >= foerrors.MaxParseErrors {
		foerrors.HandleErrors(p.Errors...)
	}
	p.ensureErrorList()
	switch e := err.(type) {
	case helpers.ErrorInterface:
		p.Errors = append(p.Errors, e)
	case []helpers.ErrorInterface:
		remaining := foerrors.MaxParseErrors - len(p.Errors)
		if len(e) > remaining {
			p.Errors = append(p.Errors, e[:remaining]...)
		} else {
			p.Errors = append(p.Errors, e...)
		}
	}
}
func (p *parser) expectOrRecover(kind scanlex.TokenKind, syncKinds ...scanlex.TokenKind) bool {
	if _, err := p.expect(kind); err != nil {
		p.addErr(err)
		if len(syncKinds) > 0 {
			p.sync(syncKinds...)
		}
		return false
	}
	return true
}

func (p *parser) sync(until ...scanlex.TokenKind) {
	if len(until) == 0 {
		return
	}
	set := make(map[scanlex.TokenKind]struct{}, len(until))
	for _, k := range until {
		set[k] = struct{}{}
	}
	for {
		t := p.currentToken()
		if t.Kind == scanlex.EOF {
			return
		}
		if _, ok := set[t.Kind]; ok {
			return
		}
		p.advance()
	}
}

// syncUntilBalanced respects bracket balancing and also stops at provided sentinels.
func (p *parser) syncUntilBalanced(open, close scanlex.TokenKind, alsoStopAt ...scanlex.TokenKind) {
	stop := make(map[scanlex.TokenKind]struct{}, len(alsoStopAt))
	for _, k := range alsoStopAt {
		stop[k] = struct{}{}
	}
	depth := 0
	for {
		t := p.currentToken()
		switch {
		case t.Kind == scanlex.EOF:
			return
		case t.Kind == open:
			depth++
			p.advance()
		case t.Kind == close:
			if depth == 0 {
				return
			}
			depth--
			p.advance()
		default:
			if _, ok := stop[t.Kind]; ok && depth == 0 {
				return
			}
			p.advance()
		}
	}
}
