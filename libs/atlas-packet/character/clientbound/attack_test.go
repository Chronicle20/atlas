package clientbound

import (
	"bytes"
	"context"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// --- Attack round-trip tests ---

// packet-audit:verify packet=character/clientbound/Attack version=gms_v83 ida=0x9803ab
// packet-audit:verify packet=character/clientbound/Attack version=gms_v87 ida=0xa05a50
// packet-audit:verify packet=character/clientbound/Attack version=gms_v95 ida=0x95a670
// packet-audit:verify packet=character/clientbound/Attack version=gms_v84 ida=0x9c0572
// packet-audit:verify packet=character/clientbound/Attack version=jms_v185 ida=0xa537ef
func TestAttackMeleeNoSkillRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)

			ai := model.NewAttackInfo(model.AttackTypeMelee)
			ai.SetDamage(2).SetHits(3).SetOption(0x10).SetLeft(true).SetAttackAction(0x05).SetActionSpeed(4)
			di := model.NewDamageInfo(3)
			di.SetMonsterId(9001).SetHitAction(0x07).SetDamages([]uint32{1000, 2000, 3000})
			ai.AddDamageInfo(*di)
			di2 := model.NewDamageInfo(3)
			di2.SetMonsterId(9002).SetHitAction(0x08).SetDamages([]uint32{4000, 5000, 6000})
			ai.AddDamageInfo(*di2)

			input := NewAttackMelee(12345, 50, 0, 15, 2070000, false, false, *ai)
			output := NewAttackForDecode(CharacterAttackMeleeWriter, 0, false, false, false)
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)

			assertAttack(t, ctx, input, output)
		})
	}
}

func TestAttackMeleeWithSkillRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)

			ai := model.NewAttackInfo(model.AttackTypeMelee)
			ai.SetDamage(1).SetHits(2).SetSkillId(1001004).SetOption(0x08).SetLeft(false).SetAttackAction(0x10).SetActionSpeed(5)
			di := model.NewDamageInfo(2)
			di.SetMonsterId(8001).SetHitAction(0x03).SetDamages([]uint32{7777, 8888})
			ai.AddDamageInfo(*di)

			input := NewAttackMelee(54321, 70, 10, 20, 0, false, false, *ai)
			output := NewAttackForDecode(CharacterAttackMeleeWriter, 1001004, false, false, false)
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)

			assertAttack(t, ctx, input, output)
		})
	}
}

func TestAttackRangedWithSkillRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)

			ai := model.NewAttackInfo(model.AttackTypeRanged)
			ai.SetDamage(1).SetHits(1).SetSkillId(3001004).SetOption(0x04).SetLeft(true).SetAttackAction(0x08).SetActionSpeed(3)
			ai.SetBulletPosition(100, 200)
			di := model.NewDamageInfo(1)
			di.SetMonsterId(5001).SetHitAction(0x01).SetDamages([]uint32{9999})
			ai.AddDamageInfo(*di)

			input := NewAttackRanged(11111, 80, 15, 25, 2070006, false, false, *ai)
			output := NewAttackForDecode(CharacterAttackRangedWriter, 3001004, false, false, false)
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)

			assertAttack(t, ctx, input, output)
			assertBulletPosition(t, input, output)
		})
	}
}

func TestAttackMeleeWithMesoExplosionRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)

			ai := model.NewAttackInfo(model.AttackTypeMelee)
			// With meso explosion, damage count per target varies independently of hits.
			ai.SetDamage(2).SetHits(1).SetSkillId(4211006).SetOption(0x02).SetLeft(false).SetAttackAction(0x0A).SetActionSpeed(6)
			di := model.NewDamageInfo(1)
			di.SetMonsterId(7001).SetHitAction(0x05).SetDamages([]uint32{11111, 22222, 33333})
			ai.AddDamageInfo(*di)
			di2 := model.NewDamageInfo(1)
			di2.SetMonsterId(7002).SetHitAction(0x06).SetDamages([]uint32{44444})
			ai.AddDamageInfo(*di2)

			input := NewAttackMelee(22222, 90, 20, 30, 0, true, false, *ai)
			output := NewAttackForDecode(CharacterAttackMeleeWriter, 4211006, false, true, false)
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)

			assertAttack(t, ctx, input, output)
			// Verify damage counts per target are correct with meso explosion.
			outAI := output.AttackInfo()
			inAI := input.AttackInfo()
			outDI := outAI.DamageInfo()
			inDI := inAI.DamageInfo()
			for i := range inDI {
				if len(outDI[i].Damages()) != len(inDI[i].Damages()) {
					t.Errorf("target %d damage count: got %d, want %d", i, len(outDI[i].Damages()), len(inDI[i].Damages()))
				}
			}
		})
	}
}

func TestAttackMeleeWithKeydownRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)

			ai := model.NewAttackInfo(model.AttackTypeMelee)
			ai.SetDamage(1).SetHits(1).SetSkillId(5001002).SetOption(0x01).SetLeft(true).SetAttackAction(0x03).SetActionSpeed(2)
			ai.SetKeydown(500)
			di := model.NewDamageInfo(1)
			di.SetMonsterId(6001).SetHitAction(0x02).SetDamages([]uint32{55555})
			ai.AddDamageInfo(*di)

			input := NewAttackMelee(33333, 100, 20, 10, 0, false, true, *ai)
			output := NewAttackForDecode(CharacterAttackMeleeWriter, 5001002, false, false, true)
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)

			assertAttack(t, ctx, input, output)
			assertKeydown(t, input, output)
		})
	}
}

func TestAttackRangedStrafeGMS95RoundTrip(t *testing.T) {
	// Strafe passive SLV byte only appears on GMS v95+.
	ctx := pt.CreateContext("GMS", 95, 1)

	ai := model.NewAttackInfo(model.AttackTypeRanged)
	ai.SetDamage(1).SetHits(4).SetSkillId(3111006).SetOption(0x00).SetLeft(false).SetAttackAction(0x20).SetActionSpeed(4)
	ai.SetBulletPosition(300, 400)
	di := model.NewDamageInfo(4)
	di.SetMonsterId(4001).SetHitAction(0x09).SetDamages([]uint32{1111, 2222, 3333, 4444})
	ai.AddDamageInfo(*di)

	input := NewAttackRanged(44444, 120, 20, 15, 2060000, true, false, *ai)
	output := NewAttackForDecode(CharacterAttackRangedWriter, 3111006, true, false, false)
	pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)

	assertAttack(t, ctx, input, output)
}

func TestAttackMeleeNoTargetsRoundTrip(t *testing.T) {
	// Test with attackAction <= 0x110 but zero damage targets.
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)

			ai := model.NewAttackInfo(model.AttackTypeMelee)
			ai.SetDamage(0).SetHits(0).SetOption(0x00).SetLeft(false).SetAttackAction(0x05).SetActionSpeed(4)

			input := NewAttackMelee(99999, 30, 0, 10, 0, false, false, *ai)
			output := NewAttackForDecode(CharacterAttackMeleeWriter, 0, false, false, false)
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)

			assertAttack(t, ctx, input, output)
		})
	}
}

