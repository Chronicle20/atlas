package monster

import (
	"atlas-maps/map/character"
	"atlas-maps/monster"
	"context"
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"math"
	"math/rand"
	"sync"
	"time"
)

type Processor interface {
	SpawnPointProvider(mapId uint32) model.Provider[[]SpawnPoint]
	SpawnableSpawnPointProvider(mapId uint32) model.Provider[[]SpawnPoint]
	GetSpawnPoints(mapId uint32) ([]SpawnPoint, error)
	GetSpawnableSpawnPoints(mapId uint32) ([]SpawnPoint, error)
	SpawnMonsters(transactionId uuid.UUID) func(worldId world.Id) func(channelId channel.Id) func(mapId _map.Id) error
}

type ProcessorImpl struct {
	l                  logrus.FieldLogger
	ctx                context.Context
	t                  tenant.Model
	spawnPointRegistry map[character.MapKey][]*CooldownSpawnPoint
	spawnPointMu       map[character.MapKey]*sync.RWMutex
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:                  l,
		ctx:                ctx,
		t:                  tenant.MustFromContext(ctx),
		spawnPointRegistry: make(map[character.MapKey][]*CooldownSpawnPoint),
		spawnPointMu:       make(map[character.MapKey]*sync.RWMutex),
	}
}

func (p *ProcessorImpl) SpawnPointProvider(mapId uint32) model.Provider[[]SpawnPoint] {
	return requests.SliceProvider[RestModel, SpawnPoint](p.l, p.ctx)(requestSpawnPoints(mapId), Extract, model.Filters[SpawnPoint]())
}

func (p *ProcessorImpl) SpawnableSpawnPointProvider(mapId uint32) model.Provider[[]SpawnPoint] {
	return model.FilteredProvider(p.SpawnPointProvider(mapId), model.Filters(p.Spawnable))
}

func (p *ProcessorImpl) GetSpawnPoints(mapId uint32) ([]SpawnPoint, error) {
	return p.SpawnPointProvider(mapId)()
}

func (p *ProcessorImpl) GetSpawnableSpawnPoints(mapId uint32) ([]SpawnPoint, error) {
	return p.SpawnableSpawnPointProvider(mapId)()
}

func (p *ProcessorImpl) Spawnable(point SpawnPoint) bool {
	return point.MobTime >= 0
}

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

				// Get spawn points from registry with initialization if needed
				spawnPoints, mutex, err := p.getOrInitializeSpawnPoints(mapKey)
				if err != nil {
					p.l.WithError(err).Errorf("Failed to get spawn points for world [%d] channel [%d] map [%d].", worldId, channelId, mapId)
					return err
				}

				cp := character.NewProcessor(p.l, p.ctx)
				cs, err := cp.GetCharactersInMap(transactionId, worldId, channelId, mapId)
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
				var eligibleSpawnPoints []SpawnPoint
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

				monstersInMap, err := monster.NewProcessor(p.l, p.ctx).CountInMap(transactionId, worldId, channelId, mapId)
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
					
					go func(sp SpawnPoint) {
						monster.NewProcessor(p.l, p.ctx).CreateMonster(transactionId, worldId, channelId, mapId, sp.Template, sp.X, sp.Y, sp.Fh, sp.Team)
					}(sp)
				}
				
				p.l.Debugf("Spawned %d monsters out of %d needed for world [%d] channel [%d] map [%d]. %d spawn points were on cooldown.",
					spawned, toSpawn, worldId, channelId, mapId, len(spawnPoints)-len(eligibleSpawnPoints))
				return nil
			}
		}
	}
}

func (p *ProcessorImpl) shuffle(vals []SpawnPoint) []SpawnPoint {
	r := rand.New(rand.NewSource(time.Now().Unix()))
	ret := make([]SpawnPoint, len(vals))
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

func (p *ProcessorImpl) initializeRegistryForMap(mapKey character.MapKey) error {
	// Check if already initialized
	if _, exists := p.spawnPointRegistry[mapKey]; exists {
		return nil
	}

	// Get spawn points from the provider
	spawnPoints, err := p.GetSpawnableSpawnPoints(uint32(mapKey.MapId))
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
	p.spawnPointRegistry[mapKey] = cooldownSpawnPoints
	p.spawnPointMu[mapKey] = &sync.RWMutex{}

	p.l.Debugf("Initialized spawn point registry for map key: Tenant [%s] World [%d] Channel [%d] Map [%d] with %d spawn points",
		mapKey.Tenant.String(), mapKey.WorldId, mapKey.ChannelId, mapKey.MapId, len(cooldownSpawnPoints))

	return nil
}

func (p *ProcessorImpl) getOrInitializeSpawnPoints(mapKey character.MapKey) ([]*CooldownSpawnPoint, *sync.RWMutex, error) {
	// Initialize if needed
	if err := p.initializeRegistryForMap(mapKey); err != nil {
		return nil, nil, err
	}

	// Get the spawn points and mutex
	spawnPoints := p.spawnPointRegistry[mapKey]
	mutex := p.spawnPointMu[mapKey]

	return spawnPoints, mutex, nil
}
