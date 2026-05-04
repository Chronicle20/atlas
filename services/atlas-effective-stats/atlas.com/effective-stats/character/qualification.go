package character

import (
	"context"

	"atlas-effective-stats/external/data/equipment"
	"atlas-effective-stats/stat"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
)

// AppliedStats is the per-evaluation snapshot of wearer numeric stats used
// to test equipment requirements. It is the sum of base stats + always-on
// (buff/passive) flat bonuses + flat bonuses from the currently-qualifying
// equipment subset.
type AppliedStats struct {
	Strength     uint32
	Dexterity    uint32
	Intelligence uint32
	Luck         uint32
}

// wearerClassMask maps an internal atlas job id to the v83 reqJob bitmask.
// atlas internal jobIds are NOT raw v83 client bitmasks (Magician 1st = 200,
// not 2), so a direct AND would silently misqualify every class-restricted
// item. This helper centralises the mapping.
//
// v83 bits: Warrior=1, Magician=2, Bowman=4, Thief=8, Pirate=16. Beginner
// classes (no class restriction in v83 reqJob semantics) map to 0.
func wearerClassMask(id job.Id) uint16 {
	branch := uint16(id) / 100
	switch branch {
	case 0, 10, 20: // Beginner / Noblesse / Legend
		return 0
	case 1, 11, 21: // Warrior, DawnWarrior, Aran (jobId/100 = 21)
		return 1
	case 2, 12, 22: // Magician, BlazeWizard, Evan (jobId/100 = 22)
		return 2
	case 3, 13: // Bowman, WindArcher
		return 4
	case 4, 14: // Thief, NightWalker
		return 8
	case 5, 15: // Pirate, ThunderBreaker
		return 16
	default:
		return 0
	}
}

// meetsRequirements returns true when the wearer satisfies every populated
// requirement on the equipment template. A zero req is "no restriction".
func meetsRequirements(r equipment.EquipmentRequirements, s AppliedStats, level byte, jobId job.Id) bool {
	if r.ReqLevel > 0 && level < r.ReqLevel {
		return false
	}
	if r.ReqJob > 0 && wearerClassMask(jobId)&r.ReqJob == 0 {
		return false
	}
	if r.ReqStr > 0 && s.Strength < uint32(r.ReqStr) {
		return false
	}
	if r.ReqDex > 0 && s.Dexterity < uint32(r.ReqDex) {
		return false
	}
	if r.ReqInt > 0 && s.Intelligence < uint32(r.ReqInt) {
		return false
	}
	if r.ReqLuk > 0 && s.Luck < uint32(r.ReqLuk) {
		return false
	}
	return true
}

// QualifiedEquipment runs the fixed-point iteration described in design §4.3
// and returns the set of asset ids whose template requirements are satisfied
// under the wearer's base stats + non-equipment bonuses + the qualifying
// equipment subset itself.
//
// Provider failures (cold cache + atlas-data unreachable) drop the asset
// from this evaluation; callers do NOT see a separate error path.
func (m Model) QualifiedEquipment(reqProvider equipment.Provider, ctx context.Context) map[uint32]bool {
	qualified := make(map[uint32]bool, len(m.equipped))
	if len(m.equipped) == 0 {
		return qualified
	}

	flatNonEquip := sumFlatNonEquipBonuses(m.bonuses)

	computeApplied := func() AppliedStats {
		s := AppliedStats{
			Strength:     uint32max0(int32(m.baseStats.Strength()) + flatNonEquip[stat.TypeStrength]),
			Dexterity:    uint32max0(int32(m.baseStats.Dexterity()) + flatNonEquip[stat.TypeDexterity]),
			Intelligence: uint32max0(int32(m.baseStats.Intelligence()) + flatNonEquip[stat.TypeIntelligence]),
			Luck:         uint32max0(int32(m.baseStats.Luck()) + flatNonEquip[stat.TypeLuck]),
		}
		for assetId, snap := range m.equipped {
			if !qualified[assetId] {
				continue
			}
			for _, b := range snap.bonuses {
				switch b.StatType() {
				case stat.TypeStrength:
					s.Strength = addClamp(s.Strength, b.Amount())
				case stat.TypeDexterity:
					s.Dexterity = addClamp(s.Dexterity, b.Amount())
				case stat.TypeIntelligence:
					s.Intelligence = addClamp(s.Intelligence, b.Amount())
				case stat.TypeLuck:
					s.Luck = addClamp(s.Luck, b.Amount())
				}
			}
		}
		return s
	}

	for {
		applied := computeApplied()
		added := false
		for assetId, snap := range m.equipped {
			if qualified[assetId] {
				continue
			}
			req, ok := reqProvider(ctx, snap.templateId)
			if !ok {
				continue
			}
			if meetsRequirements(req, applied, m.wearer.level, m.wearer.jobId) {
				qualified[assetId] = true
				added = true
			}
		}
		if !added {
			return qualified
		}
	}
}

func sumFlatNonEquipBonuses(bs []stat.Bonus) map[stat.Type]int32 {
	out := make(map[stat.Type]int32, 8)
	for _, b := range bs {
		out[b.StatType()] += b.Amount()
	}
	return out
}

func uint32max0(v int32) uint32 {
	if v < 0 {
		return 0
	}
	return uint32(v)
}

func addClamp(s uint32, delta int32) uint32 {
	if delta >= 0 {
		return s + uint32(delta)
	}
	d := uint32(-delta)
	if d > s {
		return 0
	}
	return s - d
}
