package idasrc

import "testing"

// TestReadsAreConditionalFlat: a handler whose only packet reads are at the
// function's top level (no switch/if/for above them) is NOT conditional — the
// flat read-order compare the audit performs is valid.
func TestReadsAreConditionalFlat(t *testing.T) {
	text := "void __thiscall Foo::OnA(Foo *this, CInPacket *a2)\n" +
		"{\n" +
		"  CInPacket::Decode1(a2);\n" +
		"  CInPacket::Decode4(a2);\n" +
		"  CInPacket::DecodeStr(a2);\n" +
		"}\n"
	if ReadsAreConditional(text, DirClientbound) {
		t.Errorf("flat top-level reads must be NON-conditional")
	}
}

// TestReadsAreConditionalSwitch: a `switch ( CInPacket::Decode1(a2) )` dispatch
// with reads inside the case bodies branches on an early read → conditional.
func TestReadsAreConditionalSwitch(t *testing.T) {
	text := "void __thiscall Foo::OnSwitch(Foo *this, CInPacket *a2)\n" +
		"{\n" +
		"  switch ( CInPacket::Decode1(a2) )\n" +
		"  {\n" +
		"    case 1:\n" +
		"      CInPacket::Decode4(a2);\n" +
		"      break;\n" +
		"    case 2:\n" +
		"      CInPacket::DecodeStr(a2);\n" +
		"      break;\n" +
		"  }\n" +
		"}\n"
	if !ReadsAreConditional(text, DirClientbound) {
		t.Errorf("switch-dispatch with case-body reads must be CONDITIONAL")
	}
}

// TestReadsAreConditionalBareIf: the DropDestroy shape — read a leave-type byte
// at top level, then `if ( v == 2 ) { Decode4(...); }`. The guarded Decode4
// makes the handler conditional (the flat-union compare is invalid).
func TestReadsAreConditionalBareIf(t *testing.T) {
	text := "void __thiscall CDropPool::OnDropLeaveField(CDropPool *this, CInPacket *a2)\n" +
		"{\n" +
		"  v3 = CInPacket::Decode1(a2);\n" +
		"  CInPacket::Decode4(a2);\n" +
		"  if ( v3 == 2 )\n" +
		"  {\n" +
		"    CInPacket::Decode4(a2);\n" +
		"  }\n" +
		"  CInPacket::Decode1(a2);\n" +
		"}\n"
	if !ReadsAreConditional(text, DirClientbound) {
		t.Errorf("bare if-on-byte guarded read (DropDestroy shape) must be CONDITIONAL")
	}
}

// TestReadsAreConditionalLoopBody: a read inside a `for` loop body is
// conditional (the count gates how many times it fires).
func TestReadsAreConditionalLoopBody(t *testing.T) {
	text := "void __thiscall Foo::OnLoop(Foo *this, CInPacket *a2)\n" +
		"{\n" +
		"  count = CInPacket::Decode4(a2);\n" +
		"  for ( i = 0; i < count; ++i )\n" +
		"  {\n" +
		"    CInPacket::Decode4(a2);\n" +
		"  }\n" +
		"}\n"
	if !ReadsAreConditional(text, DirClientbound) {
		t.Errorf("read inside a for-loop body must be CONDITIONAL")
	}
}

// TestReadsAreConditionalBenignNullCheck: top-level reads with a TRAILING
// benign `if ( a2 )` null-check that contains NO read must stay NON-conditional
// — a conditional block with no relevant read inside does not make the handler
// branch on a read value.
func TestReadsAreConditionalBenignNullCheck(t *testing.T) {
	text := "void __thiscall Foo::OnBenign(Foo *this, CInPacket *a2)\n" +
		"{\n" +
		"  CInPacket::Decode1(a2);\n" +
		"  CInPacket::Decode4(a2);\n" +
		"  if ( a2 )\n" +
		"  {\n" +
		"    CWnd::Refresh(this);\n" +
		"  }\n" +
		"}\n"
	if ReadsAreConditional(text, DirClientbound) {
		t.Errorf("trailing null-check with no read inside must be NON-conditional")
	}
}

