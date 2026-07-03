package serverbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// buildSampleAttack mirrors model.sampleAttackInfo: a plain (skillId 0) attack so
// the keydown/charging/special-skill branches stay quiet and the wire structure is
// driven purely by attackType + tenant version.
func buildSampleAttack(at model.AttackType) model.AttackInfo {
	ai := model.NewAttackInfo(at)
	ai.SetHits(2)
	ai.SetDamage(1)
	ai.SetSkillId(0)
	ai.SetOption(0x10)
	ai.SetLeft(true)
	ai.SetAttackAction(0x05)
	ai.SetActionSpeed(4)
	di := model.NewDamageInfo(2)
	di.SetMonsterId(9001).SetHitAction(0x07).SetDamages([]uint32{1000, 2000})
	ai.AddDamageInfo(*di)
	if at == model.AttackTypeRanged {
		ai.SetBulletPosition(100, 200)
	}
	return *ai
}

// The four serverbound attack ops verify through their thin per-op wrappers, which
// delegate to the shared model.AttackInfo codec (production-tested in
// model/attack_info_test.go: round-trip across all types×versions + the v84 dr-block
// boundary). RoundTrip here pins that the wrapper delegates symmetrically per version.
//
// All four attacks are now verified across all five versions: the senders were named
// in every IDB (the v84/jms melee/ranged/magic senders were named this task) and the
// ops are routed in every tenant template. CLOSE_RANGE_ATTACK's registry-primary
// sender is CUserLocal::TryDoingNormalAttack.

// packet-audit:verify packet=character/serverbound/CharacterAttackMeleeRequest version=gms_v83 ida=0x95719b
// packet-audit:verify packet=character/serverbound/CharacterAttackMeleeRequest version=gms_v84 ida=0x989692
// packet-audit:verify packet=character/serverbound/CharacterAttackMeleeRequest version=gms_v87 ida=0x9d8efc
// packet-audit:verify packet=character/serverbound/CharacterAttackMeleeRequest version=gms_v95 ida=0x9123c0
// packet-audit:verify packet=character/serverbound/CharacterAttackMeleeRequest version=jms_v185 ida=0xa122be
func TestAttackMeleeRequest(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			m := AttackMeleeRequest{attackInfo: buildSampleAttack(model.AttackTypeMelee)}
			pt.RoundTrip(t, ctx, m.Encode, m.Decode, nil)
		})
	}
}

// packet-audit:verify packet=character/serverbound/CharacterAttackRangedRequest version=gms_v83 ida=0x9537d5
// packet-audit:verify packet=character/serverbound/CharacterAttackRangedRequest version=gms_v84 ida=0x98da5f
// packet-audit:verify packet=character/serverbound/CharacterAttackRangedRequest version=gms_v87 ida=0x9d1a9c
// packet-audit:verify packet=character/serverbound/CharacterAttackRangedRequest version=gms_v95 ida=0x925a00
// packet-audit:verify packet=character/serverbound/CharacterAttackRangedRequest version=jms_v185 ida=0xa19266
func TestAttackRangedRequest(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			m := AttackRangedRequest{attackInfo: buildSampleAttack(model.AttackTypeRanged)}
			pt.RoundTrip(t, ctx, m.Encode, m.Decode, nil)
		})
	}
}

// packet-audit:verify packet=character/serverbound/CharacterAttackMagicRequest version=gms_v83 ida=0x95571f
// packet-audit:verify packet=character/serverbound/CharacterAttackMagicRequest version=gms_v84 ida=0x99137f
// packet-audit:verify packet=character/serverbound/CharacterAttackMagicRequest version=gms_v87 ida=0x9d55a4
// packet-audit:verify packet=character/serverbound/CharacterAttackMagicRequest version=gms_v95 ida=0x92a240
// packet-audit:verify packet=character/serverbound/CharacterAttackMagicRequest version=jms_v185 ida=0xa1d280
func TestAttackMagicRequest(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			m := AttackMagicRequest{attackInfo: buildSampleAttack(model.AttackTypeMagic)}
			pt.RoundTrip(t, ctx, m.Encode, m.Decode, nil)
		})
	}
}

