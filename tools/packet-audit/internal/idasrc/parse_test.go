package idasrc

import (
	"os"
	"strings"
	"testing"
)

func mustFixture(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return string(b)
}

func TestParseLinearReads(t *testing.T) {
	calls, err := ParseDecompile(mustFixture(t, "linear.c"), DirClientbound)
	if err != nil {
		t.Fatalf("ParseDecompile: %v", err)
	}
	want := []struct {
		op      string
		comment string
	}{
		{"Decode1", "resultCode"},
		{"Decode4", "accountId"},
		{"DecodeStr", "name"},
	}
	if len(calls) != len(want) {
		t.Fatalf("got %d calls, want %d: %+v", len(calls), len(want), calls)
	}
	for i, w := range want {
		if calls[i].Op != w.op {
			t.Errorf("call[%d].Op = %q, want %q", i, calls[i].Op, w.op)
		}
		if calls[i].Comment != w.comment {
			t.Errorf("call[%d].Comment = %q, want %q", i, calls[i].Comment, w.comment)
		}
	}
}

func TestParseStructHelperDelegate(t *testing.T) {
	calls, err := ParseDecompile(mustFixture(t, "struct_helper.c"), DirClientbound)
	if err != nil {
		t.Fatalf("ParseDecompile: %v", err)
	}
	// Expect: Decode4 friendId, DecodeStr name, Delegate->CFriend::Insert, Decode1 inShop
	if len(calls) != 4 {
		t.Fatalf("got %d calls, want 4: %+v", len(calls), calls)
	}
	if calls[2].Op != "Delegate" || calls[2].Ref != "CFriend::Insert" {
		t.Errorf("call[2] = %+v, want Delegate ref=CFriend::Insert", calls[2])
	}
	if calls[3].Op != "Decode1" {
		t.Errorf("call[3].Op = %q, want Decode1 (trailing inShop not truncated)", calls[3].Op)
	}
}

func TestParseCountLoop(t *testing.T) {
	calls, err := ParseDecompile(mustFixture(t, "count_loop.c"), DirClientbound)
	if err != nil {
		t.Fatalf("ParseDecompile: %v", err)
	}
	if calls[0].Op != "Decode4" {
		t.Fatalf("call[0] = %+v, want count Decode4", calls[0])
	}
	if !strings.HasPrefix(calls[1].Guard, "loop ") {
		t.Errorf("call[1].Guard = %q, want 'loop ...' prefix", calls[1].Guard)
	}
	if calls[1].Op != "Decode4" || calls[2].Op != "DecodeStr" {
		t.Errorf("loop body ops = %q,%q want Decode4,DecodeStr", calls[1].Op, calls[2].Op)
	}
}

func TestParseModeSwitch(t *testing.T) {
	calls, err := ParseDecompile(mustFixture(t, "mode_switch.c"), DirClientbound)
	if err != nil {
		t.Fatalf("ParseDecompile: %v", err)
	}
	// mode read unguarded; case bodies guarded "mode == N"
	if calls[0].Op != "Decode1" || calls[0].Guard != "" {
		t.Fatalf("call[0] = %+v, want unguarded mode Decode1", calls[0])
	}
	byGuard := map[string][]string{}
	for _, c := range calls[1:] {
		byGuard[c.Guard] = append(byGuard[c.Guard], c.Op)
	}
	if got := byGuard["mode == 0"]; len(got) != 1 || got[0] != "Decode4" {
		t.Errorf("case 0 = %v, want [Decode4]", got)
	}
	if got := byGuard["mode == 1"]; len(got) != 2 || got[0] != "DecodeStr" || got[1] != "Decode2" {
		t.Errorf("case 1 = %v, want [DecodeStr Decode2]", got)
	}
}

