package monster

import (
	"atlas-monsters/monster/mobskill"

	monster2 "github.com/Chronicle20/atlas/libs/atlas-constants/monster"
)

// positionedCharacter pairs a character id with the world coordinates that
// were successfully resolved for it. Characters whose position could not be
// resolved never become a positionedCharacter (FR-1.4).
type positionedCharacter struct {
	id uint32
	x  int16
	y  int16
}

// selectDiseaseTargets picks the characters a mob skill's disease applies to.
//
// It is a pure function of its arguments — no I/O, no randomness — so that a
// fixed candidate list always yields the same target list (FR-4.2). The
// bounding box is the skill's lt/rb offsets translated by the caster's
// position, tested inclusively, using the same arithmetic as the ally-heal
// AoE in executeHeal. The rectangle is never mirrored by facing (FR-1.3).
//
// The count cap is applied to SEDUCE only, matching the reference server,
// where the `i < count` guard sits inside the seduce branch. Every other
// disease — plus banish and dispel, which share this selector — applies to
// every candidate inside the rectangle.
func selectDiseaseTargets(mobX, mobY int16, sd mobskill.Model, skillId byte, candidates []positionedCharacter) []uint32 {
	var ids []uint32
	for _, c := range candidates {
		dx := int32(c.x) - int32(mobX)
		dy := int32(c.y) - int32(mobY)
		if dx >= sd.LtX() && dx <= sd.RbX() && dy >= sd.LtY() && dy <= sd.RbY() {
			ids = append(ids, c.id)
		}
	}

	if uint16(skillId) == monster2.SkillTypeSeduce && sd.Count() > 0 && uint32(len(ids)) > sd.Count() {
		ids = ids[:sd.Count()]
	}
	return ids
}