// packet-audit:verify packet=character/serverbound/CharacterAttackTouchRequest version=gms_v83 ida=0x95f135
// packet-audit:verify packet=character/serverbound/CharacterAttackTouchRequest version=gms_v84 ida=0x99d42a
// packet-audit:verify packet=character/serverbound/CharacterAttackTouchRequest version=gms_v87 ida=0x9e17dc
// packet-audit:verify packet=character/serverbound/CharacterAttackTouchRequest version=gms_v95 ida=0x930710
// packet-audit:verify packet=character/serverbound/CharacterAttackTouchRequest version=jms_v185 ida=0xa2ac53
func TestAttackTouchRequest(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			m := AttackTouchRequest{attackInfo: buildSampleAttack(model.AttackTypeEnergy)}
			pt.RoundTrip(t, ctx, m.Encode, m.Decode, nil)
		})
	}
}

// --- GMS v79 (legacy pre-83) ---
//
// The v79 client attack senders were IDA-verified (GMS_v79_1_DEVM.exe, port 13340).
// The AttackInfo base path (all the >=84 dr-block / >=95 gates are false at v79) plus
// the per-mob DamageInfo match the client Encode order field-for-field, with ONE
// legacy fix applied this task: the per-mob anti-hack CRC. All three multi-target
// senders — TryDoingMeleeAttack (Encode4 sub_640131 @0x8c2c57), TryDoingBodyAttack
// (@0x8b77d3) and TryDoingMagicAttack (@0x8af1c4) — write the CRC as the final
// per-target field, so model.DamageInfo now emits it for GMS >= 79 (was >= 83).
// v79 has no TryDoingNormalAttack; CLOSE_RANGE_ATTACK is sent by TryDoingMeleeAttack.
//
// These round-trips pin the wrapper->AttackInfo delegation on the v79 base path,
// matching the shared-model verification standard used for the other five versions.

// --- GMS v72 (legacy pre-79) ---
//
// v72 serverbound attack senders diverge from v79 in the shared AttackInfo head
// (IDA-verified against the v72 melee sender sub_85DDD2 @0x85ddd2, packet-build
// @0x85f8c0-0x85fbc6, and cross-checked against the v79 melee sender @0x8c22fd):
//   - the attack-action/direction field is a SINGLE byte (Encode1 @0x85f9c2,
//     (nAction&0x7F)|(bLeft<<7)) vs the v79 2-byte short (Encode2 @0x8c2adc);
//   - the head carries ONE skill-data CRC (Encode4 @0x85f96c) vs v79's two
//     (@0x8c2ab2 + @0x8c2abb);
//   - the per-mob DamageInfo (monsterId, 4 bytes, 5 shorts incl. delay, damages
//     loop over the outer hit-count, trailing mob CRC @0x85fb50) matches the Atlas
//     model.DamageInfo structure exactly, with the CRC present (gate lowered to
//     GMS>=72). All three legacy gates leave v79/v83/84/87/95/jms unchanged.
//
// TestAttackMeleeRequestBytesV72 pins the full v72 melee wire byte-for-byte.
// packet-audit:verify packet=character/serverbound/CharacterAttackMeleeRequest version=gms_v72 ida=0x85ddd2
func TestAttackMeleeRequestBytesV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	m := AttackMeleeRequest{attackInfo: buildSampleAttack(model.AttackTypeMelee)}
	got := pt.Encode(t, ctx, m.Encode, nil)
	want := []byte{
		0x00,                   // fieldKey                        @0x85f8de
		0x12,                   // (hits=2)|(damage=1<<4)          @0x85f8f2
		0x00, 0x00, 0x00, 0x00, // skillId=0                       @0x85f8fd
		0x00, 0x00, 0x00, 0x00, // skillDataCrc (ONE crc on v72)   @0x85f96c
		0x10,                   // mask1/option                    @0x85f9ac
		0x00,                   // mask2/action (1 BYTE on v72)    @0x85f9c2
		0x00,                   // attackActionType                @0x85f9d0
		0x04,                   // attackSpeed                     @0x85f9de
		0x00, 0x00, 0x00, 0x00, // attackTime                      @0x85f9e9
		// --- DamageInfo[0] (per-mob loop @0x85f9ee) ---
		0x29, 0x23, 0x00, 0x00, // monsterId=9001                  @0x85fa1c
		0x07,                   // hitAction                       @0x85fa2a
		0x00,                   // forceAction                     @0x85fa45
		0x00,                   // frameIdx                        @0x85fa53
		0x00,                   // calcDamageStatIndex             @0x85fab0
		0x00, 0x00, // hitPositionX                                @0x85fac6
		0x00, 0x00, // hitPositionY                                @0x85fadd
		0x00, 0x00, // previousPositionX                           @0x85faf3
		0x00, 0x00, // previousPositionY                           @0x85fb0a
		0x00, 0x00, // delay                                       @0x85fb19
		0xE8, 0x03, 0x00, 0x00, // damage[0]=1000 (loop @0x85fb37)
		0xD0, 0x07, 0x00, 0x00, // damage[1]=2000
		0x00, 0x00, 0x00, 0x00, // per-mob CRC                     @0x85fb50
		// --- trailer ---
		0x00, 0x00, // characterX                                  @0x85fb76
		0x00, 0x00, // characterY                                  @0x85fb8a
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v72 melee bytes:\n got=% X\nwant=% X", got, want)
	}
}

