package factory

import (
	"atlas-character-factory/configuration/tenant/characters/preset"
	"atlas-character-factory/configuration/tenant/maplelife"
	"errors"
	"strconv"
	"strings"

	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

var (
	ErrMapleLifeNotConfigured = errors.New("maple life not configured for tenant")
	ErrClassOrdinalUnknown    = errors.New("maple life class ordinal not configured")
	ErrLookInvalid            = errors.New("maple life look selection invalid")
	ErrSPInvalid              = errors.New("maple life sp selection invalid")
)

func findMapleLifeClass(classes []maplelife.ClassEntry, ordinal uint32, gender byte) (maplelife.ClassEntry, bool) {
	for _, c := range classes {
		if c.Ordinal == ordinal && c.Gender == gender {
			return c, true
		}
	}
	return maplelife.ClassEntry{}, false
}

func findMapleLifeLook(looks []maplelife.LookOptions, gender byte) (maplelife.LookOptions, bool) {
	for _, l := range looks {
		if l.Gender == gender {
			return l, true
		}
	}
	return maplelife.LookOptions{}, false
}

// toPreset projects a Maple Life class entry plus the player's own choices
// onto the preset shape, so buildPresetCharacterCreationSaga -- already in
// production for the admin path -- builds the saga. Three fields come from
// the player rather than the configuration, which is the whole difference
// between this and a preset: the four look values, the gender, and the level
// of the class's SP skill.
//
// effectX is the atlas-data-sourced per-level HP/MP gain bonus for
// e.SpSkillId at level in.SP (0 when the class has no SP skill, or the
// player invested none of it). Taking it as a parameter rather than
// resolving it here -- a deviation from the plan text's toPreset(e, in)
// two-argument shape -- is the CONTROLLER AMENDMENT's requirement that Task
// 22 compute the SP-skill's own HP/MP contribution: CreateMapleLife has
// already run the atlas-data lookup by the time it calls this, and this
// function stays a pure projection rather than repeating that I/O.
func toPreset(e maplelife.ClassEntry, in MapleLifeCreateRestModel, effectX int16) preset.RestModel {
	equipment := make([]preset.EquipmentEntry, 0, len(e.Equipment))
	for _, eq := range e.Equipment {
		equipment = append(equipment, preset.EquipmentEntry{TemplateId: eq.TemplateId, UseAverageStats: eq.UseAverageStats})
	}
	inventory := make([]preset.InventoryEntry, 0, len(e.Inventory))
	for _, inv := range e.Inventory {
		inventory = append(inventory, preset.InventoryEntry{TemplateId: inv.TemplateId, Quantity: inv.Quantity})
	}

	var skills []preset.SkillEntry
	if e.SpSkillId != 0 && in.SP > 0 {
		skills = []preset.SkillEntry{{SkillId: e.SpSkillId, Level: in.SP}}
	}

	// maple-life-content.md §5.3(b): the seeded Stats.Hp/Mp is the
	// skill-EXCLUDED midpoint (Task 20's job). The SP-skill's own
	// contribution is `29 x levelUps x(nSP)`, mirroring
	// ProcessLevelChange's single-batch resolve-once-per-call shape for
	// Maple Life's synthetic level 1->30 history. Only Warrior (HP) and
	// Magician (MP) ever carry a non-zero effectX; the other three classes
	// pass their seeded stats through untouched.
	stats := e.Stats
	if e.SpSkillId != 0 && in.SP > 0 && effectX != 0 {
		contribution := uint16(29 * int(effectX))
		switch e.SpSkillId {
		case uint32(skill.WarriorImprovedMaxHpIncreaseId):
			stats.Hp += contribution
		case uint32(skill.MagicianImprovedMaxMpIncreaseId):
			stats.Mp += contribution
		}
	}

	remainingSP := spendSPPool(e.SP, in.SP)

	return preset.RestModel{
		Attributes: preset.Attributes{
			JobId:     e.JobId,
			Gender:    in.Gender,
			Face:      in.Face,
			Hair:      in.Hair,
			HairColor: in.HairColor,
			SkinColor: in.SkinColor,
			MapId:     e.MapId,
			Level:     e.Level,
			Meso:      e.Meso,
			Gm:        0,
			Stats: preset.StatBlock{
				Str: stats.Str,
				Dex: stats.Dex,
				Int: stats.Int,
				Luk: stats.Luk,
				Hp:  stats.Hp,
				Mp:  stats.Mp,
			},
			Equipment: equipment,
			Inventory: inventory,
			Skills:    skills,
			AP:        e.AP,
			SP:        remainingSP,
		},
	}
}

// parseSPPool decodes the ten-book SP string atlas-character persists
// (character/entity.go:58, "0,0,0,0,0,0,0,0,0,0") into its per-book ints.
// Maple Life only ever reads/spends slot 0.
func parseSPPool(sp string) ([]int, bool) {
	parts := strings.Split(sp, ",")
	out := make([]int, len(parts))
	for i, part := range parts {
		v, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil {
			return nil, false
		}
		out[i] = v
	}
	return out, true
}

// spendSPPool spends nSP out of slot 0 of the ten-book SP string, leaving the
// other nine books untouched, and re-encodes it in the same form. An
// unparseable pool (should not happen -- entry.SP is validated seed data)
// passes through unchanged rather than panicking.
func spendSPPool(sp string, nSP byte) string {
	pool, ok := parseSPPool(sp)
	if !ok || len(pool) == 0 {
		return sp
	}
	pool[0] -= int(nSP)
	parts := make([]string, len(pool))
	for i, v := range pool {
		parts[i] = strconv.Itoa(v)
	}
	return strings.Join(parts, ",")
}
