package periodic

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
)

// effects is the periodic-effect table (task-214 FR-1.1). Every value here is
// WZ-verified in docs/tasks/task-214-buff-tick-effects/design.md §2:
//
//	POISON        1s HP drain, no floor — preserves the pre-task-214 tick
//	              behavior exactly; poison is allowed to reach 0 HP.
//	DRAGON_BLOOD  4s HP drain, floor 1 — Skill.wz 1311008 level nodes carry
//	              mpCon/x/time/pad; x decreases with level while pad rises, and
//	              String.wz reads "Use 12 MP, 48 HP in every 4 seconds,
//	              Attack + 1 in 8 Seconds". x is the per-4s HP cost, pad the
//	              attack bonus.
//	RECOVERY      5s HP restore — Skill.wz 10001001 level 1 is x=4, time=30;
//	              String.wz reads "Recover HP 24 in 30 sec." 24/4 = 6 ticks
//	              over 30s = one tick per 5s (levels 2 and 3 confirm).
//
// Intervals are compile-time constants, never configuration and never fetched
// per tick (FR-1.3). This map is the ONLY place a periodic stat type is named
// (FR-1.2) — no tick-path code compares a stat type to a literal.
var effects = map[character.TemporaryStatType]Effect{
	character.TemporaryStatTypePoison: {
		statType:  character.TemporaryStatTypePoison,
		interval:  time.Second,
		resource:  ResourceHP,
		direction: Drain,
		floor:     false,
	},
	character.TemporaryStatTypeDragonBlood: {
		statType:  character.TemporaryStatTypeDragonBlood,
		interval:  4 * time.Second,
		resource:  ResourceHP,
		direction: Drain,
		floor:     true,
	},
	character.TemporaryStatTypeRecovery: {
		statType:  character.TemporaryStatTypeRecovery,
		interval:  5 * time.Second,
		resource:  ResourceHP,
		direction: Restore,
		floor:     false,
	},
}

// Lookup resolves a stored buff change's stat type to its periodic row.
// The parameter is a plain string because buff/stat.Model.Type() is a string;
// the conversion to the typed constant happens here so callers never handle a
// raw literal.
func Lookup(statType string) (Effect, bool) {
	e, ok := effects[character.TemporaryStatType(statType)]
	return e, ok
}
