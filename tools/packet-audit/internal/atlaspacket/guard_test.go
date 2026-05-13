package atlaspacket

import "testing"

func TestGuardParseRegion(t *testing.T) {
	g, err := ParseGuard(`t.Region() == "GMS"`)
	if err != nil {
		t.Fatal(err)
	}
	ctx := GuardContext{Region: "GMS", MajorVersion: 95}
	if !g.Eval(ctx) {
		t.Errorf("expected eval=true for GMS context")
	}
	ctx.Region = "JMS"
	if g.Eval(ctx) {
		t.Errorf("expected eval=false for JMS context")
	}
}

func TestGuardParseMajorGE(t *testing.T) {
	g, err := ParseGuard(`t.MajorVersion() >= 95`)
	if err != nil {
		t.Fatal(err)
	}
	if !g.Eval(GuardContext{MajorVersion: 95}) {
		t.Error("v95 should satisfy >=95")
	}
	if g.Eval(GuardContext{MajorVersion: 83}) {
		t.Error("v83 should not satisfy >=95")
	}
}

func TestGuardParseAnd(t *testing.T) {
	g, err := ParseGuard(`t.Region() == "GMS" && t.MajorVersion() > 12`)
	if err != nil {
		t.Fatal(err)
	}
	if !g.Eval(GuardContext{Region: "GMS", MajorVersion: 95}) {
		t.Error("GMS v95 should satisfy")
	}
	if g.Eval(GuardContext{Region: "GMS", MajorVersion: 12}) {
		t.Error("GMS v12 should not satisfy >12")
	}
}

func TestNestedIfFromAnalyzer(t *testing.T) {
	calls, err := AnalyzeFile("testdata/nested_encode.go.txt", "Nested", "Encode")
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 3 {
		t.Fatalf("calls=%d", len(calls))
	}
	if calls[0].Guard != nil {
		t.Errorf("calls[0] should be unguarded")
	}
	if calls[2].Guard == nil {
		t.Errorf("calls[2] should be guarded")
	}
	if !calls[2].Guard.Eval(GuardContext{Region: "GMS", MajorVersion: 95}) {
		t.Errorf("calls[2] should eval true for GMS v95")
	}
	if calls[2].Guard.Eval(GuardContext{Region: "GMS", MajorVersion: 83}) {
		t.Errorf("calls[2] should eval false for GMS v83")
	}
}
