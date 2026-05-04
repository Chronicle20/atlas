package character

import (
	"atlas-effective-stats/external/data/equipment"

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
