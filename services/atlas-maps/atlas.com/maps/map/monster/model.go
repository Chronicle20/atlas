package monster

import (
	"atlas-maps/data/map/monster"
	"time"
)

// CooldownSpawnPoint extends SpawnPoint with cooldown tracking functionality.
// This is used by the spawn point cooldown mechanism to prevent over-spawning.
//
// The cooldown mechanism works as follows:
// - When a monster is spawned from this spawn point, NextSpawnAt is set to time.Now() + 5 seconds
// - During spawn filtering, only spawn points with NextSpawnAt.Before(now) are considered eligible
// - This prevents immediate re-spawning and enforces a 5-second cooldown period
// - The registry is scoped per MapKey (tenant/world/channel/map) for multi-tenant support
type CooldownSpawnPoint struct {
	monster.SpawnPoint           // Embedded base spawn point data
	NextSpawnAt        time.Time // Time when this spawn point becomes eligible again (cooldown expiry)
}
