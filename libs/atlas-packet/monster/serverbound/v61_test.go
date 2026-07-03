package serverbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// v61 monster serverbound fixtures. Every send-site is byte-identical to the
// verified GMS v72 anchor: v61 is GMS major 61 (<79), so the movement request
// OMITS the v79+ flyCtxTargetX/Y (and the v84+ multiTarget/randTime, v87+ CRC/
// chase). Body-verified from each COutPacket() send site in GMS_v61.1_U_DEVM.exe
// @port 13338:
//
//	CMob::GenerateMovePath sub_5CADA5@0x5cada5 — COutPacket(155) @0x5cb2aa then
//	  Encode4 mobId, Encode2 moveId, Encode1 dwFlag, Encode1 nActionAndDir,
//	  Encode4 skillData, Encode1 moveFlags, Encode4 hackedCode, CMovePath::Flush.
//	CMob::SendDropPickUpRequest sub_5CD6D2@0x5cd6d2 — COutPacket(157) @0x5cd74e +
//	  Encode4 mobCrc, Encode4 dropId.
//	CMob::TryFirstSelfDestruction sub_5CD3FD@0x5cd3fd — COutPacket(160) @0x5cd525 +
//	  Encode4 mobId.
//	CMob::SetDamagedByMob sub_5CED89@0x5ced89 — COutPacket(161) @0x5cefaa +
//	  Encode4 attacker, Encode4 characterId, Encode4 mobId, Encode1 attackIdx,
//	  Encode4 damage, Encode1 reflect, Encode2 x, Encode2 y.
//	CMobPool::OnMobCrcKeyChanged @0x5d4d23 — COutPacket(136) @0x5d4d7c + SendPacket
//	  with zero Encode* (empty reply).

// TestMonsterMovementBytesV61 pins the v61 MOVE_LIFE (op 155) wire = v72:
// seven scalar fields (flyCtx OMITTED, <79) + empty CMovePath blob.
// packet-audit:verify packet=monster/serverbound/MonsterMovementRequest version=gms_v61 ida=0x5cada5
func TestMonsterMovementBytesV61(t *testing.T) {
	p := MovementRequest{}
	p.uniqueId = 1001
	p.moveId = 55
	p.dwFlag = 1
	p.nActionAndDir = -3
	p.skillData = 0x0305
	p.moveFlags = 0
	p.hackedCode = 0
	// v79+ fields set but gated off at v61:
	p.flyCtxTargetX = 100
	p.flyCtxTargetY = 200
	p.hackedCodeCRC = 999
	p.tChaseDuration = 500

	ctx := test.CreateContext("GMS", 61, 1)
	want := []byte{
		0xE9, 0x03, 0x00, 0x00, // uniqueId 1001 (Encode4)
		0x37, 0x00, // moveId 55 (Encode2)
		0x01, // dwFlag 1 (Encode1)
		0xFD, // nActionAndDir -3 (Encode1)
		0x05, 0x03, 0x00, 0x00, // skillData 0x0305 (Encode4)
		0x00,                   // moveFlags 0 (Encode1)
		0x00, 0x00, 0x00, 0x00, // hackedCode 0 (Encode4)
		// flyCtxTargetX/Y OMITTED (v79+)
		// opaque movement (empty): StartX int16, StartY int16, count byte
		0x00, 0x00, 0x00, 0x00, 0x00,
	}
	got := test.Encode(t, ctx, p.Encode, nil)
	if !bytes.Equal(got, want) {
		t.Errorf("v61 movement bytes:\n got % x\nwant % x", got, want)
	}
}

// TestMobDropPickupRequestBytesV61 pins the v61 MOB_DROP_PICKUP_REQUEST (op 157)
// wire = v72: Encode4 mobCrc + Encode4 dropId.
// packet-audit:verify packet=monster/serverbound/MonsterMobDropPickupRequest version=gms_v61 ida=0x5cd6d2
func TestMobDropPickupRequestBytesV61(t *testing.T) {
	input := MobDropPickupRequest{mobCrc: 0xAABBCCDD, dropId: 0x01020304}
	ctx := test.CreateContext("GMS", 61, 1)
	want := []byte{
		0xDD, 0xCC, 0xBB, 0xAA, // mobCrc uint32 LE (Encode4 @0x5cd76f)
		0x04, 0x03, 0x02, 0x01, // dropId uint32 LE (Encode4 @0x5cd778)
	}
	got := input.Encode(nil, ctx)(nil)
	if !bytes.Equal(got, want) {
		t.Errorf("v61 mobDropPickupRequest bytes:\n got % x\nwant % x", got, want)
	}
}

