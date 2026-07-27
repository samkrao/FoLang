package parser

import (
	"slices"

	"github.com/samkrao/fo-lang/frontend/src/ast"
	"github.com/samkrao/fo-lang/frontend/src/scanlex"
)

// ParenKind classifies the syntactic role of a parenthesized group (function, pattern, call, etc.).
type ParenKind int

const (
	Function ParenKind = iota
	FunctionPattern
	CLOSURE_CURRY
	FunctionObject
	Receiver
	GRP_EXPR_CALL
	ILLEGAL_GRP
)

type nextIndex = int

func stmtKind(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt, ind int) (nextIndex, ParenKind) {
	defer p.traceCurrent()()

	count := 1
	start := 0
	if ind != -1 {
		count = ind + 1
		start = ind
	}
	illegal := false
	open := 0

	if (start == 0 && p.currentTokenKind() == scanlex.OPEN_PAREN) || (start != 0 && p.nextTokenSafe(start).Kind == scanlex.OPEN_PAREN) {
		for {
			for p.nextTokenSafe(count).Kind != scanlex.CLOSE_PAREN {
				if p.nextTokenSafe(ind).Kind == scanlex.OPEN_PAREN {
					open += 1
				} else if p.nextTokenSafe(ind).Kind == scanlex.CLOSE_PAREN {
					open -= 1
				}

				count += 1
				if p.nextTokenSafe(count).Kind == scanlex.EOF {
					illegal = true
					break
				}
			}
			if !illegal {

				count += 1
				//curried functions
				if p.nextTokenSafe(count).Kind == scanlex.OPEN_PAREN {
					count += 1
					open = 0
					continue
				} else {
					break
				}
			}
		}
		if !illegal {
			count += 1
			if p.nextTokenSafe(count).Kind == scanlex.ARROW {
				return count - 1, Function
			} else if p.nextTokenSafe(count).Kind == scanlex.EQGT {
				if p.nextTokenSafe(count+2).Kind == scanlex.OPEN_PAREN {
					return count - 1, CLOSURE_CURRY
				} else if p.nextTokenSafe(count+2).Kind == scanlex.OPEN_CURLY {
					return count - 1, FunctionPattern
				}
				return count - 1, CLOSURE_CURRY
			} else if p.nextTokenSafe(count).Kind == scanlex.IDENTIFIER {
				return count - 1, Receiver
			}
		}
	} else {
		count -= 1
	}
	if illegal {
		return count, ILLEGAL_GRP
	}
	return count, GRP_EXPR_CALL

}

func isLetBindings(p *parser, index int, ddaps map[scanlex.DirectiveKind][]ast.Stmt) bool {
	defer p.traceCurrent()()

	start := 0
	if index != -1 {
		start = index
	}

	if (start == 0 && p.currentTokenKind() == scanlex.KEYWORD && p.currentToken().Value == "let") || (start != 0 && p.nextTokenSafe(start).Kind == scanlex.KEYWORD && p.nextTokenSafe(start).Value == "let") {
		if p.nextTokenSafe(start+1).Kind == scanlex.IDENTIFIER {
			return true
		}
	}
	return false
}

// check package.fol in a give package folder

func existsPackageFile(p *parser) bool {
	return false
}
func isLibrary(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) bool {
	dv := ddaps[scanlex.DIRECTIVE]
	for _, v := range dv {
		if v.GetName() == "@co.dap.library" {
			return true
		}
	}

	return false
}

func isPackageSource(p *parser, ddaps map[scanlex.DirectiveKind][]ast.Stmt) bool {
	return p.fileInfo.PackagePath == ""

}

func isClassDecl(p *parser, index int) bool {
	defer p.traceCurrent()()

	start := 0
	if index != -1 {
		start = index
	}

	if (start == 0 && p.currentTokenKind() == scanlex.IDENTIFIER) || (start != 0 && p.nextTokenSafe(start).Kind == scanlex.IDENTIFIER) {

		if p.nextTokenSafe(start+1).Value == "co.lang.class" {
			return true

		}
	}
	return false

}

