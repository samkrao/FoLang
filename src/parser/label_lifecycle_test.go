package parser

import (
	"strings"
	"testing"

	"github.com/samkrao/fo-lang/src/ast"
)

// bodyOfFirstUnitFunction returns the statements of the first function declared
// by a single-function unit, which is where every statement-level case below
// puts its subject.
func bodyOfFirstUnitFunction(t *testing.T, body string) []ast.Stmt {
	t.Helper()
	source := "_ co.lang.unit = {\n    subject()->() = {\n" + body + "\n    }\n}"
	root, p := parsePackageSource(t, source, "control.unit.fol")
	if len(p.diags) != 0 {
		t.Fatalf("unit produced diagnostics: %v", p.diags)
	}

	unit := root.(ast.PackageStmt).Body[0].(ast.TypeDeclarationStmt)
	function := unit.Body[0].(ast.FunctionDeclarationStmt)
	return function.Body
}

// TestLabeledBlockAndLabeledLoopAreDistinguished covers labeled-block,
// labeled-loop-statement and the guard between them.
//
// Both spell the label the same way, so the only thing that separates them is
// what follows the ":". IsLoop is the field that separates them afterwards, and
// it is what a labeled `this.continue` is later resolved against.
func TestLabeledBlockAndLabeledLoopAreDistinguished(t *testing.T) {
	statements := bodyOfFirstUnitFunction(t, `
        'outer: {
            inner := 1;
        }
        'repeat: (inner > 0).loop({
            step();
        });`)

	block, ok := statements[0].(ast.LabeledStmt)
	if !ok {
		t.Fatalf("labeled block = %T, want ast.LabeledStmt", statements[0])
	}
	if block.Label != "'outer" {
		t.Fatalf("block label = %q, want %q", block.Label, "'outer")
	}
	if block.IsLoop {
		t.Fatal("a labeled block was recorded as a loop, which would make it a valid continue target")
	}

	loop, ok := statements[1].(ast.LabeledStmt)
	if !ok {
		t.Fatalf("labeled loop = %T, want ast.LabeledStmt", statements[1])
	}
	if loop.Label != "'repeat" {
		t.Fatalf("loop label = %q, want %q", loop.Label, "'repeat")
	}
	if !loop.IsLoop {
		t.Fatal("a labeled .loop chain was not recorded as a loop")
	}
}

// TestBreakAndContinueCarryTheirOptionalLabel covers break-statement and
// continue-statement in both their labeled and unlabeled forms.
//
// The label is retained WITH its apostrophe, because a label occupies its own
// namespace: stripping the prefix would let `'outer` collide with an ordinary
// identifier `outer` in any consumer that keys on the string.
func TestBreakAndContinueCarryTheirOptionalLabel(t *testing.T) {
	statements := bodyOfFirstUnitFunction(t, `
        'outer: (ready).loop({
            this.break 'outer;
            this.continue 'outer;
            this.break;
            this.continue;
        });`)

	loop := statements[0].(ast.LabeledStmt)
	chain := loop.Body.(ast.ExpressionStmt).Expression
	block := loopBlockOf(t, chain)

	if got := block[0].(ast.BreakStmt).Label; got != "'outer" {
		t.Fatalf("labeled break = %q, want %q", got, "'outer")
	}
	if got := block[1].(ast.ContinueStmt).Label; got != "'outer" {
		t.Fatalf("labeled continue = %q, want %q", got, "'outer")
	}
	if got := block[2].(ast.BreakStmt).Label; got != "" {
		t.Fatalf("unlabeled break = %q, want the empty label", got)
	}
	if got := block[3].(ast.ContinueStmt).Label; got != "" {
		t.Fatalf("unlabeled continue = %q, want the empty label", got)
	}
}

// loopBlockOf digs the block argument out of a `(cond).loop({…})` chain.
func loopBlockOf(t *testing.T, chain ast.Expr) []ast.Stmt {
	t.Helper()
	call, ok := chain.(ast.CallExpr)
	if !ok || len(call.Arguments) != 1 {
		t.Fatalf("loop chain = %T, want a called .loop segment", chain)
	}
	wrapper, ok := call.Arguments[0].(ast.StatementExpr)
	if !ok {
		t.Fatalf("loop argument = %T, want a block", call.Arguments[0])
	}
	return wrapper.Statement.(*ast.BlockStmt).Body
}

