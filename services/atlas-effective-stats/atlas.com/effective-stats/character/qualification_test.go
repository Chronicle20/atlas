package character

import (
	"context"
	"testing"

	"atlas-effective-stats/external/data/equipment"
	"atlas-effective-stats/stat"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
)

func TestWearerClassMask_StandardClasses(t *testing.T) {
	cases := []struct {
		name string
		id   job.Id
		want uint16
	}{
		{"Beginner", 0, 0},
		{"Warrior 1st", 100, 1},
		{"Fighter 2nd", 110, 1},
		{"Crusader 3rd", 111, 1},
		{"Hero 4th", 112, 1},
		{"Magician 1st", 200, 2},
		{"FP Wizard 2nd", 210, 2},
		{"Bowman 1st", 300, 4},
		{"Thief 1st", 400, 8},
		{"Pirate 1st", 500, 16},
		{"DawnWarrior 1st", 1100, 1},
		{"BlazeWizard 1st", 1200, 2},
		{"WindArcher 1st", 1300, 4},
		{"NightWalker 1st", 1400, 8},
		{"ThunderBreaker 1st", 1500, 16},
		{"Aran 1st (2100)", 2100, 1},
		{"Evan 2nd (2200)", 2200, 2},
		{"Noblesse beginner", 1000, 0},
		{"Legend beginner", 2000, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := wearerClassMask(c.id); got != c.want {
				t.Errorf("mask(%d) = %d, want %d", c.id, got, c.want)
			}
		})
	}
}

func TestMeetsRequirements_AllZerosAlwaysPass(t *testing.T) {
	r := equipment.EquipmentRequirements{}
	if !meetsRequirements(r, AppliedStats{}, 0, job.Id(0)) {
		t.Error("zero reqs should always pass, even with zero wearer")
	}
}

func TestMeetsRequirements_LevelGate(t *testing.T) {
	r := equipment.EquipmentRequirements{ReqLevel: 30}
	if meetsRequirements(r, AppliedStats{}, 29, job.Id(100)) {
		t.Error("level 29 should fail reqLevel=30")
	}
	if !meetsRequirements(r, AppliedStats{}, 30, job.Id(100)) {
		t.Error("level 30 should pass reqLevel=30")
	}
	if !meetsRequirements(r, AppliedStats{}, 31, job.Id(100)) {
		t.Error("level 31 should pass reqLevel=30")
	}
}

func TestMeetsRequirements_JobBitmask(t *testing.T) {
	// Magician-only item.
	r := equipment.EquipmentRequirements{ReqJob: 2}
	if meetsRequirements(r, AppliedStats{}, 1, job.Id(100)) {
		t.Error("Warrior should not pass Magician-only item")
	}
	if !meetsRequirements(r, AppliedStats{}, 1, job.Id(200)) {
		t.Error("Magician should pass Magician-only item")
	}
	if meetsRequirements(r, AppliedStats{}, 1, job.Id(0)) {
		t.Error("Beginner (mask 0) should not pass class-restricted item")
	}
	// Cross-class (Warrior | Magician).
	rCross := equipment.EquipmentRequirements{ReqJob: 1 | 2}
	if !meetsRequirements(rCross, AppliedStats{}, 1, job.Id(100)) {
		t.Error("Warrior should pass W|M cross-class item")
	}
	if !meetsRequirements(rCross, AppliedStats{}, 1, job.Id(200)) {
		t.Error("Magician should pass W|M cross-class item")
	}
	if meetsRequirements(rCross, AppliedStats{}, 1, job.Id(300)) {
		t.Error("Bowman should not pass W|M cross-class item")
	}
}

func TestMeetsRequirements_StatGates_OffByOne(t *testing.T) {
	r := equipment.EquipmentRequirements{ReqStr: 100, ReqDex: 50, ReqInt: 10, ReqLuk: 40}
	pass := AppliedStats{Strength: 100, Dexterity: 50, Intelligence: 10, Luck: 40}
	if !meetsRequirements(r, pass, 1, job.Id(100)) {
		t.Error("exact match should pass")
	}
	below := AppliedStats{Strength: 99, Dexterity: 50, Intelligence: 10, Luck: 40}
	if meetsRequirements(r, below, 1, job.Id(100)) {
		t.Error("STR-1 should fail")
	}
	below = AppliedStats{Strength: 100, Dexterity: 49, Intelligence: 10, Luck: 40}
	if meetsRequirements(r, below, 1, job.Id(100)) {
		t.Error("DEX-1 should fail")
	}
	below = AppliedStats{Strength: 100, Dexterity: 50, Intelligence: 9, Luck: 40}
	if meetsRequirements(r, below, 1, job.Id(100)) {
		t.Error("INT-1 should fail")
	}
	below = AppliedStats{Strength: 100, Dexterity: 50, Intelligence: 10, Luck: 39}
	if meetsRequirements(r, below, 1, job.Id(100)) {
		t.Error("LUK-1 should fail (the diagnosis case)")
	}
}

func newTestModel(t *testing.T, base stat.Base, wp WearerProfile, snaps ...EquippedAsset) Model {
	t.Helper()
	tn, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	m := NewModel(tn, channel.NewModel(0, 0), 12345).
		WithBaseStats(base).
		WithWearer(wp)
	for _, s := range snaps {
		m = m.WithEquippedAsset(s)
	}
	return m
}

