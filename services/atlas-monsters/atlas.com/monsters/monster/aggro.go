package monster

import "time"

// Aggro decay constants. Mirror Cosmic's MonsterAggroCoordinator
// (handlers/MonsterAggroCoordinator.java:110-148) so behavior matches reference.
const (
	// AggroIdleThresholdMs is the duration in milliseconds an entry can sit without
	// a fresh hit before the decay sweep begins reducing it.
	AggroIdleThresholdMs = int64(10_000)

	// AggroDecayMultiplier is applied to a damage entry's accumulated damage on
	// each sweep tick once the entry is idle (15% reduction per 1.5s tick).
	AggroDecayMultiplier = 0.85

	// AggroDecayFloor is the minimum damage value an entry can hold; once a
	// decayed value falls below this floor the entry is pruned.
	AggroDecayFloor = uint32(1)
)

// AggroSweepInterval is the cadence at which MonsterAggroDecayTask runs.
const AggroSweepInterval = 1500 * time.Millisecond

// IsAggroIdle reports whether the entry's last hit is older than the idle
// threshold.
func IsAggroIdle(e entry, nowMs int64) bool {
	return nowMs-e.LastHitMs > AggroIdleThresholdMs
}