// TestLifecycleCallIsNotAnOrdinaryMemberCall covers lifecycle-call-suffix.
//
// The two spellings sit side by side deliberately: `Employee.new(…)` and
// `Employee::new(…)` are different declarations reached through different
// semantic channels, and the reference is explicit that FoLang therefore does
// not reserve `new` or `init` as ordinary method names. Producing one node kind
// for both would merge exactly the channels the language keeps apart.
func TestLifecycleCallIsNotAnOrdinaryMemberCall(t *testing.T) {
	statements := bodyOfFirstUnitFunction(t, `
        a := Employee::new(co.lang.int);
        b := a::init(1);
        c := Employee.new(co.lang.int);`)

	lifecycleNew := inferredValueOf(t, statements[0]).(ast.LifecycleCallExpr)
	if lifecycleNew.Name != "new" || lifecycleNew.Declaration != "@@new" {
		t.Fatalf("lifecycle call = %q -> %q, want new -> @@new", lifecycleNew.Name, lifecycleNew.Declaration)
	}
	if len(lifecycleNew.Arguments) != 1 {
		t.Fatalf("lifecycle arguments = %d, want 1", len(lifecycleNew.Arguments))
	}

	lifecycleInit := inferredValueOf(t, statements[1]).(ast.LifecycleCallExpr)
	if lifecycleInit.Name != "init" || lifecycleInit.Declaration != "@@init" {
		t.Fatalf("lifecycle call = %q -> %q, want init -> @@init", lifecycleInit.Name, lifecycleInit.Declaration)
	}

	// The ordinary member call keeps its ordinary node.
	if _, isLifecycle := inferredValueOf(t, statements[2]).(ast.LifecycleCallExpr); isLifecycle {
		t.Fatal("`Employee.new(…)` was read as a lifecycle invocation; \".\" and \"::\" are separate lookups")
	}
}

// inferredValueOf returns the bound value of an inferred variable declaration.
func inferredValueOf(t *testing.T, statement ast.Stmt) ast.Expr {
	t.Helper()
	declaration, ok := statement.(ast.VarDeclarationStmt)
	if !ok {
		t.Fatalf("statement = %T, want an inferred variable declaration", statement)
	}
	return declaration.AssignedValue
}

// TestLifecycleCustomizationRequiresGenericPermission covers
// class-lifecycle-capability-guard and lifecycle-declaration-context-guard.
//
// Each case is refused for a DIFFERENT reason, and the diagnostic has to say
// which: "this class is not generic" and "this generic class did not opt in"
// have different fixes, and a single permission flag could report neither.
func TestLifecycleCustomizationRequiresGenericPermission(t *testing.T) {
	tests := []struct {
		name     string
		metadata string
		want     string
	}{
		{
			name:     "non-generic class",
			metadata: "",
			want:     "only a generic class may do",
		},
		{
			name:     "generic without the lifecycle field",
			metadata: "@co.dap.generic(types=[{name=T}])\n",
			want:     "must carry lifecycle=true",
		},
		{
			name:     "generic with lifecycle=false",
			metadata: "@co.dap.generic(types=[{name=T}], lifecycle=false)\n",
			want:     "must carry lifecycle=true",
		},
		{
			name:     "lifecycle field without an explicit types list",
			metadata: "@co.dap.generic(lifecycle=true)\n",
			want:     "only a generic class may do",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source := test.metadata + "_ co.lang.class = {\n    @@init() = {}\n}"
			_, p := parsePackageSource(t, source, "Employee.fol")
			if len(p.diags) == 0 {
				t.Fatal("a lifecycle customization was accepted without generic lifecycle permission")
			}
			if got := p.diags[0].Error(); !strings.Contains(got, test.want) {
				t.Fatalf("diagnostic = %q, want text containing %q", got, test.want)
			}
		})
	}
}

// TestLifecycleCustomizationIsAdmittedWithPermission is the positive half of the
// guard: the same declaration is accepted once the class opts in.
func TestLifecycleCustomizationIsAdmittedWithPermission(t *testing.T) {
	source := "@co.dap.generic(types=[{name=T}], lifecycle=true)\n" +
		"_ co.lang.class = {\n    @@init(id T) = { this.id = id; }\n}"
	_, p := parsePackageSource(t, source, "Employee.fol")
	if len(p.diags) != 0 {
		t.Fatalf("a permitted lifecycle customization produced diagnostics: %v", p.diags)
	}
}

// TestAnonymousClassDoesNotInheritLifecyclePermission keeps the capability from
// leaking through a nested anonymous class.
//
// An anonymous `co.lang.class { … }` carries no declaration metadata of its own,
// so it can never be a generic class with lifecycle=true — even when it is
// written inside the body of one that is.
func TestAnonymousClassDoesNotInheritLifecyclePermission(t *testing.T) {
	source := `@co.dap.generic(types=[{name=T}], lifecycle=true)
_ co.lang.class = {
    build()->() = {
        nested := co.lang.class {
            @@init() = {}
        };
    }
}`
	_, p := parsePackageSource(t, source, "Employee.fol")
	if len(p.diags) == 0 {
		t.Fatal("an anonymous class inherited the enclosing class's lifecycle permission")
	}
	if got := p.diags[0].Error(); !strings.Contains(got, "only a generic class may do") {
		t.Fatalf("diagnostic = %q, want the lifecycle capability rule", got)
	}
}