// attackV79MeleeBody pins the GMS v79 CLOSE_RANGE_ATTACK (172) wire byte-for-byte
// against the live client reader CUserRemote::OnAttack@0x8d66a1 (GMS_v79_1_DEVM.exe,
// port 13340). The four attack ops (CLOSE_RANGE 172 / RANGED 173 / MAGIC 174 /
// ENERGY 175) all funnel through this one reader, so this fixture covers the shared
// character/clientbound/Attack struct for all four cells.
//
// Read order (each byte traced to a Decode call in OnAttack):
//
//	Decode1 @0x8d66bd  packed = (nMob<<4)|nDamagePerMob      -> 0x23  (damage 2, hits 3)
//	Decode1 @0x8d66d2  skillLevel byte (gates skillId)       -> 0x00  (no skill; no Decode4)
//	  --- NO character-level byte here on v79 (v83+ inserts *(this+10976)=Decode1) ---
//	Decode1 @0x8d6733  mask1/option (client keeps &0x20)      -> 0x10
//	Decode2 @0x8d67a4  mask2 = (bLeft<<15)|nAction            -> 0x8005 (left, action 5)
//	Decode1 @0x8d67cb  nActionSpeed                           -> 0x04
//	Decode1 @0x8d67d8  nMastery                               -> 0x0F
//	Decode4 @0x8d67e0  nBulletItemID                          -> 2070000
//	per target (loop nMob=2, @0x8d680c):
//	  Decode4 @0x8d6811 monsterOid; if !=0:
//	    Decode1 @0x8d6822 hitAction; (non-meso) loop nDamagePerMob(3) Decode4 @0x8d68f7 damages
//	a2==173? (RANGED only) -> no bulletX/Y for melee
//	keydown skill? skillId 0 -> none
// packet-audit:verify packet=character/clientbound/Attack version=gms_v79 ida=0x8d66a1
var attackV79MeleeBody = []byte{
	0x39, 0x30, 0x00, 0x00, // characterId 12345
	0x23,                   // packed (nMob 2 << 4) | hits 3
	0x00,                   // skillLevel byte = 0 (no skill) -- NO level byte on v79
	0x10,                   // mask1/option
	0x05, 0x80,             // mask2 = (1<<15)|0x05
	0x04,                   // actionSpeed
	0x0F,                   // mastery
	0xF0, 0x95, 0x1F, 0x00, // bulletItemId 2070000 (=0x1F95F0)
	0x29, 0x23, 0x00, 0x00, // target0 monsterId 9001
	0x07,                   // target0 hitAction
	0xE8, 0x03, 0x00, 0x00, // 1000
	0xD0, 0x07, 0x00, 0x00, // 2000
	0xB8, 0x0B, 0x00, 0x00, // 3000
	0x2A, 0x23, 0x00, 0x00, // target1 monsterId 9002
	0x08,                   // target1 hitAction
	0xA0, 0x0F, 0x00, 0x00, // 4000
	0x88, 0x13, 0x00, 0x00, // 5000
	0x70, 0x17, 0x00, 0x00, // 6000
}

func TestAttackBytesV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)

	ai := model.NewAttackInfo(model.AttackTypeMelee)
	ai.SetDamage(2).SetHits(3).SetOption(0x10).SetLeft(true).SetAttackAction(0x05).SetActionSpeed(4)
	di := model.NewDamageInfo(3)
	di.SetMonsterId(9001).SetHitAction(0x07).SetDamages([]uint32{1000, 2000, 3000})
	ai.AddDamageInfo(*di)
	di2 := model.NewDamageInfo(3)
	di2.SetMonsterId(9002).SetHitAction(0x08).SetDamages([]uint32{4000, 5000, 6000})
	ai.AddDamageInfo(*di2)

	// level=50 is supplied but MUST NOT appear on the v79 wire (gated >=83).
	in := NewAttackMelee(12345, 50, 0, 15, 2070000, false, false, *ai)
	got := pt.Encode(t, ctx, in.Encode, nil)
	if !bytes.Equal(got, attackV79MeleeBody) {
		t.Fatalf("v79 bytes = % X, want % X", got, attackV79MeleeBody)
	}
}

