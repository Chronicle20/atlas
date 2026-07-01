package serverbound

import (
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

// packet-audit:verify packet=character/serverbound/CharacterAttackMeleeRequest version=gms_v79 ida=0x8c22fd
func TestAttackMeleeRequestV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	m := AttackMeleeRequest{attackInfo: buildSampleAttack(model.AttackTypeMelee)}
	pt.RoundTrip(t, ctx, m.Encode, m.Decode, nil)
}

// packet-audit:verify packet=character/serverbound/CharacterAttackRangedRequest version=gms_v79 ida=0x8abbfc
func TestAttackRangedRequestV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	m := AttackRangedRequest{attackInfo: buildSampleAttack(model.AttackTypeRanged)}
	pt.RoundTrip(t, ctx, m.Encode, m.Decode, nil)
}

// packet-audit:verify packet=character/serverbound/CharacterAttackMagicRequest version=gms_v79 ida=0x8adb26
func TestAttackMagicRequestV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	m := AttackMagicRequest{attackInfo: buildSampleAttack(model.AttackTypeMagic)}
	pt.RoundTrip(t, ctx, m.Encode, m.Decode, nil)
}

// packet-audit:verify packet=character/serverbound/CharacterAttackTouchRequest version=gms_v79 ida=0x8b70d8
func TestAttackTouchRequestV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	m := AttackTouchRequest{attackInfo: buildSampleAttack(model.AttackTypeEnergy)}
	pt.RoundTrip(t, ctx, m.Encode, m.Decode, nil)
}