func isInterfaceDecl(p *parser, index int) bool {
	defer p.traceCurrent()()

	start := 0
	if index != -1 {
		start = index
	}

	if (start == 0 && p.currentTokenKind() == scanlex.IDENTIFIER) || (start != 0 && p.nextTokenSafe(start).Kind == scanlex.IDENTIFIER) {

		if p.nextTokenSafe(start+1).Value == "co.lang.interface" {
			return true

		}
	}
	return false

}

func isStructDecl(p *parser, index int) bool {
	defer p.traceCurrent()()

	start := 0
	if index != -1 {
		start = index
	}

	if (start == 0 && p.currentTokenKind() == scanlex.IDENTIFIER) || (start != 0 && p.nextTokenSafe(start).Kind == scanlex.IDENTIFIER) {

		val := p.nextTokenSafe(start + 1).Value
		if val == "co.lang.struct" || val == "co.lang.cstruct" {
			return true
		}
	}
	return false

}

func isUnionDecl(p *parser, index int) bool {
	defer p.traceCurrent()()

	start := 0
	if index != -1 {
		start = index
	}
	if (start == 0 && p.currentTokenKind() == scanlex.IDENTIFIER) || (start != 0 && p.nextTokenSafe(start).Kind == scanlex.IDENTIFIER) {
		if p.nextTokenSafe(start+1).Value == "co.lang.union" {
			return true
		}
	}
	return false
}

var special = []string{"@co.dap.macro", "@co.dap.operator", "@co.dap.annotation",
	"@co.dap.decorator", "@co.dap.pragma", "@co.dap.directive", "@co.dap.template",
	"@co.dap.inline", "@co.dap.generic", "@co.dap.monoid", "@co.dap.monod",
	"@co.dap.functor", "@co.dap.applicative", "@co.dap.transformer",
	"@co.dap.matcher", "@co.dap.delegate", "@co.dap.indexer", "@co.dap.hokrt",
	"@co.dap.extension", "@co.dap.process", "@co.dap.anonymous", "@co.dap.exec",
	"@co.dap.spawn", "@co.dap.fork", "@co.dap.async", "@co.dap.thread", "@co.dap.coroutine",
	"@co.dap.task", "@co.dap.fiber", "@co.dap.greenlet", "@co.dap.subroutine",
	"@co.dap.generator", "@co.dap.goroutine", "@co.dap.event", "@co.dap.actor", "@co.dap.continuation",
	"@co.dap.csp", "@co.dap.callback", "@co.dap.promise", "@co.dap.defer",
	"@co.dap.lambda", "@co.dap.channel", "@co.dap.abstract", "@co.dap.final",
	"@co.dap.method.instance", "@co.dap.method.static", "@co.dap.method.class",
	"@co.dap.method.object", "@co.dap.virtual", "@co.dap.override", "@co.dap.overload",
	"@co.dap.shadow", "@co.dap.new", "@co.dap.sealed", "@co.dap.modulesig",
	"@co.dap.module", "@co.dap.use", "@co.dap.typeclass"}

func isFuncDecl(p *parser, index int, ddaps map[scanlex.DirectiveKind][]ast.Stmt) bool {
	defer p.traceCurrent()()

	_, kind := stmtKind(p, ddaps, index)
	if kind == Function {

		stmts := ddaps[scanlex.ANNOTATION]
		stmts = append(stmts, ddaps[scanlex.DECORATOR]...)
		flag := false
		for _, stmt := range stmts {
			dir, ok := stmt.(ast.DirectiveStmt)
			if !ok {
				continue
			}
			if slices.Contains(special, dir.Name) {

				continue
			} else {
				flag = true
			}
		}
		return flag

	}
	return false
}

