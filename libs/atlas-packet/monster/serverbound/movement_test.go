package serverbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestMonsterMovementVersionBoundary pins the v84 mob-move structure against the
// client. CONFIRMED via v83 CMob::GenerateMovePath (@0x66b6fc, opcode 0xBC) vs
// v84 sub_6818C3 (opcode 0xC1): v84 inserts multiTargetForBall +
// randTimeForAreaAttack between skillData and moveFlags, but (like v83) writes
// neither hackedCodeCRC nor the trailing chase block (those remain v87+). So a
// v84 encode must be longer than v83 (added skill fields) yet shorter than v87
// (no CRC/chase).
// packet-audit:verify packet=monster/serverbound/MonsterMovementRequest version=gms_v83 ida=0x66b6fc
// packet-audit:verify packet=monster/serverbound/MonsterMovementRequest version=gms_v87 ida=0x6a6381
// packet-audit:verify packet=monster/serverbound/MonsterMovementRequest version=gms_v95 ida=0x651100
// packet-audit:verify packet=monster/serverbound/MonsterMovementRequest version=gms_v84 ida=0x6818c3
//
// jms_v185 is verified by TestMonsterMovementBytesJMS185 below. A RoundTrip
// cannot catch a tail the client sends and the decoder never reads, or a tail
// placed at the wrong offset relative to the trailing chase block.
func TestMonsterMovementVersionBoundary(t *testing.T) {
	p := MovementRequest{uniqueId: 1, moveId: 2, skillData: 0x0305, hackedCodeCRC: 9, tChaseDuration: 9}
	enc := func(major uint16) []byte {
		ctx := test.CreateContext("GMS", major, 1)
		return test.Encode(t, ctx, p.Encode, nil)
	}
	v83, v84, v87 := enc(83), enc(84), enc(87)
	if len(v84) <= len(v83) {
		t.Errorf("v84 (%d) must be longer than v83 (%d): multiTarget/randTime added in v84", len(v84), len(v83))
	}
	if len(v84) >= len(v87) {
		t.Errorf("v84 (%d) must be shorter than v87 (%d): hackedCodeCRC/chase block are v87+, not v84", len(v84), len(v87))
	}
}

func TestMonsterMovement(t *testing.T) {
	p := MovementRequest{}
	p.uniqueId = 1001
	p.moveId = 55
	p.dwFlag = 1
	p.nActionAndDir = -3
	p.skillData = 0x0305 // skillId=5, skillLevel=3
	p.moveFlags = 0
	p.hackedCode = 0
	p.flyCtxTargetX = 100
	p.flyCtxTargetY = 200
	p.hackedCodeCRC = 999
	p.bChasing = 1
	p.hasTarget = 0
	p.bChasing2 = 1
	p.bChasingHack = 0
	p.tChaseDuration = 500

	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, p.Encode, p.Decode, nil)

			if p.UniqueId() != 1001 {
				t.Errorf("expected uniqueId 1001, got %d", p.UniqueId())
			}
			if p.MoveId() != 55 {
				t.Errorf("expected moveId 55, got %d", p.MoveId())
			}
			if p.DwFlag() != 1 {
				t.Errorf("expected dwFlag 1, got %d", p.DwFlag())
			}
			if !p.MonsterMoveStartResult() {
				t.Error("expected monsterMoveStartResult true")
			}
			if p.ActionAndDir() != -3 {
				t.Errorf("expected nActionAndDir -3, got %d", p.ActionAndDir())
			}
			if p.SkillId() != 5 {
				t.Errorf("expected skillId 5, got %d", p.SkillId())
			}
			if p.SkillLevel() != 3 {
				t.Errorf("expected skillLevel 3, got %d", p.SkillLevel())
			}
		})
	}
}

