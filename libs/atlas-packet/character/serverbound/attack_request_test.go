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
// Markers are pinned where the op is implemented AND routed by the tenant template
// AND its sender resolves in the IDB:
//   - v83/v87/v95 route all four attacks -> melee/ranged/magic/touch verified.
//   - v84 routes all four but only TryDoingBodyAttack is named in the v84 IDB -> touch only.
//   - jms routes all four (added this task) but only TryDoingBodyAttack is named in the
//     jms-DEVM IDB -> touch only; jms melee/ranged/magic stay ❌ (senders unnamed there).
// CLOSE_RANGE_ATTACK's registry-primary sender is CUserLocal::TryDoingNormalAttack.

// packet-audit:verify packet=character/serverbound/CharacterAttackMeleeRequest version=gms_v83 ida=0x95719b
// packet-audit:verify packet=character/serverbound/CharacterAttackMeleeRequest version=gms_v87 ida=0x9d8efc
// packet-audit:verify packet=character/serverbound/CharacterAttackMeleeRequest version=gms_v95 ida=0x9123c0
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
// packet-audit:verify packet=character/serverbound/CharacterAttackRangedRequest version=gms_v87 ida=0x9d1a9c
// packet-audit:verify packet=character/serverbound/CharacterAttackRangedRequest version=gms_v95 ida=0x925a00
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
// packet-audit:verify packet=character/serverbound/CharacterAttackMagicRequest version=gms_v87 ida=0x9d55a4
// packet-audit:verify packet=character/serverbound/CharacterAttackMagicRequest version=gms_v95 ida=0x92a240
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
