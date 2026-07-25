package clientbound

import (
	"bytes"
	"testing"
	"time"

	testlog "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/monster"
	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// v61 monster clientbound fixtures. Every monster writer is byte-identical to
// the verified GMS v72 wire: v61 is GMS major 61 (>12, <79, <87), so it takes
// the same legacy branches as v72 — the control byte is present (MajorVersion()>12),
// the mob temp-stat mask is the LEGACY bare 4-byte word (model.go legacyMobStatMask,
// <79), and the blob omits the v87+ phase. No monster codec gate crosses the
// 61/72 boundary, so the read orders coincide. Live-verified against
// GMS_v61.1_U_DEVM.exe @port 13338:
//
//	CMobPool::OnMobEnterField @0x5d49c4 — Decode4 uniqueId, Decode1 control,
//	  Decode4 monsterId, CMob::SetTemporaryStat @0x5ce6ae (single Decode4 mask) + Init.
//	CMob::OnMove @0x5cb45f — Decode1, Decode1 (only two leading bytes, no
//	  bNextAttackPossible), Decode4 skill word, CMovePath::OnMovePacket.
//	CMob::OnStatSet (unnamed sub_5CB8ED) @0x5cb8ed — Decode4 mask,
//	  MobStat::DecodeTemporary, Decode2 tDelay, Decode1 calcDamageStatIndex.

// TestMonsterSpawnBytesV61 pins the v61 spawn wire = v72 against
// CMobPool::OnMobEnterField @0x5d49c4.
// packet-audit:verify packet=monster/clientbound/MonsterSpawn version=gms_v61 ida=0x5d49c4
func TestMonsterSpawnBytesV61(t *testing.T) {
	m := model.NewMonster(100, 200, 5, 300, model.MonsterAppearTypeRegen, 0)
	input := NewMonsterSpawn(5001, true, 100100, m)
	ctx := test.CreateContext("GMS", 61, 1)
	want := []byte{
		0x89, 0x13, 0x00, 0x00, // uniqueId 5001 — Decode4
		0x01,                   // control byte (controlled) — Decode1
		0x04, 0x87, 0x01, 0x00, // monsterId 100100 (0x18704) — Decode4
		0x00, 0x00, 0x00, 0x00, // temp-stat mask (empty, LEGACY 4-byte) — SetTemporaryStat Decode4
		0x64, 0x00, // x 100
		0xC8, 0x00, // y 200
		0x05,       // moveAction 5
		0x00, 0x00, // foothold 0
		0x2C, 0x01, // homeFoothold 300
		0xFE,                   // appearType -2 (Regen)
		0x00,                   // team 0
		0x00, 0x00, 0x00, 0x00, // effectItemId 0
		// phase omitted (<87)
	}
	got := input.Encode(nil, ctx)(nil)
	if !bytes.Equal(got, want) {
		t.Errorf("v61 spawn bytes:\n got % x\nwant % x", got, want)
	}
}

// TestMonsterDestroyBytesV61 pins the v61 destroy wire = v72 against
// CMobPool::OnMobLeaveField @0x5d4b87 (Decode4 uniqueId + Decode1 destroyType).
// packet-audit:verify packet=monster/clientbound/MonsterDestroy version=gms_v61 ida=0x5d4b87
func TestMonsterDestroyBytesV61(t *testing.T) {
	input := NewMonsterDestroy(5001, DestroyTypeFadeOut)
	ctx := test.CreateContext("GMS", 61, 1)
	want := []byte{
		0x89, 0x13, 0x00, 0x00, // uniqueId 5001
		0x01, // destroyType FadeOut
	}
	got := input.Encode(nil, ctx)(nil)
	if !bytes.Equal(got, want) {
		t.Errorf("v61 destroy bytes:\n got % x\nwant % x", got, want)
	}
}

// TestMonsterControlBytesV61 pins the v61 control wire = v72 against
// CMobPool::OnMobChangeController @0x5d4cdc (controlType, uniqueId, aggro, then
// the shared monster blob with the legacy 4-byte mask).
// packet-audit:verify packet=monster/clientbound/MonsterControl version=gms_v61 ida=0x5d4cdc
func TestMonsterControlBytesV61(t *testing.T) {
	m := model.NewMonster(100, 200, 5, 300, model.MonsterAppearTypeRegen, 0)
	input := NewMonsterControl(ControlTypeActiveRequest, 5001, 100100, m, true)
	ctx := test.CreateContext("GMS", 61, 1)
	want := []byte{
		0x02,                   // controlType ActiveRequest
		0x89, 0x13, 0x00, 0x00, // uniqueId 5001
		0x01,                   // aggro true
		0x04, 0x87, 0x01, 0x00, // monsterId 100100
		0x00, 0x00, 0x00, 0x00, // temp-stat mask (empty, LEGACY 4-byte)
		0x64, 0x00, // x 100
		0xC8, 0x00, // y 200
		0x05,       // moveAction 5
		0x00, 0x00, // foothold 0
		0x2C, 0x01, // homeFoothold 300
		0xFE,                   // appearType -2 (Regen)
		0x00,                   // team 0
		0x00, 0x00, 0x00, 0x00, // effectItemId 0
	}
	got := input.Encode(nil, ctx)(nil)
	if !bytes.Equal(got, want) {
		t.Errorf("v61 control bytes:\n got % x\nwant % x", got, want)
	}
}

// TestMonsterMovementBytesV61 pins the v61 movement wire = v72 against
// CMob::OnMove @0x5cb45f: two leading bytes (bNotForceLandingWhenDiscard,
// action|left) — bNextAttackPossible OMITTED (<79) — skill word, then the
// CMovePath blob.
// packet-audit:verify packet=monster/clientbound/MonsterMovement version=gms_v61 ida=0x5cb45f
func TestMonsterMovementBytesV61(t *testing.T) {
	input := NewMonsterMovement(5001, true, true, true, 1, 100, 5, model.MultiTargetForBall{}, model.RandTimeForAreaAttack{}, model.Movement{})
	ctx := test.CreateContext("GMS", 61, 1)
	want := []byte{
		0x89, 0x13, 0x00, 0x00, // uniqueId 5001 — pool Decode4
		0x01,       // bNotForceLandingWhenDiscard
		0x01,       // bLeft 1 (action|left byte; bNextAttackPossible OMITTED)
		0x64, 0x00, // skillId 100 int16
		0x05, 0x00, // skillLevel 5 int16
		0x00, 0x00, // movement StartX
		0x00, 0x00, // movement StartY
		0x00, // movement element count = 0
	}
	got := input.Encode(nil, ctx)(nil)
	if !bytes.Equal(got, want) {
		t.Errorf("v61 movement bytes:\n got % x\nwant % x", got, want)
	}
}

// TestMonsterMovementAckBytesV61 pins the v61 wire = v72 against CMob::OnCtrlAck
// @0x5cb827 (Decode2 moveId, Decode1 useSkills, Decode2 mp, Decode1 skillId,
// Decode1 skillLevel).
// packet-audit:verify packet=monster/clientbound/MonsterMovementAck version=gms_v61 ida=0x5cb827
func TestMonsterMovementAckBytesV61(t *testing.T) {
	input := NewMonsterMovementAck(5001, 42, 300, true, 10, 3)
	ctx := test.CreateContext("GMS", 61, 1)
	want := []byte{
		0x89, 0x13, 0x00, 0x00, // uniqueId 5001
		0x2A, 0x00, // moveId 42
		0x01,       // useSkills true
		0x2C, 0x01, // mp 300
		0x0A, // skillId 10
		0x03, // skillLevel 3
	}
	got := input.Encode(nil, ctx)(nil)
	if !bytes.Equal(got, want) {
		t.Errorf("v61 movement-ack bytes:\n got % x\nwant % x", got, want)
	}
}

// TestMonsterStatSetBytesV61 pins the v61 wire = v72 against CMob::OnStatSet
// (unnamed sub_5CB8ED) @0x5cb8ed: Decode4 LEGACY 4-byte mask, DecodeTemporary
// body, Decode2 tDelay, Decode1 calcDamageStatIndex.
// packet-audit:verify packet=monster/clientbound/MonsterStatSet version=gms_v61 ida=0x5cb8ed
func TestMonsterStatSetBytesV61(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 61, 1)
	tn := tenant.MustFromContext(ctx)

	stat := model.NewMonsterTemporaryStat()
	stat.AddStat(l)(tn)(string(monster.TemporaryStatTypeSpeed), monster.SkillTypeSlow, 1, -40, time.Time{})
	input := NewMonsterStatSet(5001, stat)

	want := []byte{
		0x89, 0x13, 0x00, 0x00, // mobId 5001 — pool Decode4
		0x40, 0x00, 0x00, 0x00, // LEGACY 4-byte mask — Speed bit (shift 6)
		0xD8, 0xFF, // value -40 (opaque body)
		0x7E, 0x00, // sourceId 126
		0x01, 0x00, // sourceLevel 1
		0xFF, 0xFF, // expiry sentinel -1
		0x00, 0x00, // tDelay — Decode2
		0x00, // m_nCalcDamageStatIndex — Decode1
		0x00, // bStat (movement-affecting; absorbed in opaque body)
	}
	got := input.Encode(l, ctx)(nil)
	if !bytes.Equal(got, want) {
		t.Errorf("v61 statset bytes:\n got % x\nwant % x", got, want)
	}
}