// TestMonsterBombBytesV61 pins the v61 MONSTER_BOMB (op 160) wire = v72: a single
// Encode4 of the self-destruct mobId.
// packet-audit:verify packet=monster/serverbound/MonsterMonsterBomb version=gms_v61 ida=0x5cd3fd
func TestMonsterBombBytesV61(t *testing.T) {
	input := MonsterBomb{mobId: 0xAABBCCDD}
	ctx := test.CreateContext("GMS", 61, 1)
	want := []byte{
		0xDD, 0xCC, 0xBB, 0xAA, // mobId uint32 LE (Encode4 @0x5cd546)
	}
	got := input.Encode(nil, ctx)(nil)
	if !bytes.Equal(got, want) {
		t.Errorf("v61 monsterBomb bytes:\n got % x\nwant % x", got, want)
	}
}

// TestMobDamageMobBytesV61 pins the v61 MOB_DAMAGE_MOB (op 161) wire = v72:
// 3xEncode4 + Encode1 + Encode4 + Encode1 + 2xEncode2.
// packet-audit:verify packet=monster/serverbound/MonsterMobDamageMob version=gms_v61 ida=0x5ced89
func TestMobDamageMobBytesV61(t *testing.T) {
	input := MobDamageMob{
		attackerMobId: 0x11223344,
		characterId:   0x0010F447,
		mobId:         0xAABBCCDD,
		attackIndex:   0x03,
		damage:        0x000003E7,
		reflect:       0x01,
		x:             0x0102,
		y:             0x0304,
	}
	ctx := test.CreateContext("GMS", 61, 1)
	want := []byte{
		0x44, 0x33, 0x22, 0x11, // attackerMobId uint32 LE (Encode4 @0x5cefce)
		0x47, 0xF4, 0x10, 0x00, // characterId uint32 LE (Encode4 @0x5cefe1)
		0xDD, 0xCC, 0xBB, 0xAA, // mobId uint32 LE (Encode4 @0x5ceffe)
		0x03,                   // attackIndex byte (Encode1 @0x5cf009)
		0xE7, 0x03, 0x00, 0x00, // damage uint32 LE (Encode4 @0x5cf014)
		0x01,       // reflect byte (Encode1 @0x5cf024)
		0x02, 0x01, // x uint16 LE (Encode2 @0x5cf02d)
		0x04, 0x03, // y uint16 LE (Encode2 @0x5cf038)
	}
	got := input.Encode(nil, ctx)(nil)
	if !bytes.Equal(got, want) {
		t.Errorf("v61 mobDamageMob bytes:\n got % x\nwant % x", got, want)
	}
}

// TestMobCrcKeyChangedReplyBytesV61 pins the v61 MOB_CRC_KEY_CHANGED_REPLY (op
// 136) wire: EMPTY payload. CMobPool::OnMobCrcKeyChanged @0x5d4d23 reads the
// clientbound crcKey (Decode4 @0x5d4d3b) then builds COutPacket(136) @0x5d4d7c
// and SendPacket()s with zero Encode* calls.
// packet-audit:verify packet=monster/serverbound/MonsterMobCrcKeyChangedReply version=gms_v61 ida=0x5d4d23
func TestMobCrcKeyChangedReplyBytesV61(t *testing.T) {
	input := MobCrcKeyChangedReply{}
	ctx := test.CreateContext("GMS", 61, 1)
	got := input.Encode(nil, ctx)(nil)
	if !bytes.Equal(got, []byte{}) {
		t.Errorf("v61 mobCrcKeyChangedReply bytes: got % x, want empty", got)
	}
}

// FIELD_DAMAGE_MOB (op158) — CMob::Update field-damage send site sub_5C71B7
// @0x5c78b5: COutPacket(158); Encode4(SecureFuse(m_dwMobID)) @0x5c78d9;
// Encode4(nFieldDamage v155) @0x5c78e4; SendPacket. Two Encode4, no version gate
// (identical layout to the verified v72 anchor; v61 op158 = v72 op181 − 23).
//
// packet-audit:verify packet=monster/serverbound/MonsterFieldDamageMob version=gms_v61 ida=0x5c71b7
func TestFieldDamageMobV61(t *testing.T) {
	ctx := test.CreateContext("GMS", 61, 1)
	input := FieldDamageMob{mobCrc: 0xAABBCCDD, damage: 0x000003E7}
	got := input.Encode(nil, ctx)(nil)
	want := []byte{
		0xDD, 0xCC, 0xBB, 0xAA, // mobCrc uint32 LE (Encode4 @0x5c78d9)
		0xE7, 0x03, 0x00, 0x00, // damage uint32 LE = 999 (Encode4 @0x5c78e4)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v61 FieldDamageMob layout mismatch\n got % x\nwant % x", got, want)
	}
}