func assertAttack(t *testing.T, ctx context.Context, input Attack, output Attack) {
	t.Helper()
	if output.CharacterId() != input.CharacterId() {
		t.Errorf("characterId: got %v, want %v", output.CharacterId(), input.CharacterId())
	}
	// The character-level byte only rides the wire on GMS v83+ / JMS (see
	// Attack.Encode). On the legacy pre-83 client it is absent, so it does not
	// round-trip and must not be asserted there.
	te := tenant.MustFromContext(ctx)
	if te.MajorVersion() >= 83 && output.Level() != input.Level() {
		t.Errorf("level: got %v, want %v", output.Level(), input.Level())
	}
	if output.SkillLevel() != input.SkillLevel() {
		t.Errorf("skillLevel: got %v, want %v", output.SkillLevel(), input.SkillLevel())
	}
	if output.Mastery() != input.Mastery() {
		t.Errorf("mastery: got %v, want %v", output.Mastery(), input.Mastery())
	}
	if output.BulletItemId() != input.BulletItemId() {
		t.Errorf("bulletItemId: got %v, want %v", output.BulletItemId(), input.BulletItemId())
	}

	outAI := output.AttackInfo()
	inAI := input.AttackInfo()
	if outAI.Damage() != inAI.Damage() {
		t.Errorf("ai.damage: got %v, want %v", outAI.Damage(), inAI.Damage())
	}
	if outAI.Hits() != inAI.Hits() {
		t.Errorf("ai.hits: got %v, want %v", outAI.Hits(), inAI.Hits())
	}
	if outAI.Option() != inAI.Option() {
		t.Errorf("ai.option: got %v, want %v", outAI.Option(), inAI.Option())
	}
	if outAI.Left() != inAI.Left() {
		t.Errorf("ai.left: got %v, want %v", outAI.Left(), inAI.Left())
	}
	if outAI.AttackAction() != inAI.AttackAction() {
		t.Errorf("ai.attackAction: got %v, want %v", outAI.AttackAction(), inAI.AttackAction())
	}
	if outAI.ActionSpeed() != inAI.ActionSpeed() {
		t.Errorf("ai.actionSpeed: got %v, want %v", outAI.ActionSpeed(), inAI.ActionSpeed())
	}

	outDI := outAI.DamageInfo()
	inDI := inAI.DamageInfo()
	if len(outDI) != len(inDI) {
		t.Fatalf("damageInfo count: got %d, want %d", len(outDI), len(inDI))
	}
	for i := range inDI {
		if outDI[i].MonsterId() != inDI[i].MonsterId() {
			t.Errorf("target %d monsterId: got %v, want %v", i, outDI[i].MonsterId(), inDI[i].MonsterId())
		}
		if outDI[i].HitAction() != inDI[i].HitAction() {
			t.Errorf("target %d hitAction: got %v, want %v", i, outDI[i].HitAction(), inDI[i].HitAction())
		}
		if len(outDI[i].Damages()) != len(inDI[i].Damages()) {
			t.Errorf("target %d damages count: got %d, want %d", i, len(outDI[i].Damages()), len(inDI[i].Damages()))
		} else {
			for j := range inDI[i].Damages() {
				if outDI[i].Damages()[j] != inDI[i].Damages()[j] {
					t.Errorf("target %d damage %d: got %v, want %v", i, j, outDI[i].Damages()[j], inDI[i].Damages()[j])
				}
			}
		}
	}
}

func assertBulletPosition(t *testing.T, input Attack, output Attack) {
	t.Helper()
	outAI := output.AttackInfo()
	inAI := input.AttackInfo()
	if outAI.BulletX() != inAI.BulletX() {
		t.Errorf("bulletX: got %v, want %v", outAI.BulletX(), inAI.BulletX())
	}
	if outAI.BulletY() != inAI.BulletY() {
		t.Errorf("bulletY: got %v, want %v", outAI.BulletY(), inAI.BulletY())
	}
}

func assertKeydown(t *testing.T, input Attack, output Attack) {
	t.Helper()
	outAI := output.AttackInfo()
	inAI := input.AttackInfo()
	if outAI.Keydown() != inAI.Keydown() {
		t.Errorf("keydown: got %v, want %v", outAI.Keydown(), inAI.Keydown())
	}
}

// --- EffectSkillUse round-trip tests ---

func TestEffectSkillUseNoFlagsRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewEffectSkillUse(1, 1001004, 50, 10, false, false, false, false, false, false)
			output := NewEffectSkillUseForDecode(false, false, false)
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			assertEffectSkillUse(t, input, output)
		})
	}
}

func TestEffectSkillUseBerserkRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewEffectSkillUse(1, 1320006, 120, 30, true, true, false, false, false, false)
			output := NewEffectSkillUseForDecode(true, false, false)
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			assertEffectSkillUse(t, input, output)
			if output.BerserkDarkForce() != input.BerserkDarkForce() {
				t.Errorf("berserkDarkForce: got %v, want %v", output.BerserkDarkForce(), input.BerserkDarkForce())
			}
		})
	}
}

func TestEffectSkillUseAllFlagsRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewEffectSkillUse(1, 2221006, 200, 20, true, true, true, true, true, true)
			output := NewEffectSkillUseForDecode(true, true, true)
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			assertEffectSkillUse(t, input, output)
			if output.BerserkDarkForce() != input.BerserkDarkForce() {
				t.Errorf("berserkDarkForce: got %v, want %v", output.BerserkDarkForce(), input.BerserkDarkForce())
			}
			if output.DragonFuryCreate() != input.DragonFuryCreate() {
				t.Errorf("dragonFuryCreate: got %v, want %v", output.DragonFuryCreate(), input.DragonFuryCreate())
			}
			if output.MonsterMagnetLeft() != input.MonsterMagnetLeft() {
				t.Errorf("monsterMagnetLeft: got %v, want %v", output.MonsterMagnetLeft(), input.MonsterMagnetLeft())
			}
		})
	}
}

func assertEffectSkillUse(t *testing.T, input EffectSkillUse, output EffectSkillUse) {
	t.Helper()
	if output.Mode() != input.Mode() {
		t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
	}
	if output.SkillId() != input.SkillId() {
		t.Errorf("skillId: got %v, want %v", output.SkillId(), input.SkillId())
	}
	if output.CharacterLevel() != input.CharacterLevel() {
		t.Errorf("characterLevel: got %v, want %v", output.CharacterLevel(), input.CharacterLevel())
	}
	if output.SkillLevel() != input.SkillLevel() {
		t.Errorf("skillLevel: got %v, want %v", output.SkillLevel(), input.SkillLevel())
	}
}

// --- EffectSkillUseForeign round-trip tests ---

func TestEffectSkillUseForeignNoFlagsRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewEffectSkillUseForeign(12345, 1, 1001004, 50, 10, false, false, false, false, false, false)
			output := NewEffectSkillUseForeignForDecode(false, false, false)
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			assertEffectSkillUseForeign(t, input, output)
		})
	}
}

func TestEffectSkillUseForeignAllFlagsRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewEffectSkillUseForeign(67890, 1, 2221006, 200, 20, true, false, true, true, true, true)
			output := NewEffectSkillUseForeignForDecode(true, true, true)
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			assertEffectSkillUseForeign(t, input, output)
			if output.BerserkDarkForce() != input.BerserkDarkForce() {
				t.Errorf("berserkDarkForce: got %v, want %v", output.BerserkDarkForce(), input.BerserkDarkForce())
			}
			if output.DragonFuryCreate() != input.DragonFuryCreate() {
				t.Errorf("dragonFuryCreate: got %v, want %v", output.DragonFuryCreate(), input.DragonFuryCreate())
			}
			if output.MonsterMagnetLeft() != input.MonsterMagnetLeft() {
				t.Errorf("monsterMagnetLeft: got %v, want %v", output.MonsterMagnetLeft(), input.MonsterMagnetLeft())
			}
		})
	}
}

func assertEffectSkillUseForeign(t *testing.T, input EffectSkillUseForeign, output EffectSkillUseForeign) {
	t.Helper()
	if output.CharacterId() != input.CharacterId() {
		t.Errorf("characterId: got %v, want %v", output.CharacterId(), input.CharacterId())
	}
	if output.Mode() != input.Mode() {
		t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
	}
	if output.SkillId() != input.SkillId() {
		t.Errorf("skillId: got %v, want %v", output.SkillId(), input.SkillId())
	}
	if output.CharacterLevel() != input.CharacterLevel() {
		t.Errorf("characterLevel: got %v, want %v", output.CharacterLevel(), input.CharacterLevel())
	}
	if output.SkillLevel() != input.SkillLevel() {
		t.Errorf("skillLevel: got %v, want %v", output.SkillLevel(), input.SkillLevel())
	}
}