// packet-audit:verify packet=character/serverbound/CharacterAttackMeleeRequest version=gms_v79 ida=0x8c22fd
func TestAttackMeleeRequestV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	m := AttackMeleeRequest{attackInfo: buildSampleAttack(model.AttackTypeMelee)}
	pt.RoundTrip(t, ctx, m.Encode, m.Decode, nil)
}

// TestAttackRangedRequestBytesV72 pins the v72 ranged wire. Head/damage/CRC gates
// as melee; the ranged branch adds properBulletPosition(2)+cashBulletPosition(2)+
// nShootRange(1) after attackTime (v72 TryDoingShootAttack @0x8603fe: E2 @0x86216a,
// E2 @0x862178, E1 @0x862183; javlin false → no bulletItemId) and bulletX/bulletY
// after characterX/Y (E2 @0x86232e..0x86235e). Head single-crc @0x8620d6, 1-byte
// mask2 @0x862135.
// packet-audit:verify packet=character/serverbound/CharacterAttackRangedRequest version=gms_v72 ida=0x8603fe
func TestAttackRangedRequestBytesV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	m := AttackRangedRequest{attackInfo: buildSampleAttack(model.AttackTypeRanged)}
	got := pt.Encode(t, ctx, m.Encode, nil)
	want := []byte{
		0x00,                   // fieldKey
		0x12,                   // (hits=2)|(damage=1<<4)
		0x00, 0x00, 0x00, 0x00, // skillId=0
		0x00, 0x00, 0x00, 0x00, // skillDataCrc (ONE crc)
		0x10,                   // mask1
		0x00,                   // mask2 (1 BYTE)
		0x00,                   // attackActionType
		0x04,                   // attackSpeed
		0x00, 0x00, 0x00, 0x00, // attackTime
		0x00, 0x00, // properBulletPosition   @0x86216a
		0x00, 0x00, // cashBulletPosition     @0x862178
		0x00,       // nShootRange            @0x862183
		// --- DamageInfo[0] ---
		0x29, 0x23, 0x00, 0x00, // monsterId=9001
		0x07,                   // hitAction
		0x00, 0x00, 0x00,       // forceAction, frameIdx, calcDamageStatIndex
		0x00, 0x00, 0x00, 0x00, // hitPositionX, hitPositionY
		0x00, 0x00, 0x00, 0x00, // previousPositionX, previousPositionY
		0x00, 0x00,             // delay
		0xE8, 0x03, 0x00, 0x00, // damage[0]=1000
		0xD0, 0x07, 0x00, 0x00, // damage[1]=2000
		0x00, 0x00, 0x00, 0x00, // per-mob CRC
		// --- trailer ---
		0x00, 0x00, // characterX
		0x00, 0x00, // characterY
		0x64, 0x00, // bulletX=100            @0x86232e
		0xC8, 0x00, // bulletY=200            @0x862342
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v72 ranged bytes:\n got=% X\nwant=% X", got, want)
	}
}

// packet-audit:verify packet=character/serverbound/CharacterAttackRangedRequest version=gms_v79 ida=0x8abbfc
func TestAttackRangedRequestV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	m := AttackRangedRequest{attackInfo: buildSampleAttack(model.AttackTypeRanged)}
	pt.RoundTrip(t, ctx, m.Encode, m.Decode, nil)
}