// providerOf builds a stub Provider for the given templates. Missing entries
// return (_, false), simulating an atlas-data fetch failure.
func providerOf(reqs map[uint32]equipment.EquipmentRequirements) equipment.Provider {
	return func(_ context.Context, id uint32) (equipment.EquipmentRequirements, bool) {
		r, ok := reqs[id]
		return r, ok
	}
}

func TestQualifiedEquipment_EmptyEquippedReturnsEmpty(t *testing.T) {
	m := newTestModel(t, stat.NewBase(0, 0, 0, 0, 0, 0), NewWearerProfile(30, job.Id(100)))
	got := m.QualifiedEquipment(providerOf(nil), context.Background())
	if len(got) != 0 {
		t.Errorf("expected empty map, got %v", got)
	}
}

func TestQualifiedEquipment_DiagnosisCase_LukBelowReq(t *testing.T) {
	overall := NewEquippedAsset(42, 1052095, []stat.Bonus{
		stat.NewBonus("equipment:42", stat.TypeMaxMp, 50),
	})
	base := stat.NewBase(4, 25, 39 /*luk*/, 4, 1430, 6330)
	m := newTestModel(t, base, NewWearerProfile(30, job.Id(200)), overall)
	prov := providerOf(map[uint32]equipment.EquipmentRequirements{
		1052095: {ReqLuk: 40},
	})
	got := m.QualifiedEquipment(prov, context.Background())
	if got[42] {
		t.Error("LUK 39 should NOT qualify reqLuk=40 (diagnosis case)")
	}
}

func TestQualifiedEquipment_DiagnosisCase_LukAtReq(t *testing.T) {
	overall := NewEquippedAsset(42, 1052095, []stat.Bonus{
		stat.NewBonus("equipment:42", stat.TypeMaxMp, 50),
	})
	base := stat.NewBase(4, 25, 40 /*luk*/, 4, 1430, 6330)
	m := newTestModel(t, base, NewWearerProfile(30, job.Id(200)), overall)
	prov := providerOf(map[uint32]equipment.EquipmentRequirements{
		1052095: {ReqLuk: 40},
	})
	got := m.QualifiedEquipment(prov, context.Background())
	if !got[42] {
		t.Error("LUK 40 should qualify reqLuk=40")
	}
}

func TestQualifiedEquipment_ChainQualification(t *testing.T) {
	a := NewEquippedAsset(1, 1001, []stat.Bonus{
		stat.NewBonus("equipment:1", stat.TypeStrength, 5),
	})
	b := NewEquippedAsset(2, 1002, []stat.Bonus{
		stat.NewBonus("equipment:2", stat.TypeStrength, 5),
	})
	c := NewEquippedAsset(3, 1003, nil)
	base := stat.NewBase(50, 0, 0, 0, 0, 0)
	m := newTestModel(t, base, NewWearerProfile(30, job.Id(100)), a, b, c)
	prov := providerOf(map[uint32]equipment.EquipmentRequirements{
		1001: {},
		1002: {ReqStr: 55},
		1003: {ReqStr: 60},
	})
	got := m.QualifiedEquipment(prov, context.Background())
	if !got[1] || !got[2] || !got[3] {
		t.Errorf("chain should converge to {1,2,3}; got %v", got)
	}
}

func TestQualifiedEquipment_MutualCycle_NeitherQualifies(t *testing.T) {
	a := NewEquippedAsset(1, 1001, []stat.Bonus{
		stat.NewBonus("equipment:1", stat.TypeStrength, 5),
	})
	b := NewEquippedAsset(2, 1002, []stat.Bonus{
		stat.NewBonus("equipment:2", stat.TypeDexterity, 5),
	})
	base := stat.NewBase(50, 5, 0, 0, 0, 0)
	m := newTestModel(t, base, NewWearerProfile(30, job.Id(100)), a, b)
	prov := providerOf(map[uint32]equipment.EquipmentRequirements{
		1001: {ReqDex: 10},
		1002: {ReqStr: 55},
	})
	got := m.QualifiedEquipment(prov, context.Background())
	if got[1] || got[2] {
		t.Errorf("mutual cycle should leave both unqualified; got %v", got)
	}
}

func TestQualifiedEquipment_ProviderFailureExcludesAsset(t *testing.T) {
	a := NewEquippedAsset(1, 1001, nil)
	b := NewEquippedAsset(2, 1002, nil)
	base := stat.NewBase(0, 0, 0, 0, 0, 0)
	m := newTestModel(t, base, NewWearerProfile(30, job.Id(100)), a, b)
	prov := providerOf(map[uint32]equipment.EquipmentRequirements{
		1002: {}, // 1001 deliberately missing → provider returns (_, false)
	})
	got := m.QualifiedEquipment(prov, context.Background())
	if got[1] {
		t.Error("provider miss should exclude asset 1")
	}
	if !got[2] {
		t.Error("asset 2 should still qualify")
	}
}

func TestQualifiedEquipment_BuffsAndPassivesContributeToApplied(t *testing.T) {
	a := NewEquippedAsset(1, 1001, nil)
	base := stat.NewBase(100, 0, 0, 0, 0, 0)
	m := newTestModel(t, base, NewWearerProfile(30, job.Id(100)), a).
		WithBonus(stat.NewBonus("buff:9001", stat.TypeStrength, 5)).
		WithBonus(stat.NewBonus("passive:9002", stat.TypeStrength, 5))
	prov := providerOf(map[uint32]equipment.EquipmentRequirements{
		1001: {ReqStr: 110},
	})
	got := m.QualifiedEquipment(prov, context.Background())
	if !got[1] {
		t.Error("buff+passive should help asset 1 qualify")
	}
}