func TestParseLoopBreakInsideCase(t *testing.T) {
	calls, err := ParseDecompile(mustFixture(t, "loop_in_case.c"), DirClientbound)
	if err != nil {
		t.Fatalf("ParseDecompile: %v", err)
	}
	// Map op -> guard for the four case-body reads (a,b,c,d). The discriminator
	// read (mode) is calls[0], unguarded.
	if calls[0].Guard != "" {
		t.Fatalf("discriminator must be unguarded, got %q", calls[0].Guard)
	}
	body := calls[1:]
	wants := []struct{ op, guard string }{
		{"Decode4", "mode == 1"},
		{"Decode2", "mode == 1 && loop count"},
		{"Decode1", "mode == 1 && loop count"}, // read after the LOOP break, still in loop+case
		{"Decode8", "mode == 1"},               // read after the loop, still in case
	}
	if len(body) != len(wants) {
		t.Fatalf("got %d body reads, want %d: %+v", len(body), len(wants), body)
	}
	for i, w := range wants {
		if body[i].Op != w.op || body[i].Guard != w.guard {
			t.Errorf("read[%d] = {%s %q}, want {%s %q}", i, body[i].Op, body[i].Guard, w.op, w.guard)
		}
	}
}

func TestParseUnresolvedIndirect(t *testing.T) {
	calls, err := ParseDecompile(mustFixture(t, "indirect_dispatch.c"), DirClientbound)
	if err != nil {
		t.Fatalf("ParseDecompile: %v", err)
	}
	last := calls[len(calls)-1]
	if last.Op != "Unresolved" {
		t.Fatalf("last call = %+v, want Unresolved (indirect dispatch must not be guessed)", last)
	}
	if last.Comment == "" {
		t.Errorf("Unresolved must carry a reason comment")
	}
}

func TestParseUnresolvedNoFalsePositives(t *testing.T) {
	src := "int __thiscall CFoo::OnBar(CFoo *this, CInPacket *a2)\n" +
		"{\n" +
		"  int id = CInPacket::Decode4(a2);   // id\n" +
		"  if ( a2 )                          // null check — NOT a packet read\n" +
		"  {\n" +
		"    CInPacket::Decode1(a2);          // flag\n" +
		"  }\n" +
		"  while ( a2 )                       // condition — NOT a packet read\n" +
		"  {\n" +
		"    CInPacket::Decode2(a2);          // more\n" +
		"  }\n" +
		"  CUIFadeYesNo::Create(a2);          // denylisted helper — skip\n" +
		"  return id;\n" +
		"}\n"
	calls, err := ParseDecompile(src, DirClientbound)
	if err != nil {
		t.Fatalf("ParseDecompile: %v", err)
	}
	for i, c := range calls {
		if c.Op == "Unresolved" {
			t.Errorf("call[%d] = %+v — no Unresolved expected (if/while conditions + denylisted helper must not flood)", i, c)
		}
	}
	// The three real reads must still be present.
	var ops []string
	for _, c := range calls {
		ops = append(ops, c.Op)
	}
	if len(calls) != 3 || ops[0] != "Decode4" || ops[1] != "Decode1" || ops[2] != "Decode2" {
		t.Fatalf("got ops %v, want [Decode4 Decode1 Decode2] (no spurious Unresolved, denylisted skipped)", ops)
	}
}

func TestParseAliasSetDelegate(t *testing.T) {
	// v3 is never directly Decoded — it is an alias of a2 (the seeded packet var
	// via the Decode4 call). A helper passing v3 must still be descended as a
	// Delegate, recognized via the alias set. Also exercises a /* line: N */
	// prefix on one line.
	src := "int __thiscall CFoo::OnBar(CFoo *this, CInPacket *a2)\n" +
		"{\n" +
		"  int id = CInPacket::Decode4(a2);   // id\n" +
		"/* line: 5 */  v3 = a2;\n" +
		"  someHelper(v3);                    // descend via alias\n" +
		"  return id;\n" +
		"}\n"
	calls, err := ParseDecompile(src, DirClientbound)
	if err != nil {
		t.Fatalf("ParseDecompile: %v", err)
	}
	var del *rawCall
	for i := range calls {
		if calls[i].Op == "Delegate" {
			del = &calls[i]
		}
	}
	if del == nil {
		t.Fatalf("expected a Delegate for someHelper(v3) via alias set, got %+v", calls)
	}
	if del.Ref != "someHelper" {
		t.Errorf("Delegate.Ref = %q, want someHelper", del.Ref)
	}
	// The Decode4 read must also be present.
	if calls[0].Op != "Decode4" {
		t.Errorf("call[0] = %+v, want Decode4 id", calls[0])
	}
}