// TestAttackMagicRequestBytesV72 pins the v72 magic wire. Head/damage/CRC gates as
// melee; v72 TryDoingMagicAttack @0x8625da writes characterX/Y then SendPacket with
// NO trailing Evan-dragon bool (Evan is v84+) — the codec's magic dragon block is
// gated off pre-79. Head single-crc @0x8639bb, 1-byte mask2 @0x863a07, characterX/Y
// @0x863be8/@0x863bff then immediate SendPacket @0x863c11.
// packet-audit:verify packet=character/serverbound/CharacterAttackMagicRequest version=gms_v72 ida=0x8625da
func TestAttackMagicRequestBytesV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	m := AttackMagicRequest{attackInfo: buildSampleAttack(model.AttackTypeMagic)}
	got := pt.Encode(t, ctx, m.Encode, nil)
	want := []byte{
		0x00,                   // fieldKey
		0x12,                   // (hits=2)|(damage=1<<4)
		0x00, 0x00, 0x00, 0x00, // skillId=0
		0x00, 0x00, 0x00, 0x00, // skillDataCrc (ONE crc)
		0x10,                   // mask1
		0x00,                   // mask2 (1 BYTE)
		0x00,                   // attackActionType
		0x04,                   // attackSpeed
		0x00, 0x00, 0x00, 0x00, // attackTime
		// --- DamageInfo[0] ---
		0x29, 0x23, 0x00, 0x00, // monsterId=9001
		0x07,                   // hitAction
		0x00, 0x00, 0x00,       // forceAction, frameIdx, calcDamageStatIndex
		0x00, 0x00, 0x00, 0x00, // hitPositionX, hitPositionY
		0x00, 0x00, 0x00, 0x00, // previousPositionX, previousPositionY
		0x00, 0x00,             // delay
		0xE8, 0x03, 0x00, 0x00, // damage[0]=1000
		0xD0, 0x07, 0x00, 0x00, // damage[1]=2000
		0x00, 0x00, 0x00, 0x00, // per-mob CRC
		// --- trailer (no dragon bool on v72) ---
		0x00, 0x00, // characterX
		0x00, 0x00, // characterY
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v72 magic bytes:\n got=% X\nwant=% X", got, want)
	}
}

// packet-audit:verify packet=character/serverbound/CharacterAttackMagicRequest version=gms_v79 ida=0x8adb26
func TestAttackMagicRequestV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	m := AttackMagicRequest{attackInfo: buildSampleAttack(model.AttackTypeMagic)}
	pt.RoundTrip(t, ctx, m.Encode, m.Decode, nil)
}

// TestAttackTouchRequestBytesV72 pins the v72 touch/body-attack wire. Energy type
// has no type-specific tail; v72 TryDoingBodyAttack @0x86b732 = head + damage +
// characterX/Y. Head single-crc @0x86bc4c, 1-byte mask2 @0x86bc64.
// packet-audit:verify packet=character/serverbound/CharacterAttackTouchRequest version=gms_v72 ida=0x86b732
func TestAttackTouchRequestBytesV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	m := AttackTouchRequest{attackInfo: buildSampleAttack(model.AttackTypeEnergy)}
	got := pt.Encode(t, ctx, m.Encode, nil)
	want := []byte{
		0x00,                   // fieldKey
		0x12,                   // (hits=2)|(damage=1<<4)
		0x00, 0x00, 0x00, 0x00, // skillId=0
		0x00, 0x00, 0x00, 0x00, // skillDataCrc (ONE crc)
		0x10,                   // mask1
		0x00,                   // mask2 (1 BYTE)
		0x00,                   // attackActionType
		0x04,                   // attackSpeed
		0x00, 0x00, 0x00, 0x00, // attackTime
		// --- DamageInfo[0] ---
		0x29, 0x23, 0x00, 0x00, // monsterId=9001
		0x07,                   // hitAction
		0x00, 0x00, 0x00,       // forceAction, frameIdx, calcDamageStatIndex
		0x00, 0x00, 0x00, 0x00, // hitPositionX, hitPositionY
		0x00, 0x00, 0x00, 0x00, // previousPositionX, previousPositionY
		0x00, 0x00,             // delay
		0xE8, 0x03, 0x00, 0x00, // damage[0]=1000
		0xD0, 0x07, 0x00, 0x00, // damage[1]=2000
		0x00, 0x00, 0x00, 0x00, // per-mob CRC
		// --- trailer ---
		0x00, 0x00, // characterX
		0x00, 0x00, // characterY
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v72 touch bytes:\n got=% X\nwant=% X", got, want)
	}
}

// packet-audit:verify packet=character/serverbound/CharacterAttackTouchRequest version=gms_v79 ida=0x8b70d8
func TestAttackTouchRequestV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	m := AttackTouchRequest{attackInfo: buildSampleAttack(model.AttackTypeEnergy)}
	pt.RoundTrip(t, ctx, m.Encode, m.Decode, nil)
}

