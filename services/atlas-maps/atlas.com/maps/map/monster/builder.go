package monster

import (
	"time"

	monster2 "atlas-maps/data/map/monster"
)

// Builder constructs a CooldownSpawnPoint via fluent setters.
//
// The struct's fields stay exported: CooldownSpawnPoint is constructed as a
// literal across map/monster and dozens of tests, and toStored/fromStored
// depend on the exported shape for their JSON round-trip. This Builder is
// added alongside the existing exported fields, not in place of them.
type Builder struct {
	spawnPoint  monster2.SpawnPoint
	nextSpawnAt time.Time
}

// NewBuilder constructs a Builder with zero-valued fields.
func NewBuilder() *Builder {
	return &Builder{}
}

// SetSpawnPoint sets the embedded base spawn point data.
func (b *Builder) SetSpawnPoint(sp monster2.SpawnPoint) *Builder {
	b.spawnPoint = sp
	return b
}

// SetNextSpawnAt sets the time when this spawn point becomes eligible again.
func (b *Builder) SetNextSpawnAt(t time.Time) *Builder {
	b.nextSpawnAt = t
	return b
}

// Build returns the constructed CooldownSpawnPoint.
func (b *Builder) Build() CooldownSpawnPoint {
	return CooldownSpawnPoint{
		SpawnPoint:  b.spawnPoint,
		NextSpawnAt: b.nextSpawnAt,
	}
}
