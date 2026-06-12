package idasrc

import (
	"context"
	"strings"
	"testing"
)

// resolveRealOnFriendResult Harvests the committed real fixtures (same addrs /
// decomp / decompErr as TestHarvestRealBuddyInviteDescentChain) and resolves
// CWvsContext::OnFriendResult.
func resolveRealOnFriendResult(t *testing.T) Fields {
	t.Helper()
	fc := &fakeClient{
		addrs: map[string]string{
			"CWvsContext::OnFriendResult": "0xa3f2e8",
			"sub_A40028":                  "0xa40028",
			"sub_4E4427":                  "0x4e4427",
		},
		decomp: map[string]string{
			"0xa3f2e8": mustFixture(t, "real_onfriendresult_v83.c"),
			"0xa40028": mustFixture(t, "real_sub_a40028_v83.c"),
		},
		decompErr: map[string]error{
			"0x4e4427": NewRPCDecompileError("0x4e4427"),
		},
	}
	ef, err := Harvest(context.Background(), fc,
		[]string{"CWvsContext::OnFriendResult"}, HarvestOpts{DescentDepth: 8})
	if err != nil {
		t.Fatalf("Harvest: %v", err)
	}
	return resolveHarvested(t, ef, "CWvsContext::OnFriendResult")
}

func TestExtractShapeOnFriendResultCase9(t *testing.T) {
	f := resolveRealOnFriendResult(t)
	got := ExtractShape(f, []Selector{{Discriminator: "switch", Case: 9}})
	// pre-branch discriminator Decode1 (mode) + case-9 body; GW_Friend bulk = Unresolved:
	wantOps := []Primitive{Decode1, Decode4, DecodeStr, Unresolved, Decode1}
	if len(got) != len(wantOps) {
		t.Fatalf("extract case9 = %d calls, want %d: %+v", len(got), len(wantOps), got)
	}
	for i, w := range wantOps {
		if got[i].Op != w {
			t.Errorf("extract[%d].Op = %v, want %v (full=%+v)", i, got[i].Op, w, got)
		}
	}
	// No OTHER case's reads leaked in:
	for _, c := range got {
		if strings.Contains(c.Guard, "== 0x14") || strings.Contains(c.Guard, "== 20") ||
			strings.Contains(c.Guard, "== 8") {
			t.Errorf("leaked a different case's read: %+v", c)
		}
	}
}

func TestParseIntLitSuffix(t *testing.T) {
	cases := map[string]int64{"9u": 9, "0xAu": 10, "9ul": 9, "0x12u": 18, "27": 27, "0xA": 10}
	for in, want := range cases {
		if got, ok := parseIntLit(in); !ok || got != want {
			t.Errorf("parseIntLit(%q) = %d,%v want %d", in, got, ok, want)
		}
	}
}

// Also a focused unit test for hex/decimal + composed-guard matching:
func TestExtractShapeHexAndComposedGuards(t *testing.T) {
	f := Fields{Calls: []FieldCall{
		{Op: Decode1, Guard: ""},                              // discriminator (pre-branch)
		{Op: Decode4, Guard: "switch == 8"},                   // case 8
		{Op: Decode2, Guard: "switch == 0xA"},                 // case 10 (hex)
		{Op: DecodeStr, Guard: "(switch == 0xA) && (loop n)"}, // case 10, looped
		{Op: Decode8, Guard: "switch == 9"},                   // case 9
	}}
	got := ExtractShape(f, []Selector{{Discriminator: "switch", Case: 10}})
	// pre-branch Decode1 + the two case-10 reads (hex 0xA == 10); composed guard still matches.
	wantOps := []Primitive{Decode1, Decode2, DecodeStr}
	if len(got) != len(wantOps) {
		t.Fatalf("got %+v", got)
	}
	for i, w := range wantOps {
		if got[i].Op != w {
			t.Errorf("[%d]=%v want %v", i, got[i].Op, w)
		}
	}
}

func TestExtractShape_DefaultArm(t *testing.T) {
	// disc==N reads plus a trailing default-arm read.
	f := Fields{Calls: []FieldCall{
		{Op: Decode1, Guard: ""},            // common header (pre-branch)
		{Op: Decode2, Guard: "mode == 1"},   // case 1
		{Op: Decode4, Guard: "mode == 2"},   // case 2
		{Op: DecodeStr, Guard: "<default>"}, // else/default arm
	}}

	// Default selector: header + the default-arm read only.
	got := ExtractShape(f, []Selector{{Discriminator: "mode", Default: true}})
	wantOps := []Primitive{Decode1, DecodeStr}
	if len(got) != len(wantOps) {
		t.Fatalf("default: got %d reads, want %d (%v)", len(got), len(wantOps), got)
	}
	for i := range wantOps {
		if got[i].Op != wantOps[i] {
			t.Fatalf("default[%d]=%s want %s", i, got[i].Op, wantOps[i])
		}
	}

	// A concrete case selector must NOT pick up the default-arm read.
	got2 := ExtractShape(f, []Selector{{Discriminator: "mode", Case: 1}})
	for _, c := range got2 {
		if c.Guard == "<default>" {
			t.Fatalf("case selector wrongly matched default-arm read: %v", got2)
		}
	}
}

func TestExtractShape_VerbatimGuard(t *testing.T) {
	f := Fields{Calls: []FieldCall{
		{Op: Decode1, Guard: ""},                 // pre-branch discriminator read
		{Op: Decode2, Guard: "v5 < 5"},           // non-equality arm
		{Op: Decode4, Guard: "v5 < 5 && loop n"}, // composed (arm + loop)
		{Op: DecodeStr, Guard: "v5 >= 5"},        // sibling arm
	}}
	got := ExtractShape(f, []Selector{{Guard: "v5 < 5"}})
	wantOps := []Primitive{Decode1, Decode2, Decode4} // header + both reads under "v5 < 5"
	if len(got) != len(wantOps) {
		t.Fatalf("got %d reads, want %d: %v", len(got), len(wantOps), got)
	}
	for i := range wantOps {
		if got[i].Op != wantOps[i] {
			t.Fatalf("[%d]=%s want %s", i, got[i].Op, wantOps[i])
		}
	}
	for _, c := range got {
		if c.Guard == "v5 >= 5" {
			t.Fatalf("verbatim selector matched sibling arm: %v", got)
		}
	}
}
