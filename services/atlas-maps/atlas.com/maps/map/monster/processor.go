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
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// defaultSpawnCooldown is the cooldown applied to a spawn point when its
// MobTime is not positive. Matches the historical default.
const defaultSpawnCooldown = 5 * time.Second

type Processor interface {
	SpawnMonsters(transactionId uuid.UUID, field field.Model) error
	ResetField(f field.Model, difficulty int) error
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

var _ Processor = (*ProcessorImpl)(nil)

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

	totalCount, err := registry.Count(p.ctx, mapKey)
	if err != nil {
		p.l.WithError(err).Errorf("Failed to count spawn points for field [%s].", f.Id())
		return err
	}
	if totalCount == 0 {
		return nil
	}

	monstersInMap, err := p.mp.CountInMap(transactionId, f)
	if err != nil {
		// Skip this pass rather than assuming zero monsters: a transient count
		// failure treated as zero would spawn the full deficit and over-populate
		// the map. Under-spawning for one tick is the safe direction.
		p.l.WithError(err).Errorf("Unable to count monsters in map; skipping spawn for field [%s] to avoid over-spawn.", f.Id())
		return err
	}

	monstersMax := p.getMonsterMax(c, totalCount)
	toSpawn := monstersMax - monstersInMap
	if toSpawn <= 0 {
		return nil
	}

	// Atomically select and reserve up to toSpawn eligible spawn points. Folding
	// selection and cooldown reservation into one Redis operation prevents
	// concurrent spawn passes (character-enter + periodic task) from reserving
	// the same points and over-spawning beyond the spawn-point count.
	// Mask the seed into Lua's exact-integer range (< 2^31) so the in-script LCG
	// shuffle stays uniform; Lua numbers are float64 and lose precision above 2^53.
	seed := time.Now().UnixNano() & 0x7fffffff
	reserved, err := registry.ReserveEligibleSpawnPoints(p.ctx, mapKey, toSpawn, defaultSpawnCooldown, seed)
	if err != nil {
		p.l.WithError(err).Errorf("Failed to reserve spawn points for field [%s].", f.Id())
		return err
	}
	if len(reserved) == 0 {
		p.l.Debugf("No eligible spawn points available (all on cooldown) for field [%s].", f.Id())
		return nil
	}

	for _, csp := range reserved {
		sp := csp.SpawnPoint
		p.l.Debugf("Spawning monster at spawn point [%d] with template [%d] at position (%d, %d)", sp.Id, sp.Template, sp.X, sp.Y)
		routine.Go(p.l, p.ctx, func(_ context.Context) {
			p.mp.CreateMonster(transactionId, f, sp.Template, sp.X, sp.Y, sp.Fh, sp.Team)
		})
	}

	p.l.Debugf("Spawned %d monsters out of %d needed for field [%s].", len(reserved), toSpawn, f.Id())
	return nil
}

func (p *ProcessorImpl) getMonsterMax(characterCount int, spawnPointCount int) int {
	spawnRate := 0.70 + (0.05 * math.Min(6, float64(characterCount)))
	return int(math.Ceil(spawnRate * float64(spawnPointCount)))
}

// ResetField is the atlas-maps composition of Cosmic's
// MapleMap.resetPQ(difficulty) / resetMapObjects(difficulty, true)
// (MapleMap.java:3962-3975): clearMapObjects() + restoreMapSpawnPoints() +
// instanceMapFirstSpawn(difficulty, isPq=true).
//
// Object-class scope of "clear" (Cosmic's clearMapObjects removes monsters,
// drops, reactors, and NPCs): this composes only the monster clear (via
// atlas-monsters' existing DELETE .../monsters) plus the spawn-point
// restore. Drops, reactors, and NPCs are NOT cleared here -- they have
// their own dedicated reset actions (clear_drops, reset_reactors,
// shuffle_reactors) landed by sibling tasks in this plan, and a map script
// composes whichever subset it needs rather than this method silently
// reaching across those domains.
//
// difficulty is accepted for parity with Cosmic's resetPQ(int difficulty)
// but is otherwise unused: atlas-maps' spawn-point/spawn-data model has no
// difficulty-bucket concept, so there is nothing for it to select between.
// Every G5 script passes 1 today. Recording that explicitly here rather
// than dropping the parameter, since dropping it would silently lose the
// only signal Cosmic gives.
//
// There is no Atlas equivalent of instanceMapFirstSpawn: restoring the
// spawn points to "eligible" is sufficient to let the existing
// character-enter/periodic SpawnMonsters passes repopulate the field on
// their normal cadence, so no additional spawn pass is triggered here.
//
// Accepted trade-off: the monster-clear leg is not compensated if the
// spawn-point restore leg fails afterward. That window is reachable today
// only for a map with no spawn-point data; TestResetFieldOnUnknownMapErrors
// pins that path.
func (p *ProcessorImpl) ResetField(f field.Model, difficulty int) error {
	_ = difficulty

	if err := p.mp.DeleteInMap(f); err != nil {
		p.l.WithError(err).Errorf("Unable to clear monsters for field [%s] during reset.", f.Id())
		return err
	}

	mapKey := character.MapKey{
		Tenant: p.t,
		Field:  f,
	}

	if _, err := GetRegistry().RestoreSpawnPoints(p.ctx, mapKey); err != nil {
		p.l.WithError(err).Errorf("Unable to restore spawn points for field [%s] during reset.", f.Id())
		return err
	}

	return nil
}