func isMacrocDecl(p *parser, index int, ddaps map[scanlex.DirectiveKind][]ast.Stmt) bool {
	defer p.traceCurrent()()

	_, kind := stmtKind(p, ddaps, index)
	if kind == Function {

		stmts := ddaps[scanlex.ANNOTATION]
		flag := false
		for _, stmt := range stmts {
			dir, ok := stmt.(ast.DirectiveStmt)
			if !ok {
				continue
			}
			if dir.Name == "@co.dap.macro" {

				flag = true
				break
			}
		}
		return flag

	}
	return false
}
func isTemplatecDecl(p *parser, index int, ddaps map[scanlex.DirectiveKind][]ast.Stmt) bool {

	_, kind := stmtKind(p, ddaps, index)
	if kind == Function {

		stmts := ddaps[scanlex.ANNOTATION]
		flag := false
		for _, stmt := range stmts {
			dir, ok := stmt.(ast.DirectiveStmt)
			if !ok {
				continue
			}
			if dir.Name == "@co.dap.template" {

				flag = true
				break
			}
		}
		return flag

	}
	return false
}

func isGenericDecl(p *parser, index int, ddaps map[scanlex.DirectiveKind][]ast.Stmt) bool {
	defer p.traceCurrent()()

	_, kind := stmtKind(p, ddaps, index)
	if kind == Function {

		stmts := ddaps[scanlex.ANNOTATION]
		flag := false
		for _, stmt := range stmts {
			dir, ok := stmt.(ast.DirectiveStmt)
			if !ok {
				continue
			}
			if dir.Name == "@co.dap.generic" {

				flag = true
				break
			}
		}
		return flag

	}
	return false
}

func isOperatorDecl(p *parser, index int, ddaps map[scanlex.DirectiveKind][]ast.Stmt) bool {

	_, kind := stmtKind(p, ddaps, index)
	if kind == Function {

		stmts := ddaps[scanlex.ANNOTATION]
		flag := false
		for _, stmt := range stmts {
			dir, ok := stmt.(ast.DirectiveStmt)
			if !ok {
				continue
			}
			if dir.Name == "@co.dap.operator" {

				flag = true
				break
			}
		}
		return flag

	}
	return false
}

func isAnncDecl(p *parser, index int, ddaps map[scanlex.DirectiveKind][]ast.Stmt) bool {
	defer p.traceCurrent()()

	_, kind := stmtKind(p, ddaps, index)
	if kind == Function {

		stmts := ddaps[scanlex.ANNOTATION]
		flag := false
		for _, stmt := range stmts {
			dir, ok := stmt.(ast.DirectiveStmt)
			if !ok {
				continue
			}
			if dir.Name == "@co.dap.annotation" {

				flag = true
				break
			}
		}
		return flag

	}
	return false
}

func isDirectivecDecl(p *parser, index int, ddaps map[scanlex.DirectiveKind][]ast.Stmt) bool {
	defer p.traceCurrent()()

	_, kind := stmtKind(p, ddaps, index)
	if kind == Function {

		stmts := ddaps[scanlex.DIRECTIVE]
		flag := false
		for _, stmt := range stmts {
			dir, ok := stmt.(ast.DirectiveStmt)
			if !ok {
				continue
			}
			if dir.Name == "@co.dap.directive" {

				flag = true
				break
			}
		}
		return flag

	}
	return false
}

func isPragmacDecl(p *parser, index int, ddaps map[scanlex.DirectiveKind][]ast.Stmt) bool {
	defer p.traceCurrent()()

	_, kind := stmtKind(p, ddaps, index)
	if kind == Function {

		stmts := ddaps[scanlex.DIRECTIVE]
		flag := false
		for _, stmt := range stmts {
			dir, ok := stmt.(ast.DirectiveStmt)
			if !ok {
				continue
			}
			if dir.Name == "@co.dap.pragma" {

				flag = true
				break
			}
		}
		return flag

	}
	return false
}

