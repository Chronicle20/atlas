package idasrc

import (
	"context"
	"strings"
	"testing"
)

func TestHarvestDescendsHelper(t *testing.T) {
	fc := &fakeClient{
		addrs: map[string]string{
			"CWvsContext::OnFriendResult": "0xA0",
			"CFriend::Insert":             "0xB0",
		},
		decomp: map[string]string{
			"0xA0": mustFixture(t, "struct_helper.c"), // emits Delegate->CFriend::Insert
			"0xB0": "int CFriend::Insert(GW_Friend *r, CInPacket *a2)\n" +
				"{\n" +
				"  CInPacket::Decode4(a2);\n" +
				"  CInPacket::Decode2(a2);\n" +
				"}\n",
		},
	}
	ef, err := Harvest(context.Background(), fc,
		[]string{"CWvsContext::OnFriendResult"}, HarvestOpts{DescentDepth: 4})
	if err != nil {
		t.Fatalf("Harvest: %v", err)
	}
	if _, ok := ef.Functions["CWvsContext::OnFriendResult"]; !ok {
		t.Fatal("parent missing from export")
	}
	helper, ok := ef.Functions["CFriend::Insert"]
	if !ok {
		t.Fatal("descended helper CFriend::Insert missing from export")
	}
	if len(helper.Calls) != 2 {
		t.Errorf("helper calls = %d, want 2", len(helper.Calls))
	}
	parent := ef.Functions["CWvsContext::OnFriendResult"]
	foundDelegate := false
	for _, c := range parent.Calls {
		if c.Op == "Delegate" && c.Ref == "CFriend::Insert" {
			foundDelegate = true
		}
	}
	if !foundDelegate {
		t.Error("parent missing Delegate->CFriend::Insert")
	}
}

func TestHarvestDepthBoundIsResolvable(t *testing.T) {
	// P -> H -> G, with DescentDepth=1: H is at the bound (depth 1, allowed),
	// but H's Delegate->G would be depth 2 > bound. The over-bound ref must be
	// neutralized to Unresolved so the whole export resolves WITHOUT a hard error.
	fc := &fakeClient{
		addrs: map[string]string{"P::f": "0x1", "H::f": "0x2", "G::f": "0x3"},
		decomp: map[string]string{
			"0x1": "void P::f(X *x, CInPacket *a2)\n{\n  CInPacket::Decode4(a2);\n  H::f(x, a2);\n}\n",
			"0x2": "void H::f(X *x, CInPacket *a2)\n{\n  CInPacket::Decode2(a2);\n  G::f(x, a2);\n}\n",
			"0x3": "void G::f(X *x, CInPacket *a2)\n{\n  CInPacket::Decode1(a2);\n}\n",
		},
	}
	ef, err := Harvest(context.Background(), fc, []string{"P::f"}, HarvestOpts{DescentDepth: 1})
	if err != nil {
		t.Fatalf("Harvest: %v", err)
	}
	// H must be exported, its Delegate->G replaced by an Unresolved op, and G absent.
	h, ok := ef.Functions["H::f"]
	if !ok {
		t.Fatal("H::f missing from export")
	}
	if !h.Unresolved {
		t.Errorf("H::f should be flagged Unresolved (descent depth exceeded)")
	}
	for _, c := range h.Calls {
		if c.Op == "Delegate" {
			t.Errorf("H::f retains a dangling Delegate ref %q — must be neutralized to Unresolved", c.Ref)
		}
	}
	if _, present := ef.Functions["G::f"]; present {
		t.Errorf("G::f should NOT be harvested (beyond depth bound)")
	}
	// The export must resolve cleanly (no 'function not in export' hard error).
	src := newExportSourceFromFile(ef)
	if _, err := src.Resolve(context.Background(), "P::f"); err != nil {
		t.Fatalf("Resolve(P::f) must not hard-error on a depth-bounded export: %v", err)
	}
}

func resolveHarvested(t *testing.T, ef exportFile, fname string) Fields {
	t.Helper()
	src := newExportSourceFromFile(ef)
	f, err := src.Resolve(context.Background(), fname)
	if err != nil {
		t.Fatalf("resolve %s: %v", fname, err)
	}
	return f
}