func TestParseRealOnFriendResult(t *testing.T) {
	calls, err := ParseDecompile(mustFixture(t, "real_onfriendresult_v83.c"), DirClientbound)
	if err != nil {
		t.Fatalf("ParseDecompile: %v", err)
	}
	if len(calls) == 0 {
		t.Fatalf("no calls parsed from real fixture")
	}

	// calls[0] is the switch discriminator: Decode1(Index), read before any case
	// => empty guard.
	if calls[0].Op != "Decode1" {
		t.Fatalf("call[0].Op = %q, want Decode1 (switch discriminator)", calls[0].Op)
	}
	if calls[0].Guard != "" {
		t.Errorf("call[0].Guard = %q, want empty (discriminator read before cases)", calls[0].Guard)
	}

	// Exactly one sub_A40028 Delegate — the GW_Friend struct helper is DESCENDED,
	// not dropped, not Unresolved.
	var subDelegate *rawCall
	subCount := 0
	for i := range calls {
		if calls[i].Op == "Delegate" && calls[i].Ref == "sub_A40028" {
			subCount++
			subDelegate = &calls[i]
		}
	}
	if subCount != 1 {
		t.Fatalf("got %d sub_A40028 Delegates, want exactly 1: %+v", subCount, calls)
	}
	// Not mistraced as a loop.
	if strings.Contains(subDelegate.Guard, "loop ") {
		t.Errorf("sub_A40028 Delegate Guard = %q, must NOT contain 'loop '", subDelegate.Guard)
	}

	// case 9 inline reads: Decode4 (friendId) + DecodeStr (name).
	var hasDecode4, hasDecodeStr bool
	for _, c := range calls {
		if c.Op == "Decode4" {
			hasDecode4 = true
		}
		if c.Op == "DecodeStr" {
			hasDecodeStr = true
		}
	}
	if !hasDecode4 {
		t.Errorf("expected a Decode4 read (friendId)")
	}
	if !hasDecodeStr {
		t.Errorf("expected a DecodeStr read (name)")
	}

	// Named packet helpers are also descended (best-effort assertion).
	var hasFriendHelper bool
	for _, c := range calls {
		if c.Op == "Delegate" && strings.HasPrefix(c.Ref, "CWvsContext::CFriend::") {
			hasFriendHelper = true
		}
	}
	if !hasFriendHelper {
		t.Errorf("expected a Delegate to a CWvsContext::CFriend:: helper")
	}

	// Case 9 (#Invite) must be EXACTLY [Decode4 friendId, DecodeStr name,
	// Delegate sub_A40028] — no phantom Unresolved (from the unmatched
	// templated `ZXString<char>::_Release(&Index)` falling through), and NO
	// phantom Delegates (`sub_428211`, `sub_A4046F`) synthesized from the
	// recycled `Index` scratch var that Hex-Rays reassigns mid-case. The
	// discriminator is an inline `switch ( CInPacket::Decode1(Index) )`
	// expression, so the synthesized guard label is "switch == 9".
	const case9Guard = "switch == 9"
	var case9 []rawCall
	for _, c := range calls {
		if c.Guard == case9Guard {
			case9 = append(case9, c)
		}
	}
	type opref struct{ op, ref string }
	wantCase9 := []opref{
		{"Decode4", ""},
		{"DecodeStr", ""},
		{"Delegate", "sub_A40028"},
	}
	if len(case9) != len(wantCase9) {
		t.Fatalf("case 9 (%q) = %d calls, want exactly %d %+v: got %+v",
			case9Guard, len(case9), len(wantCase9), wantCase9, case9)
	}
	for i, w := range wantCase9 {
		if case9[i].Op != w.op || case9[i].Ref != w.ref {
			t.Errorf("case 9 call[%d] = {Op:%q Ref:%q}, want {Op:%q Ref:%q}",
				i, case9[i].Op, case9[i].Ref, w.op, w.ref)
		}
	}
	// Belt-and-suspenders: no Unresolved and none of the known phantom
	// Delegate refs may appear anywhere in case 9.
	for _, c := range case9 {
		if c.Op == "Unresolved" {
			t.Errorf("case 9 has phantom Unresolved (templated denylist bypass): %+v", c)
		}
		if c.Op == "Delegate" && (c.Ref == "sub_428211" || c.Ref == "sub_A4046F") {
			t.Errorf("case 9 has phantom Delegate %q (recycled Index scratch var)", c.Ref)
		}
	}
}