func isDecoratorcDecl(p *parser, index int, ddaps map[scanlex.DirectiveKind][]ast.Stmt) bool {
	defer p.traceCurrent()()

	_, kind := stmtKind(p, ddaps, index)
	if kind == Function {

		stmts := ddaps[scanlex.ANNOTATION]
		flag := false
		for _, stmt := range stmts {
			dir, ok := stmt.(ast.DirectiveStmt)
			if !ok {
				continue
			}
			if dir.Name == "@co.dap.decorator" {

				flag = true
				break
			}
		}
		return flag

	}
	return false
}

var rangeOps = []scanlex.TokenKind{scanlex.DOT_DOT, scanlex.LT_DOT_DOT, scanlex.DOT_DOT_LT, scanlex.LT_DOT_DOT_LT}

func isVarDefn(p *parser, index int, ddaps map[scanlex.DirectiveKind][]ast.Stmt) bool {
	defer p.traceCurrent()()

	start := 0
	if index != -1 {
		start = index
	}

	if (start == 0 && p.currentTokenKind() == scanlex.IDENTIFIER) || (start != 0 && p.nextTokenSafe(start).Kind == scanlex.IDENTIFIER) {
		if slices.Contains(scanlex.Builtin_types, p.nextTokenSafe(start+1).Value) {
			return true
		} else if p.nextTokenSafe(start+1).Kind == scanlex.WALRUS {
			if slices.Contains(rangeOps, p.nextTokenSafe(start+2).Kind) || slices.Contains(rangeOps, p.nextTokenSafe(start+3).Kind) {
				return false
			}
			return true
		} else if p.nextTokenSafe(start+1).Kind == scanlex.QEQ {
			if slices.Contains(rangeOps, p.nextTokenSafe(start+2).Kind) || slices.Contains(rangeOps, p.nextTokenSafe(start+3).Kind) {
				return false
			}
			return true
		}
	}
	return true
}
func isRangeDecl(p *parser, index int, ddaps map[scanlex.DirectiveKind][]ast.Stmt) bool {
	defer p.traceCurrent()()

	start := 0
	if index != -1 {
		start = index
	}

	if (start == 0 && p.currentTokenKind() == scanlex.IDENTIFIER) || (start != 0 && p.nextTokenSafe(start).Kind == scanlex.IDENTIFIER) {
		if p.nextTokenSafe(start+1).Kind == scanlex.WALRUS {
			if slices.Contains(rangeOps, p.nextTokenSafe(start+2).Kind) || slices.Contains(rangeOps, p.nextTokenSafe(start+3).Kind) {
				return true
			}
			return false
		} else if p.nextTokenSafe(start+1).Kind == scanlex.QEQ {
			if slices.Contains(rangeOps, p.nextTokenSafe(start+2).Kind) || slices.Contains(rangeOps, p.nextTokenSafe(start+3).Kind) {
				return true
			}
			return false
		}
	}
	return false
}
func isVarDecl(p *parser, index int, ddaps map[scanlex.DirectiveKind][]ast.Stmt) bool {
	defer p.traceCurrent()()

	stmts := ddaps[scanlex.ANNOTATION]
	flag := false
	for _, stmt := range stmts {
		dir, ok := stmt.(ast.DirectiveStmt)
		if !ok {
			continue
		}
		if dir.Name == "@co.dap.extern" {

			flag = true
			break
		}
	}
	return isVarDefn(p, index, ddaps) && flag
}

var typeDecls = []string{"co.lang.type", "co.lang.newtype",
	"co.lang.opaquetype", "co.lang.supertype", "co.lang.subtype",
	"co.lang.alias", "co.lang.typekind", "co.lang.dependentType",
	"co.lang.typetype", "co.lang.associatedtype"}

