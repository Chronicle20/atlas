package discover

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseDispatchSwitch(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "process_packet_v83.c.txt"))
	if err != nil {
		t.Fatal(err)
	}
	cases, err := ParseDispatch(string(raw))
	if err != nil {
		t.Fatal(err)
	}
	byOp := map[int]DispatchCase{}
	for _, c := range cases {
		byOp[c.Opcode] = c
	}
	if c := byOp[0x11]; c.Handler != "CLogin::OnFoo" {
		t.Errorf("0x11 -> %+v", c)
	}
	// fallthrough pair: both opcodes map to the same handler
	if byOp[0x12].Handler == "" || byOp[0x12].Handler != byOp[0x13].Handler {
		t.Errorf("fallthrough not shared: %+v / %+v", byOp[0x12], byOp[0x13])
	}
	// decimal label (200 == 0xC8) must be found
	if c := byOp[200]; c.Handler == "" {
		t.Errorf("decimal label 200 not found; got %+v", c)
	}
	// unnamed callee preserved as sub_ address-name, not dropped
	found := false
	for _, c := range cases {
		if c.Handler == "sub_5E1230" {
			found = true
		}
	}
	if !found {
		t.Error("sub_ handler dropped — discovery must keep unnamed handlers")
	}
}

// TestParseDispatchNoiseLines verifies that noise calls inside a case arm are
// skipped and the real handler that follows is still bound (probe finding 3).
func TestParseDispatchNoiseLines(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "process_packet_v83.c.txt"))
	if err != nil {
		t.Fatal(err)
	}
	cases, err := ParseDispatch(string(raw))
	if err != nil {
		t.Fatal(err)
	}
	byOp := map[int]DispatchCase{}
	for _, c := range cases {
		byOp[c.Opcode] = c
	}

	// 0x30: arm has COutPacket ctor, alloca, operator new, TSingleton::GetInstance,
	// vtable void-cast noise; the real handler is CLogin::OnSelectWorld.
	c30 := byOp[0x30]
	if c30.Handler != "CLogin::OnSelectWorld" {
		t.Errorf("0x30 noise-arm: got handler %q, want CLogin::OnSelectWorld", c30.Handler)
	}

	// Ensure none of the noise names leaked as a handler for any opcode.
	noiseNames := []string{
		"COutPacket", "alloca", "operator", "new", "void",
		"TSingleton", "GetInstance",
	}
	for _, c := range cases {
		for _, n := range noiseNames {
			if c.Handler == n {
				t.Errorf("noise name %q leaked as handler for opcode 0x%X", n, c.Opcode)
			}
		}
	}
}

// TestParseDispatchGotoTail verifies that a goto at the end of a case body
// clears pending labels so the next case is not tainted (probe finding 2).
func TestParseDispatchGotoTail(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "process_packet_v83.c.txt"))
	if err != nil {
		t.Fatal(err)
	}
	cases, err := ParseDispatch(string(raw))
	if err != nil {
		t.Fatal(err)
	}
	byOp := map[int]DispatchCase{}
	for _, c := range cases {
		byOp[c.Opcode] = c
	}

	// 0x31: goto-tail case; must bind its OWN handler, not bleed.
	c31 := byOp[0x31]
	if c31.Handler != "CLogin::OnExtraWork" {
		t.Errorf("0x31 goto-tail: got handler %q, want CLogin::OnExtraWork", c31.Handler)
	}

	// 0x32: the case AFTER the goto; must have its own handler, not the goto case's.
	c32 := byOp[0x32]
	if c32.Handler != "CLogin::OnCharList" {
		t.Errorf("0x32 after-goto: got handler %q, want CLogin::OnCharList", c32.Handler)
	}
	if c32.Handler == c31.Handler {
		t.Errorf("0x32 bled handler from goto case 0x31: %q", c32.Handler)
	}
}

// TestParseDispatchNestedSwitch verifies that case labels inside a nested
// switch body are NOT emitted as top-level dispatch opcodes (probe finding 1).
func TestParseDispatchNestedSwitch(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "process_packet_v83.c.txt"))
	if err != nil {
		t.Fatal(err)
	}
	cases, err := ParseDispatch(string(raw))
	if err != nil {
		t.Fatal(err)
	}
	byOp := map[int]DispatchCase{}
	for _, c := range cases {
		byOp[c.Opcode] = c
	}

	// 0x01 and 0x02 are inner case labels inside 0x40's nested switch;
	// they must NOT appear as top-level dispatch entries.
	if _, ok := byOp[0x01]; ok {
		t.Error("inner nested-switch label 0x01 was emitted as top-level opcode (ghost entry)")
	}
	if _, ok := byOp[0x02]; ok {
		t.Error("inner nested-switch label 0x02 was emitted as top-level opcode (ghost entry)")
	}

	// 0x40 itself must bind the handler that follows the nested switch body.
	c40 := byOp[0x40]
	if c40.Handler != "CLogin::OnMigrateIn" {
		t.Errorf("0x40 nested-switch parent: got handler %q, want CLogin::OnMigrateIn", c40.Handler)
	}
}

// TestParseDispatchVtableNoVoid ensures a vtable cast line containing 'void'
// does not bind 'void' as a handler (probe finding 3 — vtable variant).
func TestParseDispatchVtableNoVoid(t *testing.T) {
	const vtableSnippet = `
void __thiscall Dispatcher(void *this, CInPacket *a2)
{
  switch ( CInPacket::Decode2(a2) )
  {
    case 0x50u:
      (*(void(**)(void *, CInPacket *))(*((_DWORD *)this) + 8))(this, a2);
      CGame::OnSomePacket(this, a2);
      break;
  }
}
`
	cases, err := ParseDispatch(vtableSnippet)
	if err != nil {
		t.Fatal(err)
	}
	byOp := map[int]DispatchCase{}
	for _, c := range cases {
		byOp[c.Opcode] = c
	}
	c50 := byOp[0x50]
	if c50.Handler == "void" {
		t.Error("vtable line bound 'void' as handler — noise filter missed it")
	}
	if c50.Handler != "CGame::OnSomePacket" {
		t.Errorf("0x50 vtable arm: got handler %q, want CGame::OnSomePacket", c50.Handler)
	}
}