func TestBuddyInviteFourVersion(t *testing.T) {
	cases := []struct {
		name     string
		fixture  string
		wantHead []Primitive // before GW_Friend
	}{
		{"v83", "friend_v83.c", []Primitive{Decode4, DecodeStr}},
		{"v87", "friend_v87.c", []Primitive{Decode4, DecodeStr, Decode4, Decode4}},
		{"jms", "friend_jms.c", []Primitive{Decode4, DecodeStr, Decode4, Decode4}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fc := &fakeClient{
				addrs: map[string]string{"CWvsContext::OnFriendResult": "0xA", "CFriend::Insert": "0xB"},
				decomp: map[string]string{
					"0xA": mustFixture(t, tc.fixture),
					"0xB": mustFixture(t, "friend_insert.c"),
				},
			}
			ef, err := Harvest(context.Background(), fc,
				[]string{"CWvsContext::OnFriendResult"}, HarvestOpts{DescentDepth: 4})
			if err != nil {
				t.Fatalf("Harvest: %v", err)
			}
			f := resolveHarvested(t, ef, "CWvsContext::OnFriendResult")
			// Filter to the case-9 (#Invite) reads via guard "mode == 9".
			var inv []FieldCall
			for _, c := range f.Calls {
				if strings.Contains(c.Guard, "mode == 9") {
					inv = append(inv, c)
				}
			}
			// head + 5 GW_Friend prims (4,13,1,4,17 -> 5 calls) + inShop(1)
			wantLen := len(tc.wantHead) + 5 + 1
			if len(inv) != wantLen {
				t.Fatalf("%s: got %d invite calls, want %d: %+v", tc.name, len(inv), wantLen, inv)
			}
			for i, w := range tc.wantHead {
				if inv[i].Op != w {
					t.Errorf("%s head[%d] = %v, want %v", tc.name, i, inv[i].Op, w)
				}
			}
			// MUST NOT be a count-loop and MUST NOT truncate before inShop.
			last := inv[len(inv)-1]
			if last.Op != Decode1 {
				t.Errorf("%s: last invite call = %v, want Decode1 inShop (no truncation)", tc.name, last.Op)
			}
			for _, c := range inv {
				if strings.Contains(c.Guard, "loop ") {
					t.Errorf("%s: GW_Friend mistraced as a loop (%+v)", tc.name, c)
				}
			}
		})
	}
}

func TestHarvestRealBuddyInviteDescentChain(t *testing.T) {
	fc := &fakeClient{
		addrs: map[string]string{
			"CWvsContext::OnFriendResult": "0xa3f2e8",
			"sub_A40028":                  "0xa40028",
			"sub_4E4427":                  "0x4e4427",
		},
		decomp: map[string]string{
			"0xa3f2e8": mustFixture(t, "real_onfriendresult_v83.c"),
			"0xa40028": mustFixture(t, "real_sub_a40028_v83.c"),
			// 0x4e4427 has no decomp — it soft-fails below
		},
		decompErr: map[string]error{
			"0x4e4427": NewRPCDecompileError("0x4e4427"),
		},
	}
	ef, err := Harvest(context.Background(), fc,
		[]string{"CWvsContext::OnFriendResult"}, HarvestOpts{DescentDepth: 8})
	if err != nil {
		t.Fatalf("Harvest must not error on a decompile soft-fail in the chain: %v", err)
	}
	// sub_A40028 descended and exported; sub_4E4427 exported as Unresolved (not found-error, not abort).
	if _, ok := ef.Functions["sub_A40028"]; !ok {
		t.Fatal("sub_A40028 not harvested (struct helper must be descended)")
	}
	h4, ok := ef.Functions["sub_4E4427"]
	if !ok || !h4.Unresolved {
		t.Fatalf("sub_4E4427 must be an Unresolved entry (undecompilable), got %+v ok=%v", h4, ok)
	}
	f := resolveHarvested(t, ef, "CWvsContext::OnFriendResult")
	// Filter to the case-9 (#Invite) reads via the case guard.
	var inv []FieldCall
	for _, c := range f.Calls {
		if strings.Contains(c.Guard, "switch == 9") {
			inv = append(inv, c)
		}
	}
	// Faithful chain: friendId, name, <GW_Friend descent: Unresolved>, flag.
	wantOps := []Primitive{Decode4, DecodeStr, Unresolved, Decode1}
	if len(inv) != len(wantOps) {
		t.Fatalf("invite chain = %d calls, want %d: %+v", len(inv), len(wantOps), inv)
	}
	for i, w := range wantOps {
		if inv[i].Op != w {
			t.Errorf("invite[%d].Op = %v, want %v (full=%+v)", i, inv[i].Op, w, inv)
		}
	}
	// Must NOT be a count-loop, must NOT silently drop the struct helper.
	for _, c := range inv {
		if strings.HasPrefix(c.Guard, "loop ") || strings.Contains(c.Guard, "&& loop ") {
			t.Errorf("invite call mistraced as loop: %+v", c)
		}
	}
}