func isTypeDecl(p *parser, index int, ddaps map[scanlex.DirectiveKind][]ast.Stmt) bool {
	defer p.traceCurrent()()

	start := 0
	if index != -1 {
		start = index
	}

	if (start == 0 && p.currentTokenKind() == scanlex.IDENTIFIER) || (start != 0 && p.nextTokenSafe(start).Kind == scanlex.IDENTIFIER) {
		if slices.Contains(typeDecls, p.nextTokenSafe(start+1).Value) {
			return true
		}
	}
	return false
}

func isKindDecl(p *parser, index int, ddaps map[scanlex.DirectiveKind][]ast.Stmt) bool {
	defer p.traceCurrent()()

	start := 0
	if index != -1 {
		start = index
	}

	if (start == 0 && p.currentTokenKind() == scanlex.IDENTIFIER) || (start != 0 && p.nextTokenSafe(start).Kind == scanlex.IDENTIFIER) {
		if slices.Contains(scanlex.Builtin_Kinds, p.nextTokenSafe(start+1).Value) {
			return true
		}
	}
	return false
}

func isKindDefn(p *parser, index int, ddaps map[scanlex.DirectiveKind][]ast.Stmt) bool {
	defer p.traceCurrent()()

	start := 0
	if index != -1 {
		start = index
	}

	if (start == 0 && p.currentTokenKind() == scanlex.IDENTIFIER) || (start != 0 && p.nextTokenSafe(start).Kind == scanlex.IDENTIFIER) {
		if p.nextTokenSafe(start+1).Value == "co.lang.kind" || p.nextTokenSafe(start+1).Value == "co.lang.kindkind" {
			return true
		}
	}
	return false
}

func isDDapDecl(p *parser, index int, ddaps map[scanlex.DirectiveKind][]ast.Stmt) bool {
	defer p.traceCurrent()()

	start := 0
	if index != -1 {
		start = index
	}

	if _, ok := scanlex.Built_in_directives(p.nextTokenSafe(start).Value); ok {
		return true
	}
	return false
}

// isTypeConstructorDecl returns true when:
//  1. The current identifier is decorated with @co.dap.hokrt, and
//  2. It is followed by ( ... ) co.lang.type = <variants>
//
// Example: @co.dap.hokrt  Option(T) co.lang.type = Some(T) | None();
func isTypeConstructorDecl(p *parser, index int, ddaps map[scanlex.DirectiveKind][]ast.Stmt) bool {
	defer p.traceCurrent()()

	if _, ok := isAnnDec(ddaps, "@co.dap.hokrt"); !ok {
		return false
	}
	start := 0
	if index != -1 {
		start = index
	}
	if start == 0 {
		return p.currentTokenKind() == scanlex.IDENTIFIER
	}
	return p.nextTokenSafe(start).Kind == scanlex.IDENTIFIER
}

func collect_hints(p *parser, kind scanlex.DirectiveKind, ddaps map[scanlex.DirectiveKind][]ast.Stmt) []ast.Stmt {
	defer p.traceCurrent()()

	dapStmts := make([]ast.Stmt, 0)
	for p.currentTokenKind() == scanlex.ATDAP || p.currentTokenKind() == scanlex.BUILT_IN_DIRECTIVES {
		dr := parse_atdap(p, kind, ddaps)
		if dr.Break {
			break
		}
		if dr.Empty {
			// Directive was consumed but recorded as an error (unsupported
			// name or wrong kind for this call site); keep scanning for more.
			continue
		}
		dapStmts = append(dapStmts, dr.Node.(ast.Stmt))

	}
	return dapStmts
}
func isTypeClass(p *parser, index int, ddaps map[scanlex.DirectiveKind][]ast.Stmt) bool {
	return false
}

