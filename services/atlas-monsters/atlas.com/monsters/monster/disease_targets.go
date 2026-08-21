package monster

import (
	"atlas-monsters/monster/mobskill"
	"sync"

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

// positionLookupConcurrency bounds the number of in-flight atlas-character
// reads for a single AoE cast. A crowded field must not serialize N round
// trips into the mob's skill execution path (FR-5.3), and must not open N
// sockets at once either.
const positionLookupConcurrency = 8

// resolvePositions looks up each character's world position through the
// positionFn seam, bounded to positionLookupConcurrency in flight.
//
// Results are assembled by input index, not by completion order, so the
// returned slice is always in field-listing order no matter how the
// goroutines interleave — that is what makes the concurrency compatible
// with the selector's determinism guarantee (FR-4.1).
//
// A character whose position cannot be resolved is logged at warn and
// dropped. One unresolvable character never aborts the cast for the others
// (FR-1.4), and the candidate set only ever shrinks — it never widens back
// to "everyone in the field".
func (p *ProcessorImpl) resolvePositions(uniqueId uint32, ids []uint32) []positionedCharacter {
	slots := make([]*positionedCharacter, len(ids))
	sem := make(chan struct{}, positionLookupConcurrency)
	var wg sync.WaitGroup
	for i, id := range ids {
		wg.Add(1)
		go func(i int, id uint32) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			x, y, err := p.positionFn(id)
			if err != nil {
				p.l.WithError(err).Warnf("Unable to resolve position for character [%d] targeted by monster [%d].", id, uniqueId)
				return
			}
			slots[i] = &positionedCharacter{id: id, x: x, y: y}
		}(i, id)
	}
	wg.Wait()

	out := make([]positionedCharacter, 0, len(ids))
	for _, s := range slots {
		if s != nil {
			out = append(out, *s)
		}
	}
	return out
}

// getDiseaseTargets returns the character ids a mob skill's disease, banish,
// or dispel applies to.
//
// A skill with no bounding box targets the mob's controller and nothing
// else, regardless of the skill's count (FR-2.1) — that path makes no
// position lookup at all. A skill with a bounding box lists the field,
// resolves each character's position, and hands the ordered candidate list
// to selectDiseaseTargets.
func (p *ProcessorImpl) getDiseaseTargets(m Model, sd mobskill.Model, skillId byte) []uint32 {
	if !sd.HasBoundingBox() {
		if m.ControlCharacterId() == 0 {
			return nil
		}
		return []uint32{m.ControlCharacterId()}
	}

	ids, err := p.inFieldFn(m.Field())
	if err != nil {
		p.l.WithError(err).Errorf("Unable to get characters in field for monster [%d] disease targeting.", m.UniqueId())
		return nil
	}

	candidates := p.resolvePositions(m.UniqueId(), ids)
	targets := selectDiseaseTargets(m.X(), m.Y(), sd, skillId, candidates)
	p.l.Debugf("Monster [%d] skill [%d] AoE: [%d] in field, [%d] positioned, [%d] targeted.",
		m.UniqueId(), skillId, len(ids), len(candidates), len(targets))
	return targets
}