// --- GMS v61 (very-legacy pre-72) ---
//
// v61 serverbound attack senders diverge from v72 in the shared AttackInfo head:
// the head carries NO skill-data CRC at all. IDA-verified against three of the
// four senders (GMS_v61.1_U_DEVM.exe, port 13338):
//   - CLOSE_RANGE  sub_7A45F1        (COutPacket 41, @0x7a5b7e)
//   - RANGED       TryDoingShootAttack (COutPacket 42, @0x7a7fc6)
//   - MAGIC        sub_7A8572        (COutPacket 43, @0x7a96db)
// In all three the Encode4(skillId) (@0x7a5bc3 / @0x7a8013 / @0x7a9721) is
// followed DIRECTLY by the mask1/option Encode1 (@0x7a5d3d / @0x7a80ad /
// @0x7a9753) — there is no head-CRC Encode4 in between (only a conditional
// keydown Encode4 for charge skills). v72 TryDoingMeleeAttack @0x85f96c writes
// exactly one head CRC, so model.legacyGmsNoSkillDataCrc drops it for GMS<72.
// The action/direction field is a SINGLE byte (v61 @0x7a5d56, (nAction&0x7F)|
// (bLeft<<7)) as on v72 (<79). The per-mob DamageInfo keeps the trailing anti-hack
// CRC (v61 sub_7A45F1 @0x7a5f14, Encode4 sub_5CF2AF), so model.DamageInfo's gate
// is >= 61. Type-specific tails (ranged properBullet/cashBullet/nShootRange +
// bulletX/Y, magic no-dragon) are the same <79/<95-gated legacy paths as v72.
//
// Each v61 fixture is the corresponding v72 wire MINUS the 4-byte head CRC.

// packet-audit:verify packet=character/serverbound/CharacterAttackMeleeRequest version=gms_v61 ida=0x7a45f1
func TestAttackMeleeRequestBytesV61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	m := AttackMeleeRequest{attackInfo: buildSampleAttack(model.AttackTypeMelee)}
	got := pt.Encode(t, ctx, m.Encode, nil)
	want := []byte{
		0x00,                   // fieldKey                        @0x7a5b9e
		0x12,                   // (hits=2)|(damage=1<<4)          @0x7a5bb5
		0x00, 0x00, 0x00, 0x00, // skillId=0                       @0x7a5bc3
		// NO head skill-data CRC (v61 < 72)
		0x10,                   // mask1/option                    @0x7a5d3d
		0x00,                   // mask2/action (1 BYTE)           @0x7a5d56
		0x00,                   // attackActionType                @0x7a5d67
		0x04,                   // attackSpeed                     @0x7a5d78
		0x00, 0x00, 0x00, 0x00, // attackTime                      @0x7a5d89
		// --- DamageInfo[0] (per-mob loop @0x7a5da3) ---
		0x29, 0x23, 0x00, 0x00, // monsterId=9001                  @0x7a5dbf
		0x07,                   // hitAction                       @0x7a5dd0
		0x00,                   // forceAction                     @0x7a5dee
		0x00,                   // frameIdx                        @0x7a5dff
		0x00,                   // calcDamageStatIndex             @0x7a5e5f
		0x00, 0x00, // hitPositionX                                @0x7a5e78
		0x00, 0x00, // hitPositionY                                @0x7a5e92
		0x00, 0x00, // previousPositionX                           @0x7a5eab
		0x00, 0x00, // previousPositionY                           @0x7a5ec5
		0x00, 0x00, // delay                                       @0x7a5ed7
		0xE8, 0x03, 0x00, 0x00, // damage[0]=1000 (loop @0x7a5ef8)
		0xD0, 0x07, 0x00, 0x00, // damage[1]=2000
		0x00, 0x00, 0x00, 0x00, // per-mob CRC                     @0x7a5f14
		// --- trailer ---
		0x00, 0x00, // characterX                                  @0x7a5f3d
		0x00, 0x00, // characterY                                  @0x7a5f54
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v61 melee bytes:\n got=% X\nwant=% X", got, want)
	}
}