func TestHarvestCycleGuard(t *testing.T) {
	fc := &fakeClient{
		addrs: map[string]string{"A::f": "0x1", "B::f": "0x2"},
		decomp: map[string]string{
			"0x1": "void A::f(X *x, CInPacket *a2)\n{\n  CInPacket::Decode1(a2);\n  B::f(x, a2);\n}\n",
			"0x2": "void B::f(X *x, CInPacket *a2)\n{\n  CInPacket::Decode1(a2);\n  A::f(x, a2);\n}\n",
		},
	}
	ef, err := Harvest(context.Background(), fc, []string{"A::f"}, HarvestOpts{DescentDepth: 8})
	if err != nil {
		t.Fatalf("Harvest must not loop forever / error on cycle: %v", err)
	}
	if _, ok := ef.Functions["A::f"]; !ok {
		t.Error("A::f missing")
	}
	if _, ok := ef.Functions["B::f"]; !ok {
		t.Error("B::f missing")
	}
}

// TestHarvestDecompileSoftFail verifies that a per-function DecompileFunction
// soft error (IsDecompilationFailed == true) does NOT abort the whole export.
// The failing function must become an Unresolved entry; all other functions
// must be exported normally.
func TestHarvestDecompileSoftFail(t *testing.T) {
	// sub_4E4427 is a real v83 function that soft-fails decompilation.
	softErr := NewRPCDecompileError("0x4e4427")
	if !IsDecompilationFailed(softErr) {
		t.Fatal("test setup: softErr is not recognized by IsDecompilationFailed — fix the test")
	}

	fc := &fakeClient{
		addrs: map[string]string{
			"sub_4E4427":     "0x4e4427",
			"CWvsContext::f": "0xA0",
		},
		decomp: map[string]string{
			"0xA0": "void CWvsContext::f(CInPacket *a2)\n{\n  CInPacket::Decode4(a2);\n}\n",
		},
		decompErr: map[string]error{
			"0x4e4427": softErr,
		},
	}

	ef, err := Harvest(context.Background(), fc,
		[]string{"sub_4E4427", "CWvsContext::f"}, HarvestOpts{DescentDepth: 4})

	// The export must NOT return an error — soft decompile failure is not fatal.
	if err != nil {
		t.Fatalf("Harvest must not abort on decompile soft-fail; got error: %v", err)
	}

	// The failing function must be an Unresolved entry with the expected sentinel call.
	bad, ok := ef.Functions["sub_4E4427"]
	if !ok {
		t.Fatal("sub_4E4427 missing from export — must be present as Unresolved")
	}
	if !bad.Unresolved {
		t.Error("sub_4E4427 must be marked Unresolved")
	}
	if len(bad.Calls) != 1 {
		t.Fatalf("sub_4E4427 Calls len = %d, want 1", len(bad.Calls))
	}
	if bad.Calls[0].Op != "Unresolved" {
		t.Errorf("sub_4E4427 Calls[0].Op = %q, want \"Unresolved\"", bad.Calls[0].Op)
	}
	if bad.Calls[0].Comment != "decompilation failed; hand-trace" {
		t.Errorf("sub_4E4427 Calls[0].Comment = %q, want \"decompilation failed; hand-trace\"",
			bad.Calls[0].Comment)
	}

	// The other function must be exported normally (decompiled and parsed).
	good, ok := ef.Functions["CWvsContext::f"]
	if !ok {
		t.Fatal("CWvsContext::f missing from export — must be exported normally")
	}
	if good.Unresolved {
		t.Error("CWvsContext::f must NOT be flagged Unresolved")
	}
	if len(good.Calls) != 1 {
		t.Errorf("CWvsContext::f Calls len = %d, want 1 (Decode4)", len(good.Calls))
	}
}
