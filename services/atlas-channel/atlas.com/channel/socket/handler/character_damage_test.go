package handler

import (
	"atlas-channel/character"
	"atlas-channel/character/buff"
	"atlas-channel/character/buff/stat"
	"atlas-channel/data/skill/effect"
	"atlas-channel/monster"
	"context"
	"testing"
	"time"

	skill2 "atlas-channel/character/skill"
	monsterdata "atlas-channel/data/monster"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"

	charconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	skillconst "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type emissions struct {
	hp       []int16
	mp       []int16
	meso     []int32
	reflects []uint32
}

func fakeDeps(em *emissions, buffs []buff.Model, skills []skill2.Model, eff effect.Model, mob monster.Model, tmpl monsterdata.Model) damageMitigationDeps {
	return damageMitigationDeps{
		getBuffs:           func(uint32) ([]buff.Model, error) { return buffs, nil },
		getSkills:          func(uint32) ([]skill2.Model, error) { return skills, nil },
		getEffect:          func(uint32, byte) (effect.Model, error) { return eff, nil },
		getMonster:         func(uint32) (monster.Model, error) { return mob, nil },
		getMonsterTemplate: func(uint32) (monsterdata.Model, error) { return tmpl, nil },
		changeHP: func(_ field.Model, _ uint32, amount int16) error {
			em.hp = append(em.hp, amount)
			return nil
		},
		changeMP: func(_ field.Model, _ uint32, amount int16) error {
			em.mp = append(em.mp, amount)
			return nil
		},
		requestChangeMeso: func(_ field.Model, _ uint32, _ uint32, _ string, amount int32) error {
			em.meso = append(em.meso, amount)
			return nil
		},
		damageMonster: func(_ field.Model, _ uint32, _ uint32, damages []uint32, _ byte) error {
			em.reflects = append(em.reflects, damages...)
			return nil
		},
	}
}

func activeBuff(statType charconst.TemporaryStatType, amount int32) buff.Model {
	future := time.Now().Add(time.Hour)
	return buff.NewBuff(2001002, 20, 3600, []stat.Model{stat.NewStat(string(statType), amount)}, time.Now(), future)
}

