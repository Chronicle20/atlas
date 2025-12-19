// Package monster implements spawn point management with cooldown tracking.
//
// Spawn Point Cooldown Mechanism:
// This package provides a spawn point registry system that prevents over-spawning
// by enforcing cooldown periods on individual spawn points. The key features include:
//
// - In-memory registry scoped by MapKey (tenant/world/channel/map)
// - 5-second cooldown enforcement per spawn point
// - Lazy initialization from REST provider
// - Thread-safe concurrent access with per-map mutexes
// - Maintains existing spawn rate calculations
//
// Architecture:
// - CooldownSpawnPoint: Extends SpawnPoint with NextSpawnAt timestamp
// - ProcessorImpl: Maintains registry and implements spawn logic
// - Thread safety: Per-map RWMutex for concurrent access
// - Multi-tenant: Separate registries per MapKey
//
// Usage:
// The system is transparent to existing code - SpawnMonsters() method
// maintains the same interface while adding cooldown enforcement internally.
package monster

import (
	monster2 "atlas-maps/data/map/monster"
	"atlas-maps/map/character"
	"atlas-maps/monster"
	"context"
	"math"
	"math/rand"
	"time"

	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	SpawnMonsters(transactionId uuid.UUID) func(worldId world.Id) func(channelId channel.Id) func(mapId _map.Id) error
}

// ProcessorImpl implements the Processor interface with spawn point cooldown functionality.
//
// Spawn Point Cooldown Mechanism:
// The ProcessorImpl uses a singleton SpawnPointRegistry to track spawn point cooldowns.
// This ensures that spawn point state persists across processor instances.
//
// Key Features:
// - Uses singleton SpawnPointRegistry for persistent state
// - Per-map spawn point registry scoped by MapKey (tenant/world/channel/map)
// - 5-second cooldown enforcement after each spawn
// - Thread-safe concurrent access with per-map RWMutex
// - Lazy initialization from data provider on first access
// - Maintains existing spawn rate calculations and character-based logic
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model

	dp monster2.Processor
	cp character.Processor
	mp monster.Processor
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		t:   tenant.MustFromContext(ctx),
		dp:  monster2.NewProcessor(l, ctx),
		cp:  character.NewProcessor(l, ctx),
		mp:  monster.NewProcessor(l, ctx),
	}
}

