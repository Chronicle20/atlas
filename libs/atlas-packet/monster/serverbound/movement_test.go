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
// packet-audit:verify packet=monster/serverbound/MonsterMovementRequest version=jms_v185 ida=0x6e8892
// packet-audit:verify packet=monster/serverbound/MonsterMovementRequest version=gms_v84 ida=0x6818c3
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
		0x01, // dwFlag 1 (Encode1 @0x63a803)
		0xFD, // nActionAndDir -3 (Encode1 @0x63a80e)
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

func TestMonsterMovementOperationString(t *testing.T) {
	p := MovementRequest{}
	if p.Operation() != MonsterMovementHandle {
		t.Errorf("expected operation %s, got %s", MonsterMovementHandle, p.Operation())
	}
	if p.String() == "" {
		t.Error("expected non-empty string")
	}
}