// packet-audit:verify packet=character/serverbound/CharacterAttackRangedRequest version=gms_v61 ida=0x7a67e9
func TestAttackRangedRequestBytesV61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	m := AttackRangedRequest{attackInfo: buildSampleAttack(model.AttackTypeRanged)}
	got := pt.Encode(t, ctx, m.Encode, nil)
	want := []byte{
		0x00,                   // fieldKey
		0x12,                   // (hits=2)|(damage=1<<4)
		0x00, 0x00, 0x00, 0x00, // skillId=0
		// NO head skill-data CRC (v61 < 72)
		0x10,                   // mask1
		0x00,                   // mask2 (1 BYTE)
		0x00,                   // attackActionType
		0x04,                   // attackSpeed
		0x00, 0x00, 0x00, 0x00, // attackTime
		0x00, 0x00, // properBulletPosition
		0x00, 0x00, // cashBulletPosition
		0x00, // nShootRange
		// --- DamageInfo[0] ---
		0x29, 0x23, 0x00, 0x00, // monsterId=9001
		0x07,             // hitAction
		0x00, 0x00, 0x00, // forceAction, frameIdx, calcDamageStatIndex
		0x00, 0x00, 0x00, 0x00, // hitPositionX, hitPositionY
		0x00, 0x00, 0x00, 0x00, // previousPositionX, previousPositionY
		0x00, 0x00, // delay
		0xE8, 0x03, 0x00, 0x00, // damage[0]=1000
		0xD0, 0x07, 0x00, 0x00, // damage[1]=2000
		0x00, 0x00, 0x00, 0x00, // per-mob CRC
		// --- trailer ---
		0x00, 0x00, // characterX
		0x00, 0x00, // characterY
		0x64, 0x00, // bulletX=100
		0xC8, 0x00, // bulletY=200
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v61 ranged bytes:\n got=% X\nwant=% X", got, want)
	}
}

// packet-audit:verify packet=character/serverbound/CharacterAttackMagicRequest version=gms_v61 ida=0x7a8572
func TestAttackMagicRequestBytesV61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	m := AttackMagicRequest{attackInfo: buildSampleAttack(model.AttackTypeMagic)}
	got := pt.Encode(t, ctx, m.Encode, nil)
	want := []byte{
		0x00,                   // fieldKey
		0x12,                   // (hits=2)|(damage=1<<4)
		0x00, 0x00, 0x00, 0x00, // skillId=0
		// NO head skill-data CRC (v61 < 72)
		0x10,                   // mask1
		0x00,                   // mask2 (1 BYTE)
		0x00,                   // attackActionType
		0x04,                   // attackSpeed
		0x00, 0x00, 0x00, 0x00, // attackTime
		// --- DamageInfo[0] ---
		0x29, 0x23, 0x00, 0x00, // monsterId=9001
		0x07,             // hitAction
		0x00, 0x00, 0x00, // forceAction, frameIdx, calcDamageStatIndex
		0x00, 0x00, 0x00, 0x00, // hitPositionX, hitPositionY
		0x00, 0x00, 0x00, 0x00, // previousPositionX, previousPositionY
		0x00, 0x00, // delay
		0xE8, 0x03, 0x00, 0x00, // damage[0]=1000
		0xD0, 0x07, 0x00, 0x00, // damage[1]=2000
		0x00, 0x00, 0x00, 0x00, // per-mob CRC
		// --- trailer (no dragon bool on v61, Evan is v84+) ---
		0x00, 0x00, // characterX
		0x00, 0x00, // characterY
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v61 magic bytes:\n got=% X\nwant=% X", got, want)
	}
}

// packet-audit:verify packet=character/serverbound/CharacterAttackTouchRequest version=gms_v61 ida=0x7b084b
func TestAttackTouchRequestBytesV61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	m := AttackTouchRequest{attackInfo: buildSampleAttack(model.AttackTypeEnergy)}
	got := pt.Encode(t, ctx, m.Encode, nil)
	want := []byte{
		0x00,                   // fieldKey
		0x12,                   // (hits=2)|(damage=1<<4)
		0x00, 0x00, 0x00, 0x00, // skillId=0
		// NO head skill-data CRC (v61 < 72)
		0x10,                   // mask1
		0x00,                   // mask2 (1 BYTE)
		0x00,                   // attackActionType
		0x04,                   // attackSpeed
		0x00, 0x00, 0x00, 0x00, // attackTime
		// --- DamageInfo[0] ---
		0x29, 0x23, 0x00, 0x00, // monsterId=9001
		0x07,             // hitAction
		0x00, 0x00, 0x00, // forceAction, frameIdx, calcDamageStatIndex
		0x00, 0x00, 0x00, 0x00, // hitPositionX, hitPositionY
		0x00, 0x00, 0x00, 0x00, // previousPositionX, previousPositionY
		0x00, 0x00, // delay
		0xE8, 0x03, 0x00, 0x00, // damage[0]=1000
		0xD0, 0x07, 0x00, 0x00, // damage[1]=2000
		0x00, 0x00, 0x00, 0x00, // per-mob CRC
		// --- trailer ---
		0x00, 0x00, // characterX
		0x00, 0x00, // characterY
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v61 touch bytes:\n got=% X\nwant=% X", got, want)
	}
}