func testTenantModel(t *testing.T, region string, major uint16) tenant.Model {
	t.Helper()
	m, err := tenant.Create(uuid.New(), region, major, 1)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func testCharacter(t *testing.T, jobId job.Id, mp uint16, meso uint32, skills []skill2.Model) character.Model {
	t.Helper()
	c, err := character.NewModelBuilder().
		SetId(42).
		SetJobId(jobId).
		SetHp(1000).SetMaxHp(2000).
		SetMp(mp).SetMaxMp(2000).
		SetMeso(meso).
		SetSkills(skills).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// damagePacket builds a decoded-equivalent DamageTakenInfo via the codec:
// struct fields are package-private to packetmodel, so encode+decode through
// the real codec is the construction path.
func damagePacket(t *testing.T, tm tenant.Model, attackIdx packetmodel.DamageType, damage int32, withPGExt bool) packetmodel.DamageTakenInfo {
	t.Helper()
	return decodeDamagePacket(t, tm, attackIdx, damage, withPGExt)
}

func damageTestField() field.Model {
	return field.NewBuilder(2, 1, 100000000).Build()
}

// decodeDamagePacket produces a DamageTakenInfo as the handler would see
// it, by round-tripping raw bytes through the real decoder.
func decodeDamagePacket(t *testing.T, tm tenant.Model, attackIdx packetmodel.DamageType, damage int32, withPGExt bool) packetmodel.DamageTakenInfo {
	t.Helper()
	ctx := tenant.WithContext(context.Background(), tm)
	l, _ := test.NewNullLogger()

	w := response.NewWriter(l)
	w.WriteInt(uint32(12345))
	w.WriteInt8(int8(attackIdx))
	w.WriteInt8(0)
	w.WriteInt32(damage)
	if attackIdx >= packetmodel.DamageTypePhysical {
		w.WriteInt(uint32(200100)) // mobTemplateId
		w.WriteInt(uint32(42))     // mobId
		w.WriteBool(true)          // left
		if withPGExt {
			w.WriteByte(30) // reflect echo
		} else {
			w.WriteByte(0)
		}
		if tm.Region() == "GMS" && tm.MajorVersion() >= 95 {
			w.WriteBool(false)
		}
		w.WriteByte(0) // blockByte
		if withPGExt {
			w.WriteBool(true)      // isPowerGuard
			w.WriteInt(uint32(42)) // reflectTargetMobId == attacking mob
			w.WriteByte(3)         // hitAction
			w.WriteInt16(100)
			w.WriteInt16(200)
			w.WriteInt16(110)
			w.WriteInt16(210)
		}
	} else {
		w.WriteInt16(0) // obstacleData
	}
	w.WriteByte(0) // stanceFlags

	req := request.Request(w.Bytes())
	reader := request.NewRequestReader(&req, 0)
	m := packetmodel.NewDamageTakenInfo(42)
	m.Decode(l, ctx)(&reader, nil)
	if reader.Available() != 0 {
		t.Fatalf("test packet under-consumed: %d bytes left", reader.Available())
	}
	return m
}

func TestProcessDamageTakenNoBuffAppliesFullDamage(t *testing.T) {
	l, _ := test.NewNullLogger()
	tm := testTenantModel(t, "GMS", 83)
	em := &emissions{}
	deps := fakeDeps(em, nil, nil, effect.Model{}, monster.Model{}, monsterdata.Model{})

	p := damagePacket(t, tm, packetmodel.DamageTypePhysical, 500, false)
	c := testCharacter(t, job.Id(100), 100, 0, nil)
	processDamageTaken(l, tm, damageTestField(), p, c, deps)

	if len(em.hp) != 1 || em.hp[0] != -500 {
		t.Fatalf("hp emissions=%v, want [-500]", em.hp)
	}
	if len(em.mp)+len(em.meso)+len(em.reflects) != 0 {
		t.Fatalf("unexpected side effects: %+v", em)
	}
}

func TestProcessDamageTakenMagicGuardEmitsHPAndMP(t *testing.T) {
	l, _ := test.NewNullLogger()
	tm := testTenantModel(t, "GMS", 83)
	em := &emissions{}
	buffs := []buff.Model{activeBuff(charconst.TemporaryStatTypeMagicGuard, 80)}
	deps := fakeDeps(em, buffs, nil, effect.Model{}, monster.Model{}, monsterdata.Model{})

	p := damagePacket(t, tm, packetmodel.DamageTypePhysical, 1000, false)
	c := testCharacter(t, job.Id(200), 2000, 0, nil)
	processDamageTaken(l, tm, damageTestField(), p, c, deps)

	if len(em.hp) != 1 || em.hp[0] != -200 {
		t.Fatalf("hp=%v, want [-200]", em.hp)
	}
	if len(em.mp) != 1 || em.mp[0] != -800 {
		t.Fatalf("mp=%v, want [-800]", em.mp)
	}
}

func TestProcessDamageTakenMesoGuardEmitsMesoDeduction(t *testing.T) {
	l, _ := test.NewNullLogger()
	tm := testTenantModel(t, "GMS", 83)
	em := &emissions{}
	buffs := []buff.Model{activeBuff(charconst.TemporaryStatTypeMesoGuard, 81)}
	deps := fakeDeps(em, buffs, nil, effect.Model{}, monster.Model{}, monsterdata.Model{})

	p := damagePacket(t, tm, packetmodel.DamageTypePhysical, 1000, false)
	c := testCharacter(t, job.Id(422), 100, 1000000, nil)
	processDamageTaken(l, tm, damageTestField(), p, c, deps)

	if len(em.hp) != 1 || em.hp[0] != -500 {
		t.Fatalf("hp=%v, want [-500]", em.hp)
	}
	if len(em.meso) != 1 || em.meso[0] != -405 {
		t.Fatalf("meso=%v, want [-405]", em.meso)
	}
}

func TestProcessDamageTakenPowerGuardReflects(t *testing.T) {
	l, _ := test.NewNullLogger()
	tm := testTenantModel(t, "GMS", 83)
	em := &emissions{}
	buffs := []buff.Model{activeBuff(charconst.TemporaryStatTypePowerGuard, 30)}
	mob, err := monster.NewModelBuilder(42, damageTestField(), 200100).SetHp(50000).SetMaxHp(100000).Build()
	if err != nil {
		t.Fatal(err)
	}
	deps := fakeDeps(em, buffs, nil, effect.Model{}, mob, monsterdata.Model{})

	p := damagePacket(t, tm, packetmodel.DamageTypePhysical, 1000, true)
	c := testCharacter(t, job.Id(110), 100, 0, nil)
	processDamageTaken(l, tm, damageTestField(), p, c, deps)

	if len(em.reflects) != 1 || em.reflects[0] != 300 {
		t.Fatalf("reflects=%v, want [300]", em.reflects)
	}
	if len(em.hp) != 1 || em.hp[0] != -700 {
		t.Fatalf("hp=%v, want [-700]", em.hp)
	}
}

// Forged claim: isPowerGuard set on the wire but no POWER_GUARD buff —
// the claim is ignored, full damage applies, nothing reflects.
func TestProcessDamageTakenForgedPowerGuardIgnored(t *testing.T) {
	l, _ := test.NewNullLogger()
	tm := testTenantModel(t, "GMS", 83)
	em := &emissions{}
	deps := fakeDeps(em, nil, nil, effect.Model{}, monster.Model{}, monsterdata.Model{})

	p := damagePacket(t, tm, packetmodel.DamageTypePhysical, 1000, true)
	c := testCharacter(t, job.Id(110), 100, 0, nil)
	processDamageTaken(l, tm, damageTestField(), p, c, deps)

	if len(em.reflects) != 0 {
		t.Fatalf("reflects=%v, want none", em.reflects)
	}
	if len(em.hp) != 1 || em.hp[0] != -1000 {
		t.Fatalf("hp=%v, want [-1000]", em.hp)
	}
}

// Forged oversized damage is clamped, and the int16 conversion is bounded.
func TestProcessDamageTakenForgedDamageClamped(t *testing.T) {
	l, _ := test.NewNullLogger()
	tm := testTenantModel(t, "GMS", 83)
	em := &emissions{}
	deps := fakeDeps(em, nil, nil, effect.Model{}, monster.Model{}, monsterdata.Model{})

	p := damagePacket(t, tm, packetmodel.DamageTypePhysical, 50000000, false)
	c := testCharacter(t, job.Id(100), 100, 0, nil)
	processDamageTaken(l, tm, damageTestField(), p, c, deps)

	if len(em.hp) != 1 || em.hp[0] != -32767 {
		t.Fatalf("hp=%v, want [-32767] (clamped int16)", em.hp)
	}
}

// Block sentinel (-1) must not touch HP; the old handler healed +1.
func TestProcessDamageTakenSentinelNoHPChange(t *testing.T) {
	l, _ := test.NewNullLogger()
	tm := testTenantModel(t, "GMS", 83)
	em := &emissions{}
	deps := fakeDeps(em, nil, nil, effect.Model{}, monster.Model{}, monsterdata.Model{})

	p := damagePacket(t, tm, packetmodel.DamageTypePhysical, -1, false)
	c := testCharacter(t, job.HeroId, 100, 0, nil)
	processDamageTaken(l, tm, damageTestField(), p, c, deps)

	if len(em.hp) != 0 {
		t.Fatalf("hp=%v, want none for block sentinel", em.hp)
	}
}

func TestProcessDamageTakenAchillesPassive(t *testing.T) {
	l, _ := test.NewNullLogger()
	tm := testTenantModel(t, "GMS", 83)
	em := &emissions{}
	skills := []skill2.Model{buildSkillModel(t, skillconst.HeroAchillesId, 30)}
	eff, err := effect.Extract(effect.RestModel{X: 850})
	if err != nil {
		t.Fatal(err)
	}
	deps := fakeDeps(em, nil, skills, eff, monster.Model{}, monsterdata.Model{})

	p := damagePacket(t, tm, packetmodel.DamageTypePhysical, 1000, false)
	c := testCharacter(t, job.HeroId, 100, 0, skills)
	processDamageTaken(l, tm, damageTestField(), p, c, deps)

	if len(em.hp) != 1 || em.hp[0] != -850 {
		t.Fatalf("hp=%v, want [-850] (Achilles x=850)", em.hp)
	}
}
