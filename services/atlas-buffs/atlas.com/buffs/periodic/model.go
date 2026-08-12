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
	statType  character.TemporaryStatType
	interval  time.Duration
	resource  Resource
	direction Direction
	floor     bool
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