func TestParseSkipsNonPacketHelpers(t *testing.T) {
	calls, err := ParseDecompile(mustFixture(t, "nonpacket_skip.c"), DirClientbound)
	if err != nil {
		t.Fatalf("ParseDecompile: %v", err)
	}
	for _, c := range calls {
		if c.Op == "Delegate" {
			t.Errorf("unexpected Delegate %q — non-packet/denylisted helpers must be skipped", c.Ref)
		}
	}
	if len(calls) != 1 || calls[0].Op != "Decode4" {
		t.Fatalf("got %+v, want only [Decode4 id]", calls)
	}
}

// TestParseClientboundIgnoresResponse proves direction filtering for a
// clientbound handler that READS an incoming packet and then BUILDS+SENDS a
// response: only the CInPacket::Decode* reads must be captured. The outgoing
// COutPacket::Encode* writes and the SendPacket delegate must NOT appear.
func TestParseClientboundIgnoresResponse(t *testing.T) {
	calls, err := ParseDecompile(mustFixture(t, "clientbound_with_response.c"), DirClientbound)
	if err != nil {
		t.Fatalf("ParseDecompile: %v", err)
	}
	var ops []string
	for _, c := range calls {
		ops = append(ops, c.Op)
		if c.Op == "Delegate" {
			t.Errorf("clientbound handler must not emit a Delegate (oPacket is a COutPacket, not a CInPacket alias): %+v", c)
		}
	}
	if len(calls) != 2 || ops[0] != "Decode1" || ops[1] != "Decode4" {
		t.Fatalf("got ops %v, want exactly [Decode1 Decode4] (outgoing Encode/SendPacket ignored)", ops)
	}
}

func TestParseCaseLabelUnsignedSuffix(t *testing.T) {
	calls, err := ParseDecompile(mustFixture(t, "switch_usuffix.c"), DirClientbound)
	if err != nil {
		t.Fatalf("ParseDecompile: %v", err)
	}
	// case 9u body reads must be guarded "switch == 9" (suffix stripped).
	var got []string
	for _, c := range calls {
		if strings.Contains(c.Guard, "switch == 9") {
			got = append(got, c.Op)
		}
	}
	if len(got) != 2 || got[0] != "Decode4" || got[1] != "DecodeStr" {
		t.Fatalf("case 9u body = %v, want [Decode4 DecodeStr] guarded 'switch == 9'", got)
	}
	// 0x12u (=18) must NOT leak into case 9.
	for _, c := range calls {
		if strings.Contains(c.Guard, "switch == 9") && c.Op == "Decode2" {
			t.Errorf("0x12u case leaked into case 9: %+v", c)
		}
	}
}

// TestParseServerboundCapturesWrites proves direction filtering for a
// serverbound Send function that WRITES a COutPacket: only the
// COutPacket::Encode* writes must be captured (emitted as canonical Decode*
// op strings). The SendPacket delegate must NOT appear.
func TestParseServerboundCapturesWrites(t *testing.T) {
	calls, err := ParseDecompile(mustFixture(t, "serverbound_send.c"), DirServerbound)
	if err != nil {
		t.Fatalf("ParseDecompile: %v", err)
	}
	var ops []string
	for _, c := range calls {
		ops = append(ops, c.Op)
		if c.Op == "Delegate" {
			t.Errorf("serverbound Send must not emit a Delegate for SendPacket: %+v", c)
		}
	}
	if len(calls) != 2 || ops[0] != "Decode4" || ops[1] != "Decode1" {
		t.Fatalf("got ops %v, want exactly [Decode4 Decode1] (Encode4->Decode4, Encode1->Decode1 canonical)", ops)
	}
}