// TestReadsAreConditionalServerbound: the serverbound (COutPacket::Encode*)
// matcher is honored — an Encode read inside an if-block is conditional.
func TestReadsAreConditionalServerbound(t *testing.T) {
	text := "void __thiscall Foo::Send(Foo *this, COutPacket *a2)\n" +
		"{\n" +
		"  COutPacket::Encode1(a2, 1);\n" +
		"  if ( this->flag )\n" +
		"  {\n" +
		"    COutPacket::Encode4(a2, this->id);\n" +
		"  }\n" +
		"}\n"
	if !ReadsAreConditional(text, DirServerbound) {
		t.Errorf("serverbound Encode inside an if-block must be CONDITIONAL")
	}
	// And a clientbound classification of the same text reads NO CInPacket calls,
	// so it must be non-conditional (direction scoping).
	if ReadsAreConditional(text, DirClientbound) {
		t.Errorf("clientbound view of a serverbound handler has no CInPacket reads → non-conditional")
	}
}

// -- HasRepeatingRun tests --

// TestHasRepeatingRunD4D2D2x3: [D4,D2,D2, D4,D2,D2, D4,D2,D2] — block [D4,D2,D2]
// repeats 3 times; L=3, 3*3=9 >= 6 → true.
func TestHasRepeatingRunD4D2D2x3(t *testing.T) {
	ops := []Primitive{Decode4, Decode2, Decode2, Decode4, Decode2, Decode2, Decode4, Decode2, Decode2}
	if !HasRepeatingRun(ops) {
		t.Error("[D4,D2,D2]×3 must be detected as a repeating run")
	}
}

// TestHasRepeatingRunD1x3: [D1,D1,D1] — only 3 reads; L=1, 1*3=3 < 6 → false.
func TestHasRepeatingRunD1x3(t *testing.T) {
	ops := []Primitive{Decode1, Decode1, Decode1}
	if HasRepeatingRun(ops) {
		t.Error("[D1,D1,D1] has only 3 reads (L*3=3 < 6), must NOT be a repeating run")
	}
}

// TestHasRepeatingRunD1x6: [D1,D1,D1,D1,D1,D1] — 6 reads; L=1, 1*3=3 < 6 → but
// wait — L=2 would be [D1,D1]×3 = 6 reads; L*3=6 >= 6 → true.
// Note: L=1 triplet check: L*3=3 < 6, skip. L=2: [D1,D1]×3, 2*3=6 >= 6 → true.
func TestHasRepeatingRunD1x6(t *testing.T) {
	ops := []Primitive{Decode1, Decode1, Decode1, Decode1, Decode1, Decode1}
	if !HasRepeatingRun(ops) {
		t.Error("[D1]×6 must be a repeating run (L=2 block [D1,D1]×3, 2*3=6 >= 6)")
	}
}

// TestHasRepeatingRunMixed: [D1,D4,DecodeStr] — no repetition → false.
func TestHasRepeatingRunMixed(t *testing.T) {
	ops := []Primitive{Decode1, Decode4, DecodeStr}
	if HasRepeatingRun(ops) {
		t.Error("[D1,D4,DecodeStr] must NOT be a repeating run")
	}
}

// TestHasRepeatingRunRepeatNotAtStart: [D1, D4,D2,D2, D4,D2,D2, D4,D2,D2, D1] —
// the [D4,D2,D2]×3 block starts at index 1; surrounded by non-repeating elements.
func TestHasRepeatingRunRepeatNotAtStart(t *testing.T) {
	ops := []Primitive{Decode1, Decode4, Decode2, Decode2, Decode4, Decode2, Decode2, Decode4, Decode2, Decode2, Decode1}
	if !HasRepeatingRun(ops) {
		t.Error("[D1, D4,D2,D2, D4,D2,D2, D4,D2,D2, D1] must be a repeating run (block not at start)")
	}
}
