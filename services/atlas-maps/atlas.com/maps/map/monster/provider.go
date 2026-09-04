package monster

import (
	"atlas-maps/map/character"
	"context"
	"encoding/json"
)

// Count returns the number of RECURRING spawn points registered for a field.
// One-time points are deliberately excluded: this count is the denominator of
// getMonsterMax, and a one-time batch must not inflate the recurring
// population target (FR-2.5).
func (r *SpawnPointRegistry) Count(ctx context.Context, mapKey character.MapKey) (int, error) {
	n, err := r.hashes.Len(ctx, mapKey.Tenant, mapKey)
	if err != nil {
		return 0, err
	}
	return int(n), nil
}

// CountOneTime returns the number of one-time spawn points registered for a
// field. Used only on the zero-recurring branch of SpawnMonsters, to keep the
// "no spawn points" log from lying about an already-disarmed one-time field
// (FR-4.1).
func (r *SpawnPointRegistry) CountOneTime(ctx context.Context, mapKey character.MapKey) (int, error) {
	n, err := r.oneTime.Len(ctx, mapKey.Tenant, mapKey)
	if err != nil {
		return 0, err
	}
	return int(n), nil
}

// GetSpawnPointsForMap returns the spawn points for a specific map key.
// Primarily used for testing and debugging.
func (r *SpawnPointRegistry) GetSpawnPointsForMap(ctx context.Context, mapKey character.MapKey) ([]*CooldownSpawnPoint, bool) {
	entries, err := r.hashes.GetAll(ctx, mapKey.Tenant, mapKey)
	if err != nil || len(entries) == 0 {
		return nil, false
	}

	var spawnPoints []*CooldownSpawnPoint
	for _, value := range entries {
		var stored storedSpawnPoint
		if err := json.Unmarshal([]byte(value), &stored); err != nil {
			continue
		}
		spawnPoints = append(spawnPoints, fromStored(stored))
	}

	return spawnPoints, true
}