func TestMonsterMovementGMS28(t *testing.T) {
	// GMS v28 does not have multiTargetForBall, randTimeForAreaAttack, hackedCodeCRC, or chasing fields.
	p := MovementRequest{}
	p.uniqueId = 2002
	p.moveId = 10
	p.dwFlag = 0
	p.nActionAndDir = 1
	p.skillData = 0
	p.moveFlags = 0
	p.hackedCode = 0
	p.flyCtxTargetX = 0
	p.flyCtxTargetY = 0

	ctx := test.CreateContext("GMS", 28, 1)
	test.RoundTrip(t, ctx, p.Encode, p.Decode, nil)

	if p.UniqueId() != 2002 {
		t.Errorf("expected uniqueId 2002, got %d", p.UniqueId())
	}
	if p.MonsterMoveStartResult() {
		t.Error("expected monsterMoveStartResult false")
	}
}

// TestMonsterMovementBytesV79 pins the exact wire bytes against the v79 client
// send order. MOVE_LIFE (CMob move flush) is the unnamed sub_63A226 @0x63a226
// (GMS_v79_1_DEVM.exe, port 13340), opcode 180; the COutPacket build block is at
// @0x63a799:
//
//	COutPacket(180) @0x63a799
//	Encode4 @0x63a7ba — fused mob id (sub_4DC1C0(this+380, m_dwMobID)) -> uniqueId
//	Encode2 @0x63a7ec — move SN counter                               -> moveId
//	Encode1 @0x63a803 — flags                                         -> dwFlag
//	Encode1 @0x63a80e — (2*action)|dir                               -> nActionAndDir
//	Encode4 @0x63a819 — skillData (HIDWORD)                          -> skillData
//	Encode1 @0x63a83b — moveFlags                                    -> moveFlags
//	Encode4 @0x63a849 — hackedCode (v12[288])                        -> hackedCode
//	Encode4 @0x63a867 — flyCtx target X                              -> flyCtxTargetX
//	Encode4 @0x63a880 — flyCtx target Y                              -> flyCtxTargetY
//	CMovePath::Flush @0x63a8c6 — opaque movement payload (§5)
//
// v79 (<84) writes NO multiTargetForBall/randTimeForAreaAttack (v84+), NO
// hackedCodeCRC and NO trailing chase block (v87+) — exactly the v83 baseline
// path of the existing codec. model.Movement is OPAQUE (§5); fixtured empty
// (StartX/StartY int16 + 0 element-count = 5 deterministic bytes). No codec change.
//
// packet-audit:verify packet=monster/serverbound/MonsterMovementRequest version=gms_v79 ida=0x63a226
func TestMonsterMovementBytesV79(t *testing.T) {
	p := MovementRequest{}
	p.uniqueId = 1001
	p.moveId = 55
	p.dwFlag = 1
	p.nActionAndDir = -3
	p.skillData = 0x0305
	p.moveFlags = 0
	p.hackedCode = 0
	p.flyCtxTargetX = 100
	p.flyCtxTargetY = 200
	// v87+ fields set but gated off at v79:
	p.hackedCodeCRC = 999
	p.tChaseDuration = 500

	ctx := test.CreateContext("GMS", 79, 1)
	want := []byte{
		0xE9, 0x03, 0x00, 0x00, // uniqueId 1001 (Encode4 @0x63a7ba)
		0x37, 0x00, // moveId 55 (Encode2 @0x63a7ec)
		0x01,                   // dwFlag 1 (Encode1 @0x63a803)
		0xFD,                   // nActionAndDir -3 (Encode1 @0x63a80e)
		0x05, 0x03, 0x00, 0x00, // skillData 0x0305 (Encode4 @0x63a819)
		0x00,                   // moveFlags 0 (Encode1 @0x63a83b)
		0x00, 0x00, 0x00, 0x00, // hackedCode 0 (Encode4 @0x63a849)
		0x64, 0x00, 0x00, 0x00, // flyCtxTargetX 100 (Encode4 @0x63a867)
		0xC8, 0x00, 0x00, 0x00, // flyCtxTargetY 200 (Encode4 @0x63a880)
		// opaque movement (empty): StartX int16, StartY int16, count byte
		0x00, 0x00, 0x00, 0x00, 0x00,
	}
	got := test.Encode(t, ctx, p.Encode, nil)
	if !bytes.Equal(got, want) {
		t.Errorf("v79 movement bytes:\n got % x\nwant % x", got, want)
	}
}

