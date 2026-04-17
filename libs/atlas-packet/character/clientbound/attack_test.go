package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// --- Attack round-trip tests ---

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

			assertAttack(t, input, output)
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

			assertAttack(t, input, output)
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

			assertAttack(t, input, output)
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

			assertAttack(t, input, output)
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

			assertAttack(t, input, output)
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

	assertAttack(t, input, output)
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

			assertAttack(t, input, output)
		})
	}
}

func assertAttack(t *testing.T, input Attack, output Attack) {
	t.Helper()
	if output.CharacterId() != input.CharacterId() {
		t.Errorf("characterId: got %v, want %v", output.CharacterId(), input.CharacterId())
	}
	if output.Level() != input.Level() {
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
