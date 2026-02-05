package monster

// SpawnPoint represents a location where monsters can spawn in a map.
// It contains the basic spawn point data including position, template, and timing information.
type SpawnPoint struct {
	Id       uint32 // Unique identifier for this spawn point
	Template uint32 // Monster template ID to spawn
	MobTime  uint32 // Time-related spawn behavior (negative values indicate non-spawnable)
	Team     int8   // Team assignment for spawned monsters
	Cy       int16  // Y coordinate for spawn behavior
	F        uint32 // Flags for spawn behavior
	Fh       int16  // Foothold for spawned monsters
	Rx0      int16  // Left boundary of spawn area
	Rx1      int16  // Right boundary of spawn area
	X        int16  // X coordinate for spawn position
	Y        int16  // Y coordinate for spawn position
}
