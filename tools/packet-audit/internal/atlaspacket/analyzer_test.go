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
