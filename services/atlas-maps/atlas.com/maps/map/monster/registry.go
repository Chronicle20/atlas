package monster

import (
	monster2 "atlas-maps/data/map/monster"
	"atlas-maps/map/character"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// SpawnPointRegistry is a singleton that manages spawn point cooldowns across all processor instances.
// It ensures that spawn point cooldown state persists even when processors are recreated.
//
// Key Features:
// - Singleton pattern ensures single instance per application
// - Per-map spawn point registry scoped by MapKey (tenant/world/channel/map)
// - Thread-safe concurrent access with per-map mutexes
// - Lazy initialization from data provider on first access
// - Maintains spawn point cooldown state across processor lifecycles
type SpawnPointRegistry struct {
	// spawnPointRegistry maintains the in-memory spawn point registry with cooldown tracking.
	// Key: MapKey (tenant/world/channel/map combination)
	// Value: Array of CooldownSpawnPoint instances for that map
	// This registry is lazily initialized when first accessed for each map.
	spawnPointRegistry map[character.MapKey][]*CooldownSpawnPoint

	// spawnPointMu provides thread-safe access to the spawn point registry.
	// Each MapKey has its own RWMutex to allow concurrent access across different maps
	// while maintaining safety within each map's spawn point operations.
	spawnPointMu map[character.MapKey]*sync.RWMutex

	// registryMu protects the registry maps themselves during initialization
	registryMu sync.RWMutex
}

// singleton instance
var (
	registryInstance *SpawnPointRegistry
	registryOnce     sync.Once
)

// GetRegistry returns the singleton instance of SpawnPointRegistry.
// This ensures that spawn point cooldown state is maintained across processor instances.
func GetRegistry() *SpawnPointRegistry {
	registryOnce.Do(func() {
		registryInstance = &SpawnPointRegistry{
			spawnPointRegistry: make(map[character.MapKey][]*CooldownSpawnPoint),
			spawnPointMu:       make(map[character.MapKey]*sync.RWMutex),
		}
	})
	return registryInstance
}

// InitializeForMap performs lazy initialization of the spawn point registry for a specific map.
// This method is called on first access to ensure the registry is populated with spawn points
// from the data provider and properly configured with cooldown tracking.
//
// Process:
// 1. Check if registry already exists for this MapKey (avoid duplicate initialization)
// 2. Fetch spawn points from the data provider
// 3. Convert each SpawnPoint to CooldownSpawnPoint with NextSpawnAt = time.Now()
// 4. Initialize the registry entry and per-map mutex for thread safety
// 5. Log the initialization for debugging purposes
//
// The method is thread-safe and idempotent - multiple calls will not cause issues.
func (r *SpawnPointRegistry) InitializeForMap(mapKey character.MapKey, dp monster2.Processor, l logrus.FieldLogger) error {
	r.registryMu.Lock()
	defer r.registryMu.Unlock()

	// Check if already initialized
	if _, exists := r.spawnPointRegistry[mapKey]; exists {
		return nil
	}

	// Get spawn points from the data provider
	spawnPoints, err := dp.GetSpawnableSpawnPoints(uint32(mapKey.MapId))
	if err != nil {
		return err
	}

	// Convert to CooldownSpawnPoint with initial NextSpawnAt
	now := time.Now()
	cooldownSpawnPoints := make([]*CooldownSpawnPoint, len(spawnPoints))
	for i, sp := range spawnPoints {
		cooldownSpawnPoints[i] = &CooldownSpawnPoint{
			SpawnPoint:  sp,
			NextSpawnAt: now,
		}
	}

	// Initialize registry entry and mutex
	r.spawnPointRegistry[mapKey] = cooldownSpawnPoints
	r.spawnPointMu[mapKey] = &sync.RWMutex{}

	l.Debugf("Initialized spawn point registry for map key: Tenant [%s] World [%d] Channel [%d] Map [%d] with %d spawn points",
		mapKey.Tenant.String(), mapKey.WorldId, mapKey.ChannelId, mapKey.MapId, len(cooldownSpawnPoints))

	return nil
}

// GetOrInitializeSpawnPoints is a helper function that ensures the spawn point registry
// is initialized for the given MapKey and returns the spawn points and mutex for safe access.
//
// This function combines initialization and access in a single operation:
// 1. Calls InitializeForMap to ensure registry exists (lazy initialization)
// 2. Retrieves the spawn points array for the map
// 3. Retrieves the associated mutex for thread-safe operations
//
// Returns:
// - []*CooldownSpawnPoint: Array of spawn points with cooldown tracking
// - *sync.RWMutex: Mutex for thread-safe access to the spawn points
// - error: Any error that occurred during initialization
//
// Usage pattern:
//
//	spawnPoints, mutex, err := registry.GetOrInitializeSpawnPoints(mapKey, dp, l)
//	if err != nil { handle error }
//	mutex.RLock() // or mutex.Lock() for writes
//	// access spawnPoints safely
//	mutex.RUnlock() // or mutex.Unlock() for writes
func (r *SpawnPointRegistry) GetOrInitializeSpawnPoints(mapKey character.MapKey, dp monster2.Processor, l logrus.FieldLogger) ([]*CooldownSpawnPoint, *sync.RWMutex, error) {
	// Initialize if needed
	if err := r.InitializeForMap(mapKey, dp, l); err != nil {
		return nil, nil, err
	}

	// Get the spawn points and mutex (protected by read lock)
	r.registryMu.RLock()
	spawnPoints := r.spawnPointRegistry[mapKey]
	mutex := r.spawnPointMu[mapKey]
	r.registryMu.RUnlock()

	return spawnPoints, mutex, nil
}

// Reset clears all spawn point registries. This is primarily used for testing.
func (r *SpawnPointRegistry) Reset() {
	r.registryMu.Lock()
	defer r.registryMu.Unlock()

	r.spawnPointRegistry = make(map[character.MapKey][]*CooldownSpawnPoint)
	r.spawnPointMu = make(map[character.MapKey]*sync.RWMutex)
}

// GetSpawnPointsForMap returns the spawn points for a specific map key (read-only access).
// This is primarily used for testing and debugging.
func (r *SpawnPointRegistry) GetSpawnPointsForMap(mapKey character.MapKey) ([]*CooldownSpawnPoint, bool) {
	r.registryMu.RLock()
	defer r.registryMu.RUnlock()

	spawnPoints, exists := r.spawnPointRegistry[mapKey]
	return spawnPoints, exists
}