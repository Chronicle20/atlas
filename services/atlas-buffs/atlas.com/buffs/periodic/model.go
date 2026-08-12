// Package periodic holds the declarative table of temporary-stat types that
// carry an ongoing periodic change to the buffed character's own HP/MP, and
// nothing else. It is pure: no Redis, no Kafka, no REST — so the table is
// unit-testable on its own and the tick path in character/ has exactly one
// place to ask "is this stat type periodic, and on what schedule?"
// (task-214 FR-1.1/FR-1.2).
package periodic

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
)

// Resource names the character resource a periodic effect moves.
type Resource string

// ResourceHP is the only resource any current row targets. Adding an MP row
// means adding ResourceMP here AND an emit arm in character.ProcessPeriodicTicks
// — the emitter's default arm logs and skips rather than silently emitting
// nothing.
const ResourceHP Resource = "HP"

// Direction is the sign applied to an effect's per-tick magnitude.
type Direction int8

const (
	// Drain reduces the resource.
	Drain Direction = -1
	// Restore increases the resource.
	Restore Direction = 1
)

// Effect is one row of the periodic-effect table. Fields are unexported with
// accessors so the table cannot be mutated by a caller (project immutable-model
// convention).
type Effect struct {
	statType      character.TemporaryStatType
	interval      time.Duration
	resource      Resource
	direction     Direction
	floor         bool
	specialEffect bool
}

// StatType is the temporary-stat type stored on the buff change this row keys off.
func (e Effect) StatType() character.TemporaryStatType { return e.statType }

// Interval is the cadence between ticks for this effect.
func (e Effect) Interval() time.Duration { return e.interval }

// Resource is the character resource the tick moves.
func (e Effect) Resource() Resource { return e.resource }

// Direction is the sign applied to the per-tick magnitude.
func (e Effect) Direction() Direction { return e.direction }

// Floor reports whether the tick must clamp the resource at 1 rather than let
// it reach 0. atlas-character emits a DIED status event whenever an adjusted HP
// lands on 0, so a self-inflicted drain that must not kill sets this true.
func (e Effect) Floor() bool { return e.floor }

// SpecialEffect reports whether each tick should also broadcast the skill's
// SKILL_SPECIAL user effect, so the caster and everyone on their map see the
// skill animation pulse alongside the resource change.
//
// This is a per-row property because it depends on the SOURCE SKILL'S WZ DATA,
// not on the stat type being periodic. GMS v83 Skill.wz was surveyed directly:
// 340 skills carry an `effect` node, 57 an `affected` node, 37 a `special`
// node. The client's CUser::OnEffect case 5 (SKILL_SPECIAL) routes to
// CUser::ShowSkillSpecialEffect, which reads SKILLENTRY::GetSpecialUOL -- the
// `special` node -- and returns without drawing when that node is absent.
//
//	DRAGON_BLOOD (1311008)  has `special`  -> true; the pulse renders.
//	RECOVERY     (0001001)  has neither `special` nor `affected`, only the
//	                        `effect` (cast) node -> false. Broadcasting any
//	                        user effect for it would draw nothing at best, or
//	                        replay the cast animation every 5s at worst.
//	POISON                  is a mob debuff with no caster skill to pulse.
//
// Sending SKILL_AFFECTED instead would render nothing for either: the
// `affected` set is party/target buffs (Rage 1101006, Iron Will 1301006,
// Hyper Body 1301007, Maple Warrior 1121000/1221000/1321000) -- the animation
// a RECIPIENT plays -- and neither 1311008 nor 0001001 is in it.
func (e Effect) SpecialEffect() bool { return e.specialEffect }
