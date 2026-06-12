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

// TestParseDispatchBracedCaseBody verifies that a case body wrapped in braces
// (Hex-Rays sometimes emits these) binds the handler to the correct opcode and
// does not leak it to the following case.
func TestParseDispatchBracedCaseBody(t *testing.T) {
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

	// 0x50 has a braced body: the handler must bind to 0x50, not leak to 0x51.
	c50 := byOp[0x50]
	if c50.Handler != "CLogin::OnBracedBody" {
		t.Errorf("0x50 braced-body: got handler %q, want CLogin::OnBracedBody", c50.Handler)
	}

	// 0x51 must bind its own handler independently.
	c51 := byOp[0x51]
	if c51.Handler != "CLogin::OnAfterBraced" {
		t.Errorf("0x51 after-braced: got handler %q, want CLogin::OnAfterBraced", c51.Handler)
	}

	// They must not share a handler (no leak from braced case).
	if c50.Handler == c51.Handler {
		t.Errorf("0x50 leaked handler to 0x51: both have %q", c50.Handler)
	}
}

// TestParseDispatchBracedIfInCase verifies that a handler inside a braced if
// body within a case arm is bound to the case's pending opcode.
func TestParseDispatchBracedIfInCase(t *testing.T) {
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

	// 0x52: handler is inside an if (...) { } block; must still bind to 0x52.
	c52 := byOp[0x52]
	if c52.Handler != "CLogin::OnBracedIf" {
		t.Errorf("0x52 braced-if: got handler %q, want CLogin::OnBracedIf", c52.Handler)
	}
}

// TestParseDispatchNestedSwitchOuterHandlerAfterInner verifies that the outer
// case's handler (which appears AFTER the nested switch body closes) is still
// bound to the outer opcode.
func TestParseDispatchNestedSwitchOuterHandlerAfterInner(t *testing.T) {
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

	// 0x40's nested switch contains 0x01 and 0x02; after the inner switch closes,
	// CLogin::OnMigrateIn is called at the outer case level.
	c40 := byOp[0x40]
	if c40.Handler != "CLogin::OnMigrateIn" {
		t.Errorf("0x40 outer handler after nested switch: got %q, want CLogin::OnMigrateIn", c40.Handler)
	}

	// Inner labels must still be suppressed.
	if _, ok := byOp[0x01]; ok {
		t.Error("inner nested-switch label 0x01 leaked as top-level opcode after outer-handler fix")
	}
	if _, ok := byOp[0x02]; ok {
		t.Error("inner nested-switch label 0x02 leaked as top-level opcode after outer-handler fix")
	}
}

// TestParseDispatchZeroCases verifies that ParseDispatch returns an empty (not
// nil) slice — not an error — when given text with no switch statement. Callers
// that receive zero cases should treat this as suspicious (wrong function or
// if-based dispatch; see ParseDispatch contract).
func TestParseDispatchZeroCases(t *testing.T) {
	const noSwitch = `
void __thiscall Dispatcher(void *this, CInPacket *a2)
{
  if (op == 1) { CFoo::OnOne(this, a2); }
  if (op == 2) { CFoo::OnTwo(this, a2); }
}
`
	cases, err := ParseDispatch(noSwitch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cases == nil {
		t.Error("ParseDispatch returned nil; want empty slice for zero-case result")
	}
	if len(cases) != 0 {
		t.Errorf("expected 0 cases from if-based dispatch, got %d", len(cases))
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

// TestParseDispatchSameLineSwitchBrace covers the non-Allman shape
// `switch ( v ) {` — the open brace on the same line as the nested switch.
// Hex-Rays never emits this, but the suppression must not silently bind an
// inner arm's handler to the outer case label if input is hand-edited.
func TestParseDispatchSameLineSwitchBrace(t *testing.T) {
	src := `
  switch ( op )
  {
    case 0xA0u:
      switch ( v ) {
        case 1u:
          CLogin::OnInnerA1(v3);
          break;
      }
      CLogin::OnOuterA0(v3, a2);
      break;
    case 0xA1u:
      CLogin::OnNextA1(v3);
      break;
  }
`
	cases, err := ParseDispatch(src)
	if err != nil {
		t.Fatal(err)
	}
	byOp := map[int]DispatchCase{}
	for _, c := range cases {
		byOp[c.Opcode] = c
	}
	if _, ok := byOp[0x01]; ok {
		t.Error("inner nested-switch label leaked as top-level opcode")
	}
	if got := byOp[0xA0].Handler; got != "CLogin::OnOuterA0" {
		t.Errorf("0xA0 handler = %q, want CLogin::OnOuterA0", got)
	}
	if got := byOp[0xA1].Handler; got != "CLogin::OnNextA1" {
		t.Errorf("0xA1 handler = %q, want CLogin::OnNextA1", got)
	}
}
