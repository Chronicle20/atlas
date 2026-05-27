package atlaspacket

import "testing"

func TestSimpleEncodeExtractsThreeCalls(t *testing.T) {
	calls, err := AnalyzeFile("testdata/simple_encode.go.txt", "Simple", "Encode")
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 3 {
		t.Fatalf("calls: got %d, want 3 (%+v)", len(calls), calls)
	}
	if calls[0].Op != Encode1 || calls[1].Op != Encode4 || calls[2].Op != EncodeStr {
		t.Errorf("ops: got %v %v %v", calls[0].Op, calls[1].Op, calls[2].Op)
	}
}

func TestEarlyReturnThenTaintsSuffix(t *testing.T) {
	calls, err := AnalyzeFile("testdata/early_return_then.go.txt", "EarlyReturnThen", "Encode")
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 2 {
		t.Fatalf("calls: got %d, want 2 (%+v)", len(calls), calls)
	}
	// First call: WriteByte under guard t.MajorVersion() >= 95.
	if calls[0].Op != Encode1 || calls[0].Guard == nil || calls[0].Guard.String() != "t.MajorVersion() >= 95" {
		t.Errorf("call[0]: op=%v guard=%v; want Encode1 guard=t.MajorVersion() >= 95", calls[0].Op, guardText(calls[0].Guard))
	}
	// Second call: WriteInt under guard !(t.MajorVersion() >= 95).
	if calls[1].Op != Encode4 || calls[1].Guard == nil || calls[1].Guard.String() != "!(t.MajorVersion() >= 95)" {
		t.Errorf("call[1]: op=%v guard=%v; want Encode4 guard=!(t.MajorVersion() >= 95)", calls[1].Op, guardText(calls[1].Guard))
	}
}

func TestEarlyReturnElseTaintsSuffix(t *testing.T) {
	calls, err := AnalyzeFile("testdata/early_return_else.go.txt", "EarlyReturnElse", "Encode")
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 3 {
		t.Fatalf("calls: got %d, want 3 (%+v)", len(calls), calls)
	}
	// calls[0]: WriteByte under guard t.MajorVersion() >= 95.
	if calls[0].Op != Encode1 || calls[0].Guard == nil || calls[0].Guard.String() != "t.MajorVersion() >= 95" {
		t.Errorf("call[0]: op=%v guard=%v; want Encode1 guard=t.MajorVersion() >= 95", calls[0].Op, guardText(calls[0].Guard))
	}
	// calls[1]: WriteShort under guard !(t.MajorVersion() >= 95).
	if calls[1].Op != Encode2 || calls[1].Guard == nil || calls[1].Guard.String() != "!(t.MajorVersion() >= 95)" {
		t.Errorf("call[1]: op=%v guard=%v; want Encode2 guard=!(t.MajorVersion() >= 95)", calls[1].Op, guardText(calls[1].Guard))
	}
	// calls[2]: WriteInt under guard t.MajorVersion() >= 95 (because the else-branch returned).
	if calls[2].Op != Encode4 || calls[2].Guard == nil || calls[2].Guard.String() != "t.MajorVersion() >= 95" {
		t.Errorf("call[2]: op=%v guard=%v; want Encode4 guard=t.MajorVersion() >= 95", calls[2].Op, guardText(calls[2].Guard))
	}
}

func TestEarlyReturnNegativeLeavesSuffixUnconditional(t *testing.T) {
	calls, err := AnalyzeFile("testdata/early_return_negative.go.txt", "EarlyReturnNegative", "Encode")
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 2 {
		t.Fatalf("calls: got %d, want 2 (%+v)", len(calls), calls)
	}
	if calls[1].Op != Encode4 || calls[1].Guard != nil {
		t.Errorf("call[1]: op=%v guard=%v; want Encode4 guard=nil", calls[1].Op, guardText(calls[1].Guard))
	}
}

// guardText is a test helper: returns "" for nil guards so format-string callers
// don't have to nil-check inline.
func guardText(g *GuardExpr) string {
	return g.String()
}

// TestWireMutexCollapsesIfElse verifies task-065 item 5: when an if/else
// writes the same wire shape in both branches, the analyzer collapses it into
// a SINGLE position rather than emitting two consecutive calls that misalign
// the diff. The DropSpawn / MonsterSpawn analyzer FPs documented in
// docs/tasks/task-065-combat-domain-audit/post-phase-b.md trace to this
// pattern.
func TestWireMutexCollapsesIfElse(t *testing.T) {
	calls, err := AnalyzeFile("testdata/wire_mutex.go.txt", "WireMutex", "Encode")
	if err != nil {
		t.Fatal(err)
	}
	// Expected: Encode1, Encode4 (mutex-collapsed from the if/else),
	// Encode1. NOT four calls.
	if len(calls) != 3 {
		t.Fatalf("calls: got %d, want 3 (Encode1 + collapsed Encode4 + Encode1) — %+v", len(calls), calls)
	}
	if calls[0].Op != Encode1 {
		t.Errorf("calls[0]: got %v, want Encode1", calls[0].Op)
	}
	if calls[1].Op != Encode4 {
		t.Errorf("calls[1]: got %v, want Encode4 (collapsed)", calls[1].Op)
	}
	if calls[2].Op != Encode1 {
		t.Errorf("calls[2]: got %v, want Encode1", calls[2].Op)
	}
	// The collapsed Encode4 should have NO guard — wire shape is invariant
	// across the if/else, so the position is unconditional under any outer
	// scope.
	if calls[1].Guard != nil {
		t.Errorf("calls[1].Guard = %q; want nil (collapsed mutex is unconditional)", calls[1].Guard.String())
	}
}

// TestWireDivergentKeepsBothBranches is the negative case: when the if/else
// branches write DIVERGENT wire shapes (different widths, different lengths,
// or different recurse types), the analyzer must keep both branches with
// their respective guards. Verifies the mutex detection doesn't over-trigger.
func TestWireDivergentKeepsBothBranches(t *testing.T) {
	calls, err := AnalyzeFile("testdata/wire_divergent.go.txt", "WireDivergent", "Encode")
	if err != nil {
		t.Fatal(err)
	}
	// Expected: Encode4 (guarded extended()), Encode1 (guarded extended()),
	// Encode4 (guarded !extended()). Three calls, not collapsed.
	if len(calls) != 3 {
		t.Fatalf("calls: got %d, want 3 (divergent branches stay expanded) — %+v", len(calls), calls)
	}
	// First two come from the then-branch (Encode4 + Encode1).
	if calls[0].Op != Encode4 || calls[1].Op != Encode1 {
		t.Errorf("then-branch ops: got %v %v, want Encode4 Encode1", calls[0].Op, calls[1].Op)
	}
	// Last one is from the else-branch (Encode4).
	if calls[2].Op != Encode4 {
		t.Errorf("else-branch op: got %v, want Encode4", calls[2].Op)
	}
	// All three carry guards (extended() or !extended()) — divergent shapes
	// keep their conditionality. Guard text comes through as <unparsed:…>
	// because the DSL only models t.Region/Major/MinorVersion comparisons.
	for i, c := range calls {
		if c.Guard == nil {
			t.Errorf("calls[%d].Guard = nil; divergent-branch calls must keep guards", i)
		}
	}
}
