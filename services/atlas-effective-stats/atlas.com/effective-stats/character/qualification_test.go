package character

import (
	"testing"

	"atlas-effective-stats/external/data/equipment"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
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
