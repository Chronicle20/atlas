package idasrc

import (
	"context"
	"testing"
)

func TestResolveLiveOnFriendResultByAddress(t *testing.T) {
	fc := &fakeClient{
		decomp: map[string]string{
			"0xa3f2e8": mustFixture(t, "real_onfriendresult_v83.c"),
			"0xa40028": mustFixture(t, "real_sub_a40028_v83.c"),
		},
		decompErr: map[string]error{"0x4e4427": NewRPCDecompileError("0x4e4427")},
		// Address-based descent: the base's callees include the sub_A40028 helper
		// at 0xa40028; the sub's callees include sub_4E4427 at 0x4e4427 (whose
		// decompile soft-fails → Unresolved).
		callees: map[string][]Callee{
			"0xa3f2e8": {{Name: "sub_A40028", Addr: "0xa40028"}},
			"0xa40028": {{Name: "sub_4E4427", Addr: "0x4e4427"}},
		},
	}
	f, err := ResolveLive(context.Background(), fc, "0xa3f2e8", DirClientbound, HarvestOpts{DescentDepth: 8})
	if err != nil {
		t.Fatalf("ResolveLive: %v", err)
	}
	inv := ExtractShape(f, []Selector{{Discriminator: "switch", Case: 9}})
	want := []Primitive{Decode1, Decode4, DecodeStr, Unresolved, Decode1}
	if len(inv) != len(want) {
		t.Fatalf("case9 = %d, want %d: %+v", len(inv), len(want), inv)
	}
	for i, w := range want {
		if inv[i].Op != w {
			t.Errorf("[%d]=%v want %v", i, inv[i].Op, w)
		}
	}
}

// TestResolveLiveDescendsDemangledHelperViaCallees proves the core fix: a base
// that delegates to a DEMANGLED helper name (CWvsContext::CFriend::Insert) which
// GetFunctionByName cannot resolve is now descended BY ADDRESS via the parent's
// callees (mangled name + addr) + demangle — yielding the helper's real reads
// instead of an Unresolved gap.
func TestResolveLiveDescendsDemangledHelperViaCallees(t *testing.T) {
	base := `/* line: 0, address: 0xA0 */ void __thiscall CWvsContext::OnFoo(CWvsContext *this, CInPacket *a2)
{
  CInPacket::Decode1(a2);
  CWvsContext::CFriend::Insert(this, a2);
}`
	helper := `/* line: 0, address: 0xB0 */ void __thiscall CWvsContext::CFriend::Insert(CWvsContext *this, CInPacket *a2)
{
  CInPacket::Decode4(a2);
  CInPacket::Decode1(a2);
}`
	fc := &fakeClient{
		// GetFunctionByName MUST NOT resolve the demangled name — this is the gap.
		addrs: map[string]string{},
		decomp: map[string]string{
			"0xA0": base,
			"0xB0": helper,
		},
		callees: map[string][]Callee{
			"0xA0": {{Name: "?Insert@CFriend@CWvsContext@@QAEXAAVCInPacket@@@Z", Addr: "0xB0"}},
		},
	}
	f, err := ResolveLive(context.Background(), fc, "0xA0", DirClientbound, HarvestOpts{})
	if err != nil {
		t.Fatalf("ResolveLive: %v", err)
	}
	// base Decode1 (discriminator), then the helper's two reads spliced in via
	// callees+demangle (Decode4, Decode1) — NOT an Unresolved gap.
	want := []Primitive{Decode1, Decode4, Decode1}
	if len(f.Calls) != len(want) {
		t.Fatalf("got %d calls, want %d: %+v", len(f.Calls), len(want), f.Calls)
	}
	for _, call := range f.Calls {
		if call.Op == Unresolved {
			t.Fatalf("unexpected Unresolved in resolved base: %+v", f.Calls)
		}
	}
	for i, w := range want {
		if f.Calls[i].Op != w {
			t.Errorf("[%d]=%v want %v", i, f.Calls[i].Op, w)
		}
	}
}

func TestResolveLiveBaseDecompileSoftFail(t *testing.T) {
	fc := &fakeClient{decompErr: map[string]error{"0xDEAD": NewRPCDecompileError("0xDEAD")}}
	_, err := ResolveLive(context.Background(), fc, "0xDEAD", DirClientbound, HarvestOpts{})
	if !IsDecompilationFailed(err) {
		t.Fatalf("want IsDecompilationFailed, got %v", err)
	}
}

// TestResolveLivePopulatesCaseLabels proves the base function's full dispatch
// case-label set is collected onto the returned Fields (for the bijection check).
func TestResolveLiveCaseLabels(t *testing.T) {
	const dec = "void __thiscall Foo::OnBar(Foo *this, CInPacket *a2)\n{\n" +
		"  unsigned __int8 mode = CInPacket::Decode1(a2);\n" +
		"  switch ( mode )\n  {\n" +
		"    case 1:\n      CInPacket::Decode4(a2);\n      break;\n" +
		"    case 2:\n      break;\n  }\n}\n"
	fc := &fakeClient{decomp: map[string]string{"0x10": dec}}
	f, err := ResolveLive(context.Background(), fc, "0x10", DirClientbound, HarvestOpts{})
	if err != nil {
		t.Fatal(err)
	}
	cs := f.CaseLabels["mode"]
	if cs == nil || !cs.Has(1) || !cs.Has(2) {
		t.Fatalf("CaseLabels[mode] missing 1/2: %+v", f.CaseLabels)
	}
}