func isTemplateDecl(p *parser, index int, ddaps map[scanlex.DirectiveKind][]ast.Stmt) bool {
	return false
}
func isAnnDec(ddaps map[scanlex.DirectiveKind][]ast.Stmt, names ...string) (string, bool) {

	lst, ok := ddaps[scanlex.ANNOTATION]
	if !ok {
		return "", false
	}

	// Build a set of names we're interested in.
	wanted := make(map[string]struct{}, len(names))
	for _, n := range names {
		wanted[n] = struct{}{}
	}

	// Scan the directives once.
	for _, stmt := range lst {
		dir, ok := stmt.(ast.DirectiveStmt)
		if !ok {
			continue // or panic/log if this should never happen
		}
		if _, ok := wanted[dir.Name]; ok {
			// return the actual matched name
			return dir.Name, true
		}
	}

	return "", false
}
func getAnn(ddaps map[scanlex.DirectiveKind][]ast.Stmt, name string) (ast.DirectiveStmt, bool) {
	lst, ok := ddaps[scanlex.ANNOTATION]
	if !ok {
		return ast.DirectiveStmt{}, false
	}

	for _, stmt := range lst {
		dir, ok := stmt.(ast.DirectiveStmt)
		if !ok {
			continue
		}
		if dir.Name == name {
			return dir, true
		}
	}

	return ast.DirectiveStmt{}, false
}

func isAnnDec_G(ddaps map[scanlex.DirectiveKind][]ast.Stmt, names ...string) (string, bool) {

	// 1. Get the list of statements for Annotation_Decorator once
	statements, ok := ddaps[scanlex.ANNOTATION]
	if !ok {
		// If the map key doesn't exist, we can't find anything, so return false early.
		return "", false
	}

	// 2. Iterate through the target names list first (more efficient)
	for _, name := range names {
		// 3. Iterate through the list of statements
		for _, stmt := range statements {

			// 4. Use a type assertion combined with a check for nil/type safety
			dirStmt, ok := stmt.(ast.DirectiveStmt)
			if !ok {
				// If the statement is not a DirectiveStmt, skip it and continue the loop.
				continue
			}

			// 5. Compare the name
			if dirStmt.Name == name {
				return name, true // Return the matched name immediately
			}
		}
	}

	// If the loops complete without a match
	return "", false
}

func isFuncPattern(p *parser, index int, ddaps map[scanlex.DirectiveKind][]ast.Stmt) bool {
	defer p.traceCurrent()()

	_, kind := stmtKind(p, ddaps, index)
	if kind == FunctionPattern {
		return true
	}
	return false
}

func isClosureCurryExpr(p *parser, index int, ddaps map[scanlex.DirectiveKind][]ast.Stmt) bool {
	defer p.traceCurrent()()

	_, kind := stmtKind(p, ddaps, index)
	if kind == CLOSURE_CURRY {
		return true
	}
	return false
}

// isClosureDecl detects closure syntax: name(captures) => (params) = expr
func isClosureDecl(p *parser) bool {
	defer p.traceCurrent()()

	if p.currentTokenKind() != scanlex.IDENTIFIER || p.nextTokenSafe(1).Kind != scanlex.OPEN_PAREN {
		return false
	}
	i := 2
	depth := 1
	for depth > 0 {
		if p.nextTokenSafe(i).Kind == scanlex.OPEN_PAREN {
			depth++
		}
		if p.nextTokenSafe(i).Kind == scanlex.CLOSE_PAREN {
			depth--
		}
		if p.nextTokenSafe(i).Kind == scanlex.EOF {
			return false
		}
		i++
	}
	return p.nextTokenSafe(i).Kind == scanlex.EQGT
}