// SpawnMonsters implements the core spawn logic with cooldown enforcement.
// This method uses the spawn point registry to track cooldowns and prevent over-spawning.
//
// Cooldown Mechanism Implementation:
// 1. Create MapKey for registry scope (tenant/world/channel/map)
// 2. Initialize or retrieve spawn points registry for this map
// 3. Filter spawn points by cooldown eligibility (NextSpawnAt.Before(now))
// 4. Calculate spawn requirements based on character count and monster limits
// 5. Randomly select from eligible spawn points
// 6. Update cooldown (NextSpawnAt = now + 5 seconds) before spawning
// 7. Spawn monsters asynchronously and log activity
//
// Thread Safety:
// - Uses per-map RWMutex for safe concurrent access
// - RLock for reading spawn points during filtering
// - Lock for updating cooldowns after spawning
// - Supports concurrent spawning across different maps
//
// The method maintains existing spawn rate calculations while adding cooldown enforcement.
func (p *ProcessorImpl) SpawnMonsters(transactionId uuid.UUID) func(worldId world.Id) func(channelId channel.Id) func(mapId _map.Id) error {
	return func(worldId world.Id) func(channelId channel.Id) func(mapId _map.Id) error {
		return func(channelId channel.Id) func(mapId _map.Id) error {
			return func(mapId _map.Id) error {
				p.l.Debugf("Executing spawn mechanism for Tenant [%s] World [%d] Channel [%d] Map [%d].", p.t.String(), worldId, channelId, mapId)

				// Create MapKey for registry access
				mapKey := character.MapKey{
					Tenant:    p.t,
					WorldId:   worldId,
					ChannelId: channelId,
					MapId:     mapId,
				}

				// Get spawn points from singleton registry with initialization if needed
				registry := GetRegistry()
				spawnPoints, mutex, err := registry.GetOrInitializeSpawnPoints(mapKey, p.dp, p.l)
				if err != nil {
					p.l.WithError(err).Errorf("Failed to get spawn points for world [%d] channel [%d] map [%d].", worldId, channelId, mapId)
					return err
				}

				cs, err := p.cp.GetCharactersInMap(transactionId, worldId, channelId, mapId)
				if err != nil {
					p.l.WithError(err).Errorf("Unable to retrieve characters in map. Aborting spawning for world [%d] channel [%d] map [%d].", worldId, channelId, mapId)
					return err
				}

				c := len(cs)
				if c <= 0 {
					return nil
				}

				// Lock for reading spawn points
				mutex.RLock()

				// Filter spawn points by cooldown expiry
				now := time.Now()
				var eligibleSpawnPoints []monster2.SpawnPoint
				var eligibleIndices []int
				for i, csp := range spawnPoints {
					if csp.NextSpawnAt.Before(now) || csp.NextSpawnAt.Equal(now) {
						eligibleSpawnPoints = append(eligibleSpawnPoints, csp.SpawnPoint)
						eligibleIndices = append(eligibleIndices, i)
					}
				}

				mutex.RUnlock()

				if len(eligibleSpawnPoints) == 0 {
					p.l.Debugf("No eligible spawn points available (all on cooldown) for world [%d] channel [%d] map [%d].", worldId, channelId, mapId)
					return nil
				}

				monstersInMap, err := p.mp.CountInMap(transactionId, worldId, channelId, mapId)
				if err != nil {
					p.l.WithError(err).Warnf("Assuming no monsters in map.")
				}

				monstersMax := p.getMonsterMax(c, len(spawnPoints))

				toSpawn := monstersMax - monstersInMap
				if toSpawn <= 0 {
					return nil
				}

				// Shuffle eligible spawn points
				shuffledIndices := p.shuffleIndices(eligibleIndices)

				// Spawn monsters from eligible spawn points
				spawned := 0
				for _, idx := range shuffledIndices {
					if spawned >= toSpawn {
						break
					}

					originalIdx := eligibleIndices[idx]
					sp := spawnPoints[originalIdx].SpawnPoint

					// Update cooldown before spawning
					mutex.Lock()
					spawnPoints[originalIdx].NextSpawnAt = now.Add(5 * time.Second)
					mutex.Unlock()

					spawned++
					p.l.Debugf("Spawning monster at spawn point [%d] with template [%d] at position (%d, %d)", sp.Id, sp.Template, sp.X, sp.Y)

					go func(sp monster2.SpawnPoint) {
						p.mp.CreateMonster(transactionId, worldId, channelId, mapId, sp.Template, sp.X, sp.Y, sp.Fh, sp.Team)
					}(sp)
				}

				p.l.Debugf("Spawned %d monsters out of %d needed for world [%d] channel [%d] map [%d]. %d spawn points were on cooldown.",
					spawned, toSpawn, worldId, channelId, mapId, len(spawnPoints)-len(eligibleSpawnPoints))
				return nil
			}
		}
	}
}

func (p *ProcessorImpl) shuffle(vals []monster2.SpawnPoint) []monster2.SpawnPoint {
	r := rand.New(rand.NewSource(time.Now().Unix()))
	ret := make([]monster2.SpawnPoint, len(vals))
	perm := r.Perm(len(vals))
	for i, randIndex := range perm {
		ret[i] = vals[randIndex]
	}
	return ret
}

func (p *ProcessorImpl) shuffleIndices(indices []int) []int {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	ret := make([]int, len(indices))
	perm := r.Perm(len(indices))
	copy(ret, perm)
	return ret
}

func (p *ProcessorImpl) getMonsterMax(characterCount int, spawnPointCount int) int {
	spawnRate := 0.70 + (0.05 * math.Min(6, float64(characterCount)))
	return int(math.Ceil(spawnRate * float64(spawnPointCount)))
}