// TestMonsterStatResetBytesV61 pins the v61 wire = v72 against CMob::OnStatReset
// @0x5cb9f5 (Decode4 LEGACY 4-byte mask, reset body, Decode1 calcDamageStatIndex).
// packet-audit:verify packet=monster/clientbound/MonsterStatReset version=gms_v61 ida=0x5cb9f5
func TestMonsterStatResetBytesV61(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 61, 1)
	tn := tenant.MustFromContext(ctx)

	stat := model.NewMonsterTemporaryStat()
	stat.AddStat(l)(tn)(string(monster.TemporaryStatTypeSpeed), monster.SkillTypeSlow, 1, -40, time.Time{})
	input := NewMonsterStatReset(5001, stat)

	want := []byte{
		0x89, 0x13, 0x00, 0x00, // mobId 5001 — pool Decode4
		0x40, 0x00, 0x00, 0x00, // LEGACY 4-byte mask — Speed bit (shift 6)
		0xD8, 0xFF, // value -40 (opaque reset body)
		0x7E, 0x00, // sourceId 126
		0x01, 0x00, // sourceLevel 1
		0xFF, 0xFF, // expiry sentinel -1
		0x00, 0x00, // tDelay (absorbed in opaque reset body)
		0x00, // m_nCalcDamageStatIndex — Decode1
		0x00, // bStat (movement-affecting; absorbed)
	}
	got := input.Encode(l, ctx)(nil)
	if !bytes.Equal(got, want) {
		t.Errorf("v61 statreset bytes:\n got % x\nwant % x", got, want)
	}
}