// isCurryDecl detects curried function syntax: name(params1)(params2) = expr
func isCurryDecl(p *parser) bool {

	defer p.traceCurrent()()

	if p.currentTokenKind() != scanlex.IDENTIFIER || p.nextTokenSafe(1).Kind != scanlex.OPEN_PAREN {
		return false
	}
	// First content must look like a parameter declaration (name type)
	if p.nextTokenSafe(2).Kind != scanlex.IDENTIFIER {
		return false
	}
	if p.nextTokenSafe(3).Kind != scanlex.BUILT_IN_TYPE && p.nextTokenSafe(3).Kind != scanlex.IDENTIFIER {
		return false
	}
	// Scan to matching )
	i := 2
	depth := 1
	for depth > 0 {
		if p.nextTokenSafe(i).Kind == scanlex.OPEN_PAREN {
			depth++
		}
		if p.nextTokenSafe(i).Kind == scanlex.CLOSE_PAREN {
			depth--
		}
		if p.nextTokenSafe(i).Kind == scanlex.EOF {
			return false
		}
		i++
	}
	// After first ), must be another (
	if p.nextTokenSafe(i).Kind != scanlex.OPEN_PAREN {
		return false
	}
	// Scan past second group
	i++
	depth = 1
	for depth > 0 {
		if p.nextTokenSafe(i).Kind == scanlex.OPEN_PAREN {
			depth++
		}
		if p.nextTokenSafe(i).Kind == scanlex.CLOSE_PAREN {
			depth--
		}
		if p.nextTokenSafe(i).Kind == scanlex.EOF {
			return false
		}
		i++
	}
	// After second ), must be = (not ->)
	return p.nextTokenSafe(i).Kind == scanlex.ASSIGNMENT
}
func isFunctionObject(p *parser, index int, ddaps map[scanlex.DirectiveKind][]ast.Stmt) bool {
	defer p.traceCurrent()()

	start := 0
	if index != -1 {
		start = index
	}

	if (start == 0 && p.currentTokenKind() == scanlex.IDENTIFIER) || (start != 0 && p.nextTokenSafe(start).Kind == scanlex.IDENTIFIER) {
		if p.nextTokenSafe(start+1).Value == "co.lang.function" {
			return true
		}
	}
	return false
}

func isLambda(p *parser, index int, ddaps map[scanlex.DirectiveKind][]ast.Stmt) bool {
	defer p.traceCurrent()()

	start := 0
	count := 0
	if index != -1 {
		start = index
		count = index
	}
	flag := false

	if (start == 0 && p.currentTokenKind() == scanlex.PIPE) || (start != 0 && p.nextTokenSafe(start).Kind == scanlex.PIPE) {
		count = count + 1
		for p.nextTokenSafe(count).Kind != scanlex.PIPE {
			count += 1
			if p.nextTokenSafe(count).Kind == scanlex.EOF {
				return false
			} else if p.nextTokenSafe(count).Kind == scanlex.PIPE {
				flag = true
				break
			}

		}
	}
	return flag
}

func isSigDecl(p *parser, index int, ddaps map[scanlex.DirectiveKind][]ast.Stmt) bool {
	defer p.traceCurrent()()

	start := 0
	if index != -1 {
		start = index
	}

	if (start == 0 && p.currentTokenKind() == scanlex.IDENTIFIER) || (start != 0 && p.nextTokenSafe(start).Kind == scanlex.IDENTIFIER) {
		if p.nextTokenSafe(start+1).Value == "co.lang.signature" {
			return true
		}
	}
	return false
}
func isModuleDecl(p *parser, index int, ddaps map[scanlex.DirectiveKind][]ast.Stmt) bool {
	defer p.traceCurrent()()

	start := 0
	if index != -1 {
		start = index
	}

	// Name co.lang.module (spec syntax with @co.dap.module annotation)
	if (start == 0 && p.currentTokenKind() == scanlex.IDENTIFIER) || (start != 0 && p.nextTokenSafe(start).Kind == scanlex.IDENTIFIER) {
		if p.nextTokenSafe(start+1).Value == "co.lang.module" {
			return true
		}
	}
	return false
}

func isInstanceDecl(p *parser, index int) bool {
	defer p.traceCurrent()()

	start := 0
	if index != -1 {
		start = index
	}

	if (start == 0 && p.currentTokenKind() == scanlex.IDENTIFIER) || (start != 0 && p.nextTokenSafe(start).Kind == scanlex.IDENTIFIER) {
		if p.nextTokenSafe(start+1).Value == "co.lang.instance" {
			return true
		}
	}
	return false
}

