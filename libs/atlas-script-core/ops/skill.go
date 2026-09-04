package ops

import (
	"strconv"
	"time"

	saga "github.com/Chronicle20/atlas/libs/atlas-saga"
)

const (
	opCreateSkill = "create_skill"
	opUpdateSkill = "update_skill"
)

type skillParams struct {
	skillId     uint32
	level       byte
	masterLevel byte
	expiration  time.Time
}

// decodeSkillParams reads the parameter contract shared by create_skill and
// update_skill.
//
// Parameters:
//   - skillId     (required)
//   - level       (optional) defaults to 1.
//   - masterLevel (optional) defaults to 1.
//   - expiration  (optional) epoch milliseconds. The sentinel "-1" means "no
//     expiration" and resolves to 100 years out. A non-positive value other
//     than the sentinel falls back to the 1-year default. Absent defaults to
//     1 year out. (npc-conversations previously ignored this parameter and
//     always used the 1-year default — design §5.4.)
func decodeSkillParams(p map[string]string, r Resolver, cid uint32, op string) (skillParams, error) {
	skillIdInt, err := requiredInt(p, r, cid, op, "skillId")
	if err != nil {
		return skillParams{}, err
	}
	skillId, err := rangedUint32(op, "skillId", skillIdInt)
	if err != nil {
		return skillParams{}, err
	}

	levelInt, err := optionalInt(p, r, cid, op, "level", 1)
	if err != nil {
		return skillParams{}, err
	}
	level, err := rangedByte(op, "level", levelInt)
	if err != nil {
		return skillParams{}, err
	}

	masterLevelInt, err := optionalInt(p, r, cid, op, "masterLevel", 1)
	if err != nil {
		return skillParams{}, err
	}
	masterLevel, err := rangedByte(op, "masterLevel", masterLevelInt)
	if err != nil {
		return skillParams{}, err
	}

	// expiration is read as an opaque string, not through the int resolver:
	// it is a 64-bit epoch-millisecond value that must not go through the
	// int-width resolver (Int is scoped to platform int arithmetic), and the
	// "-1" sentinel must be compared before any numeric parse.
	raw, err := optionalString(p, r, cid, op, "expiration", "")
	if err != nil {
		return skillParams{}, err
	}
	expiration := now().Add(365 * 24 * time.Hour)
	if raw != "" {
		if raw == "-1" {
			expiration = now().Add(100 * 365 * 24 * time.Hour)
		} else {
			expMs, err := strconv.ParseInt(raw, 10, 64)
			if err != nil {
				return skillParams{}, invalidParam(op, "expiration", raw, err)
			}
			if expMs > 0 {
				expiration = time.UnixMilli(expMs)
			}
		}
	}

	return skillParams{
		skillId:     skillId,
		level:       level,
		masterLevel: masterLevel,
		expiration:  expiration,
	}, nil
}

// CreateSkill builds a CreateSkill step, granting a character a new skill.
func CreateSkill(p map[string]string, r Resolver, t Target, characterId uint32) (Step, error) {
	sp, err := decodeSkillParams(p, r, characterId, opCreateSkill)
	if err != nil {
		return Step{}, err
	}
	return newStep(saga.CreateSkill, saga.CreateSkillPayload{
		CharacterId: characterId,
		SkillId:     sp.skillId,
		Level:       sp.level,
		MasterLevel: sp.masterLevel,
		Expiration:  sp.expiration,
	}), nil
}

// UpdateSkill builds an UpdateSkill step, updating an existing character
// skill's level/masterLevel/expiration.
func UpdateSkill(p map[string]string, r Resolver, t Target, characterId uint32) (Step, error) {
	sp, err := decodeSkillParams(p, r, characterId, opUpdateSkill)
	if err != nil {
		return Step{}, err
	}
	return newStep(saga.UpdateSkill, saga.UpdateSkillPayload{
		CharacterId: characterId,
		SkillId:     sp.skillId,
		Level:       sp.level,
		MasterLevel: sp.masterLevel,
		Expiration:  sp.expiration,
	}), nil
}
