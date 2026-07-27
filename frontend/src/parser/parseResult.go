package parser

import (
	"github.com/samkrao/fo-lang/frontend/src/ast"

	"github.com/samkrao/fo-lang/frontend/src/helpers"
)

// ParseResult holds the outcome of a parse operation, including the AST node, errors, and advancement state.
type ParseResult struct {
	Errors                     []helpers.ErrorInterface
	Node                       ast.SET
	LastRegisteredAdvanceCount int
	AdvanceCount               int
	ToReverseCount             int
	BuildLibs                  bool
	OtherDetails               map[string]any
	Empty                      bool
	Break                      bool
	LetBindings                bool
}

// RegisterAdvancement records that the parser advanced by one token.
func (pr *ParseResult) RegisterAdvancement() {
	pr.LastRegisteredAdvanceCount = 1
	pr.AdvanceCount++
}

// Register merges a child ParseResult into this one, accumulating errors and advancement counts.
func (pr *ParseResult) Register(res *ParseResult) ast.SET {
	pr.LastRegisteredAdvanceCount = res.AdvanceCount
	pr.AdvanceCount += res.AdvanceCount
	if res.Errors != nil {
		pr.Errors = append(pr.Errors, res.Errors...)
	}
	return res.Node
}

// TryRegister attempts to register a child result, rolling back on error instead of propagating it.
func (pr *ParseResult) TryRegister(res *ParseResult) ast.SET {
	if res.Errors != nil {
		pr.ToReverseCount = res.AdvanceCount
		return nil
	}
	return pr.Register(res)
}

// Success marks this ParseResult as successful with the given AST node.
func (pr *ParseResult) Success(node ast.SET) *ParseResult {
	pr.Node = node
	return pr
}

// Failure records an error in this ParseResult.
func (pr *ParseResult) Failure(err helpers.ErrorInterface) *ParseResult {
	if pr.Errors == nil || pr.LastRegisteredAdvanceCount == 0 {
		pr.Errors = append(pr.Errors, err)
	}
	return pr
}
