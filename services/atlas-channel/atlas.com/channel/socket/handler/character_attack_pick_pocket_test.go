package handler

import (
	"atlas-channel/character/buff"
	"atlas-channel/character/buff/stat"
	"atlas-channel/data/skill/effect"
	"errors"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	charconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	skill3 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
)

func TestPickPocketWhitelisted(t *testing.T) {
	cases := []struct {
		name    string
		skillId uint32
		want    bool
	}{
		{"basic attack", 0, true},
		{"Double Stab", uint32(skill3.RogueDoubleStabId), true},
		{"Savage Blow", uint32(skill3.BanditSavageBlowId), true},
		{"Assaulter", uint32(skill3.ChiefBanditAssaulterId), true},
		{"Band of Thieves", uint32(skill3.ChiefBanditBandOfThievesId), true},
		{"Assassinate", uint32(skill3.ShadowerAssassinateId), true},
		{"Taunt", uint32(skill3.ShadowerTauntId), true},
		{"Boomerang Step", uint32(skill3.ShadowerBoomerangStepId), true},
		{"Meso Explosion not whitelisted", uint32(skill3.ChiefBanditMesoExplosionId), false},
		{"Pick Pocket itself not whitelisted", uint32(skill3.ChiefBanditPickpocketId), false},
		{"ranged skill not whitelisted", uint32(skill3.BowmasterHurricaneId), false},
		{"magic skill not whitelisted", uint32(skill3.MagicianMagicClawId), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := pickPocketWhitelisted(tc.skillId); got != tc.want {
				t.Fatalf("pickPocketWhitelisted(%d) = %v; want %v", tc.skillId, got, tc.want)
			}
		})
	}
}

func TestPickPocketMesoAmount(t *testing.T) {
	cases := []struct {
		name    string
		damage  uint32
		maxmeso int32
		want    uint32
	}{
		{"zero damage yields floor of 1", 0, 60, 1},
		{"exact maxmeso at 20000 damage", 20000, 60, 60},
		{"half maxmeso at 10000 damage", 10000, 60, 30},
		{"huge damage clamps to maxmeso", 2000000, 60, 60},
		{"zero maxmeso yields nothing", 10000, 0, 0},
		{"negative maxmeso yields nothing", 10000, -5, 0},
		{"product below 1 raised to floor (0.999)", 333, 60, 1},
		{"non-integer product truncates (25.5 -> 25)", 8500, 60, 25},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := pickPocketMesoAmount(tc.damage, tc.maxmeso); got != tc.want {
				t.Fatalf("pickPocketMesoAmount(%d, %d) = %d; want %d", tc.damage, tc.maxmeso, got, tc.want)
			}
		})
	}
}

// ppTestBuff builds an active (or expired) buff carrying a PICK_POCKET
// stat change of the given amount at the given level.
func ppTestBuff(level byte, amount int32, expired bool) buff.Model {
	expiresAt := time.Now().Add(time.Minute)
	if expired {
		expiresAt = time.Now().Add(-time.Minute)
	}
	return buff.NewBuff(
		int32(skill3.ChiefBanditPickpocketId),
		level,
		60000,
		[]stat.Model{stat.NewStat(string(charconst.TemporaryStatTypePickPocket), amount)},
		time.Now().Add(-time.Second),
		expiresAt,
	)
}

func ppTestEffect(t *testing.T, prop float64) effect.Model {
	t.Helper()
	se, err := effect.Extract(effect.RestModel{Prop: prop})
	if err != nil {
		t.Fatalf("effect.Extract: %v", err)
	}
	return se
}

func TestPickPocketResolveState_NonWhitelistedSkillMakesNoLookups(t *testing.T) {
	buffCalls := 0
	getBuffs := func(characterId uint32) ([]buff.Model, error) {
		buffCalls++
		return []buff.Model{ppTestBuff(15, 40, false)}, nil
	}
	getEffect := func(uniqueId uint32, level byte) (effect.Model, error) {
		t.Fatal("effect lookup must not run for a non-whitelisted skill")
		return effect.Model{}, nil
	}

	st := pickPocketResolveState(logrus.New(), getBuffs, getEffect, uint32(skill3.ChiefBanditMesoExplosionId), 1)

	if st.enabled {
		t.Fatal("state.enabled = true; want false for non-whitelisted skill")
	}
	if buffCalls != 0 {
		t.Fatalf("buff lookups = %d; want 0 for non-whitelisted skill", buffCalls)
	}
}

func TestPickPocketResolveState_BuffLookupErrorDisables(t *testing.T) {
	getBuffs := func(characterId uint32) ([]buff.Model, error) {
		return nil, errors.New("atlas-buffs unavailable")
	}
	getEffect := func(uniqueId uint32, level byte) (effect.Model, error) {
		t.Fatal("effect lookup must not run when the buff lookup fails")
		return effect.Model{}, nil
	}

	st := pickPocketResolveState(logrus.New(), getBuffs, getEffect, 0, 1)

	if st.enabled {
		t.Fatal("state.enabled = true; want false on buff lookup error")
	}
}

func TestPickPocketResolveState_NoPickPocketBuffDisables(t *testing.T) {
	other := buff.NewBuff(2311003, 10, 60000,
		[]stat.Model{stat.NewStat("HOLY_SYMBOL", 150)},
		time.Now(), time.Now().Add(time.Minute))
	getBuffs := func(characterId uint32) ([]buff.Model, error) {
		return []buff.Model{other}, nil
	}
	getEffect := func(uniqueId uint32, level byte) (effect.Model, error) {
		t.Fatal("effect lookup must not run without a PICK_POCKET buff")
		return effect.Model{}, nil
	}

	st := pickPocketResolveState(logrus.New(), getBuffs, getEffect, 0, 1)

	if st.enabled {
		t.Fatal("state.enabled = true; want false without a PICK_POCKET buff")
	}
}

