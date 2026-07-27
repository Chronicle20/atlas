package handler

import (
	"atlas-channel/character"
	"atlas-channel/character/buff"
	"atlas-channel/data/skill/effect"
	"atlas-channel/data/skill/effect/statup"
	"context"
	"math"

	"github.com/sirupsen/logrus"

	charconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
)

// MountBuffDuration is the duration applied to the MONSTER_RIDING buff. Mounts
// persist until the player toggles dismount, changes job, or logs out — there
// is no "never expires" path through atlas-buffs (it rejects duration <= 0), so
// we use the largest representable positive int32. See context.md §4.
const MountBuffDuration = int32(math.MaxInt32)

// Tamed-mount equip slots (libs/atlas-constants/inventory/slot).
const (
	tamingMobSlot int16 = -18 // the taming-mob item is the vehicle id
	saddleSlot    int16 = -19 // required for a tamed mount to engage
)

// mountDeps holds the collaborators HandleMount needs, injected as function
// seams so the five toggle cases are unit-testable offline (no Kafka, REST, or
// session). Production wiring builds these from the buff and character
// processors in UseSkill.
type mountDeps struct {
	// isMounted reports whether the character currently has an active
	// MONSTER_RIDING buff sourced from sourceId (the mount skill id).
	isMounted func(characterId uint32, sourceId int32) (bool, error)
	// equipInSlot returns the item id equipped at pos (a negative equip slot),
	// found=false when the slot is empty.
	equipInSlot func(characterId uint32, pos int16) (int32, bool, error)
	// applyBuff applies a buff (MONSTER_RIDING) carrying statups for characterId.
	applyBuff func(f field.Model, characterId uint32, sourceId int32, level byte, duration int32, statups []statup.Model) error
	// cancelBuff cancels the buff sourced from sourceId for characterId.
	cancelBuff func(f field.Model, characterId uint32, sourceId int32) error
}

// isSkillOnlyMount reports whether the skill is a skill-only mount (SpaceShip,
// Yeti, Broomstick, Balrog) — one whose vehicle id is fixed by the skill rather
// than read from an equipped taming-mob item.
func isSkillOnlyMount(id skill2.Id, level byte) bool {
	_, ok := skill2.SkillOnlyMountVehicleId(id, int(level))
	return ok
}

// tamedMountStatups returns the effect's statups with the MONSTER_RIDING amount
// overridden to vehicleId (the equipped taming-mob item id), preserving any
// other stats the mount skill grants (e.g. WEAPON_DEFENSE/MAGIC_DEFENSE). The
// vehicle id is the cross-task contract: the CTS encoder and buff consumer read
// the MONSTER_RIDING change Amount as the vehicle id. If the effect carries no
// MONSTER_RIDING statup, one is appended so the mount always renders.
func tamedMountStatups(e effect.Model, vehicleId int32) []statup.Model {
	out := make([]statup.Model, 0, len(e.StatUps())+1)
	hasRiding := false
	for _, su := range e.StatUps() {
		if su.Mask() == string(charconst.TemporaryStatTypeMonsterRiding) {
			out = append(out, statup.NewModel(su.Mask(), vehicleId))
			hasRiding = true
			continue
		}
		out = append(out, su)
	}
	if !hasRiding {
		out = append(out, statup.NewModel(string(charconst.TemporaryStatTypeMonsterRiding), vehicleId))
	}
	return out
}