// --- GMS v48 (very-legacy pre-61) ---
//
// v48 serverbound attack senders diverge from v61 in ONE field: the per-mob
// DamageInfo carries NO trailing anti-hack CRC. IDA-verified against all three
// senders (GMS_v48_1_DEVM.exe, port 13337):
//   - CLOSE_RANGE  sub_6A0528  (COutPacket 36, @0x6a1711): per-mob loop @0x6a17b5
//     Encode4(monsterId)@0x6a17e0 .. Encode2(delay)@0x6a18f8, damages loop
//     Encode4@0x6a1910, then v106+=152 and loops — NO Encode4(crc) before the next
//     mob or the characterX/Y trailer @0x6a193e/0x6a1952.
//   - RANGED       sub_6A228C  (COutPacket 37, @0x6a36bd): same per-mob shape with
//     no CRC; head properBullet/cashBullet/nShootRange @0x6a37a4/0x6a37b2/0x6a37bd;
//     trailer is characterX/Y @0x6a3965/0x6a3979 then SendPacket @0x6a3988 — NO
//     bulletX/bulletY (the v61 shoot sender @0x7a67e9 also ends at characterX/Y; the
//     4-byte bullet trailer is gated off for GMS<61 via legacyGmsNoRangedBulletCoords).
//   - MAGIC        sub_6A3AC7  (COutPacket 38, @0x6a4af8): per-mob loop with no CRC;
//     trailer characterX/Y @0x6a4d5e/0x6a4d75 then SendPacket @0x6a4d87 (no dragon,
//     Evan is v84+; legacyGmsByteAction already gates it <79).
// The head is byte-identical to v61 (v48 < 72 → legacyGmsNoSkillDataCrc drops both
// skill-data CRCs; v48 < 79 → single-byte action). Each v48 fixture is the v61 wire
// MINUS the 4-byte per-mob CRC (model.DamageInfo gates the mob CRC to GMS >= 61).

// packet-audit:verify packet=character/serverbound/CharacterAttackMeleeRequest version=gms_v48 ida=0x6a0528
func TestAttackMeleeRequestBytesV48(t *testing.T) {
	ctx := pt.CreateContext("GMS", 48, 1)
	m := AttackMeleeRequest{attackInfo: buildSampleAttack(model.AttackTypeMelee)}
	got := pt.Encode(t, ctx, m.Encode, nil)
	want := []byte{
		0x00,                   // fieldKey                        @0x6a172e
		0x12,                   // (hits=2)|(damage=1<<4)          @0x6a1742
		0x00, 0x00, 0x00, 0x00, // skillId=0                       @0x6a174d
		// NO head skill-data CRC (v48 < 72)
		0x10,                   // mask1/option                    @0x6a176a
		0x00,                   // mask2/action (1 BYTE)           @0x6a1780
		0x00,                   // attackActionType                @0x6a1791
		0x04,                   // attackSpeed                     @0x6a179f
		0x00, 0x00, 0x00, 0x00, // attackTime                      @0x6a17ad
		// --- DamageInfo[0] (per-mob loop @0x6a17b5) ---
		0x29, 0x23, 0x00, 0x00, // monsterId=9001                  @0x6a17e0
		0x07,                   // hitAction                       @0x6a17ee
		0x00,                   // forceAction                     @0x6a1809
		0x00,                   // frameIdx                        @0x6a1817
		0x00,                   // calcDamageStatIndex             @0x6a188b
		0x00, 0x00, // hitPositionX                                @0x6a18a2
		0x00, 0x00, // hitPositionY                                @0x6a18ba
		0x00, 0x00, // previousPositionX                           @0x6a18d1
		0x00, 0x00, // previousPositionY                           @0x6a18e9
		0x00, 0x00, // delay                                       @0x6a18f8
		0xE8, 0x03, 0x00, 0x00, // damage[0]=1000 (loop @0x6a1910)
		0xD0, 0x07, 0x00, 0x00, // damage[1]=2000
		// NO per-mob CRC (v48 < 61)
		// --- trailer ---
		0x00, 0x00, // characterX                                  @0x6a193e
		0x00, 0x00, // characterY                                  @0x6a1952
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v48 melee bytes:\n got=% X\nwant=% X", got, want)
	}
}