func TestParseDecompile_IfElseDispatch(t *testing.T) {
	calls, err := ParseDecompile(mustFixture(t, "ifelse_chain.c"), DirClientbound)
	if err != nil {
		t.Fatal(err)
	}
	type rc struct{ op, guard string }
	var got []rc
	for _, c := range calls {
		switch c.Op {
		case "Decode1", "Decode2", "Decode4", "Decode8", "DecodeStr", "DecodeBuf":
			got = append(got, rc{c.Op, c.Guard})
		}
	}
	want := []rc{
		{"Decode1", ""},            // result code, pre-branch
		{"Decode1", "result == 2"}, // arm result==2
		{"Decode8", "result == 2"},
		{"Decode4", "result == 5"}, // arm result==5
		{"Decode2", "<default>"},   // bare else -> default token
	}
	if len(got) != len(want) {
		t.Fatalf("got %d guarded reads, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("read[%d]=%+v want %+v", i, got[i], want[i])
		}
	}
}

func TestParseDecompile_IfElseTrailingElse(t *testing.T) {
	calls, err := ParseDecompile(mustFixture(t, "ifelse_else.c"), DirClientbound)
	if err != nil {
		t.Fatal(err)
	}
	sawDefault := false
	for _, c := range calls {
		if c.Guard == DefaultGuardToken {
			sawDefault = true
		}
	}
	if !sawDefault {
		t.Fatalf("expected a <default> guard on the bare-else arm; got %+v", calls)
	}
}

func TestParseDecompile_CaseLabelSet(t *testing.T) {
	f, err := ParseDecompileFields(mustFixture(t, "switch_emptycase.c"), DirClientbound)
	if err != nil {
		t.Fatal(err)
	}
	got := f.CaseLabels["mode"]
	if got == nil {
		t.Fatal("no case labels for 'mode'")
	}
	for _, want := range []int64{1, 2, 3} {
		if !got.Has(want) {
			t.Fatalf("missing case label %d; have %v", want, got.Values())
		}
	}
}

func TestParseDecompile_NonEqVerbatimGuards(t *testing.T) {
	calls, err := ParseDecompile(mustFixture(t, "ifelse_noneq.c"), DirClientbound)
	if err != nil {
		t.Fatal(err)
	}
	byOp := map[string]string{}
	for _, c := range calls {
		byOp[c.Op] = c.Guard
	}
	if byOp["Decode2"] != "v5 < 5" {
		t.Errorf("Decode2 guard = %q, want \"v5 < 5\"", byOp["Decode2"])
	}
	if byOp["Decode4"] != "v5 & 0x10" {
		t.Errorf("Decode4 guard = %q, want \"v5 & 0x10\"", byOp["Decode4"])
	}
	if byOp["Decode8"] != DefaultGuardToken {
		t.Errorf("Decode8 guard = %q, want %q", byOp["Decode8"], DefaultGuardToken)
	}
}

func TestParseDecompileFields_HasMultiwayDispatch(t *testing.T) {
	cases := []struct {
		fixture string
		want    bool
	}{
		{"leaf_linear.c", false},     // lone optional if
		{"multiway_if.c", true},      // 2-arm equality chain
		{"multiway_compound.c", true}, // 2nd arm compound predicate (else if a && b)
		{"mode_switch.c", true},      // switch with 2 cases
		{"switch_emptycase.c", true}, // switch with 3 cases
		{"linear.c", false},          // no branches at all
	}
	for _, tc := range cases {
		f, err := ParseDecompileFields(mustFixture(t, tc.fixture), DirClientbound)
		if err != nil {
			t.Fatalf("%s: %v", tc.fixture, err)
		}
		if f.HasMultiwayDispatch != tc.want {
			t.Errorf("%s: HasMultiwayDispatch=%v want %v", tc.fixture, f.HasMultiwayDispatch, tc.want)
		}
	}
}