func TestPickPocketResolveState_ExpiredBuffDisables(t *testing.T) {
	getBuffs := func(characterId uint32) ([]buff.Model, error) {
		return []buff.Model{ppTestBuff(15, 40, true)}, nil
	}
	getEffect := func(uniqueId uint32, level byte) (effect.Model, error) {
		t.Fatal("effect lookup must not run for an expired buff")
		return effect.Model{}, nil
	}

	st := pickPocketResolveState(logrus.New(), getBuffs, getEffect, 0, 1)

	if st.enabled {
		t.Fatal("state.enabled = true; want false for an expired buff")
	}
}

func TestPickPocketResolveState_NonPositiveMaxMesoDisables(t *testing.T) {
	getBuffs := func(characterId uint32) ([]buff.Model, error) {
		return []buff.Model{ppTestBuff(15, 0, false)}, nil
	}
	getEffect := func(uniqueId uint32, level byte) (effect.Model, error) {
		t.Fatal("effect lookup must not run when maxmeso <= 0")
		return effect.Model{}, nil
	}

	st := pickPocketResolveState(logrus.New(), getBuffs, getEffect, 0, 1)

	if st.enabled {
		t.Fatal("state.enabled = true; want false when maxmeso <= 0")
	}
}

func TestPickPocketResolveState_EffectLookupErrorDisables(t *testing.T) {
	getBuffs := func(characterId uint32) ([]buff.Model, error) {
		return []buff.Model{ppTestBuff(15, 40, false)}, nil
	}
	getEffect := func(uniqueId uint32, level byte) (effect.Model, error) {
		return effect.Model{}, errors.New("atlas-data unavailable")
	}

	st := pickPocketResolveState(logrus.New(), getBuffs, getEffect, 0, 1)

	if st.enabled {
		t.Fatal("state.enabled = true; want false on effect lookup error")
	}
}

func TestPickPocketResolveState_NonPositivePropDisables(t *testing.T) {
	getBuffs := func(characterId uint32) ([]buff.Model, error) {
		return []buff.Model{ppTestBuff(15, 40, false)}, nil
	}
	getEffect := func(uniqueId uint32, level byte) (effect.Model, error) {
		return ppTestEffect(t, 0), nil
	}

	st := pickPocketResolveState(logrus.New(), getBuffs, getEffect, 0, 1)

	if st.enabled {
		t.Fatal("state.enabled = true; want false when prop <= 0")
	}
}

func TestPickPocketResolveState_HappyPath(t *testing.T) {
	var gotUniqueId uint32
	var gotLevel byte
	getBuffs := func(characterId uint32) ([]buff.Model, error) {
		return []buff.Model{ppTestBuff(15, 40, false)}, nil
	}
	getEffect := func(uniqueId uint32, level byte) (effect.Model, error) {
		gotUniqueId = uniqueId
		gotLevel = level
		return ppTestEffect(t, 0.6), nil
	}

	st := pickPocketResolveState(logrus.New(), getBuffs, getEffect, uint32(skill3.ShadowerBoomerangStepId), 1)

	if !st.enabled {
		t.Fatal("state.enabled = false; want true on happy path")
	}
	if st.maxmeso != 40 {
		t.Fatalf("maxmeso = %d; want 40 (buff-captured stat amount)", st.maxmeso)
	}
	if st.prop != 0.6 {
		t.Fatalf("prop = %v; want 0.6", st.prop)
	}
	if gotUniqueId != uint32(skill3.ChiefBanditPickpocketId) {
		t.Fatalf("effect looked up skill %d; want %d (Pick Pocket)", gotUniqueId, uint32(skill3.ChiefBanditPickpocketId))
	}
	if gotLevel != 15 {
		t.Fatalf("effect looked up level %d; want 15 (buff-captured level)", gotLevel)
	}
}

// TestOnDamageApplied_CarriesPerLineDamages pins the reason the hook was
// widened to carry the DamageInfo: Pick Pocket rolls once per damage line,
// so the per-line breakdown (not just the summed total) must reach the hook.
func TestOnDamageApplied_CarriesPerLineDamages(t *testing.T) {
	ai := *packetmodel.NewAttackInfo(packetmodel.AttackTypeMelee)
	di := *packetmodel.NewDamageInfo(3).SetMonsterId(4101).SetDamages([]uint32{100, 250, 400})

	var gotMonsterId uint32
	var gotLines []uint32
	calls := 0
	deps := damageInfoEntryDeps{
		applyDamage: func(_ field.Model, _, _ uint32, _ []uint32, _ byte) error { return nil },
		onDamageApplied: func(di packetmodel.DamageInfo, _ uint32) {
			calls++
			gotMonsterId = di.MonsterId()
			gotLines = di.Damages()
		},
	}

	processDamageInfoEntry(discardLogger(), di, ai, effect.Model{}, 1, 999, 0, 0, testDrainField(), testTenant(t), "", deps)

	if calls != 1 {
		t.Fatalf("onDamageApplied calls = %d; want 1", calls)
	}
	if gotMonsterId != 4101 {
		t.Errorf("monsterId = %d; want 4101", gotMonsterId)
	}
	if len(gotLines) != 3 || gotLines[0] != 100 || gotLines[1] != 250 || gotLines[2] != 400 {
		t.Fatalf("per-line damages = %v; want [100 250 400]", gotLines)
	}
}