// TestMonsterMovementBytesV72 pins the exact wire bytes against the v72 client
// send order. MOVE_LIFE is sub_61AA54 @0x61aa54 (GMS_v72.1_U_DEVM.exe, port
// 13339), opcode 178; the COutPacket build block is at @0x61af58:
//
//	COutPacket(178) @0x61af58
//	Encode4 @0x61af78 — fused mob id                 -> uniqueId
//	Encode2 @0x61afaa — move SN counter              -> moveId
//	Encode1 @0x61afbc — flags                        -> dwFlag
//	Encode1 @0x61afc7 — (2*action)|dir               -> nActionAndDir
//	Encode4 @0x61afd2 — skillData (a4)               -> skillData
//	Encode1 @0x61aff4 — moveFlags                    -> moveFlags
//	Encode4 @0x61b002 — hackedCode (v14[286])        -> hackedCode
//	CMovePath::Flush @0x61b048 — opaque movement payload (§5)
//
// v72 (<79) writes hackedCode then goes STRAIGHT to Flush — it OMITS
// flyCtxTargetX/flyCtxTargetY (added at v79), plus NO multiTarget/randTime (v84+),
// hackedCodeCRC and chase block (v87+). Legacy gate movement.go. model.Movement
// OPAQUE (§5); fixtured empty (5 bytes).
//
// packet-audit:verify packet=monster/serverbound/MonsterMovementRequest version=gms_v72 ida=0x61aa54
func TestMonsterMovementBytesV72(t *testing.T) {
	p := MovementRequest{}
	p.uniqueId = 1001
	p.moveId = 55
	p.dwFlag = 1
	p.nActionAndDir = -3
	p.skillData = 0x0305
	p.moveFlags = 0
	p.hackedCode = 0
	// v79+ fields set but gated off at v72:
	p.flyCtxTargetX = 100
	p.flyCtxTargetY = 200
	p.hackedCodeCRC = 999
	p.tChaseDuration = 500

	ctx := test.CreateContext("GMS", 72, 1)
	want := []byte{
		0xE9, 0x03, 0x00, 0x00, // uniqueId 1001 (Encode4 @0x61af78)
		0x37, 0x00, // moveId 55 (Encode2 @0x61afaa)
		0x01,                   // dwFlag 1 (Encode1 @0x61afbc)
		0xFD,                   // nActionAndDir -3 (Encode1 @0x61afc7)
		0x05, 0x03, 0x00, 0x00, // skillData 0x0305 (Encode4 @0x61afd2)
		0x00,                   // moveFlags 0 (Encode1 @0x61aff4)
		0x00, 0x00, 0x00, 0x00, // hackedCode 0 (Encode4 @0x61b002)
		// flyCtxTargetX/Y OMITTED (v79+)
		// opaque movement (empty): StartX int16, StartY int16, count byte
		0x00, 0x00, 0x00, 0x00, 0x00,
	}
	got := test.Encode(t, ctx, p.Encode, nil)
	if !bytes.Equal(got, want) {
		t.Errorf("v72 movement bytes:\n got % x\nwant % x", got, want)
	}
}

func TestMonsterMovementOperationString(t *testing.T) {
	p := MovementRequest{}
	if p.Operation() != MonsterMovementHandle {
		t.Errorf("expected operation %s, got %s", MonsterMovementHandle, p.Operation())
	}
	if p.String() == "" {
		t.Error("expected non-empty string")
	}
}