// TestMonsterDamageBytesV61 pins the v61 wire = v72 against CMob::OnDamaged
// @0x5cbc82 (Decode1 damageType, Decode4 damage, Decode4 hp, Decode4 maxHp).
// packet-audit:verify packet=monster/clientbound/MonsterDamage version=gms_v61 ida=0x5cbc82
func TestMonsterDamageBytesV61(t *testing.T) {
	input := NewMonsterDamage(5001, MonsterDamageTypeUnk2, 1500, 8500, 10000)
	ctx := test.CreateContext("GMS", 61, 1)
	want := []byte{
		0x89, 0x13, 0x00, 0x00, // uniqueId 5001 — pool Decode4
		0x01,                   // damageType Unk2
		0xDC, 0x05, 0x00, 0x00, // damage 1500
		0x34, 0x21, 0x00, 0x00, // hp 8500
		0x10, 0x27, 0x00, 0x00, // maxHp 10000
	}
	got := input.Encode(nil, ctx)(nil)
	if !bytes.Equal(got, want) {
		t.Errorf("v61 damage bytes:\n got % x\nwant % x", got, want)
	}
}

// TestMonsterSpecialEffectBySkillBytesV61 pins the v61 wire = v72 against
// CMob::OnSpecialEffectBySkill @0x5cc6e3 (Decode4 skillId; no characterId/delay).
// packet-audit:verify packet=monster/clientbound/MonsterMonsterSpecialEffectBySkill version=gms_v61 ida=0x5cc6e3
func TestMonsterSpecialEffectBySkillBytesV61(t *testing.T) {
	input := NewMonsterSpecialEffectBySkill(0x07654321, 0x002F1801, 0x0000A1B2, 0x0190)
	ctx := test.CreateContext("GMS", 61, 1)
	want := []byte{
		0x21, 0x43, 0x65, 0x07, // uniqueId int32 LE (pool Decode4)
		0x01, 0x18, 0x2F, 0x00, // skillId int32 LE = 0x002F1801
	}
	got := input.Encode(nil, ctx)(nil)
	if !bytes.Equal(got, want) {
		t.Errorf("v61 specialEffectBySkill bytes:\n got % x\nwant % x", got, want)
	}
}