// packet-audit:verify packet=character/serverbound/CharacterAttackRangedRequest version=gms_v48 ida=0x6a228c
func TestAttackRangedRequestBytesV48(t *testing.T) {
	ctx := pt.CreateContext("GMS", 48, 1)
	m := AttackRangedRequest{attackInfo: buildSampleAttack(model.AttackTypeRanged)}
	got := pt.Encode(t, ctx, m.Encode, nil)
	want := []byte{
		0x00,                   // fieldKey                        @0x6a36da
		0x12,                   // (hits=2)|(damage=1<<4)          @0x6a36ee
		0x00, 0x00, 0x00, 0x00, // skillId=0                       @0x6a36f7
		// NO head skill-data CRC (v48 < 72)
		0x10,                   // mask1                           @0x6a3753
		0x00,                   // mask2 (1 BYTE)                  @0x6a3769
		0x00,                   // attackActionType                @0x6a377a
		0x04,                   // attackSpeed                     @0x6a3788
		0x00, 0x00, 0x00, 0x00, // attackTime                      @0x6a3796
		0x00, 0x00, // properBulletPosition                        @0x6a37a4
		0x00, 0x00, // cashBulletPosition                          @0x6a37b2
		0x00, // nShootRange                                       @0x6a37bd
		// --- DamageInfo[0] ---
		0x29, 0x23, 0x00, 0x00, // monsterId=9001
		0x07,             // hitAction
		0x00, 0x00, 0x00, // forceAction, frameIdx, calcDamageStatIndex
		0x00, 0x00, 0x00, 0x00, // hitPositionX, hitPositionY
		0x00, 0x00, 0x00, 0x00, // previousPositionX, previousPositionY
		0x00, 0x00, // delay
		0xE8, 0x03, 0x00, 0x00, // damage[0]=1000
		0xD0, 0x07, 0x00, 0x00, // damage[1]=2000
		// NO per-mob CRC (v48 < 61)
		// --- trailer (characterX/Y only; NO bulletX/Y on v48, @0x6a3965/0x6a3979) ---
		0x00, 0x00, // characterX
		0x00, 0x00, // characterY
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v48 ranged bytes:\n got=% X\nwant=% X", got, want)
	}
}

// packet-audit:verify packet=character/serverbound/CharacterAttackMagicRequest version=gms_v48 ida=0x6a3ac7
func TestAttackMagicRequestBytesV48(t *testing.T) {
	ctx := pt.CreateContext("GMS", 48, 1)
	m := AttackMagicRequest{attackInfo: buildSampleAttack(model.AttackTypeMagic)}
	got := pt.Encode(t, ctx, m.Encode, nil)
	want := []byte{
		0x00,                   // fieldKey                        @0x6a4b18
		0x12,                   // (hits=2)|(damage=1<<4)          @0x6a4b2f
		0x00, 0x00, 0x00, 0x00, // skillId=0                       @0x6a4b3e
		// NO head skill-data CRC (v48 < 72)
		0x10,                   // mask1                           @0x6a4b71 (client sends 0; codec writes model option)
		0x00,                   // mask2 (1 BYTE)                  @0x6a4b8a
		0x00,                   // attackActionType                @0x6a4b9e
		0x04,                   // attackSpeed                     @0x6a4bb3
		0x00, 0x00, 0x00, 0x00, // attackTime                      @0x6a4bc4
		// --- DamageInfo[0] ---
		0x29, 0x23, 0x00, 0x00, // monsterId=9001
		0x07,             // hitAction
		0x00, 0x00, 0x00, // forceAction, frameIdx, calcDamageStatIndex
		0x00, 0x00, 0x00, 0x00, // hitPositionX, hitPositionY
		0x00, 0x00, 0x00, 0x00, // previousPositionX, previousPositionY
		0x00, 0x00, // delay
		0xE8, 0x03, 0x00, 0x00, // damage[0]=1000
		0xD0, 0x07, 0x00, 0x00, // damage[1]=2000
		// NO per-mob CRC (v48 < 61)
		// --- trailer (no dragon on v48, Evan is v84+) ---
		0x00, 0x00, // characterX                                  @0x6a4d5e
		0x00, 0x00, // characterY                                  @0x6a4d75
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v48 magic bytes:\n got=% X\nwant=% X", got, want)
	}
}