// HandleMount implements the server-driven mount toggle. It runs BEFORE the
// generic buff apply in UseSkill and short-circuits it for mount skills.
//
// Cases (design §5.1):
//  1. Already mounted (active MONSTER_RIDING from this skill) -> Cancel, no Apply.
//  2. Tamed, slots -18 AND -19 both present, not mounted -> Apply the effect's
//     statups with MONSTER_RIDING amount = item@-18 (the taming-mob/vehicle id),
//     sourceId = skillId, duration = MaxInt32.
//  3. Tamed, slot -18 empty -> silent no-op.
//  4. Tamed, slot -19 empty -> silent no-op.
//  5. Skill-only, not mounted -> Apply the effect's full statup set (the vehicle
//     id atlas-data injected into MONSTER_RIDING plus any stats the skill grants),
//     no slot lookup.
//
// All no-op paths return nil; the caller (character_skill_use.go) unconditionally
// re-enables actions after UseSkill returns, so HandleMount never needs to.
func HandleMount(l logrus.FieldLogger, f field.Model, characterId uint32, info packetmodel.SkillUsageInfo, e effect.Model, deps mountDeps) error {
	skillId := skill2.Id(info.SkillId())
	sourceId := int32(info.SkillId())

	mounted, err := deps.isMounted(characterId, sourceId)
	if err != nil {
		l.WithError(err).Warnf("Character [%d] mount toggle: unable to resolve mounted state for skill [%d]; treating as not mounted.", characterId, info.SkillId())
		mounted = false
	}

	// Case 1: already mounted -> dismount toggle. Cancel, never Apply.
	if mounted {
		if err := deps.cancelBuff(f, characterId, sourceId); err != nil {
			l.WithError(err).Errorf("Character [%d] failed to cancel mount buff for skill [%d].", characterId, info.SkillId())
			return err
		}
		return nil
	}

	// Case 5: skill-only mount -> apply the effect's full statup set. atlas-data
	// already injected the vehicle id into the MONSTER_RIDING statup, so the
	// effect carries the vehicle plus any stats the skill grants (e.g. the Yeti
	// Rider's +10 weapon/magic defense). No equip-slot lookup.
	if isSkillOnlyMount(skillId, info.SkillLevel()) {
		if len(monsterRidingStatups(e)) == 0 {
			l.Warnf("Character [%d] cast skill-only mount [%d] but effect carries no MONSTER_RIDING statup; no-op.", characterId, info.SkillId())
			return nil
		}
		return deps.applyBuff(f, characterId, sourceId, info.SkillLevel(), MountBuffDuration, e.StatUps())
	}

	// Cases 2-4: tamed mount. Require BOTH the taming-mob (-18) and saddle (-19).
	tamingMobId, hasTamingMob, err := deps.equipInSlot(characterId, tamingMobSlot)
	if err != nil {
		l.WithError(err).Debugf("Character [%d] mount toggle: failed to read taming-mob slot for skill [%d]; treating as empty.", characterId, info.SkillId())
		hasTamingMob = false
	}
	if !hasTamingMob {
		// Case 3: no taming mob equipped -> silent no-op.
		return nil
	}

	_, hasSaddle, err := deps.equipInSlot(characterId, saddleSlot)
	if err != nil {
		l.WithError(err).Debugf("Character [%d] mount toggle: failed to read saddle slot for skill [%d]; treating as empty.", characterId, info.SkillId())
		hasSaddle = false
	}
	if !hasSaddle {
		// Case 4: no saddle equipped -> silent no-op.
		return nil
	}

	// Case 2: both slots present -> mount. The vehicle id is the taming-mob item
	// id (overriding atlas-data's skill-id placeholder); other granted stats are
	// preserved.
	return deps.applyBuff(f, characterId, sourceId, info.SkillLevel(), MountBuffDuration, tamedMountStatups(e, tamingMobId))
}

// monsterRidingStatups filters the effect's statups down to MONSTER_RIDING.
// Skill-only mounts carry the vehicle id as the amount of this statup (produced
// by atlas-data, Task 7).
func monsterRidingStatups(e effect.Model) []statup.Model {
	out := make([]statup.Model, 0, 1)
	for _, su := range e.StatUps() {
		if su.Mask() == string(charconst.TemporaryStatTypeMonsterRiding) {
			out = append(out, su)
		}
	}
	return out
}

// newMountDeps builds the production collaborators for HandleMount from the
// buff and character processors.
func newMountDeps(l logrus.FieldLogger, ctx context.Context) mountDeps {
	bp := buff.NewProcessor(l, ctx)
	cp := character.NewProcessor(l, ctx)
	return mountDeps{
		isMounted: func(characterId uint32, sourceId int32) (bool, error) {
			bs, err := bp.GetByCharacterId(characterId)
			if err != nil {
				return false, err
			}
			for _, b := range bs {
				if b.SourceId() == sourceId && !b.Expired() {
					return true, nil
				}
			}
			return false, nil
		},
		equipInSlot: func(characterId uint32, pos int16) (int32, bool, error) {
			a, err := cp.GetEquipableInSlot(characterId, pos)()
			if err != nil {
				// "equipable not found" means an empty slot, not a hard failure.
				return 0, false, nil
			}
			return int32(a.TemplateId()), true, nil
		},
		applyBuff: func(f field.Model, characterId uint32, sourceId int32, level byte, duration int32, statups []statup.Model) error {
			return bp.Apply(f, characterId, sourceId, level, duration, statups)(characterId)
		},
		cancelBuff: func(f field.Model, characterId uint32, sourceId int32) error {
			return bp.Cancel(f, characterId, sourceId)
		},
	}
}
