package combo

import (
	"atlas-channel/character"
	chskill "atlas-channel/character/skill"
	"atlas-channel/data/skill/effect"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

// ComboAbilityId picks the Combo Ability id the character's job uses. The
// client's own selector is `job != 2000 ? 21000000 : 20000017`, verified in
// CMob::OnHit and ClearCombo on every in-scope version (task-217 design.md
// §2.4). Neither id is on tools/skill-job-id-guard.sh's version-divergent
// list, so the direct comparison here is permitted.
func ComboAbilityId(jobId job.Id) skill.Id {
	if jobId == job.LegendId {
		return skill.LegendComboAbilityId
	}
	return skill.AranStage1ComboAbilityId
}

// Evaluate re-derives the client's gates from authoritative state. On failure
// it names the gate that rejected ("skill", "weapon", "effect") so the caller
// can debug-log it without a second pass.
//
// There is deliberately NO job-range check: owning Combo Ability at level > 0
// IS the job gate, exactly as the client applies it. A range check would
// reject legitimate states the client accepts (design.md §3.5).
func Evaluate(c character.Model, getEffect func(skillId uint32, level byte) (effect.Model, error)) (Eligibility, string, bool) {
	comboId := ComboAbilityId(c.JobId())

	level := chskill.GetLevel(c.Skills(), comboId)
	if level == 0 {
		return Eligibility{}, "skill", false
	}

	s, ok := c.Equipment().Get("weapon")
	if !ok || s.Equipable == nil {
		return Eligibility{}, "weapon", false
	}
	if item.GetWeaponType(item.Id(s.Equipable.TemplateId())) != item.WeaponTypePolearm {
		return Eligibility{}, "weapon", false
	}

	e, err := getEffect(uint32(comboId), level)
	if err != nil {
		return Eligibility{}, "effect", false
	}

	return NewEligibility(comboId, level, int32(e.X())), "", true
}
