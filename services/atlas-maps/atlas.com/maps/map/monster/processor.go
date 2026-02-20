// Package monster implements spawn point management with cooldown tracking.
//
// Spawn Point Cooldown Mechanism:
// This package provides a spawn point registry system that prevents over-spawning
// by enforcing cooldown periods on individual spawn points. The key features include:
//
// - Redis-backed registry scoped by MapKey (tenant/world/channel/map)
// - MobTime-based cooldown enforcement per spawn point (default 5s for normal monsters)
// - Lazy initialization from REST provider
// - Lua scripts for atomic eligibility checks and cooldown updates
// - Maintains existing spawn rate calculations
//
// Architecture:
// - CooldownSpawnPoint: Extends SpawnPoint with NextSpawnAt timestamp
// - ProcessorImpl: Implements spawn logic using Redis-backed SpawnPointRegistry
// - Thread safety: Redis atomicity via Lua scripts
// - Multi-tenant: Separate Redis hashes per MapKey
package monster

import (
	monster2 "atlas-maps/data/map/monster"
	"atlas-maps/map/character"
	"atlas-maps/monster"
	"context"
	"math"
	"math/rand"
	"time"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	SpawnMonsters(transactionId uuid.UUID, field field.Model) error
}

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
//
// 1. Initialize spawn point registry for this map (lazy, from data provider)
// 2. Get eligible spawn points from Redis via Lua script (NextSpawnAt <= now)
// 3. Calculate spawn requirements based on character count and total spawn points
// 4. Randomly select from eligible spawn points
// 5. Batch update cooldowns in Redis and spawn monsters asynchronously
func (p *ProcessorImpl) SpawnMonsters(transactionId uuid.UUID, f field.Model) error {
	p.l.Debugf("Executing spawn mechanism for Tenant [%s] Field [%s].", p.t.String(), f.Id())

	mapKey := character.MapKey{
		Tenant: p.t,
		Field:  f,
	}

	registry := GetRegistry()
	if err := registry.InitializeForMap(p.ctx, mapKey, p.dp, p.l); err != nil {
		p.l.WithError(err).Errorf("Failed to initialize spawn points for field [%s].", f.Id())
		return err
	}

	cs, err := p.cp.GetCharactersInMap(transactionId, f)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to retrieve characters in map. Aborting spawning for field [%s].", f.Id())
		return err
	}

	c := len(cs)
	if c <= 0 {
		return nil
	}

	eligibleSpawnPoints, totalCount, err := registry.GetEligibleSpawnPoints(p.ctx, mapKey)
	if err != nil {
		p.l.WithError(err).Errorf("Failed to get eligible spawn points for field [%s].", f.Id())
		return err
	}

	if len(eligibleSpawnPoints) == 0 {
		p.l.Debugf("No eligible spawn points available (all on cooldown) for field [%s].", f.Id())
		return nil
	}

	monstersInMap, err := p.mp.CountInMap(transactionId, f)
	if err != nil {
		p.l.WithError(err).Warnf("Assuming no monsters in map.")
	}

	monstersMax := p.getMonsterMax(c, totalCount)

	toSpawn := monstersMax - monstersInMap
	if toSpawn <= 0 {
		return nil
	}

	// Shuffle eligible spawn points
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(eligibleSpawnPoints), func(i, j int) {
		eligibleSpawnPoints[i], eligibleSpawnPoints[j] = eligibleSpawnPoints[j], eligibleSpawnPoints[i]
	})

	// Spawn monsters and collect cooldown updates
	spawned := 0
	now := time.Now()
	cooldownUpdates := make(map[uint32]time.Time)

	for _, csp := range eligibleSpawnPoints {
		if spawned >= toSpawn {
			break
		}

		sp := csp.SpawnPoint

		cooldown := 5 * time.Second
		if sp.MobTime > 0 {
			cooldown = time.Duration(sp.MobTime) * time.Second
		}
		cooldownUpdates[sp.Id] = now.Add(cooldown)

		spawned++
		p.l.Debugf("Spawning monster at spawn point [%d] with template [%d] at position (%d, %d)", sp.Id, sp.Template, sp.X, sp.Y)

		go func(sp monster2.SpawnPoint) {
			p.mp.CreateMonster(transactionId, f, sp.Template, sp.X, sp.Y, sp.Fh, sp.Team)
		}(sp)
	}

	// Batch update cooldowns in Redis
	if err := registry.UpdateCooldowns(p.ctx, mapKey, cooldownUpdates); err != nil {
		p.l.WithError(err).Errorf("Failed to update spawn point cooldowns for field [%s].", f.Id())
	}

	p.l.Debugf("Spawned %d monsters out of %d needed for field [%s]. %d spawn points were on cooldown.",
		spawned, toSpawn, f.Id(), totalCount-len(eligibleSpawnPoints))
	return nil
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
