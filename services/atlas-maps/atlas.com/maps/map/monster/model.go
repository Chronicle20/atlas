package monster

import "time"

// SpawnPoint represents a location where monsters can spawn in a map.
// It contains the basic spawn point data including position, template, and timing information.
type SpawnPoint struct {
	Id       uint32 // Unique identifier for this spawn point
	Template uint32 // Monster template ID to spawn
	MobTime  uint32 // Time-related spawn behavior (negative values indicate non-spawnable)
	Team     int32  // Team assignment for spawned monsters
	Cy       int16  // Y coordinate for spawn behavior
	F        uint32 // Flags for spawn behavior
	Fh       uint16 // Foothold for spawned monsters
	Rx0      int16  // Left boundary of spawn area
	Rx1      int16  // Right boundary of spawn area
	X        int16  // X coordinate for spawn position
	Y        int16  // Y coordinate for spawn position
}

// CooldownSpawnPoint extends SpawnPoint with cooldown tracking functionality.
// This is used by the spawn point cooldown mechanism to prevent over-spawning.
//
// The cooldown mechanism works as follows:
// - When a monster is spawned from this spawn point, NextSpawnAt is set to time.Now() + 5 seconds
// - During spawn filtering, only spawn points with NextSpawnAt.Before(now) are considered eligible
// - This prevents immediate re-spawning and enforces a 5-second cooldown period
// - The registry is scoped per MapKey (tenant/world/channel/map) for multi-tenant support
type CooldownSpawnPoint struct {
	SpawnPoint              // Embedded base spawn point data
	NextSpawnAt time.Time   // Time when this spawn point becomes eligible again (cooldown expiry)
}
