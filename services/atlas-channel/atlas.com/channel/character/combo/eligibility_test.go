package combo

import (
	"atlas-channel/asset"
	"atlas-channel/character"
	chskill "atlas-channel/character/skill"
	"atlas-channel/data/skill/effect"
	"atlas-channel/equipment"
	equipslot "atlas-channel/equipment/slot"
	"errors"
	"testing"

	"github.com/google/uuid"

	slottype "github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

func testSkill(t *testing.T, id skill.Id, level byte) chskill.Model {
	t.Helper()
	m, err := chskill.Extract(chskill.RestModel{Id: uint32(id), Level: level})
	if err != nil {
		t.Fatalf("skill.Extract: %v", err)
	}
	return m
}

func buildCharacter(t *testing.T, jobId job.Id, skillId skill.Id, level byte, weaponTemplateId uint32) character.Model {
	t.Helper()
	eq := equipment.NewModel()
	if weaponTemplateId != 0 {
		w := asset.NewBuilder(uuid.New(), weaponTemplateId).SetId(1).MustBuild()
		eq.Set(slottype.Type("weapon"), equipslot.Model{Equipable: &w})
	}
	return character.NewModelBuilder().
		SetId(1).
		SetJobId(jobId).
		SetSkills([]chskill.Model{testSkill(t, skillId, level)}).
		SetEquipment(eq).
		MustBuild()
}

func stubEffect(t *testing.T, x int16) func(uint32, byte) (effect.Model, error) {
	t.Helper()
	return func(uint32, byte) (effect.Model, error) {
		se, err := effect.Extract(effect.RestModel{X: x})
		if err != nil {
			t.Fatalf("effect.Extract: %v", err)
		}
		return se, nil
	}
}

func failingEffect() func(uint32, byte) (effect.Model, error) {
	return func(uint32, byte) (effect.Model, error) {
		return effect.Model{}, errors.New("atlas-data unreachable")
	}
}

func TestComboAbilityIdSelectsLegendVariant(t *testing.T) {
	if got := ComboAbilityId(job.LegendId); got != skill.LegendComboAbilityId {
		t.Errorf("Legend: want %d, got %d", skill.LegendComboAbilityId, got)
	}
	for _, j := range []job.Id{job.AranStage1Id, job.AranStage2Id, job.AranStage3Id, job.AranStage4Id} {
		if got := ComboAbilityId(j); got != skill.AranStage1ComboAbilityId {
			t.Errorf("job %d: want %d, got %d", j, skill.AranStage1ComboAbilityId, got)
		}
	}
}

func TestEvaluateGates(t *testing.T) {
	// polearmItemId: (id/10000)%100 == 44 -> item.WeaponTypePolearm.
	const polearmItemId = uint32(1442000)
	// swordItemId: (id/10000)%100 == 30 -> one-handed sword.
	const swordItemId = uint32(1302000)

	tests := []struct {
		name     string
		jobId    job.Id
		skillId  skill.Id
		level    byte
		weapon   uint32
		wantOk   bool
		wantGate string
	}{
		{"aran with combo ability and polearm", job.AranStage1Id, skill.AranStage1ComboAbilityId, 5, polearmItemId, true, ""},
		{"legend with 20000017 and polearm", job.LegendId, skill.LegendComboAbilityId, 1, polearmItemId, true, ""},
		{"aran without combo ability", job.AranStage1Id, skill.AranStage1DoubleSwingId, 5, polearmItemId, false, "skill"},
		{"aran with combo ability at level 0", job.AranStage1Id, skill.AranStage1ComboAbilityId, 0, polearmItemId, false, "skill"},
		{"aran with a sword", job.AranStage1Id, skill.AranStage1ComboAbilityId, 5, swordItemId, false, "weapon"},
		{"non-aran", job.WarriorId, skill.AranStage1ComboAbilityId, 0, polearmItemId, false, "skill"},
		{"legend holding the aran id", job.LegendId, skill.AranStage1ComboAbilityId, 5, polearmItemId, false, "skill"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := buildCharacter(t, tc.jobId, tc.skillId, tc.level, tc.weapon)
			got, gate, ok := Evaluate(c, stubEffect(t, 7))
			if ok != tc.wantOk {
				t.Fatalf("eligible: want %v, got %v (gate %q)", tc.wantOk, ok, gate)
			}
			if !ok {
				if gate != tc.wantGate {
					t.Errorf("failing gate: want %q, got %q", tc.wantGate, gate)
				}
				return
			}
			if got.ComboId() != ComboAbilityId(tc.jobId) {
				t.Errorf("combo id: want %d, got %d", ComboAbilityId(tc.jobId), got.ComboId())
			}
			if got.ComboLevel() != tc.level {
				t.Errorf("combo level: want %d, got %d", tc.level, got.ComboLevel())
			}
			if got.StatAmount() != 7 {
				t.Errorf("stat amount: want 7 (effect x), got %d", got.StatAmount())
			}
		})
	}
}

func TestEvaluateEffectLookupFailure(t *testing.T) {
	c := buildCharacter(t, job.AranStage1Id, skill.AranStage1ComboAbilityId, 5, uint32(1442000))
	_, gate, ok := Evaluate(c, failingEffect())
	if ok {
		t.Fatal("effect lookup failed: want ineligible")
	}
	if gate != "effect" {
		t.Errorf("failing gate: want \"effect\", got %q", gate)
	}
}