// TestMonsterMovementBytesJMS185 pins the jms_v185 MOVE_LIFE wire, hand-built
// from the decompiled field order — this is NOT captured wire (no live
// MOVE_LIFE frame was obtained; log sweeps timed out). See
// docs/tasks/fix-jms185-attack-decode/sibling-movement-ops-findings.md §2.
//
// CMob::GenerateMovePath @0x6e8892 calls CMovePath::Flush @0x6e9423 (which
// writes the keypad+rect tail — see monsterMoveKeyPadTail) and only THEN
// writes bChasing/hasTarget/bChasing2/bChasingHack/tChaseDuration, so the
// tail must sit BETWEEN the movement blob and that trailing block, not after
// it. This fixture asserts that exact placement byte-for-byte.
//
// packet-audit:verify packet=monster/serverbound/MonsterMovementRequest version=jms_v185 ida=0x6e8892
func TestMonsterMovementBytesJMS185(t *testing.T) {
	p := MovementRequest{}
	p.uniqueId = 1001
	p.moveId = 55
	p.dwFlag = 1
	p.nActionAndDir = -3
	p.skillData = 0x0305
	p.moveFlags = 0
	p.hackedCode = 0
	p.flyCtxTargetX = 100
	p.flyCtxTargetY = 200
	p.hackedCodeCRC = 999
	// movement stays empty (StartX/StartY + 0 elements = 5 bytes)
	p.movement.StartX = 10
	p.movement.StartY = 20
	// keyPadStates has an ODD entry count (3), exercising the final-byte
	// low-nibble-only packing rule.
	p.keyPadStates = []byte{1, 2, 3}
	p.moveRectLeft = -10
	p.moveRectTop = 20
	p.moveRectRight = 30
	p.moveRectBottom = -40
	p.bChasing = 1
	p.hasTarget = 0
	p.bChasing2 = 1
	p.bChasingHack = 0
	p.tChaseDuration = 500

	ctx := test.CreateContext("JMS", 185, 1)
	want := []byte{
		0xE9, 0x03, 0x00, 0x00, // uniqueId 1001
		0x37, 0x00, // moveId 55
		0x01,                   // dwFlag 1
		0xFD,                   // nActionAndDir -3
		0x05, 0x03, 0x00, 0x00, // skillData 0x0305
		0x00, 0x00, 0x00, 0x00, // multiTargetForBall (empty: count=0)
		0x00, 0x00, 0x00, 0x00, // randTimeForAreaAttack (empty: count=0)
		0x00,                   // moveFlags 0
		0x00, 0x00, 0x00, 0x00, // hackedCode 0
		0x64, 0x00, 0x00, 0x00, // flyCtxTargetX 100
		0xC8, 0x00, 0x00, 0x00, // flyCtxTargetY 200
		0xE7, 0x03, 0x00, 0x00, // hackedCodeCRC 999
		0x0A, 0x00, 0x14, 0x00, 0x00, // movement: StartX=10, StartY=20, count=0
		// --- keypad + rect tail (monsterMoveKeyPadTail) ---
		0x03,       // keypad entry count = 3
		0x21, 0x03, // packed nibbles: (1|2<<4)=0x21, (3, no high nibble)=0x03
		0xF6, 0xFF, // moveRectLeft -10
		0x14, 0x00, // moveRectTop 20
		0x1E, 0x00, // moveRectRight 30
		0xD8, 0xFF, // moveRectBottom -40
		// --- trailing chase block (AFTER the tail, per Flush ordering) ---
		0x01,                   // bChasing 1
		0x00,                   // hasTarget 0
		0x01,                   // bChasing2 1
		0x00,                   // bChasingHack 0
		0xF4, 0x01, 0x00, 0x00, // tChaseDuration 500
	}
	got := test.Encode(t, ctx, p.Encode, nil)
	if !bytes.Equal(got, want) {
		t.Fatalf("jms185 movement bytes:\n got % x\nwant % x", got, want)
	}

	// Round-trip the pinned bytes back through Decode to confirm the tail
	// lands in the right place on read too, not just write.
	var out MovementRequest
	test.RoundTrip(t, ctx, p.Encode, out.Decode, nil)
	if len(out.KeyPadStates()) != 3 {
		t.Errorf("expected 3 keypad entries, got %d", len(out.KeyPadStates()))
	}
	gl, gt, gr, gb := out.MoveRect()
	if gl != -10 || gt != 20 || gr != 30 || gb != -40 {
		t.Errorf("expected move rect -10,20,30,-40, got %d,%d,%d,%d", gl, gt, gr, gb)
	}
	if out.bChasing != 1 || out.tChaseDuration != 500 {
		t.Errorf("expected trailing chase block intact after tail, got bChasing=%d tChaseDuration=%d", out.bChasing, out.tChaseDuration)
	}
}