func isMatcherInstanceDecl(p *parser, index int) bool {
	defer p.traceCurrent()()

	start := 0
	if index != -1 {
		start = index
	}

	if (start == 0 && p.currentTokenKind() == scanlex.IDENTIFIER) || (start != 0 && p.nextTokenSafe(start).Kind == scanlex.IDENTIFIER) {
		if p.nextTokenSafe(start+1).Value == "co.lang.matcher" {
			return true
		}
	}
	return false
}

func isObjectDecl(p *parser, index int) bool {
	defer p.traceCurrent()()

	start := 0
	if index != -1 {
		start = index
	}

	if (start == 0 && p.currentTokenKind() == scanlex.IDENTIFIER) || (start != 0 && p.nextTokenSafe(start).Kind == scanlex.IDENTIFIER) {
		if p.nextTokenSafe(start+1).Value == "co.lang.object" {
			return true
		}
	}
	return false
}

func isDelegateDecl(p *parser, index int, ddaps map[scanlex.DirectiveKind][]ast.Stmt) bool {
	defer p.traceCurrent()()

	start := 0
	if index != -1 {
		start = index
	}

	// Feature example:
	//   @co.dap.delegate
	//   someDelegate co.lang.delegate = (a co.lang.int, b co.lang.int) -> (co.lang.int);
	//
	// Delegates are declaration-shaped symbols, not regular functions. We only
	// classify them as delegates when the declaration is explicitly annotated
	// with @co.dap.delegate and the token stream has the expected
	//   IDENTIFIER co.lang.delegate
	// prefix.
	_, hasDelegateAnnotation := isAnnDec(ddaps, "@co.dap.delegate")
	if !hasDelegateAnnotation {
		return false
	}

	if (start == 0 && p.currentTokenKind() == scanlex.IDENTIFIER) || (start != 0 && p.nextTokenSafe(start).Kind == scanlex.IDENTIFIER) {
		if p.nextTokenSafe(start+1).Value == "co.lang.delegate" {
			return true
		}
	}
	return false
}

func isEnumDecl(p *parser, index int) bool {
	defer p.traceCurrent()()

	start := 0
	if index != -1 {
		start = index
	}

	if (start == 0 && p.currentTokenKind() == scanlex.IDENTIFIER) || (start != 0 && p.nextTokenSafe(start).Kind == scanlex.IDENTIFIER) {
		if p.nextTokenSafe(start+1).Value == "co.lang.enum" {
			return true
		}
	}
	return false
}

func isNamedBlockStmt(p *parser, index int) bool {
	defer p.traceCurrent()()

	start := 0
	if index != -1 {
		start = index
	}

	if (start == 0 && p.currentTokenKind() == scanlex.IDENTIFIER) || (start != 0 && p.nextTokenSafe(start).Kind == scanlex.IDENTIFIER) {
		if p.nextTokenSafe(start+1).Value == "co.lang.block" {
			return true
		}
	}
	return false
}

func isDependentTypeDecl(p *parser, index int) bool {
	defer p.traceCurrent()()

	start := 0
	if index != -1 {
		start = index
	}

	if (start == 0 && p.currentTokenKind() == scanlex.IDENTIFIER) || (start != 0 && p.nextTokenSafe(start).Kind == scanlex.IDENTIFIER) {
		if p.nextTokenSafe(start+1).Value == "co.lang.dependentType" {
			return true
		}
	}
	return false
}

func isLabel(p *parser, index int) bool {
	defer p.traceCurrent()()

	start := 0
	if index != -1 {
		start = index
	}

	if (start == 0 && p.currentTokenKind() == scanlex.IDENTIFIER) || (start != 0 && p.nextTokenSafe(start).Kind == scanlex.IDENTIFIER) {
		if p.nextTokenSafe(start+1).Kind == scanlex.COLON && p.nextTokenSafe(start+2).Kind == scanlex.OPEN_CURLY {
			return true
		}
	}
	return false
}