// TestMobCrcKeyChangedBytesV61 pins the v61 wire = v72 against the pool-level
// CMobPool::OnMobCrcKeyChanged @0x5d4d23 (Decode4 crcKey; NO uniqueId prefix).
// packet-audit:verify packet=monster/clientbound/MonsterMobCrcKeyChanged version=gms_v61 ida=0x5d4d23
func TestMobCrcKeyChangedBytesV61(t *testing.T) {
	input := NewMobCrcKeyChanged(0x12345678)
	ctx := test.CreateContext("GMS", 61, 1)
	want := []byte{
		0x78, 0x56, 0x34, 0x12, // crcKey uint32 LE = 0x12345678
	}
	got := input.Encode(nil, ctx)(nil)
	if !bytes.Equal(got, want) {
		t.Errorf("v61 mobCrcKeyChanged bytes:\n got % x\nwant % x", got, want)
	}
}

// TestMonsterHealthBytesV61 pins the v61 wire = v72 against CMob::OnHPIndicator
// @0x5cc480 (Decode1 hpPercent).
// packet-audit:verify packet=monster/clientbound/MonsterHealth version=gms_v61 ida=0x5cc480
func TestMonsterHealthBytesV61(t *testing.T) {
	input := NewMonsterHealth(5001, 85)
	ctx := test.CreateContext("GMS", 61, 1)
	want := []byte{
		0x89, 0x13, 0x00, 0x00, // uniqueId 5001 — pool Decode4
		0x55, // hpPercent 85
	}
	got := input.Encode(nil, ctx)(nil)
	if !bytes.Equal(got, want) {
		t.Errorf("v61 health bytes:\n got % x\nwant % x", got, want)
	}
}

// TestCatchMonsterBytesV61 pins the v61 wire = v72 against CMob::OnCatchEffect
// @0x5cc506 (uniqueId prefix + Decode1 result byte; NO success byte, v95CatchLayout
// is false).
// packet-audit:verify packet=monster/clientbound/MonsterCatchMonster version=gms_v61 ida=0x5cc506
func TestCatchMonsterBytesV61(t *testing.T) {
	input := NewCatchMonster(0x07654321, 0x42, 0x01)
	ctx := test.CreateContext("GMS", 61, 1)
	want := []byte{
		0x21, 0x43, 0x65, 0x07, // uniqueId int32 LE (pool Decode4)
		0x42, // result byte
	}
	got := input.Encode(nil, ctx)(nil)
	if !bytes.Equal(got, want) {
		t.Errorf("v61 catchMonster bytes:\n got % x\nwant % x", got, want)
	}
}
